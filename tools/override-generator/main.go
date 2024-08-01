package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/repo"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/bazelbuild/buildtools/build"
)

const (
	_name  = "override-generator"
	_usage = "usage: This script converts `go_repository` rules to Gazelle `go_deps` overrides to assist in the migration to Bzlmod."
)

// override kinds.
const (
	_goDepsf         = `go_deps = use_extension("%s//:extensions.bzl", "go_deps")`
	_gazelleOverride = "go_deps.gazelle_override"
	_archiveOverride = "go_deps.archive_override"
	_moduleOverride  = "go_deps.module_override"
)

// attribute constants that are used multiple times.
const (
	_buildFileGenerationAttr = "build_file_generation"
	_buildFileProtoModeAttr  = "build_file_proto_mode"
	_patchArgsAttr           = "patch_args"
	_buildDirectivesAttr     = "build_directives"
	_directivesAttr          = "directives"
)

var _defaultValues = map[string]string{
	_buildFileGenerationAttr: "auto",
	_buildFileProtoModeAttr:  "default",
}

var _mapAttrToOverride = map[string]string{
	_buildDirectivesAttr:     _gazelleOverride,
	_buildFileGenerationAttr: _gazelleOverride,
	_patchArgsAttr:           _moduleOverride,
	"patches":                _moduleOverride,
	"build_extra_args":       _gazelleOverride,
	"urls":                   _archiveOverride,
	"strip_prefix":           _archiveOverride,
	"sha256":                 _archiveOverride,
}

var _attrOverrideKeys = map[string]string{
	_buildDirectivesAttr: _directivesAttr,
}

type overrideSet map[string]*rule.Rule

type mainArgs struct {
	macroPath       string
	workspace       string
	defName         string
	outputFile      string
	gazelleRepoName string
}

func main() {
	a, err := parseArgs(os.Stderr, os.Args[1:])
	if err != nil && !errors.Is(err, flag.ErrHelp) {
		log.Fatalf("%+v", err)
	}
	if err := run(*a, os.Stderr); err != nil {
		log.Fatal(err)
	}
}

func parseArgs(stderr io.Writer, osArgs []string) (*mainArgs, error) {
	a := &mainArgs{}
	flag := flag.NewFlagSet(_name, flag.ContinueOnError)
	flag.SetOutput(stderr)
	flag.Usage = func() {
		fmt.Fprintf(flag.Output(), _usage)
		flag.PrintDefaults()
	}
	flag.StringVar(&a.macroPath, "macro", "", "path to the macro file")
	flag.StringVar(&a.workspace, "workspace", "", "path to workspace")
	flag.StringVar(&a.defName, "def_name", "", "name of the macro definition")
	flag.StringVar(&a.outputFile, "output", "", "path to the output file")
	flag.StringVar(&a.gazelleRepoName, "gazelle_repo_name", "@bazel_gazelle", "name of the gazelle repo to load go_deps, (default: @bazel_gazelle)")
	flag.Parse(osArgs)

	if a.macroPath != "" && a.workspace != "" {
		return nil, fmt.Errorf("only one of -macro or -workspace can be specified")
	}
	if a.macroPath == "" && a.workspace == "" {
		return nil, fmt.Errorf("missing required flag: -macro or -workspace")
	}
	if a.macroPath != "" && a.defName == "" {
		return nil, fmt.Errorf("missing required flag: -def_name when -macro is specified")
	}
	if a.outputFile == "" {
		return nil, fmt.Errorf("missing required flag: -output")
	}
	return a, nil
}

func run(a mainArgs, stderr io.Writer) error {
	var w *rule.File
	var err error
	if a.macroPath != "" {
		w, err = rule.LoadMacroFile(a.macroPath, "", a.defName)
		if err != nil {
			return err
		}
	} else {
		w, err = rule.LoadWorkspaceFile(a.workspace, "")
		if err != nil {
			return err
		}
	}
	repos, _, err := repo.ListRepositories(w)
	if err != nil {
		return err
	}
	sort.Slice(repos, func(i, j int) bool {
		return repos[i].Name() < repos[j].Name()
	})

	var outputOverrides []*rule.Rule

	// Iterate over all repositories and convert them to override rules
	// The repos are ordered by "name", and the sets are sorted, so the output
	// will be deterministic.
	for _, r := range repos {
		if r.Kind() == "go_repository" {
			repoOverrides := goRepositoryToOverrideSet(r)
			outputOverrides = append(outputOverrides, setToOverridesSlice(repoOverrides)...)
		}
	}

	if len(outputOverrides) == 0 {
		fmt.Fprintln(stderr, "no overrides are needed for these repos!")
		return nil
	}

	f, err := rule.LoadData(a.outputFile, "", []byte(fmt.Sprintf(_goDepsf, a.gazelleRepoName)))
	if err != nil {
		return err
	}
	for _, o := range outputOverrides {
		o.Insert(f)
	}

	if err := f.Save(a.outputFile); err != nil {
		return fmt.Errorf("error saving file: %w", err)
	}

	return nil
}

func goRepositoryToOverrideSet(r *rule.Rule) overrideSet {
	// each repo has its own override set, and can't have multiple
	// duplicate overrides. This set is created to be populated and read
	set := make(overrideSet)
	importPath := r.AttrString("importpath")

	// Load the attribute keys from the rule.
	attrs := r.AttrKeys()
	for _, attr := range attrs {
		if _, ok := _mapAttrToOverride[attr]; !ok {
			continue
		}

		attrValue := r.Attr(attr)
		if attrValue == nil || attr == _buildFileProtoModeAttr {
			continue
		}

		kind := _mapAttrToOverride[attr]
		override := rule.NewRule(kind, "")
		if o, ok := set[kind]; ok {
			override = o
		}
		override.SetAttr("path", importPath)
		val := r.Attr(attr)

		// Special case for certain renamed attributes like "build_directives"
		// attribute to convert to "directives" attribute.
		if k, ok := _attrOverrideKeys[attr]; ok {
			attr = k
		}

		if def, ok := _defaultValues[attr]; def == r.AttrString(attr) && ok {
			continue
		}

		if val != nil {
			switch v := val.(type) {
			case *build.StringExpr:
				override.SetAttr(attr, v)
			case *build.ListExpr:
				// Special case for "patch_args" attribute to convert to
				// "patch_strip" attribute.
				if attr == _patchArgsAttr {
					setPatchArgs(r.AttrStrings(_patchArgsAttr), override)
				} else {
					override.SetAttr(attr, v)
				}
			}
		}

		set[kind] = override
	}

	// Since "build_file_proto_mode" is added to the "directives", we need
	// to run it after the fact to make sure that "directives" is set.
	applyBuildFileProtoMode(r, set)
	return set
}

func applyBuildFileProtoMode(r *rule.Rule, set overrideSet) {
	if r.Attr(_buildFileProtoModeAttr) == nil {
		return
	}

	if def, ok := _defaultValues[_buildFileProtoModeAttr]; def == r.AttrString(_buildFileProtoModeAttr) && ok {
		return
	}

	directive := "gazelle:proto " + r.AttrString(_buildFileProtoModeAttr)
	kind := _gazelleOverride
	override := rule.NewRule(kind, "")
	if o, ok := set[kind]; ok {
		override = o
	}
	directives := override.AttrStrings(_directivesAttr)
	directives = append(directives, directive)
	override.SetAttr(_directivesAttr, directives)
	set[kind] = override
}

func setPatchArgs(patchArgs []string, override *rule.Rule) {
	for _, arg := range patchArgs {
		if !strings.HasPrefix(arg, "-p") {
			continue
		}
		numStr := strings.TrimPrefix(arg, "-p")
		if num, err := strconv.Atoi(numStr); err == nil {
			override.SetAttr("patch_strip", num)
			return
		}
	}
}

func setToOverridesSlice(set overrideSet) []*rule.Rule {
	// Check if both archive and module overrides exist
	if archiveOverride, archiveExists := set[_archiveOverride]; archiveExists {
		if moduleOverride, moduleExists := set[_moduleOverride]; moduleExists {
			// Merge attributes from module override into archive override
			mergeAttributes(moduleOverride, archiveOverride)
			// Remove the module override as its attributes are now merged
			delete(set, _moduleOverride)
		}
	}

	// Create a sorted slice of the remaining override keys
	var keys []string
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Create a slice of overrides based on the sorted keys
	var overrides []*rule.Rule
	for _, k := range keys {
		overrides = append(overrides, set[k])
	}
	return overrides
}

func mergeAttributes(source, destination *rule.Rule) {
	for _, attr := range source.AttrKeys() {
		if val := source.Attr(attr); val != nil {
			destination.SetAttr(attr, val)
		}
	}
}
