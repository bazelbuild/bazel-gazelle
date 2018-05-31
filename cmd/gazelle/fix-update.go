/* Copyright 2017 The Bazel Authors. All rights reserved.

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
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/internal/config"
	"github.com/bazelbuild/bazel-gazelle/internal/generator"
	"github.com/bazelbuild/bazel-gazelle/internal/label"
	"github.com/bazelbuild/bazel-gazelle/internal/labeler"
	"github.com/bazelbuild/bazel-gazelle/internal/merger"
	"github.com/bazelbuild/bazel-gazelle/internal/packages"
	"github.com/bazelbuild/bazel-gazelle/internal/repos"
	"github.com/bazelbuild/bazel-gazelle/internal/resolve"
	"github.com/bazelbuild/bazel-gazelle/internal/rule"
	"github.com/bazelbuild/bazel-gazelle/internal/wspace"
	bzl "github.com/bazelbuild/buildtools/build"
)

// updateConfig holds configuration information needed to run the fix and
// update commands. This includes everything in config.Config, but it also
// includes some additional fields that aren't relevant to other packages.
type updateConfig struct {
	emit              emitFunc
	outDir, outSuffix string
	repos             []repos.Repo
}

type emitFunc func(*config.Config, *bzl.File, string) error

var modeFromName = map[string]emitFunc{
	"print": printFile,
	"fix":   fixFile,
	"diff":  diffFile,
}

func newUpdateConfig() *updateConfig {
	return &updateConfig{emit: printFile}
}

const updateName = "_update"

func getUpdateConfig(c *config.Config) *updateConfig {
	return c.Exts[updateName].(*updateConfig)
}

// visitRecord stores information about about a directory visited with
// packages.Walk.
type visitRecord struct {
	// pkgRel is the slash-separated path to the visited directory, relative to
	// the repository root. "" for the repository root itself.
	pkgRel string

	// rules is a list of generated Go rules.
	rules []*rule.Rule

	// empty is a list of empty Go rules that may be deleted.
	empty []*rule.Rule

	// file is the build file being processed.
	file *rule.File
}

type byPkgRel []visitRecord

func (vs byPkgRel) Len() int           { return len(vs) }
func (vs byPkgRel) Less(i, j int) bool { return vs[i].pkgRel < vs[j].pkgRel }
func (vs byPkgRel) Swap(i, j int)      { vs[i], vs[j] = vs[j], vs[i] }

func runFixUpdate(cmd command, args []string) error {
	cexts := []config.Configurer{&config.CommonConfigurer{}}
	c, err := newFixUpdateConfiguration(cmd, args, cexts)
	if err != nil {
		return err
	}
	if cmd == fixCmd {
		// Only check the version when "fix" is run. Generated build files
		// frequently work with older version of rules_go, and we don't want to
		// nag too much since there's no way to disable this warning.
		checkRulesGoVersion(c.RepoRoot)
	}

	l := labeler.NewLabeler(c)
	ruleIndex := resolve.NewRuleIndex()

	var visits []visitRecord

	// Visit all directories in the repository.
	packages.Walk(c, cexts, func(dir, rel string, c *config.Config, pkg *packages.Package, file *rule.File, isUpdateDir bool) {
		// If this file is ignored or if Gazelle was not asked to update this
		// directory, just index the build file and move on.
		if !isUpdateDir {
			if file != nil {
				ruleIndex.AddRulesFromFile(c, file)
			}
			return
		}

		// Fix any problems in the file.
		if file != nil {
			merger.FixFile(c, file)
		}

		// If the file exists, but no Go code is present, create an empty package.
		// This lets us delete existing rules.
		if pkg == nil && file != nil {
			pkg = packages.EmptyPackage(c, dir, rel)
		}

		// Generate new rules and merge them into the existing file (if present).
		if pkg != nil {
			g := generator.NewGenerator(c, l, file)
			rules, empty := g.GenerateRules(pkg)
			if file == nil {
				file = rule.EmptyFile(filepath.Join(c.RepoRoot, filepath.FromSlash(rel), c.DefaultBuildFileName()))
				for _, r := range rules {
					r.Insert(file)
				}
			} else {
				merger.MergeFile(file, empty, rules, merger.PreResolveAttrs)
			}
			visits = append(visits, visitRecord{
				pkgRel: rel,
				rules:  rules,
				empty:  empty,
				file:   file,
			})
		}

		// Add library rules to the dependency resolution table.
		if file != nil {
			ruleIndex.AddRulesFromFile(c, file)
		}
	})

	// Finish building the index for dependency resolution.
	ruleIndex.Finish()

	// Resolve dependencies.
	uc := getUpdateConfig(c)
	rc := repos.NewRemoteCache(uc.repos)
	resolver := resolve.NewResolver(c, l, ruleIndex, rc)
	for _, v := range visits {
		for _, r := range v.rules {
			resolver.ResolveRule(r, v.pkgRel)
		}
		merger.MergeFile(v.file, v.empty, v.rules, merger.PostResolveAttrs)
	}

	// Emit merged files.
	for _, v := range visits {
		merger.FixLoads(v.file)
		v.file.Sync()
		bzl.Rewrite(v.file.File, nil) // have buildifier 'format' our rules.

		path := v.file.Path
		if uc.outDir != "" {
			stem := filepath.Base(v.file.Path) + uc.outSuffix
			path = filepath.Join(uc.outDir, v.pkgRel, stem)
		}
		if err := uc.emit(c, v.file.File, path); err != nil {
			log.Print(err)
		}
	}
	return nil
}

func newFixUpdateConfiguration(cmd command, args []string, cexts []config.Configurer) (*config.Config, error) {
	var err error
	c := config.New()
	uc := newUpdateConfig()
	c.Exts[updateName] = uc

	fs := flag.NewFlagSet("gazelle", flag.ContinueOnError)
	// Flag will call this on any parse error. Don't print usage unless
	// -h or -help were passed explicitly.
	fs.Usage = func() {}

	knownImports := multiFlag{}
	buildTags := fs.String("build_tags", "", "comma-separated list of build tags. If not specified, Gazelle will not\n\tfilter sources with build constraints.")
	external := fs.String("external", "external", "external: resolve external packages with go_repository\n\tvendored: resolve external packages as packages in vendor/")
	var goPrefix explicitFlag
	fs.Var(&goPrefix, "go_prefix", "prefix of import paths in the current workspace")
	var repoRoot string
	fs.StringVar(&repoRoot, "repo_root", "", "path to a directory which corresponds to go_prefix, otherwise gazelle searches for it.")
	fs.Var(&knownImports, "known_import", "import path for which external resolution is skipped (can specify multiple times)")
	mode := fs.String("mode", "fix", "print: prints all of the updated BUILD files\n\tfix: rewrites all of the BUILD files in place\n\tdiff: computes the rewrite but then just does a diff")
	outDir := fs.String("experimental_out_dir", "", "write build files to an alternate directory tree")
	outSuffix := fs.String("experimental_out_suffix", "", "extra suffix appended to build file names. Only used if -experimental_out_dir is also set.")
	var proto explicitFlag
	fs.Var(&proto, "proto", "default: generates new proto rules\n\tdisable: does not touch proto rules\n\tlegacy (deprecated): generates old proto rules")

	for _, cext := range cexts {
		cext.RegisterFlags(fs, cmd.String(), c)
	}

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			fixUpdateUsage(fs)
			os.Exit(0)
		}
		// flag already prints the error; don't print it again.
		log.Fatal("Try -help for more information.")
	}

	c.Dirs = fs.Args()
	if len(c.Dirs) == 0 {
		c.Dirs = []string{"."}
	}
	if repoRoot == "" {
		if len(c.Dirs) == 1 {
			repoRoot, err = wspace.Find(c.Dirs[0])
		} else {
			repoRoot, err = wspace.Find(".")
		}
		if err != nil {
			return nil, fmt.Errorf("-repo_root not specified, and WORKSPACE cannot be found: %v", err)
		}
	}
	c.RepoRoot, err = filepath.Abs(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to find absolute path of repo root: %v", repoRoot, err)
	}
	c.RepoRoot, err = filepath.EvalSymlinks(c.RepoRoot)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to resolve symlinks: %v", repoRoot, err)
	}
	for i, dir := range c.Dirs {
		dir, err = filepath.Abs(c.Dirs[i])
		if err != nil {
			return nil, fmt.Errorf("%s: failed to find absolute path: %v", c.Dirs[i], err)
		}
		dir, err = filepath.EvalSymlinks(dir)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to resolve symlinks: %v", c.Dirs[i], err)
		}
		if !isDescendingDir(dir, c.RepoRoot) {
			return nil, fmt.Errorf("dir %q is not a subdirectory of repo root %q", dir, c.RepoRoot)
		}
		c.Dirs[i] = dir
	}

	for _, cext := range cexts {
		if err := cext.CheckFlags(c); err != nil {
			return nil, err
		}
	}

	c.SetBuildTags(*buildTags)
	c.PreprocessTags()

	if goPrefix.set {
		c.GoPrefix = goPrefix.value
	} else {
		c.GoPrefix, err = loadGoPrefix(c)
		if err != nil {
			return nil, err
		}
	}
	if err := config.CheckPrefix(c.GoPrefix); err != nil {
		return nil, err
	}

	c.ShouldFix = cmd == fixCmd

	c.DepMode, err = config.DependencyModeFromString(*external)
	if err != nil {
		return nil, err
	}

	if proto.set {
		c.ProtoMode, err = config.ProtoModeFromString(proto.value)
		if err != nil {
			return nil, err
		}
		c.ProtoModeExplicit = true
	}

	emit, ok := modeFromName[*mode]
	if !ok {
		return nil, fmt.Errorf("unrecognized emit mode: %q", *mode)
	}
	uc.emit = emit

	uc.outDir = *outDir
	uc.outSuffix = *outSuffix

	workspacePath := filepath.Join(c.RepoRoot, "WORKSPACE")
	if workspace, err := rule.LoadFile(workspacePath); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	} else {
		if err := fixWorkspace(c, workspace); err != nil {
			return nil, err
		}
		c.RepoName = findWorkspaceName(workspace)
		uc.repos = repos.ListRepositories(workspace)
	}
	repoPrefixes := make(map[string]bool)
	for _, r := range uc.repos {
		repoPrefixes[r.GoPrefix] = true
	}
	for _, imp := range knownImports {
		if repoPrefixes[imp] {
			continue
		}
		repo := repos.Repo{
			Name:     label.ImportPathToBazelRepoName(imp),
			GoPrefix: imp,
		}
		uc.repos = append(uc.repos, repo)
	}

	return c, nil
}

func fixUpdateUsage(fs *flag.FlagSet) {
	fmt.Fprint(os.Stderr, `usage: gazelle [fix|update] [flags...] [package-dirs...]

The update command creates new build files and update existing BUILD files
when needed.

The fix command also creates and updates build files, and in addition, it may
make potentially breaking updates to usage of rules. For example, it may
delete obsolete rules or rename existing rules.

There are several output modes which can be selected with the -mode flag. The
output mode determines what Gazelle does with updated BUILD files.

  fix (default) - write updated BUILD files back to disk.
  print - print updated BUILD files to stdout.
  diff - diff updated BUILD files against existing files in unified format.

Gazelle accepts a list of paths to Go package directories to process (defaults
to the working directory if none are given). It recursively traverses
subdirectories. All directories must be under the directory specified by
-repo_root; if -repo_root is not given, this is the directory containing the
WORKSPACE file.

FLAGS:

`)
	fs.PrintDefaults()
}

func loadBuildFile(c *config.Config, dir string) (*bzl.File, error) {
	var buildPath string
	for _, base := range c.ValidBuildFileNames {
		p := filepath.Join(dir, base)
		fi, err := os.Stat(p)
		if err == nil {
			if fi.Mode().IsRegular() {
				buildPath = p
				break
			}
			continue
		}
		if !os.IsNotExist(err) {
			return nil, err
		}
	}
	if buildPath == "" {
		return nil, os.ErrNotExist
	}

	data, err := ioutil.ReadFile(buildPath)
	if err != nil {
		return nil, err
	}
	return bzl.Parse(buildPath, data)
}

func loadGoPrefix(c *config.Config) (string, error) {
	f, err := loadBuildFile(c, c.RepoRoot)
	if err != nil {
		return "", errors.New("-go_prefix not set")
	}
	for _, d := range rule.ParseDirectives(f) {
		if d.Key == "prefix" {
			return d.Value, nil
		}
	}
	for _, s := range f.Stmt {
		c, ok := s.(*bzl.CallExpr)
		if !ok {
			continue
		}
		l, ok := c.X.(*bzl.LiteralExpr)
		if !ok {
			continue
		}
		if l.Token != "go_prefix" {
			continue
		}
		if len(c.List) != 1 {
			return "", fmt.Errorf("-go_prefix not set, and %s has go_prefix(%v) with too many args", f.Path, c.List)
		}
		v, ok := c.List[0].(*bzl.StringExpr)
		if !ok {
			return "", fmt.Errorf("-go_prefix not set, and %s has go_prefix(%v) which is not a string", f.Path, bzl.FormatString(c.List[0]))
		}
		return v.Value, nil
	}
	return "", fmt.Errorf("-go_prefix not set, and no # gazelle:prefix directive found in %s", f.Path)
}

func fixWorkspace(c *config.Config, workspace *rule.File) error {
	uc := getUpdateConfig(c)
	if !c.ShouldFix {
		return nil
	}
	shouldFix := false
	for _, d := range c.Dirs {
		if d == c.RepoRoot {
			shouldFix = true
		}
	}
	if !shouldFix {
		return nil
	}

	merger.FixWorkspace(workspace)
	merger.FixLoads(workspace)
	if err := merger.CheckGazelleLoaded(workspace); err != nil {
		return err
	}
	workspace.Sync()
	return uc.emit(c, workspace.File, workspace.Path)
}

func findWorkspaceName(f *rule.File) string {
	for _, r := range f.Rules {
		if r.Kind() == "workspace" {
			return r.Name()
		}
	}
	return ""
}

func isDescendingDir(dir, root string) bool {
	rel, err := filepath.Rel(root, dir)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return !strings.HasPrefix(rel, "..")
}
