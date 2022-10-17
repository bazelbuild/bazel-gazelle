package visibility

import (
	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/repo"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

const (
	_extName = "visibility_extension"
)

// Name returns the extension name.
func (*visibilityExtension) Name() string {
	return _extName
}

// Imports noops because no imports are needed to leverage this functionality.
func (*visibilityExtension) Imports(c *config.Config, r *rule.Rule, f *rule.File) []resolve.ImportSpec {
	return nil
}

// Embeds noops because we are not a traditional rule, and cannot be embedded.
func (*visibilityExtension) Embeds(r *rule.Rule, from label.Label) []label.Label {
	return nil
}

// Resolve noops because we don't have deps=[] to resolve.
func (*visibilityExtension) Resolve(c *config.Config, ix *resolve.RuleIndex, rc *repo.RemoteCache, r *rule.Rule, imports interface{}, from label.Label) {
}
