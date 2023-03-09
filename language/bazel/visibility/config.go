package visibility

import (
	"flag"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

const (
	_directiveName = "default_visibility"
)

type visConfig struct {
	visibilityTargets []string
}

// getVisConfig directly returns the internal configuration struct rather
// than a pointer because we explicitly want pass-by-value symantics so
// configurations down a directory tree don't accidentially update upstream.
func getVisConfig(c *config.Config) visConfig {
	cfg := c.Exts[_extName]
	if cfg == nil {
		return visConfig{}
	}
	return cfg.(visConfig)
}

// RegisterFlags noops because we only parameterize behavior with a directive.
func (*visibilityExtension) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {}

// CheckFlags noops because no flags are referenced.
func (*visibilityExtension) CheckFlags(fs *flag.FlagSet, c *config.Config) error {
	return nil
}

// KnownDirectives returns the only directive this extension operates on.
func (*visibilityExtension) KnownDirectives() []string {
	return []string{_directiveName}
}

// Configure identifies the visibility targets from the directive value, if it exists.
//
// To set multiple visibility targets, either multiple directives can be used, or a
// list can be provided with comma-separated values.
func (*visibilityExtension) Configure(c *config.Config, _ string, f *rule.File) {
	cfg := getVisConfig(c)
	if f == nil {
		return
	}

	var newVisTargets []string
	for _, d := range f.Directives {
		switch d.Key {
		case _directiveName:
			for _, target := range strings.Split(d.Value, ",") {
				newVisTargets = append(newVisTargets, target)
			}
		}
	}

	// if visibility targets were specified, overwrite the config
	if len(newVisTargets) != 0 {
		cfg.visibilityTargets = newVisTargets
	}

	c.Exts[_extName] = cfg
}

// /Configurator embed
