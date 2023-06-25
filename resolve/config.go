/* Copyright 2018 The Bazel Authors. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package resolve

import (
	"flag"
	"log"
	"regexp"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// FindRuleWithOverride searches the current configuration for user-specified
// dependency resolution overrides. Overrides specified later (in configuration
// files in deeper directories, or closer to the end of the file) are
// returned first. If no override is found, label.NoLabel is returned.
func FindRuleWithOverride(c *config.Config, imp ImportSpec, lang string) (label.Label, bool) {
	rc := getResolveConfig(c)
	for i := len(rc.overrides) - 1; i >= 0; i-- {
		o := rc.overrides[i]
		if o.matches(imp, lang) {
			return o.dep, true
		}
	}
	for i := len(rc.regexpOverrides) - 1; i >= 0; i-- {
		o := rc.regexpOverrides[i]
		if o.matches(imp, lang) {
			return o.dep, true
		}
	}

	return label.NoLabel, false
}

type overrideSpec struct {
	imp  ImportSpec
	lang string
	dep  label.Label
}

func (o overrideSpec) matches(imp ImportSpec, lang string) bool {
	return imp.Lang == o.imp.Lang &&
		imp.Imp == o.imp.Imp &&
		(o.lang == "" || o.lang == lang)
}

type regexpOverrideSpec struct {
	ImpLang  string
	ImpRegex *regexp.Regexp
	lang     string
	dep      label.Label
}

func (o regexpOverrideSpec) matches(imp ImportSpec, lang string) bool {
	return imp.Lang == o.ImpLang &&
		o.ImpRegex.MatchString(imp.Imp) &&
		(o.lang == "" || o.lang == lang)
}

type resolveConfig struct {
	overrides       []overrideSpec
	regexpOverrides []regexpOverrideSpec
}

const resolveName = "_resolve"

func getResolveConfig(c *config.Config) *resolveConfig {
	return c.Exts[resolveName].(*resolveConfig)
}

type Configurer struct{}

func (*Configurer) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
	c.Exts[resolveName] = &resolveConfig{}
}

func (*Configurer) CheckFlags(fs *flag.FlagSet, c *config.Config) error { return nil }

func (*Configurer) KnownDirectives() []string {
	return []string{"resolve", "resolve_regexp"}
}

func (*Configurer) Configure(c *config.Config, rel string, f *rule.File) {
	rc := getResolveConfig(c)
	rcCopy := &resolveConfig{
		overrides:       rc.overrides[:len(rc.overrides):len(rc.overrides)],
		regexpOverrides: rc.regexpOverrides[:len(rc.regexpOverrides):len(rc.regexpOverrides)],
	}

	if f != nil {
		for _, d := range f.Directives {
			if d.Key == "resolve" {
				parts := strings.Fields(d.Value)
				o := overrideSpec{}
				var lbl string
				if len(parts) == 3 {
					o.imp.Lang = parts[0]
					o.imp.Imp = parts[1]
					lbl = parts[2]
				} else if len(parts) == 4 {
					o.imp.Lang = parts[0]
					o.lang = parts[1]
					o.imp.Imp = parts[2]
					lbl = parts[3]
				} else {
					log.Printf("could not parse directive: %s\n\texpected gazelle:resolve source-language [import-language] import-string label", d.Value)
					continue
				}
				var err error
				o.dep, err = label.Parse(lbl)
				if err != nil {
					log.Printf("gazelle:resolve %s: %v", d.Value, err)
					continue
				}
				o.dep = o.dep.Abs("", rel)
				rcCopy.overrides = append(rcCopy.overrides, o)
			} else if d.Key == "resolve_regexp" {
				parts := strings.Fields(d.Value)
				o := regexpOverrideSpec{}
				var lbl string
				if len(parts) == 3 {
					o.ImpLang = parts[0]
					var err error
					o.ImpRegex, err = regexp.Compile(parts[1])
					if err != nil {
						log.Printf("gazelle:resolve_exp %s: %v", d.Value, err)
						continue
					}
					lbl = parts[2]
				} else if len(parts) == 4 {
					o.ImpLang = parts[0]
					o.lang = parts[1]
					var err error
					o.ImpRegex, err = regexp.Compile(parts[2])
					if err != nil {
						log.Printf("gazelle:resolve_exp %s: %v", d.Value, err)
						continue
					}

					lbl = parts[3]
				} else {
					log.Printf("could not parse directive: %s\n\texpected gazelle:resolve_regexp source-language [import-language] import-string-regex label", d.Value)
					continue
				}
				var err error
				o.dep, err = label.Parse(lbl)
				if err != nil {
					log.Printf("gazelle:resolve_regexp %s: %v", d.Value, err)
					continue
				}
				o.dep = o.dep.Abs("", rel)
				rcCopy.regexpOverrides = append(rcCopy.regexpOverrides, o)
			}
		}
	}

	c.Exts[resolveName] = rcCopy
}
