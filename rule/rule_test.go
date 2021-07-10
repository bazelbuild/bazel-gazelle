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

package rule

import (
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	bzl "github.com/bazelbuild/buildtools/build"
)

// This file contains tests for File, Load, Rule, and related functions.
// Tests only cover some basic functionality and a few non-obvious cases.
// Most test coverage will come from clients of this package.

func TestEditAndSync(t *testing.T) {
	old := []byte(`
load("a.bzl", "x_library")

x_library(name = "foo")

load("b.bzl", y_library = "y")

y_library(name = "bar")
`)
	f, err := LoadData(filepath.Join("old", "BUILD.bazel"), "", old)
	if err != nil {
		t.Fatal(err)
	}

	loadA := f.Loads[0]
	loadA.Delete()
	loadB := f.Loads[1]
	loadB.Add("x_library")
	loadB.Remove("y_library")
	loadC := NewLoad("c.bzl")
	loadC.Add("z_library")
	loadC.Add("y_library")
	loadC.Insert(f, 3)

	foo := f.Rules[0]
	foo.Delete()
	bar := f.Rules[1]
	bar.SetAttr("srcs", []string{"bar.y"})
	baz := NewRule("z_library", "baz")
	baz.SetAttr("srcs", GlobValue{
		Patterns: []string{"**"},
		Excludes: []string{"*.pem"},
	})
	baz.Insert(f)

	got := strings.TrimSpace(string(f.Format()))
	want := strings.TrimSpace(`
load("b.bzl", "x_library")
load("c.bzl", "y_library", "z_library")

y_library(
    name = "bar",
    srcs = ["bar.y"],
)

z_library(
    name = "baz",
    srcs = glob(
        ["**"],
        exclude = ["*.pem"],
    ),
)
`)
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestPassInserted(t *testing.T) {
	old := []byte(`
load("a.bzl", "baz")

def foo():
    go_repository(name = "bar")
`)
	f, err := LoadMacroData(filepath.Join("old", "repo.bzl"), "", "foo", old)
	if err != nil {
		t.Fatal(err)
	}

	f.Rules[0].Delete()
	f.Sync()
	got := strings.TrimSpace(string(f.Format()))
	want := strings.TrimSpace(`
load("a.bzl", "baz")

def foo():
    pass
`)

	if got != want {
		t.Errorf("got:\n%s\nwant:%s", got, want)
	}
}

func TestPassRemoved(t *testing.T) {
	old := []byte(`
load("a.bzl", "baz")

def foo():
    pass
`)
	f, err := LoadMacroData(filepath.Join("old", "repo.bzl"), "", "foo", old)
	if err != nil {
		t.Fatal(err)
	}

	bar := NewRule("go_repository", "bar")
	bar.Insert(f)
	f.Sync()
	got := strings.TrimSpace(string(f.Format()))
	want := strings.TrimSpace(`
load("a.bzl", "baz")

def foo():
    go_repository(name = "bar")
`)

	if got != want {
		t.Errorf("got:\n%s\nwant:%s", got, want)
	}
}

func TestFunctionInserted(t *testing.T) {
	f, err := LoadMacroData(filepath.Join("old", "repo.bzl"), "", "foo", nil)
	if err != nil {
		t.Fatal(err)
	}

	bar := NewRule("go_repository", "bar")
	bar.Insert(f)
	f.Sync()
	got := strings.TrimSpace(string(f.Format()))
	want := strings.TrimSpace(`
def foo():
    go_repository(name = "bar")
`)

	if got != want {
		t.Errorf("got:\n%s\nwant:%s", got, want)
	}
}

func TestDeleteSyncDelete(t *testing.T) {
	old := []byte(`
x_library(name = "foo")

# comment

x_library(name = "bar")
`)
	f, err := LoadData(filepath.Join("old", "BUILD.bazel"), "", old)
	if err != nil {
		t.Fatal(err)
	}

	foo := f.Rules[0]
	bar := f.Rules[1]
	foo.Delete()
	f.Sync()
	bar.Delete()
	f.Sync()
	got := strings.TrimSpace(string(f.Format()))
	want := strings.TrimSpace(`# comment`)
	if got != want {
		t.Errorf("got:\n%s\nwant:%s", got, want)
	}
}

func TestSymbolsReturnsKeys(t *testing.T) {
	f, err := LoadData(filepath.Join("load", "BUILD.bazel"), "", []byte(`load("a.bzl", "y", z = "a")`))
	if err != nil {
		t.Fatal(err)
	}
	got := f.Loads[0].Symbols()
	want := []string{"y", "z"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v; want %#v", got, want)
	}
}

func TestLoadCommentsAreRetained(t *testing.T) {
	f, err := LoadData(filepath.Join("load", "BUILD.bazel"), "", []byte(`
load(
    "a.bzl",
    # Comment for a symbol that will be deleted.
    "baz",
    # Some comment without remapping.
    "foo",
    # Some comment with remapping.
    my_bar = "bar",
)
`))
	if err != nil {
		t.Fatal(err)
	}
	l := f.Loads[0]
	l.Remove("baz")
	f.Sync()
	l.Add("new_baz")
	f.Sync()

	got := strings.TrimSpace(string(f.Format()))
	want := strings.TrimSpace(`
load(
    "a.bzl",
    # Some comment without remapping.
    "foo",
    "new_baz",
    # Some comment with remapping.
    my_bar = "bar",
)
`)

	if got != want {
		t.Errorf("got:\n%s\nwant:%s", got, want)
	}
}

func TestKeepRule(t *testing.T) {
	for _, tc := range []struct {
		desc, src string
		want      bool
	}{
		{
			desc: "prefix",
			src: `
# keep
x_library(name = "x")
`,
			want: true,
		}, {
			desc: "compact_suffix",
			src: `
x_library(name = "x") # keep
`,
			want: true,
		}, {
			desc: "multiline_internal",
			src: `
x_library( # keep
    name = "x",
)
`,
			want: false,
		}, {
			desc: "multiline_suffix",
			src: `
x_library(
    name = "x",
) # keep
`,
			want: true,
		}, {
			desc: "after",
			src: `
x_library(name = "x")
# keep
`,
			want: false,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			f, err := LoadData(filepath.Join(tc.desc, "BUILD.bazel"), "", []byte(tc.src))
			if err != nil {
				t.Fatal(err)
			}
			if got := f.Rules[0].ShouldKeep(); got != tc.want {
				t.Errorf("got %v; want %v", got, tc.want)
			}
		})
	}
}

func TestShouldKeepExpr(t *testing.T) {
	for _, tc := range []struct {
		desc, src string
		path      func(e bzl.Expr) bzl.Expr
		want      bool
	}{
		{
			desc: "before",
			src: `
# keep
"s"
`,
			want: true,
		}, {
			desc: "suffix",
			src: `
"s" # keep
`,
			want: true,
		}, {
			desc: "after",
			src: `
"s"
# keep
`,
			want: false,
		}, {
			desc: "list_elem_prefix",
			src: `
[
    # keep
    "s",
]
`,
			path: func(e bzl.Expr) bzl.Expr { return e.(*bzl.ListExpr).List[0] },
			want: true,
		}, {
			desc: "list_elem_suffix",
			src: `
[
    "s", # keep
]
`,
			path: func(e bzl.Expr) bzl.Expr { return e.(*bzl.ListExpr).List[0] },
			want: true,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			ast, err := bzl.Parse(tc.desc, []byte(tc.src))
			if err != nil {
				t.Fatal(err)
			}
			expr := ast.Stmt[0]
			if tc.path != nil {
				expr = tc.path(expr)
			}
			got := ShouldKeep(expr)
			if got != tc.want {
				t.Errorf("got %v; want %v", got, tc.want)
			}
		})
	}
}

func TestInternalVisibility(t *testing.T) {
	var tests = []struct {
		rel      string
		expected string
	}{
		{rel: "internal", expected: "//:__subpackages__"},
		{rel: "a/b/internal", expected: "//a/b:__subpackages__"},
		{rel: "a/b/internal/c", expected: "//a/b:__subpackages__"},
		{rel: "a/b/internal/c/d", expected: "//a/b:__subpackages__"},
		{rel: "a/b/internal/c/internal", expected: "//a/b/internal/c:__subpackages__"},
		{rel: "a/b/internal/c/internal/d", expected: "//a/b/internal/c:__subpackages__"},
	}

	for _, tt := range tests {
		if actual := CheckInternalVisibility(tt.rel, "default"); actual != tt.expected {
			t.Errorf("got %v; want %v", actual, tt.expected)
		}
	}
}

func TestSortRulesByName(t *testing.T) {
	f, err := LoadMacroData(
		filepath.Join("third_party", "repos.bzl"),
		"", "repos",
		[]byte(`load("@bazel_gazelle//:deps.bzl", "go_repository")
def repos():
    go_repository(
        name = "com_github_andybalholm_cascadia",
    )
    go_repository(
        name = "com_github_bazelbuild_buildtools",
    )
    go_repository(
        name = "com_github_bazelbuild_rules_go",
    )
    go_repository(
        name = "com_github_bazelbuild_bazel_gazelle",
    )
`))
	if err != nil {
		t.Error(err)
	}
	sort.Stable(byName{
		rules: f.Rules,
		exprs: f.function.stmt.Body,
	})
	repos := []string{
		"com_github_andybalholm_cascadia",
		"com_github_bazelbuild_bazel_gazelle",
		"com_github_bazelbuild_buildtools",
		"com_github_bazelbuild_rules_go",
	}
	for i, r := range repos {
		rule := f.Rules[i]
		if rule.Name() != r {
			t.Errorf("expect rule %s at %d, got %s", r, i, rule.Name())
		}
		if rule.Index() != i {
			t.Errorf("expect rule %s with index %d, got %d", r, i, rule.Index())
		}
		if f.function.stmt.Body[i] != rule.expr {
			t.Errorf("underlying syntax tree of rule %s not sorted", r)
		}
	}
}

func TestCheckFile(t *testing.T) {
	f := File{Rules: []*Rule{
		NewRule("go_repository", "com_google_cloud_go_pubsub"),
		NewRule("go_repository", "com_google_cloud_go_pubsub"),
	}}
	err := checkFile(&f)
	if err == nil {
		t.Errorf("muliple rules with the same name should not be tolerated")
	}

	f = File{Rules: []*Rule {
		NewRule("go_rules_dependencies", ""),
		NewRule("go_register_toolchains", ""),
	}}
	err = checkFile(&f)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}