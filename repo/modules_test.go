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

package repo

import "testing"

func TestGoModuleSpecialCases(t *testing.T) {
	for _, tc := range []struct {
		in, wantCommit, wantTag string
	}{
		{in: "v0.30.0", wantTag: "v0.30.0"},
		{in: "v0.0.0-20180718195005-e651d75abec6", wantCommit: "e651d75abec6"},
		{in: "v2.0.0+incompatible", wantTag: "v2.0.0"},
		{in: "v1.0.0-20170511165959-379148ca0225", wantCommit: "379148ca0225"},
		{in: "v0.8.3-0.20180104185457-379148ca0225", wantCommit: "379148ca0225"},
		{in: "v0.8.3-pre.0.20180104185457-379148ca0235", wantCommit: "379148ca0235"},
	} {
		t.Run(tc.in, func(t *testing.T) {
			repo := toRepoRule(module{Version: tc.in})
			if repo.Commit != tc.wantCommit {
				t.Errorf("commit for %q: got %q; want %q", tc.in, repo.Commit, tc.wantCommit)
			} else if repo.Tag != tc.wantTag {
				t.Errorf("tag for %q: got %q; want %q", tc.in, repo.Tag, tc.wantTag)
			}
		})
	}
}
