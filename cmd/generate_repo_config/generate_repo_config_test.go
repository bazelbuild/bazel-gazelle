/* Copyright 2016 The Bazel Authors. All rights reserved.

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

package main

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/testtools"
)

func TestGenerateRepoConfig(t *testing.T) {
	files := []testtools.FileSpec{
		{
			Path: "WORKSPACE",
			Content: `
# gazelle:repo test
go_repository(
    name = "com_github_pkg_errors",
    build_file_generation = "off",
    commit = "645ef00459ed84a119197bfb8d8205042c6df63d",
    importpath = "github.com/pkg/errors",
)
# gazelle:repository_macro repositories.bzl%go_repositories
`,
		}, {
			Path: "repositories.bzl",
			Content: `
load("@bazel_gazelle//:deps.bzl", "go_repository")
# gazelle:repo test2
def go_repositories():
    go_repository(
        name = "org_golang_x_net",
        importpath = "golang.org/x/net",
        tag = "1.2",
    )
    # keep
    go_repository(
        name = "org_golang_x_sys",
        importpath = "golang.org/x/sys",
        remote = "https://github.com/golang/sys",
    )
`,
		},
	}
	dir, cleanup := testtools.CreateFiles(t, files)
	defer cleanup()

	tmp, err := ioutil.TempDir(os.Getenv("TEST_TEMPDIR"), "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)

	if err := generateRepoConfig(tmp+"/WORKSPACE", dir+"/WORKSPACE"); err != nil {
		t.Fatal(err)
	}

	testtools.CheckFiles(t, tmp, []testtools.FileSpec{
		{
			Path: "WORKSPACE",
			Content: `
# Code generated by generate_repo_config.go; DO NOT EDIT.
# gazelle:repo test
# gazelle:repo test2

go_repository(
    name = "com_github_pkg_errors",
    importpath = "github.com/pkg/errors",
)

go_repository(
    name = "org_golang_x_net",
    importpath = "golang.org/x/net",
)

go_repository(
    name = "org_golang_x_sys",
    importpath = "golang.org/x/sys",
)
`,
		},
	})
}

func TestEmptyConfig(t *testing.T) {
	tmp, err := ioutil.TempDir(os.Getenv("TEST_TEMPDIR"), "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)

	if err := generateRepoConfig(tmp+"/WORKSPACE", ""); err != nil {
		t.Fatal(err)
	}

	testtools.CheckFiles(t, tmp, []testtools.FileSpec{
		{
			Path: "WORKSPACE",
			Content: `
# Code generated by generate_repo_config.go; DO NOT EDIT.

`,
		},
	})
}
