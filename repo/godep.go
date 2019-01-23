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
	for _, p := range file.Deps {
		repos = append(repos, Repo{
			Name:     label.ImportPathToBazelRepoName(p.ImportPath),
			GoPrefix: p.ImportPath,
			Commit:   p.Rev,
		})
	}
	return repos, nil
}
