/* Copyright 2019 The Bazel Authors. All rights reserved.

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
	"encoding/json"
	"io/ioutil"

	"github.com/bazelbuild/bazel-gazelle/label"
)

type goDepLockFile struct {
	ImportPath   string
	GoVersion    string
	GodepVersion string
	Packages     []string
	Deps         []goDepProject
}

type goDepProject struct {
	ImportPath string
	Rev        string
}

func importRepoRulesGoDep(filename string, cache *RemoteCache) ([]Repo, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	file := goDepLockFile{}

	if err := json.Unmarshal(data, &file); err != nil {
		return nil, err
	}

	var repos []Repo

	seenRepos := make(map[string]bool)
	for _, p := range file.Deps {
		repoRoot, err := cache.RepoRootForImportPath(p.ImportPath, false)
		if err != nil {
			return nil, err
		}
		if seen := seenRepos[repoRoot.Root]; !seen {

			repos = append(repos, Repo{
				Name:     label.ImportPathToBazelRepoName(repoRoot.Root),
				GoPrefix: repoRoot.Root,
				Commit:   p.Rev,
			})
			seenRepos[repoRoot.Root] = true
		}
	}
	return repos, nil
}
