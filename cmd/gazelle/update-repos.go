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

package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/bazelbuild/bazel-gazelle/internal/merger"
	"github.com/bazelbuild/bazel-gazelle/internal/repos"
	"github.com/bazelbuild/bazel-gazelle/internal/wspace"
	bf "github.com/bazelbuild/buildtools/build"
)

type updateReposConfiguration struct {
	repoRoot     string
	lockFilename string
}

func updateRepos(args []string) error {
	c, err := newUpdateReposConfiguration(args)
	if err != nil {
		return err
	}

	workspacePath := filepath.Join(c.repoRoot, "WORKSPACE")
	content, err := ioutil.ReadFile(workspacePath)
	if err != nil {
		return fmt.Errorf("error reading %q: %v", workspacePath, err)
	}
	oldFile, err := bf.Parse(workspacePath, content)
	if err != nil {
		return fmt.Errorf("error parsing %q: %v", workspacePath, err)
	}

	genRules, err := repos.ImportRepoRules(c.lockFilename)
	if err != nil {
		return err
	}

	mergedFile, _ := merger.MergeFile(genRules, nil, oldFile, merger.RepoAttrs)
	mergedFile = merger.FixLoads(mergedFile)
	if err := ioutil.WriteFile(mergedFile.Path, bf.Format(mergedFile), 0666); err != nil {
		return fmt.Errorf("error writing %q: %v", mergedFile.Path, err)
	}
	return nil
}

func newUpdateReposConfiguration(args []string) (*updateReposConfiguration, error) {
	c := new(updateReposConfiguration)
	fs := flag.NewFlagSet("gazelle", flag.ContinueOnError)
	// Flag will call this on any parse error. Don't print usage unless
	// -h or -help were passed explicitly.
	fs.Usage = func() {}

	fromFileFlag := fs.String("from_file", "", "Gazelle will translate repositories listed in this file into repository rules in WORKSPACE. Currently only dep's Gopkg.lock is supported.")
	repoRootFlag := fs.String("repo_root", "", "path to the root directory of the repository. If unspecified, this is assumed to be the directory containing WORKSPACE.")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			updateReposUsage(fs)
			os.Exit(0)
		}
		// flag already prints the error; don't print it again.
		return nil, errors.New("Try -help for more information")
	}
	if len(fs.Args()) != 0 {
		return nil, fmt.Errorf("Got %d positional arguments; wanted 0.\nTry -help for more information.")
	}

	c.lockFilename = *fromFileFlag
	if c.lockFilename == "" {
		return nil, errors.New("-from_file not provided.\nTry -help for more information.")
	}

	c.repoRoot = *repoRootFlag
	if c.repoRoot == "" {
		if repoRoot, err := wspace.Find("."); err != nil {
			return nil, err
		} else {
			c.repoRoot = repoRoot
		}
	}

	return c, nil
}

func updateReposUsage(fs *flag.FlagSet) {
	fmt.Fprintln(os.Stderr, `usage: gazelle update-repos -from_file file

The update-repos command updates repository rules in the WORKSPACE file.

Currently, update-repos can only import repository rules from the lock file
of a vendoring tool, and only dep's Gopkg.lock is supported. More functionality
is planned though, and you have to start somewhere.

FLAGS:
`)
}
