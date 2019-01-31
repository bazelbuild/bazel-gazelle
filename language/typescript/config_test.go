package typescript

import (
	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/language"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/testtools"
	"github.com/bazelbuild/bazel-gazelle/walk"
	"strings"
	"testing"
)

func testConfig(t *testing.T, args ...string) (*config.Config, []language.Language, []config.Configurer) {
	// Add a -repo_root argument if none is present. Without this,
	// config.CommonConfigurer will try to auto-detect a WORKSPACE file,
	// which will fail.
	haveRoot := false
	for _, arg := range args {
		if strings.HasPrefix(arg, "-repo_root") {
			haveRoot = true
			break
		}
	}
	if !haveRoot {
		args = append(args, "-repo_root=.")
	}

	cexts := []config.Configurer{
		&config.CommonConfigurer{},
		&walk.Configurer{},
		&resolve.Configurer{},
	}
	// TODO: support proto as well
	// langs := []language.Language{proto.NewLanguage(), NewLanguage()}
	langs := []language.Language{NewLanguage()}
	c := testtools.NewTestConfig(t, cexts, langs, args)
	for _, lang := range langs {
		cexts = append(cexts, lang)
	}
	return c, langs, cexts
}
