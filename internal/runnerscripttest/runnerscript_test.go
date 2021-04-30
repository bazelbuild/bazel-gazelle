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

func TestInternalGazelleNoManifest(t *testing.T) {
	hostWorkspace := "bazel_gazelle"
	tmpDir := setupRunfiles(t, hostWorkspace, hostWorkspace)

	fakeGazellePath := findGazelle(t)
	// run inside the host workspace directory in the runfiles tree
	runGazelle(t, fakeGazellePath, filepath.Join(tmpDir, hostWorkspace))
}

func TestExternalGazelleNoManifest(t *testing.T) {
	hostWorkspace := "foo"
	gazelleWorkspace := "bazel_gazelle"
	tmpDir := setupRunfiles(t, hostWorkspace, gazelleWorkspace)

	fakeGazellePath := findGazelle(t)
	contents, err := os.ReadFile(fakeGazellePath)
	if err != nil {
		t.Fatalf("failed to read fake gazelle script: %v", err)
	}
	contents = []byte(strings.Replace(string(contents),
		"GAZELLE_SHORT_PATH='internal",
		fmt.Sprintf("GAZELLE_SHORT_PATH='../%s/internal", gazelleWorkspace),
		1))

	destPath := filepath.Join(tmpDir, filepath.Base(fakeGazellePath))
	err = os.WriteFile(destPath, contents, 0777)
	if err != nil {
		t.Fatalf("failed to write transformed gazelle script %v", err)
	}

	// run inside the host workspace directory in the runfiles tree
	runGazelle(t, destPath, filepath.Join(tmpDir, hostWorkspace))
}

func TestInternalGazelleManifest(t *testing.T) {
	testManifest(t, "bazel_gazelle", "bazel_gazelle")
}
func TestExternalGazelleManifest(t *testing.T) {

	testManifest(t, "foo", "bazel_gazelle")
}

func setupRunfiles(t *testing.T, hostWorkspace, gazelleWorkspace string) string {
	manifestMap := makeManifestMap(hostWorkspace, gazelleWorkspace)
	tmpDir, err := bazel.NewTmpDir(t.Name())
	if err != nil {
		t.Fatalf("failed to setup tmp dir: %v", err)
	}

	for in, ext := range manifestMap {
		realPath, err := bazel.Runfile(in)
		if err != nil {
			t.Fatalf("failed to get %s from runfiles: %v", in, err)
		}
		destPath := filepath.Join(tmpDir, ext)
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			t.Fatalf("failed to create %s: %v", destPath, err)
		}
		copyTo(t, realPath, destPath)
	}
	return tmpDir
}

func testManifest(t *testing.T, hostWorkspace, gazelleWorkspace string) {
	manifestMap := makeManifestMap(hostWorkspace, gazelleWorkspace)
	tmpDir, err := bazel.NewTmpDir(t.Name())
	if err != nil {
		t.Fatalf("failed to setup tmp dir: %v", err)
	}

	manifestLines := make([]string, len(manifestMap))
	i := 0
	for in, ext := range manifestMap {
		realPath, err := bazel.Runfile(in)
		if err != nil {
			t.Fatalf("failed to get %s from runfiles: %v", in, err)
		}
		manifestLines[i] = ext + " " + realPath
		i++
	}
	manifest := strings.Join(manifestLines, "\n") + "\n"
	err = os.WriteFile(path.Join(tmpDir, "MANIFEST"), []byte(manifest), 0666)
	if err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}
	fakeGazellePath := findGazelle(t)
	runGazelle(t, fakeGazellePath, tmpDir)
}

func makeManifestMap(hostWorkspace, gazelleWorkspace string) map[string]string {
	base := map[string]string{
		// lookup in this runfiles 			--> write to simulated runfiles
		"external/go_sdk/bin/go": path.Join(hostWorkspace, "external/go_sdk/bin/go"),
		"bin/go":                 "go_sdk/bin/go",
	}

	exe := ""
	if runtime.GOOS == "windows" {
		exe = ".exe"
	}

	gazelleLocations := map[string]string{
		"internal/runnerscripttest/fake_gazelle":                                   "",
		"internal/runnerscripttest/fake_gazelle_binary_/fake_gazelle_binary" + exe: "_binary_/fake_gazelle_binary" + exe,
	}

	if exe != "" {
		// on windows bazel makes a bash launcher exe to launch the fake_gazelle bash script.
		gazelleLocations["internal/runnerscripttest/fake_gazelle"+exe] = exe
	}

	prefixes := []string{gazelleWorkspace}
	if gazelleWorkspace != hostWorkspace {
		//prefixes = append(prefixes, path.Join(hostWorkspace, "external", gazelleWorkspace))
	}
	for _, prefix := range prefixes {
		for in, ext := range gazelleLocations {
			base[in] = path.Join(prefix, "internal/runnerscripttest/fake_gazelle") + ext
		}
	}

	return base
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
