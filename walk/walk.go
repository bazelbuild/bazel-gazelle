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

// Package walk provides customizable functionality for visiting each
// subdirectory in a directory tree.
package walk

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// Mode determines which directories Walk visits and which directories
// should be updated.
type Mode int

const (
	// In VisitAllUpdateSubdirsMode, Walk visits every directory in the
	// repository. The directories given to Walk and their subdirectories are
	// updated.
	VisitAllUpdateSubdirsMode Mode = iota

	// In VisitAllUpdateDirsMode, Walk visits every directory in the repository.
	// Only the directories given to Walk are updated (not their subdirectories).
	VisitAllUpdateDirsMode

	// In UpdateDirsMode, Walk only visits and updates directories given to Walk.
	// Build files in parent directories are read in order to produce a complete
	// configuration, but the callback is not called for parent directories.
	UpdateDirsMode

	// In UpdateSubdirsMode, Walk visits and updates the directories given to Walk
	// and their subdirectories. Build files in parent directories are read in
	// order to produce a complete configuration, but the callback is not called
	// for parent directories.
	UpdateSubdirsMode
)

// WalkFunc is a callback called by Walk in each visited directory.
//
// dir is the absolute file system path to the directory being visited.
//
// rel is the relative slash-separated path to the directory from the
// repository root. Will be "" for the repository root directory itself.
//
// c is the configuration for the current directory. This may have been
// modified by directives in the directory's build file.
//
// update is true when the build file may be updated.
//
// f is the existing build file in the directory. Will be nil if there
// was no file.
//
// subdirs is a list of base names of subdirectories within dir, not
// including excluded files.
//
// regularFiles is a list of base names of regular files within dir, not
// including excluded files or symlinks.
//
// genFiles is a list of names of generated files, found by reading
// "out" and "outs" attributes of rules in f.
type WalkFunc func(dir, rel string, c *config.Config, update bool, f *rule.File, subdirs, regularFiles, genFiles []string)

// Walk traverses the directory tree rooted at c.RepoRoot. Walk visits
// subdirectories in depth-first post-order.
//
// When Walk visits a directory, it lists the files and subdirectories within
// that directory. If a build file is present, Walk reads the build file and
// applies any directives to the configuration (a copy of the parent directory's
// configuration is made, and the copy is modified). After visiting
// subdirectories, the callback wf may be called, depending on the mode.
//
// c is the root configuration to start with. This includes changes made by
// command line flags, but not by the root build file. This configuration
// should not be modified.
//
// cexts is a list of configuration extensions. When visiting a directory,
// before visiting subdirectories, Walk makes a copy of the parent configuration
// and Configure for each extension on the copy. If Walk sees a directive
// that is not listed in KnownDirectives of any extension, an error will
// be logged.
//
// dirs is a list of absolute, canonical file system paths of directories
// to visit.
//
// mode determines whether subdirectories of dirs should be visited recursively,
// when the wf callback should be called, and when the "update" argument
// to the wf callback should be set.
//
// wf is a function that may be called in each directory.
func Walk(c *config.Config, cexts []config.Configurer, dirs []string, mode Mode, wf WalkFunc) {
	knownDirectives := make(map[string]bool)
	for _, cext := range cexts {
		for _, d := range cext.KnownDirectives() {
			knownDirectives[d] = true
		}
	}

	updateRels := buildUpdateRelMap(c.RepoRoot, dirs)

	var visit func(*config.Config, string, string, bool)
	visit = func(c *config.Config, dir, rel string, updateParent bool) {
		haveError := false

		// TODO: OPT: ReadDir stats all the files, which is slow. We just care about
		// names and modes, so we should use something like
		// golang.org/x/tools/internal/fastwalk to speed this up.
		files, err := ioutil.ReadDir(dir)
		if err != nil {
			log.Print(err)
			return
		}

		f, err := loadBuildFile(c, rel, dir, files)
		if err != nil {
			log.Print(err)
			if c.Strict {
				// TODO(https://github.com/bazelbuild/bazel-gazelle/issues/1029):
				// Refactor to accumulate and propagate errors to main.
				log.Fatal("Exit as strict mode is on")
			}
			haveError = true
		}

		c = configure(cexts, knownDirectives, c, rel, f)
		wc := getWalkConfig(c)

		if wc.isExcluded(rel, ".") {
			return
		}

		var subdirs, regularFiles []string
		for _, fi := range files {
			base := fi.Name()
			switch {
			case base == "" || wc.isExcluded(rel, base) || fi.Mode()&os.ModeSymlink != 0:
				continue

			case fi.IsDir():
				subdirs = append(subdirs, base)

			default:
				regularFiles = append(regularFiles, base)
			}
		}

		shouldUpdate := shouldUpdate(rel, mode, updateParent, updateRels)
		for _, sub := range subdirs {
			if subRel := path.Join(rel, sub); shouldVisit(subRel, mode, shouldUpdate, updateRels) {
				visit(c, filepath.Join(dir, sub), subRel, shouldUpdate)
			}
		}

		update := !haveError && !wc.ignore && shouldUpdate
		if shouldCall(rel, mode, updateParent, updateRels) {
			genFiles := findGenFiles(wc, f)
			wf(dir, rel, c, update, f, subdirs, regularFiles, genFiles)
		}
	}
	visit(c, c.RepoRoot, "", false)
}

// buildUpdateRelMap builds a table of prefixes, used to determine which
// directories to update and visit.
//
// root and dirs must be absolute, canonical file paths. Each entry in dirs
// must be a subdirectory of root. The caller is responsible for checking this.
//
// buildUpdateRelMap returns a map from slash-separated paths relative to the
// root directory ("" for the root itself) to a boolean indicating whether
// the directory should be updated.
func buildUpdateRelMap(root string, dirs []string) map[string]bool {
	relMap := make(map[string]bool)
	for _, dir := range dirs {
		rel, _ := filepath.Rel(root, dir)
		rel = filepath.ToSlash(rel)
		if rel == "." {
			rel = ""
		}

		i := 0
		for {
			next := strings.IndexByte(rel[i:], '/') + i
			if next-i < 0 {
				relMap[rel] = true
				break
			}
			prefix := rel[:next]
			if _, ok := relMap[prefix]; !ok {
				relMap[prefix] = false
			}
			i = next + 1
		}
	}
	return relMap
}

// shouldCall returns true if Walk should call the callback in the
// directory rel.
func shouldCall(rel string, mode Mode, updateParent bool, updateRels map[string]bool) bool {
	switch mode {
	case VisitAllUpdateSubdirsMode, VisitAllUpdateDirsMode:
		return true
	case UpdateSubdirsMode:
		return updateParent || updateRels[rel]
	default: // UpdateDirsMode
		return updateRels[rel]
	}
}

// shouldUpdate returns true if Walk should pass true to the callback's update
// parameter in the directory rel. This indicates the build file should be
// updated.
func shouldUpdate(rel string, mode Mode, updateParent bool, updateRels map[string]bool) bool {
	if (mode == VisitAllUpdateSubdirsMode || mode == UpdateSubdirsMode) && updateParent {
		return true
	}
	return updateRels[rel]
}

// shouldVisit returns true if Walk should visit the subdirectory rel.
func shouldVisit(rel string, mode Mode, updateParent bool, updateRels map[string]bool) bool {
	switch mode {
	case VisitAllUpdateSubdirsMode, VisitAllUpdateDirsMode:
		return true
	case UpdateSubdirsMode:
		_, ok := updateRels[rel]
		return ok || updateParent
	default: // UpdateDirsMode
		_, ok := updateRels[rel]
		return ok
	}
}

func loadBuildFile(c *config.Config, pkg, dir string, files []os.FileInfo) (*rule.File, error) {
	var err error
	readDir := dir
	readFiles := files
	if c.ReadBuildFilesDir != "" {
		readDir = filepath.Join(c.ReadBuildFilesDir, filepath.FromSlash(pkg))
		readFiles, err = ioutil.ReadDir(readDir)
		if err != nil {
			return nil, err
		}
	}
	path := rule.MatchBuildFileName(readDir, c.ValidBuildFileNames, readFiles)
	if path == "" {
		return nil, nil
	}
	return rule.LoadFile(path, pkg)
}

func configure(cexts []config.Configurer, knownDirectives map[string]bool, c *config.Config, rel string, f *rule.File) *config.Config {
	if rel != "" {
		c = c.Clone()
	}
	if f != nil {
		for _, d := range f.Directives {
			if !knownDirectives[d.Key] {
				log.Printf("%s: unknown directive: gazelle:%s", f.Path, d.Key)
				if c.Strict {
					// TODO(https://github.com/bazelbuild/bazel-gazelle/issues/1029):
					// Refactor to accumulate and propagate errors to main.
					log.Fatal("Exit as strict mode is on")
				}
			}
		}
	}
	for _, cext := range cexts {
		cext.Configure(c, rel, f)
	}
	return c
}

func findGenFiles(wc *walkConfig, f *rule.File) []string {
	if f == nil {
		return nil
	}
	var strs []string
	for _, r := range f.Rules {
		for _, key := range []string{"out", "outs"} {
			if s := r.AttrString(key); s != "" {
				strs = append(strs, s)
			} else if ss := r.AttrStrings(key); len(ss) > 0 {
				strs = append(strs, ss...)
			}
		}
	}

	var genFiles []string
	for _, s := range strs {
		if !wc.isExcluded(f.Pkg, s) {
			genFiles = append(genFiles, s)
		}
	}
	return genFiles
}
