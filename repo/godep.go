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

func importRepoRulesGoDep(filename string) ([]Repo, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	file := goDepLockFile{}

	if err := json.Unmarshal(data, &file); err != nil {
		return nil, err
	}

	var repos []Repo
	cache := NewRemoteCache(repos)

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
