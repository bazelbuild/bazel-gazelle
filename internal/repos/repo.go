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

package repos

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/bazelbuild/bazel-gazelle/internal/rules"
	bf "github.com/bazelbuild/buildtools/build"
)

type repo struct {
	name       string
	importPath string
	commit     string
	remote     string
}

type byName []repo

func (s byName) Len() int           { return len(s) }
func (s byName) Less(i, j int) bool { return s[i].name < s[j].name }
func (s byName) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

type lockFileFormat int

const (
	unknownFormat lockFileFormat = iota
	depFormat
)

var lockFileParsers = map[lockFileFormat]func(string) ([]repo, error){
	depFormat: importRepoRulesDep,
}

// ImportRepoRules reads the lock file of a vendoring tool and returns
// a list of equivalent repository rules that can be merged into a WORKSPACE
// file. The format of the file is inferred from its basename. Currently,
// only Gopkg.lock is supported.
func ImportRepoRules(filename string) ([]bf.Expr, error) {
	format := getLockFileFormat(filename)
	if format == unknownFormat {
		return nil, fmt.Errorf(`%s: unrecognized lock file format. Expected "Gopkg.lock"`, filename)
	}
	parser := lockFileParsers[format]
	repos, err := parser(filename)
	if err != nil {
		return nil, fmt.Errorf("error parsing %q: %v", filename, err)
	}
	sort.Stable(byName(repos))

	rules := make([]bf.Expr, 0, len(repos))
	for _, repo := range repos {
		rules = append(rules, generateRepoRule(repo))
	}
	return rules, nil
}

func getLockFileFormat(filename string) lockFileFormat {
	switch filepath.Base(filename) {
	case "Gopkg.lock":
		return depFormat
	default:
		return unknownFormat
	}
}

func generateRepoRule(repo repo) bf.Expr {
	attrs := []rules.KeyValue{
		{Key: "name", Value: repo.name},
		{Key: "commit", Value: repo.commit},
		{Key: "importpath", Value: repo.importPath},
	}
	if repo.remote != "" {
		attrs = append(attrs, rules.KeyValue{Key: "remote", Value: repo.remote})
	}
	return rules.NewRule("go_repository", attrs)
}
