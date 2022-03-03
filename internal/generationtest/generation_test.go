package generationtest

import (
	"io/ioutil"
	"path"
	"strings"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/testtools"
	"github.com/bazelbuild/rules_go/go/tools/bazel"
	yaml "gopkg.in/yaml.v2"
)

// TestFullGeneration runs the gazelle binary on a few example
// workspaces and confirms that the generated BUILD files match expectation.
func TestFullGeneration(t *testing.T) {
	tests := map[string][]bazel.RunfileEntry{}
	runfiles, err := bazel.ListRunfiles()
	if err != nil {
		t.Fatalf("bazel.ListRunfiles() error: %v", err)
	}
	var manifest *manifestYAML
	for _, f := range runfiles {
		if path.Base(f.ShortPath) == "generation_test_manifest.yaml" {
			manifest = new(manifestYAML)
			content, err := ioutil.ReadFile(f.Path)
			if err != nil {
				t.Errorf("ioutil.ReadFile(%q) error: %v", f.Path, err)
			}

			if err := yaml.Unmarshal(content, manifest); err != nil {
				t.Fatal(err)
			}
			break
		}
	}
	testDataDir := manifest.TestDataDir
	for _, f := range runfiles {
		if strings.HasPrefix(f.ShortPath, testDataDir) {
			relativePath := strings.TrimPrefix(f.ShortPath, testDataDir)
			// Given a path like /my_test_case/my_file_being_tested,
			// split it into "", "my_test_case", "my_file_beign_tested".
			// If this split is less than 3, that means we don't have a test case folder.
			// For example, if we received /README.md for the test case folder, this would just create
			// "", "README.md", so the number of parts is only 2.
			parts := strings.SplitN(relativePath, "/", 3)
			if len(parts) < 3 {
				// This file is not a part of a testcase since it must be in a dir that
				// is the test case and then have a path inside of that.
				continue
			}

			tests[parts[1]] = append(tests[parts[1]], f)
		}
	}
	if len(tests) == 0 {
		t.Fatal("no tests found")
	}

	testArgs := testtools.NewTestGazelleGenerationArgs()
	gazelleBinaryDir := manifest.GazelleBinaryDir
	gazelleBinaryName := manifest.GazelleBinaryName

	for testName, files := range tests {
		testArgs.Name = testName
		testArgs.TestDataPath = testDataDir
		testArgs.GazelleBinaryDir = gazelleBinaryDir
		testArgs.GazelleBinaryName = gazelleBinaryName
		testtools.TestGazelleGenerationOnPath(t, testArgs, files)
	}
}

type manifestYAML struct {
	GazelleBinaryName string `yaml:"gazelle_binary_name"`
	GazelleBinaryDir  string `yaml:"gazelle_binary_dir"`
	TestDataDir       string `yaml:"test_data_dir"`
}
