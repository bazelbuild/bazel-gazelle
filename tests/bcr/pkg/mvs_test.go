package pkg

import (
	"testing"

	"github.com/DataDog/sketches-go/ddsketch"
	"github.com/bmatcuk/doublestar/v4"
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
