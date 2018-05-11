/* Copyright 2017 The Bazel Authors. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package merger

import (
	"fmt"
	"log"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/internal/config"
	"github.com/bazelbuild/bazel-gazelle/internal/rule"
	bzl "github.com/bazelbuild/buildtools/build"
)

// Much of this file could be simplified by using
// github.com/bazelbuild/buildtools/edit. However, through a transitive
// dependency, that library depends on a proto in Bazel itself, which is
// a 95MB download. Not worth it.

// FixFile updates rules in f that were generated by an older version of
// Gazelle to a newer form that can be merged with freshly generated rules.
//
// If c.ShouldFix is true, FixFile may perform potentially destructive
// transformations, such as squashing or deleting rules (e.g., cgo_library).
// If not, FixFile will perform a set of low-risk transformations (e.g., removing
// unused attributes) and will print a message about transformations it
// would have performed.
//
// FixLoads should be called after this, since it will fix load statements that
// may be broken by transformations applied by this function.
func FixFile(c *config.Config, f *rule.File) {
	migrateLibraryEmbed(c, f)
	migrateGrpcCompilers(c, f)
	flattenSrcs(c, f)
	squashCgoLibrary(c, f)
	squashXtest(c, f)
	removeLegacyProto(c, f)
}

// migrateLibraryEmbed converts "library" attributes to "embed" attributes,
// preserving comments. This only applies to Go rules, and only if there is
// no keep comment on "library" and no existing "embed" attribute.
func migrateLibraryEmbed(c *config.Config, f *rule.File) {
	for _, r := range f.Rules {
		libExpr := r.Attr("library")
		if libExpr == nil || rule.ShouldKeep(libExpr) || r.Attr("embed") != nil {
			continue
		}
		r.DelAttr("library")
		r.SetAttr("embed", &bzl.ListExpr{List: []bzl.Expr{libExpr}})
	}
}

// migrateGrpcCompilers converts "go_grpc_library" rules into "go_proto_library"
// rules with a "compilers" attribute.
func migrateGrpcCompilers(c *config.Config, f *rule.File) {
	for _, r := range f.Rules {
		if r.Kind() != "go_grpc_library" || r.ShouldKeep() || r.Attr("compilers") != nil {
			continue
		}
		r.SetKind("go_proto_library")
		r.SetAttr("compilers", []string{config.GrpcCompilerLabel})
	}
}

// squashCgoLibrary removes cgo_library rules with the default name and
// merges their attributes with go_library with the default name. If no
// go_library rule exists, a new one will be created.
//
// Note that the library attribute is disregarded, so cgo_library and
// go_library attributes will be squashed even if the cgo_library was unlinked.
// MergeFile will remove unused values and attributes later.
func squashCgoLibrary(c *config.Config, f *rule.File) {
	// Find the default cgo_library and go_library rules.
	var cgoLibrary, goLibrary *rule.Rule
	for _, r := range f.Rules {
		if r.Kind() == "cgo_library" && r.Name() == config.DefaultCgoLibName && !r.ShouldKeep() {
			if cgoLibrary != nil {
				log.Printf("%s: when fixing existing file, multiple cgo_library rules with default name found", f.Path)
				continue
			}
			cgoLibrary = r
			continue
		}
		if r.Kind() == "go_library" && r.Name() == config.DefaultLibName {
			if goLibrary != nil {
				log.Printf("%s: when fixing existing file, multiple go_library rules with default name referencing cgo_library found", f.Path)
			}
			goLibrary = r
			continue
		}
	}

	if cgoLibrary == nil {
		return
	}
	if !c.ShouldFix {
		log.Printf("%s: cgo_library is deprecated. Run 'gazelle fix' to squash with go_library.", f.Path)
		return
	}

	if goLibrary == nil {
		cgoLibrary.SetKind("go_library")
		cgoLibrary.SetName(config.DefaultLibName)
		cgoLibrary.SetAttr("cgo", true)
		return
	}

	if err := rule.SquashRules(cgoLibrary, goLibrary, f.Path); err != nil {
		log.Print(err)
		return
	}
	goLibrary.DelAttr("embed")
	goLibrary.SetAttr("cgo", true)
	cgoLibrary.Delete()
}

// squashXtest removes go_test rules with the default external name and merges
// their attributes with a go_test rule with the default internal name. If
// no internal go_test rule exists, a new one will be created (effectively
// renaming the old rule).
func squashXtest(c *config.Config, f *rule.File) {
	// Search for internal and external tests.
	var itest, xtest *rule.Rule
	for _, r := range f.Rules {
		if r.Kind() != "go_test" {
			continue
		}
		if r.Name() == config.DefaultTestName {
			itest = r
		} else if r.Name() == config.DefaultXTestName {
			xtest = r
		}
	}

	if xtest == nil || xtest.ShouldKeep() || (itest != nil && itest.ShouldKeep()) {
		return
	}
	if !c.ShouldFix {
		if itest == nil {
			log.Printf("%s: go_default_xtest is no longer necessary. Run 'gazelle fix' to rename to go_default_test.", f.Path)
		} else {
			log.Printf("%s: go_default_xtest is no longer necessary. Run 'gazelle fix' to squash with go_default_test.", f.Path)
		}
		return
	}

	// If there was no internal test, we can just rename the external test.
	if itest == nil {
		xtest.SetName(config.DefaultTestName)
		return
	}

	// Attempt to squash.
	if err := rule.SquashRules(xtest, itest, f.Path); err != nil {
		log.Print(err)
		return
	}
	xtest.Delete()
}

// removeLegacyProto removes uses of the old proto rules. It deletes loads
// from go_proto_library.bzl. It deletes proto filegroups. It removes
// go_proto_library attributes which are no longer recognized. New rules
// are generated in place of the deleted rules, but attributes and comments
// are not migrated.
func removeLegacyProto(c *config.Config, f *rule.File) {
	// Don't fix if the proto mode was set to something other than the default.
	if c.ProtoMode != config.DefaultProtoMode {
		return
	}

	// Scan for definitions to delete.
	var protoLoads []*rule.Load
	for _, l := range f.Loads {
		if l.Name() == "@io_bazel_rules_go//proto:go_proto_library.bzl" {
			protoLoads = append(protoLoads, l)
		}
	}
	var protoFilegroups, protoRules []*rule.Rule
	for _, r := range f.Rules {
		if r.Kind() == "filegroup" && r.Name() == config.DefaultProtosName {
			protoFilegroups = append(protoFilegroups, r)
		}
		if r.Kind() == "go_proto_library" {
			protoRules = append(protoRules, r)
		}
	}
	if len(protoLoads)+len(protoFilegroups) == 0 {
		return
	}
	if !c.ShouldFix {
		log.Printf("%s: go_proto_library.bzl is deprecated. Run 'gazelle fix' to replace old rules.", f.Path)
		return
	}

	// Delete legacy proto loads and filegroups. Only delete go_proto_library
	// rules if we deleted a load.
	for _, l := range protoLoads {
		l.Delete()
	}
	for _, r := range protoFilegroups {
		r.Delete()
	}
	if len(protoLoads) > 0 {
		for _, r := range protoRules {
			r.Delete()
		}
	}
}

// flattenSrcs transforms srcs attributes structured as concatenations of
// lists and selects (generated from PlatformStrings; see
// extractPlatformStringsExprs for matching details) into a sorted,
// de-duplicated list. Comments are accumulated and de-duplicated across
// duplicate expressions.
func flattenSrcs(c *config.Config, f *rule.File) {
	for _, r := range f.Rules {
		if !isGoRule(r.Kind()) {
			continue
		}
		oldSrcs := r.Attr("srcs")
		if oldSrcs == nil {
			continue
		}
		flatSrcs := rule.FlattenExpr(oldSrcs)
		if flatSrcs != oldSrcs {
			r.SetAttr("srcs", flatSrcs)
		}
	}
}

// FixLoads removes loads of unused go rules and adds loads of newly used rules.
// This should be called after FixFile and MergeFile, since symbols
// may be introduced that aren't loaded.
//
// This function calls File.Sync before processing loads.
func FixLoads(f *rule.File) {
	// Sync the file. We need File.Loads and File.Rules to contain inserted
	// statements and not deleted statements.
	f.Sync()

	// Scan load statements in the file. Keep track of loads of known files,
	// since these may be changed. Keep track of symbols loaded from unknown
	// files; we will not add loads for these.
	var loads []*rule.Load
	otherLoadedKinds := make(map[string]bool)
	for _, l := range f.Loads {
		if knownFiles[l.Name()] {
			loads = append(loads, l)
			continue
		}
		for _, sym := range l.Symbols() {
			otherLoadedKinds[sym] = true
		}
	}

	// Make a map of all the symbols from known files used in this file.
	usedKinds := make(map[string]map[string]bool)
	for _, r := range f.Rules {
		kind := r.Kind()
		if file, ok := knownKinds[kind]; ok && !otherLoadedKinds[kind] {
			if usedKinds[file] == nil {
				usedKinds[file] = make(map[string]bool)
			}
			usedKinds[file][kind] = true
		}
	}

	// Fix the load statements. The order is important, so we iterate over
	// knownLoads instead of knownFiles.
	for _, known := range knownLoads {
		file := known.file
		first := true
		for _, l := range loads {
			if l.Name() != file {
				continue
			}
			if first {
				fixLoad(l, file, usedKinds[file])
				first = false
			} else {
				fixLoad(l, file, nil)
			}
			if l.IsEmpty() {
				l.Delete()
			}
		}
		if first {
			load := fixLoad(nil, file, usedKinds[file])
			if load != nil {
				index := newLoadIndex(f, known.after)
				load.Insert(f, index)
			}
		}
	}
}

// knownLoads is a list of files Gazelle will generate loads from and
// the symbols it knows about. All symbols Gazelle ever generated
// loads for are present, including symbols it no longer uses (e.g.,
// cgo_library). Manually loaded symbols (e.g., go_embed_data) are not
// included.
//
// Some symbols have a list of function calls that they should be loaded
// after. This is important for WORKSPACE, where function calls may
// introduce new repository names.
//
// The order of the files here will match the order of generated load
// statements. The symbols should be sorted lexicographically. If a
// symbol appears in more than one file (e.g., because it was moved),
// it will be loaded from the last file in this list.
var knownLoads = []struct {
	file  string
	kinds []string
	after []string
}{
	{
		file: "@io_bazel_rules_go//go:def.bzl",
		kinds: []string{
			"cgo_library",
			"go_binary",
			"go_library",
			"go_prefix",
			"go_repository",
			"go_test",
		},
	}, {
		file: "@io_bazel_rules_go//proto:def.bzl",
		kinds: []string{
			"go_grpc_library",
			"go_proto_library",
		},
	}, {
		file: "@bazel_gazelle//:deps.bzl",
		kinds: []string{
			"go_repository",
		},
		after: []string{
			"go_rules_dependencies",
			"go_register_toolchains",
			"gazelle_dependencies",
		},
	},
}

// knownFiles is the set of labels for files that Gazelle loads symbols from.
var knownFiles map[string]bool

// knownKinds is a map from symbols to labels of the files they are loaded
// from.
var knownKinds map[string]string

func init() {
	knownFiles = make(map[string]bool)
	knownKinds = make(map[string]string)
	for _, l := range knownLoads {
		knownFiles[l.file] = true
		for _, k := range l.kinds {
			knownKinds[k] = l.file
		}
	}
}

// fixLoad updates a load statement with the given symbols. If load is nil,
// a new load may be created and returned. Symbols in kinds will be added
// to the load if they're not already present. Known symbols not in kinds
// will be removed if present. Other symbols will be preserved. If load is
// empty, nil is returned.
func fixLoad(load *rule.Load, file string, kinds map[string]bool) *rule.Load {
	if load == nil {
		if len(kinds) == 0 {
			return nil
		}
		load = rule.NewLoad(file)
	}

	for k := range kinds {
		load.Add(k)
	}
	for _, k := range load.Symbols() {
		if knownKinds[k] != "" && !kinds[k] {
			load.Remove(k)
		}
	}
	return load
}

// newLoadIndex returns the index in stmts where a new load statement should
// be inserted. after is a list of function names that the load should not
// be inserted before.
func newLoadIndex(f *rule.File, after []string) int {
	if len(after) == 0 {
		return 0
	}
	index := 0
	for _, r := range f.Rules {
		for _, a := range after {
			if r.Kind() == a && r.Index() >= index {
				index = r.Index() + 1
			}
		}
	}
	return index
}

// FixWorkspace updates rules in the WORKSPACE file f that were used with an
// older version of rules_go or gazelle.
func FixWorkspace(f *rule.File) {
	removeLegacyGoRepository(f)
}

// CheckGazelleLoaded searches the given WORKSPACE file for a repository named
// "bazel_gazelle". If no such repository is found *and* the repo is not
// declared with a directive *and* at least one load statement mentions
// the repository, a descriptive error will be returned.
//
// This should be called after modifications have been made to WORKSPACE
// (i.e., after FixLoads) before writing it to disk.
func CheckGazelleLoaded(f *rule.File) error {
	needGazelle := false
	for _, l := range f.Loads {
		if strings.HasPrefix(l.Name(), "@bazel_gazelle//") {
			needGazelle = true
		}
	}
	if !needGazelle {
		return nil
	}
	for _, r := range f.Rules {
		if r.Name() == "bazel_gazelle" {
			return nil
		}
	}
	for _, d := range f.Directives {
		if d.Key != "repo" {
			continue
		}
		if fs := strings.Fields(d.Value); len(fs) > 0 && fs[0] == "bazel_gazelle" {
			return nil
		}
	}
	return fmt.Errorf(`%s: error: bazel_gazelle is not declared in WORKSPACE.
Without this repository, Gazelle cannot safely modify the WORKSPACE file.
See the instructions at https://github.com/bazelbuild/bazel-gazelle.
If the bazel_gazelle is declared inside a macro, you can suppress this error
by adding a comment like this to WORKSPACE:
    # gazelle:repo bazel_gazelle
`, f.Path)
}

// removeLegacyGoRepository removes loads of go_repository from
// @io_bazel_rules_go. FixLoads should be called after this; it will load from
// @bazel_gazelle.
func removeLegacyGoRepository(f *rule.File) {
	for _, l := range f.Loads {
		if l.Name() == "@io_bazel_rules_go//go:def.bzl" {
			l.Remove("go_repository")
			if l.IsEmpty() {
				l.Delete()
			}
		}
	}
}

func isGoRule(kind string) bool {
	return kind == "go_library" ||
		kind == "go_binary" ||
		kind == "go_test" ||
		kind == "go_proto_library" ||
		kind == "go_grpc_library"
}
