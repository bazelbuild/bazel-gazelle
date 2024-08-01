/* Copyright 2023 The Bazel Authors. All rights reserved.

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

package generationtest

import (
	"flag"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bazelbuild/bazel-gazelle/testtools"
	"github.com/bazelbuild/rules_go/go/runfiles"
	"github.com/bazelbuild/rules_go/go/tools/bazel"
)

var (
	gazelleBinaryPath = flag.String("gazelle_binary_path", "", "Runfiles path of the gazelle binary to test.")
	buildInSuffix     = flag.String("build_in_suffix", ".in", "The suffix on the test input BUILD.bazel files. Defaults to .in. "+
		" By default, will use files named BUILD.in as the BUILD files before running gazelle.")
	buildOutSuffix = flag.String("build_out_suffix", ".out", "The suffix on the expected BUILD.bazel files after running gazelle. Defaults to .out. "+
		" By default, will use files named BUILD.out as the expected results of the gazelle run.")
	timeout = flag.Duration("timeout", 2*time.Second, "Time to allow the gazelle process to run before killing.")
)

// TestFullGeneration runs the gazelle binary on a few example
// workspaces and confirms that the generated BUILD files match expectation.
func TestFullGeneration(t *testing.T) {
	tests := []*testtools.TestGazelleGenerationArgs{}
	r, err := runfiles.New()
	if err != nil {
		t.Fatalf("Failed to create runfiles: %v", err)
	}
	gazelleBinary, err := r.Rlocation(*gazelleBinaryPath)
	if err != nil {
		t.Fatalf("Failed to find gazelle binary %s in runfiles. Error: %v", *gazelleBinaryPath, err)
	}

	testDir, err := bazel.NewTmpDir("gazelle_generation_test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}

	// Collect tests in the same repo as the gazelle binary.
	repo := strings.Split(*gazelleBinaryPath, "/")[0]
	_, err = r.Open(*gazelleBinaryPath)
	if err != nil {
		t.Fatalf("Failed to open gazelle binary %s in runfiles. Error: %v", *gazelleBinaryPath, err)
	}
	fs.WalkDir(r, ".", func(p string, d fs.DirEntry, err error) error {
		println("p: ", p)
		return nil
	})
	err = fs.WalkDir(r, repo, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		println("p: ", p)
		// Each repo boundary file marks a test case.
		if d.Name() == "WORKSPACE" || d.Name() == "MODULE.bazel" || d.Name() == "REPO.bazel" {
			repoRelativeDir := path.Dir(strings.TrimPrefix(p, repo+"/"))
			// name is the name of the test directory. For example, my_test_case.
			// The name of the directory doubles as the name of the test.
			name := filepath.Base(repoRelativeDir)
			absolutePath := filepath.Join(testDir, filepath.FromSlash(repoRelativeDir))

			tests = append(tests, &testtools.TestGazelleGenerationArgs{
				Name:                 name,
				TestDataPathRelative: repoRelativeDir,
				TestDataPathAbsolute: absolutePath,
				GazelleBinaryPath:    gazelleBinary,
				BuildInSuffix:        *buildInSuffix,
				BuildOutSuffix:       *buildOutSuffix,
				Timeout:              *timeout,
			})
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to collect tests in runfiles: %v", err)
	}
	if len(tests) == 0 {
		t.Fatal("no tests found")
	}

	// Copy all files under test repos to the temporary directory.
	for _, test := range tests {
		err = fs.WalkDir(r, path.Join(repo, test.TestDataPathRelative), func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			f, err := r.Open(p)
			if err != nil {
				return err
			}
			defer f.Close()

			targetPath := filepath.Join(testDir, filepath.FromSlash(strings.TrimPrefix(p, repo+"/")))
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return err
			}
			out, err := os.Create(targetPath)
			if err != nil {
				return err
			}
			defer out.Close()

			if _, err := io.Copy(out, f); err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			t.Fatalf("Failed to copy tests from runfiles: %v", err)
		}
	}

	for _, args := range tests {
		testtools.TestGazelleGenerationOnPath(t, args)
	}
}
