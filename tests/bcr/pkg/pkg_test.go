package pkg

import (
	"crypto/rand"
	"os"
	"testing"

	"github.com/DataDog/sketches-go/ddsketch"
	"github.com/bazelbuild/bazel-gazelle/tests/bcr/pkg/data"
	"github.com/bazelbuild/rules_go/go/runfiles"
	"github.com/bmatcuk/doublestar/v4"
	"github.com/cloudflare/circl/dh/x25519"
	"github.com/fmeum/dep_on_gazelle"
	"github.com/google/safetext/yamltemplate"
	"github.com/stretchr/testify/require"

	_ "github.com/envoyproxy/protoc-gen-validate/validate"
)

func TestReplace(t *testing.T) {
	// doublestar.StandardOS does NOT exist in doublestar/v4
	// See: https://pkg.go.dev/github.com/bmatcuk/doublestar#OS
	// If we are able to initialize this variable, it validates that the dependency is properly
	// being replaced with github.com/bmatcuk/doublestar@v1.3.4
	_ = doublestar.StandardOS
}

func TestPatch(t *testing.T) {
	// a patch is used to add this constant.
	require.Equal(t, "hello", require.Hello)
}

func TestBuildFileGeneration(t *testing.T) {
	// github.com/google/safetext@v0.0.0-20220905092116-b49f7bc46da2 requires overwriting the BUILD
	// files it provides as well as directives.
	yamltemplate.HTMLEscapeString("<b>foo</b>")
}

func TestGeneratedFilesPreferredOverProtos(t *testing.T) {
	_, _ = ddsketch.NewDefaultDDSketch(0.01)
}

func TestPlatformDependentDep(t *testing.T) {
	PlatformDependentFunction()
}

func TestNoGoRepositoryForRulesGoAndGazelle(t *testing.T) {
	path, err := runfiles.Rlocation(data.RepoConfigRlocationPath)
	require.NoError(t, err)
	config, err := os.ReadFile(path)
	require.NoError(t, err)

	content := string(config)
	require.NotContains(t, content, "com_github_bazelbuild_rules_go")
	require.NotContains(t, content, "com_github_bazelbuild_bazel_gazelle")
	require.Contains(t, content, "module_name = \"rules_go\"")
	require.Contains(t, content, "module_name = \"gazelle\"")
}

func TestIndirectlyUseGazelle(t *testing.T) {
	dep_on_gazelle.MakeLabel("foo", "bar", "baz")
}

func TestBazelDepUsedAsGoDep(t *testing.T) {
	var public, secret x25519.Key
	_, err := rand.Read(secret[:])
	require.NoError(t, err)
	x25519.KeyGen(&public, &secret)
}
