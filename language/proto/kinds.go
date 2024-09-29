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

package proto

import (
	"fmt"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

var protoKinds = map[string]rule.KindInfo{
	"proto_library": {
		MatchAttrs:    []string{"srcs"},
		NonEmptyAttrs: map[string]bool{"srcs": true},
		MergeableAttrs: map[string]bool{
			"srcs":                true,
			"import_prefix":       true,
			"strip_import_prefix": true,
		},
		ResolveAttrs: map[string]bool{"deps": true},
	},
}

func (*protoLang) Kinds() map[string]rule.KindInfo { return protoKinds }

func (pl *protoLang) Loads() []rule.LoadInfo {
	panic("ApparentLoads should be called instead")
}

func (*protoLang) ApparentLoads(moduleToApparentName func(string) string) []rule.LoadInfo {
	rulesProto := moduleToApparentName("rules_proto")
	if rulesProto == "" {
		rulesProto = "rules_proto"
	}
	return []rule.LoadInfo{
		{
			Name: fmt.Sprintf("@%s//proto:defs.bzl", rulesProto),
			Symbols: []string{
				"proto_library",
			},
		},
	}
}

func isRuleKind(c *config.Config, r *rule.Rule, expectedKind string) bool {
	kind := r.Kind()
	if kind == expectedKind {
		return true
	}

	if c == nil {
		return false
	}

	if mappedKind, ok := c.KindMap[expectedKind]; ok {
		if mappedKind.KindName == kind {
			return true
		}
	}

	return false
}
