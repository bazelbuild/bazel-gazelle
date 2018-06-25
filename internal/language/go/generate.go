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
	"fmt"
	"go/build"
	"log"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/bazelbuild/bazel-gazelle/internal/config"
	"github.com/bazelbuild/bazel-gazelle/internal/language/proto"
	"github.com/bazelbuild/bazel-gazelle/internal/pathtools"
	"github.com/bazelbuild/bazel-gazelle/internal/rule"
)

func (gl *goLang) GenerateRules(c *config.Config, dir, rel string, f *rule.File, subdirs, regularFiles, genFiles []string, other []*rule.Rule) (empty, gen []*rule.Rule) {
	// Extract information about proto files. We need this to exclude .pb.go
	// files and generate go_proto_library rules.
	pc := proto.GetProtoConfig(c)
	var protoName string
	var protoFileInfo map[string]proto.FileInfo
	if pc.Mode != proto.DisableMode {
		protoFileInfo = make(map[string]proto.FileInfo)
		for _, r := range other {
			if r.Kind() != "proto_library" {
				continue
			}
			if protoName != "" {
				// TODO(jayconrod): Currently, the proto extension generates at most one
				// proto_library rule because this extension can't handle multiple
				// packages. We should remove this limitation soon.
				log.Panicf("%s: cannot generate Go rules for multiple proto_library rules", dir)
			}
			protoName = r.Name()
			for _, pfi := range r.PrivateAttr(proto.FileInfoKey).([]proto.FileInfo) {
				protoFileInfo[pfi.Name] = pfi
			}
		}
	}

	// If proto rule generation is enabled, exclude .pb.go files that correspond
	// to any .proto files present.
	if pc.Mode != proto.DisableMode {
		keep := func(f string) bool {
			if strings.HasSuffix(f, ".pb.go") {
				_, ok := protoFileInfo[strings.TrimSuffix(f, ".pb.go")+".proto"]
				return !ok
			}
			return true
		}
		filterFiles(&regularFiles, keep)
		filterFiles(&genFiles, keep)
	}

	// Split regular files into files which can determine the package name and
	// import path and other files.
	var pkgFiles, otherFiles []string
	for _, f := range regularFiles {
		if strings.HasSuffix(f, ".go") ||
			pc.Mode != proto.DisableMode && strings.HasSuffix(f, ".proto") {
			pkgFiles = append(pkgFiles, f)
		} else {
			otherFiles = append(otherFiles, f)
		}
	}

	// Look for a subdirectory named testdata. Only treat it as data if it does
	// not contain a buildable package.
	var hasTestdata bool
	for _, sub := range subdirs {
		if sub == "testdata" {
			isPkg, ok := gl.testdataPkgs[path.Join(rel, sub)]
			hasTestdata = !ok || !isPkg
			break
		}
	}

	// Build a package from files in this directory.
	pkg := buildPackage(c, dir, rel, pkgFiles, otherFiles, genFiles, hasTestdata, protoName, protoFileInfo)
	if pkg != nil && path.Base(rel) == "testdata" {
		gl.testdataPkgs[rel] = true
	}
	if pkg == nil {
		pkg = emptyPackage(c, dir, rel)
	}

	g := newGenerator(c, f, rel)
	return g.generateRules(pkg)
}

func filterFiles(files *[]string, pred func(string) bool) {
	w := 0
	for r := 0; r < len(*files); r++ {
		f := (*files)[r]
		if pred(f) {
			(*files)[w] = f
			w++
		}
	}
	*files = (*files)[:w]
}

// buildPackage reads source files in a given directory and returns a goPackage
// containing information about those files and how to build them.
//
// If no buildable .go files are found in the directory, nil will be returned.
// If the directory contains multiple buildable packages, the package whose
// name matches the directory base name will be returned. If there is no such
// package or if an error occurs, an error will be logged, and nil will be
// returned.
func buildPackage(c *config.Config, dir, rel string, pkgFiles, otherFiles, genFiles []string, hasTestdata bool, protoName string, protoInfo map[string]proto.FileInfo) *goPackage {
	// Process .go and .proto files first, since these determine the package name.
	packageMap := make(map[string]*goPackage)
	cgo := false
	var pkgFilesWithUnknownPackage []fileInfo
	for _, f := range pkgFiles {
		path := filepath.Join(dir, f)
		var info fileInfo
		if strings.HasSuffix(f, ".go") {
			info = goFileInfo(path, rel)
		} else {
			info = protoFileInfo(path, protoInfo[f])
		}
		if info.packageName == "" {
			pkgFilesWithUnknownPackage = append(pkgFilesWithUnknownPackage, info)
			continue
		}
		if info.packageName == "documentation" {
			// go/build ignores this package
			continue
		}

		cgo = cgo || info.isCgo

		if _, ok := packageMap[info.packageName]; !ok {
			packageMap[info.packageName] = &goPackage{
				name:        info.packageName,
				dir:         dir,
				rel:         rel,
				hasTestdata: hasTestdata,
			}
		}
		if err := packageMap[info.packageName].addFile(c, info, false); err != nil {
			log.Print(err)
		}
	}

	// Select a package to generate rules for.
	pkg, err := selectPackage(c, dir, packageMap)
	if err != nil {
		if _, ok := err.(*build.NoGoError); !ok {
			log.Print(err)
		}
		return nil
	}

	// Add files with unknown packages. This happens when there are parse
	// or I/O errors. We should keep the file in the srcs list and let the
	// compiler deal with the error.
	for _, info := range pkgFilesWithUnknownPackage {
		if err := pkg.addFile(c, info, cgo); err != nil {
			log.Print(err)
		}
	}

	// Process the other static files.
	for _, file := range otherFiles {
		info := otherFileInfo(filepath.Join(dir, file))
		if err := pkg.addFile(c, info, cgo); err != nil {
			log.Print(err)
		}
	}

	// Process generated files. Note that generated files may have the same names
	// as static files. Bazel will use the generated files, but we will look at
	// the content of static files, assuming they will be the same.
	staticFiles := make(map[string]bool)
	for _, f := range pkgFiles {
		staticFiles[f] = true
	}
	for _, f := range otherFiles {
		staticFiles[f] = true
	}
	for _, f := range genFiles {
		if staticFiles[f] {
			continue
		}
		info := fileNameInfo(filepath.Join(dir, f))
		if err := pkg.addFile(c, info, cgo); err != nil {
			log.Print(err)
		}
	}

	if pkg.importPath == "" {
		if err := pkg.inferImportPath(c); err != nil {
			inferImportPathErrorOnce.Do(func() { log.Print(err) })
			return nil
		}
	}
	pkg.proto.name = protoName
	return pkg
}

var inferImportPathErrorOnce sync.Once

// selectPackages selects one Go packages out of the buildable packages found
// in a directory. If multiple packages are found, it returns the package
// whose name matches the directory if such a package exists.
func selectPackage(c *config.Config, dir string, packageMap map[string]*goPackage) (*goPackage, error) {
	buildablePackages := make(map[string]*goPackage)
	for name, pkg := range packageMap {
		if pkg.isBuildable(c) {
			buildablePackages[name] = pkg
		}
	}

	if len(buildablePackages) == 0 {
		return nil, &build.NoGoError{Dir: dir}
	}

	if len(buildablePackages) == 1 {
		for _, pkg := range buildablePackages {
			return pkg, nil
		}
	}

	if pkg, ok := buildablePackages[defaultPackageName(c, dir)]; ok {
		return pkg, nil
	}

	err := &build.MultiplePackageError{Dir: dir}
	for name, pkg := range buildablePackages {
		// Add the first file for each package for the error message.
		// Error() method expects these lists to be the same length. File
		// lists must be non-empty. These lists are only created by
		// buildPackage for packages with .go files present.
		err.Packages = append(err.Packages, name)
		err.Files = append(err.Files, pkg.firstGoFile())
	}
	return nil, err
}

func emptyPackage(c *config.Config, dir, rel string) *goPackage {
	pkg := &goPackage{
		name: defaultPackageName(c, dir),
		dir:  dir,
		rel:  rel,
	}
	pkg.inferImportPath(c)
	return pkg
}

func defaultPackageName(c *config.Config, rel string) string {
	gc := getGoConfig(c)
	return pathtools.RelBaseName(rel, gc.prefix, "")
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

type generator struct {
	c                   *config.Config
	rel                 string
	shouldSetVisibility bool
}

func newGenerator(c *config.Config, f *rule.File, rel string) *generator {
	shouldSetVisibility := f == nil || !hasDefaultVisibility(f)
	return &generator{c: c, rel: rel, shouldSetVisibility: shouldSetVisibility}
}

func (g *generator) generateRules(pkg *goPackage) (empty, gen []*rule.Rule) {
	protoMode := proto.GetProtoConfig(g.c).Mode
	protoEmbed, rules := g.generateProto(protoMode, pkg)
	var libName string
	lib := g.generateLib(pkg, protoEmbed)
	rules = append(rules, lib)
	if !lib.IsEmpty(goKinds[lib.Kind()]) {
		libName = lib.Name()
	}
	rules = append(rules,
		g.generateBin(pkg, libName),
		g.generateTest(pkg, libName))
	for _, r := range rules {
		if !r.IsEmpty(goKinds[r.Kind()]) {
			gen = append(gen, r)
		} else {
			empty = append(empty, r)
		}
	}
	return empty, gen
}

func (g *generator) generateProto(mode proto.Mode, pkg *goPackage) (string, []*rule.Rule) {
	if mode == proto.DisableMode {
		// Don't create or delete proto rules in this mode. Any existing rules
		// are likely hand-written.
		return "", nil
	}

	filegroupName := config.DefaultProtosName
	protoName := pkg.proto.name
	if protoName == "" {
		protoName = proto.RuleName("", g.rel, getGoConfig(g.c).prefix)
	}
	goProtoName := strings.TrimSuffix(protoName, "_proto") + "_go_proto"
	visibility := []string{checkInternalVisibility(pkg.rel, "//visibility:public")}

	if mode == proto.LegacyMode {
		filegroup := rule.NewRule("filegroup", filegroupName)
		if pkg.proto.sources.isEmpty() {
			return "", []*rule.Rule{filegroup}
		}
		filegroup.SetAttr("srcs", pkg.proto.sources.build())
		if g.shouldSetVisibility {
			filegroup.SetAttr("visibility", visibility)
		}
		return "", []*rule.Rule{filegroup}
	}

	if pkg.proto.sources.isEmpty() {
		return "", []*rule.Rule{
			rule.NewRule("filegroup", filegroupName),
			rule.NewRule("go_proto_library", goProtoName),
		}
	}

	goProtoLibrary := rule.NewRule("go_proto_library", goProtoName)
	goProtoLibrary.SetAttr("proto", ":"+protoName)
	g.setImportAttrs(goProtoLibrary, pkg)
	if pkg.proto.hasServices {
		goProtoLibrary.SetAttr("compilers", []string{"@io_bazel_rules_go//proto:go_grpc"})
	}
	if g.shouldSetVisibility {
		goProtoLibrary.SetAttr("visibility", visibility)
	}
	goProtoLibrary.SetPrivateAttr(config.GazelleImportsKey, pkg.proto.imports.build())
	return goProtoName, []*rule.Rule{goProtoLibrary}
}

func (g *generator) generateLib(pkg *goPackage, embed string) *rule.Rule {
	goLibrary := rule.NewRule("go_library", config.DefaultLibName)
	if !pkg.library.sources.hasGo() && embed == "" {
		return goLibrary // empty
	}
	var visibility string
	if pkg.isCommand() {
		// Libraries made for a go_binary should not be exposed to the public.
		visibility = "//visibility:private"
	} else {
		visibility = checkInternalVisibility(pkg.rel, "//visibility:public")
	}
	g.setCommonAttrs(goLibrary, pkg.rel, visibility, pkg.library, embed)
	g.setImportAttrs(goLibrary, pkg)
	return goLibrary
}

func (g *generator) generateBin(pkg *goPackage, library string) *rule.Rule {
	name := pathtools.RelBaseName(pkg.rel, getGoConfig(g.c).prefix, g.c.RepoRoot)
	goBinary := rule.NewRule("go_binary", name)
	if !pkg.isCommand() || pkg.binary.sources.isEmpty() && library == "" {
		return goBinary // empty
	}
	visibility := checkInternalVisibility(pkg.rel, "//visibility:public")
	g.setCommonAttrs(goBinary, pkg.rel, visibility, pkg.binary, library)
	return goBinary
}

func (g *generator) generateTest(pkg *goPackage, library string) *rule.Rule {
	goTest := rule.NewRule("go_test", config.DefaultTestName)
	if !pkg.test.sources.hasGo() {
		return goTest // empty
	}
	g.setCommonAttrs(goTest, pkg.rel, "", pkg.test, library)
	if pkg.hasTestdata {
		goTest.SetAttr("data", rule.GlobValue{Patterns: []string{"testdata/**"}})
	}
	return goTest
}

func (g *generator) setCommonAttrs(r *rule.Rule, pkgRel, visibility string, target goTarget, embed string) {
	if !target.sources.isEmpty() {
		r.SetAttr("srcs", target.sources.buildFlat())
	}
	if target.cgo {
		r.SetAttr("cgo", true)
	}
	if !target.clinkopts.isEmpty() {
		r.SetAttr("clinkopts", g.options(target.clinkopts.build(), pkgRel))
	}
	if !target.copts.isEmpty() {
		r.SetAttr("copts", g.options(target.copts.build(), pkgRel))
	}
	if g.shouldSetVisibility && visibility != "" {
		r.SetAttr("visibility", []string{visibility})
	}
	if embed != "" {
		r.SetAttr("embed", []string{":" + embed})
	}
	r.SetPrivateAttr(config.GazelleImportsKey, target.imports.build())
}

func (g *generator) setImportAttrs(r *rule.Rule, pkg *goPackage) {
	r.SetAttr("importpath", pkg.importPath)
	goConf := getGoConfig(g.c)
	if goConf.importMapPrefix != "" {
		fromPrefixRel := pathtools.TrimPrefix(pkg.rel, goConf.importMapPrefixRel)
		importMap := path.Join(goConf.importMapPrefix, fromPrefixRel)
		if importMap != pkg.importPath {
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
func (g *generator) options(opts rule.PlatformStrings, pkgRel string) rule.PlatformStrings {
	fixPath := func(opt string) string {
		if strings.HasPrefix(opt, "/") {
			return opt
		}
		return path.Clean(path.Join(pkgRel, opt))
	}

	fixGroups := func(groups []string) ([]string, error) {
		fixedGroups := make([]string, len(groups))
		for i, group := range groups {
			opts := strings.Split(group, optSeparator)
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
