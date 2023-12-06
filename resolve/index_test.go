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

package resolve_test

import (
	"path"
	"strings"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/repo"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/google/go-cmp/cmp"
)

// testResolver implements the resolve.Resolve interface, with a configurable language name and
// Imports/Embeds taken from the rule attributes.
type testResolver struct{ name string }

func (tr testResolver) Name() string { return tr.name }

// imports is expected to be a list of the form ["Lang|Imp"]
func (tr testResolver) Imports(c *config.Config, r *rule.Rule, f *rule.File) []resolve.ImportSpec {
	if r.Attr("imports") == nil {
		return nil
	}

	result := []resolve.ImportSpec{}
	for _, string := range r.AttrStrings("imports") {
		parts := strings.SplitN(string, "|", 2)
		result = append(result, resolve.ImportSpec{parts[0], parts[1]})
	}

	return result
}

// Embeds is parsed as a label list
func (tr testResolver) Embeds(r *rule.Rule, from label.Label) []label.Label {
	var result []label.Label
	for _, str := range r.AttrStrings("embeds") {
		label, err := label.Parse(str)
		if err != nil {
			panic(err)
		}
		result = append(result, label)
	}
	return result
}

func (tr testResolver) Resolve(c *config.Config, ix *resolve.RuleIndex, rc *repo.RemoteCache, r *rule.Rule, imports interface{}, from label.Label) {
	panic("not implemented")
}

func TestIndex(t *testing.T) {
	content := `
test(
	name = "simple",
	imports = ["|root.simple"],
)

test(
	name = "embedded",
	imports = ["|root.embedded"],
)

test(
	name = "embeds",
	imports = [],
	embeds = [":embedded"],
)

test(
	name = "nil",
	embeds = [":embedded"],
)
`

	testCases := []struct {
		desc       string
		lang       string
		query      resolve.ImportSpec
		wantNames  []string
		wantEmbeds [][]string
	}{
		{
			desc:      "Simple",
			lang:      "test",
			query:     resolve.ImportSpec{"", "root.simple"},
			wantNames: []string{"simple"},
		},
		{
			desc:       "Embedded",
			lang:       "test",
			query:      resolve.ImportSpec{"", "root.embedded"},
			wantNames:  []string{"embeds"},
			wantEmbeds: [][]string{{"embedded"}},
		},
	}

	ix := resolve.NewRuleIndex(func(r *rule.Rule, pkgRel string) resolve.Resolver { return testResolver{r.Kind()} })
	file, err := rule.LoadData("", "", []byte(content))
	if err != nil {
		t.Fatal(err)
	}
	for _, rule := range file.Rules {
		ix.AddRule(config.New(), rule, file)
	}

	dir := t.TempDir()
	savedFile := path.Join(dir, "data.json")
	if err := ix.WriteToFile(savedFile); err != nil {
		t.Fatal(err)
	}
	loaded := resolve.NewRuleIndex(nil)
	loaded.ReadFromFile(savedFile, nil)

	indexes := []struct {
		name      string
		ruleIndex *resolve.RuleIndex
	}{
		{"parsed", ix},
		{"loaded", loaded},
	}

	for _, index := range indexes {
		t.Run(index.name, func(t *testing.T) {
			for _, tC := range testCases {
				t.Run(tC.desc, func(t *testing.T) {
					got := index.ruleIndex.FindRulesByImport(tC.query, tC.lang)
					var gotNames []string
					var gotEmbeds [][]string
					for _, val := range got {
						gotNames = append(gotNames, val.Label.Name)
						var embeds []string
						for _, embed := range val.Embeds {
							embeds = append(embeds, embed.Name)
						}
						gotEmbeds = append(gotEmbeds, embeds)
					}

					if !cmp.Equal(gotNames, tC.wantNames) {
						t.Errorf("Querying for rule %v: expected %v got %v", tC.query, tC.wantNames, gotNames)
					}

					if tC.wantEmbeds != nil && !cmp.Equal(gotEmbeds, tC.wantEmbeds) {
						t.Errorf("Embeds for query %v: expected %v got %v", tC.query, tC.wantEmbeds, gotEmbeds)
					}
				})
			}
		})
	}
}
