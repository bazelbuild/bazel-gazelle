/* Copyright 2023 The Bazel Authors. All rights reserved.

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
	"testing"

	bzl "github.com/bazelbuild/buildtools/build"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/google/go-cmp/cmp"
)

func TestExprFromValue(t *testing.T) {
	for name, tt := range map[string]struct {
		val  interface{}
		want bzl.Expr
	}{
		"glob value": {
			val: GlobValue{
				Patterns: []string{"a", "b"},
			},
			want: &bzl.CallExpr{
				X: &bzl.LiteralExpr{Token: "glob"},
				List: []bzl.Expr{
					&bzl.ListExpr{
						List: []bzl.Expr{
							&bzl.StringExpr{Value: "a"},
							&bzl.StringExpr{Value: "b"},
						},
					},
				},
			},
		},
		"glob value with excludes": {
			val: GlobValue{
				Patterns: []string{"a", "b"},
				Excludes: []string{"c", "d"},
			},
			want: &bzl.CallExpr{
				X: &bzl.LiteralExpr{Token: "glob"},
				List: []bzl.Expr{
					&bzl.ListExpr{
						List: []bzl.Expr{
							&bzl.StringExpr{Value: "a"},
							&bzl.StringExpr{Value: "b"},
						},
					},
					&bzl.AssignExpr{
						LHS: &bzl.LiteralExpr{Token: "exclude"},
						Op:  "=",
						RHS: &bzl.ListExpr{
							List: []bzl.Expr{
								&bzl.StringExpr{Value: "c"},
								&bzl.StringExpr{Value: "d"},
							},
						},
					},
				},
			},
		},
		"sorted strings": {
			val: SortedStrings{"@b", ":a", "//:target"},
			want: &bzl.ListExpr{
				List: []bzl.Expr{
					&bzl.StringExpr{Value: ":a"},
					&bzl.StringExpr{Value: "//:target"},
					&bzl.StringExpr{Value: "@b"},
				},
			},
		},
		"unsorted strings": {
			val: UnsortedStrings{"@d", ":a", "//:b"},
			want: &bzl.ListExpr{
				List: []bzl.Expr{
					&bzl.StringExpr{Value: "@d"},
					&bzl.StringExpr{Value: ":a"},
					&bzl.StringExpr{Value: "//:b"},
				},
			},
		},
		"labels": {
			val:  label.New("repo", "pkg", "name"),
			want: &bzl.StringExpr{Value: "@repo//pkg:name"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			got := ExprFromValue(tt.val)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("ExprFromValue() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
