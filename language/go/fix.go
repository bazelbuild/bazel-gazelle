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

package golang

import (
	"log"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/language/proto"
	"github.com/bazelbuild/bazel-gazelle/rule"
	bzl "github.com/bazelbuild/buildtools/build"
)

func (_ *goLang) Fix(c *config.Config, f *rule.File) {
	migrateLibraryEmbed(c, f)
	migrateGrpcCompilers(c, f)
	flattenSrcs(c, f)
	squashCgoLibrary(c, f)
	squashXtest(c, f)
	migrateNamingConvention(c, f)
	removeLegacyProto(c, f)
	removeLegacyGazelle(c, f)
}

// migrateNamingConvention renames rules according to go_naming_convention directives.
func migrateNamingConvention(c *config.Config, f *rule.File) {
	nc := getGoConfig(c).goNamingConvention

	binName := binName(f)
	importPath := importPath(f)
	libName := libNameByConvention(nc, binName, importPath)
	testName := testNameByConvention(nc, binName, importPath)
	var migrateLibName, migrateTestName string
	switch nc {
	case goDefaultLibraryNamingConvention:
		migrateLibName = libNameByConvention(importNamingConvention, binName, importPath)
		migrateTestName = testNameByConvention(importNamingConvention, binName, importPath)
	case importNamingConvention, importAliasNamingConvention:
		migrateLibName = defaultLibName
		migrateTestName = defaultTestName
	default:
		return
	}

	for _, r := range f.Rules {
		switch r.Kind() {
		case "go_binary":
			replaceInStrListAttr(r, "embed", ":"+migrateLibName, ":"+libName)
		case "go_library":
			if r.Name() == migrateLibName {
				r.SetName(libName)
			}
		case "go_test":
			if r.Name() == migrateTestName {
				r.SetName(testName)
				replaceInStrListAttr(r, "embed", ":"+migrateLibName, ":"+libName)
			}
		}
	}

	// Alias migration
	if binName == "" {
		var ar *rule.Rule
		var lib *rule.Rule
		for _, r := range f.Rules {
			if r.Kind() == "alias" && r.Name() == defaultLibName {
				ar = r
			} else if r.Kind() == "go_library" && r.Name() == libName {
				lib = r
			}
		}
		if nc == importAliasNamingConvention {
			if ar == nil && lib != nil {
				r := rule.NewRule("alias", defaultLibName)
				r.SetAttr("actual", ":"+lib.Name())
				visibility := lib.Attr("visibility")
				if visibility != nil {
					r.SetAttr("visibility", visibility)
				}
				r.Insert(f)
			}
		} else {
			if ar != nil {
				ar.Delete()
			}
		}
	}
}

// binName returns the name of a go_binary rule if one can be found.
func binName(f *rule.File) string {
	for _, r := range f.Rules {
		if r.Kind() == "go_binary" {
			return r.Name()
		}
	}
	return ""
}

// import path returns the existing import path from the first encountered Go rule with the attribute set.
func importPath(f *rule.File) string {
	for _, r := range f.Rules {
		switch r.Kind() {
		case "go_binary", "go_library", "go_test":
			if ip, ok := r.Attr("importpath").(*bzl.StringExpr); ok && ip.Value != "" {
				return ip.Value
			}
		}
	}
	return f.Pkg
}

func replaceInStrListAttr(r *rule.Rule, attr, old, new string) {
	l := r.AttrStrings(attr)
	var shouldAdd = true
	var items []string
	for _, v := range l {
		if v != old {
			items = append(items, v)
		}
		shouldAdd = shouldAdd && v != new
	}
	if shouldAdd {
		items = append(items, new)
	}
	r.SetAttr(attr, items)
}

// migrateLibraryEmbed converts "library" attributes to "embed" attributes,
// preserving comments. This only applies to Go rules, and only if there is
// no keep comment on "library" and no existing "embed" attribute.
func migrateLibraryEmbed(c *config.Config, f *rule.File) {
	for _, r := range f.Rules {
		if !isGoRule(r.Kind()) {
			continue
		}
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
		r.SetAttr("compilers", []string{grpcCompilerLabel})
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
	binName := binName(f)
	importPath := importPath(f)
	libName := libNameByConvention(getGoConfig(c).goNamingConvention, binName, importPath)
	var cgoLibrary, goLibrary *rule.Rule
	for _, r := range f.Rules {
		if r.Kind() == "cgo_library" && r.Name() == "cgo_default_library" && !r.ShouldKeep() {
			if cgoLibrary != nil {
				log.Printf("%s: when fixing existing file, multiple cgo_library rules with default name found", f.Path)
				continue
			}
			cgoLibrary = r
			continue
		}
		if r.Kind() == "go_library" && r.Name() == libName {
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

	// If there wasn't an existing library to squash into, we'll have to guess at its name.
	if goLibrary == nil {
		cgoLibrary.SetKind("go_library")
		cgoLibrary.SetName(libName)
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
	binName := binName(f)
	importPath := importPath(f)
	testName := testNameByConvention(getGoConfig(c).goNamingConvention, binName, importPath)
	var itest, xtest *rule.Rule
	for _, r := range f.Rules {
		if r.Kind() != "go_test" {
			continue
		}
		if r.Name() == testName {
			itest = r
		} else if r.Name() == "go_default_xtest" {
			xtest = r
		}
	}

	if xtest == nil || xtest.ShouldKeep() || (itest != nil && itest.ShouldKeep()) {
		return
	}
	if !c.ShouldFix {
		if itest == nil {
			log.Printf("%s: go_default_xtest is no longer necessary. Run 'gazelle fix' to rename to %s.", f.Path, testName)
		} else {
			log.Printf("%s: go_default_xtest is no longer necessary. Run 'gazelle fix' to squash with %s.", f.Path, testName)
		}
		return
	}

	// If there was no internal test, we can just rename the external test.
	if itest == nil {
		xtest.SetName(testName)
		return
	}

	// Attempt to squash.
	if err := rule.SquashRules(xtest, itest, f.Path); err != nil {
		log.Print(err)
		return
	}
	xtest.Delete()
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

// removeLegacyProto removes uses of the old proto rules. It deletes loads
// from go_proto_library.bzl. It deletes proto filegroups. It removes
// go_proto_library attributes which are no longer recognized. New rules
// are generated in place of the deleted rules, but attributes and comments
// are not migrated.
func removeLegacyProto(c *config.Config, f *rule.File) {
	// Don't fix if the proto mode was set to something other than the default.
	if pcMode := getProtoMode(c); pcMode != proto.DefaultMode {
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
		if r.Kind() == "filegroup" && r.Name() == legacyProtoFilegroupName {
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

// removeLegacyGazelle removes loads of the "gazelle" macro from
// @io_bazel_rules_go//go:def.bzl. The definition has moved to
// @bazel_gazelle//:def.bzl, and the old one will be deleted soon.
func removeLegacyGazelle(c *config.Config, f *rule.File) {
	for _, l := range f.Loads {
		if l.Name() == "@io_bazel_rules_go//go:def.bzl" && l.Has("gazelle") {
			l.Remove("gazelle")
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
