/* Copyright 2022 The Bazel Authors. All rights reserved.

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

package golang

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/build"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/language"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

func importReposFromWork(args language.ImportReposArgs) language.ImportReposResult {
	// dir where go.work is located
	dir := filepath.Dir(args.Path)

	// List all modules except for the main module, including implicit indirect
	// dependencies.
	type module struct {
		Path, Version, Sum, Error string
		Main                      bool
		Replace                   *struct {
			Path, Version string
		}
	}
	// path@version can be used as a unique identifier for looking up sums
	pathToModule := map[string]*module{}
	data, err := goListModules(dir)
	if err != nil {
		return language.ImportReposResult{Error: err}
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	for dec.More() {
		mod := new(module)
		if err := dec.Decode(mod); err != nil {
			return language.ImportReposResult{Error: err}
		}
		if mod.Main {
			continue
		}
		if mod.Replace != nil {
			if filepath.IsAbs(mod.Replace.Path) || build.IsLocalImport(mod.Replace.Path) {
				log.Printf("go_repository does not support file path replacements for %s -> %s", mod.Path,
					mod.Replace.Path)
				continue
			}
			pathToModule[mod.Replace.Path+"@"+mod.Replace.Version] = mod
		} else {
			pathToModule[mod.Path+"@"+mod.Version] = mod
		}
	}

	// Sums are missing by default for go.work.
	// That's why we run 'go mod download' to get them.
	// Once https://github.com/golang/go/issues/52792 is resolved, we can use go list.
	// This must be done in a temporary directory because 'go mod download'
	// may modify go.mod and go.sum. It does not support -mod=readonly.
	var missingSumArgs []string
	for pathVer, mod := range pathToModule {
		if mod.Sum == "" {
			missingSumArgs = append(missingSumArgs, pathVer)
		}
	}

	if len(missingSumArgs) > 0 {
		tmpDir, err := ioutil.TempDir("", "")
		if err != nil {
			return language.ImportReposResult{Error: fmt.Errorf("finding module sums: %v", err)}
		}
		defer os.RemoveAll(tmpDir)
		data, err := goModDownload(tmpDir, missingSumArgs)
		dec = json.NewDecoder(bytes.NewReader(data))
		if err != nil {
			// Best-effort try to adorn specific error details from the JSON output.
			for dec.More() {
				var dl module
				if err := dec.Decode(&dl); err != nil {
					// If we couldn't parse a possible error description, just ignore this part of the output.
					continue
				}
				if dl.Error != "" {
					err = fmt.Errorf("%v\nError downloading %v: %v", err, dl.Path, dl.Error)
				}
			}

			return language.ImportReposResult{Error: err}
		}
		for dec.More() {
			var dl module
			if err := dec.Decode(&dl); err != nil {
				return language.ImportReposResult{Error: err}
			}
			if mod, ok := pathToModule[dl.Path+"@"+dl.Version]; ok {
				mod.Sum = dl.Sum
			}
		}
	}

	// Translate to repository rules.
	gen := make([]*rule.Rule, 0, len(pathToModule))
	for pathVer, mod := range pathToModule {
		if mod.Sum == "" {
			log.Printf("could not determine sum for module %s", pathVer)
			continue
		}
		r := rule.NewRule("go_repository", label.ImportPathToBazelRepoName(mod.Path))
		r.SetAttr("importpath", mod.Path)
		r.SetAttr("sum", mod.Sum)
		if mod.Replace == nil {
			r.SetAttr("version", mod.Version)
		} else {
			r.SetAttr("replace", mod.Replace.Path)
			r.SetAttr("version", mod.Replace.Version)
		}
		gen = append(gen, r)
	}
	sort.Slice(gen, func(i, j int) bool {
		return gen[i].Name() < gen[j].Name()
	})
	return language.ImportReposResult{Gen: gen}
}
