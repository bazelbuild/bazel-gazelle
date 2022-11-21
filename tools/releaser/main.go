/* Copyright 2022 The Bazel Authors. All rights reserved.

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

// releaser is a tool for managing part of the process to release a new version of gazelle.
//
// This CLI is intended to be run directly with the go toolchain. It is not suitable for
// running through 'bazel run' because it expectes to be able to mutate local files.
package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strings"
)

const (
	_tmpBzl = "tmp.bzl"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	if err := cli(ctx, os.Stderr, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func cli(ctx context.Context, stderr *os.File, args []string) error {
	if len(args) > 1 || len(args) == 1 && strings.HasPrefix(strings.TrimLeft(args[0], "-"), "h") {
		fmt.Println(`usage: go run tools/releaser/main.go

This utility is intended to handle many of the steps to release a new version.

Only run it from the root of the bazel-gazelle repository.`)
		return nil
	}

	var verbose bool
	if len(args) == 1 && strings.HasPrefix(strings.TrimLeft(args[0], "-"), "v") {
		verbose = true
	}

	if verbose {
		fmt.Println("Running initial go update commands")
	}
	initialCommands := []struct {
		cmd  string
		args []string
	}{
		{cmd: "go", args: []string{"get", "-t", "-u", "./..."}},
		{cmd: "go", args: []string{"mod", "tidy", "-compat=1.16"}},
		{cmd: "go", args: []string{"mod", "vendor"}},
		{cmd: "find", args: []string{"vendor", "-name", "BUILD.bazel", "-delete"}},
	}
	for _, c := range initialCommands {
		if out, err := exec.CommandContext(ctx, c.cmd, c.args...).CombinedOutput(); err != nil {
			fmt.Println(string(out))
			return err
		}
	}

	workspace, err := os.ReadFile("WORKSPACE")
	if err != nil {
		return err
	}

	workspaceBuffer := bytes.NewBuffer(workspace)
	workspaceWithoutDirectives := new(bytes.Buffer)

	if verbose {
		fmt.Println("Preparing temporary WORKSPACE without gazelle directives.")
	}
	directiveStart, err := getWorkspaceWithouthDirectives(workspaceBuffer, workspaceWithoutDirectives)
	if err != nil {
		return err
	}

	// write the directive-less workspace and update repos
	if err := os.WriteFile("WORKSPACE", workspaceWithoutDirectives.Bytes(), 0666); err != nil {
		return err
	}

	if verbose {
		fmt.Println("Running update-repos outputting to temporary file.")
	}
	cmd := exec.CommandContext(ctx, "bazel", "run", "//:gazelle", "--", "update-repos", "-from_file=go.mod", fmt.Sprintf("-to_macro=%s%%gazelle_dependencies", _tmpBzl))
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Println(string(out))
		return err
	}

	depsMaybeBuff := new(bytes.Buffer)
	workspaceDirectivesBuff := new(bytes.Buffer)
	// parse the resulting tmp.bzl for deps.bzl and WORKSPACE updates
	if verbose {
		fmt.Println("Parsing temporary bzl file to prepare deps.bzl and WORKSPACE modifications.")
	}
	if err = getBuffsFromTmp(depsMaybeBuff, workspaceDirectivesBuff); err != nil {
		return err
	}

	// update deps
	depsFile, err := os.ReadFile("deps.bzl")
	if err != nil {
		return err
	}
	existingDepsScanner := bufio.NewScanner(bytes.NewBuffer(depsFile))
	newDepsBuffer := new(bytes.Buffer)
	var afterSkylib bool
	for existingDepsScanner.Scan() {
		if bytes.Contains(existingDepsScanner.Bytes(), []byte("skylib")) {
			afterSkylib = true
		}

		if afterSkylib && bytes.HasSuffix(existingDepsScanner.Bytes(), []byte("_maybe(")) {
			break
		}

		if _, err := newDepsBuffer.Write(existingDepsScanner.Bytes()); err != nil {
			return err
		}
		if _, err := newDepsBuffer.Write([]byte("\n")); err != nil {
			return err
		}
	}
	// just append the rest
	if verbose {
		fmt.Println("Writing new deps.bzl")
	}
	if _, err := newDepsBuffer.Write(depsMaybeBuff.Bytes()); err != nil {
		return err
	}
	if err := os.WriteFile("deps.bzl", newDepsBuffer.Bytes(), 0644); err != nil {
		return err
	}

	// update WORKSPACE
	if verbose {
		fmt.Println("Writing new WORKSPACE")
	}
	directivelessWorkspace := workspaceWithoutDirectives.Bytes()
	newWorkspace := make([]byte, 0, len(directivelessWorkspace))
	newWorkspace = append(newWorkspace, directivelessWorkspace[:directiveStart]...)
	newWorkspace = append(newWorkspace, workspaceDirectivesBuff.Bytes()...)
	newWorkspace = append(newWorkspace, directivelessWorkspace[directiveStart:]...)

	if err := os.WriteFile("WORKSPACE", newWorkspace, 0644); err != nil {
		return err
	}

	// cleanup before final gazelle run
	if verbose {
		fmt.Println("Cleaning up temporary files")
	}
	if err := os.Remove(_tmpBzl); err != nil {
		return err
	}

	if verbose {
		fmt.Println("Running final gazelle run, and copying some language specific build files.")
	}
	finalizationCommands := []struct {
		cmd  string
		args []string
	}{
		{cmd: "bazel", args: []string{"run", "//:gazelle"}},
		{cmd: "bazel", args: []string{"build", "//language/proto:known_imports"}},
		{cmd: "cp", args: []string{"-f", "bazel-bin/language/proto/known_imports.go", "language/proto/known_imports.go"}},
		{cmd: "bazel", args: []string{"build", "//language/proto:known_proto_imports"}},
		{cmd: "cp", args: []string{"-f", "bazel-bin/language/proto/known_proto_imports.go", "language/proto/known_proto_imports.go"}},
		{cmd: "bazel", args: []string{"build", "//language/proto:known_go_imports"}},
		{cmd: "cp", args: []string{"-f", "bazel-bin/language/proto/known_go_imports.go", "language/proto/known_go_imports.go"}},
	}
	for _, c := range finalizationCommands {
		if out, err := exec.CommandContext(ctx, c.cmd, c.args...).CombinedOutput(); err != nil {
			fmt.Println(string(out))
			return err
		}
	}
	return nil
}

func getWorkspaceWithouthDirectives(workspaceBuffer, workspaceWithoutDirectives *bytes.Buffer) (int, error) {
	var directiveStart int
	var directiveFound bool
	workspaceScanner := bufio.NewScanner(workspaceBuffer)
	for workspaceScanner.Scan() {
		if bytes.HasPrefix(workspaceScanner.Bytes(), []byte("# gazelle:repository go_repository")) {
			directiveFound = true
			continue
		}
		n, err := workspaceWithoutDirectives.Write(workspaceScanner.Bytes())
		if err != nil {
			return directiveStart, err
		}
		_, err = workspaceWithoutDirectives.Write([]byte("\n"))
		if err != nil {
			return directiveStart, err
		}
		if !directiveFound {
			directiveStart += n + 1
		}
	}

	return directiveStart, nil
}

func getBuffsFromTmp(depsMaybeBuff, workspaceDirectivesBuff *bytes.Buffer) error {
	tmpbzl, err := os.ReadFile(_tmpBzl)
	if err != nil {
		return err
	}

	attributeRegex := regexp.MustCompile(`^\s+(name|importpath) = "(.+)",$`)

	var foundDef bool
	var name string
	tmpscanner := bufio.NewScanner(bytes.NewBuffer(tmpbzl))
	for tmpscanner.Scan() {
		if bytes.HasPrefix(tmpscanner.Bytes(), []byte("def ")) {
			foundDef = true
			continue
		}
		if !foundDef {
			continue
		}

		if bytes.HasPrefix(bytes.TrimSpace(tmpscanner.Bytes()), []byte("go_repository(")) {
			if _, err := depsMaybeBuff.Write([]byte(`    _maybe(
        go_repository,
`)); err != nil {
				return err
			}
			continue
		}

		// all other lines can be copied directly for the new deps.bzl
		if _, err := depsMaybeBuff.Write(tmpscanner.Bytes()); err != nil {
			return err
		}
		if _, err := depsMaybeBuff.Write([]byte("\n")); err != nil {
			return err
		}

		// check if its a line we care about
		if m := attributeRegex.FindAllSubmatch(tmpscanner.Bytes(), -1 /* n */); len(m) > 0 {
			if bytes.Equal(m[0][1], []byte("name")) {
				name = string(m[0][2])
			}

			if bytes.Equal(m[0][1], []byte("importpath")) {
				var suffix string
				if name == "com_github_bazelbuild_buildtools" {
					suffix = " build_naming_convention=go_default_library"
				}

				fmt.Fprintf(workspaceDirectivesBuff, "# gazelle:repository go_repository name=%s importpath=%s%s\n", name, m[0][2], suffix)
			}
		}

		// if we found com_github_bazelbuild_buildtools it is extra special
		if bytes.Contains(tmpscanner.Bytes(), []byte("com_github_bazelbuild_buildtools")) {
			if _, err := depsMaybeBuff.Write([]byte(`        build_naming_convention = "go_default_library",
`)); err != nil {
				return err
			}
		}
	}
	return nil
}
