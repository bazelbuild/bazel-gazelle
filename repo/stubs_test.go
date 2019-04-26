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
	"errors"
	"fmt"
	"os"

	"github.com/bazelbuild/bazel-gazelle/pathtools"

	"golang.org/x/tools/go/vcs"
)

// InstallTestStubs replaces some functions with test stubs. This is useful
// for avoiding a dependency on the go command in Bazel.
func InstallTestStubs() {
	goListModules = goListModulesStub
	goModDownload = goModDownloadStub
}

func goListModulesStub(dir string) ([]byte, error) {
	return []byte(`{
	"Path": "github.com/bazelbuild/bazel-gazelle",
	"Main": true,
	"Dir": "/tmp/tmp.XxZ9HCw1Mq",
	"GoMod": "/tmp/tmp.XxZ9HCw1Mq/go.mod"
}
{
	"Path": "github.com/BurntSushi/toml",
	"Version": "v0.3.0",
	"Time": "2017-03-28T06:15:53Z",
	"Indirect": true,
	"Dir": "/usr/local/google/home/jayconrod/go/pkg/mod/github.com/!burnt!sushi/toml@v0.3.0",
	"GoMod": "/usr/local/google/home/jayconrod/go/pkg/mod/cache/download/github.com/!burnt!sushi/toml/@v/v0.3.0.mod"
}
{
	"Path": "github.com/bazelbuild/buildtools",
	"Version": "v0.0.0-20180226164855-80c7f0d45d7e",
	"Time": "2018-02-26T16:48:55Z",
	"Dir": "/usr/local/google/home/jayconrod/go/pkg/mod/github.com/bazelbuild/buildtools@v0.0.0-20180226164855-80c7f0d45d7e",
	"GoMod": "/usr/local/google/home/jayconrod/go/pkg/mod/cache/download/github.com/bazelbuild/buildtools/@v/v0.0.0-20180226164855-80c7f0d45d7e.mod"
}
{
	"Path": "github.com/davecgh/go-spew",
	"Version": "v1.1.0",
	"Time": "2016-10-29T20:57:26Z",
	"Indirect": true,
	"Dir": "/usr/local/google/home/jayconrod/go/pkg/mod/github.com/davecgh/go-spew@v1.1.0",
	"GoMod": "/usr/local/google/home/jayconrod/go/pkg/mod/cache/download/github.com/davecgh/go-spew/@v/v1.1.0.mod"
}
{
	"Path": "github.com/pelletier/go-toml",
	"Version": "v1.0.1",
	"Time": "2017-09-24T18:42:18Z",
	"Dir": "/usr/local/google/home/jayconrod/go/pkg/mod/github.com/pelletier/go-toml@v1.0.1",
	"Replace": {
		"Path": "github.com/fork/go-toml",
		"Version": "v0.0.0-20190425002759-70bc0436ed16",
		"Time": "2017-04-06T11:16:28Z",
		"Dir": "/usr/local/google/home/jayconrod/go/pkg/mod/github.com/fork/!go-toml@v1.0.1",
        "GoMod": "/usr/local/google/home/jayconrod/go/pkg/mod/cache/download/github.com/fork/go-toml@v/v1.0.1.mod"
	},
	"GoMod": "/usr/local/google/home/jayconrod/go/pkg/mod/cache/download/github.com/pelletier/go-toml/@v/v1.0.1.mod"
}
{
	"Path": "golang.org/x/tools",
	"Version": "v0.0.0-20170824195420-5d2fd3ccab98",
	"Time": "2017-08-24T19:54:20Z",
	"Dir": "/usr/local/google/home/jayconrod/go/pkg/mod/golang.org/x/tools@v0.0.0-20170824195420-5d2fd3ccab98",
	"GoMod": "/usr/local/google/home/jayconrod/go/pkg/mod/cache/download/golang.org/x/tools/@v/v0.0.0-20170824195420-5d2fd3ccab98.mod"
}
{
	"Path": "gopkg.in/check.v1",
	"Version": "v0.0.0-20161208181325-20d25e280405",
	"Time": "2016-12-08T18:13:25Z",
	"Indirect": true,
	"Dir": "/usr/local/google/home/jayconrod/go/pkg/mod/gopkg.in/check.v1@v0.0.0-20161208181325-20d25e280405",
	"GoMod": "/usr/local/google/home/jayconrod/go/pkg/mod/cache/download/gopkg.in/check.v1/@v/v0.0.0-20161208181325-20d25e280405.mod"
}
{
	"Path": "gopkg.in/yaml.v2",
	"Version": "v2.2.1",
	"Time": "2018-03-28T19:50:20Z",
	"Indirect": true,
	"Dir": "/usr/local/google/home/jayconrod/go/pkg/mod/gopkg.in/yaml.v2@v2.2.1",
	"GoMod": "/usr/local/google/home/jayconrod/go/pkg/mod/cache/download/gopkg.in/yaml.v2/@v/v2.2.1.mod"
}
`), nil
}

func goModDownloadStub(dir string, args []string) ([]byte, error) {
	return []byte(`{
	"Path": "golang.org/x/tools",
	"Version": "v0.0.0-20190122202912-9c309ee22fab",
	"Info": "/home/jay/go/pkg/mod/cache/download/golang.org/x/tools/@v/v0.0.0-20190122202912-9c309ee22fab.info",
	"GoMod": "/home/jay/go/pkg/mod/cache/download/golang.org/x/tools/@v/v0.0.0-20190122202912-9c309ee22fab.mod",
	"Zip": "/home/jay/go/pkg/mod/cache/download/golang.org/x/tools/@v/v0.0.0-20190122202912-9c309ee22fab.zip",
	"Dir": "/home/jay/go/pkg/mod/golang.org/x/tools@v0.0.0-20190122202912-9c309ee22fab",
	"Sum": "h1:FkAkwuYWQw+IArrnmhGlisKHQF4MsZ2Nu/fX4ttW55o=",
	"GoModSum": "h1:n7NCudcB/nEzxVGmLbDWY5pfWTLqBcC2KZ6jyYvM4mQ="
}
`), nil
}

func NewStubRemoteCache(rs []Repo) *RemoteCache {
	rc, _ := NewRemoteCache(rs)
	rc.tmpDir = os.DevNull
	rc.tmpErr = errors.New("stub remote cache cannot use temp dir")
	rc.RepoRootForImportPath = stubRepoRootForImportPath
	rc.HeadCmd = stubHeadCmd
	rc.ModInfo = stubModInfo
	return rc
}

// stubRepoRootForImportPath is a stub implementation of vcs.RepoRootForImportPath
func stubRepoRootForImportPath(importPath string, verbose bool) (*vcs.RepoRoot, error) {
	if pathtools.HasPrefix(importPath, "example.com/repo.git") {
		return &vcs.RepoRoot{
			VCS:  vcs.ByCmd("git"),
			Repo: "https://example.com/repo.git",
			Root: "example.com/repo.git",
		}, nil
	}

	if pathtools.HasPrefix(importPath, "example.com/repo") {
		return &vcs.RepoRoot{
			VCS:  vcs.ByCmd("git"),
			Repo: "https://example.com/repo.git",
			Root: "example.com/repo",
		}, nil
	}

	if pathtools.HasPrefix(importPath, "example.com") {
		return &vcs.RepoRoot{
			VCS:  vcs.ByCmd("git"),
			Repo: "https://example.com",
			Root: "example.com",
		}, nil
	}

	return nil, fmt.Errorf("could not resolve import path: %q", importPath)
}

func stubHeadCmd(remote, vcs string) (string, error) {
	if vcs == "git" && remote == "https://example.com/repo" {
		return "abcdef", nil
	}
	return "", fmt.Errorf("could not resolve remote: %q", remote)
}

func stubModInfo(importPath string) (string, error) {
	if pathtools.HasPrefix(importPath, "example.com/stub/v2") {
		return "example.com/stub/v2", nil
	}
	if pathtools.HasPrefix(importPath, "example.com/stub") {
		return "example.com/stub", nil
	}
	return "", fmt.Errorf("could not find module path for %s", importPath)
}
