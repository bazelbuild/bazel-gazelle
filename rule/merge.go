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

	// Process mergeable attributes that are in dst but not in src.
	for key, dstAttr := range dst.attrs {
		// Skip if a new mergeable value is in src, it is not mergeable, or should keep the existing value
		if _, ok := src.attrs[key]; ok || !mergeable[key] || ShouldKeep(dstAttr.expr) {
			continue
		}
		if mergedValue, err := mergeAttrValues(nil, &dstAttr); err != nil {
			start, end := dstAttr.expr.RHS.Span()
			log.Printf("%s:%d.%d-%d.%d: could not merge expression", filename, start.Line, start.LineRune, end.Line, end.LineRune)
		} else if mergedValue == nil {
			dst.DelAttr(key)
		} else {
			dst.SetAttr(key, mergedValue)
		}
	}

	// Merge attributes from src into dst.
	for key, srcAttr := range src.attrs {
		if dstAttr, ok := dst.attrs[key]; !ok {
			// Not in dst, set the new attribute value
			dst.SetAttr(key, srcAttr.expr.RHS)
		} else if mergeable[key] && !ShouldKeep(dstAttr.expr) {
			// Mergeable attribute not tagged to keep, merge the attribute values.
			if mergedValue, err := mergeAttrValues(&srcAttr, &dstAttr); err != nil {
				start, end := dstAttr.expr.RHS.Span()
				log.Printf("%s:%d.%d-%d.%d: could not merge expression", filename, start.Line, start.LineRune, end.Line, end.LineRune)
			} else if mergedValue == nil {
				dst.DelAttr(key)
			} else {
				dst.SetAttr(key, mergedValue)
			}
		} else if key != "name" && key != "visibility" {
			// Not mergeable or tagged to keep. Copy the attribute and overwrite.
			// TODO: why is "name" special?
			// TODO: why is "visibility" special?
			dst.SetAttr(key, srcAttr.expr.RHS)
		}
	}

	dst.private = src.private
}

// mergeAttrValues combines information from src and dst and returns a merged
// expression. dst may be modified during this process. The returned expression
// may be different from dst when a structural change is needed.
//
// The following kinds of expressions are recognized.
//
//   - nil
//   - strings (can only be merged with strings)
//   - lists of strings
//   - a call to select with a dict argument. The dict keys must be strings,
//     and the values must be lists of strings.
//   - a list of strings combined with a select call using +. The list must
//     be the left operand.
//   - an attr value that implements the Merger interface.
//
// An error is returned if the expressions can't be merged, for example
// because they are not in one of the above formats.
func mergeAttrValues(srcAttr, dstAttr *attrValue) (bzl.Expr, error) {
	if ShouldKeep(dstAttr.expr.RHS) {
		return nil, nil
	}
	dst := dstAttr.expr.RHS
	if srcAttr == nil && (dst == nil || isScalar(dst)) {
		return nil, nil
	}
	if srcAttr != nil && isScalar(srcAttr.expr.RHS) {
		return srcAttr.expr.RHS, nil
	}

	if _, ok := dstAttr.val.(Merger); srcAttr == nil && ok {
		return nil, nil
	}

	if srcAttr != nil {
		if srcMerger, ok := srcAttr.val.(Merger); ok {
			return srcMerger.Merge(dst), nil
		}
	}
	var srcExprs platformStringsExprs
	var err error
	if srcAttr != nil {
		srcExprs, err = extractPlatformStringsExprs(srcAttr.expr.RHS)
		if err != nil {
			return nil, err
		}
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
	ps.generic = MergeList(src.generic, dst.generic)
	if ps.os, err = MergeDict(src.os, dst.os); err != nil {
		return platformStringsExprs{}, err
	}
	if ps.arch, err = MergeDict(src.arch, dst.arch); err != nil {
		return platformStringsExprs{}, err
	}
	if ps.platform, err = MergeDict(src.platform, dst.platform); err != nil {
		return platformStringsExprs{}, err
	}
	return ps, nil
}

// MergeList merges two bzl.ListExpr of strings. The lists are merged in the
// following way:
//
//   - If a string appears in both lists, it appears in the result.
//   - If a string appears in only src list, it appears in the result.
//   - If a string appears in only dst list, it is dropped from the result.
//   - If a string appears in neither list, it is dropped from the result.
//
// The result is nil if both lists are nil or empty.
//
// If the result is non-nil, it will have ForceMultiLine set if either of the
// input lists has ForceMultiLine set or if any of the strings in the result
// have a "# keep" comment.
func MergeList(srcExpr, dstExpr bzl.Expr) *bzl.ListExpr {
	src, isSrcLis := srcExpr.(*bzl.ListExpr)
	dst, isDstLis := dstExpr.(*bzl.ListExpr)
	if !isSrcLis && !isDstLis {
		return nil
	}
	if dst == nil {
		return src
	}
	if src == nil {
		src = &bzl.ListExpr{List: []bzl.Expr{}}
	}

	// Build a list of strings from the src list and keep matching strings
	// in the dst list. This preserves comments. Also keep anything with
	// a "# keep" comment, whether or not it's in the src list.
	srcSet := make(map[string]bool)
	for _, v := range src.List {
		if s := stringValue(v); s != "" {
			srcSet[s] = true
		}
	}

	var merged []bzl.Expr
	kept := make(map[string]bool)
	keepComment := false
	for _, v := range dst.List {
		s := stringValue(v)
		if keep := ShouldKeep(v); keep || srcSet[s] {
			keepComment = keepComment || keep
			merged = append(merged, v)
			if s != "" {
				kept[s] = true
			}
		}
	}

	// Add anything in the src list that wasn't kept.
	for _, v := range src.List {
		if s := stringValue(v); kept[s] {
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

// MergeDict merges two bzl.DictExpr, src and dst, where the keys are strings
// and the values are lists of strings.
//
// If both src and dst are non-nil, the keys in src are merged into dst. If both
// src and dst have the same key, the values are merged using MergeList.
// If the same key is present in both src and dst, and the values are not compatible,
// an error is returned.
func MergeDict(srcExpr, dstExpr bzl.Expr) (*bzl.DictExpr, error) {
	src, isSrcDict := srcExpr.(*bzl.DictExpr)
	dst, isDstDict := dstExpr.(*bzl.DictExpr)
	if !isSrcDict && !isDstDict {
		return nil, fmt.Errorf("expected dict, got %s and %s", srcExpr, dstExpr)
	}
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
		e.mergedValue = MergeList(e.srcValue, e.dstValue)
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
		srcValue := srcAttr.expr.RHS
		if dstAttr, ok := dst.attrs[key]; !ok {
			dst.SetAttr(key, srcValue)
		} else if !ShouldKeep(dstAttr.expr) {
			dstValue := dstAttr.expr.RHS
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
	unique       map[string]*bzl.StringExpr
	seenComments map[elemComment]bool
}

type elemComment struct {
	elem, com string
}

func makeListSquasher() listSquasher {
	return listSquasher{
		unique:       make(map[string]*bzl.StringExpr),
		seenComments: make(map[elemComment]bool),
	}
}

func (ls *listSquasher) add(s *bzl.StringExpr) {
	sCopy, ok := ls.unique[s.Value]
	if !ok {
		// Make a copy of s. We may modify it when we consolidate comments from
		// duplicate strings. We don't want to modify the original in case this
		// function fails (due to a later failed pattern match).
		sCopy = new(bzl.StringExpr)
		*sCopy = *s
		sCopy.Comments.Before = make([]bzl.Comment, 0, len(s.Comments.Before))
		sCopy.Comments.Suffix = make([]bzl.Comment, 0, len(s.Comments.Suffix))
		ls.unique[s.Value] = sCopy
	}
	for _, c := range s.Comment().Before {
		if key := (elemComment{s.Value, c.Token}); !ls.seenComments[key] {
			sCopy.Comments.Before = append(sCopy.Comments.Before, c)
			ls.seenComments[key] = true
		}
	}
	for _, c := range s.Comment().Suffix {
		if key := (elemComment{s.Value, c.Token}); !ls.seenComments[key] {
			sCopy.Comments.Suffix = append(sCopy.Comments.Suffix, c)
			ls.seenComments[key] = true
		}
	}
}

func (ls *listSquasher) list() *bzl.ListExpr {
	sortedExprs := make([]bzl.Expr, 0, len(ls.unique))
	for _, e := range ls.unique {
		sortedExprs = append(sortedExprs, e)
	}
	sort.Slice(sortedExprs, func(i, j int) bool {
		return sortedExprs[i].(*bzl.StringExpr).Value < sortedExprs[j].(*bzl.StringExpr).Value
	})
	return &bzl.ListExpr{List: sortedExprs}
}
