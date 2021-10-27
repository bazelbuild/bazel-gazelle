/* Copyright 2021 The Bazel Authors. All rights reserved.

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

package rule_test

import (
	"testing"

	"github.com/bazelbuild/bazel-gazelle/rule"
)

func TestMergeRules(t *testing.T) {
	t.Run("private attributes are merged", func(t *testing.T) {
		src := rule.NewRule("go_library", "go_default_library")
		privateAttrKey := "_my_private_attr"
		privateAttrVal := "private_value"
		src.SetPrivateAttr(privateAttrKey, privateAttrVal)
		dst := rule.NewRule("go_library", "go_default_library")
		rule.MergeRules(src, dst, map[string]bool{}, "")
		if dst.PrivateAttr(privateAttrKey).(string) != privateAttrVal {
			t.Fatalf("private attributes are merged: got %v; want %s",
				dst.PrivateAttr(privateAttrKey), privateAttrVal)
		}
	})
}
