/* Copyright 2016 The Bazel Authors. All rights reserved.

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

// Package merger provides methods for merging parsed BUILD files.
package merger

import (
	"fmt"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/internal/config"
	"github.com/bazelbuild/bazel-gazelle/internal/rule"
)

var (
	// PreResolveAttrs is the set of attributes that should be merged before
	// dependency resolution, i.e., everything except deps.
	PreResolveAttrs config.MergeableAttrs

	// PostResolveAttrs is the set of attributes that should be merged after
	// dependency resolution, i.e., deps.
	PostResolveAttrs config.MergeableAttrs

	// BuildAttrs is the union of PreResolveAttrs and PostResolveAttrs.
	BuildAttrs config.MergeableAttrs

	// RepoAttrs is the set of attributes that should be merged in repository
	// rules in WORKSPACE.
	RepoAttrs config.MergeableAttrs

	// NonEmptyAttrs is the set of attributes that disqualify a rule from being
	// deleted after merge.
	NonEmptyAttrs config.MergeableAttrs
)

func init() {
	PreResolveAttrs = make(config.MergeableAttrs)
	PostResolveAttrs = make(config.MergeableAttrs)
	RepoAttrs = make(config.MergeableAttrs)
	NonEmptyAttrs = make(config.MergeableAttrs)
	for _, set := range []struct {
		mergeableAttrs config.MergeableAttrs
		kinds, attrs   []string
	}{
		{
			mergeableAttrs: PreResolveAttrs,
			kinds: []string{
				"go_library",
				"go_binary",
				"go_test",
				"go_proto_library",
				"proto_library",
				"filegroup",
			},
			attrs: []string{
				"srcs",
			},
		}, {
			mergeableAttrs: PreResolveAttrs,
			kinds: []string{
				"go_library",
				"go_proto_library",
			},
			attrs: []string{
				"importpath",
				"importmap",
			},
		}, {
			mergeableAttrs: PreResolveAttrs,
			kinds: []string{
				"go_library",
				"go_binary",
				"go_test",
				"go_proto_library",
			},
			attrs: []string{
				"cgo",
				"clinkopts",
				"copts",
				"embed",
			},
		}, {
			mergeableAttrs: PreResolveAttrs,
			kinds: []string{
				"go_proto_library",
			},
			attrs: []string{
				"proto",
			},
		}, {
			mergeableAttrs: PostResolveAttrs,
			kinds: []string{
				"go_library",
				"go_binary",
				"go_test",
				"go_proto_library",
				"proto_library",
			},
			attrs: []string{
				"deps",
			},
		}, {
			mergeableAttrs: RepoAttrs,
			kinds: []string{
				"go_repository",
			},
			attrs: []string{
				"commit",
				"importpath",
				"remote",
				"sha256",
				"strip_prefix",
				"tag",
				"type",
				"urls",
				"vcs",
			},
		}, {
			mergeableAttrs: NonEmptyAttrs,
			kinds: []string{
				"go_binary",
				"go_library",
				"go_test",
				"proto_library",
				"filegroup",
			},
			attrs: []string{
				"srcs",
			},
		}, {
			mergeableAttrs: NonEmptyAttrs,
			kinds: []string{
				"go_binary",
				"go_library",
				"go_test",
				"proto_library",
			},
			attrs: []string{
				"deps",
			},
		}, {
			mergeableAttrs: NonEmptyAttrs,
			kinds: []string{
				"go_binary",
				"go_library",
				"go_test",
			},
			attrs: []string{
				"embed",
			},
		}, {
			mergeableAttrs: NonEmptyAttrs,
			kinds: []string{
				"go_proto_library",
			},
			attrs: []string{
				"proto",
			},
		},
	} {
		for _, kind := range set.kinds {
			if set.mergeableAttrs[kind] == nil {
				set.mergeableAttrs[kind] = make(map[string]bool)
			}
			for _, attr := range set.attrs {
				set.mergeableAttrs[kind][attr] = true
			}
		}
	}
	BuildAttrs = make(config.MergeableAttrs)
	for _, mattrs := range []config.MergeableAttrs{PreResolveAttrs, PostResolveAttrs} {
		for kind, attrs := range mattrs {
			if BuildAttrs[kind] == nil {
				BuildAttrs[kind] = make(map[string]bool)
			}
			for attr := range attrs {
				BuildAttrs[kind][attr] = true
			}
		}
	}
}

// MergeFile merges the rules in genRules with matching rules in f and
// adds unmatched rules to the end of the merged file. MergeFile also merges
// rules in empty with matching rules in f and deletes rules that
// are empty after merging. attrs is the set of attributes to merge. Attributes
// not in this set will be left alone if they already exist.
func MergeFile(oldFile *rule.File, emptyRules, genRules []*rule.Rule, attrs config.MergeableAttrs) {
	// Merge empty rules into the file and delete any rules which become empty.
	for _, emptyRule := range emptyRules {
		if oldRule, _ := match(oldFile.Rules, emptyRule); oldRule != nil {
			rule.MergeRules(emptyRule, oldRule, attrs, oldFile.Path)
			if oldRule.IsEmpty(NonEmptyAttrs) {
				oldRule.Delete()
			}
		}
	}
	oldFile.Sync()

	// Match generated rules with existing rules in the file. Keep track of
	// rules with non-standard names.
	matchRules := make([]*rule.Rule, len(genRules))
	matchErrors := make([]error, len(genRules))
	substitutions := make(map[string]string)
	for i, genRule := range genRules {
		oldRule, err := match(oldFile.Rules, genRule)
		if err != nil {
			// TODO(jayconrod): add a verbose mode and log errors. They are too chatty
			// to print by default.
			matchErrors[i] = err
			continue
		}
		matchRules[i] = oldRule
		if oldRule != nil {
			if oldRule.Name() != genRule.Name() {
				substitutions[genRule.Name()] = oldRule.Name()
			}
		}
	}

	// Rename labels in generated rules that refer to other generated rules.
	if len(substitutions) > 0 {
		for _, genRule := range genRules {
			substituteRule(genRule, substitutions)
		}
	}

	// Merge generated rules with existing rules or append to the end of the file.
	for i, genRule := range genRules {
		if matchErrors[i] != nil {
			continue
		}
		if matchRules[i] == nil {
			genRule.Insert(oldFile)
		} else {
			rule.MergeRules(genRule, matchRules[i], attrs, oldFile.Path)
		}
	}
}

// substituteAttrs contains a list of attributes for each kind that should be
// processed by substituteRule and substituteExpr. Note that "name" does not
// need to be substituted since it's not mergeable.
var substituteAttrs = map[string][]string{
	"go_binary":        {"embed"},
	"go_library":       {"embed"},
	"go_test":          {"embed"},
	"go_proto_library": {"proto"},
}

// substituteRule replaces local labels (those beginning with ":", referring to
// targets in the same package) according to a substitution map. This is used
// to update generated rules before merging when the corresponding existing
// rules have different names. If substituteRule replaces a string, it returns
// a new expression; it will not modify the original expression.
func substituteRule(r *rule.Rule, substitutions map[string]string) {
	for _, attr := range substituteAttrs[r.Kind()] {
		if expr := r.Attr(attr); expr != nil {
			expr = rule.MapExprStrings(expr, func(s string) string {
				if rename, ok := substitutions[strings.TrimPrefix(s, ":")]; ok {
					return ":" + rename
				} else {
					return s
				}
			})
			r.SetAttr(attr, expr)
		}
	}
}

// matchAttrs contains lists of attributes for each kind that are used in
// matching. For example, importpath attributes can be used to match go_library
// rules, even when the names are different.
var matchAttrs = map[string][]string{
	"go_library":       {"importpath"},
	"go_proto_library": {"importpath"},
	"go_repository":    {"importpath"},
}

// matchAny is a set of kinds which may be matched regardless of attributes.
// For example, if there is only one go_binary in a package, any go_binary
// rule will match.
var matchAny = map[string]bool{"go_binary": true}

// match searches for a rule that can be merged with x in rules.
//
// A rule is considered a match if its kind is equal to x's kind AND either its
// name is equal OR at least one of the attributes in matchAttrs is equal.
//
// If there are no matches, nil and nil are returned.
//
// If a rule has the same name but a different kind, nill and an error
// are returned.
//
// If there is exactly one match, the rule and nil are returned.
//
// If there are multiple matches, match will attempt to disambiguate, based on
// the quality of the match (name match is best, then attribute match in the
// order that attributes are listed). If disambiguation is successful,
// the rule and nil are returned. Otherwise, nil and an error are returned.
func match(rules []*rule.Rule, x *rule.Rule) (*rule.Rule, error) {
	xname := x.Name()
	xkind := x.Kind()
	var nameMatches []*rule.Rule
	var kindMatches []*rule.Rule
	for _, y := range rules {
		if xname == y.Name() {
			nameMatches = append(nameMatches, y)
		}
		if xkind == y.Kind() {
			kindMatches = append(kindMatches, y)
		}
	}

	if len(nameMatches) == 1 {
		y := nameMatches[0]
		if xkind != y.Kind() {
			return nil, fmt.Errorf("could not merge %s(%s): a rule of the same name has kind %s", xkind, xname, y.Kind())
		}
		return y, nil
	}
	if len(nameMatches) > 1 {
		return nil, fmt.Errorf("could not merge %s(%s): multiple rules have the same name", xkind, xname)
	}

	attrs := matchAttrs[xkind]
	for _, key := range attrs {
		var attrMatches []*rule.Rule
		xvalue := x.AttrString(key)
		if xvalue == "" {
			continue
		}
		for _, y := range kindMatches {
			if xvalue == y.AttrString(key) {
				attrMatches = append(attrMatches, y)
			}
		}
		if len(attrMatches) == 1 {
			return attrMatches[0], nil
		} else if len(attrMatches) > 1 {
			return nil, fmt.Errorf("could not merge %s(%s): multiple rules have the same attribute %s = %q", xkind, xname, key, xvalue)
		}
	}

	if matchAny[xkind] {
		if len(kindMatches) == 1 {
			return kindMatches[0], nil
		} else if len(kindMatches) > 1 {
			return nil, fmt.Errorf("could not merge %s(%s): multiple rules have the same kind but different names", xkind, xname)
		}
	}

	return nil, nil
}
