package visibility

import (
	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/language"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

type visibilityExtension struct{}

// NewLanguage constructs a new language.Language modifying visibility.
func NewLanguage() language.Language {
	return &visibilityExtension{}
}

// Kinds instructs gazelle to match any 'package' rule as BUILD files can only have one.
func (*visibilityExtension) Kinds() map[string]rule.KindInfo {
	return map[string]rule.KindInfo{
		"package": {
			MatchAny: true,
			MergeableAttrs: map[string]bool{
				"default_visibility": true,
			},
		},
	}
}

func (*visibilityExtension) Loads() []rule.LoadInfo {
	panic("ApparentLoads should be called instead")
}

// ApparentLoads noops because there are no imports to add
func (*visibilityExtension) ApparentLoads(func(string) string) []rule.LoadInfo {
	return nil
}

// GenerateRules does the hard work of setting the default_visibility if a config exists.
func (*visibilityExtension) GenerateRules(args language.GenerateArgs) language.GenerateResult {
	res := language.GenerateResult{}
	cfg := getVisConfig(args.Config)

	if len(cfg.visibilityTargets) == 0 {
		return res
	}

	if args.File == nil {
		// No need to create a visibility if we're not in a visible directory.
		return res
	}

	r := rule.NewRule("package", "")
	r.SetAttr("default_visibility", cfg.visibilityTargets)

	res.Gen = append(res.Gen, r)
	// we have to add a nil to Imports because there is length-matching validation with Gen.
	res.Imports = append(res.Imports, nil)
	return res
}

// Fix noop because there is nothing out there to fix yet
func (*visibilityExtension) Fix(c *config.Config, f *rule.File) {}
