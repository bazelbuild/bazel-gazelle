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

package common

import "strings"

// PathHasPrefix returns whether the slash-separated path p has the given
// prefix. Unlike strings.HasPrefix, this function respects component
// boundaries, so "/home/foo" is not a prefix is "/home/foobar/baz". If the
// prefix is empty, this function always returns true.
func PathHasPrefix(p, prefix string) bool {
	return prefix == "" || p == prefix || strings.HasPrefix(p, prefix+"/")
}

// PathTrimPrefix returns p without the provided prefix. If p doesn't start
// with prefix, it returns p unchanged. Unlike strings.HasPrefix, this function
// respects component boundaries (assuming slash-separated paths), so
// PathTrimPrefix("foo/bar", "foo") returns "baz".
func PathTrimPrefix(p, prefix string) string {
	if prefix == "" {
		return p
	}
	if prefix == p {
		return ""
	}
	return strings.TrimPrefix(p, prefix+"/")
}
