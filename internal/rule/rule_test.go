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
	"reflect"
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
	oldBzl, err := bzl.Parse("old", old)
	if err != nil {
		t.Fatal(err)
	}
	f := LoadFile(oldBzl)

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
	baz.Insert(f)

	f.Sync()
	got := strings.TrimSpace(string(bzl.Format(oldBzl)))
	want := strings.TrimSpace(`
load("b.bzl", "x_library")
load("c.bzl", "y_library", "z_library")

y_library(
    name = "bar",
    srcs = ["bar.y"],
)

z_library(name = "baz")
`)
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestSymbolsReturnsKeys(t *testing.T) {
	loadAst, err := bzl.Parse("load", []byte(`load("a.bzl", "y", z = "a")`))
	if err != nil {
		t.Fatal(err)
	}
	f := LoadFile(loadAst)
	got := f.Loads[0].Symbols()
	want := []string{"y", "z"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v; want %#v", got, want)
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
			ast, err := bzl.Parse(tc.desc, []byte(tc.src))
			if err != nil {
				t.Fatal(err)
			}
			f := LoadFile(ast)
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

func TestRuleAttrKeysDoesNotReturnHiddenSymbols(t *testing.T) {
	r := NewRule("go_library", "go_default_library")
	r.SetAttr("_hidden", true)
	got := r.AttrKeys()
	want := []string{"name"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v; want %#v", got, want)
	}
}

func TestRuleHiddenAttrsNotSynced(t *testing.T) {
	f := EmptyFile("BUILD")
	r := NewRule("go_library", "go_default_library")
	r.SetAttr("srcs", []string{"x.go"})
	r.SetAttr("_hidden", []string{"x.go"})
	r.Insert(f)
	f.Sync()
	got := strings.TrimSpace(string(bzl.Format(f.File)))
	want := strings.TrimSpace(`
go_library(
    name = "go_default_library",
    srcs = ["x.go"],
)
`)
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}
