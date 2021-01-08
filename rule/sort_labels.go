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

package rule

import (
	"sort"
	"strings"

	bzl "github.com/bazelbuild/buildtools/build"
)

// sortExprLabels sorts lists of strings using the same order as buildifier.
// Buildifier also sorts string lists, but not those involved with "select"
// expressions. This function is intended to be used with bzl.Walk.
func sortExprLabels(e bzl.Expr, _ []bzl.Expr) {
	list, ok := e.(*bzl.ListExpr)
	if !ok || len(list.List) == 0 {
		return
	}

	keys := make([]simpleValueSortKey, len(list.List))
	for i, elem := range list.List {
		sv, err := simpleValueFromExpr(elem)
		if err != nil {
			return // don't sort lists unless all elements are simple values
		}
		keys[i] = makeSortKey(i, sv, elem)
	}

	before := keys[0].x.Comment().Before
	keys[0].x.Comment().Before = nil
	sort.Sort(bySimpleValue(keys))
	keys[0].x.Comment().Before = append(before, keys[0].x.Comment().Before...)
	for i, k := range keys {
		list.List[i] = k.x
	}
}

// Code below this point is adapted from
// github.com/bazelbuild/buildtools/build/rewrite.go

// A simpleValueSortKey records information about a single simpleValue to be
// sorted. The simpleValues are first grouped into five phases:
//  - most strings
//  - strings beginning with ":"
//  - strings beginning with "//"
//  - strings beginning with "@"
//  - call expressions of the form `call_expression("string")`
// The next significant part of the comparison is the list of elements in the
// value, where elements are split at `.' and `:'. Finally we compare by value
// and break ties by original index.
// simpleValue call expressions are sorted by identifier then string literal,
// but aren't split by
type simpleValueSortKey struct {
	phase    int
	split    []string
	value    string
	original int
	x        bzl.Expr
}

const (
	phaseDefault = iota
	phaseLocal
	phaseAbsolute
	phaseExternal
	phaseSimpleCall
)

func makeSortKey(index int, sv simpleValue, x bzl.Expr) simpleValueSortKey {
	key := simpleValueSortKey{
		value:    sv.str,
		original: index,
		x:        x,
	}

	switch {
	case sv.symbol != "":
		// all simple calls are pushed to the back of the list
		key.phase = phaseSimpleCall
		key.split = append(key.split, sv.symbol, sv.str)
		return key
	case strings.HasPrefix(sv.str, ":"):
		key.phase = phaseLocal
	case strings.HasPrefix(sv.str, "//"):
		key.phase = phaseAbsolute
	case strings.HasPrefix(sv.str, "@"):
		key.phase = phaseExternal
	}

	key.split = strings.Split(strings.Replace(sv.str, ":", ".", -1), ".")
	return key
}

// bySimpleValue implements sort.Interface for a list of simpleExprSortKey.
type bySimpleValue []simpleValueSortKey

func (x bySimpleValue) Len() int      { return len(x) }
func (x bySimpleValue) Swap(i, j int) { x[i], x[j] = x[j], x[i] }

func (x bySimpleValue) Less(i, j int) bool {
	xi := x[i]
	xj := x[j]

	if xi.phase != xj.phase {
		return xi.phase < xj.phase
	}
	for k := 0; k < len(xi.split) && k < len(xj.split); k++ {
		if xi.split[k] != xj.split[k] {
			return xi.split[k] < xj.split[k]
		}
	}
	if len(xi.split) != len(xj.split) {
		return len(xi.split) < len(xj.split)
	}
	if xi.value != xj.value {
		return xi.value < xj.value
	}
	return xi.original < xj.original
}
