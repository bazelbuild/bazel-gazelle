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

package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/pathtools"
	"github.com/bazelbuild/buildtools/build"
)

const usageMessage = `usage: move_labels [-repo_root=root] [-from=dir] -to=dir

move_labels updates Bazel labels in a tree containing build files after the
tree has been moved to a new location. This is useful for vendoring
repositories that already have Bazel build files.

`

func main() {
	log.SetPrefix("move_labels: ")
	log.SetFlags(0)
	if err := run(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

func run(args []string) error {
	c, err := newConfiguration(args)
	if err != nil {
		return err
	}

	files, err := moveLabelsInDir(c)
	if err != nil {
		return err
	}

	var errs errorList
	for _, file := range files {
		content := build.Format(file)
		if err := ioutil.WriteFile(file.Path, content, 0666); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errs
	}
	return nil
}

func moveLabelsInDir(c *configuration) ([]*build.File, error) {
	toRel, err := filepath.Rel(c.repoRoot, c.to)
	if err != nil {
		return nil, err
	}
	toRel = filepath.ToSlash(toRel)

	var files []*build.File
	var errors errorList
	err = filepath.Walk(c.to, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if name := info.Name(); name != "BUILD" && name != "BUILD.bazel" {
			return nil
		}
		content, err := ioutil.ReadFile(path)
		if err != nil {
			errors = append(errors, err)
			return nil
		}
		file, err := build.Parse(path, content)
		if err != nil {
			errors = append(errors, err)
			return nil
		}
		moveLabelsInFile(file, c.from, toRel)
		files = append(files, file)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(errors) > 0 {
		return nil, errors
	}
	return files, nil
}

func moveLabelsInFile(file *build.File, from, to string) {
	build.Edit(file, func(x build.Expr, _ []build.Expr) build.Expr {
		str, ok := x.(*build.StringExpr)
		if !ok {
			return nil
		}
		label := str.Value
		var moved string
		if strings.Contains(label, "$(location") {
			moved = moveLocations(from, to, label)
		} else {
			moved = moveLabel(from, to, label)
		}
		if moved == label {
			return nil
		}
		return &build.StringExpr{Value: moved}
	})
}

func moveLabel(from, to, str string) string {
	l, err := label.Parse(str)
	if err != nil {
		return str
	}
	if l.Relative || l.Repo != "" ||
		l.Pkg == "visibility" || l.Pkg == "conditions" ||
		pathtools.HasPrefix(l.Pkg, to) || !pathtools.HasPrefix(l.Pkg, from) {
		return str
	}
	l.Pkg = path.Join(to, pathtools.TrimPrefix(l.Pkg, from))
	return l.String()
}

var locationsRegexp = regexp.MustCompile(`\$\(locations?\s*([^)]*)\)`)

// moveLocations fixes labels within $(location) and $(locations) expansions.
func moveLocations(from, to, str string) string {
	matches := locationsRegexp.FindAllStringSubmatchIndex(str, -1)
	buf := new(bytes.Buffer)
	pos := 0
	for _, match := range matches {
		buf.WriteString(str[pos:match[2]])
		label := str[match[2]:match[3]]
		moved := moveLabel(from, to, label)
		buf.WriteString(moved)
		buf.WriteString(str[match[3]:match[1]])
		pos = match[1]
	}
	buf.WriteString(str[pos:])
	return buf.String()
}

func isBuiltinLabel(label string) bool {
	return strings.HasPrefix(label, "//visibility:") || strings.HasPrefix(label, "//conditions:")
}

type configuration struct {
	// repoRoot is the repository root directory, formatted as an absolute
	// file system path.
	repoRoot string

	// from is the original location of the build files within their repository,
	// formatted as a slash-separated relative path from the original
	// repository root.
	from string

	// to is the new location of the build files, formatted as an absolute
	// file system path.
	to string
}

func newConfiguration(args []string) (*configuration, error) {
	var err error
	c := &configuration{}
	fs := flag.NewFlagSet("move_labels", flag.ContinueOnError)
	fs.Usage = func() {}
	fs.StringVar(&c.repoRoot, "repo_root", "", "repository root directory; inferred to be parent directory containing WORKSPACE file")
	fs.StringVar(&c.from, "from", "", "original location of build files, formatted as a slash-separated relative path from the original repository root")
	fs.StringVar(&c.to, "to", "", "new location of build files, formatted as a file system path")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			fmt.Fprint(os.Stderr, usageMessage)
			fs.PrintDefaults()
			os.Exit(0)
		}
		// flag already prints an error; don't print again.
		return nil, errors.New("Try -help for more information")
	}

	if c.repoRoot == "" {
		c.repoRoot, err = findRepoRoot()
		if err != nil {
			return nil, err
		}
	}
	c.repoRoot, err = filepath.Abs(c.repoRoot)
	if err != nil {
		return nil, err
	}

	if c.to == "" {
		return nil, errors.New("-to must be specified. Try -help for more information.")
	}
	c.to, err = filepath.Abs(c.to)
	if err != nil {
		return nil, err
	}

	if len(fs.Args()) != 0 {
		return nil, errors.New("No positional arguments expected. Try -help for more information.")
	}

	return c, nil
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		workspacePath := filepath.Join(dir, "WORKSPACE")
		_, err := os.Stat(workspacePath)
		if err == nil {
			return dir, nil
		}
		if strings.HasSuffix(dir, string(os.PathSeparator)) {
			// root directory
			break
		}
		dir = filepath.Dir(dir)
	}
	return "", fmt.Errorf("could not find WORKSPACE file. -repo_root must be set explicitly.")
}

type errorList []error

func (e errorList) Error() string {
	buf := new(bytes.Buffer)
	for _, err := range e {
		fmt.Fprintln(buf, err.Error())
	}
	return buf.String()
}
