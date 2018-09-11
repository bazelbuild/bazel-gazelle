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
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/internal/config"
	"github.com/bazelbuild/bazel-gazelle/internal/rule"
	"github.com/bazelbuild/bazel-gazelle/internal/testtools"
)

func TestConfigureCallbackOrder(t *testing.T) {
	dir, err := createFiles([]fileSpec{{path: "a/b/"}})
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
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
	dir, err := createFiles([]fileSpec{
		{path: "update/sub/"},
		{
			path:    "update/ignore/BUILD.bazel",
			content: "# gazelle:ignore",
		},
		{path: "update/ignore/sub/"},
		{
			path:    "update/error/BUILD.bazel",
			content: "(",
		},
		{path: "update/error/sub/"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

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
	dir, err := createFiles([]fileSpec{
		{
			path:    "BUILD.bazel",
			content: "# gazelle:build_file_name BUILD.test",
		}, {
			path: "BUILD",
		}, {
			path: "sub/BUILD.test",
		}, {
			path: "sub/BUILD.bazel",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
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
	dir, err := createFiles([]fileSpec{
		{
			path: "BUILD.bazel",
			content: `
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
		{path: "a.go"},
		{path: ".dot"},
		{path: "_blank"},
		{path: "sub/b.go"},
		{path: "ign/bad"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
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
	want := []string{"BUILD.bazel"}
	if !reflect.DeepEqual(files, want) {
		t.Errorf("got %#v; want %#v", files, want)
	}
}

func TestGeneratedFiles(t *testing.T) {
	dir, err := createFiles([]fileSpec{
		{
			path: "BUILD.bazel",
			content: `
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
		{path: "gen-and-static"},
		{path: "static"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
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
	files := []fileSpec{
		{path: "root/a.go", content: "package a"},
		{path: "root/b", symlink: "../b"},   // symlink outside repo is followed
		{path: "root/c", symlink: "c"},      // symlink inside repo is not followed.
		{path: "root/d", symlink: "../b/d"}, // symlink under root/b not followed
		{path: "root/e", symlink: "../e"},
		{path: "c/c.go", symlink: "package c"},
		{path: "b/b.go", content: "package b"},
		{path: "b/d/d.go", content: "package d"},
		{path: "e/loop", symlink: "loop2"}, // symlink loop
		{path: "e/loop2", symlink: "loop"},
	}
	dir, err := createFiles(files)
	if err != nil {
		t.Fatalf("createFiles() failed with %v; want success", err)
	}
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
	files := []fileSpec{
		{
			path:    "root/BUILD",
			content: "# gazelle:exclude b",
		},
		{path: "root/b", symlink: "../b"},
		{path: "b/b.go", content: "package b"},
	}
	dir, err := createFiles(files)
	if err != nil {
		t.Fatalf("createFiles() failed with %v; want success", err)
	}
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
	files := []fileSpec{
		{
			path:    "root/BUILD",
			content: "# gazelle:exclude b",
		},
		{path: "root/b", symlink: "../b"},  // ignored
		{path: "root/b2", symlink: "../b"}, // not ignored
		{path: "b/b.go", content: "package b"},
	}
	dir, err := createFiles(files)
	if err != nil {
		t.Fatalf("createFiles() failed with %v; want success", err)
	}
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
	files := []fileSpec{
		{path: "root/b", symlink: "../link0"},
		{path: "link0", symlink: "b"},
		{path: "root/b2", symlink: "../b"},
		{path: "b/b.go", content: "package b"},
	}
	dir, err := createFiles(files)
	if err != nil {
		t.Fatalf("createFiles() failed with %v; want success", err)
	}
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
	files := []fileSpec{
		{path: "root/b", symlink: "../b"},
	}
	dir, err := createFiles(files)
	if err != nil {
		t.Fatalf("createFiles() failed with %v; want success", err)
	}
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

type fileSpec struct {
	path, content, symlink string
}

func createFiles(files []fileSpec) (string, error) {
	dir, err := ioutil.TempDir(os.Getenv("TEST_TMPDIR"), "walk_test")
	if err != nil {
		return "", err
	}
	for _, f := range files {
		path := filepath.Join(dir, f.path)
		if strings.HasSuffix(f.path, "/") {
			if err := os.MkdirAll(path, 0700); err != nil {
				return dir, err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return "", err
		}
		if f.symlink != "" {
			if err := os.Symlink(f.symlink, path); err != nil {
				return "", err
			}
			continue
		}
		if err := ioutil.WriteFile(path, []byte(f.content), 0600); err != nil {
			return "", err
		}
	}
	return dir, nil
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
