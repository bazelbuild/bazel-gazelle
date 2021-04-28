package main_test

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
)

var fakeBinRunfilesPath = "internal/runnerscripttest/fake_gazelle_binary_/fake_gazelle_binary"

// this test fails on windows without the fix
func TestNoManifest(t *testing.T) {
	fakeGazellePath := findGazelle(t)

	// run at the root of the runfiles tree
	runGazelle(t, fakeGazellePath, "..")
}

// this test fails on all platforms because it forces the manifest evaluation
func TestManifest(t *testing.T) {
	fakeGazellePath := findGazelle(t)

	exe := ""
	if runtime.GOOS == "windows" {
		exe = ".exe"
		fakeBinRunfilesPath += exe
	}

	tmpDir, err := bazel.NewTmpDir(t.Name())

	if err != nil {
		t.Fatalf("failed to setup tmp dir: %v", err)
	}

	copyToTmp := func(src, dest string) string {
		dest = filepath.Join(tmpDir, dest)

		srcContents, err := os.ReadFile(src)
		if err != nil {
			t.Fatalf("failed to read fake gazelle: %v", err)
		}

		err = os.WriteFile(dest, srcContents, 0777)
		if err != nil {
			t.Fatalf("failed to read fake gazelle: %v", err)
		}
		return dest
	}

	copiedRunner := copyToTmp(fakeGazellePath, "fake_gazelle")
	if exe != "" {
		copiedRunner = copyToTmp(fakeGazellePath+exe, "fake_gazelle"+exe)
	}

	entries, err := bazel.ListRunfiles()
	if err != nil {
		t.Fatalf("failed to list runfiles %v", err)
	}
	var lines []string
	for _, e := range entries {
		lines = append(lines, fmt.Sprintf("%s %s", path.Join(e.Workspace, e.ShortPath), e.Path))
	}

	manifest := strings.Join(lines, "\n") + "\n"

	err = os.WriteFile(path.Join(tmpDir, "MANIFEST"), []byte(manifest), 0666)
	if err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	runGazelle(t, copiedRunner, tmpDir)
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

	for _, l := range lines {
		log.Println(l)
	}

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
