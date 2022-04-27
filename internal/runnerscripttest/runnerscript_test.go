package main_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
)

var exe = ""
var thisWorkspace = "bazel_gazelle"

func setupExe() {
	if runtime.GOOS == "windows" {
		exe = ".exe"
	}
}

// these tests construct four scenarios of runfiles and asserts that the gazelle runner script
// properly locates the correct runfiles given those scenarios
// The tests construct a simulated MANIFEST file, and a simulated runfiles tree for a gazelle binary
// built 1) in the host workspace i.e. `gazelle = //:gazelle`
// and 2) in an external workspace i.e. `gazelle = @bazel_gazelle//cmd/gazelle`
// these tests perform the same actions on both windows and non-windows (with the exception of some
// .exe suffixing) via a manually constructed runfiles tree of actual folders, not symlinks.

func TestInternalGazelleNoManifest(t *testing.T) {
	setupExe()
	hostWorkspace := thisWorkspace
	tmpDir := setupRunfiles(t, thisWorkspace)

	destPath := copyGazelle(t, tmpDir, hostWorkspace, false)

	// run inside the host workspace directory in the runfiles tree
	runGazelle(t, destPath, filepath.Join(tmpDir, hostWorkspace))
}

func TestExternalGazelleNoManifest(t *testing.T) {
	setupExe()
	hostWorkspace := "foo"
	tmpDir := setupRunfiles(t, thisWorkspace)

	destPath := copyGazelle(t, tmpDir, hostWorkspace, true)

	// run inside the host workspace directory in the runfiles tree
	startDir := filepath.Join(tmpDir, hostWorkspace)
	err := os.Mkdir(startDir, 0755)
	if err != nil {
		t.Fatalf("failed to create starting directroy: %v", err)
	}
	runGazelle(t, destPath, startDir)
}

func TestInternalGazelleManifest(t *testing.T) {
	setupExe()

	testManifest(t, thisWorkspace, false)
}

func TestExternalGazelleManifest(t *testing.T) {
	setupExe()

	testManifest(t, "foo", true)
}

func copyGazelle(t *testing.T, tmpDir, hostWorkspace string, transform bool) string {
	// copy the gazelle runner script in all cases because on windows we'll be executing a bash launcher exe that
	// looks for the script in the current directory
	fakeGazellePath := findGazelle(t)
	contents, err := os.ReadFile(fakeGazellePath)
	if err != nil {
		t.Fatalf("failed to read fake gazelle script: %v", err)
	}
	if transform {
		contents = []byte(strings.Replace(string(contents),
			"GAZELLE_SHORT_PATH='internal", "GAZELLE_SHORT_PATH='../bazel_gazelle/internal",
			1))
		contents = []byte(strings.Replace(string(contents),
			"GAZELLE_LABEL='//internal", "GAZELLE_LABEL='@bazel_gazelle//internal",
			1))
		contents = []byte(strings.Replace(string(contents),
			"WORKSPACE_NAME='bazel_gazelle'", fmt.Sprintf("WORKSPACE_NAME='%s'", hostWorkspace),
			1))
	}

	destPath := filepath.Join(tmpDir, filepath.Base(fakeGazellePath))
	err = os.WriteFile(destPath, contents, 0777)
	if err != nil {
		t.Fatalf("failed to write transformed gazelle script %v", err)
	}
	if exe != "" {
		copyTo(t, fakeGazellePath+exe, destPath+exe)
	}

	return destPath
}

func setupRunfiles(t *testing.T, gazelleWorkspace string) string {
	specs := getRunfileSpecs(gazelleWorkspace)
	tmpDir, err := bazel.NewTmpDir(t.Name())
	if err != nil {
		t.Fatalf("failed to setup tmp dir: %v", err)
	}

	for _, spec := range specs {
		realPath, err := bazel.Runfile(spec.shortPath)
		if err != nil {
			t.Fatalf("failed to get %s from runfiles: %v", spec.shortPath, err)
		}
		destPath := filepath.Join(tmpDir, spec.fullPath)
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			t.Fatalf("failed to create %s: %v", destPath, err)
		}
		copyTo(t, realPath, destPath)
	}
	return tmpDir
}

func testManifest(t *testing.T, hostWorkspace string, transform bool) {
	specs := getRunfileSpecs(thisWorkspace)
	tmpDir, err := bazel.NewTmpDir(t.Name())
	if err != nil {
		t.Fatalf("failed to setup tmp dir: %v", err)
	}

	manifestLines := make([]string, len(specs))
	for i, spec := range specs {
		realPath, err := bazel.Runfile(spec.shortPath)
		if err != nil {
			t.Fatalf("failed to get %s from runfiles: %v", spec.shortPath, err)
		}

		line := spec.fullPath + " " + realPath
		t.Logf(line)
		manifestLines[i] = line
	}

	manifest := strings.Join(manifestLines, "\n") + "\n"
	err = os.WriteFile(path.Join(tmpDir, "MANIFEST"), []byte(manifest), 0666)
	if err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	var gazellePath string
	if transform {
		gazellePath = copyGazelle(t, tmpDir, hostWorkspace, transform)
	} else {
		gazellePath = findGazelle(t) + exe
	}

	runGazelle(t, gazellePath, tmpDir)
}

type runfileSpec = struct {
	shortPath    string
	workspace    string
	fullPath     string
	absolutePath string
}

func spec(workspace, shortPath, ext string) *runfileSpec {
	return &runfileSpec{
		shortPath: shortPath + ext,
		workspace: workspace,
		fullPath:  path.Join(workspace, shortPath+ext),
	}
}

func getRunfileSpecs(hostWorkspace string) []*runfileSpec {
	specs := []*runfileSpec{
		spec("go_sdk", "bin/go", exe),
		spec(hostWorkspace, "internal/runnerscripttest/fake_gazelle", ""),
		spec(hostWorkspace, "internal/runnerscripttest/fake_gazelle_binary_/fake_gazelle_binary", exe),
	}

	if exe != "" {
		// on windows bazel makes a bash launcher exe to launch the fake_gazelle bash script.
		specs = append(specs, spec(hostWorkspace, "internal/runnerscripttest/fake_gazelle", exe))
	}
	return specs
}

func findGazelle(t *testing.T) string {
	fakeGazellePath, err := bazel.Runfile("internal/runnerscripttest/fake_gazelle")
	if err != nil {
		t.Fatalf("failed to locate fake gazelle: %v", err)
	}
	return fakeGazellePath
}

func runGazelle(t *testing.T, fakeGazellePath, dir string) {

	cmd := exec.Command(fakeGazellePath)
	if dir != "" {
		cmd.Dir = dir
	}

	cmd.Env = append(os.Environ(), "BUILD_WORKSPACE_DIRECTORY=foo")
	cmd.Stdin = nil
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	rawOut := stdout.String()
	rawErr := stderr.String()

	if err != nil {
		t.Fatalf("%v\nstdout: %s\nstderr: %s", err, rawOut, rawErr)
	}

	lines := strings.Split(rawOut, "\n")

	if len(lines) != 2 {
		t.Errorf("Unexpected output lines %d", len(lines))
	}
	if lines[0] != "Hello from fake gazelle!" {
		t.Errorf("Unexpected greeting: %s", lines[0])
	}

	if rawErr != "" {
		t.Errorf("Unexpected error: %s", rawErr)
	}
}

func copyTo(t *testing.T, src, dest string) string {
	srcContents, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	err = os.WriteFile(dest, srcContents, 0777)
	if err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	return dest
}
