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

package golang

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/language"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
	"golang.org/x/tools/go/packages"
)

func importReposFromParse(args language.ImportReposArgs) language.ImportReposResult {
	// Parse go.sum for checksum
	checksumIdx := make(map[string]string)
	goSumPath := filepath.Join(filepath.Dir(args.Path), "go.sum")
	goSumFile, err := os.Open(goSumPath)
	if err != nil {
		return language.ImportReposResult{
			Error: fmt.Errorf("failed to open go.sum file at %s: %v", goSumPath, err),
		}
	}
	scanner := bufio.NewScanner(goSumFile)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		fields := strings.Fields(line)
		if len(fields) != 3 {
			continue
		}
		path, version, sum := fields[0], fields[1], fields[2]
		if strings.HasSuffix(version, "/go.mod") {
			continue
		}
		checksumIdx[path+"@"+version] = sum
	}
	if scanner.Err() != nil {
		return language.ImportReposResult{
			Error: fmt.Errorf("failed to parse go.sum file at %s: %v", goSumPath, scanner.Err()),
		}
	}

	// Parse go.mod for modules information
	b, err := os.ReadFile(args.Path)
	if err != nil {
		return language.ImportReposResult{
			Error: fmt.Errorf("failed to read go.mod file at %s: %v", args.Path, err),
		}
	}
	modFile, err := modfile.Parse(filepath.Base(args.Path), b, nil)
	if err != nil {
		return language.ImportReposResult{
			Error: fmt.Errorf("failed to parse go.mod file at %s: %v", args.Path, err),
		}
	}

	// Build an index of 'replace' directives
	replaceIdx := make(map[string]module.Version, len(modFile.Replace))
	for _, replace := range modFile.Replace {
		replaceIdx[replace.Old.String()] = replace.New
	}

	pathToModule := make(map[string]*moduleFromList, len(modFile.Require))
	for _, require := range modFile.Require {
		modKey := require.Mod.String()
		pathToModule[modKey] = &moduleFromList{
			Module: packages.Module{
				Path:    require.Mod.Path,
				Version: require.Mod.Version,
			},
		}

		// If there is a match replacement, add .Replace and change .Sum to the checksum of the new module
		replace, foundReplace := replaceIdx[modKey]
		if !foundReplace {
			replace, foundReplace = replaceIdx[require.Mod.Path]
		}
		if foundReplace {
			replaceChecksum, foundReplaceChecksum := checksumIdx[replace.String()]
			if !foundReplaceChecksum {
				return language.ImportReposResult{
					Error: fmt.Errorf("module %s is missing from go.sum. Run 'go mod tidy' to fix.", replace.String()),
				}
			}
			pathToModule[modKey].Replace = &packages.Module{
				Path:    replace.Path,
				Version: replace.Version,
			}
			pathToModule[modKey].Sum = replaceChecksum
			continue
		}

		checksum, ok := checksumIdx[modKey]
		if !ok {
			return language.ImportReposResult{
				Error: fmt.Errorf("module %s is missing from go.sum. Run 'go mod tidy' to fix.", modKey),
			}
		}
		pathToModule[modKey].Sum = checksum
	}

	return language.ImportReposResult{Gen: toRepositoryRules(pathToModule)}
}

func importReposFromModules(args language.ImportReposArgs) language.ImportReposResult {
	if args.ParseOnly {
		return importReposFromParse(args)
	}
	// run go list in the dir where go.mod is located
	data, err := goListModules(filepath.Dir(args.Path))
	if err != nil {
		return language.ImportReposResult{Error: processGoListError(err, data)}
	}

	pathToModule, err := extractModules(data)
	if err != nil {
		return language.ImportReposResult{Error: err}
	}

	// Load sums from go.sum. Ideally, they're all there.
	goSumPath := filepath.Join(filepath.Dir(args.Path), "go.sum")
	data, _ = os.ReadFile(goSumPath)
	lines := bytes.Split(data, []byte("\n"))
	for _, line := range lines {
		line = bytes.TrimSpace(line)
		fields := bytes.Fields(line)
		if len(fields) != 3 {
			continue
		}
		path, version, sum := string(fields[0]), string(fields[1]), string(fields[2])
		if strings.HasSuffix(version, "/go.mod") {
			continue
		}
		if mod, ok := pathToModule[path+"@"+version]; ok {
			mod.Sum = sum
		}
	}

	pathToModule, err = fillMissingSums(pathToModule)
	if err != nil {
		return language.ImportReposResult{Error: fmt.Errorf("finding module sums: %v", err)}
	}

	return language.ImportReposResult{Gen: toRepositoryRules(pathToModule)}
}
