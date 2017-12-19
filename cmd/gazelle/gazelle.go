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

// Command gazelle is a BUILD file generator for Go projects.
// See "gazelle --help" for more details.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

type command int

const (
	updateCmd command = iota
	fixCmd
)

var commandFromName = map[string]command{
	"update": updateCmd,
	"fix":    fixCmd,
}

func usage(fs *flag.FlagSet) {
	fmt.Fprintln(os.Stderr, `usage: gazelle <command> [flags...] [package-dirs...]

Gazelle is a BUILD file generator for Go projects. It can create new BUILD files
for a project that follows "go build" conventions, and it can update BUILD files
if they already exist. It can be invoked directly in a project workspace, or
it can be run on an external dependency during the build as part of the
go_repository rule.

Gazelle may be run with one of the commands below. If no command is given,
Gazelle defaults to "update".

  update - Gazelle will create new BUILD files or update existing BUILD files
      if needed.
	fix - in addition to the changes made in update, Gazelle will make potentially
	    breaking changes. For example, it may delete obsolete rules or rename
      existing rules.

Gazelle has several output modes which can be selected with the -mode flag. The
output mode determines what Gazelle does with updated BUILD files.

  fix (default) - write updated BUILD files back to disk.
  print - print updated BUILD files to stdout.
  diff - diff updated BUILD files against existing files in unified format.

Gazelle accepts a list of paths to Go package directories to process (defaults
to . if none given). It recursively traverses subdirectories. All directories
must be under the directory specified by -repo_root; if -repo_root is not given,
this is the directory containing the WORKSPACE file.

Gazelle is under active delevopment, and its interface may change
without notice.

FLAGS:
`)
	fs.PrintDefaults()
}

func main() {
	log.SetPrefix("gazelle: ")
	log.SetFlags(0) // don't print timestamps

	run(os.Args[1:])
}

func run(args []string) error {
	cmd := updateCmd
	if len(args) > 0 {
		c, ok := commandFromName[args[0]]
		if ok {
			cmd = c
			args = args[1:]
		}
	}

	switch cmd {
	case fixCmd, updateCmd:
		return runFixUpdate(cmd, args)
	default:
		log.Panicf("unknown command: %v", cmd)
	}
	return nil
}
