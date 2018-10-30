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

package main

import (
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/testtools"
)

func TestMain(m *testing.M) {
	tmpdir := os.Getenv("TEST_TMPDIR")
	flag.Set("repo_root", tmpdir)
	os.Exit(m.Run())
}

func defaultArgs(dir string) []string {
	return []string{
		"-repo_root", dir,
		"-go_prefix", "example.com/repo",
		dir,
	}
}

func TestCreateFile(t *testing.T) {
	// Create a directory with a simple .go file.
	tmpdir := os.Getenv("TEST_TMPDIR")
	dir, err := ioutil.TempDir(tmpdir, "")
	if err != nil {
		t.Fatalf("ioutil.TempDir(%q, %q) failed with %v; want success", tmpdir, "", err)
	}
	defer os.RemoveAll(dir)

	goFile := filepath.Join(dir, "main.go")
	if err = ioutil.WriteFile(goFile, []byte("package main"), 0600); err != nil {
		t.Fatalf("error writing file %q: %v", goFile, err)
	}

	// Check that Gazelle creates a new file named "BUILD.bazel".
	run(defaultArgs(dir))

	buildFile := filepath.Join(dir, "BUILD.bazel")
	if _, err = os.Stat(buildFile); err != nil {
		t.Errorf("could not stat BUILD.bazel: %v", err)
	}
}

func TestUpdateFile(t *testing.T) {
	// Create a directory with a simple .go file and an empty BUILD file.
	tmpdir := os.Getenv("TEST_TMPDIR")
	dir, err := ioutil.TempDir(tmpdir, "")
	if err != nil {
		t.Fatalf("ioutil.TempDir(%q, %q) failed with %v; want success", tmpdir, "", err)
	}
	defer os.RemoveAll(dir)

	goFile := filepath.Join(dir, "main.go")
	if err = ioutil.WriteFile(goFile, []byte("package main"), 0600); err != nil {
		t.Fatalf("error writing file %q: %v", goFile, err)
	}

	buildFile := filepath.Join(dir, "BUILD")
	if err = ioutil.WriteFile(buildFile, nil, 0600); err != nil {
		t.Fatalf("error writing file %q: %v", buildFile, err)
	}

	// Check that Gazelle updates the BUILD file in place.
	run(defaultArgs(dir))
	if st, err := os.Stat(buildFile); err != nil {
		t.Errorf("could not stat BUILD: %v", err)
	} else if st.Size() == 0 {
		t.Errorf("BUILD was not updated")
	}

	if _, err = os.Stat(filepath.Join(dir, "BUILD.bazel")); err == nil {
		t.Errorf("BUILD.bazel should not exist")
	}
}

func TestFixReadWriteDir(t *testing.T) {
	buildInFile := testtools.FileSpec{
		Path: "in/BUILD.in",
		Content: `
go_binary(
    name = "hello",
    pure = "on",
)
`,
	}
	buildSrcFile := testtools.FileSpec{
		Path:    "src/BUILD.bazel",
		Content: `# src build file`,
	}
	oldFiles := []testtools.FileSpec{
		buildInFile,
		buildSrcFile,
		{
			Path: "src/hello.go",
			Content: `
package main

func main() {}
`,
		}, {
			Path:    "out/BUILD",
			Content: `this should get replaced`,
		},
	}

	for _, tc := range []struct {
		desc string
		args []string
		want []testtools.FileSpec
	}{
		{
			desc: "read",
			args: []string{
				"-repo_root={{dir}}/src",
				"-experimental_read_build_files_dir={{dir}}/in",
				"-build_file_name=BUILD.bazel,BUILD,BUILD.in",
				"-go_prefix=example.com/repo",
				"{{dir}}/src",
			},
			want: []testtools.FileSpec{
				buildInFile,
				{
					Path: "src/BUILD.bazel",
					Content: `
load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

go_binary(
    name = "hello",
    embed = [":go_default_library"],
    pure = "on",
    visibility = ["//visibility:public"],
)

go_library(
    name = "go_default_library",
    srcs = ["hello.go"],
    importpath = "example.com/repo",
    visibility = ["//visibility:private"],
)
`,
				},
			},
		}, {
			desc: "write",
			args: []string{
				"-repo_root={{dir}}/src",
				"-experimental_write_build_files_dir={{dir}}/out",
				"-build_file_name=BUILD.bazel,BUILD,BUILD.in",
				"-go_prefix=example.com/repo",
				"{{dir}}/src",
			},
			want: []testtools.FileSpec{
				buildInFile,
				buildSrcFile,
				{
					Path: "out/BUILD",
					Content: `
load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

# src build file

go_library(
    name = "go_default_library",
    srcs = ["hello.go"],
    importpath = "example.com/repo",
    visibility = ["//visibility:private"],
)

go_binary(
    name = "repo",
    embed = [":go_default_library"],
    visibility = ["//visibility:public"],
)
`,
				},
			},
		}, {
			desc: "read_and_write",
			args: []string{
				"-repo_root={{dir}}/src",
				"-experimental_read_build_files_dir={{dir}}/in",
				"-experimental_write_build_files_dir={{dir}}/out",
				"-build_file_name=BUILD.bazel,BUILD,BUILD.in",
				"-go_prefix=example.com/repo",
				"{{dir}}/src",
			},
			want: []testtools.FileSpec{
				buildInFile,
				{
					Path: "out/BUILD",
					Content: `
load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

go_binary(
    name = "hello",
    embed = [":go_default_library"],
    pure = "on",
    visibility = ["//visibility:public"],
)

go_library(
    name = "go_default_library",
    srcs = ["hello.go"],
    importpath = "example.com/repo",
    visibility = ["//visibility:private"],
)
`,
				},
			},
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			dir, cleanup := testtools.CreateFiles(t, oldFiles)
			defer cleanup()
			replacer := strings.NewReplacer("{{dir}}", dir, "/", string(os.PathSeparator))
			for i := range tc.args {
				tc.args[i] = replacer.Replace(tc.args[i])
			}
			if err := run(tc.args); err != nil {
				t.Error(err)
			}
			testtools.CheckFiles(t, dir, tc.want)
		})
	}
}
