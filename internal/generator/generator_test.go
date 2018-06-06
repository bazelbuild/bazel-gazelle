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

package generator_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/internal/config"
	"github.com/bazelbuild/bazel-gazelle/internal/generator"
	"github.com/bazelbuild/bazel-gazelle/internal/label"
	"github.com/bazelbuild/bazel-gazelle/internal/merger"
	"github.com/bazelbuild/bazel-gazelle/internal/packages"
	"github.com/bazelbuild/bazel-gazelle/internal/rule"

	bzl "github.com/bazelbuild/buildtools/build"
)

func testConfig(repoRoot, goPrefix string) *config.Config {
	c := &config.Config{
		RepoRoot:            repoRoot,
		Dirs:                []string{repoRoot},
		GoPrefix:            goPrefix,
		GenericTags:         config.BuildTags{},
		ValidBuildFileNames: []string{"BUILD.old"},
	}
	c.PreprocessTags()
	return c
}

func packageFromDir(conf *config.Config, dir string) (*config.Config, *packages.Package, *rule.File) {
	var pkg *packages.Package
	var oldFile *rule.File
	packages.Walk(conf, dir, func(_, rel string, c *config.Config, p *packages.Package, f *rule.File, _ bool) {
		if p != nil && p.Dir == dir {
			conf = c
			pkg = p
			oldFile = f
		}
	})
	return conf, pkg, oldFile
}

// convertPrivateAttrs copies private attributes to regular attributes, which
// will later be written out to build files. This allows tests to check the
// values of private attributes with simple string comparison.
func convertPrivateAttrs(f *rule.File) {
	for _, r := range f.Rules {
		for _, k := range r.PrivateAttrKeys() {
			r.SetAttr(k, r.PrivateAttr(k))
		}
	}
}

func TestGenerator(t *testing.T) {
	repoRoot := filepath.FromSlash("testdata/repo")
	goPrefix := "example.com/repo"
	c := testConfig(repoRoot, goPrefix)
	l := label.NewLabeler(c)

	var dirs []string
	err := filepath.Walk(repoRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if filepath.Base(path) == "BUILD.want" {
			dirs = append(dirs, filepath.Dir(path))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, dir := range dirs {
		rel, _ := filepath.Rel(repoRoot, dir)
		t.Run(rel, func(t *testing.T) {
			c, pkg, oldFile := packageFromDir(c, dir)
			g := generator.NewGenerator(c, l, oldFile)
			rs, _ := g.GenerateRules(pkg)
			f := rule.EmptyFile("test")
			for _, r := range rs {
				r.Insert(f)
			}
			merger.FixLoads(f)
			convertPrivateAttrs(f)
			f.Sync()
			got := string(bzl.Format(f.File))

			wantPath := filepath.Join(pkg.Dir, "BUILD.want")
			wantBytes, err := ioutil.ReadFile(wantPath)
			if err != nil {
				t.Fatalf("error reading %s: %v", wantPath, err)
			}
			want := string(wantBytes)

			if got != want {
				t.Errorf("g.Generate(%q, %#v) = %s; want %s", rel, pkg, got, want)
			}
		})
	}
}

func TestGeneratorEmpty(t *testing.T) {
	c := testConfig("", "example.com/repo")
	l := label.NewLabeler(c)
	g := generator.NewGenerator(c, l, nil)

	pkg := packages.Package{Name: "foo"}
	want := `filegroup(name = "go_default_library_protos")

proto_library(name = "foo_proto")

go_proto_library(name = "foo_go_proto")

go_library(name = "go_default_library")

go_binary(name = "repo")

go_test(name = "go_default_test")
`
	_, empty := g.GenerateRules(&pkg)
	f := rule.EmptyFile("test")
	for _, e := range empty {
		e.Insert(f)
	}
	f.Sync()
	got := string(bzl.Format(f.File))
	if got != want {
		t.Errorf("got '%s' ;\nwant %s", got, want)
	}
}

func TestGeneratorEmptyLegacyProto(t *testing.T) {
	c := testConfig("", "example.com/repo")
	c.ProtoMode = config.LegacyProtoMode
	l := label.NewLabeler(c)
	g := generator.NewGenerator(c, l, nil)

	pkg := packages.Package{Name: "foo"}
	_, empty := g.GenerateRules(&pkg)
	for _, e := range empty {
		if kind := e.Kind(); kind == "proto_library" || kind == "go_proto_library" || kind == "go_grpc_library" {
			t.Errorf("deleted rule %s ; should not delete in legacy proto mode", kind)
		}
	}
}
