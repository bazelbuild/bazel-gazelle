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
	"log"
	"sort"

	"github.com/bazelbuild/bazel-gazelle/internal/config"
	bf "github.com/bazelbuild/buildtools/build"
)

// Much of this file could be simplified by using
// github.com/bazelbuild/buildtools/edit. However, through a transitive
// dependency, that library depends on a proto in Bazel itself, which is
// a 95MB download. Not worth it.

// FixFile updates rules in oldFile that were generated by an older version of
// Gazelle to a newer form that can be merged with freshly generated rules.
//
// FixLoads should be called after this, since it will fix load
// statements that may be broken by transformations applied by this function.
func FixFile(c *config.Config, oldFile *bf.File) *bf.File {
	fixedFile := squashCgoLibrary(oldFile)
	return removeLegacyProto(c, fixedFile)
}

// squashCgoLibrary removes cgo_library rules with the default name and
// merges their attributes with go_library with the default name. If no
// go_library rule exists, a new one will be created.
//
// Note that the library attribute is disregarded, so cgo_library and
// go_library attributes will be squashed even if the cgo_library was unlinked.
// MergeWithExisting will remove unused values and attributes later.
func squashCgoLibrary(oldFile *bf.File) *bf.File {
	// Find the default cgo_library and go_library rules.
	var cgoLibrary, goLibrary bf.Rule
	cgoLibraryIndex := -1
	goLibraryIndex := -1

	for i, stmt := range oldFile.Stmt {
		c, ok := stmt.(*bf.CallExpr)
		if !ok {
			continue
		}
		r := bf.Rule{Call: c}
		if r.Kind() == "cgo_library" && r.Name() == config.DefaultCgoLibName && !shouldKeep(c) {
			if cgoLibrary.Call != nil {
				log.Printf("%s: when fixing existing file, multiple cgo_library rules with default name found", oldFile.Path)
				continue
			}
			cgoLibrary = r
			cgoLibraryIndex = i
			continue
		}
		if r.Kind() == "go_library" && r.Name() == config.DefaultLibName {
			if goLibrary.Call != nil {
				log.Printf("%s: when fixing existing file, multiple go_library rules with default name referencing cgo_library found", oldFile.Path)
				continue
			}
			goLibrary = r
			goLibraryIndex = i
		}
	}

	if cgoLibrary.Call == nil {
		return oldFile
	}

	// If go_library has a '# keep' comment, just delete cgo_library.
	if goLibrary.Call != nil && shouldKeep(goLibrary.Call) {
		fixedFile := *oldFile
		fixedFile.Stmt = append(fixedFile.Stmt[:cgoLibraryIndex], fixedFile.Stmt[cgoLibraryIndex+1:]...)
		return &fixedFile
	}

	// Copy the comments and attributes from cgo_library into go_library. If no
	// go_library exists, create an empty one.
	var fixedGoLibraryExpr bf.CallExpr
	fixedGoLibrary := bf.Rule{&fixedGoLibraryExpr}
	if goLibrary.Call == nil {
		fixedGoLibrary.SetKind("go_library")
		fixedGoLibrary.SetAttr("name", &bf.StringExpr{Value: config.DefaultLibName})
		if vis := cgoLibrary.Attr("visibility"); vis != nil {
			fixedGoLibrary.SetAttr("visibility", vis)
		}
	} else {
		fixedGoLibraryExpr = *goLibrary.Call
		fixedGoLibraryExpr.List = append([]bf.Expr{}, goLibrary.Call.List...)
	}

	fixedGoLibrary.DelAttr("embed")
	fixedGoLibrary.SetAttr("cgo", &bf.LiteralExpr{Token: "True"})

	fixedGoLibraryExpr.Comments.Before = append(fixedGoLibraryExpr.Comments.Before, cgoLibrary.Call.Comments.Before...)
	fixedGoLibraryExpr.Comments.Suffix = append(fixedGoLibraryExpr.Comments.Suffix, cgoLibrary.Call.Comments.Suffix...)
	fixedGoLibraryExpr.Comments.After = append(fixedGoLibraryExpr.Comments.After, cgoLibrary.Call.Comments.After...)

	for _, key := range []string{"cdeps", "clinkopts", "copts", "data", "deps", "gc_goopts", "srcs"} {
		goLibraryAttr := fixedGoLibrary.Attr(key)
		cgoLibraryAttr := cgoLibrary.Attr(key)
		if cgoLibraryAttr == nil {
			continue
		}
		if fixedAttr, err := squashExpr(goLibraryAttr, cgoLibraryAttr); err == nil {
			fixedGoLibrary.SetAttr(key, fixedAttr)
		}
	}

	// Rebuild the file with the cgo_library removed and the go_library replaced.
	// If the go_library didn't already exist, it will replace cgo_library.
	fixedFile := *oldFile
	if goLibrary.Call == nil {
		fixedFile.Stmt = append([]bf.Expr{}, oldFile.Stmt...)
		fixedFile.Stmt[cgoLibraryIndex] = &fixedGoLibraryExpr
	} else {
		fixedFile.Stmt = append(oldFile.Stmt[:cgoLibraryIndex], oldFile.Stmt[cgoLibraryIndex+1:]...)
		if goLibraryIndex > cgoLibraryIndex {
			goLibraryIndex--
		}
		fixedFile.Stmt[goLibraryIndex] = &fixedGoLibraryExpr
	}
	return &fixedFile
}

// squashExpr combines two expressions. Unlike mergeExpr, squashExpr does not
// discard information from an "old" expression. It does not sort or de-duplicate
// elements. Any non-scalar expressions that mergeExpr understands can be
// squashed.
func squashExpr(x, y bf.Expr) (bf.Expr, error) {
	xExprs, err := extractPlatformStringsExprs(x)
	if err != nil {
		return nil, err
	}
	yExprs, err := extractPlatformStringsExprs(y)
	if err != nil {
		return nil, err
	}
	squashedExprs, err := squashPlatformStringsExprs(xExprs, yExprs)
	if err != nil {
		return nil, err
	}
	return makePlatformStringsExpr(squashedExprs), nil
}

func squashPlatformStringsExprs(x, y platformStringsExprs) (platformStringsExprs, error) {
	var ps platformStringsExprs
	var err error
	ps.generic = squashList(x.generic, y.generic)
	if ps.os, err = squashDict(x.os, y.os); err != nil {
		return platformStringsExprs{}, err
	}
	if ps.arch, err = squashDict(x.arch, y.arch); err != nil {
		return platformStringsExprs{}, err
	}
	if ps.platform, err = squashDict(x.platform, y.platform); err != nil {
		return platformStringsExprs{}, err
	}
	return ps, nil
}

func squashList(x, y *bf.ListExpr) *bf.ListExpr {
	if x == nil {
		return y
	}
	if y == nil {
		return x
	}
	squashed := *x
	squashed.Comments.Before = append(x.Comments.Before, y.Comments.Before...)
	squashed.Comments.Suffix = append(x.Comments.Suffix, y.Comments.Suffix...)
	squashed.Comments.After = append(x.Comments.After, y.Comments.After...)
	squashed.List = append(x.List, y.List...)
	return &squashed
}

func squashDict(x, y *bf.DictExpr) (*bf.DictExpr, error) {
	if x == nil {
		return y, nil
	}
	if y == nil {
		return x, nil
	}

	squashed := *x
	squashed.Comments.Before = append(x.Comments.Before, y.Comments.Before...)
	squashed.Comments.Suffix = append(x.Comments.Suffix, y.Comments.Suffix...)
	squashed.Comments.After = append(x.Comments.After, y.Comments.After...)

	xCaseIndex := make(map[string]int)
	for i, e := range x.List {
		kv, ok := e.(*bf.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := kv.Key.(*bf.StringExpr)
		if !ok {
			continue
		}
		xCaseIndex[key.Value] = i
	}

	for _, e := range y.List {
		kv, ok := e.(*bf.KeyValueExpr)
		if !ok {
			squashed.List = append(squashed.List, e)
			continue
		}
		key, ok := e.(*bf.StringExpr)
		if !ok {
			squashed.List = append(squashed.List, e)
			continue
		}
		i, ok := xCaseIndex[key.Value]
		if !ok {
			squashed.List = append(squashed.List, e)
			continue
		}
		squashedElem, err := squashExpr(x.List[i], kv.Value)
		if err != nil {
			return nil, err
		}
		x.List[i] = squashedElem
	}

	return &squashed, nil
}

// removeLegacyProto removes uses of the old proto rules. It deletes loads
// from go_proto_library.bzl. It deletes proto filegroups. It removes
// go_proto_library attributes which are no longer recognized. New rules
// are generated in place of the deleted rules, but attributes and comments
// are not migrated.
func removeLegacyProto(c *config.Config, oldFile *bf.File) *bf.File {
	// Don't fix if the proto mode was set to something other than the default.
	if c.ProtoMode != config.DefaultProtoMode {
		return oldFile
	}

	// Scan for definitions to delete.
	var deletedIndices []int
	var protoIndices []int
	shouldDeleteProtos := false
	for i, stmt := range oldFile.Stmt {
		c, ok := stmt.(*bf.CallExpr)
		if !ok {
			continue
		}
		x, ok := c.X.(*bf.LiteralExpr)
		if !ok {
			continue
		}

		if x.Token == "load" && len(c.List) > 0 {
			if name, ok := c.List[0].(*bf.StringExpr); ok && name.Value == "@io_bazel_rules_go//proto:go_proto_library.bzl" {
				deletedIndices = append(deletedIndices, i)
				shouldDeleteProtos = true
			}
			continue
		}
		if x.Token == "filegroup" {
			r := bf.Rule{Call: c}
			if r.Name() == config.DefaultProtosName {
				deletedIndices = append(deletedIndices, i)
			}
			continue
		}
		if x.Token == "go_proto_library" {
			protoIndices = append(protoIndices, i)
		}
	}
	if len(deletedIndices) == 0 {
		return oldFile
	}

	// Rebuild the file without deleted statements. Only delete go_proto_library
	// rules if we deleted a load.
	if shouldDeleteProtos {
		deletedIndices = append(deletedIndices, protoIndices...)
		sort.Ints(deletedIndices)
	}
	fixedFile := *oldFile
	fixedFile.Stmt = deleteIndices(oldFile.Stmt, deletedIndices)
	return &fixedFile
}

// FixFileMinor updates rules in oldFile that were generated by an older version
// of Gazelle to a newer form that can be merged with freshly generated rules.
//
// FixFileMinor includes only small, low-risk fixes that can be applied in
// update mode. When both FixFileMinor and FixFile are called, FixFileMinor
// should be called first.
//
// FixLoads should be called after this, since it will fix load
// statements that may be broken by transformations applied by this function.
func FixFileMinor(c *config.Config, oldFile *bf.File) *bf.File {
	fixedFile := migrateLibraryEmbed(c, oldFile)
	fixedFile = migrateGrpcCompilers(c, fixedFile)
	return removeBinaryImportPath(c, fixedFile)
}

// migrateLibraryEmbed converts "library" attributes to "embed" attributes,
// preserving comments. This only applies to Go rules, and only if there is
// no keep comment on "library" and no existing "embed" attribute.
func migrateLibraryEmbed(c *config.Config, oldFile *bf.File) *bf.File {
	fixed := false
	fixedFile := *oldFile
	for i, stmt := range fixedFile.Stmt {
		call, ok := stmt.(*bf.CallExpr)
		if !ok {
			continue
		}
		rule := bf.Rule{Call: call}
		if kind := rule.Kind(); !isGoRule(kind) || shouldKeep(stmt) {
			continue
		}
		libExpr := rule.Attr("library")
		if libExpr == nil || shouldKeep(libExpr) || rule.Attr("embed") != nil {
			continue
		}

		fixedCall := *call
		rule.Call = &fixedCall
		rule.DelAttr("library")
		rule.SetAttr("embed", &bf.ListExpr{List: []bf.Expr{libExpr}})
		fixedFile.Stmt[i] = &fixedCall
		fixed = true
	}
	if !fixed {
		return oldFile
	}
	return &fixedFile
}

// migrateGrpcCompilers converts "go_grpc_library" rules into "go_proto_library"
// rules with a "compilers" attribute.
func migrateGrpcCompilers(c *config.Config, oldFile *bf.File) *bf.File {
	fixed := false
	fixedFile := *oldFile
	for i, stmt := range fixedFile.Stmt {
		call, ok := stmt.(*bf.CallExpr)
		if !ok {
			continue
		}
		rule := bf.Rule{Call: call}
		if rule.Kind() != "go_grpc_library" || shouldKeep(stmt) || rule.Attr("compilers") != nil {
			continue
		}

		fixedCall := *call
		fixedCall.List = make([]bf.Expr, len(call.List))
		copy(fixedCall.List, call.List)
		rule.Call = &fixedCall
		rule.SetKind("go_proto_library")
		rule.SetAttr("compilers", &bf.ListExpr{
			List: []bf.Expr{&bf.StringExpr{Value: config.GrpcCompilerLabel}},
		})
		fixedFile.Stmt[i] = &fixedCall
		fixed = true
	}
	if !fixed {
		return oldFile
	}
	return &fixedFile
}

// removeBinaryImportPath removes "importpath" attributes from "go_binary"
// and "go_test" rules. These are now deprecated.
func removeBinaryImportPath(c *config.Config, oldFile *bf.File) *bf.File {
	fixed := false
	fixedFile := *oldFile
	for i, stmt := range fixedFile.Stmt {
		call, ok := stmt.(*bf.CallExpr)
		if !ok {
			continue
		}
		rule := bf.Rule{Call: call}
		if rule.Kind() != "go_binary" && rule.Kind() != "go_test" || rule.Attr("importpath") == nil {
			continue
		}

		fixedCall := *call
		fixedCall.List = make([]bf.Expr, len(call.List))
		copy(fixedCall.List, call.List)
		rule.Call = &fixedCall
		rule.DelAttr("importpath")
		fixedFile.Stmt[i] = &fixedCall
		fixed = true
	}
	if !fixed {
		return oldFile
	}
	return &fixedFile
}

// FixLoads removes loads of unused go rules and adds loads of newly used rules.
// This should be called after FixFile and MergeWithExisting, since symbols
// may be introduced that aren't loaded.
func FixLoads(oldFile *bf.File) *bf.File {
	// Make a list of load statements in the file. Keep track of loads of known
	// files, since these may be changed. Keep track of known symbols loaded from
	// unknown files; we will not add loads for these.
	type loadInfo struct {
		index      int
		file       string
		old, fixed *bf.CallExpr
	}
	var loads []loadInfo
	otherLoadedKinds := make(map[string]bool)
	for i, stmt := range oldFile.Stmt {
		c, ok := stmt.(*bf.CallExpr)
		if !ok {
			continue
		}
		x, ok := c.X.(*bf.LiteralExpr)
		if !ok || x.Token != "load" {
			continue
		}

		if len(c.List) == 0 {
			continue
		}
		label, ok := c.List[0].(*bf.StringExpr)
		if !ok {
			continue
		}

		if knownFiles[label.Value] {
			loads = append(loads, loadInfo{index: i, file: label.Value, old: c})
			continue
		}
		for _, arg := range c.List[1:] {
			switch sym := arg.(type) {
			case *bf.StringExpr:
				otherLoadedKinds[sym.Value] = true
			case *bf.BinaryExpr:
				if sym.Op != "=" {
					continue
				}
				if x, ok := sym.X.(*bf.LiteralExpr); ok {
					otherLoadedKinds[x.Token] = true
				}
			}
		}
	}

	// Make a map of all the symbols from known files used in this file.
	usedKinds := make(map[string]map[string]bool)
	for _, stmt := range oldFile.Stmt {
		c, ok := stmt.(*bf.CallExpr)
		if !ok {
			continue
		}
		x, ok := c.X.(*bf.LiteralExpr)
		if !ok {
			continue
		}

		kind := x.Token
		if file, ok := knownKinds[kind]; ok && !otherLoadedKinds[kind] {
			if usedKinds[file] == nil {
				usedKinds[file] = make(map[string]bool)
			}
			usedKinds[file][kind] = true
		}
	}

	// Fix the load statements. The order is important, so we iterate over
	// knownLoads instead of knownFiles.
	changed := false
	var newFirstLoads []*bf.CallExpr
	for _, l := range knownLoads {
		file := l.file
		first := true
		for i, _ := range loads {
			li := &loads[i]
			if li.file != file {
				continue
			}
			if first {
				li.fixed = fixLoad(li.old, file, usedKinds[file])
				first = false
			} else {
				li.fixed = fixLoad(li.old, file, nil)
			}
			changed = changed || li.fixed != li.old
		}
		if first {
			load := fixLoad(nil, file, usedKinds[file])
			if load != nil {
				newFirstLoads = append(newFirstLoads, load)
				changed = true
			}
		}
	}
	if !changed {
		return oldFile
	}

	// Rebuild the file.
	fixedFile := *oldFile
	fixedFile.Stmt = make([]bf.Expr, 0, len(oldFile.Stmt)+len(newFirstLoads))
	for _, l := range newFirstLoads {
		fixedFile.Stmt = append(fixedFile.Stmt, l)
	}
	loadIndex := 0
	for i, stmt := range oldFile.Stmt {
		if loadIndex < len(loads) && i == loads[loadIndex].index {
			if loads[loadIndex].fixed != nil {
				fixedFile.Stmt = append(fixedFile.Stmt, loads[loadIndex].fixed)
			}
			loadIndex++
			continue
		}
		fixedFile.Stmt = append(fixedFile.Stmt, stmt)
	}
	return &fixedFile
}

// knownLoads is a list of files Gazelle will generate loads from and
// the symbols it knows about.  All symbols Gazelle ever generated
// loads for are present, including symbols it no longer uses (e.g.,
// cgo_library). Manually loaded symbols (e.g., go_embed_data) are not
// included. The order of the files here will match the order of
// generated load statements. The symbols should be sorted
// lexicographically.
var knownLoads = []struct {
	file  string
	kinds []string
}{
	{
		"@io_bazel_rules_go//go:def.bzl",
		[]string{
			"cgo_library",
			"go_binary",
			"go_library",
			"go_prefix",
			"go_repository",
			"go_test",
		},
	}, {
		"@io_bazel_rules_go//proto:def.bzl",
		[]string{
			"go_grpc_library",
			"go_proto_library",
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

// fixLoad updates a load statement. load must be a load statement for
// the Go rules or nil. If nil, a new statement may be created. Symbols in
// kinds are added if they are not already present, symbols in knownKinds
// are removed if they are not in kinds, and other symbols and arguments
// are preserved. nil is returned if the statement should be deleted because
// it is empty.
func fixLoad(load *bf.CallExpr, file string, kinds map[string]bool) *bf.CallExpr {
	var fixed bf.CallExpr
	if load == nil {
		fixed = bf.CallExpr{
			X: &bf.LiteralExpr{Token: "load"},
			List: []bf.Expr{
				&bf.StringExpr{Value: file},
			},
			ForceCompact: true,
		}
	} else {
		fixed = *load
	}

	var symbols []*bf.StringExpr
	var otherArgs []bf.Expr
	loadedKinds := make(map[string]bool)
	var added, removed int
	for _, arg := range fixed.List[1:] {
		if s, ok := arg.(*bf.StringExpr); ok {
			if knownKinds[s.Value] == "" || kinds != nil && kinds[s.Value] {
				symbols = append(symbols, s)
				loadedKinds[s.Value] = true
			} else {
				removed++
			}
		} else {
			otherArgs = append(otherArgs, arg)
		}
	}
	if kinds != nil {
		for kind, _ := range kinds {
			if _, ok := loadedKinds[kind]; !ok {
				symbols = append(symbols, &bf.StringExpr{Value: kind})
				added++
			}
		}
	}
	if added == 0 && removed == 0 {
		if load != nil && len(load.List) == 1 {
			// Special case: delete existing empty load.
			return nil
		}
		return load
	}

	sort.Stable(byString(symbols))
	fixed.List = fixed.List[:1]
	for _, sym := range symbols {
		fixed.List = append(fixed.List, sym)
	}
	fixed.List = append(fixed.List, otherArgs...)
	if len(fixed.List) == 1 {
		return nil
	}
	return &fixed
}

type byString []*bf.StringExpr

func (s byString) Len() int {
	return len(s)
}

func (s byString) Less(i, j int) bool {
	return s[i].Value < s[j].Value
}

func (s byString) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func isGoRule(kind string) bool {
	return kind == "go_library" ||
		kind == "go_binary" ||
		kind == "go_test" ||
		kind == "go_proto_library" ||
		kind == "go_grpc_library"
}
