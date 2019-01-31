package typescript

import (
	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/repo"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

func (_ *typescriptLang) Imports(c *config.Config, r *rule.Rule, f *rule.File) []resolve.ImportSpec {
	panic("implement me")
}

func (_ *typescriptLang) Embeds(r *rule.Rule, from label.Label) []label.Label {
	panic("implement me")
}

func (_ *typescriptLang) Resolve(c *config.Config, ix *resolve.RuleIndex, rc *repo.RemoteCache, r *rule.Rule, imports interface{}, from label.Label) {
	panic("implement me")
}
