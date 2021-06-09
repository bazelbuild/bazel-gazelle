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

// Package repo provides functionality for managing Go repository rules.
//
// UNSTABLE: The exported APIs in this package may change. In the future,
// language extensions should implement an interface for repository
// rule management. The update-repos command will call interface methods,
// and most if this package's functionality will move to language/go.
// Moving this package to an internal directory would break existing
// extensions, since RemoteCache is referenced through the resolve.Resolver
// interface, which extensions are required to implement.
package repo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/rule"
)

type byRuleName []*rule.Rule

func (s byRuleName) Len() int           { return len(s) }
func (s byRuleName) Less(i, j int) bool { return s[i].Name() < s[j].Name() }
func (s byRuleName) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// FindExternalRepo attempts to locate the directory where Bazel has fetched
// the external repository with the given name. An error is returned if the
// repository directory cannot be located.
func FindExternalRepo(repoRoot, name string) (string, error) {
	// See https://docs.bazel.build/versions/master/output_directories.html
	// for documentation on Bazel directory layout.
	// We expect the bazel-out symlink in the workspace root directory to point to
	// <output-base>/execroot/<workspace-name>/bazel-out
	// We expect the external repository to be checked out at
	// <output-base>/external/<name>
	// Note that users can change the prefix for most of the Bazel symlinks with
	// --symlink_prefix, but this does not include bazel-out.
	externalPath := strings.Join([]string{repoRoot, "bazel-out", "..", "..", "..", "external", name}, string(os.PathSeparator))
	cleanPath, err := filepath.EvalSymlinks(externalPath)
	if err != nil {
		return "", err
	}
	st, err := os.Stat(cleanPath)
	if err != nil {
		return "", err
	}
	if !st.IsDir() {
		return "", fmt.Errorf("%s: not a directory", externalPath)
	}
	return cleanPath, nil
}

// ListRepositories extracts metadata about repositories declared in a
// file.
func ListRepositories(workspace *rule.File) (repos []*rule.Rule, repoFileMap map[string]*rule.File, err error) {
	repoIndexMap := make(map[string]int)
	repoFileMap = make(map[string]*rule.File)
	checked := make(map[string]bool)
	for _, repo := range workspace.Rules {
		if name := repo.Name(); name != "" {
			repos = append(repos, repo)
			repoFileMap[name] = workspace
			repoIndexMap[name] = len(repos) - 1
		}
	}
	repos, err = loadExtraRepos(workspace, repos, repoFileMap, repoIndexMap)
	if err != nil {
		return nil, nil, err
	}

	for _, d := range workspace.Directives {
		switch d.Key {
		case "repository_macro":
			f, defName, err := parseRepositoryMacroDirective(d.Value)
			if err != nil {
				return nil, nil, err
			}
			leveled := strings.HasPrefix(f, "+")
			f = strings.TrimPrefix(f, "+")
			f = filepath.Join(filepath.Dir(workspace.Path), filepath.Clean(f))
			checked[f+"%"+defName] = true

			la := &loadArgs{
				leveled: leveled,
				workspace: workspace.Path,
				f: f,
				defName: defName,
				repos: repos,
				repoFileMap: repoFileMap,
				repoIndexMap: repoIndexMap,
				checked: checked,
			}

			if err := loadRepositoriesFromMacro(la); err != nil {
				return nil, nil, err
			}
			repos = la.repos
		}
	}
	return repos, repoFileMap, nil
}

type loadArgs struct {
	leveled bool
	workspace string
	f string
	defName string
	repos []*rule.Rule
	repoFileMap map[string]*rule.File
	repoIndexMap map[string]int
	checked map[string]bool
}

func loadRepositoriesFromMacro(la *loadArgs) error {
	macroFile, err := rule.LoadMacroFile(la.f, "", la.defName)
	if err != nil {
		return err
	}
	for _, repo := range macroFile.Rules {
		if name := repo.Name(); name != "" {
				la.repos = append(la.repos, repo)
				la.repoFileMap[name] = macroFile
				la.repoIndexMap[name] = len(la.repos) - 1
		} else if la.leveled {
			// This block will only be entered if leveled==true
			var callFile string
			kind := repo.Kind()
			for _, l := range macroFile.Loads {
				if l.Has(kind) {
					callFile = filepath.Join(filepath.Dir(la.workspace), filepath.Clean(l.Name()))
					la.defName = l.Unalias(kind)
					break
				}
			}
			// TODO: Also handle the case where one macro calls another macro in the same bzl file
			if len(callFile) == 0 {
				continue
			}
			if !la.checked[la.f+"%"+la.defName] {
				la.f = callFile
				la.checked[la.f+"%"+la.defName] = true
				if err := loadRepositoriesFromMacro(la); err != nil {
					return err
				}
			}
		}
	}
	la.repos, err = loadExtraRepos(macroFile, la.repos, la.repoFileMap, la.repoIndexMap)
	return err
}

func loadExtraRepos(f *rule.File, repos []*rule.Rule, repoFileMap map[string]*rule.File, repoIndexMap map[string]int) ([]*rule.Rule, error) {
	extraRepos, err := parseRepositoryDirectives(f.Directives)
	if err != nil {
		return nil, err
	}
	for _, repo := range extraRepos {
		if i, ok := repoIndexMap[repo.Name()]; ok {
			repos[i] = repo
		} else {
			repos = append(repos, repo)
		}
		repoFileMap[repo.Name()] = f
	}
	return repos, nil
}

func parseRepositoryDirectives(directives []rule.Directive) (repos []*rule.Rule, err error) {
	for _, d := range directives {
		switch d.Key {
		case "repository":
			vals := strings.Fields(d.Value)
			if len(vals) < 2 {
				return nil, fmt.Errorf("failure parsing repository: %s, expected repository kind and attributes", d.Value)
			}
			kind := vals[0]
			r := rule.NewRule(kind, "")
			for _, val := range vals[1:] {
				kv := strings.SplitN(val, "=", 2)
				if len(kv) != 2 {
					return nil, fmt.Errorf("failure parsing repository: %s, expected format for attributes is attr1_name=attr1_value", d.Value)
				}
				r.SetAttr(kv[0], kv[1])
			}
			if r.Name() == "" {
				return nil, fmt.Errorf("failure parsing repository: %s, expected a name attribute for the given repository", d.Value)
			}
			repos = append(repos, r)
		}
	}
	return repos, nil
}

func parseRepositoryMacroDirective(directive string) (string, string, error) {
	vals := strings.Split(directive, "%")
	if len(vals) != 2 {
		return "", "", fmt.Errorf("Failure parsing repository_macro: %s, expected format is macroFile%%defName", directive)
	}
	f := vals[0]
	if strings.HasPrefix(f, "..") {
		return "", "", fmt.Errorf("Failure parsing repository_macro: %s, macro file path %s should not start with \"..\"", directive, f)
	}
	return f, vals[1], nil
}
