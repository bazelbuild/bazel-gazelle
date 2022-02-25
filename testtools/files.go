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

package testtools

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/google/go-cmp/cmp"
	yaml "gopkg.in/yaml.v2"
)

// FileSpec specifies the content of a test file.
type FileSpec struct {
	// Path is a slash-separated path relative to the test directory. If Path
	// ends with a slash, it indicates a directory should be created
	// instead of a file.
	Path string

	// Symlink is a slash-separated path relative to the test directory. If set,
	// it indicates a symbolic link should be created with this path instead of a
	// file.
	Symlink string

	// Content is the content of the test file.
	Content string

	// NotExist asserts that no file at this path exists.
	// It is only valid in CheckFiles.
	NotExist bool
}

// CreateFiles creates a directory of test files. This is a more compact
// alternative to testdata directories. CreateFiles returns a canonical path
// to the directory and a function to call to clean up the directory
// after the test.
func CreateFiles(t *testing.T, files []FileSpec) (dir string, cleanup func()) {
	t.Helper()
	dir, err := ioutil.TempDir(os.Getenv("TEST_TEMPDIR"), "gazelle_test")
	if err != nil {
		t.Fatal(err)
	}
	dir, err = filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range files {
		if f.NotExist {
			t.Fatalf("CreateFiles: NotExist may not be set: %s", f.Path)
		}
		path := filepath.Join(dir, filepath.FromSlash(f.Path))
		if strings.HasSuffix(f.Path, "/") {
			if err := os.MkdirAll(path, 0700); err != nil {
				os.RemoveAll(dir)
				t.Fatal(err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			os.RemoveAll(dir)
			t.Fatal(err)
		}
		if f.Symlink != "" {
			if err := os.Symlink(f.Symlink, path); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := ioutil.WriteFile(path, []byte(f.Content), 0600); err != nil {
			os.RemoveAll(dir)
			t.Fatal(err)
		}
	}

	return dir, func() { os.RemoveAll(dir) }
}

// CheckFiles checks that files in "dir" exist and have the content specified
// in "files". Files not listed in "files" are not tested, so extra files
// are allowed.
func CheckFiles(t *testing.T, dir string, files []FileSpec) {
	t.Helper()
	for _, f := range files {
		path := filepath.Join(dir, f.Path)

		st, err := os.Stat(path)
		if f.NotExist {
			if err == nil {
				t.Errorf("asserted to not exist, but does: %s", f.Path)
			} else if !os.IsNotExist(err) {
				t.Errorf("could not stat %s: %v", f.Path, err)
			}
			continue
		}

		if strings.HasSuffix(f.Path, "/") {
			if err != nil {
				t.Errorf("could not stat %s: %v", f.Path, err)
			} else if !st.IsDir() {
				t.Errorf("not a directory: %s", f.Path)
			}
		} else {
			want := strings.TrimSpace(f.Content)
			gotBytes, err := ioutil.ReadFile(filepath.Join(dir, f.Path))
			if err != nil {
				t.Errorf("could not read %s: %v", f.Path, err)
				continue
			}
			got := strings.TrimSpace(string(gotBytes))
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("%s diff (-want,+got):\n%s", f.Path, diff)
			}
		}
	}
}

type TestGazelleGenerationArgs struct {
	// name is the name of the test.
	Name string
	// testDataPath is the workspace relative path to the test data directory.
	TestDataPath string
	// gazelleBinaryDir is the workspace relative path to the location of the gazelle binary
	// we want to test.
	GazelleBinaryDir string
	// gazelleBinaryName is the name of the gazelle binary target that we want to test.
	GazelleBinaryName string

	// The suffix for all test input build files. Includes the ".".
	// Default: ".in", so input BUILD files should be named BUILD.in.
	BuildInSuffix string

	// The suffix for all test output build files. Includes the ".".
	// Default: ".out", so out BUILD files should be named BUILD.out.
	BuildOutSuffix string
}

func NewTestGazelleGenerationArgs() *TestGazelleGenerationArgs {
	return &TestGazelleGenerationArgs{
		BuildInSuffix:  ".in",
		BuildOutSuffix: ".out",
	}
}

// TestGazelleGenerationOnPath runs a full gazelle binary on a testdata directory.
// With a test data directory of the form:
//└── <testDataPath>
//    └── some_test
//        ├── WORKSPACE
//        ├── README.md --> README describing what the test does.
//        ├── test.yaml --> YAML file for test configuration.
//        └── app
//            └── sourceFile.foo
//            └── BUILD.in --> BUILD file prior to running gazelle.
//            └── BUILD.out --> BUILD file expected after running gazelle.
func TestGazelleGenerationOnPath(t *testing.T, args *TestGazelleGenerationArgs, files []bazel.RunfileEntry) {
	t.Run(args.Name, func(t *testing.T) {
		var inputs []FileSpec
		var goldens []FileSpec

		var config *testYAML
		for _, f := range files {
			path := f.Path
			trim := filepath.Join(args.TestDataPath, args.Name) + "/"
			shortPath := strings.TrimPrefix(f.ShortPath, trim)

			info, err := os.Stat(path)
			if err != nil {
				t.Fatalf("os.Stat(%q) error: %v", path, err)
			}

			if info.IsDir() {
				continue
			}

			content, err := ioutil.ReadFile(path)
			if err != nil {
				t.Errorf("ioutil.ReadFile(%q) error: %v", path, err)
			}

			if filepath.Base(shortPath) == "test.yaml" {
				if config != nil {
					t.Fatal("only 1 test.yaml is supported")
				}
				config = new(testYAML)
				if err := yaml.Unmarshal(content, config); err != nil {
					t.Fatal(err)
				}
			}

			if strings.HasSuffix(shortPath, args.BuildInSuffix) {
				inputs = append(inputs, FileSpec{
					Path:    filepath.Join(args.Name, strings.TrimSuffix(shortPath, args.BuildInSuffix)+".bazel"),
					Content: string(content),
				})
			} else if strings.HasSuffix(shortPath, args.BuildOutSuffix) {
				goldens = append(goldens, FileSpec{
					Path:    filepath.Join(args.Name, strings.TrimSuffix(shortPath, args.BuildOutSuffix)+".bazel"),
					Content: string(content),
				})
			} else {
				inputs = append(inputs, FileSpec{
					Path:    filepath.Join(args.Name, shortPath),
					Content: string(content),
				})
				goldens = append(goldens, FileSpec{
					Path:    filepath.Join(args.Name, shortPath),
					Content: string(content),
				})
			}
		}

		testdataDir, cleanup := CreateFiles(t, inputs)
		defer cleanup()
		defer func() {
			if t.Failed() {
				shouldUpdate := os.Getenv("UPDATE_SNAPSHOTS") != ""
				buildWorkspaceDirectory := os.Getenv("BUILD_WORKSPACE_DIRECTORY")
				if buildWorkspaceDirectory == "" {
					// Default to ~/<workspace>.
					homeDir, err := os.UserHomeDir()
					if err != nil {
						t.Fatalf("Could not get user's home directory. Error: %v\n", err)
					}
					testWorkspace, err := bazel.TestWorkspace()
					if err != nil {
						t.Fatalf("Could not get the test workspace. Error; %v\n", err)
					}
					buildWorkspaceDirectory = path.Join(homeDir, testWorkspace)
				}
				filepath.Walk(testdataDir, func(walkedPath string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}
					relativePath := strings.TrimPrefix(walkedPath, testdataDir)
					if shouldUpdate {
						if path.Base(walkedPath) == "BUILD.bazel" {
							destFile := strings.TrimSuffix(path.Join(buildWorkspaceDirectory, args.TestDataPath+relativePath), ".bazel") + ".out"

							err := copyFile(walkedPath, destFile)
							if err != nil {
								t.Fatalf("Failed to copy file %v to %v. Error: %v\n", walkedPath, destFile, err)
							}
						}
					}
					t.Logf("%q exists in %v", relativePath, testdataDir)
					return nil
				})
				if !shouldUpdate {
					t.Logf(`
=====================================================================================

Run UPDATE_SNAPSHOTS=true bazel run //path/to/this/test to update BUILD.out files.

=====================================================================================
`)
				}
			}
		}()

		workspaceRoot := filepath.Join(testdataDir, args.Name)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		gazellePath := mustFindGazelle(args.GazelleBinaryDir, args.GazelleBinaryName)
		cmd := exec.CommandContext(ctx, gazellePath)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		cmd.Dir = workspaceRoot
		cmd.Env = append(os.Environ(), fmt.Sprintf("BUILD_WORKSPACE_DIRECTORY=%v", workspaceRoot))
		if err := cmd.Run(); err != nil {
			var e *exec.ExitError
			if !errors.As(err, &e) {
				t.Fatal(err)
			}
		}
		errs := make([]error, 0)
		actualExitCode := cmd.ProcessState.ExitCode()
		if config.Expect.ExitCode != actualExitCode {
			errs = append(errs, fmt.Errorf("expected gazelle exit code: %d\ngot: %d",
				config.Expect.ExitCode, actualExitCode,
			))
		}
		actualStdout := stdout.String()
		if strings.TrimSpace(config.Expect.Stdout) != strings.TrimSpace(actualStdout) {
			errs = append(errs, fmt.Errorf("expected gazelle stdout: %s\ngot: %s",
				config.Expect.Stdout, actualStdout,
			))
		}
		actualStderr := stderr.String()
		if strings.TrimSpace(config.Expect.Stderr) != strings.TrimSpace(actualStderr) {
			errs = append(errs, fmt.Errorf("expected gazelle stderr: %s\ngot: %s",
				config.Expect.Stderr, actualStderr,
			))
		}
		if len(errs) > 0 {
			for _, err := range errs {
				t.Log(err)
			}
			t.FailNow()
		}

		CheckFiles(t, testdataDir, goldens)
	})
}

func mustFindGazelle(gazelleBinaryDir, gazelleBinaryName string) string {
	gazellePath, ok := bazel.FindBinary(gazelleBinaryDir, gazelleBinaryName)
	if !ok {
		panic(fmt.Sprintf("Could not find gazelle binary at %v", gazellePath))
	}
	return gazellePath
}

func copyFile(src string, dest string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return err
	}
	err = destFile.Sync()
	if err != nil {
		return err
	}
	return nil
}

type testYAML struct {
	Expect struct {
		ExitCode int    `json:"exit_code"`
		Stdout   string `json:"stdout"`
		Stderr   string `json:"stderr"`
	} `json:"expect"`
}
