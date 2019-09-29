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

func NewStubRemoteCache(rs []Repo) *RemoteCache {
	rc, _ := NewRemoteCache(rs)
	rc.tmpDir = os.DevNull
	rc.tmpErr = errors.New("stub remote cache cannot use temp dir")
	rc.RepoRootForImportPath = stubRepoRootForImportPath
	rc.HeadCmd = stubHeadCmd
	rc.ModInfo = stubModInfo
	rc.ModVersionInfo = stubModVersionInfo
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

func stubModVersionInfo(modPath, query string) (version, sum string, err error) {
	if modPath == "example.com/known" || modPath == "example.com/unknown" {
		return "v1.2.3", "h1:abcdef", nil
	}
	return "", "", fmt.Errorf("no such module: %s", modPath)
}
