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

package proto

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/internal/config"
	"github.com/bazelbuild/bazel-gazelle/internal/pathtools"
	"github.com/bazelbuild/bazel-gazelle/internal/rule"
)

func (_ *protoLang) GenerateRules(c *config.Config, dir, rel string, f *rule.File, subdirs, regularFiles, genFiles []string, other []*rule.Rule) (empty, gen []*rule.Rule) {
	pc := GetProtoConfig(c)
	if pc.Mode != DefaultMode {
		// Don't create or delete proto rules in this mode. Any existing rules
		// are likely hand-written.
		return nil, nil
	}

	var regularProtoFiles []string
	for _, name := range regularFiles {
		if strings.HasSuffix(name, ".proto") {
			regularProtoFiles = append(regularProtoFiles, name)
		}
	}
	var genProtoFiles []string
	for _, name := range genFiles {
		if strings.HasSuffix(name, ".proto") {
			genProtoFiles = append(genFiles, name)
		}
	}
	pkg := buildPackage(dir, rel, regularProtoFiles, genProtoFiles)

	if pkg == nil {
		empty = append(empty, rule.NewRule("proto_library", RuleName("", rel, pc.GoPrefix)))
		return empty, gen
	}

	name := RuleName(goPackageName(pkg), rel, pc.GoPrefix)
	r := rule.NewRule("proto_library", name)
	srcs := make([]string, 0, len(pkg.files))
	for f := range pkg.files {
		srcs = append(srcs, f)
	}
	sort.Strings(srcs)
	r.SetAttr("srcs", srcs)
	info := make([]FileInfo, len(srcs))
	for i, src := range srcs {
		info[i] = pkg.files[src]
	}
	r.SetPrivateAttr(FileInfoKey, info)
	imports := make([]string, 0, len(pkg.imports))
	for i := range pkg.imports {
		imports = append(imports, i)
	}
	sort.Strings(imports)
	r.SetPrivateAttr(config.GazelleImportsKey, imports)
	for k, v := range pkg.options {
		r.SetPrivateAttr(k, v)
	}
	if !hasDefaultVisibility(f) {
		vis := checkInternalVisibility(rel, "//visibility:public")
		r.SetAttr("visibility", []string{vis})
	}
	gen = append(gen, r)
	return empty, gen
}

// RuleName returns a name for the proto_library in the given directory.
// TODO(jayconrod): remove Go-specific functionality. This is here temporarily
// for compatibility.
func RuleName(goPkgName, rel, goPrefix string) string {
	base := goPkgName
	if base == "" {
		base = pathtools.RelBaseName(rel, goPrefix, "")
	}
	return base + "_proto"
}

// buildPackage extracts metadata from the .proto files in a directory and
// constructs possibly several packages, then selects a package to generate
// a proto_library rule for.
func buildPackage(dir, rel string, protoFiles, genFiles []string) *protoPackage {
	packageMap := make(map[string]*protoPackage)
	for _, name := range protoFiles {
		info := protoFileInfo(dir, name)
		if packageMap[info.PackageName] == nil {
			packageMap[info.PackageName] = newProtoPackage(info.PackageName)
		}
		packageMap[info.PackageName].addFile(info)
	}

	pkg, err := selectPackage(dir, rel, packageMap)
	if err != nil {
		log.Print(err)
		return nil
	}
	if pkg != nil {
		for _, name := range genFiles {
			pkg.addGenFile(dir, name)
		}
	}
	return pkg
}

// selectPackage chooses a package to generate rules for.
func selectPackage(dir, rel string, packageMap map[string]*protoPackage) (*protoPackage, error) {
	if len(packageMap) == 0 {
		return nil, nil
	}
	if len(packageMap) == 1 {
		for _, pkg := range packageMap {
			return pkg, nil
		}
	}
	defaultPackageName := strings.Replace(rel, "/", ".", -1)
	for _, pkg := range packageMap {
		if pkgName := goPackageName(pkg); pkgName != "" && pkgName == defaultPackageName {
			return pkg, nil
		}
	}
	return nil, fmt.Errorf("%s: directory contains multiple proto packages. Gazelle can only generate a proto_library for one package.", dir)
}

// goPackageName guesses the identifier in package declarations at the top of
// the .pb.go files that will be generated for this package. "" is returned
// if the package name cannot be determined.
//
// TODO(jayconrod): remove all Go-specific functionality. This is here
// temporarily for compatibility.
func goPackageName(pkg *protoPackage) string {
	if opt, ok := pkg.options["go_package"]; ok {
		if i := strings.IndexByte(opt, ';'); i >= 0 {
			return opt[i+1:]
		} else if i := strings.LastIndexByte(opt, '/'); i >= 0 {
			return opt[i+1:]
		} else {
			return opt
		}
	}
	if pkg.name != "" {
		return strings.Replace(pkg.name, ".", "_", -1)
	}
	if len(pkg.files) == 1 {
		for s := range pkg.files {
			return strings.TrimSuffix(s, ".proto")
		}
	}
	return ""
}

// hasDefaultVisibility returns whether oldFile contains a "package" rule with
// a "default_visibility" attribute. Rules generated by Gazelle should not
// have their own visibility attributes if this is the case.
func hasDefaultVisibility(f *rule.File) bool {
	if f == nil {
		return false
	}
	for _, r := range f.Rules {
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
