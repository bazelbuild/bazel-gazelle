/* Copyright 2016 The Bazel Authors. All rights reserved.

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

package resolve

import (
	"fmt"
	"path"
	"regexp"

	"github.com/bazelbuild/bazel-gazelle/internal/label"
	"github.com/bazelbuild/bazel-gazelle/internal/pathtools"
	"github.com/bazelbuild/bazel-gazelle/internal/repos"
	"golang.org/x/tools/go/vcs"
)

// externalResolver resolves import paths to external repositories. It uses
// vcs to determine the prefix of the import path that corresponds to the root
// of the repository (this will perform a network fetch for unqualified paths).
// The prefix is converted to a Bazel external name repo according to the
// guidelines in http://bazel.io/docs/be/functions.html#workspace. The remaining
// portion of the import path is treated as the package name.
type externalResolver struct {
	l *label.Labeler

	// repoRootForImportPath is vcs.RepoRootForImportPath by default. It may
	// be overridden by tests.
	repoRootForImportPath func(string, bool) (*vcs.RepoRoot, error)

	// cache stores lookup results, both positive and negative to reduce
	// network fetches when there are multiple imports on the same external repo.
	cache map[string]repoRootCacheEntry

	// prefixToName maps import path prefixes to repository names.
	prefixToName map[string]string
}

type repoRootCacheEntry struct {
	// prefix is part of an import path that corresponds to a repository root,
	// possibly with some components missing.
	prefix string

	// missing is the number of components missing from prefix to make a full
	// repository root prefix. For most repositories, this is 0, meaning the
	// prefix is the full path to the repository root. For some well-known sites,
	// this is non-zero. For example, we can store the prefix "github.com" with
	// missing as 2, since GitHub always has two path components before the
	// actual repository.
	missing int

	// err is an error we encountered when resolving this prefix. This is used
	// for caching negative results.
	err error
}

var _ nonlocalResolver = (*externalResolver)(nil)

func newExternalResolver(l *label.Labeler, repos []repos.Repo) *externalResolver {
	resolver := &externalResolver{
		l:                     l,
		cache:                 make(map[string]repoRootCacheEntry),
		prefixToName:          make(map[string]string),
		repoRootForImportPath: vcs.RepoRootForImportPath,
	}

	for _, e := range []repoRootCacheEntry{
		{prefix: "golang.org/x", missing: 1},
		{prefix: "google.golang.org", missing: 1},
		{prefix: "cloud.google.com", missing: 1},
		{prefix: "github.com", missing: 2},
	} {
		resolver.cache[e.prefix] = e
	}

	for _, e := range repos {
		resolver.cache[e.GoPrefix] = repoRootCacheEntry{prefix: e.GoPrefix, missing: 0}
		resolver.prefixToName[e.GoPrefix] = e.Name
	}

	return resolver
}

// Resolve resolves "importpath" into a label, assuming that it is a label in an
// external repository. It also assumes that the external repository follows the
// recommended reverse-DNS form of workspace name as described in
// http://bazel.io/docs/be/functions.html#workspace.
func (r *externalResolver) resolve(importpath string) (label.Label, error) {
	prefix, err := r.lookupPrefix(importpath)
	if err != nil {
		return label.NoLabel, err
	}

	repo, ok := r.prefixToName[prefix]
	if !ok {
		repo = label.ImportPathToBazelRepoName(prefix)
		r.prefixToName[prefix] = repo
	}

	var pkg string
	if importpath != prefix {
		pkg = pathtools.TrimPrefix(importpath, prefix)
	}

	l := r.l.LibraryLabel(pkg)
	l.Repo = repo
	return l, nil
}

var gopkginPattern = regexp.MustCompile("^(gopkg.in/(?:[^/]+/)?[^/]+\\.v\\d+)(?:/|$)")

// lookupPrefix determines the prefix of "importpath" that corresponds to
// the root of the repository. Results are cached.
func (r *externalResolver) lookupPrefix(importpath string) (string, error) {
	// subpaths contains slices of importpath with components removed. For
	// example:
	//   golang.org/x/tools/go/vcs
	//   golang.org/x/tools/go
	//   golang.org/x/tools
	subpaths := []string{importpath}

	// Check the cache for prefixes of the import path.
	prefix := importpath
	for {
		if e, ok := r.cache[prefix]; ok {
			if e.missing >= len(subpaths) {
				return "", fmt.Errorf("import path %q is shorter than the known prefix %q", prefix, e.prefix)
			}
			// Cache hit. Restore n components of the import path to get the
			// repository root.
			return subpaths[len(subpaths)-e.missing-1], e.err
		}

		// Prefix not found. Remove the last component and try again.
		prefix = path.Dir(prefix)
		if prefix == "." || prefix == "/" {
			// Cache miss.
			break
		}
		subpaths = append(subpaths, prefix)
	}

	// gopkg.in is special, and might have either one or two levels of
	// missing paths. See http://labix.org/gopkg.in for URL patterns.
	if match := gopkginPattern.FindStringSubmatch(importpath); len(match) > 0 {
		return match[1], nil
	}

	// Look up the import path using vcs.
	root, err := r.repoRootForImportPath(importpath, false)
	if err != nil {
		r.cache[importpath] = repoRootCacheEntry{prefix: importpath, err: err}
		return "", err
	}
	prefix = root.Root
	r.cache[prefix] = repoRootCacheEntry{prefix: prefix}
	return prefix, nil
}
