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

package main

import (
	"path/filepath"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/testtools"
)

func TestMoveLabels(t *testing.T) {
	for _, tc := range []struct {
		desc, from, to string
		files, want    []testtools.FileSpec
	}{
		{
			desc: "move",
			from: "old",
			to:   "new",
			files: []testtools.FileSpec{{
				Path: "new/a/BUILD",
				Content: `load("//old:def.bzl", "x_binary")

x_binary(
    name = "a",
    deps = [
        ":a_lib",
        "//old/b",
        "//old/b:b_lib",
        "@c//old:c_lib",
    ],
    locs = [
        "abc $(location  //old/b) $(locations //old/b:b_lib) xyz",
    ],
)
`,
			}},
			want: []testtools.FileSpec{{
				Path: "new/a/BUILD",
				Content: `load("//new:def.bzl", "x_binary")

x_binary(
    name = "a",
    deps = [
        ":a_lib",
        "//new/b",
        "//new/b:b_lib",
        "@c//old:c_lib",
    ],
    locs = [
        "abc $(location  //new/b) $(locations //new/b:b_lib) xyz",
    ],
)
`,
			}},
		}, {
			desc: "vendor",
			from: "",
			to:   "vendor/github.com/bazelbuild/buildtools",
			files: []testtools.FileSpec{
				{
					Path: "vendor/github.com/bazelbuild/buildtools/BUILD.bazel",
					Content: `
load("@io_bazel_rules_go//go:def.bzl", "go_prefix")

go_prefix("github.com/bazelbuild/buildtools")

config_setting(
    name = "windows",
    values = {"cpu": "x64_windows"},
)

test_suite(
    name = "tests",
    tests = [
        "//api_proto:api.gen.pb.go_checkshtest",
        "//build:go_default_test",
        "//build:parse.y.go_checkshtest",
        "//build_proto:build.gen.pb.go_checkshtest",
        "//deps_proto:deps.gen.pb.go_checkshtest",
        "//edit:go_default_test",
        "//extra_actions_base_proto:extra_actions_base.gen.pb.go_checkshtest",
        "//lang:tables.gen.go_checkshtest",
        "//tables:go_default_test",
        "//wspace:go_default_test",
    ],
)
`,
				}, {
					Path: `vendor/github.com/bazelbuild/buildtools/edit/BUILD.bazel`,
					Content: `
load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "buildozer.go",
        "edit.go",
        "fix.go",
        "types.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//api_proto:go_default_library",
        "//build:go_default_library",
        "//build_proto:go_default_library",
        "//file:go_default_library",
        "//lang:go_default_library",
        "//tables:go_default_library",
        "//wspace:go_default_library",
        "@com_github_golang_protobuf//proto:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    srcs = ["edit_test.go"],
    library = ":go_default_library",
    deps = ["//build:go_default_library"],
)
`,
				},
			},
			want: []testtools.FileSpec{
				{
					Path: "vendor/github.com/bazelbuild/buildtools/BUILD.bazel",
					Content: `
load("@io_bazel_rules_go//go:def.bzl", "go_prefix")

go_prefix("github.com/bazelbuild/buildtools")

config_setting(
    name = "windows",
    values = {"cpu": "x64_windows"},
)

test_suite(
    name = "tests",
    tests = [
        "//vendor/github.com/bazelbuild/buildtools/api_proto:api.gen.pb.go_checkshtest",
        "//vendor/github.com/bazelbuild/buildtools/build:go_default_test",
        "//vendor/github.com/bazelbuild/buildtools/build:parse.y.go_checkshtest",
        "//vendor/github.com/bazelbuild/buildtools/build_proto:build.gen.pb.go_checkshtest",
        "//vendor/github.com/bazelbuild/buildtools/deps_proto:deps.gen.pb.go_checkshtest",
        "//vendor/github.com/bazelbuild/buildtools/edit:go_default_test",
        "//vendor/github.com/bazelbuild/buildtools/extra_actions_base_proto:extra_actions_base.gen.pb.go_checkshtest",
        "//vendor/github.com/bazelbuild/buildtools/lang:tables.gen.go_checkshtest",
        "//vendor/github.com/bazelbuild/buildtools/tables:go_default_test",
        "//vendor/github.com/bazelbuild/buildtools/wspace:go_default_test",
    ],
)
`,
				}, {
					Path: `vendor/github.com/bazelbuild/buildtools/edit/BUILD.bazel`,
					Content: `
load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "buildozer.go",
        "edit.go",
        "fix.go",
        "types.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//vendor/github.com/bazelbuild/buildtools/api_proto:go_default_library",
        "//vendor/github.com/bazelbuild/buildtools/build:go_default_library",
        "//vendor/github.com/bazelbuild/buildtools/build_proto:go_default_library",
        "//vendor/github.com/bazelbuild/buildtools/file:go_default_library",
        "//vendor/github.com/bazelbuild/buildtools/lang:go_default_library",
        "//vendor/github.com/bazelbuild/buildtools/tables:go_default_library",
        "//vendor/github.com/bazelbuild/buildtools/wspace:go_default_library",
        "@com_github_golang_protobuf//proto:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    srcs = ["edit_test.go"],
    library = ":go_default_library",
    deps = ["//vendor/github.com/bazelbuild/buildtools/build:go_default_library"],
)
`,
				},
			},
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			dir, cleanup := testtools.CreateFiles(t, tc.files)
			defer cleanup()

			args := []string{"-repo_root", dir, "-from", tc.from, "-to", filepath.Join(dir, filepath.FromSlash(tc.to))}
			if err := run(args); err != nil {
				t.Fatal(err)
			}

			testtools.CheckFiles(t, dir, tc.want)
		})
	}
}
