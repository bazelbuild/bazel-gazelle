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

package golang

import (
	"flag"
	"path"
	"path/filepath"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/internal/config"
	"github.com/bazelbuild/bazel-gazelle/internal/language"
	"github.com/bazelbuild/bazel-gazelle/internal/language/proto"
	"github.com/bazelbuild/bazel-gazelle/internal/rule"
)

func testConfig() (*config.Config, *flag.FlagSet, []language.Language) {
	c := config.New()
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	langs := []language.Language{proto.New(), New()}
	for _, lang := range langs {
		lang.RegisterFlags(fs, "update", c)
	}
	return c, fs, langs
}

func TestCommandLine(t *testing.T) {
	c, fs, langs := testConfig()
	args := []string{"-build_tags", "foo,bar", "-go_prefix", "example.com/repo", "-external", "vendored"}
	if err := fs.Parse(args); err != nil {
		t.Fatal(err)
	}
	for _, lang := range langs {
		if err := lang.CheckFlags(fs, c); err != nil {
			t.Fatal(err)
		}
	}
	gc := getGoConfig(c)
	for _, tag := range []string{"foo", "bar", "gc"} {
		if !gc.genericTags[tag] {
			t.Errorf("expected tag %q to be set", tag)
		}
	}
	if gc.prefix != "example.com/repo" {
		t.Errorf(`got prefix %q; want "example.com/repo"`, gc.prefix)
	}
	if gc.depMode != vendorMode {
		t.Errorf("got dep mode %v; want %v", gc.depMode, vendorMode)
	}
}

func TestDirectives(t *testing.T) {
	c, _, langs := testConfig()
	content := []byte(`
# gazelle:build_tags foo,bar
# gazelle:importmap_prefix x
# gazelle:prefix y
`)
	f, err := rule.LoadData(filepath.FromSlash("test/BUILD.bazel"), content)
	if err != nil {
		t.Fatal(err)
	}
	for _, lang := range langs {
		lang.Configure(c, "test", f)
	}
	gc := getGoConfig(c)
	for _, tag := range []string{"foo", "bar", "gc"} {
		if !gc.genericTags[tag] {
			t.Errorf("expected tag %q to be set", tag)
		}
	}
	if gc.prefix != "y" {
		t.Errorf(`got prefix %q; want "y"`, gc.prefix)
	}
	if gc.prefixRel != "test" {
		t.Errorf(`got prefixRel %q; want "test"`, gc.prefixRel)
	}
	if gc.importMapPrefix != "x" {
		t.Errorf(`got importmapPrefix %q; want "x"`, gc.importMapPrefix)
	}
	if gc.importMapPrefixRel != "test" {
		t.Errorf(`got importmapPrefixRel %q; want "test"`, gc.importMapPrefixRel)
	}
}

func TestVendorConfig(t *testing.T) {
	c, _, langs := testConfig()
	gc := getGoConfig(c)
	gc.prefix = "example.com/repo"
	gc.prefixRel = ""
	gc.importMapPrefix = "bad-importmap-prefix"
	gc.importMapPrefixRel = ""
	for _, lang := range langs {
		lang.Configure(c, "x/vendor", nil)
	}
	gc = getGoConfig(c)
	if gc.prefix != "" {
		t.Errorf(`prefix: got %q; want ""`, gc.prefix)
	}
	if gc.prefixRel != "x/vendor" {
		t.Errorf(`prefixRel: got %q; want "x/vendor"`, gc.prefixRel)
	}
	if gc.importMapPrefix != "example.com/repo/x/vendor" {
		t.Errorf(`importMapPrefix: got %q; want "example.com/repo/x/vendor"`, gc.importMapPrefix)
	}
	if gc.importMapPrefixRel != "x/vendor" {
		t.Errorf(`importMapPrefixRel: got %q; want "x/vendor"`, gc.importMapPrefixRel)
	}
}

func TestInferProtoMode(t *testing.T) {
	c, _, langs := testConfig()
	for _, tc := range []struct {
		desc, rel, content string
		old                proto.Mode
		explicit           bool
		want               proto.Mode
	}{
		{
			desc: "default_empty",
			old:  proto.DefaultMode,
			want: proto.DefaultMode,
		}, {
			desc: "default_to_legacy",
			old:  proto.DefaultMode,
			content: `
load("@io_bazel_rules_go//proto:go_proto_library.bzl", "go_proto_library")

go_proto_library(
    name = "foo_proto",
)
`,
			want: proto.LegacyMode,
		}, {
			desc: "default_to_disable",
			old:  proto.DefaultMode,
			content: `
load("@some_repo//:custom.bzl", "go_proto_library")
`,
			want: proto.DisableMode,
		}, {
			desc: "vendor_disable",
			old:  proto.DefaultMode,
			rel:  "vendor",
			want: proto.DisableMode,
		}, {
			desc:     "explicit_override_legacy",
			old:      proto.DefaultMode,
			explicit: true,
			content: `
load("@io_bazel_rules_go//proto:go_proto_library.bzl", "go_proto_library")
`,
			want: proto.DefaultMode,
		}, {
			desc:     "explicit_override_vendor",
			old:      proto.DefaultMode,
			explicit: true,
			rel:      "vendor",
			want:     proto.DefaultMode,
		}, {
			desc:     "disable_override_legacy",
			old:      proto.DisableMode,
			explicit: false,
			content: `
load("@io_bazel_rules_go//proto:go_proto_library.bzl", "go_proto_library")
`,
			want: proto.DisableMode,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			c := c.Clone()
			pc := proto.GetProtoConfig(c)
			pc.Mode = tc.old
			pc.ModeExplicit = tc.explicit
			var f *rule.File
			if tc.content != "" {
				var err error
				f, err = rule.LoadData(path.Join(tc.rel, "BUILD.bazel"), []byte(tc.content))
				if err != nil {
					t.Fatal(err)
				}
			}
			for _, lang := range langs {
				lang.Configure(c, tc.rel, f)
			}
			pc = proto.GetProtoConfig(c)
			if pc.Mode != tc.want {
				t.Errorf("got %v; want %v", pc.Mode, tc.want)
			}
		})
	}
}

func TestPreprocessTags(t *testing.T) {
	gc := newGoConfig()
	expectedTags := []string{"gc"}
	for _, tag := range expectedTags {
		if !gc.genericTags[tag] {
			t.Errorf("tag %q not set", tag)
		}
	}
	unexpectedTags := []string{"x", "cgo", "go1.8", "go1.7"}
	for _, tag := range unexpectedTags {
		if gc.genericTags[tag] {
			t.Errorf("tag %q unexpectedly set", tag)
		}
	}
}
