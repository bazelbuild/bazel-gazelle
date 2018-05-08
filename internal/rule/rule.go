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
	"sort"

	bzl "github.com/bazelbuild/buildtools/build"
	bt "github.com/bazelbuild/buildtools/tables"
)

// EmptyRule generates an empty rule with the given kind and name.
func EmptyRule(kind, name string) *bzl.CallExpr {
	return NewRule(kind, []KeyValue{{"name", name}})
}

// NewRule generates a rule of the given kind with the given attributes.
func NewRule(kind string, kwargs []KeyValue) *bzl.CallExpr {
	sort.Sort(byAttrName(kwargs))

	var list []bzl.Expr
	for _, arg := range kwargs {
		expr := ExprFromValue(arg.Value)
		list = append(list, &bzl.BinaryExpr{
			X:  &bzl.LiteralExpr{Token: arg.Key},
			Op: "=",
			Y:  expr,
		})
	}

	return &bzl.CallExpr{
		X:    &bzl.LiteralExpr{Token: kind},
		List: list,
	}
}

type byAttrName []KeyValue

var _ sort.Interface = byAttrName{}

func (s byAttrName) Len() int {
	return len(s)
}

func (s byAttrName) Less(i, j int) bool {
	if cmp := bt.NamePriority[s[i].Key] - bt.NamePriority[s[j].Key]; cmp != 0 {
		return cmp < 0
	}
	return s[i].Key < s[j].Key
}

func (s byAttrName) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
