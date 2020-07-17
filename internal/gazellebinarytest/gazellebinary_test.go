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

package gazellebinarytest

import (
	"os"
	"os/exec"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/testtools"
	"github.com/bazelbuild/rules_go/go/tools/bazel"
)

func TestGazelleBinary(t *testing.T) {
	gazellePath, ok := bazel.FindBinary("internal/gazellebinarytest", "gazelle_go_x")
	if !ok {
		t.Fatal("could not find gazelle binary")
	}

	files := []testtools.FileSpec{
		{Path: "WORKSPACE"},
		{Path: "BUILD.bazel", Content: "# gazelle:prefix example.com/test"},
		{Path: "foo.go", Content: "package foo"},
		{Path: "foo.proto", Content: `syntax = "proto3";`},
	}
	dir, cleanup := testtools.CreateFiles(t, files)
	defer cleanup()

	cmd := exec.Command(gazellePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	testtools.CheckFiles(t, dir, []testtools.FileSpec{{
		Path: "BUILD.bazel",
		Content: `
load("@io_bazel_rules_go//go:def.bzl", "go_library")

# gazelle:prefix example.com/test

go_library(
    name = "test",
    srcs = ["foo.go"],
    importpath = "example.com/test",
    visibility = ["//visibility:public"],
)

x_library(name = "x_default_library")
`,
	}})
}
