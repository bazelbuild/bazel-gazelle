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
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"regexp"
	"strings"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	if err := run(ctx, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, stderr *os.File) error {
	var (
		help, verbose bool
	)

	flag.BoolVar(&help, "help", false, "print usage / help information")
	flag.BoolVar(&help, "h", false, "print usage / help information (shorthand)")

	flag.BoolVar(&verbose, "verbose", false, "increase verbosity")
	flag.BoolVar(&verbose, "v", false, "increase verbosity (shorthand)")

	flag.Parse()

	if help {
		fmt.Println(`usage: bazel run //tools/releaser

This utility is intended to handle many of the steps to release a new version.`)
		return nil
	}

	workspacePath := path.Join(os.Getenv("BUILD_WORKSPACE_DIRECTORY"), "WORKSPACE")
	depsPath := path.Join(os.Getenv("BUILD_WORKSPACE_DIRECTORY"), "deps.bzl")
	_tmpBzl := "tmp.bzl"
	tmpBzlPath := path.Join(os.Getenv("BUILD_WORKSPACE_DIRECTORY"), _tmpBzl)

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
		cmd := exec.CommandContext(ctx, c.cmd, c.args...)
		cmd.Dir = os.Getenv("BUILD_WORKSPACE_DIRECTORY")
		if out, err := cmd.CombinedOutput(); err != nil {
			fmt.Println(string(out))
			return err
		}
	}

	workspace, err := os.ReadFile(workspacePath)
	if err != nil {
		return err
	}

	workspaceBuffer := bytes.NewBuffer(workspace)
	workspaceWithoutDirectives := new(bytes.Buffer)

	if verbose {
		fmt.Println("Preparing temporary WORKSPACE without gazelle directives.")
	}
	if err := getWorkspaceWithouthDirectives(workspaceBuffer, workspaceWithoutDirectives); err != nil {
		return err
	}

	// write the directive-less workspace and update repos
	if err := os.WriteFile(workspacePath, workspaceWithoutDirectives.Bytes(), 0666); err != nil {
		return err
	}

	if verbose {
		fmt.Println("Running update-repos outputting to temporary file.")
	}
	cmd := exec.CommandContext(ctx, "bazel", "run", "//:gazelle", "--", "update-repos", "-from_file=go.mod", fmt.Sprintf("-to_macro=%s%%gazelle_dependencies", _tmpBzl))
	cmd.Dir = os.Getenv("BUILD_WORKSPACE_DIRECTORY")
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
	if err = getBuffsFromTmp(tmpBzlPath, depsMaybeBuff, workspaceDirectivesBuff); err != nil {
		return err
	}

	// update deps
	depsFile, err := os.Open(depsPath)
	if err != nil {
		return err
	}
	defer depsFile.Close()
	existingDepsScanner := bufio.NewScanner(depsFile)
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
	if err := os.WriteFile(depsPath, newDepsBuffer.Bytes(), 0644); err != nil {
		return err
	}

	// append WORKSPACE with directives at the end
	if verbose {
		fmt.Println("Append WORKSPACE with directives")
	}
	file, err := os.OpenFile(workspacePath, os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	if _, err := file.Write(workspaceWithoutDirectives.Bytes()); err != nil {
		return err
	}
	if _, err := file.Write(workspaceDirectivesBuff.Bytes()); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}

	// cleanup before final gazelle run
	if verbose {
		fmt.Println("Cleaning up temporary files")
	}
	if err := os.Remove(tmpBzlPath); err != nil {
		return err
	}

	if verbose {
		fmt.Println("Running final gazelle run, and copying some language specific build files.")
	}
	finalizationCommands := []struct {
		cmd  string
		args []string

		// I don't love this continue boolean but apparently cp has no way to suppress a "identical file" error
		continueOnError bool
	}{
		{cmd: "bazel", args: []string{"run", "//:gazelle"}},
		{cmd: "bazel", args: []string{"build", "//language/proto:known_imports"}},
		{cmd: "cp", continueOnError: true, args: []string{"-f", path.Join(os.Getenv("BINDIR"), "language/proto/known_imports.go"), "language/proto/known_imports.go"}},
		{cmd: "bazel", args: []string{"build", "//language/proto:known_proto_imports"}},
		{cmd: "cp", continueOnError: true, args: []string{"-f", path.Join(os.Getenv("BINDIR"), "language/proto/known_proto_imports.go"), "language/proto/known_proto_imports.go"}},
		{cmd: "bazel", args: []string{"build", "//language/proto:known_go_imports"}},
		{cmd: "cp", continueOnError: true, args: []string{"-f", path.Join(os.Getenv("BINDIR"), "language/proto/known_go_imports.go"), "language/proto/known_go_imports.go"}},
	}
	for _, c := range finalizationCommands {
		cmd := exec.CommandContext(ctx, c.cmd, c.args...)
		cmd.Dir = os.Getenv("BUILD_WORKSPACE_DIRECTORY")
		if out, err := cmd.CombinedOutput(); err != nil {
			prefix := "ERROR"
			if c.continueOnError {
				prefix = "WARNING"
			}
			fmt.Printf("%s - %s", prefix, string(out))
			if !c.continueOnError {
				return err
			}
		}
	}

	if verbose {
		fmt.Println("Release prepared.")
	}
	return nil
}

func getWorkspaceWithouthDirectives(workspaceBuffer, workspaceWithoutDirectives *bytes.Buffer) error {
	workspaceScanner := bufio.NewScanner(workspaceBuffer)
	for workspaceScanner.Scan() {
		if strings.HasPrefix(workspaceScanner.Text(), "# gazelle:repository go_repository") {
			continue
		}
		_, err := workspaceWithoutDirectives.Write(workspaceScanner.Bytes())
		if err != nil {
			return err
		}
		_, err = workspaceWithoutDirectives.Write([]byte("\n"))
		if err != nil {
			return err
		}
	}
	return nil
}

func getBuffsFromTmp(tmpBzlPath string, depsMaybeBuff, workspaceDirectivesBuff *bytes.Buffer) error {
	tmpbzl, err := os.Open(tmpBzlPath)
	if err != nil {
		return err
	}
	defer tmpbzl.Close()

	attributeRegex := regexp.MustCompile(`^\s+(name|importpath) = "(.+)",$`)

	var foundDef bool
	var name string
	tmpscanner := bufio.NewScanner(tmpbzl)
	for tmpscanner.Scan() {
		currentLine := tmpscanner.Text()
		if strings.HasPrefix(currentLine, "def gazelle_dependencies") {
			foundDef = true
			continue
		}
		if !foundDef {
			continue
		}

		if strings.HasPrefix(strings.TrimSpace(currentLine), "go_repository(") {
			if _, err := depsMaybeBuff.WriteString(`    _maybe(
        go_repository,
`); err != nil {
				return err
			}
			continue
		}

		// all other lines can be copied directly for the new deps.bzl
		if _, err := depsMaybeBuff.WriteString(currentLine); err != nil {
			return err
		}
		if _, err := depsMaybeBuff.WriteString("\n"); err != nil {
			return err
		}

		// check if its a line we care about
		if m := attributeRegex.FindAllStringSubmatch(currentLine, -1 /* n */); len(m) > 0 {
			if m[0][1] == "name" {
				name = string(m[0][2])
			}

			if m[0][1] == "importpath" {
				var suffix string
				if name == "com_github_bazelbuild_buildtools" {
					suffix = " build_naming_convention=go_default_library"
				}

				fmt.Fprintf(workspaceDirectivesBuff, "# gazelle:repository go_repository name=%s importpath=%s%s\n", name, m[0][2], suffix)
			}
		}

		// if we found com_github_bazelbuild_buildtools it is extra special
		if strings.Contains(currentLine, "com_github_bazelbuild_buildtools") {
			if _, err := depsMaybeBuff.WriteString(`        build_naming_convention = "go_default_library",
`); err != nil {
				return err
			}
		}
	}
	return nil
}
