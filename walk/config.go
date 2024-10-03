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

package walk

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/bmatcuk/doublestar/v4"
	"github.com/go-git/go-git/v5/plumbing/format/gitignore"

	gzflag "github.com/bazelbuild/bazel-gazelle/flag"
)

// TODO(#472): store location information to validate each exclude. They
// may be set in one directory and used in another. Excludes work on
// declared generated files, so we can't just stat.

type walkConfig struct {
	excludes []string
	ignore   bool
	follow   []string

	gitignorePatterns []gitignore.Pattern
	gitignoreMatcher  gitignore.Matcher
}

const walkName = "_walk"

func getWalkConfig(c *config.Config) *walkConfig {
	return c.Exts[walkName].(*walkConfig)
}

func (wc *walkConfig) isExcluded(p string) bool {
	return matchAnyGlob(wc.excludes, p)
}

func (wc *walkConfig) shouldFollow(p string) bool {
	return matchAnyGlob(wc.follow, p)
}

func (wc *walkConfig) isGitIgnored(p []string, isDir bool) bool {
	return wc.gitignoreMatcher != nil && wc.gitignoreMatcher.Match(p, isDir)
}

var _ config.Configurer = (*Configurer)(nil)

type Configurer struct{}

func (*Configurer) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
	wc := &walkConfig{}
	c.Exts[walkName] = wc
	fs.Var(&gzflag.MultiFlag{Values: &wc.excludes}, "exclude", "pattern that should be ignored (may be repeated)")
}

func (*Configurer) CheckFlags(fs *flag.FlagSet, c *config.Config) error { return nil }

func (*Configurer) KnownDirectives() []string {
	return []string{"gitignore", "exclude", "follow", "ignore"}
}

func (cr *Configurer) Configure(c *config.Config, rel string, f *rule.File) {
	wc := getWalkConfig(c)
	wcCopy := &walkConfig{}
	*wcCopy = *wc
	wcCopy.ignore = false

	if f != nil {
		for _, d := range f.Directives {
			switch d.Key {
			case "exclude":
				if err := checkPathMatchPattern(path.Join(rel, d.Value)); err != nil {
					log.Printf("the exclusion pattern is not valid %q: %s", path.Join(rel, d.Value), err)
					continue
				}
				wcCopy.excludes = append(wcCopy.excludes, path.Join(rel, d.Value))
			case "follow":
				if err := checkPathMatchPattern(path.Join(rel, d.Value)); err != nil {
					log.Printf("the follow pattern is not valid %q: %s", path.Join(rel, d.Value), err)
					continue
				}
				wcCopy.follow = append(wcCopy.follow, path.Join(rel, d.Value))
			case "ignore":
				wcCopy.ignore = true
			case "gitignore":
				if d.Value == "on" {
					wcCopy.gitignoreMatcher = gitignore.NewMatcher(wcCopy.gitignorePatterns)
				} else {
					wcCopy.gitignoreMatcher = nil
				}
			}
		}
	}

	c.Exts[walkName] = wcCopy
}

type isIgnoredFunc = func(string) bool

var nothingIgnored isIgnoredFunc = func(string) bool { return false }

func loadBazelIgnore(repoRoot string) (isIgnoredFunc, error) {
	ignorePath := path.Join(repoRoot, ".bazelignore")
	file, err := os.Open(ignorePath)
	if errors.Is(err, fs.ErrNotExist) {
		return nothingIgnored, nil
	}
	if err != nil {
		return nothingIgnored, fmt.Errorf(".bazelignore exists but couldn't be read: %v", err)
	}
	defer file.Close()

	excludes := make(map[string]struct{})

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		ignore := strings.TrimSpace(scanner.Text())
		if ignore == "" || string(ignore[0]) == "#" {
			continue
		}
		// Bazel ignore paths are always relative to repo root.
		// Glob patterns are not supported.
		if strings.ContainsAny(ignore, "*?[") {
			log.Printf("the .bazelignore exclusion pattern must not be a glob %s", ignore)
			continue
		}

		// Clean the path to remove any extra '.', './' etc otherwise
		// the exclude matching won't work correctly.
		ignore = path.Clean(ignore)

		excludes[ignore] = struct{}{}
	}

	isIgnored := func(p string) bool {
		_, ok := excludes[p]
		return ok
	}

	return isIgnored, nil
}

func (wc *walkConfig) loadGitIgnore(rel string, ignoreReader io.Reader) {
	var domain []string
	if rel != "" {
		domain = strings.Split(rel, "/")
	}

	// Load all the patterns
	reader := bufio.NewScanner(ignoreReader)
	for reader.Scan() {
		// Trim, ignore empty lines and comments
		i := strings.TrimSpace(reader.Text())
		if i == "" || strings.HasPrefix(i, "#") {
			continue
		}

		wc.gitignorePatterns = append(wc.gitignorePatterns, gitignore.ParsePattern(i, domain))
	}

	// Override any existing matcher to include new gitignore patterns.
	if wc.gitignoreMatcher != nil {
		wc.gitignoreMatcher = gitignore.NewMatcher(wc.gitignorePatterns)
	}
}

func checkPathMatchPattern(pattern string) error {
	_, err := doublestar.Match(pattern, "x")
	return err
}

func matchAnyGlob(patterns []string, path string) bool {
	for _, x := range patterns {
		matched, err := doublestar.Match(x, path)
		if err != nil {
			// doublestar.Match returns only one possible error, and only if the
			// pattern is not valid. During the configuration of the walker (see
			// Configure below), we discard any invalid pattern and thus an error
			// here should not be possible.
			log.Panicf("error during doublestar.Match. This should not happen, please file an issue https://github.com/bazelbuild/bazel-gazelle/issues/new: %s", err)
		}
		if matched {
			return true
		}
	}
	return false
}
