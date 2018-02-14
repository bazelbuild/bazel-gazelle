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
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fileSpec struct {
	path, content string
}

func TestMoveLabels(t *testing.T) {
	for _, tc := range []struct {
		desc, from, to string
		files, want    []fileSpec
	}{
		{
			desc: "move",
			from: "old",
			to:   "new",
			files: []fileSpec{{
				path: "new/a/BUILD",
				content: `
load("//old:def.bzl", "x_binary")

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
			want: []fileSpec{{
				path: "new/a/BUILD",
				content: `
load("//new:def.bzl", "x_binary")

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
			files: []fileSpec{
				{
					path: "vendor/github.com/bazelbuild/buildtools/BUILD.bazel",
					content: `
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
					path: `vendor/github.com/bazelbuild/buildtools/edit/BUILD.bazel`,
					content: `
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
			want: []fileSpec{
				{
					path: "vendor/github.com/bazelbuild/buildtools/BUILD.bazel",
					content: `
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
					path: `vendor/github.com/bazelbuild/buildtools/edit/BUILD.bazel`,
					content: `
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
			dir, err := createFiles(tc.files)
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(dir)

			args := []string{"-repo_root", dir, "-from", tc.from, "-to", filepath.Join(dir, filepath.FromSlash(tc.to))}
			if err := run(args); err != nil {
				t.Fatal(err)
			}

			checkFiles(t, dir, tc.want)
		})
	}
}

func createFiles(files []fileSpec) (string, error) {
	dir, err := ioutil.TempDir(os.Getenv("TEST_TEMPDIR"), "integration_test")
	if err != nil {
		return "", err
	}

	for _, f := range files {
		path := filepath.Join(dir, filepath.FromSlash(f.path))
		if strings.HasSuffix(f.path, "/") {
			if err := os.MkdirAll(path, 0700); err != nil {
				os.RemoveAll(dir)
				return "", err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			os.RemoveAll(dir)
			return "", err
		}
		if err := ioutil.WriteFile(path, []byte(f.content), 0600); err != nil {
			os.RemoveAll(dir)
			return "", err
		}
	}
	return dir, nil
}

func checkFiles(t *testing.T, dir string, files []fileSpec) {
	for _, f := range files {
		path := filepath.Join(dir, f.path)
		if strings.HasSuffix(f.path, "/") {
			if st, err := os.Stat(path); err != nil {
				t.Errorf("could not stat %s: %v", f.path, err)
			} else if !st.IsDir() {
				t.Errorf("not a directory: %s", f.path)
			}
		} else {
			want := f.content
			if len(want) > 0 && want[0] == '\n' {
				// Strip leading newline, added for readability.
				want = want[1:]
			}
			gotBytes, err := ioutil.ReadFile(filepath.Join(dir, f.path))
			if err != nil {
				t.Errorf("could not read %s: %v", f.path, err)
				continue
			}
			got := string(gotBytes)
			if got != want {
				t.Errorf("%s: got %s ; want %s", f.path, got, f.content)
			}
		}
	}
}
