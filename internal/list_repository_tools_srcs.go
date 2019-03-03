// +build ignore

// Copyright 2014 The Bazel Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// list_go_repository_tools prints Bazel labels for source files that
// gazelle and fetch_repo depend on. go_repository_tools resolves these
// labels so that when a source file changes, the gazelle and fetch_repo
// binaries are rebuilt for go_repository.

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	dir := filepath.FromSlash("src/github.com/bazelbuild/bazel-gazelle")
	if err := os.Chdir(dir); err != nil {
		log.Fatal(err)
	}

	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		base := filepath.Base(path)
		if base == "vendor" || base == "third_party" || base == "testdata" {
			return filepath.SkipDir
		}
		if !info.IsDir() &&
			(strings.HasSuffix(base, ".go") && !strings.HasSuffix(base, "_test.go") ||
				base == "BUILD.bazel" || base == "BUILD") {
			label := filepath.ToSlash(path)
			if i := strings.LastIndexByte(label, '/'); i >= 0 {
				label = "@bazel_gazelle//" + label[:i] + ":" + label[i+1:]
			} else {
				label = "@bazel_gazelle//:" + label
			}
			fmt.Println(label)
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}
