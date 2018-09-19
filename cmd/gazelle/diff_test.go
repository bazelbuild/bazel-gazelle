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

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestDiffExisting(t *testing.T) {
	files := []fileSpec{
		{path: "WORKSPACE"},
		{
			path: "BUILD.bazel",
			content: `
# gazelle:prefix example.com/hello
`,
		}, {
			path:    "hello.go",
			content: `package hello`,
		},
	}
	dir, err := createFiles(files)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	if err := runGazelle(dir, []string{"-mode=diff", "-patch=p"}); err != nil {
		t.Fatal(err)
	}

	want := append(files, fileSpec{
		path: "p",
		content: `
--- BUILD.bazel	1970-01-01 00:00:00.000000000 +0000
+++ BUILD.bazel	1970-01-01 00:00:00.000000000 +0000
@@ -1,3 +1,11 @@
+load("@io_bazel_rules_go//go:def.bzl", "go_library")
 
 # gazelle:prefix example.com/hello
 
+go_library(
+    name = "go_default_library",
+    srcs = ["hello.go"],
+    importpath = "example.com/hello",
+    visibility = ["//visibility:public"],
+)
+
`,
	})
	checkFiles(t, dir, want)
}

func TestDiffNew(t *testing.T) {
	files := []fileSpec{
		{path: "WORKSPACE"},
		{
			path:    "hello.go",
			content: `package hello`,
		},
	}
	dir, err := createFiles(files)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	if err := runGazelle(dir, []string{"-go_prefix=example.com/hello", "-mode=diff", "-patch=p"}); err != nil {
		t.Fatal(err)
	}

	want := append(files, fileSpec{
		path: "p",
		content: `
--- /dev/null	1970-01-01 00:00:00.000000000 +0000
+++ BUILD.bazel	1970-01-01 00:00:00.000000000 +0000
@@ -0,0 +1,9 @@
+load("@io_bazel_rules_go//go:def.bzl", "go_library")
+
+go_library(
+    name = "go_default_library",
+    srcs = ["hello.go"],
+    importpath = "example.com/hello",
+    visibility = ["//visibility:public"],
+)
+
`,
	})
	checkFiles(t, dir, want)
}

func TestDiffReadWriteDir(t *testing.T) {
	files := []fileSpec{
		{
			path:    "repo/hello.go",
			content: "package hello",
		}, {
			path:    "read/BUILD.bazel",
			content: "# gazelle:prefix example.com/hello",
		},
	}
	dir, err := createFiles(files)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	args := []string{
		"-repo_root=repo",
		"-mode=diff",
		"-patch=p",
		"-experimental_read_build_files_dir=read",
		"-experimental_write_build_files_dir=write",
		"repo",
	}
	if err := runGazelle(dir, args); err != nil {
		t.Fatal(err)
	}

	wantPatch := fmt.Sprintf(`
--- %s	1970-01-01 00:00:00.000000000 +0000
+++ %s	1970-01-01 00:00:00.000000000 +0000
@@ -1 +1,11 @@
+load("@io_bazel_rules_go//go:def.bzl", "go_library")
+
 # gazelle:prefix example.com/hello
+
+go_library(
+    name = "go_default_library",
+    srcs = ["hello.go"],
+    importpath = "example.com/hello",
+    visibility = ["//visibility:public"],
+)
+
`,
		filepath.Join(dir, "read", "BUILD.bazel"),
		filepath.Join(dir, "write", "BUILD.bazel"))
	want := append(files, fileSpec{path: "p", content: wantPatch})
	checkFiles(t, dir, want)
}
