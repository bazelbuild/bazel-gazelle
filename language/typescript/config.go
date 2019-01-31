package typescript

import (
	"flag"
	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// typescriptConfig contains configuration values related to Typescript rules.
type typescriptConfig struct {
}

func (tc *typescriptConfig) clone() *typescriptConfig {
	tcCopy := *tc
	return &tcCopy
}

func newTypescriptConfig() *typescriptConfig {
	tc := &typescriptConfig{}
	return tc
}

func (_ *typescriptLang) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
	tc := newTypescriptConfig()
	c.Exts[typescriptName] = tc
}

func (_ *typescriptLang) CheckFlags(fs *flag.FlagSet, c *config.Config) error {
	return nil
}

func (_ *typescriptLang) KnownDirectives() []string {
	return []string{}
}

func (_ *typescriptLang) Configure(c *config.Config, rel string, f *rule.File) {
	var tc *typescriptConfig
	if raw, ok := c.Exts[typescriptName]; !ok {
		tc = newTypescriptConfig()
	} else {
		tc = raw.(*typescriptConfig).clone()
	}
	c.Exts[typescriptName] = tc
}
