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

package repo_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/repo"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/bazelbuild/bazel-gazelle/testtools"
)

func TestFindExternalRepo(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks not supported on windows")
	}

	dir, err := ioutil.TempDir(os.Getenv("TEST_TEMPDIR"), "TestFindExternalRepo")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	dir, err = filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}

	name := "foo"
	externalPath := filepath.Join(dir, "bazel", "output-base", "external", name)
	if err := os.MkdirAll(externalPath, 0o777); err != nil {
		t.Fatal(err)
	}

	bazelOutPath := filepath.Join(dir, "bazel", "output-base", "execroot", "test", "bazel-out")
	if err := os.MkdirAll(bazelOutPath, 0o777); err != nil {
		t.Fatal(err)
	}

	workspacePath := filepath.Join(dir, "workspace")
	if err := os.MkdirAll(workspacePath, 0o777); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(bazelOutPath, filepath.Join(workspacePath, "bazel-out")); err != nil {
		t.Fatal(err)
	}

	if got, err := repo.FindExternalRepo(workspacePath, name); err != nil {
		t.Fatal(err)
	} else if got != externalPath {
		t.Errorf("got %q ; want %q", got, externalPath)
	}
}

func TestListRepositories(t *testing.T) {
	for _, tc := range []struct {
		desc, workspace, want string
	}{
		{
			desc: "empty",
			want: "",
		}, {
			desc: "go_repository",
			workspace: `
go_repository(
    name = "custom_repo",
    commit = "123456",
    remote = "https://example.com/repo",
    importpath = "example.com/repo",
)
`,
			want: "custom_repo example.com/repo",
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			workspace, err := rule.LoadData("WORKSPACE", "", []byte(tc.workspace))
			if err != nil {
				t.Fatal(err)
			}
			repos, _, err := repo.ListRepositories(workspace)
			if err != nil {
				t.Fatal(err)
			}
			got := reposToString(repos)
			if got != tc.want {
				t.Errorf("got\n%s\n\nwant:\n%s", got, tc.want)
			}
		})
	}
}

func TestListRepositoriesWithRepositoryDirective(t *testing.T) {
	for _, tc := range []struct {
		desc, workspace, want string
	}{
		{
			desc: "empty",
			want: "",
		}, {
			desc: "git_repository",
			workspace: `
git_repository(
    name = "custom_repo",
    commit = "123456",
    remote = "https://example.com/repo",
    importpath = "example.com/repo",
)
# gazelle:repository go_repository name=custom_repo importpath=example.com/repo1
# gazelle:repository go_repository name=custom_repo_2 importpath=example.com/repo2
`,
			want: `custom_repo example.com/repo1
custom_repo_2 example.com/repo2`,
		}, {
			desc: "directive_prefer_latest",
			workspace: `
			# gazelle:repository go_repository name=custom_repo importpath=example.com/repo1
			# gazelle:repository go_repository name=custom_repo_2 importpath=example.com/repo2
			# gazelle:repository go_repository name=custom_repo importpath=example.com/repo3
`,
			want: `custom_repo example.com/repo3
custom_repo_2 example.com/repo2`,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			workspace, err := rule.LoadData("WORKSPACE", "", []byte(tc.workspace))
			if err != nil {
				t.Fatal(err)
			}
			repos, _, err := repo.ListRepositories(workspace)
			if err != nil {
				t.Fatal(err)
			}
			got := reposToString(repos)
			if got != tc.want {
				t.Errorf("got\n%s\n\nwant:\n%s", got, tc.want)
			}
		})
	}
}

func TestListRepositoriesWithRepositoryMacroDirective(t *testing.T) {
	files := []testtools.FileSpec{{
		Path: "repos1.bzl",
		Content: `
def go_repositories():
    go_repository(
        name = "go_repo",
        commit = "123456",
        remote = "https://example.com/go",
        importpath = "example.com/go",
    )

def foo_repositories():
    go_repository(
        name = "foo_repo",
        commit = "123456",
        remote = "https://example.com/foo",
        importpath = "example.com/foo",
    )
`,
	}, {
		Path: "repos2.bzl",
		Content: `
def bar_repositories():
    # gazelle:repository go_repository name=extra_repo importpath=example.com/extra
    go_repository(
        name = "bar_repo",
        commit = "123456",
        remote = "https://example.com/bar",
        importpath = "example.com/bar",
    )

def baz_repositories():
    # gazelle:repository go_repository name=ignored_repo importpath=example.com/ignored
    go_repository(
        name = "ignored_repo",
        commit = "123456",
        remote = "https://example.com/ignored",
        importpath = "example.com/ignored",
    )
`,
	}}
	dir, cleanup := testtools.CreateFiles(t, files)
	defer cleanup()
	workspaceString := `
# gazelle:repository_macro repos1.bzl%go_repositories
# gazelle:repository_macro repos1.bzl%foo_repositories
# gazelle:repository_macro repos2.bzl%bar_repositories
`
	workspace, err := rule.LoadData(dir+"/WORKSPACE", "", []byte(workspaceString))
	if err != nil {
		t.Fatal(err)
	}
	repos, _, err := repo.ListRepositories(workspace)
	if err != nil {
		t.Fatal(err)
	}
	got := reposToString(repos)
	want := `go_repo example.com/go
foo_repo example.com/foo
bar_repo example.com/bar
extra_repo example.com/extra`
	if got != want {
		t.Errorf("got\n%s\n\nwant:\n%s", got, want)
	}
}

func TestListRepositoriesWithPlusRepositoryMacroDirective(t *testing.T) {
	files := []testtools.FileSpec{{
		Path: "repos1.bzl",
		Content: `
load("repos2.bzl", "bar_repositories", alias = "alias_repositories")
def go_repositories():

    bar_repositories()
    alias()

    go_repository(
        name = "go_repo",
        commit = "123456",
        remote = "https://example.com/go",
        importpath = "example.com/go",
    )

`,
	}, {
		Path: "repos2.bzl",
		Content: `
def bar_repositories():
    go_repository(
        name = "bar_repo",
        commit = "123456",
        remote = "https://example.com/bar",
        importpath = "example.com/bar",
    )

def alias_repositories():
    go_repository(
        name = "alias_repo",
        commit = "123456",
        remote = "https://example.com/alias",
        importpath = "example.com/alias",
    )
`,
	}}
	dir, cleanup := testtools.CreateFiles(t, files)
	defer cleanup()
	workspaceString := `
# gazelle:repository_macro +repos1.bzl%go_repositories`
	workspace, err := rule.LoadData(filepath.Join(dir, "WORKSPACE"), "", []byte(workspaceString))
	if err != nil {
		t.Fatal(err)
	}
	repos, _, err := repo.ListRepositories(workspace)
	if err != nil {
		t.Fatal(err)
	}
	got := reposToString(repos)
	want := `bar_repo example.com/bar
alias_repo example.com/alias
go_repo example.com/go`
	if got != want {
		t.Errorf("got\n%s\n\nwant:\n%s", got, want)
	}
}

func reposToString(repos []*rule.Rule) string {
	buf := &strings.Builder{}
	sep := ""
	for _, r := range repos {
		fmt.Fprintf(buf, "%s%s %s", sep, r.Name(), r.AttrString("importpath"))
		sep = "\n"
	}
	return buf.String()
}
