/* Copyright 2021 The Bazel Authors. All rights reserved.

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

package golang

import (
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"golang.org/x/mod/module"
)

// embedResolver maps go:embed patterns in source files to lists of files that
// should appear in embedsrcs attributes.
type embedResolver struct {
	// files is a list of embeddable files and directories, stored in
	// depth-first pre-order (nearly the same order as filepath.Walk, except
	// files in the top-level directory appear before subdirectories.
	files []embeddableFile
}

type embeddableFile struct {
	path  string
	isDir bool
}

// newEmbedResolver builds a list of files that may be embedded. This is
// approximately all files in a Bazel package including explicitly declared
// generated files and files in subdirectories without build files.
// Files in other Bazel packages are not listed, since it might not be possible
// to reference those files if they aren't listed in an export_files
// declaration.
//
// This function walks subdirectory trees and may be expensive. Don't call it
// unless a go:embed directive is actually present.
//
// dir is the absolute path to the directory containing the embed directive.
//
// rel is the relative path from the workspace root to the same directory
// (or "" if the directory is the workspace root itself).
//
// validBuildFileNames is the configured list of recognized build file names.
// These are used to identify Bazel packages in subdirectories that Gazelle
// did not visit.
//
// pkgRels is a set of relative paths from the workspace root to directories
// that contain (or will contain) build files. It doesn't need to contain
// entries for the entire workspace, but it should contain entries for
// subdirectories processed earlier (this avoids redundant O(n^2) I/O).
//
// subdirs, regFiles, and genFiles are lists of subdirectories, regular files,
// and declared generated files in dir, respectively.
func newEmbedResolver(dir, rel string, validBuildFileNames []string, pkgRels map[string]bool, subdirs, regFiles, genFiles []string) *embedResolver {
	var files []embeddableFile
	for _, fs := range [...][]string{regFiles, genFiles} {
		for _, f := range fs {
			if !isBadEmbedName(f) {
				files = append(files, embeddableFile{path: f})
			}
		}
	}

	for _, subdir := range subdirs {
		err := filepath.Walk(filepath.Join(dir, subdir), func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			fileRel, _ := filepath.Rel(dir, p)
			base := filepath.Base(p)
			if !info.IsDir() {
				if !isBadEmbedName(base) {
					files = append(files, embeddableFile{path: fileRel})
				}
				return nil
			}
			if isBadEmbedName(base) {
				return filepath.SkipDir
			}
			if pkgRels[path.Join(rel, fileRel)] {
				// Directory contains a Go package and will contain a build file,
				// if it doesn't already.
				return filepath.SkipDir
			}
			for _, name := range validBuildFileNames {
				if _, err := os.Stat(filepath.Join(p, name)); err == nil {
					// Directory already contains a build file.
					return filepath.SkipDir
				}
			}
			files = append(files, embeddableFile{path: fileRel, isDir: true})
			return nil
		})
		if err != nil {
			log.Printf("listing embeddable files in %s: %v", dir, err)
		}
	}

	return &embedResolver{files: files}
}

// resolve expands a single go:embed pattern into a list of files that should
// be included in embedsrcs. Directory paths are not included in the returned
// list. This means there's no way to embed an empty directory.
func (er *embedResolver) resolve(embed fileEmbed) (list []string, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("%v: pattern %s: %w", embed.pos, embed.path, err)
		}
	}()

	// Check whether the pattern is valid at all.
	if _, err := path.Match(embed.path, ""); err != nil || !validEmbedPattern(embed.path) {
		return nil, fmt.Errorf("invalid pattern syntax")
	}

	// Match the pattern against each path in the list. If the pattern matches a
	// directory, we need to include each file in that directory, even if the file
	// doesn't match the pattern separate, unless it is a hidden file (starting
	// with . or _).
	//
	// For example, the pattern "*" matches "a", ".b", and "_c". If "a" is a
	// directory, we would include "a/d", even though it doesn't match "*". We
	// would not include "a/.e".
	//
	// There may be many patterns, so we avoid I/O here. Instead, the list is
	// in depth-first pre-order, so iterating over it is analogous to
	// filepath.Walk. We still use a recursive function, advance, to keep track
	// of whether we're embedding all files in the current directory. Each call
	// to advance increments i.
	i := 0
	var advance func(bool)
	advance = func(add bool) {
		f := er.files[i]
		i++
		if !f.isDir {
			if add {
				list = append(list, f.path)
			}
			return
		}
		prefix := f.path + "/"
		for i < len(er.files) && strings.HasPrefix(er.files[i].path, prefix) {
			base := er.files[i].path[len(prefix):]
			hidden := base[0] == '.' || base[0] == '_'
			advance(add && !hidden)
		}
	}

	for i < len(er.files) {
		matched, _ := path.Match(embed.path, er.files[i].path)
		advance(matched)
	}

	if len(list) == 0 {
		return nil, fmt.Errorf("matched no files")
	}

	return list, nil
}

// Copied from cmd/go/internal/load.validEmbedPattern.
func validEmbedPattern(pattern string) bool {
	return pattern != "." && fsValidPath(pattern)
}

// fsValidPath reports whether the given path name
// is valid for use in a call to Open.
//
// Path names passed to open are UTF-8-encoded,
// unrooted, slash-separated sequences of path elements, like “x/y/z”.
// Path names must not contain an element that is “.” or “..” or the empty string,
// except for the special case that the root directory is named “.”.
// Paths must not start or end with a slash: “/x” and “x/” are invalid.
//
// Note that paths are slash-separated on all systems, even Windows.
// Paths containing other characters such as backslash and colon
// are accepted as valid, but those characters must never be
// interpreted by an FS implementation as path element separators.
//
// Copied from io/fs.ValidPath to avoid making go1.16 a build-time dependency
// for Gazelle.
func fsValidPath(name string) bool {
	if !utf8.ValidString(name) {
		return false
	}

	if name == "." {
		// special case
		return true
	}

	// Iterate over elements in name, checking each.
	for {
		i := 0
		for i < len(name) && name[i] != '/' {
			i++
		}
		elem := name[:i]
		if elem == "" || elem == "." || elem == ".." {
			return false
		}
		if i == len(name) {
			return true // reached clean ending
		}
		name = name[i+1:]
	}
}

// isBadEmbedName reports whether name is the base name of a file that
// can't or won't be included in modules and therefore shouldn't be treated
// as existing for embedding.
//
// Copied from cmd/go/internal/load.isBadEmbedName.
func isBadEmbedName(name string) bool {
	if err := module.CheckFilePath(name); err != nil {
		return true
	}
	switch name {
	// Empty string should be impossible but make it bad.
	case "":
		return true
	// Version control directories won't be present in module.
	case ".bzr", ".hg", ".git", ".svn":
		return true
	}
	return false
}
