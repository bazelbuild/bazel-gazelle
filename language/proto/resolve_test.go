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
	"path/filepath"
	"strings"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/repo"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
	bzl "github.com/bazelbuild/buildtools/build"
)

func TestResolveProto(t *testing.T) {
	type buildFile struct {
		rel, content string
	}
	type testCase struct {
		desc      string
		index     []buildFile
		old, want string
	}
	for _, tc := range []testCase{
		{
			desc: "well_known",
			index: []buildFile{{
				rel: "google/protobuf",
				content: `
proto_library(
    name = "bad_proto",
    srcs = ["any.proto"],
)
`,
			}},
			old: `
proto_library(
    name = "dep_proto",
    _imports = [
        "google/api/http.proto",
        "google/protobuf/any.proto",
        "google/rpc/status.proto",
        "google/type/latlng.proto",
    ],
)
`,
			want: `
proto_library(
    name = "dep_proto",
    deps = [
        "@com_google_protobuf//:any_proto",
        "@go_googleapis//google/api:annotations_proto",
        "@go_googleapis//google/rpc:status_proto",
        "@go_googleapis//google/type:latlng_proto",
    ],
)
`,
		}, {
			desc: "known",
			index: []buildFile{{
				rel: "google/rpc",
				content: `
proto_library(
    name = "bad_proto",
    srcs = ["status.proto"],
)
`,
			}},
			old: `
proto_library(
    name = "dep_proto",
    _imports = ["google/rpc/status.proto"],
)
`,
			want: `
proto_library(
    name = "dep_proto",
    deps = ["@go_googleapis//google/rpc:status_proto"],
)
`,
		}, {
			desc: "override",
			index: []buildFile{
				{
					rel: "google/rpc",
					content: `
proto_library(
    name = "bad_proto",
    srcs = ["status.proto"],
)
`,
				}, {
					rel: "",
					content: `
# gazelle:resolve proto google/rpc/status.proto //:good_proto
`,
				},
			},
			old: `
proto_library(
    name = "dep_proto",
    _imports = ["google/rpc/status.proto"],
)
`,
			want: `
proto_library(
    name = "dep_proto",
    deps = ["//:good_proto"],
)
`,
		}, {
			desc: "index",
			index: []buildFile{{
				rel: "foo",
				content: `
proto_library(
    name = "foo_proto",
    srcs = ["foo.proto"],
)
`,
			}},
			old: `
proto_library(
    name = "dep_proto",
    _imports = ["foo/foo.proto"],
)
`,
			want: `
proto_library(
    name = "dep_proto",
    deps = ["//foo:foo_proto"],
)
`,
		}, {
			desc: "index_local",
			old: `
proto_library(
    name = "foo_proto",
    srcs = ["foo.proto"],
)

proto_library(
    name = "dep_proto",
    _imports = ["test/foo.proto"],
)
`,
			want: `
proto_library(
    name = "foo_proto",
    srcs = ["foo.proto"],
)

proto_library(
    name = "dep_proto",
    deps = [":foo_proto"],
)
`,
		}, {
			desc: "index_ambiguous",
			index: []buildFile{{
				rel: "foo",
				content: `
proto_library(
    name = "a_proto",
    srcs = ["foo.proto"],
)

proto_library(
    name = "b_proto",
    srcs = ["foo.proto"],
)
`,
			}},
			old: `
proto_library(
    name = "dep_proto",
    _imports = ["foo/foo.proto"],
)
`,
			want: `proto_library(name = "dep_proto")`,
		}, {
			desc: "index_self",
			old: `
proto_library(
    name = "dep_proto",
    srcs = ["foo.proto"],
    _imports = ["test/foo.proto"],
)
`,
			want: `
proto_library(
    name = "dep_proto",
    srcs = ["foo.proto"],
)
`,
		}, {
			desc: "index_dedup",
			index: []buildFile{{
				rel: "foo",
				content: `
proto_library(
    name = "foo_proto",
    srcs = [
        "a.proto",
        "b.proto",
    ],
)
`,
			}},
			old: `
proto_library(
    name = "dep_proto",
    srcs = ["dep.proto"],
    _imports = [
        "foo/a.proto",
        "foo/b.proto",
    ],
)
`,
			want: `
proto_library(
    name = "dep_proto",
    srcs = ["dep.proto"],
    deps = ["//foo:foo_proto"],
)
`,
		}, {
			desc: "unknown",
			old: `
proto_library(
    name = "dep_proto",
    _imports = ["foo/bar/unknown.proto"],
)
`,
			want: `
proto_library(
    name = "dep_proto",
    deps = ["//foo/bar:bar_proto"],
)
`,
		}, {
			desc: "strip_import_prefix",
			index: []buildFile{{
				rel: "",
				content: `
# gazelle:proto_strip_import_prefix /foo/bar/
`,
			}, {
				rel: "foo/bar/sub",
				content: `
proto_library(
    name = "foo_proto",
    srcs = ["foo.proto"],
)
`,
			},
			},
			old: `
proto_library(
    name = "dep_proto",
    _imports = ["sub/foo.proto"],
)
`,
			want: `
proto_library(
    name = "dep_proto",
    deps = ["//foo/bar/sub:foo_proto"],
)
`,
		}, {
			desc: "skip bad strip_import_prefix",
			index: []buildFile{{
				rel: "",
				content: `
# gazelle:proto_strip_import_prefix /foo
`,
			}, {
				rel: "bar",
				content: `
proto_library(
    name = "foo_proto",
    srcs = ["foo.proto"],
)
`,
			},
			},
			old: `
proto_library(
    name = "dep_proto",
    _imports = ["bar/foo.proto"],
)
`,
			want: `
proto_library(
    name = "dep_proto",
    deps = ["//bar:bar_proto"],
)
`,
		}, {
			desc: "import_prefix",
			index: []buildFile{{
				rel: "",
				content: `
# gazelle:proto_import_prefix foo/
`,
			}, {
				rel: "bar",
				content: `
proto_library(
    name = "foo_proto",
    srcs = ["foo.proto"],
)
`,
			},
			},
			old: `
proto_library(
    name = "dep_proto",
    _imports = ["foo/bar/foo.proto"],
)
`,
			want: `
proto_library(
    name = "dep_proto",
    deps = ["//bar:foo_proto"],
)
`,
		}, {
			desc: "strip_import_prefix and import_prefix",
			index: []buildFile{{
				rel: "",
				content: `
# gazelle:proto_strip_import_prefix /foo
# gazelle:proto_import_prefix bar/
`,
			}, {
				rel: "foo",
				content: `
proto_library(
    name = "foo_proto",
    srcs = ["foo.proto"],
)
`,
			},
			},
			old: `
proto_library(
    name = "dep_proto",
    _imports = ["bar/foo.proto"],
)
`,
			want: `
proto_library(
    name = "dep_proto",
    deps = ["//foo:foo_proto"],
)
`,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			c, lang, cexts := testConfig(t, "testdata")
			mrslv := resolve.NewMetaResolver()
			mrslv.AddBuiltin("proto_library", lang)
			ix := resolve.NewRuleIndex(mrslv)
			rc := (*repo.RemoteCache)(nil)
			for _, bf := range tc.index {
				f, err := rule.LoadData(filepath.Join(bf.rel, "BUILD.bazel"), bf.rel, []byte(bf.content))
				if err != nil {
					t.Fatal(err)
				}
				if bf.rel == "" {
					for _, cext := range cexts {
						cext.Configure(c, "", f)
					}
				}
				for _, r := range f.Rules {
					ix.AddRule(c, r, f)
				}
			}
			f, err := rule.LoadData("test/BUILD.bazel", "test", []byte(tc.old))
			if err != nil {
				t.Fatal(err)
			}
			imports := make([]interface{}, len(f.Rules))
			for i, r := range f.Rules {
				imports[i] = convertImportsAttr(r)
				ix.AddRule(c, r, f)
			}
			ix.Finish()
			for i, r := range f.Rules {
				lang.Resolve(c, ix, rc, r, imports[i], label.New("", "test", r.Name()))
			}
			f.Sync()
			got := strings.TrimSpace(string(bzl.Format(f.File)))
			want := strings.TrimSpace(tc.want)
			if got != want {
				t.Errorf("got:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

func convertImportsAttr(r *rule.Rule) interface{} {
	value := r.AttrStrings("_imports")
	if value == nil {
		value = []string(nil)
	}
	r.DelAttr("_imports")
	return value
}
