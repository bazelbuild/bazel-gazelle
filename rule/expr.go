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
	"fmt"
	"log"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/label"
	bzl "github.com/bazelbuild/buildtools/build"
)

// MapExprStrings applies a function to string sub-expressions within e.
// An expression containing the results with the same structure as e is
// returned.
func MapExprStrings(e bzl.Expr, f func(string) string) bzl.Expr {
	if e == nil {
		return nil
	}
	switch expr := e.(type) {
	case *bzl.StringExpr:
		s := f(expr.Value)
		if s == "" {
			return nil
		}
		ret := *expr
		ret.Value = s
		return &ret

	case *bzl.ListExpr:
		var list []bzl.Expr
		for _, elem := range expr.List {
			elem = MapExprStrings(elem, f)
			if elem != nil {
				list = append(list, elem)
			}
		}
		if len(list) == 0 && len(expr.List) > 0 {
			return nil
		}
		ret := *expr
		ret.List = list
		return &ret

	case *bzl.DictExpr:
		var cases []*bzl.KeyValueExpr
		isEmpty := true
		for _, kv := range expr.List {
			value := MapExprStrings(kv.Value, f)
			if value != nil {
				cases = append(cases, &bzl.KeyValueExpr{Key: kv.Key, Value: value})
				if key, ok := kv.Key.(*bzl.StringExpr); !ok || key.Value != "//conditions:default" {
					isEmpty = false
				}
			}
		}
		if isEmpty {
			return nil
		}
		ret := *expr
		ret.List = cases
		return &ret

	case *bzl.CallExpr:
		if x, ok := expr.X.(*bzl.Ident); !ok || x.Name != "select" || len(expr.List) != 1 {
			log.Panicf("unexpected call expression in generated imports: %#v", e)
		}
		arg := MapExprStrings(expr.List[0], f)
		if arg == nil {
			return nil
		}
		call := *expr
		call.List[0] = arg
		return &call

	case *bzl.BinaryExpr:
		x := MapExprStrings(expr.X, f)
		y := MapExprStrings(expr.Y, f)
		if x == nil {
			return y
		}
		if y == nil {
			return x
		}
		binop := *expr
		binop.X = x
		binop.Y = y
		return &binop

	default:
		return nil
	}
}

// FlattenExpr takes an expression that may have been generated from
// PlatformStrings and returns its values in a flat, sorted, de-duplicated
// list. Comments are accumulated and de-duplicated across duplicate
// expressions. If the expression could not have been generted by
// PlatformStrings, the expression will be returned unmodified.
func FlattenExpr(e bzl.Expr) bzl.Expr {
	ps, err := extractPlatformStringsExprs(e)
	if err != nil {
		return e
	}

	ls := makeListSquasher()
	addElem := func(e bzl.Expr) bool {
		s, ok := e.(*bzl.StringExpr)
		if !ok {
			return false
		}
		ls.add(s)
		return true
	}
	addList := func(e bzl.Expr) bool {
		l, ok := e.(*bzl.ListExpr)
		if !ok {
			return false
		}
		for _, elem := range l.List {
			if !addElem(elem) {
				return false
			}
		}
		return true
	}
	addDict := func(d *bzl.DictExpr) bool {
		for _, kv := range d.List {
			if !addList(kv.Value) {
				return false
			}
		}
		return true
	}

	if ps.generic != nil {
		if !addList(ps.generic) {
			return e
		}
	}
	for _, d := range []*bzl.DictExpr{ps.os, ps.arch, ps.platform} {
		if d == nil {
			continue
		}
		if !addDict(d) {
			return e
		}
	}

	return ls.list()
}

func isScalar(e bzl.Expr) bool {
	switch e.(type) {
	case *bzl.StringExpr, *bzl.LiteralExpr, *bzl.Ident:
		return true
	default:
		return false
	}
}

func dictEntryKeyValue(e bzl.Expr) (string, *bzl.ListExpr, error) {
	kv, ok := e.(*bzl.KeyValueExpr)
	if !ok {
		return "", nil, fmt.Errorf("dict entry was not a key-value pair: %#v", e)
	}
	k, ok := kv.Key.(*bzl.StringExpr)
	if !ok {
		return "", nil, fmt.Errorf("dict key was not string: %#v", kv.Key)
	}
	v, ok := kv.Value.(*bzl.ListExpr)
	if !ok {
		return "", nil, fmt.Errorf("dict value was not list: %#v", kv.Value)
	}
	return k.Value, v, nil
}

// simpleValue encapsulates information about a string or a call expression
// with a single string argument. e.g. "string" or call_expression("string")
// Some bazel language rules (e.g. rules_python) use macro expansions of this
// nature to hide mangled labels that might change from version to version.
type simpleValue struct {
	symbol, str string
}

// simpleValueFromExpr tries to create a simpleValue from an unknown bzl.Expr.
func simpleValueFromExpr(e bzl.Expr) (simpleValue, error) {
	switch expr := e.(type) {
	case *bzl.StringExpr:
		return simpleValue{str: expr.Value}, nil
	case *bzl.CallExpr:
		// Check if calling an identifier with one string argument.
		if id, ok := expr.X.(*bzl.Ident); ok && len(expr.List) == 1 {
			if arg, ok := expr.List[0].(*bzl.StringExpr); ok {
				return simpleValue{id.Name, arg.Value}, nil
			}
		}
	}

	return simpleValue{}, fmt.Errorf("expression was not a simpleValue")
}

// platformStringsExprs is a set of sub-expressions that match the structure
// of package.PlatformStrings. ExprFromValue produces expressions that
// follow this structure for srcs, deps, and other attributes, so this matches
// all non-scalar expressions generated by Gazelle.
//
// The matched expression has the form:
//
// [] + select({}) + select({}) + select({})
//
// The four collections may appear in any order, and some or all of them may
// be omitted (all fields are nil for a nil expression).
type platformStringsExprs struct {
	generic            *bzl.ListExpr
	os, arch, platform *bzl.DictExpr
}

// extractPlatformStringsExprs matches an expression and attempts to extract
// sub-expressions in platformStringsExprs. The sub-expressions can then be
// merged with corresponding sub-expressions. Any field in the returned
// structure may be nil. An error is returned if the given expression does
// not follow the pattern described by platformStringsExprs.
func extractPlatformStringsExprs(expr bzl.Expr) (platformStringsExprs, error) {
	var ps platformStringsExprs
	if expr == nil {
		return ps, nil
	}

	// Break the expression into a sequence of expressions combined with +.
	var parts []bzl.Expr
	for {
		binop, ok := expr.(*bzl.BinaryExpr)
		if !ok {
			parts = append(parts, expr)
			break
		}
		parts = append(parts, binop.Y)
		expr = binop.X
	}

	// Process each part. They may be in any order.
	for _, part := range parts {
		switch part := part.(type) {
		case *bzl.ListExpr:
			if ps.generic != nil {
				return platformStringsExprs{}, fmt.Errorf("expression could not be matched: multiple list expressions")
			}
			ps.generic = part

		case *bzl.CallExpr:
			x, ok := part.X.(*bzl.Ident)
			if !ok || x.Name != "select" || len(part.List) != 1 {
				return platformStringsExprs{}, fmt.Errorf("expression could not be matched: callee other than select or wrong number of args")
			}
			arg, ok := part.List[0].(*bzl.DictExpr)
			if !ok {
				return platformStringsExprs{}, fmt.Errorf("expression could not be matched: select argument not dict")
			}
			var dict **bzl.DictExpr
			for _, kv := range arg.List {
				k, ok := kv.Key.(*bzl.StringExpr)
				if !ok {
					return platformStringsExprs{}, fmt.Errorf("expression could not be matched: dict keys are not all strings")
				}
				if k.Value == "//conditions:default" {
					continue
				}
				key, err := label.Parse(k.Value)
				if err != nil {
					return platformStringsExprs{}, fmt.Errorf("expression could not be matched: dict key is not label: %q", k.Value)
				}
				if KnownOSSet[key.Name] {
					dict = &ps.os
					break
				}
				if KnownArchSet[key.Name] {
					dict = &ps.arch
					break
				}
				osArch := strings.Split(key.Name, "_")
				if len(osArch) != 2 || !KnownOSSet[osArch[0]] || !KnownArchSet[osArch[1]] {
					return platformStringsExprs{}, fmt.Errorf("expression could not be matched: dict key contains unknown platform: %q", k.Value)
				}
				dict = &ps.platform
				break
			}
			if dict == nil {
				// We could not identify the dict because it's empty or only contains
				// //conditions:default. We'll call it the platform dict to avoid
				// dropping it.
				dict = &ps.platform
			}
			if *dict != nil {
				return platformStringsExprs{}, fmt.Errorf("expression could not be matched: multiple selects that are either os-specific, arch-specific, or platform-specific")
			}
			*dict = arg
		}
	}
	return ps, nil
}

// makePlatformStringsExpr constructs a single expression from the
// sub-expressions in ps.
func makePlatformStringsExpr(ps platformStringsExprs) bzl.Expr {
	makeSelect := func(dict *bzl.DictExpr) bzl.Expr {
		return &bzl.CallExpr{
			X:    &bzl.Ident{Name: "select"},
			List: []bzl.Expr{dict},
		}
	}
	forceMultiline := func(e bzl.Expr) {
		switch e := e.(type) {
		case *bzl.ListExpr:
			e.ForceMultiLine = true
		case *bzl.CallExpr:
			e.List[0].(*bzl.DictExpr).ForceMultiLine = true
		}
	}

	var parts []bzl.Expr
	if ps.generic != nil {
		parts = append(parts, ps.generic)
	}
	if ps.os != nil {
		parts = append(parts, makeSelect(ps.os))
	}
	if ps.arch != nil {
		parts = append(parts, makeSelect(ps.arch))
	}
	if ps.platform != nil {
		parts = append(parts, makeSelect(ps.platform))
	}

	if len(parts) == 0 {
		return nil
	}
	if len(parts) == 1 {
		return parts[0]
	}
	expr := parts[0]
	forceMultiline(expr)
	for _, part := range parts[1:] {
		forceMultiline(part)
		expr = &bzl.BinaryExpr{
			Op: "+",
			X:  expr,
			Y:  part,
		}
	}
	return expr
}
