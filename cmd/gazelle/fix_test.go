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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/testtools"
	"github.com/bazelbuild/rules_go/go/tools/bazel"
)

var goSdk = flag.String("go_sdk", "", "name of the go_sdk repository when invoked by Bazel")

func TestMain(m *testing.M) {
	status := 1
	defer func() {
		os.Exit(status)
	}()

	flag.Parse()

	var err error
	tmpDir, err := ioutil.TempDir(os.Getenv("TEST_TMPDIR"), "gazelle_test")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	defer func() {
		// Before deleting files in the temporary directory, add write permission
		// to any files that don't have it. Files and directories in the module cache
		// are read-only, and on Windows, the read-only bit prevents deletion and
		// prevents Bazel from cleaning up the source tree.
		_ = filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if mode := info.Mode(); mode&0o200 == 0 {
				err = os.Chmod(path, mode|0o200)
			}
			return err
		})
		os.RemoveAll(tmpDir)
	}()

	if *goSdk != "" {
		// This flag is only set when the test is run by Bazel. Figure out where
		// the Go binary is and set GOROOT appropriately.
		entries, err := bazel.ListRunfiles()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}

		var goToolPath string
		ext := ""
		if runtime.GOOS == "windows" {
			ext = ".exe"
		}
		for _, entry := range entries {
			if entry.Workspace == *goSdk && entry.ShortPath == "bin/go"+ext {
				goToolPath = entry.Path
				break
			}
		}
		if goToolPath == "" {
			fmt.Fprintln(os.Stderr, "could not locate go tool")
			return
		}
		os.Setenv("GOROOT", filepath.Dir(filepath.Dir(goToolPath)))
	}
	os.Setenv("GOCACHE", filepath.Join(tmpDir, "gocache"))
	os.Setenv("GOPATH", filepath.Join(tmpDir, "gopath"))

	status = m.Run()
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
	if err = ioutil.WriteFile(goFile, []byte("package main"), 0o600); err != nil {
		t.Fatalf("error writing file %q: %v", goFile, err)
	}

	// Check that Gazelle creates a new file named "BUILD.bazel".
	if err = run(dir, defaultArgs(dir)); err != nil {
		t.Fatalf("run failed: %v", err)
	}

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
	if err = ioutil.WriteFile(goFile, []byte("package main"), 0o600); err != nil {
		t.Fatalf("error writing file %q: %v", goFile, err)
	}

	buildFile := filepath.Join(dir, "BUILD")
	if err = ioutil.WriteFile(buildFile, nil, 0o600); err != nil {
		t.Fatalf("error writing file %q: %v", buildFile, err)
	}

	// Check that Gazelle updates the BUILD file in place.
	if err = run(dir, defaultArgs(dir)); err != nil {
		t.Fatalf("run failed: %v", err)
	}

	if st, err := os.Stat(buildFile); err != nil {
		t.Errorf("could not stat BUILD: %v", err)
	} else if st.Size() == 0 {
		t.Errorf("BUILD was not updated")
	}

	if _, err = os.Stat(filepath.Join(dir, "BUILD.bazel")); err == nil {
		t.Errorf("BUILD.bazel should not exist")
	}
}

func TestNoChanges(t *testing.T) {
	// Create a directory with a BUILD file that doesn't need any changes.
	tmpdir := os.Getenv("TEST_TMPDIR")
	dir, err := ioutil.TempDir(tmpdir, "")
	if err != nil {
		t.Fatalf("ioutil.TempDir(%q, %q) failed with %v; want success", tmpdir, "", err)
	}
	defer os.RemoveAll(dir)

	goFile := filepath.Join(dir, "main.go")
	if err = ioutil.WriteFile(goFile, []byte("package main"), 0o600); err != nil {
		t.Fatalf("error writing file %q: %v", goFile, err)
	}

	buildFile := filepath.Join(dir, "BUILD")
	if err = ioutil.WriteFile(buildFile, []byte(`load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["main.go"],
    importpath = "example.com/repo",
    visibility = ["//visibility:private"],
)

go_binary(
    name = "hello",
    embed = [":go_default_library"],
    visibility = ["//visibility:public"],
)
`), 0o600); err != nil {
		t.Fatalf("error writing file %q: %v", buildFile, err)
	}
	st, err := os.Stat(buildFile)
	if err != nil {
		t.Errorf("could not stat BUILD: %v", err)
	}
	modTime := st.ModTime()

	// Ensure that Gazelle does not write to the BUILD file.
	if err = run(dir, defaultArgs(dir)); err != nil {
		t.Fatalf("run failed: %v", err)
	}

	if st, err := os.Stat(buildFile); err != nil {
		t.Errorf("could not stat BUILD: %v", err)
	} else if !modTime.Equal(st.ModTime()) {
		t.Errorf("unexpected modificaiton to BUILD")
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
		},
		{
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
    embed = [":repo_lib"],
    pure = "on",
    visibility = ["//visibility:public"],
)

go_library(
    name = "repo_lib",
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
    name = "repo_lib",
    srcs = ["hello.go"],
    importpath = "example.com/repo",
    visibility = ["//visibility:private"],
)

go_binary(
    name = "repo",
    embed = [":repo_lib"],
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
    embed = [":repo_lib"],
    pure = "on",
    visibility = ["//visibility:public"],
)

go_library(
    name = "repo_lib",
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
				if strings.HasPrefix(tc.args[i], "-go_prefix=") {
					continue // don't put backslashes in prefix on windows
				}
				tc.args[i] = replacer.Replace(tc.args[i])
			}
			if err := run(dir, tc.args); err != nil {
				t.Error(err)
			}
			testtools.CheckFiles(t, dir, tc.want)
		})
	}
}

func TestFix_LangFilter(t *testing.T) {
	fixture := []testtools.FileSpec{
		{
			Path: "BUILD.bazel",
			Content: `
load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

go_binary(
    name = "nofix",
    library = ":go_default_library",
    visibility = ["//visibility:public"],
)

go_library(
    name = "go_default_library",
    srcs = ["main.go"],
    importpath = "example.com/repo",
    visibility = ["//visibility:public"],
)`,
		},
		{
			Path:    "main.go",
			Content: `package main`,
		},
	}

	dir, cleanup := testtools.CreateFiles(t, fixture)
	defer cleanup()

	// Check that Gazelle does not update the BUILD file, due to lang filter.
	if err := run(dir, []string{
		"-repo_root", dir,
		"-go_prefix", "example.com/repo",
		"-lang=proto",
		dir,
	}); err != nil {
		t.Fatalf("run failed: %v", err)
	}

	testtools.CheckFiles(t, dir, fixture)
}
