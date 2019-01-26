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
	"fmt"
	"io/ioutil"
	"sync"

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

	var wg sync.WaitGroup
	var updateLock sync.Mutex
	var errorGroup *error

	roots := make(map[string]string)

	wg.Add(len(file.Deps))
	for _, p := range file.Deps {
		go func(p goDepProject) {
			defer wg.Done()
			rootRepo, _, err := cache.Root(p.ImportPath)
			if err != nil {
				if errorGroup == nil {
					errorGroup = &err
				}
			} else {
				updateLock.Lock()
				roots[p.ImportPath] = rootRepo
				updateLock.Unlock()
			}
		}(p)
	}
	wg.Wait()

	var repos []Repo
	if errorGroup != nil {
		return nil, *errorGroup
	}

	seenRepos := make(map[string]string)

	for _, p := range file.Deps {
		repoRoot := roots[p.ImportPath]
		if seen := seenRepos[repoRoot]; seen == "" {

			repos = append(repos, Repo{
				Name:     label.ImportPathToBazelRepoName(repoRoot),
				GoPrefix: repoRoot,
				Commit:   p.Rev,
			})
			seenRepos[repoRoot] = p.Rev
		} else {
			if p.Rev != seenRepos[repoRoot] {
				return nil, fmt.Errorf("Repo %s imported at multiple revisions: %s, %s", repoRoot, p.Rev, seenRepos[repoRoot])
			}
		}
	}
	return repos, nil
}
