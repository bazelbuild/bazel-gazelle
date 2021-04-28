package main_test

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
)

var fakeBinRunfilesPath = "internal/runnerscripttest/fake_gazelle_binary_/fake_gazelle_binary"

func TestNoManifest(t *testing.T) {
	fakeGazellePath := findGazelle(t)

	runGazelle(t, fakeGazellePath, "")
}

func TestManifest(t *testing.T) {
	fakeGazellePath := findGazelle(t)

	exe := ""
	if runtime.GOOS == "windows" {
		exe = ".exe"
		fakeBinRunfilesPath += exe
	}

	fakeBinPath, err := bazel.Runfile(fakeBinRunfilesPath)
	if err != nil {
		t.Fatalf("failed to find fake binary: %v", err)
	}

	tmpDir, err := bazel.NewTmpDir(t.Name())

	if err != nil {
		t.Fatalf("failed to setup tmp dir: %v", err)
	}

	fakeGazelleContents, err := os.ReadFile(fakeGazellePath)
	if err != nil {
		t.Fatalf("failed to read fake gazelle: %v", err)
	}

	copiedRunner := path.Join(tmpDir, "fake_gazelle")
	err = os.WriteFile(copiedRunner, fakeGazelleContents, 0755)
	if err != nil {
		t.Fatalf("failed to read fake gazelle: %v", err)
	}

	manifest := strings.Join([]string{
		// required
		fmt.Sprintf("go_sdk/bin/go%s foo/bar", exe),
		// subject under test
		fmt.Sprintf("bazel_gazelle/%s %s", fakeBinRunfilesPath, fakeBinPath),
	}, "\n")

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
	cmd.Env = []string{"BUILD_WORKSPACE_DIRECTORY=foo"}
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
