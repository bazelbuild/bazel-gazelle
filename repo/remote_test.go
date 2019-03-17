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

package repo

import (
	"testing"
)

func TestRootSpecialCases(t *testing.T) {
	for _, tc := range []struct {
		in, wantRoot, wantName string
		repos                  []Repo
		wantError              bool
	}{
		{in: "golang.org/x/net/context", wantRoot: "golang.org/x/net", wantName: "org_golang_x_net"},
		{in: "golang.org/x/tools/go/vcs", wantRoot: "golang.org/x/tools", wantName: "org_golang_x_tools"},
		{in: "golang.org/x/goimports", wantRoot: "golang.org/x/goimports", wantName: "org_golang_x_goimports"},
		{in: "cloud.google.com/fashion/industry", wantRoot: "cloud.google.com/fashion", wantName: "com_google_cloud_fashion"},
		{in: "github.com/foo", wantError: true},
		{in: "github.com/foo/bar", wantRoot: "github.com/foo/bar", wantName: "com_github_foo_bar"},
		{in: "github.com/foo/bar/baz", wantRoot: "github.com/foo/bar", wantName: "com_github_foo_bar"},
		{in: "gopkg.in/yaml.v2", wantRoot: "gopkg.in/yaml.v2", wantName: "in_gopkg_yaml_v2"},
		{in: "gopkg.in/src-d/go-git.v4", wantRoot: "gopkg.in/src-d/go-git.v4", wantName: "in_gopkg_src_d_go_git_v4"},
		{in: "unsupported.org/x/net/context", wantError: true},
		{
			in: "private.com/my/repo/package/path",
			repos: []Repo{
				{
					Name:     "com_other_host_repo",
					GoPrefix: "other-host.com/repo",
				}, {
					Name:     "com_private_my_repo",
					GoPrefix: "private.com/my/repo",
				},
			},
			wantRoot: "private.com/my/repo",
			wantName: "com_private_my_repo",
		},
		{
			in: "unsupported.org/x/net/context",
			repos: []Repo{
				{
					Name:     "com_private_my_repo",
					GoPrefix: "private.com/my/repo",
				},
			},
			wantError: true,
		},
		{
			in: "github.com/foo/bar",
			repos: []Repo{{
				Name:     "custom_repo",
				GoPrefix: "github.com/foo/bar",
			}},
			wantRoot: "github.com/foo/bar",
			wantName: "custom_repo",
		},
	} {
		t.Run(tc.in, func(t *testing.T) {
			rc := NewStubRemoteCache(tc.repos)
			if gotRoot, gotName, err := rc.Root(tc.in); err != nil {
				if !tc.wantError {
					t.Errorf("unexpected error: %v", err)
				}
			} else if tc.wantError {
				t.Errorf("unexpected success: %v", tc.in)
			} else if gotRoot != tc.wantRoot {
				t.Errorf("root for %q: got %q; want %q", tc.in, gotRoot, tc.wantRoot)
			} else if gotName != tc.wantName {
				t.Errorf("name for %q: got %q; want %q", tc.in, gotName, tc.wantName)
			}
		})
	}
}

func TestRemote(t *testing.T) {
	for _, tc := range []struct {
		desc, root          string
		repos               []Repo
		wantRemote, wantVCS string
		wantError           bool
	}{
		{
			desc:      "unstubbed_remote",
			root:      "github.com/bazelbuild/bazel-gazelle",
			wantError: true, // stub should return an error
		}, {
			desc: "known_repo",
			root: "github.com/example/project",
			repos: []Repo{{
				Name:     "com_github_example_project",
				GoPrefix: "github.com/example/project",
				Remote:   "https://private.com/example/project",
				VCS:      "git",
			}},
			wantRemote: "https://private.com/example/project",
			wantVCS:    "git",
		}, {
			desc:       "git_repo",
			root:       "example.com/repo", // stub knows this
			wantRemote: "https://example.com/repo.git",
			wantVCS:    "git",
		}, {
			desc: "local_repo",
			root: "github.com/example/project",
			repos: []Repo{{
				Name:     "com_github_example_project",
				GoPrefix: "github.com/example/project",
				Remote:   "/home/joebob/go/src/github.com/example/project",
				VCS:      "local",
			}},
			wantRemote: "/home/joebob/go/src/github.com/example/project",
			wantVCS:    "local",
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			rc := NewStubRemoteCache(tc.repos)
			if gotRemote, gotVCS, err := rc.Remote(tc.root); err != nil {
				if !tc.wantError {
					t.Errorf("unexpected error: %v", err)
				}
			} else if tc.wantError {
				t.Errorf("unexpected success")
			} else if gotRemote != tc.wantRemote {
				t.Errorf("remote for %q: got %q ; want %q", tc.root, gotRemote, tc.wantRemote)
			} else if gotVCS != tc.wantVCS {
				t.Errorf("vcs for %q: got %q ; want %q", tc.root, gotVCS, tc.wantVCS)
			}
		})
	}
}

func TestHead(t *testing.T) {
	for _, tc := range []struct {
		desc, remote, vcs   string
		wantCommit, wantTag string
		wantError           bool
	}{
		{
			desc:      "unstubbed_remote",
			remote:    "https://github.com/bazelbuild/bazel-gazelle",
			vcs:       "git",
			wantError: true, // stub should return an error
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			rc := NewStubRemoteCache(nil)
			if gotCommit, gotTag, err := rc.Head(tc.remote, tc.vcs); err != nil {
				if !tc.wantError {
					t.Errorf("unexpected error: %v", err)
				}
			} else if tc.wantError {
				t.Errorf("unexpected success")
			} else if gotCommit != tc.wantCommit {
				t.Errorf("commit for %q: got %q ; want %q", tc.remote, gotCommit, tc.wantCommit)
			} else if gotTag != tc.wantTag {
				t.Errorf("tag for %q: got %q ; want %q", tc.remote, gotTag, tc.wantTag)
			}
		})
	}
}

func TestMod(t *testing.T) {
	for _, tc := range []struct {
		desc, importPath      string
		repos                 []Repo
		wantModPath, wantName string
		wantErr               bool
	}{
		{
			desc:       "no_special_cases",
			importPath: "golang.org/x/exp",
			wantErr:    true,
		}, {
			desc:       "known",
			importPath: "example.com/known/v2/foo",
			repos: []Repo{{
				Name:     "known",
				GoPrefix: "example.com/known",
			}},
			wantModPath: "example.com/known",
			wantName:    "known",
		}, {
			desc:        "lookup",
			importPath:  "example.com/stub/v2/foo",
			wantModPath: "example.com/stub/v2",
			wantName:    "com_example_stub_v2",
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			rc := NewStubRemoteCache(tc.repos)
			modPath, name, err := rc.Mod(tc.importPath)
			if err != nil && tc.wantErr {
				return
			} else if err == nil && tc.wantErr {
				t.Error("want error; got success")
			} else if err != nil {
				t.Fatal(err)
			}
			if modPath != tc.wantModPath {
				t.Errorf("modPath: got %s; want %s", modPath, tc.wantModPath)
			}
			if name != tc.wantName {
				t.Errorf("name: got %s; want %s", name, tc.wantName)
			}
		})
	}
}
