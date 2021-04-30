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
var gazelleWorkspace = "bazel_gazelle"

func setupExe() {
	if runtime.GOOS == "windows" {
		exe = ".exe"
	}
}

func TestInternalGazelleNoManifest(t *testing.T) {
	setupExe()
	hostWorkspace := gazelleWorkspace
	tmpDir := setupRunfiles(t, hostWorkspace)

	destPath := copyGazelle(t, tmpDir, hostWorkspace, false)

	// run inside the host workspace directory in the runfiles tree
	runGazelle(t, destPath, filepath.Join(tmpDir, hostWorkspace))
}

func TestExternalGazelleNoManifest(t *testing.T) {
	setupExe()
	hostWorkspace := "foo"
	tmpDir := setupRunfiles(t, hostWorkspace)

	destPath := copyGazelle(t, tmpDir, hostWorkspace, true)

	// run inside the host workspace directory in the runfiles tree
	runGazelle(t, destPath, filepath.Join(tmpDir, hostWorkspace))
}

func TestInternalGazelleManifest(t *testing.T) {
	setupExe()

	testManifest(t, "bazel_gazelle", false)
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

func setupRunfiles(t *testing.T, hostWorkspace string) string {
	manifestMap := makeManifestMap(hostWorkspace)
	tmpDir, err := bazel.NewTmpDir(t.Name())
	if err != nil {
		t.Fatalf("failed to setup tmp dir: %v", err)
	}

	for in, extList := range manifestMap {
		for _, ext := range extList {
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
	}
	return tmpDir
}

func testManifest(t *testing.T, hostWorkspace string, transform bool) {
	manifestMap := makeManifestMap(hostWorkspace)
	tmpDir, err := bazel.NewTmpDir(t.Name())
	if err != nil {
		t.Fatalf("failed to setup tmp dir: %v", err)
	}

	var manifestLines []string
	i := 0
	for in, extList := range manifestMap {
		for _, ext := range extList {
			realPath, err := bazel.Runfile(in)
			if err != nil {
				t.Fatalf("failed to get %s from runfiles: %v", in, err)
			}
			manifestLines = append(manifestLines, ext+" "+realPath)
			i++
		}
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

func makeManifestMap(hostWorkspace string) map[string][]string {
	base := map[string][]string{
		// lookup in this runfiles 						--> write to simulated runfiles
		"bin/go" + exe: {"go_sdk/bin/go" + exe, path.Join(hostWorkspace, "external/go_sdk/bin/go"+exe)},
	}

	gazelleLocations := map[string][]string{
		"internal/runnerscripttest/fake_gazelle":                                   {""},
		"internal/runnerscripttest/fake_gazelle_binary_/fake_gazelle_binary" + exe: {"_binary_/fake_gazelle_binary" + exe},
	}

	if exe != "" {
		// on windows bazel makes a bash launcher exe to launch the fake_gazelle bash script.
		gazelleLocations["internal/runnerscripttest/fake_gazelle"+exe] = []string{exe}
	}

	prefixes := []string{gazelleWorkspace}
	if gazelleWorkspace != hostWorkspace {
		prefixes = append(prefixes, path.Join(hostWorkspace, "external", gazelleWorkspace))
	}
	for _, prefix := range prefixes {
		for in, extList := range gazelleLocations {
			for _, ext := range extList {
				base[in] = append(base[in], path.Join(prefix, "internal/runnerscripttest/fake_gazelle")+ext)
			}
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
