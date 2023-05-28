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
	"io/fs"
	"log"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/bmatcuk/doublestar/v4"

	gzflag "github.com/bazelbuild/bazel-gazelle/flag"
)

// TODO(#472): store location information to validate each exclude. They
// may be set in one directory and used in another. Excludes work on
// declared generated files, so we can't just stat.

type walkConfig struct {
	excludes []string
	ignore   bool
	follow   []string
	loadOnce *sync.Once
}

const walkName = "_walk"

func getWalkConfig(c *config.Config) *walkConfig {
	return c.Exts[walkName].(*walkConfig)
}

func (wc *walkConfig) isExcluded(rel, base string) bool {
	if base == ".git" {
		return true
	}
	f := path.Join(rel, base)
	for _, x := range wc.excludes {
		matched, err := doublestar.Match(x, f)
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

type Configurer struct{}

func (*Configurer) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
	wc := &walkConfig{loadOnce: &sync.Once{}}
	c.Exts[walkName] = wc
	fs.Var(&gzflag.MultiFlag{Values: &wc.excludes}, "exclude", "pattern that should be ignored (may be repeated)")
}

func (*Configurer) CheckFlags(fs *flag.FlagSet, c *config.Config) error { return nil }

func (*Configurer) KnownDirectives() []string {
	return []string{"exclude", "follow", "ignore"}
}

func (cr *Configurer) Configure(c *config.Config, rel string, f *rule.File) {
	wc := getWalkConfig(c)
	wcCopy := &walkConfig{}
	*wcCopy = *wc
	wcCopy.ignore = false

	wc.loadOnce.Do(func() {
		if err := cr.loadBazelIgnore(c.RepoRoot, wcCopy); err != nil {
			log.Printf("error loading .bazelignore: %v", err)
		}
	})

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
				wcCopy.follow = append(wcCopy.follow, path.Join(rel, d.Value))
			case "ignore":
				wcCopy.ignore = true
			}
		}
	}

	c.Exts[walkName] = wcCopy
}

func (c *Configurer) loadBazelIgnore(repoRoot string, wc *walkConfig) error {
	ignorePath := path.Join(repoRoot, ".bazelignore")
	file, err := os.Open(ignorePath)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf(".bazelignore exists but couldn't be read: %v", err)
	}
	defer file.Close()

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
		// Ensure we remove trailing slashes or the exclude matching won't work correctly
		wc.excludes = append(wc.excludes, strings.TrimSuffix(ignore, "/"))
	}
	return nil
}

func checkPathMatchPattern(pattern string) error {
	_, err := doublestar.Match(pattern, "x")
	return err
}
