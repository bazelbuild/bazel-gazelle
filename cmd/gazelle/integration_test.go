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

// This file contains integration tests for all of Gazelle. It's meant to test
// common usage patterns and check for errors that are difficult to test in
// unit tests.

package main

import (
	"bytes"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/internal/wspace"
	"github.com/bazelbuild/bazel-gazelle/testtools"
)

// skipIfWorkspaceVisible skips the test if the WORKSPACE file for the
// repository is visible. This happens in newer Bazel versions when tests
// are run without sandboxing, since temp directories may be inside the
// exec root.
func skipIfWorkspaceVisible(t *testing.T, dir string) {
	if parent, err := wspace.Find(dir); err == nil {
		t.Skipf("WORKSPACE visible in parent %q of tmp %q", parent, dir)
	}
}

func runGazelle(wd string, args []string) error {
	oldWd, err := os.Getwd()
	if err != nil {
		return err
	}
	if err := os.Chdir(wd); err != nil {
		return err
	}
	defer os.Chdir(oldWd)

	return run(args)
}

// TestHelp checks that help commands do not panic due to nil flag values.
// Verifies #256.
func TestHelp(t *testing.T) {
	for _, args := range [][]string{
		{"help"},
		{"fix", "-h"},
		{"update", "-h"},
		{"update-repos", "-h"},
	} {
		t.Run(args[0], func(t *testing.T) {
			if err := runGazelle(".", args); err == nil {
				t.Errorf("%s: got success, want flag.ErrHelp", args[0])
			} else if err != flag.ErrHelp {
				t.Errorf("%s: got %v, want flag.ErrHelp", args[0], err)
			}
		})
	}
}

func TestNoRepoRootOrWorkspace(t *testing.T) {
	dir, cleanup := testtools.CreateFiles(t, nil)
	defer cleanup()
	skipIfWorkspaceVisible(t, dir)
	want := "-repo_root not specified"
	if err := runGazelle(dir, nil); err == nil {
		t.Fatalf("got success; want %q", want)
	} else if !strings.Contains(err.Error(), want) {
		t.Fatalf("got %q; want %q", err, want)
	}
}

func TestNoGoPrefixArgOrRule(t *testing.T) {
	dir, cleanup := testtools.CreateFiles(t, []testtools.FileSpec{
		{Path: "WORKSPACE", Content: ""},
		{Path: "hello.go", Content: "package hello"},
	})
	defer cleanup()
	buf := new(bytes.Buffer)
	log.SetOutput(buf)
	defer log.SetOutput(os.Stderr)
	if err := runGazelle(dir, nil); err != nil {
		t.Fatalf("got %#v; want success", err)
	}
	want := "go prefix is not set"
	if !strings.Contains(buf.String(), want) {
		t.Errorf("log does not contain %q\n--begin--\n%s--end--\n", want, buf.String())
	}
}

// TestSelectLabelsSorted checks that string lists in srcs and deps are sorted
// using buildifier order, even if they are inside select expressions.
// This applies to both new and existing lists and should preserve comments.
// buildifier does not do this yet bazelbuild/buildtools#122, so we do this
// in addition to calling build.Rewrite.
func TestSelectLabelsSorted(t *testing.T) {
	dir, cleanup := testtools.CreateFiles(t, []testtools.FileSpec{
		{Path: "WORKSPACE"},
		{
			Path: "BUILD",
			Content: `
load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = select({
        "@io_bazel_rules_go//go/platform:linux": [
            # foo comment
            "foo.go",  # side comment
            # bar comment
            "bar.go",
        ],
        "//conditions:default": [],
    }),
    importpath = "example.com/foo",
)
`,
		},
		{
			Path: "foo.go",
			Content: `
// +build linux

package foo

import (
    _ "example.com/foo/outer"
    _ "example.com/foo/outer/inner"
    _ "github.com/jr_hacker/tools"
)
`,
		},
		{
			Path: "bar.go",
			Content: `// +build linux

package foo
`,
		},
		{Path: "outer/outer.go", Content: "package outer"},
		{Path: "outer/inner/inner.go", Content: "package inner"},
	})
	want := `load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        # bar comment
        "bar.go",
        # foo comment
        "foo.go",  # side comment
    ],
    importpath = "example.com/foo",
    visibility = ["//visibility:public"],
    deps = select({
        "@io_bazel_rules_go//go/platform:linux": [
            "//outer:go_default_library",
            "//outer/inner:go_default_library",
            "@com_github_jr_hacker_tools//:go_default_library",
        ],
        "//conditions:default": [],
    }),
)
`
	defer cleanup()

	if err := runGazelle(dir, []string{"-go_prefix", "example.com/foo"}); err != nil {
		t.Fatal(err)
	}
	if got, err := ioutil.ReadFile(filepath.Join(dir, "BUILD")); err != nil {
		t.Fatal(err)
	} else if string(got) != want {
		t.Fatalf("got %s ; want %s", string(got), want)
	}
}

func TestFixAndUpdateChanges(t *testing.T) {
	files := []testtools.FileSpec{
		{Path: "WORKSPACE"},
		{
			Path: "BUILD",
			Content: `load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_prefix")
load("@io_bazel_rules_go//go:def.bzl", "cgo_library", "go_test")

go_prefix("example.com/foo")

go_library(
    name = "go_default_library",
    srcs = [
        "extra.go",
        "pure.go",
    ],
    library = ":cgo_default_library",
    visibility = ["//visibility:default"],
)

cgo_library(
    name = "cgo_default_library",
    srcs = ["cgo.go"],
)
`,
		},
		{
			Path:    "pure.go",
			Content: "package foo",
		},
		{
			Path: "cgo.go",
			Content: `package foo

import "C"
`,
		},
	}

	cases := []struct {
		cmd, want string
	}{
		{
			cmd: "update",
			want: `load("@io_bazel_rules_go//go:def.bzl", "cgo_library", "go_library", "go_prefix")

go_prefix("example.com/foo")

go_library(
    name = "go_default_library",
    srcs = [
        "cgo.go",
        "pure.go",
    ],
    cgo = True,
    importpath = "example.com/foo",
    visibility = ["//visibility:default"],
)

cgo_library(
    name = "cgo_default_library",
    srcs = ["cgo.go"],
)
`,
		}, {
			cmd: "fix",
			want: `load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_prefix")

go_prefix("example.com/foo")

go_library(
    name = "go_default_library",
    srcs = [
        "cgo.go",
        "pure.go",
    ],
    cgo = True,
    importpath = "example.com/foo",
    visibility = ["//visibility:default"],
)
`,
		},
	}

	for _, c := range cases {
		t.Run(c.cmd, func(t *testing.T) {
			dir, cleanup := testtools.CreateFiles(t, files)
			defer cleanup()

			if err := runGazelle(dir, []string{c.cmd}); err != nil {
				t.Fatal(err)
			}
			if got, err := ioutil.ReadFile(filepath.Join(dir, "BUILD")); err != nil {
				t.Fatal(err)
			} else if string(got) != c.want {
				t.Fatalf("got %s ; want %s", string(got), c.want)
			}
		})
	}
}

func TestFixUnlinkedCgoLibrary(t *testing.T) {
	files := []testtools.FileSpec{
		{Path: "WORKSPACE"},
		{
			Path: "BUILD",
			Content: `load("@io_bazel_rules_go//go:def.bzl", "cgo_library", "go_library")

cgo_library(
    name = "cgo_default_library",
    srcs = ["cgo.go"],
)

go_library(
    name = "go_default_library",
    srcs = ["pure.go"],
    importpath = "example.com/foo",
    visibility = ["//visibility:public"],
)
`,
		}, {
			Path:    "pure.go",
			Content: "package foo",
		},
	}

	dir, cleanup := testtools.CreateFiles(t, files)
	defer cleanup()

	want := `load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["pure.go"],
    importpath = "example.com/foo",
    visibility = ["//visibility:public"],
)
`
	if err := runGazelle(dir, []string{"fix", "-go_prefix", "example.com/foo"}); err != nil {
		t.Fatal(err)
	}
	if got, err := ioutil.ReadFile(filepath.Join(dir, "BUILD")); err != nil {
		t.Fatal(err)
	} else if string(got) != want {
		t.Fatalf("got %s ; want %s", string(got), want)
	}
}

// TestMultipleDirectories checks that all directories in a repository are
// indexed but only directories listed on the command line are updated.
func TestMultipleDirectories(t *testing.T) {
	files := []testtools.FileSpec{
		{Path: "WORKSPACE"},
		{
			Path: "a/BUILD.bazel",
			Content: `# This file shouldn't be modified.
load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["a.go"],
    importpath = "example.com/foo/x",
)
`,
		}, {
			Path:    "a/a.go",
			Content: "package a",
		}, {
			Path: "b/b.go",
			Content: `
package b

import _ "example.com/foo/x"
`,
		},
	}
	dir, cleanup := testtools.CreateFiles(t, files)
	defer cleanup()

	args := []string{"-go_prefix", "example.com/foo", "b"}
	if err := runGazelle(dir, args); err != nil {
		t.Fatal(err)
	}

	testtools.CheckFiles(t, dir, []testtools.FileSpec{
		files[1], // should not change
		{
			Path: "b/BUILD.bazel",
			Content: `load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["b.go"],
    importpath = "example.com/foo/b",
    visibility = ["//visibility:public"],
    deps = ["//a:go_default_library"],
)
`,
		},
	})
}

func TestErrorOutsideWorkspace(t *testing.T) {
	files := []testtools.FileSpec{
		{Path: "a/"},
		{Path: "b/"},
	}
	dir, cleanup := testtools.CreateFiles(t, files)
	defer cleanup()
	skipIfWorkspaceVisible(t, dir)

	cases := []struct {
		name, dir, want string
		args            []string
	}{
		{
			name: "outside workspace",
			dir:  dir,
			args: nil,
			want: "WORKSPACE cannot be found",
		}, {
			name: "outside repo_root",
			dir:  filepath.Join(dir, "a"),
			args: []string{"-repo_root", filepath.Join(dir, "b")},
			want: "not a subdirectory of repo root",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := runGazelle(c.dir, c.args); err == nil {
				t.Fatalf("got success; want %q", c.want)
			} else if !strings.Contains(err.Error(), c.want) {
				t.Fatalf("got %q; want %q", err, c.want)
			}
		})
	}
}

func TestBuildFileNameIgnoresBuild(t *testing.T) {
	files := []testtools.FileSpec{
		{Path: "WORKSPACE"},
		{Path: "BUILD/"},
		{
			Path:    "a/BUILD",
			Content: "!!! parse error",
		}, {
			Path:    "a.go",
			Content: "package a",
		},
	}
	dir, cleanup := testtools.CreateFiles(t, files)
	defer cleanup()

	args := []string{"-go_prefix", "example.com/foo", "-build_file_name", "BUILD.bazel"}
	if err := runGazelle(dir, args); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "BUILD.bazel")); err != nil {
		t.Errorf("BUILD.bazel not created: %v", err)
	}
}

func TestExternalVendor(t *testing.T) {
	files := []testtools.FileSpec{
		{
			Path:    "WORKSPACE",
			Content: `workspace(name = "banana")`,
		}, {
			Path: "a.go",
			Content: `package foo

import _ "golang.org/x/bar"
`,
		}, {
			Path: "vendor/golang.org/x/bar/bar.go",
			Content: `package bar

import _ "golang.org/x/baz"
`,
		}, {
			Path:    "vendor/golang.org/x/baz/baz.go",
			Content: "package baz",
		},
	}
	dir, cleanup := testtools.CreateFiles(t, files)
	defer cleanup()

	args := []string{"-go_prefix", "example.com/foo", "-external", "vendored"}
	if err := runGazelle(dir, args); err != nil {
		t.Fatal(err)
	}

	testtools.CheckFiles(t, dir, []testtools.FileSpec{
		{
			Path: config.DefaultValidBuildFileNames[0],
			Content: `load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["a.go"],
    importpath = "example.com/foo",
    visibility = ["//visibility:public"],
    deps = ["//vendor/golang.org/x/bar:go_default_library"],
)
`,
		}, {
			Path: "vendor/golang.org/x/bar/" + config.DefaultValidBuildFileNames[0],
			Content: `load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["bar.go"],
    importmap = "example.com/foo/vendor/golang.org/x/bar",
    importpath = "golang.org/x/bar",
    visibility = ["//visibility:public"],
    deps = ["//vendor/golang.org/x/baz:go_default_library"],
)
`,
		},
	})
}

func TestMigrateProtoRules(t *testing.T) {
	files := []testtools.FileSpec{
		{Path: "WORKSPACE"},
		{
			Path: config.DefaultValidBuildFileNames[0],
			Content: `
load("@io_bazel_rules_go//proto:go_proto_library.bzl", "go_proto_library")

filegroup(
    name = "go_default_library_protos",
    srcs = ["foo.proto"],
    visibility = ["//visibility:public"],
)

go_proto_library(
    name = "go_default_library",
    srcs = [":go_default_library_protos"],
)
`,
		}, {
			Path: "foo.proto",
			Content: `syntax = "proto3";

option go_package = "example.com/repo";
`,
		}, {
			Path:    "foo.pb.go",
			Content: `package repo`,
		},
	}

	for _, tc := range []struct {
		args []string
		want string
	}{
		{
			args: []string{"update", "-go_prefix", "example.com/repo"},
			want: `
load("@io_bazel_rules_go//proto:go_proto_library.bzl", "go_proto_library")

filegroup(
    name = "go_default_library_protos",
    srcs = ["foo.proto"],
    visibility = ["//visibility:public"],
)

go_proto_library(
    name = "go_default_library",
    srcs = [":go_default_library_protos"],
)
`,
		}, {
			args: []string{"fix", "-go_prefix", "example.com/repo"},
			want: `
load("@io_bazel_rules_go//go:def.bzl", "go_library")
load("@io_bazel_rules_go//proto:def.bzl", "go_proto_library")

proto_library(
    name = "repo_proto",
    srcs = ["foo.proto"],
    visibility = ["//visibility:public"],
)

go_proto_library(
    name = "repo_go_proto",
    importpath = "example.com/repo",
    proto = ":repo_proto",
    visibility = ["//visibility:public"],
)

go_library(
    name = "go_default_library",
    embed = [":repo_go_proto"],
    importpath = "example.com/repo",
    visibility = ["//visibility:public"],
)
`,
		},
	} {
		t.Run(tc.args[0], func(t *testing.T) {
			dir, cleanup := testtools.CreateFiles(t, files)
			defer cleanup()

			if err := runGazelle(dir, tc.args); err != nil {
				t.Fatal(err)
			}

			testtools.CheckFiles(t, dir, []testtools.FileSpec{{
				Path:    config.DefaultValidBuildFileNames[0],
				Content: tc.want,
			}})
		})
	}
}

func TestRemoveProtoDeletesRules(t *testing.T) {
	files := []testtools.FileSpec{
		{Path: "WORKSPACE"},
		{
			Path: config.DefaultValidBuildFileNames[0],
			Content: `
load("@io_bazel_rules_go//go:def.bzl", "go_library")
load("@io_bazel_rules_go//proto:def.bzl", "go_proto_library")

filegroup(
    name = "go_default_library_protos",
    srcs = ["foo.proto"],
    visibility = ["//visibility:public"],
)

proto_library(
    name = "repo_proto",
    srcs = ["foo.proto"],
    visibility = ["//visibility:public"],
)

go_proto_library(
    name = "repo_go_proto",
    importpath = "example.com/repo",
    proto = ":repo_proto",
    visibility = ["//visibility:public"],
)

go_library(
    name = "go_default_library",
    srcs = ["extra.go"],
    embed = [":repo_go_proto"],
    importpath = "example.com/repo",
    visibility = ["//visibility:public"],
)
`,
		}, {
			Path:    "extra.go",
			Content: `package repo`,
		},
	}
	dir, cleanup := testtools.CreateFiles(t, files)
	defer cleanup()

	args := []string{"fix", "-go_prefix", "example.com/repo"}
	if err := runGazelle(dir, args); err != nil {
		t.Fatal(err)
	}

	testtools.CheckFiles(t, dir, []testtools.FileSpec{{
		Path: config.DefaultValidBuildFileNames[0],
		Content: `
load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["extra.go"],
    importpath = "example.com/repo",
    visibility = ["//visibility:public"],
)
`,
	}})
}

func TestAddServiceConvertsToGrpc(t *testing.T) {
	files := []testtools.FileSpec{
		{Path: "WORKSPACE"},
		{
			Path: config.DefaultValidBuildFileNames[0],
			Content: `
load("@io_bazel_rules_go//go:def.bzl", "go_library")
load("@io_bazel_rules_go//proto:def.bzl", "go_proto_library")

proto_library(
    name = "repo_proto",
    srcs = ["foo.proto"],
    visibility = ["//visibility:public"],
)

go_proto_library(
    name = "repo_go_proto",
    importpath = "example.com/repo",
    proto = ":repo_proto",
    visibility = ["//visibility:public"],
)

go_library(
    name = "go_default_library",
    embed = [":repo_go_proto"],
    importpath = "example.com/repo",
    visibility = ["//visibility:public"],
)
`,
		}, {
			Path: "foo.proto",
			Content: `syntax = "proto3";

option go_package = "example.com/repo";

service {}
`,
		},
	}

	dir, cleanup := testtools.CreateFiles(t, files)
	defer cleanup()

	args := []string{"-go_prefix", "example.com/repo"}
	if err := runGazelle(dir, args); err != nil {
		t.Fatal(err)
	}

	testtools.CheckFiles(t, dir, []testtools.FileSpec{{
		Path: config.DefaultValidBuildFileNames[0],
		Content: `
load("@io_bazel_rules_go//go:def.bzl", "go_library")
load("@io_bazel_rules_go//proto:def.bzl", "go_proto_library")

proto_library(
    name = "repo_proto",
    srcs = ["foo.proto"],
    visibility = ["//visibility:public"],
)

go_proto_library(
    name = "repo_go_proto",
    compilers = ["@io_bazel_rules_go//proto:go_grpc"],
    importpath = "example.com/repo",
    proto = ":repo_proto",
    visibility = ["//visibility:public"],
)

go_library(
    name = "go_default_library",
    embed = [":repo_go_proto"],
    importpath = "example.com/repo",
    visibility = ["//visibility:public"],
)
`,
	}})
}

func TestProtoImportPrefix(t *testing.T) {
	files := []testtools.FileSpec{
		{Path: "WORKSPACE"},
		{
			Path: config.DefaultValidBuildFileNames[0],
			Content: `
load("@io_bazel_rules_go//go:def.bzl", "go_library")
load("@io_bazel_rules_go//proto:def.bzl", "go_proto_library")

proto_library(
    name = "foo_proto",
    srcs = ["foo.proto"],
    visibility = ["//visibility:public"],
)

go_proto_library(
    name = "repo_go_proto",
    importpath = "example.com/repo",
    proto = ":foo_proto",
    visibility = ["//visibility:public"],
)

go_library(
    name = "go_default_library",
    embed = [":repo_go_proto"],
    importpath = "example.com/repo",
    visibility = ["//visibility:public"],
)
`,
		}, {
			Path:    "foo.proto",
			Content: `syntax = "proto3";`,
		},
	}

	dir, cleanup := testtools.CreateFiles(t, files)
	defer cleanup()

	args := []string{"update", "-go_prefix", "example.com/repo",
		"-proto_import_prefix", "/bar"}
	if err := runGazelle(dir, args); err != nil {
		t.Fatal(err)
	}

	testtools.CheckFiles(t, dir, []testtools.FileSpec{{
		Path: config.DefaultValidBuildFileNames[0],
		Content: `
load("@io_bazel_rules_go//go:def.bzl", "go_library")
load("@io_bazel_rules_go//proto:def.bzl", "go_proto_library")

proto_library(
    name = "foo_proto",
    srcs = ["foo.proto"],
    import_prefix = "/bar",
    visibility = ["//visibility:public"],
)

go_proto_library(
    name = "repo_go_proto",
    importpath = "example.com/repo",
    proto = ":foo_proto",
    visibility = ["//visibility:public"],
)

go_library(
    name = "go_default_library",
    embed = [":repo_go_proto"],
    importpath = "example.com/repo",
    visibility = ["//visibility:public"],
)
`,
	}})
}

func TestEmptyGoPrefix(t *testing.T) {
	files := []testtools.FileSpec{
		{Path: "WORKSPACE"},
		{
			Path:    "foo/foo.go",
			Content: "package foo",
		}, {
			Path: "bar/bar.go",
			Content: `
package bar

import (
	_ "fmt"
	_ "foo"
)
`,
		},
	}

	dir, cleanup := testtools.CreateFiles(t, files)
	defer cleanup()

	args := []string{"-go_prefix", ""}
	if err := runGazelle(dir, args); err != nil {
		t.Fatal(err)
	}

	testtools.CheckFiles(t, dir, []testtools.FileSpec{{
		Path: filepath.Join("bar", config.DefaultValidBuildFileNames[0]),
		Content: `
load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["bar.go"],
    importpath = "bar",
    visibility = ["//visibility:public"],
    deps = ["//foo:go_default_library"],
)
`,
	}})
}

// TestResolveKeptImportpath checks that Gazelle can resolve dependencies
// against a library with a '# keep' comment on its importpath attribute
// when the importpath doesn't match what Gazelle would infer.
func TestResolveKeptImportpath(t *testing.T) {
	files := []testtools.FileSpec{
		{Path: "WORKSPACE"},
		{
			Path: "foo/foo.go",
			Content: `
package foo

import _ "example.com/alt/baz"
`,
		}, {
			Path: "bar/BUILD.bazel",
			Content: `load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["bar.go"],
    importpath = "example.com/alt/baz",  # keep
    visibility = ["//visibility:public"],
)
`,
		}, {
			Path:    "bar/bar.go",
			Content: "package bar",
		},
	}

	dir, cleanup := testtools.CreateFiles(t, files)
	defer cleanup()

	args := []string{"-go_prefix", "example.com/repo"}
	if err := runGazelle(dir, args); err != nil {
		t.Fatal(err)
	}

	testtools.CheckFiles(t, dir, []testtools.FileSpec{
		{
			Path: "foo/BUILD.bazel",
			Content: `
load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["foo.go"],
    importpath = "example.com/repo/foo",
    visibility = ["//visibility:public"],
    deps = ["//bar:go_default_library"],
)
`,
		}, {
			Path: "bar/BUILD.bazel",
			Content: `load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["bar.go"],
    importpath = "example.com/alt/baz",  # keep
    visibility = ["//visibility:public"],
)
`,
		},
	})
}

// TestResolveVendorSubdirectory checks that Gazelle can resolve libraries
// in a vendor directory which is not at the repository root.
func TestResolveVendorSubdirectory(t *testing.T) {
	files := []testtools.FileSpec{
		{Path: "WORKSPACE"},
		{
			Path:    "sub/vendor/example.com/foo/foo.go",
			Content: "package foo",
		}, {
			Path: "sub/bar/bar.go",
			Content: `
package bar

import _ "example.com/foo"
`,
		},
	}
	dir, cleanup := testtools.CreateFiles(t, files)
	defer cleanup()

	args := []string{"-go_prefix", "example.com/repo"}
	if err := runGazelle(dir, args); err != nil {
		t.Fatal(err)
	}

	testtools.CheckFiles(t, dir, []testtools.FileSpec{
		{
			Path: "sub/vendor/example.com/foo/BUILD.bazel",
			Content: `
load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["foo.go"],
    importmap = "example.com/repo/sub/vendor/example.com/foo",
    importpath = "example.com/foo",
    visibility = ["//visibility:public"],
)
`,
		}, {
			Path: "sub/bar/BUILD.bazel",
			Content: `
load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["bar.go"],
    importpath = "example.com/repo/sub/bar",
    visibility = ["//visibility:public"],
    deps = ["//sub/vendor/example.com/foo:go_default_library"],
)
`,
		},
	})
}

// TestDeleteProtoWithDeps checks that Gazelle will delete proto rules with
// dependencies after the proto sources are removed.
func TestDeleteProtoWithDeps(t *testing.T) {
	files := []testtools.FileSpec{
		{Path: "WORKSPACE"},
		{
			Path: "foo/BUILD.bazel",
			Content: `
load("@io_bazel_rules_go//proto:def.bzl", "go_proto_library")
load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["extra.go"],
    embed = [":scratch_go_proto"],
    importpath = "example.com/repo/foo",
    visibility = ["//visibility:public"],
)

proto_library(
    name = "foo_proto",
    srcs = ["foo.proto"],
    visibility = ["//visibility:public"],
    deps = ["//foo/bar:bar_proto"],
)

go_proto_library(
    name = "foo_go_proto",
    importpath = "example.com/repo/foo",
    proto = ":foo_proto",
    visibility = ["//visibility:public"],
    deps = ["//foo/bar:go_default_library"],
)
`,
		}, {
			Path:    "foo/extra.go",
			Content: "package foo",
		}, {
			Path: "foo/bar/bar.proto",
			Content: `
syntax = "proto3";

option go_package = "example.com/repo/foo/bar";

message Bar {};
`,
		},
	}
	dir, cleanup := testtools.CreateFiles(t, files)
	defer cleanup()

	args := []string{"-go_prefix", "example.com/repo"}
	if err := runGazelle(dir, args); err != nil {
		t.Fatal(err)
	}

	testtools.CheckFiles(t, dir, []testtools.FileSpec{
		{
			Path: "foo/BUILD.bazel",
			Content: `
load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["extra.go"],
    importpath = "example.com/repo/foo",
    visibility = ["//visibility:public"],
)
`,
		},
	})
}

func TestCustomRepoNames(t *testing.T) {
	files := []testtools.FileSpec{
		{
			Path: "WORKSPACE",
			Content: `
go_repository(
    name = "custom_repo",
    importpath = "example.com/bar",
    commit = "123456",
)
`,
		}, {
			Path: "foo.go",
			Content: `
package foo

import _ "example.com/bar"
`,
		},
	}
	dir, cleanup := testtools.CreateFiles(t, files)
	defer cleanup()

	args := []string{"-go_prefix", "example.com/foo"}
	if err := runGazelle(dir, args); err != nil {
		t.Fatal(err)
	}

	testtools.CheckFiles(t, dir, []testtools.FileSpec{
		{
			Path: "BUILD.bazel",
			Content: `
load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["foo.go"],
    importpath = "example.com/foo",
    visibility = ["//visibility:public"],
    deps = ["@custom_repo//:go_default_library"],
)
`,
		},
	})
}

func TestImportReposFromDep(t *testing.T) {
	files := []testtools.FileSpec{
		{
			Path: "WORKSPACE",
			Content: `
http_archive(
    name = "io_bazel_rules_go",
    url = "https://github.com/bazelbuild/rules_go/releases/download/0.10.1/rules_go-0.10.1.tar.gz",
    sha256 = "4b14d8dd31c6dbaf3ff871adcd03f28c3274e42abc855cb8fb4d01233c0154dc",
)
http_archive(
    name = "bazel_gazelle",
    url = "https://github.com/bazelbuild/bazel-gazelle/releases/download/0.10.0/bazel-gazelle-0.10.0.tar.gz",
    sha256 = "6228d9618ab9536892aa69082c063207c91e777e51bd3c5544c9c060cafe1bd8",
)
load("@io_bazel_rules_go//go:def.bzl", "go_rules_dependencies", "go_register_toolchains", "go_repository")
go_rules_dependencies()
go_register_toolchains()

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies")
gazelle_dependencies()

go_repository(
    name = "org_golang_x_net",
    importpath = "golang.org/x/net",
    tag = "1.2",
)

# keep
go_repository(
    name = "org_golang_x_sys",
    importpath = "golang.org/x/sys",
    remote = "https://github.com/golang/sys",
)

http_archive(
    name = "com_github_go_yaml_yaml",
    urls = ["https://example.com/yaml.tar.gz"],
    sha256 = "1234",
)
`,
		}, {
			Path: "Gopkg.lock",
			Content: `# This file is autogenerated, do not edit; changes may be undone by the next 'dep ensure'.


[[projects]]
  name = "github.com/pkg/errors"
  packages = ["."]
  revision = "645ef00459ed84a119197bfb8d8205042c6df63d"
  version = "v0.8.0"

[[projects]]
  branch = "master"
  name = "golang.org/x/net"
  packages = ["context"]
  revision = "66aacef3dd8a676686c7ae3716979581e8b03c47"

[[projects]]
  branch = "master"
  name = "golang.org/x/sys"
  packages = ["unix"]
  revision = "bb24a47a89eac6c1227fbcb2ae37a8b9ed323366"

[[projects]]
  branch = "v2"
  name = "github.com/go-yaml/yaml"
  packages = ["."]
  revision = "cd8b52f8269e0feb286dfeef29f8fe4d5b397e0b"

[solve-meta]
  analyzer-name = "dep"
  analyzer-version = 1
  inputs-digest = "05c1cd69be2c917c0cc4b32942830c2acfa044d8200fdc94716aae48a8083702"
  solver-name = "gps-cdcl"
  solver-version = 1
`,
		},
	}
	dir, cleanup := testtools.CreateFiles(t, files)
	defer cleanup()

	args := []string{"update-repos", "-build_file_generation", "off", "-from_file", "Gopkg.lock"}
	if err := runGazelle(dir, args); err != nil {
		t.Fatal(err)
	}

	testtools.CheckFiles(t, dir, []testtools.FileSpec{
		{
			Path: "WORKSPACE",
			Content: `
http_archive(
    name = "io_bazel_rules_go",
    url = "https://github.com/bazelbuild/rules_go/releases/download/0.10.1/rules_go-0.10.1.tar.gz",
    sha256 = "4b14d8dd31c6dbaf3ff871adcd03f28c3274e42abc855cb8fb4d01233c0154dc",
)

http_archive(
    name = "bazel_gazelle",
    url = "https://github.com/bazelbuild/bazel-gazelle/releases/download/0.10.0/bazel-gazelle-0.10.0.tar.gz",
    sha256 = "6228d9618ab9536892aa69082c063207c91e777e51bd3c5544c9c060cafe1bd8",
)

load("@io_bazel_rules_go//go:def.bzl", "go_register_toolchains", "go_rules_dependencies")

go_rules_dependencies()

go_register_toolchains()

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")

gazelle_dependencies()

go_repository(
    name = "org_golang_x_net",
    build_file_generation = "off",
    commit = "66aacef3dd8a676686c7ae3716979581e8b03c47",
    importpath = "golang.org/x/net",
)

# keep
go_repository(
    name = "org_golang_x_sys",
    importpath = "golang.org/x/sys",
    remote = "https://github.com/golang/sys",
)

http_archive(
    name = "com_github_go_yaml_yaml",
    urls = ["https://example.com/yaml.tar.gz"],
    sha256 = "1234",
)

go_repository(
    name = "com_github_pkg_errors",
    build_file_generation = "off",
    commit = "645ef00459ed84a119197bfb8d8205042c6df63d",
    importpath = "github.com/pkg/errors",
)
`,
		}})
}

func TestDeleteRulesInEmptyDir(t *testing.T) {
	files := []testtools.FileSpec{
		{Path: "WORKSPACE"},
		{
			Path: "BUILD.bazel",
			Content: `
load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_binary")

go_library(
    name = "go_default_library",
    srcs = [
        "bar.go",
        "foo.go",
    ],
    importpath = "example.com/repo",
    visibility = ["//visibility:private"],
)

go_binary(
    name = "cmd",
    embed = [":go_default_library"],
    visibility = ["//visibility:public"],
)
`,
		},
	}
	dir, cleanup := testtools.CreateFiles(t, files)
	defer cleanup()

	args := []string{"-go_prefix=example.com/repo"}
	if err := runGazelle(dir, args); err != nil {
		t.Fatal(err)
	}

	testtools.CheckFiles(t, dir, []testtools.FileSpec{
		{
			Path:    "BUILD.bazel",
			Content: "",
		},
	})
}

func TestFixWorkspaceWithoutGazelle(t *testing.T) {
	files := []testtools.FileSpec{
		{
			Path: "WORKSPACE",
			Content: `
load("@io_bazel_rules_go//go:def.bzl", "go_repository")

go_repository(
    name = "com_example_repo",
    importpath = "example.com/repo",
    tag = "1.2.3",
)
`,
		},
	}
	dir, cleanup := testtools.CreateFiles(t, files)
	defer cleanup()

	if err := runGazelle(dir, []string{"fix", "-go_prefix="}); err == nil {
		t.Error("got success; want error")
	} else if want := "bazel_gazelle is not declared"; !strings.Contains(err.Error(), want) {
		t.Errorf("got error %v; want error containing %q", err, want)
	}
}

// TestFixGazelle checks that loads of the gazelle macro from the old location
// in rules_go are replaced with the new location in @bazel_gazelle.
func TestFixGazelle(t *testing.T) {
	files := []testtools.FileSpec{
		{
			Path: "WORKSPACE",
		}, {
			Path: "BUILD.bazel",
			Content: `
load("@io_bazel_rules_go//go:def.bzl", "gazelle", "go_library")

gazelle(name = "gazelle")

# keep
go_library(name = "go_default_library")
`,
		},
	}
	dir, cleanup := testtools.CreateFiles(t, files)
	defer cleanup()

	if err := runGazelle(dir, nil); err != nil {
		t.Fatal(err)
	}
	testtools.CheckFiles(t, dir, []testtools.FileSpec{
		{
			Path: "BUILD.bazel",
			Content: `
load("@bazel_gazelle//:def.bzl", "gazelle")
load("@io_bazel_rules_go//go:def.bzl", "go_library")

gazelle(name = "gazelle")

# keep
go_library(name = "go_default_library")
`,
		},
	})
}

// TestKeepDeps checks rules with keep comments on the rule or on the deps
// attribute will not be modified during dependency resolution. Verifies #212.
func TestKeepDeps(t *testing.T) {
	files := []testtools.FileSpec{
		{
			Path: "WORKSPACE",
		}, {
			Path: "BUILD.bazel",
			Content: `
load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

# gazelle:prefix example.com/repo

# keep
go_library(
    name = "go_default_library",
    srcs = ["lib.go"],
    importpath = "example.com/repo",
    deps = [":dont_remove"],
)

go_test(
    name = "go_default_test",
    # keep
    srcs = ["lib_test.go"],
    # keep
    embed = [":go_default_library"],
    # keep
    deps = [":dont_remove"],
)
`,
		},
	}
	dir, cleanup := testtools.CreateFiles(t, files)
	defer cleanup()

	if err := runGazelle(dir, nil); err != nil {
		t.Fatal(err)
	}

	testtools.CheckFiles(t, dir, files)
}

func TestDontCreateBuildFileInEmptyDir(t *testing.T) {
	files := []testtools.FileSpec{
		{Path: "WORKSPACE"},
		{Path: "sub/"},
	}
	dir, cleanup := testtools.CreateFiles(t, files)
	defer cleanup()

	if err := runGazelle(dir, nil); err != nil {
		t.Error(err)
	}
	for _, f := range []string{"BUILD.bazel", "sub/BUILD.bazel"} {
		path := filepath.Join(dir, filepath.FromSlash(f))
		_, err := os.Stat(path)
		if err == nil {
			t.Errorf("%s: build file was created", f)
		}
	}
}

// TestNoIndex checks that gazelle does not index existing or generated
// library rules with the flag -index=false. Verifies #384.
func TestNoIndex(t *testing.T) {
	files := []testtools.FileSpec{
		{Path: "WORKSPACE"},
		{
			Path: "foo/foo.go",
			Content: `
package foo

import _ "example.com/bar"
`,
		}, {
			Path:    "third_party/example.com/bar/bar.go",
			Content: "package bar",
		}, {
			Path:    "third_party/BUILD.bazel",
			Content: "# gazelle:prefix",
		}, {
			Path: "third_party/example.com/bar/BUILD.bazel",
			Content: `
load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["bar.go"],
    importpath = "example.com/bar",
    visibility = ["//visibility:public"],
)
`,
		},
	}
	dir, cleanup := testtools.CreateFiles(t, files)
	defer cleanup()

	args := []string{
		"-go_prefix=example.com/repo",
		"-external=vendored",
		"-index=false",
	}
	if err := runGazelle(dir, args); err != nil {
		t.Fatal(err)
	}

	testtools.CheckFiles(t, dir, []testtools.FileSpec{{
		Path: "foo/BUILD.bazel",
		Content: `
load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["foo.go"],
    importpath = "example.com/repo/foo",
    visibility = ["//visibility:public"],
    deps = ["//vendor/example.com/bar:go_default_library"],
)
`,
	}})
}

// TestNoIndexNoRecurse checks that gazelle behaves correctly with the flags
// -r=false -index=false. Gazelle should not generate build files in
// subdirectories and should not resolve dependencies to local libraries.
func TestNoIndexNoRecurse(t *testing.T) {
	barBuildFile := testtools.FileSpec{
		Path:    "foo/bar/BUILD.bazel",
		Content: "# this should not be updated because -r=false",
	}
	files := []testtools.FileSpec{
		{Path: "WORKSPACE"},
		{
			Path: "foo/foo.go",
			Content: `package foo

import (
	_ "example.com/dep/baz"
)
`,
		},
		barBuildFile,
		{
			Path:    "foo/bar/bar.go",
			Content: "package bar",
		}, {
			Path: "third_party/baz/BUILD.bazel",
			Content: `load("@io_bazel_rules_go//go:def.bzl", "go_library")

# this should be ignored because -index=false
go_library(
    name = "go_default_library",
    srcs = ["baz.go"],
    importpath = "example.com/dep/baz",
    visibility = ["//visibility:public"],
)
`,
		}, {
			Path:    "third_party/baz/baz.go",
			Content: "package baz",
		},
	}
	dir, cleanup := testtools.CreateFiles(t, files)
	defer cleanup()

	args := []string{
		"-go_prefix=example.com/repo",
		"-external=vendored",
		"-r=false",
		"-index=false",
		"foo",
	}
	if err := runGazelle(dir, args); err != nil {
		t.Fatal(err)
	}

	testtools.CheckFiles(t, dir, []testtools.FileSpec{
		{
			Path: "foo/BUILD.bazel",
			Content: `load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["foo.go"],
    importpath = "example.com/repo/foo",
    visibility = ["//visibility:public"],
    deps = ["//vendor/example.com/dep/baz:go_default_library"],
)
`,
		},
		barBuildFile,
	})
}

// TestSubdirectoryPrefixExternal checks that directives set in subdirectories
// may be used in dependency resolution. Verifies #412.
func TestSubdirectoryPrefixExternal(t *testing.T) {
	files := []testtools.FileSpec{
		{
			Path: "WORKSPACE",
		}, {
			Path:    "BUILD.bazel",
			Content: "# gazelle:prefix",
		}, {
			Path:    "sub/BUILD.bazel",
			Content: "# gazelle:prefix example.com/sub",
		}, {
			Path: "sub/sub.go",
			Content: `
package sub

import (
	_ "example.com/sub/missing"
	_ "example.com/external"
)
`,
		},
	}
	dir, cleanup := testtools.CreateFiles(t, files)
	defer cleanup()

	if err := runGazelle(dir, []string{"-external=vendored", "-index=false"}); err != nil {
		t.Fatal(err)
	}

	testtools.CheckFiles(t, dir, []testtools.FileSpec{
		{
			Path: "sub/BUILD.bazel",
			Content: `
load("@io_bazel_rules_go//go:def.bzl", "go_library")

# gazelle:prefix example.com/sub

go_library(
    name = "go_default_library",
    srcs = ["sub.go"],
    importpath = "example.com/sub",
    visibility = ["//visibility:public"],
    deps = [
        "//sub/missing:go_default_library",
        "//vendor/example.com/external:go_default_library",
    ],
)
`,
		},
	})
}

func TestGoGrpcProtoFlag(t *testing.T) {
	files := []testtools.FileSpec{
		{
			Path: "WORKSPACE",
		}, {
			Path: "BUILD.bazel",
			Content: `
load("@io_bazel_rules_go//proto:def.bzl", "go_proto_library")
load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_proto_library(
    name = "foo_go_proto",
    importpath = "example.com/repo/foo",
    proto = ":foo_proto",
    visibility = ["//visibility:public"],
)

proto_library(
    name = "foo_proto",
    srcs = ["foo.proto"],
    visibility = ["//visibility:public"],
)

go_library(
    name = "go_default_library",
    embed = [":foo_go_proto"],
    importpath = "example.com/repo/foo",
    visibility = ["//visibility:public"],
)
`,
		}, {
			Path: "foo.proto",
			Content: `
syntax = "proto3";

option go_package = "example.com/repo/foo";

message Bar {};
`,
		}, {
			Path: "service/BUILD.bazel",
			Content: `
load("@io_bazel_rules_go//proto:def.bzl", "go_proto_library")
load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_proto_library(
    name = "service_go_proto",
    compilers = ["@io_bazel_rules_go//proto:go_grpc"],
    importpath = "example.com/repo/service",
    proto = ":service_proto",
    visibility = ["//visibility:public"],
)

proto_library(
    name = "service_proto",
    srcs = ["service.proto"],
    visibility = ["//visibility:public"],
)

go_library(
    name = "go_default_library",
    embed = [":service_go_proto"],
    importpath = "example.com/repo/service",
    visibility = ["//visibility:public"],
)
`,
		}, {
			Path: "service/service.proto",
			Content: `
syntax = "proto3";

option go_package = "example.com/repo/service";

service {}
`,
		},
	}
	dir, cleanup := testtools.CreateFiles(t, files)
	defer cleanup()

	args := []string{"update", "-go_grpc_compiler", "//foo", "-go_proto_compiler", "//bar"}
	if err := runGazelle(dir, args); err != nil {
		t.Fatal(err)
	}

	testtools.CheckFiles(t, dir, []testtools.FileSpec{
		{
			Path: "BUILD.bazel",
			Content: `
load("@io_bazel_rules_go//proto:def.bzl", "go_proto_library")
load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_proto_library(
    name = "foo_go_proto",
    compilers = ["//bar"],
    importpath = "example.com/repo/foo",
    proto = ":foo_proto",
    visibility = ["//visibility:public"],
)

proto_library(
    name = "foo_proto",
    srcs = ["foo.proto"],
    visibility = ["//visibility:public"],
)

go_library(
    name = "go_default_library",
    embed = [":foo_go_proto"],
    importpath = "example.com/repo/foo",
    visibility = ["//visibility:public"],
)
`,
		},
		{
			Path: "service/BUILD.bazel",
			Content: `
load("@io_bazel_rules_go//proto:def.bzl", "go_proto_library")
load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_proto_library(
    name = "service_go_proto",
    compilers = ["//foo"],
    importpath = "example.com/repo/service",
    proto = ":service_proto",
    visibility = ["//visibility:public"],
)

proto_library(
    name = "service_proto",
    srcs = ["service.proto"],
    visibility = ["//visibility:public"],
)

go_library(
    name = "go_default_library",
    embed = [":service_go_proto"],
    importpath = "example.com/repo/service",
    visibility = ["//visibility:public"],
)
`,
		},
	})
}

// TestMapKind tests the gazelle:map_kind directive.
// Verifies #448
func TestMapKind(t *testing.T) {
	files := []testtools.FileSpec{
		{
			Path: "WORKSPACE",
		}, {
			Path:    "BUILD.bazel",
			Content: "# gazelle:prefix example.com/mapkind",
		}, {
			Path:    "root_lib.go",
			Content: `package mapkind`,
		}, {
			Path:    "enabled/BUILD.bazel",
			Content: "# gazelle:map_kind go_library my_library //tools/go:def.bzl",
		}, {
			Path:    "enabled/enabled_lib.go",
			Content: `package enabled`,
		}, {
			Path: "enabled/inherited/BUILD.bazel",
		}, {
			Path:    "enabled/inherited/inherited_lib.go",
			Content: `package inherited`,
		}, {
			Path: "enabled/existing_rules/mapped/BUILD.bazel",
			Content: `
load("//tools/go:def.bzl", "my_library")

# An existing rule with a mapped type is updated
my_library(
    name = "go_default_library",
    srcs = ["deleted_file.go", "mapped_lib.go"],
    importpath = "example.com/mapkind/enabled/existing_rules/mapped",
    visibility = ["//visibility:public"],
)
`,
		}, {
			Path:    "enabled/existing_rules/mapped/mapped_lib.go",
			Content: `package mapped`,
		}, {
			Path:    "enabled/existing_rules/mapped/mapped_lib2.go",
			Content: `package mapped`,
		}, {
			Path: "enabled/existing_rules/unmapped/BUILD.bazel",
			Content: `
load("@io_bazel_rules_go//go:def.bzl", "go_library")

# An existing rule with an unmapped type is preserved
go_library(
    name = "go_default_library",
    srcs = ["unmapped_lib.go"],
    importpath = "example.com/mapkind/enabled/existing_rules",
    visibility = ["//visibility:public"],
)
`,
		}, {
			Path:    "enabled/existing_rules/unmapped/unmapped_lib.go",
			Content: `package unmapped`,
		}, {
			Path:    "enabled/overridden/BUILD.bazel",
			Content: "# gazelle:map_kind go_library overridden_library //tools/overridden:def.bzl",
		}, {
			Path:    "enabled/overridden/overridden_lib.go",
			Content: `package overridden`,
		}, {
			Path: "disabled/BUILD.bazel",
		}, {
			Path:    "disabled/disabled_lib.go",
			Content: `package disabled`,
		},
	}
	dir, cleanup := testtools.CreateFiles(t, files)
	defer cleanup()

	if err := runGazelle(dir, []string{"-external=vendored", "-index=false"}); err != nil {
		t.Fatal(err)
	}

	testtools.CheckFiles(t, dir, []testtools.FileSpec{
		{
			Path: "BUILD.bazel",
			Content: `
load("@io_bazel_rules_go//go:def.bzl", "go_library")

# gazelle:prefix example.com/mapkind

go_library(
    name = "go_default_library",
    srcs = ["root_lib.go"],
    importpath = "example.com/mapkind",
    visibility = ["//visibility:public"],
)
`,
		},
		{
			Path: "enabled/BUILD.bazel",
			Content: `
load("//tools/go:def.bzl", "my_library")

# gazelle:map_kind go_library my_library //tools/go:def.bzl

my_library(
    name = "go_default_library",
    srcs = ["enabled_lib.go"],
    importpath = "example.com/mapkind/enabled",
    visibility = ["//visibility:public"],
)
`,
		},
		{
			Path: "enabled/inherited/BUILD.bazel",
			Content: `
load("//tools/go:def.bzl", "my_library")

my_library(
    name = "go_default_library",
    srcs = ["inherited_lib.go"],
    importpath = "example.com/mapkind/enabled/inherited",
    visibility = ["//visibility:public"],
)
`,
		},
		{
			Path: "enabled/overridden/BUILD.bazel",
			Content: `
load("//tools/overridden:def.bzl", "overridden_library")

# gazelle:map_kind go_library overridden_library //tools/overridden:def.bzl

overridden_library(
    name = "go_default_library",
    srcs = ["overridden_lib.go"],
    importpath = "example.com/mapkind/enabled/overridden",
    visibility = ["//visibility:public"],
)
`,
		},
		{
			Path: "disabled/BUILD.bazel",
			Content: `
load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["disabled_lib.go"],
    importpath = "example.com/mapkind/disabled",
    visibility = ["//visibility:public"],
)
`,
		},
		{
			Path: "enabled/existing_rules/mapped/BUILD.bazel",
			Content: `
load("//tools/go:def.bzl", "my_library")

# An existing rule with a mapped type is updated
my_library(
    name = "go_default_library",
    srcs = [
        "mapped_lib.go",
        "mapped_lib2.go",
    ],
    importpath = "example.com/mapkind/enabled/existing_rules/mapped",
    visibility = ["//visibility:public"],
)
`,
		},
		{
			Path: "enabled/existing_rules/unmapped/BUILD.bazel",
			Content: `
load("@io_bazel_rules_go//go:def.bzl", "go_library")

# An existing rule with an unmapped type is preserved
go_library(
    name = "go_default_library",
    srcs = ["unmapped_lib.go"],
    importpath = "example.com/mapkind/enabled/existing_rules",
    visibility = ["//visibility:public"],
)
`,
		},
	})
}

// TODO(jayconrod): more tests
//   run in fix mode in testdata directories to create new files
//   run in diff mode in testdata directories to update existing files (no change)
