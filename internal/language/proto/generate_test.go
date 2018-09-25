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
	"io/ioutil"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/internal/config"
	"github.com/bazelbuild/bazel-gazelle/internal/language"
	"github.com/bazelbuild/bazel-gazelle/internal/merger"
	"github.com/bazelbuild/bazel-gazelle/internal/resolve"
	"github.com/bazelbuild/bazel-gazelle/internal/rule"
	"github.com/bazelbuild/bazel-gazelle/internal/testtools"
	"github.com/bazelbuild/bazel-gazelle/internal/walk"

	bzl "github.com/bazelbuild/buildtools/build"
)

func TestGenerateRules(t *testing.T) {
	c, lang, _ := testConfig(t, "testdata")

	walk.Walk(c, []config.Configurer{lang}, []string{"testdata"}, walk.VisitAllUpdateSubdirsMode, func(dir, rel string, c *config.Config, update bool, oldFile *rule.File, subdirs, regularFiles, genFiles []string) {
		isTest := false
		for _, name := range regularFiles {
			if name == "BUILD.want" {
				isTest = true
				break
			}
		}
		if !isTest {
			return
		}
		t.Run(rel, func(t *testing.T) {
			empty, gen := lang.GenerateRules(c, dir, rel, oldFile, subdirs, regularFiles, genFiles, nil, nil)
			if len(empty) > 0 {
				t.Errorf("got %d empty rules; want 0", len(empty))
			}
			f := rule.EmptyFile("test", "")
			for _, r := range gen {
				r.Insert(f)
			}
			merger.FixLoads(f, lang.Loads())
			f.Sync()
			got := string(bzl.Format(f.File))
			wantPath := filepath.Join(dir, "BUILD.want")
			wantBytes, err := ioutil.ReadFile(wantPath)
			if err != nil {
				t.Fatalf("error reading %s: %v", wantPath, err)
			}
			want := string(wantBytes)

			if got != want {
				t.Errorf("GenerateRules %q: got:\n%s\nwant:\n%s", rel, got, want)
			}
		})
	})
}

func TestGenerateRulesEmpty(t *testing.T) {
	lang := New()
	c := config.New()
	c.Exts[protoName] = &ProtoConfig{}

	oldContent := []byte(`
proto_library(
    name = "dead_proto",
    srcs = ["foo.proto"],
)

proto_library(
    name = "live_proto",
    srcs = ["bar.proto"],
)

COMPLICATED_SRCS = ["baz.proto"]

proto_library(
    name = "complicated_proto",
    srcs = COMPLICATED_SRCS,
)
`)
	old, err := rule.LoadData("BUILD.bazel", "", oldContent)
	if err != nil {
		t.Fatal(err)
	}
	genFiles := []string{"bar.proto"}
	empty, gen := lang.GenerateRules(c, "", "foo", old, nil, nil, genFiles, nil, nil)
	if len(gen) > 0 {
		t.Errorf("got %d generated rules; want 0", len(gen))
	}
	f := rule.EmptyFile("test", "")
	for _, r := range empty {
		r.Insert(f)
	}
	f.Sync()
	got := strings.TrimSpace(string(bzl.Format(f.File)))
	want := `proto_library(name = "dead_proto")`
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestGeneratePackage(t *testing.T) {
	lang := New()
	c, _, _ := testConfig(t, "testdata")
	dir := filepath.FromSlash("testdata/protos")
	_, gen := lang.GenerateRules(c, dir, "protos", nil, nil, []string{"foo.proto"}, nil, nil, nil)
	r := gen[0]
	got := r.PrivateAttr(PackageKey).(Package)
	want := Package{
		Name: "bar.foo",
		Files: map[string]FileInfo{
			"foo.proto": {
				Path:        filepath.Join(dir, "foo.proto"),
				Name:        "foo.proto",
				PackageName: "bar.foo",
				Options:     []Option{{Key: "go_package", Value: "example.com/repo/protos"}},
				Imports: []string{
					"google/protobuf/any.proto",
					"protos/sub/sub.proto",
				},
				HasServices: true,
			},
		},
		Imports: map[string]bool{
			"google/protobuf/any.proto": true,
			"protos/sub/sub.proto":      true,
		},
		Options: map[string]string{
			"go_package": "example.com/repo/protos",
		},
		HasServices: true,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v; want %#v", got, want)
	}
}

func testConfig(t *testing.T, repoRoot string) (*config.Config, language.Language, []config.Configurer) {
	cexts := []config.Configurer{
		&config.CommonConfigurer{},
		&walk.Configurer{},
		&resolve.Configurer{},
	}
	lang := New()
	c := testtools.NewTestConfig(t, cexts, []language.Language{lang}, []string{
		"-build_file_name=BUILD.old",
		"-repo_root=" + repoRoot,
	})
	cexts = append(cexts, lang)
	return c, lang, cexts
}
