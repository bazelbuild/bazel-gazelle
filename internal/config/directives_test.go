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
	bzl "github.com/bazelbuild/buildtools/build"
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

func TestInferProtoMode(t *testing.T) {
	for _, tc := range []struct {
		desc, content string
		c             Config
		rel           string
		want          ProtoMode
	}{
		{
			desc: "default",
		}, {
			desc: "previous",
			c:    Config{ProtoMode: LegacyProtoMode},
			want: LegacyProtoMode,
		}, {
			desc: "explicit",
			content: `# gazelle:proto default

load("@io_bazel_rules_go//proto:go_proto_library.bzl", "go_proto_library")
`,
			want: DefaultProtoMode,
		}, {
			desc:    "explicit_no_override",
			content: `load("@io_bazel_rules_go//proto:go_proto_library.bzl", "go_proto_library")`,
			c: Config{
				ProtoMode:         DefaultProtoMode,
				ProtoModeExplicit: true,
			},
			want: DefaultProtoMode,
		}, {
			desc: "vendor",
			rel:  "vendor",
			want: DisableProtoMode,
		}, {
			desc:    "legacy",
			content: `load("@io_bazel_rules_go//proto:go_proto_library.bzl", "go_proto_library")`,
			want:    LegacyProtoMode,
		}, {
			desc:    "disable",
			content: `load("@com_example_repo//proto:go_proto_library.bzl", go_proto_library = "x")`,
			want:    DisableProtoMode,
		}, {
			desc:    "fix legacy",
			content: `load("@io_bazel_rules_go//proto:go_proto_library.bzl", "go_proto_library")`,
			c:       Config{ShouldFix: true},
		}, {
			desc:    "fix disabled",
			content: `load("@com_example_repo//proto:go_proto_library.bzl", go_proto_library = "x")`,
			c:       Config{ShouldFix: true},
			want:    DisableProtoMode,
		}, {
			desc: "well known types",
			c:    Config{GoPrefix: "github.com/golang/protobuf"},
			want: LegacyProtoMode,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			var f *bzl.File
			var directives []rule.Directive
			if tc.content != "" {
				var err error
				f, err = bzl.Parse("BUILD.bazel", []byte(tc.content))
				if err != nil {
					t.Fatalf("error parsing build file: %v", err)
				}
				directives = rule.ParseDirectives(f)
			}

			got := InferProtoMode(&tc.c, tc.rel, f, directives)
			if got.ProtoMode != tc.want {
				t.Errorf("got proto mode %v ; want %v", got.ProtoMode, tc.want)
			}
		})
	}
}
