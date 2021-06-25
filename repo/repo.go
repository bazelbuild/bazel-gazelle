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
	visited := make(map[string]bool)
	for _, repo := range workspace.Rules {
		if name := repo.Name(); name != "" {
			repos = append(repos, repo)
			repoFileMap[name] = workspace
			repoIndexMap[name] = len(repos) - 1
		}
	}
	if err := loadExtraRepos(workspace, &repos, repoFileMap, repoIndexMap); err != nil {
		return nil, nil, err
	}

	for _, d := range workspace.Directives {
		switch d.Key {
		case "repository_macro":
			f, defName, leveled, err := ParseRepositoryMacroDirective(d.Value)
			if err != nil {
				return nil, nil, err
			}
			repoRoot := filepath.Dir(workspace.Path)
			f = filepath.Join(repoRoot, filepath.Clean(f))
			visited[f+"%"+defName] = true

			la := &loadArgs{
				repoRoot: repoRoot,
				repos: repos,
				repoFileMap: repoFileMap,
				repoIndexMap: repoIndexMap,
				visited: visited,
			}

			if err := loadRepositoriesFromMacro(la, leveled, f, defName); err != nil {
				return nil, nil, err
			}
			repos = la.repos
		}
	}
	return repos, repoFileMap, nil
}

type loadArgs struct {
	repoRoot string
	repos []*rule.Rule
	repoFileMap map[string]*rule.File
	repoIndexMap map[string]int
	visited map[string]bool
}

func loadRepositoriesFromMacro(la *loadArgs, leveled bool, f, defName string) error {
	macroFile, err := rule.LoadMacroFile(f, "", defName)
	if err != nil {
		return err
	}
	for _, rule := range macroFile.Rules {
		name := rule.Name()
		if name != "" {
			la.repos = append(la.repos, rule)
			la.repoFileMap[name] = macroFile
			la.repoIndexMap[name] = len(la.repos) - 1
			continue
		}
		if !leveled {
			continue
		}
		// If another repository macro is loaded that macro defName must be called.
		// When a defName is called, the defName of the function is the rule's "kind".
		// This then must be matched with the Load that it is imported with, so that
		// file can be loaded
		kind := rule.Kind()
		for _, l := range macroFile.Loads {
			if l.Has(kind) {
				f, defName = loadToMacroDef(l, la.repoRoot, kind)
				break
			}
		}
		// TODO: Also handle the case where one macro calls another macro in the same bzl file
		if f == "" {
			continue
		}
		if !la.visited[f+"%"+defName] {
			la.visited[f+"%"+defName] = true
			if err := loadRepositoriesFromMacro(la, false /* leveled */, f, defName); err != nil {
				return err
			}
		}
	}
	return loadExtraRepos(macroFile, &la.repos, la.repoFileMap, la.repoIndexMap)
}

// loadToMacroDef takes a load
// e.g. for if called on
// load("package_name:package_dir/file.bzl", alias_name="original_def_name")
// with defAlias = "alias_name", it will return:
//     -> "/Path/to/package_name/package_dir/file.bzl"
//     -> "original_def_name"
func loadToMacroDef(l *rule.Load, repoRoot, defAlias string) (string, string) {
	rel := strings.Replace(filepath.Clean(l.Name()), ":", string(filepath.Separator), 1)
	f := filepath.Join(repoRoot, rel)
	// A loaded macro may refer to the macro by a different name (alias) in the load,
	// thus, the original name must be resolved to load the macro file properly.
	defName := l.Unalias(defAlias)
	return f, defName
}

func loadExtraRepos(f *rule.File, repos *[]*rule.Rule, repoFileMap map[string]*rule.File, repoIndexMap map[string]int) error {
	extraRepos, err := parseRepositoryDirectives(f.Directives)
	if err != nil {
		return err
	}
	for _, repo := range extraRepos {
		if i, ok := repoIndexMap[repo.Name()]; ok {
			(*repos)[i] = repo
		} else {
			*repos = append(*repos, repo)
		}
		repoFileMap[repo.Name()] = f
	}
	return nil
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

// ParseRepositoryMacroDirective checks the directive is in proper format, and splits
// path and defName. Repository_macros prepended with a "+" (e.g. "# gazelle:repository_macro +file%def")
// indicates a "leveled" macro, which loads other macro files.
func ParseRepositoryMacroDirective(directive string) (string, string, bool, error) {
	vals := strings.Split(directive, "%")
	if len(vals) != 2 {
		return "", "", false, fmt.Errorf("Failure parsing repository_macro: %s, expected format is macroFile%%defName", directive)
	}
	f := vals[0]
	if strings.HasPrefix(f, "..") {
		return "", "", false, fmt.Errorf("Failure parsing repository_macro: %s, macro file path %s should not start with \"..\"", directive, f)
	}
	return strings.TrimPrefix(f, "+"), vals[1], strings.HasPrefix(f, "+"), nil
}
