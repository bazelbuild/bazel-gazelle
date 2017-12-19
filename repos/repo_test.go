/* Copyright 2017 The Bazel Authors. All rights reserved.

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

package repos

import (
	"testing"

	bf "github.com/bazelbuild/buildtools/build"
)

func TestGenerateRepoRules(t *testing.T) {
	repo := repo{
		name:       "org_golang_x_tools",
		importPath: "golang.org/x/tools",
		commit:     "123456",
	}
	got := bf.FormatString(generateRepoRule(repo))
	want := `go_repository(
    name = "org_golang_x_tools",
    commit = "123456",
    importpath = "golang.org/x/tools",
)`
	if got != want {
		t.Errorf("got %s ; want %s", got, want)
	}
}
