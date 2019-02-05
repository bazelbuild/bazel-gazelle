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
	e := checkStripImportPrefix("/example.com/idl", "example.com")
	wantErr := "invalid proto_strip_import_prefix \"/example.com/idl\" at example.com"
	if e == nil || e.Error() != wantErr {
		t.Errorf("got:\n%v\n\nwant:\n%s\n", e, wantErr)
	}
}