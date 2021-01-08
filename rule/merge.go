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
	"errors"
	"fmt"
	"log"
	"sort"

	bzl "github.com/bazelbuild/buildtools/build"
)

// MergeRules copies information from src into dst, usually discarding
// information in dst when they have the same attributes.
//
// If dst is marked with a "# keep" comment, either above the rule or as
// a suffix, nothing will be changed.
//
// If src has an attribute that is not in dst, it will be copied into dst.
//
// If src and dst have the same attribute and the attribute is mergeable and the
// attribute in dst is not marked with a "# keep" comment, values in the dst
// attribute not marked with a "# keep" comment will be dropped, and values from
// src will be copied in.
//
// If dst has an attribute not in src, and the attribute is mergeable and not
// marked with a "# keep" comment, values in the attribute not marked with
// a "# keep" comment will be dropped. If the attribute is empty afterward,
// it will be deleted.
func MergeRules(src, dst *Rule, mergeable map[string]bool, filename string) {
	if dst.ShouldKeep() {
		return
	}

	// Process attributes that are in dst but not in src.
	for key, dstAttr := range dst.attrs {
		if _, ok := src.attrs[key]; ok || !mergeable[key] || ShouldKeep(dstAttr) {
			continue
		}
		dstValue := dstAttr.RHS
		if mergedValue, err := mergeExprs(nil, dstValue); err != nil {
			start, end := dstValue.Span()
			log.Printf("%s:%d.%d-%d.%d: could not merge expression", filename, start.Line, start.LineRune, end.Line, end.LineRune)
		} else if mergedValue == nil {
			dst.DelAttr(key)
		} else {
			dst.SetAttr(key, mergedValue)
		}
	}

	// Merge attributes from src into dst.
	for key, srcAttr := range src.attrs {
		srcValue := srcAttr.RHS
		if dstAttr, ok := dst.attrs[key]; !ok {
			dst.SetAttr(key, srcValue)
		} else if mergeable[key] && !ShouldKeep(dstAttr) {
			dstValue := dstAttr.RHS
			if mergedValue, err := mergeExprs(srcValue, dstValue); err != nil {
				start, end := dstValue.Span()
				log.Printf("%s:%d.%d-%d.%d: could not merge expression", filename, start.Line, start.LineRune, end.Line, end.LineRune)
			} else {
				dst.SetAttr(key, mergedValue)
			}
		}
	}
}

// mergeExprs combines information from src and dst and returns a merged
// expression. dst may be modified during this process. The returned expression
// may be different from dst when a structural change is needed.
//
// The following kinds of expressions are recognized.
//
//   * nil
//   * strings (can only be merged with strings)
//   * lists of strings
//   * a call to select with a dict argument. The dict keys must be strings,
//     and the values must be lists of strings.
//   * a list of strings combined with a select call using +. The list must
//     be the left operand.
//
// An error is returned if the expressions can't be merged, for example
// because they are not in one of the above formats.
func mergeExprs(src, dst bzl.Expr) (bzl.Expr, error) {
	if ShouldKeep(dst) {
		return nil, nil
	}
	if src == nil && (dst == nil || isScalar(dst)) {
		return nil, nil
	}
	if isScalar(src) {
		return src, nil
	}

	srcExprs, err := extractPlatformStringsExprs(src)
	if err != nil {
		return nil, err
	}
	dstExprs, err := extractPlatformStringsExprs(dst)
	if err != nil {
		return nil, err
	}
	mergedExprs, err := mergePlatformStringsExprs(srcExprs, dstExprs)
	if err != nil {
		return nil, err
	}
	return makePlatformStringsExpr(mergedExprs), nil
}

func mergePlatformStringsExprs(src, dst platformStringsExprs) (platformStringsExprs, error) {
	var ps platformStringsExprs
	var err error
	ps.generic = mergeList(src.generic, dst.generic)
	if ps.os, err = mergeDict(src.os, dst.os); err != nil {
		return platformStringsExprs{}, err
	}
	if ps.arch, err = mergeDict(src.arch, dst.arch); err != nil {
		return platformStringsExprs{}, err
	}
	if ps.platform, err = mergeDict(src.platform, dst.platform); err != nil {
		return platformStringsExprs{}, err
	}
	return ps, nil
}

func mergeList(src, dst *bzl.ListExpr) *bzl.ListExpr {
	if dst == nil {
		return src
	}
	if src == nil {
		src = &bzl.ListExpr{List: []bzl.Expr{}}
	}

	// Build a list of simple expressions from the src list and keep matching
	// equivalent expressions in the dst list. This preserves comments.
	// Also keep anything with a "# keep" comment, whether or not it's in the
	// src list.
	srcSet := make(map[simpleValue]bool)
	for _, v := range src.List {
		if sv, err := simpleValueFromExpr(v); err == nil {
			srcSet[sv] = true
		}
	}

	var merged []bzl.Expr
	kept := make(map[simpleValue]bool)
	keepComment := false
	for _, v := range dst.List {
		sv, err := simpleValueFromExpr(v)
		if keep := ShouldKeep(v); keep || srcSet[sv] {
			keepComment = keepComment || keep
			merged = append(merged, v)
			if err == nil {
				kept[sv] = true
			}
		}
	}

	// Add anything in the src list that wasn't kept.
	for _, v := range src.List {
		if sv, err := simpleValueFromExpr(v); err == nil && kept[sv] {
			continue
		}
		merged = append(merged, v)
	}

	if len(merged) == 0 {
		return nil
	}
	return &bzl.ListExpr{
		List:           merged,
		ForceMultiLine: src.ForceMultiLine || dst.ForceMultiLine || keepComment,
	}
}

func mergeDict(src, dst *bzl.DictExpr) (*bzl.DictExpr, error) {
	if dst == nil {
		return src, nil
	}
	if src == nil {
		src = &bzl.DictExpr{List: []*bzl.KeyValueExpr{}}
	}

	var entries []*dictEntry
	entryMap := make(map[string]*dictEntry)

	for _, kv := range dst.List {
		k, v, err := dictEntryKeyValue(kv)
		if err != nil {
			return nil, err
		}
		if _, ok := entryMap[k]; ok {
			return nil, fmt.Errorf("dst dict contains more than one case named %q", k)
		}
		e := &dictEntry{key: k, dstValue: v}
		entries = append(entries, e)
		entryMap[k] = e
	}

	for _, kv := range src.List {
		k, v, err := dictEntryKeyValue(kv)
		if err != nil {
			return nil, err
		}
		e, ok := entryMap[k]
		if !ok {
			e = &dictEntry{key: k}
			entries = append(entries, e)
			entryMap[k] = e
		}
		e.srcValue = v
	}

	keys := make([]string, 0, len(entries))
	haveDefault := false
	for _, e := range entries {
		e.mergedValue = mergeList(e.srcValue, e.dstValue)
		if e.key == "//conditions:default" {
			// Keep the default case, even if it's empty.
			haveDefault = true
			if e.mergedValue == nil {
				e.mergedValue = &bzl.ListExpr{}
			}
		} else if e.mergedValue != nil {
			keys = append(keys, e.key)
		}
	}
	if len(keys) == 0 && (!haveDefault || len(entryMap["//conditions:default"].mergedValue.List) == 0) {
		return nil, nil
	}
	sort.Strings(keys)
	// Always put the default case last.
	if haveDefault {
		keys = append(keys, "//conditions:default")
	}

	mergedEntries := make([]*bzl.KeyValueExpr, len(keys))
	for i, k := range keys {
		e := entryMap[k]
		mergedEntries[i] = &bzl.KeyValueExpr{
			Key:   &bzl.StringExpr{Value: e.key},
			Value: e.mergedValue,
		}
	}

	return &bzl.DictExpr{List: mergedEntries, ForceMultiLine: true}, nil
}

type dictEntry struct {
	key                             string
	dstValue, srcValue, mergedValue *bzl.ListExpr
}

// SquashRules copies information from src into dst without discarding
// information in dst. SquashRules detects duplicate elements in lists and
// dictionaries, but it doesn't sort elements after squashing. If squashing
// fails because the expression is not understood, an error is returned,
// and neither rule is modified.
func SquashRules(src, dst *Rule, filename string) error {
	if dst.ShouldKeep() {
		return nil
	}

	for key, srcAttr := range src.attrs {
		srcValue := srcAttr.RHS
		if dstAttr, ok := dst.attrs[key]; !ok {
			dst.SetAttr(key, srcValue)
		} else if !ShouldKeep(dstAttr) {
			dstValue := dstAttr.RHS
			if squashedValue, err := squashExprs(srcValue, dstValue); err != nil {
				start, end := dstValue.Span()
				return fmt.Errorf("%s:%d.%d-%d.%d: could not squash expression", filename, start.Line, start.LineRune, end.Line, end.LineRune)
			} else {
				dst.SetAttr(key, squashedValue)
			}
		}
	}
	dst.expr.Comment().Before = append(dst.expr.Comment().Before, src.expr.Comment().Before...)
	dst.expr.Comment().Suffix = append(dst.expr.Comment().Suffix, src.expr.Comment().Suffix...)
	dst.expr.Comment().After = append(dst.expr.Comment().After, src.expr.Comment().After...)
	return nil
}

func squashExprs(src, dst bzl.Expr) (bzl.Expr, error) {
	if ShouldKeep(dst) {
		return dst, nil
	}
	if isScalar(dst) {
		// may lose src, but they should always be the same.
		return dst, nil
	}
	srcExprs, err := extractPlatformStringsExprs(src)
	if err != nil {
		return nil, err
	}
	dstExprs, err := extractPlatformStringsExprs(dst)
	if err != nil {
		return nil, err
	}
	squashedExprs, err := squashPlatformStringsExprs(srcExprs, dstExprs)
	if err != nil {
		return nil, err
	}
	return makePlatformStringsExpr(squashedExprs), nil
}

func squashPlatformStringsExprs(x, y platformStringsExprs) (platformStringsExprs, error) {
	var ps platformStringsExprs
	var err error
	if ps.generic, err = squashList(x.generic, y.generic); err != nil {
		return platformStringsExprs{}, err
	}
	if ps.os, err = squashDict(x.os, y.os); err != nil {
		return platformStringsExprs{}, err
	}
	if ps.arch, err = squashDict(x.arch, y.arch); err != nil {
		return platformStringsExprs{}, err
	}
	if ps.platform, err = squashDict(x.platform, y.platform); err != nil {
		return platformStringsExprs{}, err
	}
	return ps, nil
}

func squashList(x, y *bzl.ListExpr) (*bzl.ListExpr, error) {
	if x == nil {
		return y, nil
	}
	if y == nil {
		return x, nil
	}

	ls := makeListSquasher()
	for _, e := range x.List {
		s, ok := e.(*bzl.StringExpr)
		if !ok {
			return nil, errors.New("could not squash non-string")
		}
		ls.add(s)
	}
	for _, e := range y.List {
		s, ok := e.(*bzl.StringExpr)
		if !ok {
			return nil, errors.New("could not squash non-string")
		}
		ls.add(s)
	}
	squashed := ls.list()
	squashed.Comments.Before = append(x.Comments.Before, y.Comments.Before...)
	squashed.Comments.Suffix = append(x.Comments.Suffix, y.Comments.Suffix...)
	squashed.Comments.After = append(x.Comments.After, y.Comments.After...)
	return squashed, nil
}

func squashDict(x, y *bzl.DictExpr) (*bzl.DictExpr, error) {
	if x == nil {
		return y, nil
	}
	if y == nil {
		return x, nil
	}

	cases := make(map[string]*bzl.KeyValueExpr)
	addCase := func(e bzl.Expr) error {
		kv := e.(*bzl.KeyValueExpr)
		key, ok := kv.Key.(*bzl.StringExpr)
		if !ok {
			return errors.New("could not squash non-string dict key")
		}
		if _, ok := kv.Value.(*bzl.ListExpr); !ok {
			return errors.New("could not squash non-list dict value")
		}
		if c, ok := cases[key.Value]; ok {
			if sq, err := squashList(kv.Value.(*bzl.ListExpr), c.Value.(*bzl.ListExpr)); err != nil {
				return err
			} else {
				c.Value = sq
			}
		} else {
			kvCopy := *kv
			cases[key.Value] = &kvCopy
		}
		return nil
	}

	for _, e := range x.List {
		if err := addCase(e); err != nil {
			return nil, err
		}
	}
	for _, e := range y.List {
		if err := addCase(e); err != nil {
			return nil, err
		}
	}

	keys := make([]string, 0, len(cases))
	haveDefault := false
	for k := range cases {
		if k == "//conditions:default" {
			haveDefault = true
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if haveDefault {
		keys = append(keys, "//conditions:default") // must be last
	}

	squashed := *x
	squashed.Comments.Before = append(x.Comments.Before, y.Comments.Before...)
	squashed.Comments.Suffix = append(x.Comments.Suffix, y.Comments.Suffix...)
	squashed.Comments.After = append(x.Comments.After, y.Comments.After...)
	squashed.List = make([]*bzl.KeyValueExpr, 0, len(cases))
	for _, k := range keys {
		squashed.List = append(squashed.List, cases[k])
	}
	return &squashed, nil
}

// listSquasher builds a sorted, deduplicated list of string expressions. If
// a string expression is added multiple times, comments are consolidated.
// The original expressions are not modified.
type listSquasher struct {
	unique       map[simpleValue]uniqueExpr // track unique expressions
	seenComments map[elemComment]bool       // track unique comments on expressions
}

// uniqueExpr tracks unique expressions and has a reference to their comments
type uniqueExpr struct {
	expr     bzl.Expr
	comments *bzl.Comments
}
type elemComment struct {
	elem simpleValue
	com  string
}

func makeListSquasher() listSquasher {
	return listSquasher{
		unique:       make(map[simpleValue]uniqueExpr),
		seenComments: make(map[elemComment]bool),
	}
}

func (ls *listSquasher) add(expr bzl.Expr) {
	sv, err := simpleValueFromExpr(expr)
	if err != nil {
		return
	}

	uniqueExpr, isDuplicate := ls.unique[sv]
	if !isDuplicate {
		// Make a copy of expr. We may modify it when we consolidate comments
		// from duplicate list elements. We don't want to modify the original
		// in case this function fails (due to a later failed pattern match).
		switch e := expr.(type) {
		case *bzl.StringExpr:
			stringCopy := new(bzl.StringExpr)
			*stringCopy = *e
			uniqueExpr.expr = stringCopy
			uniqueExpr.comments = &stringCopy.Comments
		case *bzl.CallExpr:
			callCopy := new(bzl.CallExpr)
			*callCopy = *e
			uniqueExpr.expr = callCopy
			uniqueExpr.comments = &callCopy.Comments
		default:
			return // bail if type is unknown
		}
		ls.unique[sv] = uniqueExpr
		uniqueExpr.comments.Before = make([]bzl.Comment, 0, len(expr.Comment().Before))
		uniqueExpr.comments.Suffix = make([]bzl.Comment, 0, len(expr.Comment().Suffix))
	}

	// merge comments on possibly deduplicated expression
	for _, c := range expr.Comment().Before {
		if key := (elemComment{sv, c.Token}); !ls.seenComments[key] {
			uniqueExpr.comments.Before = append(uniqueExpr.comments.Before, c)
			ls.seenComments[key] = true
		}
	}
	for _, c := range expr.Comment().Suffix {
		if key := (elemComment{sv, c.Token}); !ls.seenComments[key] {
			uniqueExpr.comments.Suffix = append(uniqueExpr.comments.Suffix, c)
			ls.seenComments[key] = true
		}
	}
}

func (ls *listSquasher) list() *bzl.ListExpr {
	sortedExprs := make([]bzl.Expr, 0, len(ls.unique))
	for _, e := range ls.unique {
		sortedExprs = append(sortedExprs, e.expr)
	}
	sort.Slice(sortedExprs, func(i, j int) bool {
		iExpr, _ := simpleValueFromExpr(sortedExprs[i])
		jExpr, _ := simpleValueFromExpr(sortedExprs[j])

		// call expressions after plain strings and then lexical order
		return (iExpr.symbol < jExpr.symbol) && (iExpr.str < jExpr.str)
	})
	return &bzl.ListExpr{List: sortedExprs}
}
