package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/repo"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/bazelbuild/rules_go/go/runfiles"
)

// Injected via x_defs.
var goToolRlocationPath string
var repoConfigRlocationPath string

func main() {
	wd := os.Getenv("BUILD_WORKSPACE_DIRECTORY")
	if wd == "" {
		log.Fatal("BUILD_WORKSPACE_DIRECTORY is not set, please run this tool with bazel run")
	}

	if err := run(wd); err != nil {
		log.Fatal(err)
	}
}

func run(wd string) error {
	repoToImportPath, err := getRepoToImportPathMap()
	if err != nil {
		return err
	}

	directDeps, err := getDirectDeps(wd, repoToImportPath)
	if err != nil {
		return err
	}

	fakeImportsFile, err := emitFakeImports(wd, directDeps)
	if err != nil {
		return err
	}
	if fakeImportsFile != "" {
		defer os.Remove(fakeImportsFile)
	}

	err = runGoModTidy(wd)
	if err != nil {
		return err
	}

	return nil
}

func getRepoToImportPathMap() (map[string]string, error) {
	repoConfig, err := runfiles.Rlocation(repoConfigRlocationPath)
	if err != nil {
		return nil, err
	}

	f, err := rule.LoadWorkspaceFile(repoConfig, "")
	if err != nil {
		return nil, err
	}
	repos, _, err := repo.ListRepositories(f)
	if err != nil {
		return nil, err
	}

	repoToPath := make(map[string]string)
	for _, r := range repos {
		if r.Kind() == "go_repository" {
			repoToPath[r.Name()] = r.AttrString("importpath")
		}
	}
	return repoToPath, nil
}

func getDirectDeps(wd string, repoToImportPath map[string]string) ([]string, error) {
	queryCmd := exec.Command("bazel", "query",
		"--output=package", "--order_output=no",
		"deps(kind(go_.*, //...), 1)")
	queryCmd.Dir = wd

	stdout, err := queryCmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	_, _ = fmt.Fprintf(os.Stderr, "Running %s\n", strings.Join(queryCmd.Args, " "))
	if err := queryCmd.Start(); err != nil {
		return nil, err
	}

	repos := make(map[string]struct{})
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		pkg := scanner.Text()
		repoEnd := strings.Index(pkg, "//")
		if !strings.HasPrefix(pkg, "@") || repoEnd == -1 {
			continue
		}
		repos[pkg[1:repoEnd]] = struct{}{}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if err := queryCmd.Wait(); err != nil {
		return nil, err
	}

	var paths []string
	for repoName := range repos {
		importPath := repoToImportPath[repoName]
		if importPath != "" {
			paths = append(paths, importPath)
		}
	}
	sort.Strings(paths)
	return paths, nil
}

func emitFakeImports(wd string, directDeps []string) (string, error) {
	if len(directDeps) == 0 {
		return "", nil
	}

	f, err := os.CreateTemp(wd, "imports_for_tidy_*.go")
	if err != nil {
		return "", err
	}
	defer f.Close()

	_, err = fmt.Fprintf(f, "package main\nimport (\n\t_ \"%s\"\n)\n", strings.Join(directDeps, "\"\n\t_ \""))
	if err != nil {
		return "", err
	}

	return f.Name(), nil
}

func runGoModTidy(wd string) error {
	goBin, err := runfiles.Rlocation(goToolRlocationPath)
	if err != nil {
		return err
	}

	runfilesEnv, err := runfiles.Env()
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(os.Stderr, "Running go mod tidy %s\n", strings.Join(os.Args[1:], " "))
	cmd := exec.Command(goBin, "mod", "tidy")
	cmd.Args = append(cmd.Args, os.Args[1:]...)
	cmd.Env = append(os.Environ(), runfilesEnv...)
	cmd.Dir = wd
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
