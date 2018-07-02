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

package config

import (
	"reflect"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/internal/rule"
)

func TestApplyDirectives(t *testing.T) {
	for _, tc := range []struct {
		desc       string
		directives []rule.Directive
		rel        string
		want       Config
	}{
		{
			desc:       "empty build_tags",
			directives: []rule.Directive{{"build_tags", ""}},
			want:       Config{},
		}, {
			desc:       "build_tags",
			directives: []rule.Directive{{"build_tags", "foo,bar"}},
			want:       Config{GenericTags: BuildTags{"foo": true, "bar": true}},
		}, {
			desc:       "prefix",
			directives: []rule.Directive{{"prefix", "example.com/repo"}},
			rel:        "sub",
			want:       Config{GoPrefix: "example.com/repo", GoPrefixRel: "sub"},
		}, {
			desc:       "importmap_prefix",
			directives: []rule.Directive{{"importmap_prefix", "example.com/repo"}},
			rel:        "sub",
			want:       Config{GoImportMapPrefix: "example.com/repo", GoImportMapPrefixRel: "sub"},
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			c := &Config{}
			c.PreprocessTags()
			got := ApplyDirectives(c, tc.directives, tc.rel)
			tc.want.PreprocessTags()
			if !reflect.DeepEqual(*got, tc.want) {
				t.Errorf("got %#v ; want %#v", *got, tc.want)
			}
		})
	}
}
