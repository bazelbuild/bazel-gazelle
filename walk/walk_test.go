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

package walk

import (
	"flag"
	"path"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/bazelbuild/bazel-gazelle/testtools"
)

func TestConfigureCallbackOrder(t *testing.T) {
	dir, cleanup := testtools.CreateFiles(t, []testtools.FileSpec{{Path: "a/b/"}})
	defer cleanup()

	var configureRels, callbackRels []string
	c, cexts := testConfig(t, dir)
	cexts = append(cexts, &testConfigurer{func(_ *config.Config, rel string, _ *rule.File) {
		configureRels = append(configureRels, rel)
	}})
	Walk(c, cexts, []string{dir}, VisitAllUpdateSubdirsMode, func(_ string, rel string, _ *config.Config, _ bool, _ *rule.File, _, _, _ []string) {
		callbackRels = append(callbackRels, rel)
	})
	if want := []string{"", "a", "a/b"}; !reflect.DeepEqual(configureRels, want) {
		t.Errorf("configure order: got %#v; want %#v", configureRels, want)
	}
	if want := []string{"a/b", "a", ""}; !reflect.DeepEqual(callbackRels, want) {
		t.Errorf("callback order: got %#v; want %#v", callbackRels, want)
	}
}

func TestUpdateDirs(t *testing.T) {
	dir, cleanup := testtools.CreateFiles(t, []testtools.FileSpec{
		{Path: "update/sub/"},
		{
			Path:    "update/ignore/BUILD.bazel",
			Content: "# gazelle:ignore",
		},
		{Path: "update/ignore/sub/"},
		{
			Path:    "update/error/BUILD.bazel",
			Content: "(",
		},
		{Path: "update/error/sub/"},
	})
	defer cleanup()

	type visitSpec struct {
		rel    string
		update bool
	}
	for _, tc := range []struct {
		desc string
		rels []string
		mode Mode
		want []visitSpec
	}{
		{
			desc: "visit_all_update_subdirs",
			rels: []string{"update"},
			mode: VisitAllUpdateSubdirsMode,
			want: []visitSpec{
				{"update/error/sub", true},
				{"update/error", false},
				{"update/ignore/sub", true},
				{"update/ignore", false},
				{"update/sub", true},
				{"update", true},
				{"", false},
			},
		}, {
			desc: "visit_all_update_dirs",
			rels: []string{"update", "update/ignore/sub"},
			mode: VisitAllUpdateDirsMode,
			want: []visitSpec{
				{"update/error/sub", false},
				{"update/error", false},
				{"update/ignore/sub", true},
				{"update/ignore", false},
				{"update/sub", false},
				{"update", true},
				{"", false},
			},
		}, {
			desc: "update_dirs",
			rels: []string{"update", "update/ignore/sub"},
			mode: UpdateDirsMode,
			want: []visitSpec{
				{"update/ignore/sub", true},
				{"update", true},
			},
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			c, cexts := testConfig(t, dir)
			dirs := make([]string, len(tc.rels))
			for i, rel := range tc.rels {
				dirs[i] = filepath.Join(dir, filepath.FromSlash(rel))
			}
			var visits []visitSpec
			Walk(c, cexts, dirs, tc.mode, func(_ string, rel string, _ *config.Config, update bool, _ *rule.File, _, _, _ []string) {
				visits = append(visits, visitSpec{rel, update})
			})
			if !reflect.DeepEqual(visits, tc.want) {
				t.Errorf("got %#v; want %#v", visits, tc.want)
			}
		})
	}
}

func TestCustomBuildName(t *testing.T) {
	dir, cleanup := testtools.CreateFiles(t, []testtools.FileSpec{
		{
			Path:    "BUILD.bazel",
			Content: "# gazelle:build_file_name BUILD.test",
		}, {
			Path: "BUILD",
		}, {
			Path: "sub/BUILD.test",
		}, {
			Path: "sub/BUILD.bazel",
		},
	})
	defer cleanup()

	c, cexts := testConfig(t, dir)
	var buildRels []string
	Walk(c, cexts, []string{dir}, VisitAllUpdateSubdirsMode, func(_ string, _ string, _ *config.Config, _ bool, f *rule.File, _, _, _ []string) {
		rel, err := filepath.Rel(c.RepoRoot, f.Path)
		if err != nil {
			t.Error(err)
		} else {
			buildRels = append(buildRels, filepath.ToSlash(rel))
		}
	})
	want := []string{
		"sub/BUILD.test",
		"BUILD.bazel",
	}
	if !reflect.DeepEqual(buildRels, want) {
		t.Errorf("got %#v; want %#v", buildRels, want)
	}
}

func TestExcludeFiles(t *testing.T) {
	dir, cleanup := testtools.CreateFiles(t, []testtools.FileSpec{
		{
			Path: "BUILD.bazel",
			Content: `
# gazelle:exclude a.go
# gazelle:exclude sub/b.go
# gazelle:exclude ign
# gazelle:exclude gen

gen(
    name = "x",
    out = "gen",
)
`,
		},
		{Path: "a.go"},
		{Path: ".dot"},
		{Path: "_blank"},
		{Path: "sub/b.go"},
		{Path: "ign/bad"},
	})
	defer cleanup()

	c, cexts := testConfig(t, dir)
	var files []string
	Walk(c, cexts, []string{dir}, VisitAllUpdateSubdirsMode, func(_ string, rel string, _ *config.Config, _ bool, _ *rule.File, _, regularFiles, genFiles []string) {
		for _, f := range regularFiles {
			files = append(files, path.Join(rel, f))
		}
		for _, f := range genFiles {
			files = append(files, path.Join(rel, f))
		}
	})
	want := []string{".dot", "BUILD.bazel", "_blank"}
	if !reflect.DeepEqual(files, want) {
		t.Errorf("got %#v; want %#v", files, want)
	}
}

func TestGeneratedFiles(t *testing.T) {
	dir, cleanup := testtools.CreateFiles(t, []testtools.FileSpec{
		{
			Path: "BUILD.bazel",
			Content: `
unknown_rule(
    name = "blah1",
    out = "gen1",
)

unknown_rule(
    name = "blah2",
    outs = [
        "gen2",
        "gen-and-static",
    ],
)
`,
		},
		{Path: "gen-and-static"},
		{Path: "static"},
	})
	defer cleanup()

	c, cexts := testConfig(t, dir)
	var regularFiles, genFiles []string
	Walk(c, cexts, []string{dir}, VisitAllUpdateSubdirsMode, func(_ string, rel string, _ *config.Config, _ bool, _ *rule.File, _, reg, gen []string) {
		for _, f := range reg {
			regularFiles = append(regularFiles, path.Join(rel, f))
		}
		for _, f := range gen {
			genFiles = append(genFiles, path.Join(rel, f))
		}
	})
	if want := []string{"BUILD.bazel", "gen-and-static", "static"}; !reflect.DeepEqual(regularFiles, want) {
		t.Errorf("regularFiles: got %#v; want %#v", regularFiles, want)
	}
	if want := []string{"gen1", "gen2", "gen-and-static"}; !reflect.DeepEqual(genFiles, want) {
		t.Errorf("genFiles: got %#v; want %#v", genFiles, want)
	}
}

func TestSymlinksBasic(t *testing.T) {
	files := []testtools.FileSpec{
		{Path: "root/a.go", Content: "package a"},
		{Path: "root/b", Symlink: "../b"},   // symlink outside repo is followed
		{Path: "root/c", Symlink: "c"},      // symlink inside repo is not followed.
		{Path: "root/d", Symlink: "../b/d"}, // symlink under root/b not followed
		{Path: "root/e", Symlink: "../e"},
		{Path: "c/c.go", Symlink: "package c"},
		{Path: "b/b.go", Content: "package b"},
		{Path: "b/d/d.go", Content: "package d"},
		{Path: "e/loop", Symlink: "loop2"}, // symlink loop
		{Path: "e/loop2", Symlink: "loop"},
	}
	dir, cleanup := testtools.CreateFiles(t, files)
	defer cleanup()

	root := filepath.Join(dir, "root")
	c, cexts := testConfig(t, root)
	var rels []string
	Walk(c, cexts, []string{root}, VisitAllUpdateSubdirsMode, func(_ string, rel string, _ *config.Config, _ bool, _ *rule.File, _, _, _ []string) {
		rels = append(rels, rel)
	})
	want := []string{"b/d", "b", "e", ""}
	if !reflect.DeepEqual(rels, want) {
		t.Errorf("got %#v; want %#v", rels, want)
	}
}

func TestSymlinksIgnore(t *testing.T) {
	files := []testtools.FileSpec{
		{
			Path:    "root/BUILD",
			Content: "# gazelle:exclude b",
		},
		{Path: "root/b", Symlink: "../b"},
		{Path: "b/b.go", Content: "package b"},
	}
	dir, cleanup := testtools.CreateFiles(t, files)
	defer cleanup()

	root := filepath.Join(dir, "root")
	c, cexts := testConfig(t, root)
	var rels []string
	Walk(c, cexts, []string{root}, VisitAllUpdateSubdirsMode, func(_ string, rel string, _ *config.Config, _ bool, _ *rule.File, _, _, _ []string) {
		rels = append(rels, rel)
	})
	want := []string{""}
	if !reflect.DeepEqual(rels, want) {
		t.Errorf("got %#v; want %#v", rels, want)
	}
}

func TestSymlinksMixIgnoredAndNonIgnored(t *testing.T) {
	files := []testtools.FileSpec{
		{
			Path:    "root/BUILD",
			Content: "# gazelle:exclude b",
		},
		{Path: "root/b", Symlink: "../b"},  // ignored
		{Path: "root/b2", Symlink: "../b"}, // not ignored
		{Path: "b/b.go", Content: "package b"},
	}
	dir, cleanup := testtools.CreateFiles(t, files)
	defer cleanup()

	root := filepath.Join(dir, "root")
	c, cexts := testConfig(t, root)
	var rels []string
	Walk(c, cexts, []string{root}, VisitAllUpdateSubdirsMode, func(_ string, rel string, _ *config.Config, _ bool, _ *rule.File, _, _, _ []string) {
		rels = append(rels, rel)
	})
	want := []string{"b2", ""}
	if !reflect.DeepEqual(rels, want) {
		t.Errorf("got %#v; want %#v", rels, want)
	}
}

func TestSymlinksChained(t *testing.T) {
	files := []testtools.FileSpec{
		{Path: "root/b", Symlink: "../link0"},
		{Path: "link0", Symlink: "b"},
		{Path: "root/b2", Symlink: "../b"},
		{Path: "b/b.go", Content: "package b"},
	}
	dir, cleanup := testtools.CreateFiles(t, files)
	defer cleanup()

	root := filepath.Join(dir, "root")
	c, cexts := testConfig(t, root)
	var rels []string
	Walk(c, cexts, []string{root}, VisitAllUpdateSubdirsMode, func(_ string, rel string, _ *config.Config, _ bool, _ *rule.File, _, _, _ []string) {
		rels = append(rels, rel)
	})
	want := []string{"b", ""}
	if !reflect.DeepEqual(rels, want) {
		t.Errorf("got %#v; want %#v", rels, want)
	}
}

func TestSymlinksDangling(t *testing.T) {
	files := []testtools.FileSpec{
		{Path: "root/b", Symlink: "../b"},
	}
	dir, cleanup := testtools.CreateFiles(t, files)
	defer cleanup()

	root := filepath.Join(dir, "root")
	c, cexts := testConfig(t, root)
	var rels []string
	Walk(c, cexts, []string{root}, VisitAllUpdateSubdirsMode, func(_ string, rel string, _ *config.Config, _ bool, _ *rule.File, _, _, _ []string) {
		rels = append(rels, rel)
	})
	want := []string{""}
	if !reflect.DeepEqual(rels, want) {
		t.Errorf("got %#v; want %#v", rels, want)
	}
}

func TestSymlinksFollow(t *testing.T) {
	files := []testtools.FileSpec{
		{Path: "staging/src/k8s.io/api/"},
		{Path: "staging/src/k8s.io/BUILD.bazel", Content: "# gazelle:exclude api"},
		{Path: "vendor/k8s.io/api", Symlink: "../../staging/src/k8s.io/api"},
		{Path: "vendor/BUILD.bazel", Content: "# gazelle:follow k8s.io/api"},
	}
	dir, cleanup := testtools.CreateFiles(t, files)
	defer cleanup()

	c, cexts := testConfig(t, dir)
	var rels []string
	Walk(c, cexts, []string{dir}, VisitAllUpdateSubdirsMode, func(_ string, rel string, _ *config.Config, _ bool, _ *rule.File, _, _, _ []string) {
		rels = append(rels, rel)
	})
	want := []string{
		"staging/src/k8s.io",
		"staging/src",
		"staging",
		"vendor/k8s.io/api",
		"vendor/k8s.io",
		"vendor",
		"",
	}
	if !reflect.DeepEqual(rels, want) {
		t.Errorf("got %#v; want %#v", rels, want)
	}
}

func testConfig(t *testing.T, dir string) (*config.Config, []config.Configurer) {
	args := []string{"-repo_root", dir}
	cexts := []config.Configurer{&config.CommonConfigurer{}, &Configurer{}}
	c := testtools.NewTestConfig(t, cexts, nil, args)
	return c, cexts
}

type testConfigurer struct {
	configure func(c *config.Config, rel string, f *rule.File)
}

func (_ *testConfigurer) RegisterFlags(_ *flag.FlagSet, _ string, _ *config.Config) {}

func (_ *testConfigurer) CheckFlags(_ *flag.FlagSet, _ *config.Config) error { return nil }

func (_ *testConfigurer) KnownDirectives() []string { return nil }

func (tc *testConfigurer) Configure(c *config.Config, rel string, f *rule.File) {
	tc.configure(c, rel, f)
}
