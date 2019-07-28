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
	"fmt"
	"log"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"
	gzflag "github.com/bazelbuild/bazel-gazelle/flag"
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

type resolveConfig struct {
	overrides []overrideSpec
}

const resolveName = "_resolve"

func getResolveConfig(c *config.Config) *resolveConfig {
	return c.Exts[resolveName].(*resolveConfig)
}

type Configurer struct{
	resolves []string
}

func (cc *Configurer) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
	c.Exts[resolveName] = &resolveConfig{}
	fs.Var(&gzflag.MultiFlag{Values: &cc.resolves}, "resolve",
	"Specifies an explicit mapping from an import string to a label for dependency resolution (may be repeated)")
}

func (cc *Configurer) CheckFlags(fs *flag.FlagSet, c *config.Config) error {
	rc := c.Exts[resolveName].(*resolveConfig)
	for _, resolveString := range cc.resolves {
		o, err := overrideSpecFromString(resolveString, "|")
		if err != nil {
			log.Print(err)
			continue
		}
		rc.overrides = append(rc.overrides, o)
	}
	return nil
}

func (_ *Configurer) KnownDirectives() []string {
	return []string{"resolve"}
}

func (_ *Configurer) Configure(c *config.Config, rel string, f *rule.File) {
	rc := getResolveConfig(c)
	rcCopy := &resolveConfig{
		overrides: rc.overrides[:],
	}

	if f != nil {
		for _, d := range f.Directives {
			if d.Key == "resolve" {
				o, err := overrideSpecFromString(d.Value, "")
				if err != nil {
					log.Print(err)
					continue
				}
				o.dep = o.dep.Abs("", rel)
				rcCopy.overrides = append(rcCopy.overrides, o)
			}
		}
	}

	c.Exts[resolveName] = rcCopy
}

func overrideSpecFromString(s, sep string) (overrideSpec, error) {
	var parts []string
	if sep == "" {
		parts = strings.Fields(s)
	} else {
		parts = strings.Split(s, sep)
	}
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
		return o, fmt.Errorf("could not parse directive: %s\n\texpected gazelle:resolve source-language [import-language] import-string label", s)
	}
	var err error
	o.dep, err = label.Parse(lbl)
	if err != nil {
		return o, fmt.Errorf("gazelle:resolve %s: %v", s, err)
	}
	return o, nil
}
