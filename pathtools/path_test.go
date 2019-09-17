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

package pathtools

import "testing"

func TestHasPrefix(t *testing.T) {
	for _, tc := range []struct {
		desc, path, prefix string
		want               bool
	}{
		{
			desc:   "empty prefix",
			path:   "home/jr_hacker",
			prefix: "",
			want:   true,
		}, {
			desc:   "partial prefix",
			path:   "home/jr_hacker",
			prefix: "home",
			want:   true,
		}, {
			desc:   "full prefix",
			path:   "home/jr_hacker",
			prefix: "home/jr_hacker",
			want:   true,
		}, {
			desc:   "too long",
			path:   "home",
			prefix: "home/jr_hacker",
			want:   false,
		}, {
			desc:   "partial component",
			path:   "home/jr_hacker",
			prefix: "home/jr_",
			want:   false,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			if got := HasPrefix(tc.path, tc.prefix); got != tc.want {
				t.Errorf("got %v ; want %v", got, tc.want)
			}
		})
	}
}

func TestTrimPrefix(t *testing.T) {
	for _, tc := range []struct {
		desc, path, prefix, want string
	}{
		{
			desc:   "empty prefix",
			path:   "home/jr_hacker",
			prefix: "",
			want:   "home/jr_hacker",
		}, {
			desc:   "partial prefix",
			path:   "home/jr_hacker",
			prefix: "home",
			want:   "jr_hacker",
		}, {
			desc:   "full prefix",
			path:   "home/jr_hacker",
			prefix: "home/jr_hacker",
			want:   "",
		}, {
			desc:   "partial component",
			path:   "home/jr_hacker",
			prefix: "home/jr_",
			want:   "home/jr_hacker",
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			if got := TrimPrefix(tc.path, tc.prefix); got != tc.want {
				t.Errorf("got %q ; want %q", got, tc.want)
			}
		})
	}
}

func TestIndex(t *testing.T) {
	for _, tc := range []struct {
		desc, path, sub string
		want            int
	}{
		{"path_empty", "", "x", -1},
		{"sub_empty", "x", "", 0},
		{"path_and_sub_empty", "", "", 0},
		{"only", "ab", "ab", 0},
		{"first", "ab/cd/ef", "ab", 0},
		{"middle", "ab/cd/ef", "cd", 3},
		{"last", "ab/cd/ef", "ef", 6},
		{"multi_first", "ab/cd/ef", "ab/cd", 0},
		{"multi_last", "ab/cd/ef", "cd/ef", 3},
		{"missing", "xy", "x", -1},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			got := Index(tc.path, tc.sub)
			if got != tc.want {
				t.Errorf("got %d; want %d", got, tc.want)
			}
		})
	}
}
