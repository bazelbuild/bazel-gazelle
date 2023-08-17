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

// Package `test_load_for_packed_rules` generates packed
// rule of `selects.config_setting_group`.
//
// This extension is experimental and subject to change. It is not included
// in the default Gazelle binary.
package test_load_for_packed_rules

import (
	"context"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/language"
	"github.com/bazelbuild/bazel-gazelle/repo"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

const testLoadForPackedRulesName = "test_load_for_packed_rules"

type testLoadForPackedRulesLang struct {
	language.BaseLang

	Initialized, RulesGenerated, DepsResolved bool
}

var (
	_ language.Language         = (*testLoadForPackedRulesLang)(nil)
	_ language.LifecycleManager = (*testLoadForPackedRulesLang)(nil)
)

func NewLanguage() language.Language {
	return &testLoadForPackedRulesLang{}
}

var kinds = map[string]rule.KindInfo{
	"selects.config_setting_group": {
		NonEmptyAttrs: map[string]bool{"name": true},
		MergeableAttrs: map[string]bool{
			"match_all": true,
			"match_any": true,
		},
	},
}

var loads = []rule.LoadInfo{
	{
		Name: "@bazel_skylib//lib:selects.bzl",
		Symbols: []string{
			"selects",
		},
	},
}

func (*testLoadForPackedRulesLang) Name() string {
	return testLoadForPackedRulesName
}

func (*testLoadForPackedRulesLang) Kinds() map[string]rule.KindInfo {
	return kinds
}

func (*testLoadForPackedRulesLang) Loads() []rule.LoadInfo {
	return loads
}

func (l *testLoadForPackedRulesLang) Before(ctx context.Context) {
	l.Initialized = true
}

func (l *testLoadForPackedRulesLang) GenerateRules(args language.GenerateArgs) language.GenerateResult {
	if !l.Initialized {
		panic("GenerateRules must not be called before Before")
	}
	if l.RulesGenerated {
		panic("GenerateRules must not be called after DoneGeneratingRules")
	}

	r := rule.NewRule("selects.config_setting_group", "all_configs_group")

	match := []string{
		"//:config_a",
		"//:config_b",
	}

	r.SetAttr("match_all", match)

	return language.GenerateResult{
		Gen:     []*rule.Rule{r},
		Imports: []interface{}{nil},
	}
}

func (l *testLoadForPackedRulesLang) DoneGeneratingRules() {
	l.RulesGenerated = true
}

func (l *testLoadForPackedRulesLang) Resolve(c *config.Config, ix *resolve.RuleIndex, rc *repo.RemoteCache, r *rule.Rule, imports interface{}, from label.Label) {
	if !l.RulesGenerated {
		panic("Expected a call to DoneGeneratingRules before Resolve")
	}
	if l.DepsResolved {
		panic("Resolve must be called before calling AfterResolvingDeps")
	}
}

func (l *testLoadForPackedRulesLang) AfterResolvingDeps(ctx context.Context) {
	l.DepsResolved = true
}
