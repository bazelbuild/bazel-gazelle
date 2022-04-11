/* Copyright 2019 The Bazel Authors. All rights reserved.

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

package proto

import "testing"

func TestCheckStripImportPrefix(t *testing.T) {
	testCases := []struct {
		name, prefix, rel, wantErr string
	}{
		{
			name:    "not in directory",
			prefix:  "/example.com/idl",
			rel:     "example.com",
			wantErr: `proto_strip_import_prefix "/example.com/idl" not in directory example.com`,
		},
		{
			name:   "strip prefix at root",
			prefix: "/include",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(tt *testing.T) {
			e := checkStripImportPrefix(tc.prefix, tc.rel)
			if tc.wantErr == "" {
				if e != nil {
					t.Errorf("got:\n%v\n\nwant: nil\n", e)
				}
			} else {
				if e == nil || e.Error() != tc.wantErr {
					t.Errorf("got:\n%v\n\nwant:\n%s\n", e, tc.wantErr)
				}
			}
		})
	}
}
