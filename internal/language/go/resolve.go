/* Copyright 2018 The Bazel Authors. All rights reserved.

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
	"errors"
	"fmt"
	"go/build"
	"log"
	"path"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/internal/config"
	"github.com/bazelbuild/bazel-gazelle/internal/label"
	"github.com/bazelbuild/bazel-gazelle/internal/pathtools"
	"github.com/bazelbuild/bazel-gazelle/internal/repos"
	"github.com/bazelbuild/bazel-gazelle/internal/resolve"
	"github.com/bazelbuild/bazel-gazelle/internal/rule"
)

func (_ *goLang) Imports(_ *config.Config, r *rule.Rule, f *rule.File) []resolve.ImportSpec {
	if !isGoLibrary(r.Kind()) {
		return nil
	}
	if importPath := r.AttrString("importpath"); importPath == "" {
		return []resolve.ImportSpec{}
	} else {
		return []resolve.ImportSpec{{goName, importPath}}
	}
}

func (_ *goLang) Embeds(r *rule.Rule, from label.Label) []label.Label {
	embedStrings := r.AttrStrings("embed")
	if isGoProtoLibrary(r.Kind()) {
		embedStrings = append(embedStrings, r.AttrString("proto"))
	}
	embedLabels := make([]label.Label, 0, len(embedStrings))
	for _, s := range embedStrings {
		l, err := label.Parse(s)
		if err != nil {
			continue
		}
		l = l.Abs(from.Repo, from.Pkg)
		embedLabels = append(embedLabels, l)
	}
	return embedLabels
}

func (gl *goLang) Resolve(c *config.Config, ix *resolve.RuleIndex, rc *repos.RemoteCache, r *rule.Rule, from label.Label) {
	importsRaw := r.PrivateAttr(config.GazelleImportsKey)
	if importsRaw == nil {
		// may not be set in tests.
		return
	}
	imports := importsRaw.(rule.PlatformStrings)
	r.DelAttr("deps")
	resolve := resolveGo
	if r.Kind() == "go_proto_library" {
		resolve = resolveProto
	}
	gc := getGoConfig(c)
	deps, errs := imports.Map(func(imp string) (string, error) {
		l, err := resolve(gc, ix, rc, r, imp, from)
		if err == skipImportError {
			return "", nil
		} else if err != nil {
			return "", err
		}
		for _, embed := range gl.Embeds(r, from) {
			if embed.Equal(l) {
				return "", nil
			}
		}
		l = l.Rel(from.Repo, from.Pkg)
		return l.String(), nil
	})
	for _, err := range errs {
		log.Print(err)
	}
	if !deps.IsEmpty() {
		r.SetAttr("deps", deps)
	}
}

var (
	skipImportError = errors.New("std or self import")
	notFoundError   = errors.New("rule not found")
)

func resolveGo(gc *goConfig, ix *resolve.RuleIndex, rc *repos.RemoteCache, r *rule.Rule, imp string, from label.Label) (label.Label, error) {
	if build.IsLocalImport(imp) {
		cleanRel := path.Clean(path.Join(from.Pkg, imp))
		if build.IsLocalImport(cleanRel) {
			return label.NoLabel, fmt.Errorf("relative import path %q from %q points outside of repository", imp, from.Pkg)
		}
		imp = path.Join(gc.prefix, cleanRel)
	}

	if isStandard(imp) {
		return label.NoLabel, skipImportError
	}

	if l := resolveWellKnownGo(imp); !l.Equal(label.NoLabel) {
		return l, nil
	}

	if l, err := resolveWithIndexGo(ix, imp, from); err == nil || err == skipImportError {
		return l, err
	} else if err != notFoundError {
		return label.NoLabel, err
	}

	if pathtools.HasPrefix(imp, gc.prefix) {
		pkg := path.Join(gc.prefixRel, pathtools.TrimPrefix(imp, gc.prefix))
		return label.New("", pkg, config.DefaultLibName), nil
	}

	if gc.depMode == externalMode {
		return resolveExternal(rc, imp)
	} else {
		return resolveVendored(rc, imp)
	}
}

// isStandard returns whether a package is in the standard library.
func isStandard(imp string) bool {
	return stdPackages[imp]
}

func resolveWellKnownGo(imp string) label.Label {
	// keep in sync with @io_bazel_rules_go//proto/wkt:well_known_types.bzl
	// TODO(jayconrod): in well_known_types.bzl, write the import paths and
	// targets in a public dict. Import it here, and use it to generate this code.
	switch imp {
	case "github.com/golang/protobuf/ptypes/any",
		"github.com/golang/protobuf/ptypes/api",
		"github.com/golang/protobuf/protoc-gen-go/descriptor",
		"github.com/golang/protobuf/ptypes/duration",
		"github.com/golang/protobuf/ptypes/empty",
		"google.golang.org/genproto/protobuf/field_mask",
		"google.golang.org/genproto/protobuf/source_context",
		"github.com/golang/protobuf/ptypes/struct",
		"github.com/golang/protobuf/ptypes/timestamp",
		"github.com/golang/protobuf/ptypes/wrappers":
		return label.Label{
			Repo: config.RulesGoRepoName,
			Pkg:  config.WellKnownTypesPkg,
			Name: path.Base(imp) + "_go_proto",
		}
	case "github.com/golang/protobuf/protoc-gen-go/plugin":
		return label.Label{
			Repo: config.RulesGoRepoName,
			Pkg:  config.WellKnownTypesPkg,
			Name: "compiler_plugin_go_proto",
		}
	case "google.golang.org/genproto/protobuf/ptype":
		return label.Label{
			Repo: config.RulesGoRepoName,
			Pkg:  config.WellKnownTypesPkg,
			Name: "type_go_proto",
		}
	}
	return label.NoLabel
}

func resolveWithIndexGo(ix *resolve.RuleIndex, imp string, from label.Label) (label.Label, error) {
	matches := ix.FindRulesByImport(resolve.ImportSpec{Lang: "go", Imp: imp}, "go")
	var bestMatch resolve.FindResult
	var bestMatchIsVendored bool
	var bestMatchVendorRoot string
	var matchError error

	for _, m := range matches {
		// Apply vendoring logic for Go libraries. A library in a vendor directory
		// is only visible in the parent tree. Vendored libraries supercede
		// non-vendored libraries, and libraries closer to from.Pkg supercede
		// those further up the tree.
		isVendored := false
		vendorRoot := ""
		parts := strings.Split(m.Label.Pkg, "/")
		for i := len(parts) - 1; i >= 0; i-- {
			if parts[i] == "vendor" {
				isVendored = true
				vendorRoot = strings.Join(parts[:i], "/")
				break
			}
		}
		if isVendored {
		}
		if isVendored && !label.New(m.Label.Repo, vendorRoot, "").Contains(from) {
			// vendor directory not visible
			continue
		}
		if bestMatch.Label.Equal(label.NoLabel) || isVendored && (!bestMatchIsVendored || len(vendorRoot) > len(bestMatchVendorRoot)) {
			// Current match is better
			bestMatch = m
			bestMatchIsVendored = isVendored
			bestMatchVendorRoot = vendorRoot
			matchError = nil
		} else if (!isVendored && bestMatchIsVendored) || (isVendored && len(vendorRoot) < len(bestMatchVendorRoot)) {
			// Current match is worse
		} else {
			// Match is ambiguous
			matchError = fmt.Errorf("multiple rules (%s and %s) may be imported with %q from %s", bestMatch.Label, m.Label, imp, from)
		}
	}
	if matchError != nil {
		return label.NoLabel, matchError
	}
	if bestMatch.Label.Equal(label.NoLabel) {
		return label.NoLabel, notFoundError
	}
	if bestMatch.Label.Equal(from) {
		return label.NoLabel, skipImportError
	}
	return bestMatch.Label, nil
}

func resolveExternal(rc *repos.RemoteCache, imp string) (label.Label, error) {
	prefix, repo, err := rc.Root(imp)
	if err != nil {
		return label.NoLabel, err
	}

	var pkg string
	if imp != prefix {
		pkg = pathtools.TrimPrefix(imp, prefix)
	}

	return label.New(repo, pkg, config.DefaultLibName), nil
}

func resolveVendored(rc *repos.RemoteCache, imp string) (label.Label, error) {
	return label.New("", path.Join("vendor", imp), config.DefaultLibName), nil
}

func resolveProto(gc *goConfig, ix *resolve.RuleIndex, rc *repos.RemoteCache, r *rule.Rule, imp string, from label.Label) (label.Label, error) {
	if !strings.HasSuffix(imp, ".proto") {
		return label.NoLabel, fmt.Errorf("can't import non-proto: %q", imp)
	}
	stem := imp[:len(imp)-len(".proto")]

	if isWellKnownProto(stem) {
		return label.NoLabel, skipImportError
	}

	if l, err := resolveWithIndexProto(ix, imp, from); err == nil || err == skipImportError {
		return l, err
	} else if err != notFoundError {
		return label.NoLabel, err
	}

	// As a fallback, guess the label based on the proto file name. We assume
	// all proto files in a directory belong to the same package, and the
	// package name matches the directory base name. We also assume that protos
	// in the vendor directory must refer to something else in vendor.
	rel := path.Dir(imp)
	if rel == "." {
		rel = ""
	}
	if from.Pkg == "vendor" || strings.HasPrefix(from.Pkg, "vendor/") {
		rel = path.Join("vendor", rel)
	}
	return label.New("", rel, config.DefaultLibName), nil
}

// wellKnownProtos is the set of proto sets for which we don't need to add
// an explicit dependency in go_proto_library.
// TODO(jayconrod): generate from
// @io_bazel_rules_go//proto/wkt:WELL_KNOWN_TYPE_PACKAGES
var wellKnownProtos = map[string]bool{
	"google/protobuf/any":             true,
	"google/protobuf/api":             true,
	"google/protobuf/compiler_plugin": true,
	"google/protobuf/descriptor":      true,
	"google/protobuf/duration":        true,
	"google/protobuf/empty":           true,
	"google/protobuf/field_mask":      true,
	"google/protobuf/source_context":  true,
	"google/protobuf/struct":          true,
	"google/protobuf/timestamp":       true,
	"google/protobuf/type":            true,
	"google/protobuf/wrappers":        true,
}

func isWellKnownProto(stem string) bool {
	return wellKnownProtos[stem]
}

func resolveWithIndexProto(ix *resolve.RuleIndex, imp string, from label.Label) (label.Label, error) {
	matches := ix.FindRulesByImport(resolve.ImportSpec{Lang: "proto", Imp: imp}, "go")
	if len(matches) == 0 {
		return label.NoLabel, notFoundError
	}
	if len(matches) > 1 {
		return label.NoLabel, fmt.Errorf("multiple rules (%s and %s) may be imported with %q from %s", matches[0].Label, matches[1].Label, imp, from)
	}
	// If some go_library embeds the go_proto_library we found, use that instead.
	importpath := matches[0].Rule.AttrString("importpath")
	if l, err := resolveWithIndexGo(ix, importpath, from); err == nil {
		return l, nil
	}
	return matches[0].Label, nil
}

func isGoLibrary(kind string) bool {
	return kind == "go_library" || isGoProtoLibrary(kind)
}

func isGoProtoLibrary(kind string) bool {
	return kind == "go_proto_library" || kind == "go_grpc_library"
}
