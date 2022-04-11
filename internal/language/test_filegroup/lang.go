/* Copyright 2019 The Bazel Authors. All rights reserved.

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

// Package test_filegroup generates an "all_files" filegroup target
// in each package. This target globs files in the same package and
// depends on subpackages.
//
// These rules are used for testing with go_bazel_test.
//
// This extension is experimental and subject to change. It is not included
// in the default Gazelle binary.
package test_filegroup

import (
	"flag"
	"path"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/language"
	"github.com/bazelbuild/bazel-gazelle/repo"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

const testFilegroupName = "test_filegroup"

type testFilegroupLang struct{}

func NewLanguage() language.Language {
	return &testFilegroupLang{}
}

func (*testFilegroupLang) Name() string { return testFilegroupName }

func (*testFilegroupLang) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {}

func (*testFilegroupLang) CheckFlags(fs *flag.FlagSet, c *config.Config) error { return nil }

func (*testFilegroupLang) KnownDirectives() []string { return nil }

func (*testFilegroupLang) Configure(c *config.Config, rel string, f *rule.File) {}

func (*testFilegroupLang) Kinds() map[string]rule.KindInfo {
	return kinds
}

func (*testFilegroupLang) Loads() []rule.LoadInfo { return nil }

func (*testFilegroupLang) Fix(c *config.Config, f *rule.File) {}

func (*testFilegroupLang) Imports(c *config.Config, r *rule.Rule, f *rule.File) []resolve.ImportSpec {
	return nil
}

func (*testFilegroupLang) Embeds(r *rule.Rule, from label.Label) []label.Label { return nil }

func (*testFilegroupLang) Resolve(c *config.Config, ix *resolve.RuleIndex, rc *repo.RemoteCache, r *rule.Rule, imports interface{}, from label.Label) {
}

var kinds = map[string]rule.KindInfo{
	"filegroup": {
		NonEmptyAttrs:  map[string]bool{"srcs": true, "deps": true},
		MergeableAttrs: map[string]bool{"srcs": true},
	},
}

func (*testFilegroupLang) GenerateRules(args language.GenerateArgs) language.GenerateResult {
	r := rule.NewRule("filegroup", "all_files")
	srcs := make([]string, 0, len(args.Subdirs)+len(args.RegularFiles))
	srcs = append(srcs, args.RegularFiles...)
	for _, f := range args.Subdirs {
		pkg := path.Join(args.Rel, f)
		srcs = append(srcs, "//"+pkg+":all_files")
	}
	r.SetAttr("srcs", srcs)
	r.SetAttr("testonly", true)
	if args.File == nil || !args.File.HasDefaultVisibility() {
		r.SetAttr("visibility", []string{"//visibility:public"})
	}
	return language.GenerateResult{
		Gen:     []*rule.Rule{r},
		Imports: []interface{}{nil},
	}
}
