/* Copyright 2016 The Bazel Authors. All rights reserved.

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

package generator

import (
	"fmt"
	"log"
	"path"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/internal/config"
	"github.com/bazelbuild/bazel-gazelle/internal/label"
	"github.com/bazelbuild/bazel-gazelle/internal/merger"
	"github.com/bazelbuild/bazel-gazelle/internal/packages"
	"github.com/bazelbuild/bazel-gazelle/internal/pathtools"
	"github.com/bazelbuild/bazel-gazelle/internal/rule"
)

// NewGenerator returns a new instance of Generator.
// "oldFile" is the existing build file. May be nil.
func NewGenerator(c *config.Config, l *label.Labeler, oldFile *rule.File) *Generator {
	shouldSetVisibility := oldFile == nil || !hasDefaultVisibility(oldFile)
	return &Generator{c: c, l: l, shouldSetVisibility: shouldSetVisibility}
}

// Generator generates Bazel build rules for Go build targets.
type Generator struct {
	c                   *config.Config
	l                   *label.Labeler
	shouldSetVisibility bool
}

// GenerateRules generates a list of rules for targets in "pkg". It also returns
// a list of empty rules that may be deleted from an existing file.
func (g *Generator) GenerateRules(pkg *packages.Package) (gen, empty []*rule.Rule) {
	var rs []*rule.Rule
	protoLibName, protoRules := g.generateProto(pkg)
	rs = append(rs, protoRules...)

	libName, libRule := g.generateLib(pkg, protoLibName)
	rs = append(rs, libRule)

	rs = append(rs,
		g.generateBin(pkg, libName),
		g.generateTest(pkg, libName))

	for _, r := range rs {
		// TODO(jayconrod): don't depend on merger package. Get NonEmptyAttrs from
		// the Language interface when that's introduced.
		if r.IsEmpty(merger.NonEmptyAttrs) {
			empty = append(empty, r)
		} else {
			gen = append(gen, r)
		}
	}

	return gen, empty
}

func (g *Generator) generateProto(pkg *packages.Package) (string, []*rule.Rule) {
	if g.c.ProtoMode == config.DisableProtoMode {
		// Don't create or delete proto rules in this mode. Any existing rules
		// are likely hand-written.
		return "", nil
	}

	filegroupName := config.DefaultProtosName
	protoName := g.l.ProtoLabel(pkg.Rel, pkg.Name).Name
	goProtoName := g.l.GoProtoLabel(pkg.Rel, pkg.Name).Name

	if g.c.ProtoMode == config.LegacyProtoMode {
		filegroup := rule.NewRule("filegroup", filegroupName)
		if !pkg.Proto.HasProto() {
			return "", []*rule.Rule{filegroup}
		}
		filegroup.SetAttr("srcs", pkg.Proto.Sources)
		if g.shouldSetVisibility {
			filegroup.SetAttr("visibility", []string{checkInternalVisibility(pkg.Rel, "//visibility:public")})
		}
		return "", []*rule.Rule{filegroup}
	}

	if !pkg.Proto.HasProto() {
		return "", []*rule.Rule{
			rule.NewRule("filegroup", filegroupName),
			rule.NewRule("proto_library", protoName),
			rule.NewRule("go_proto_library", goProtoName),
		}
	}

	visibility := []string{checkInternalVisibility(pkg.Rel, "//visibility:public")}
	protoLibrary := rule.NewRule("proto_library", protoName)
	protoLibrary.SetAttr("srcs", pkg.Proto.Sources)
	if g.shouldSetVisibility {
		protoLibrary.SetAttr("visibility", visibility)
	}
	imports := pkg.Proto.Imports
	if !imports.IsEmpty() {
		protoLibrary.SetPrivateAttr(config.GazelleImportsKey, imports)
	}

	goProtoLibrary := rule.NewRule("go_proto_library", goProtoName)
	goProtoLibrary.SetAttr("proto", ":"+protoName)

	g.setImportAttrs(goProtoLibrary, pkg)
	if pkg.Proto.HasServices {
		goProtoLibrary.SetAttr("compilers", []string{"@io_bazel_rules_go//proto:go_grpc"})
	}
	if g.shouldSetVisibility {
		goProtoLibrary.SetAttr("visibility", visibility)
	}
	if !imports.IsEmpty() {
		goProtoLibrary.SetPrivateAttr(config.GazelleImportsKey, imports)
	}

	return goProtoName, []*rule.Rule{protoLibrary, goProtoLibrary}
}

func (g *Generator) generateBin(pkg *packages.Package, library string) *rule.Rule {
	name := g.l.BinaryLabel(pkg.Rel).Name
	goBinary := rule.NewRule("go_binary", name)
	if !pkg.IsCommand() || pkg.Binary.Sources.IsEmpty() && library == "" {
		return goBinary // empty
	}
	visibility := checkInternalVisibility(pkg.Rel, "//visibility:public")
	g.setCommonAttrs(goBinary, pkg.Rel, visibility, pkg.Binary)
	if library != "" {
		goBinary.SetAttr("embed", []string{":" + library})
	}
	return goBinary
}

func (g *Generator) generateLib(pkg *packages.Package, goProtoName string) (string, *rule.Rule) {
	name := g.l.LibraryLabel(pkg.Rel).Name
	goLibrary := rule.NewRule("go_library", name)
	if !pkg.Library.HasGo() && goProtoName == "" {
		return "", goLibrary // empty
	}
	var visibility string
	if pkg.IsCommand() {
		// Libraries made for a go_binary should not be exposed to the public.
		visibility = "//visibility:private"
	} else {
		visibility = checkInternalVisibility(pkg.Rel, "//visibility:public")
	}

	g.setCommonAttrs(goLibrary, pkg.Rel, visibility, pkg.Library)
	g.setImportAttrs(goLibrary, pkg)
	if goProtoName != "" {
		goLibrary.SetAttr("embed", []string{":" + goProtoName})
	}
	return name, goLibrary
}

// hasDefaultVisibility returns whether oldFile contains a "package" rule with
// a "default_visibility" attribute. Rules generated by Gazelle should not
// have their own visibility attributes if this is the case.
func hasDefaultVisibility(oldFile *rule.File) bool {
	for _, r := range oldFile.Rules {
		if r.Kind() == "package" && r.Attr("default_visibility") != nil {
			return true
		}
	}
	return false
}

// checkInternalVisibility overrides the given visibility if the package is
// internal.
func checkInternalVisibility(rel, visibility string) string {
	if i := strings.LastIndex(rel, "/internal/"); i >= 0 {
		visibility = fmt.Sprintf("//%s:__subpackages__", rel[:i])
	} else if strings.HasPrefix(rel, "internal/") {
		visibility = "//:__subpackages__"
	}
	return visibility
}

func (g *Generator) generateTest(pkg *packages.Package, library string) *rule.Rule {
	name := g.l.TestLabel(pkg.Rel).Name
	goTest := rule.NewRule("go_test", name)
	if !pkg.Test.HasGo() {
		return goTest // empty
	}
	g.setCommonAttrs(goTest, pkg.Rel, "", pkg.Test)
	if library != "" {
		goTest.SetAttr("embed", []string{":" + library})
	}
	if pkg.HasTestdata {
		goTest.SetAttr("data", rule.GlobValue{Patterns: []string{"testdata/**"}})
	}
	return goTest
}

func (g *Generator) setCommonAttrs(r *rule.Rule, pkgRel, visibility string, target packages.GoTarget) {
	if !target.Sources.IsEmpty() {
		r.SetAttr("srcs", target.Sources.Flat())
	}
	if target.Cgo {
		r.SetAttr("cgo", true)
	}
	if !target.CLinkOpts.IsEmpty() {
		r.SetAttr("clinkopts", g.options(target.CLinkOpts, pkgRel))
	}
	if !target.COpts.IsEmpty() {
		r.SetAttr("copts", g.options(target.COpts, pkgRel))
	}
	if g.shouldSetVisibility && visibility != "" {
		r.SetAttr("visibility", []string{visibility})
	}
	imports := target.Imports
	if !imports.IsEmpty() {
		r.SetPrivateAttr(config.GazelleImportsKey, imports)
	}
}

func (g *Generator) setImportAttrs(r *rule.Rule, pkg *packages.Package) {
	r.SetAttr("importpath", pkg.ImportPath)
	if g.c.GoImportMapPrefix != "" {
		fromPrefixRel := pathtools.TrimPrefix(pkg.Rel, g.c.GoImportMapPrefixRel)
		importMap := path.Join(g.c.GoImportMapPrefix, fromPrefixRel)
		if importMap != pkg.ImportPath {
			r.SetAttr("importmap", importMap)
		}
	}
}

var (
	// shortOptPrefixes are strings that come at the beginning of an option
	// argument that includes a path, e.g., -Ifoo/bar.
	shortOptPrefixes = []string{"-I", "-L", "-F"}

	// longOptPrefixes are separate arguments that come before a path argument,
	// e.g., -iquote foo/bar.
	longOptPrefixes = []string{"-I", "-L", "-F", "-iquote", "-isystem"}
)

// options transforms package-relative paths in cgo options into repository-
// root-relative paths that Bazel can understand. For example, if a cgo file
// in //foo declares an include flag in its copts: "-Ibar", this method
// will transform that flag into "-Ifoo/bar".
func (g *Generator) options(opts rule.PlatformStrings, pkgRel string) rule.PlatformStrings {
	fixPath := func(opt string) string {
		if strings.HasPrefix(opt, "/") {
			return opt
		}
		return path.Clean(path.Join(pkgRel, opt))
	}

	fixGroups := func(groups []string) ([]string, error) {
		fixedGroups := make([]string, len(groups))
		for i, group := range groups {
			opts := strings.Split(group, packages.OptSeparator)
			fixedOpts := make([]string, len(opts))
			isPath := false
			for j, opt := range opts {
				if isPath {
					opt = fixPath(opt)
					isPath = false
					goto next
				}

				for _, short := range shortOptPrefixes {
					if strings.HasPrefix(opt, short) && len(opt) > len(short) {
						opt = short + fixPath(opt[len(short):])
						goto next
					}
				}

				for _, long := range longOptPrefixes {
					if opt == long {
						isPath = true
						goto next
					}
				}

			next:
				fixedOpts[j] = escapeOption(opt)
			}
			fixedGroups[i] = strings.Join(fixedOpts, " ")
		}

		return fixedGroups, nil
	}

	opts, errs := opts.MapSlice(fixGroups)
	if errs != nil {
		log.Panicf("unexpected error when transforming options with pkg %q: %v", pkgRel, errs)
	}
	return opts
}

func escapeOption(opt string) string {
	return strings.NewReplacer(
		`\`, `\\`,
		`'`, `\'`,
		`"`, `\"`,
		` `, `\ `,
		"\t", "\\\t",
		"\n", "\\\n",
		"\r", "\\\r",
	).Replace(opt)
}
