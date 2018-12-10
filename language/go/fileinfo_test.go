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

package golang

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestOtherFileInfo(t *testing.T) {
	dir := "."
	for _, tc := range []struct {
		desc, name, source string
		wantTags           []tagLine
	}{
		{
			"empty file",
			"foo.c",
			"",
			nil,
		},
		{
			"tags file",
			"foo.c",
			`// +build foo bar
// +build baz,!ignore

`,
			[]tagLine{{{"foo"}, {"bar"}}, {{"baz", "!ignore"}}},
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			if err := ioutil.WriteFile(tc.name, []byte(tc.source), 0600); err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tc.name)

			got := otherFileInfo(filepath.Join(dir, tc.name))

			// Only check that we can extract tags. Everything else is covered
			// by other tests.
			if !reflect.DeepEqual(got.tags, tc.wantTags) {
				t.Errorf("got %#v; want %#v", got.tags, tc.wantTags)
			}
		})
	}
}

func TestFileNameInfo(t *testing.T) {
	for _, tc := range []struct {
		desc, name string
		want       fileInfo
	}{
		{
			"simple go file",
			"simple.go",
			fileInfo{
				ext: goExt,
			},
		},
		{
			"simple go test",
			"foo_test.go",
			fileInfo{
				ext:    goExt,
				isTest: true,
			},
		},
		{
			"test source",
			"test.go",
			fileInfo{
				ext:    goExt,
				isTest: false,
			},
		},
		{
			"_test source",
			"_test.go",
			fileInfo{
				ext: unknownExt,
			},
		},
		{
			"source with goos",
			"foo_linux.go",
			fileInfo{
				ext:  goExt,
				goos: "linux",
			},
		},
		{
			"source with goarch",
			"foo_amd64.go",
			fileInfo{
				ext:    goExt,
				goarch: "amd64",
			},
		},
		{
			"source with goos then goarch",
			"foo_linux_amd64.go",
			fileInfo{
				ext:    goExt,
				goos:   "linux",
				goarch: "amd64",
			},
		},
		{
			"source with goarch then goos",
			"foo_amd64_linux.go",
			fileInfo{
				ext:  goExt,
				goos: "linux",
			},
		},
		{
			"test with goos and goarch",
			"foo_linux_amd64_test.go",
			fileInfo{
				ext:    goExt,
				goos:   "linux",
				goarch: "amd64",
				isTest: true,
			},
		},
		{
			"test then goos",
			"foo_test_linux.go",
			fileInfo{
				ext:  goExt,
				goos: "linux",
			},
		},
		{
			"goos source",
			"linux.go",
			fileInfo{
				ext:  goExt,
				goos: "",
			},
		},
		{
			"goarch source",
			"amd64.go",
			fileInfo{
				ext:    goExt,
				goarch: "",
			},
		},
		{
			"goos test",
			"linux_test.go",
			fileInfo{
				ext:    goExt,
				goos:   "",
				isTest: true,
			},
		},
		{
			"c file",
			"foo_test.cxx",
			fileInfo{
				ext:    cExt,
				isTest: false,
			},
		},
		{
			"c os test file",
			"foo_linux_test.c",
			fileInfo{
				ext:    cExt,
				isTest: false,
				goos:   "linux",
			},
		},
		{
			"h file",
			"foo_linux.h",
			fileInfo{
				ext:  hExt,
				goos: "linux",
			},
		},
		{
			"go asm file",
			"foo_amd64.s",
			fileInfo{
				ext:    sExt,
				goarch: "amd64",
			},
		},
		{
			"c asm file",
			"foo.S",
			fileInfo{
				ext: csExt,
			},
		},
		{
			"unsupported file",
			"foo.m",
			fileInfo{
				ext: cExt,
			},
		},
		{
			"ignored test file",
			"foo_test.py",
			fileInfo{
				isTest: false,
			},
		},
		{
			"ignored xtest file",
			"foo_xtest.py",
			fileInfo{
				isTest: false,
			},
		},
		{
			"ignored file",
			"foo.txt",
			fileInfo{
				ext: unknownExt,
			},
		}, {
			"hidden file",
			".foo.go",
			fileInfo{
				ext: unknownExt,
			},
		},
	} {
		tc.want.name = tc.name
		tc.want.path = filepath.Join("dir", tc.name)

		if got := fileNameInfo(tc.want.path); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("case %q: got %#v; want %#v", tc.desc, got, tc.want)
		}
	}
}

func TestReadTags(t *testing.T) {
	for _, tc := range []struct {
		desc, source string
		want         []tagLine
	}{
		{
			"empty file",
			"",
			nil,
		},
		{
			"single comment without blank line",
			"// +build foo\npackage main",
			nil,
		},
		{
			"multiple comments without blank link",
			`// +build foo

// +build bar
package main

`,
			[]tagLine{{{"foo"}}},
		},
		{
			"single comment",
			"// +build foo\n\n",
			[]tagLine{{{"foo"}}},
		},
		{
			"multiple comments",
			`// +build foo
// +build bar

package main`,
			[]tagLine{{{"foo"}}, {{"bar"}}},
		},
		{
			"multiple comments with blank",
			`// +build foo

// +build bar

package main`,
			[]tagLine{{{"foo"}}, {{"bar"}}},
		},
		{
			"comment with space",
			"  //   +build   foo   bar  \n\n",
			[]tagLine{{{"foo"}, {"bar"}}},
		},
		{
			"slash star comment",
			"/* +build foo */\n\n",
			nil,
		},
	} {
		f, err := ioutil.TempFile(".", "TestReadTags")
		if err != nil {
			t.Fatal(err)
		}
		path := f.Name()
		defer os.Remove(path)
		if err = f.Close(); err != nil {
			t.Fatal(err)
		}
		if err = ioutil.WriteFile(path, []byte(tc.source), 0600); err != nil {
			t.Fatal(err)
		}

		if got, err := readTags(path); err != nil {
			t.Fatal(err)
		} else if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("case %q: got %#v; want %#v", tc.desc, got, tc.want)
		}
	}
}

func TestCheckConstraints(t *testing.T) {
	dir, err := ioutil.TempDir(os.Getenv("TEST_TEMPDIR"), "TestCheckConstraints")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	for _, tc := range []struct {
		desc                        string
		genericTags                 map[string]bool
		os, arch, filename, content string
		want                        bool
	}{
		{
			desc: "unconstrained",
			want: true,
		}, {
			desc:     "goos satisfied",
			filename: "foo_linux.go",
			os:       "linux",
			want:     true,
		}, {
			desc:     "goos unsatisfied",
			filename: "foo_linux.go",
			os:       "darwin",
			want:     false,
		}, {
			desc:     "goarch satisfied",
			filename: "foo_amd64.go",
			arch:     "amd64",
			want:     true,
		}, {
			desc:     "goarch unsatisfied",
			filename: "foo_amd64.go",
			arch:     "arm",
			want:     false,
		}, {
			desc:     "goos goarch satisfied",
			filename: "foo_linux_amd64.go",
			os:       "linux",
			arch:     "amd64",
			want:     true,
		}, {
			desc:     "goos goarch unsatisfied",
			filename: "foo_linux_amd64.go",
			os:       "darwin",
			arch:     "amd64",
			want:     false,
		}, {
			desc:     "goos unsatisfied tags satisfied",
			filename: "foo_linux.go",
			content:  "// +build foo\n\npackage foo",
			want:     false,
		}, {
			desc:        "tags all satisfied",
			genericTags: map[string]bool{"a": true, "b": true},
			content:     "// +build a,b\n\npackage foo",
			want:        true,
		}, {
			desc:        "tags some satisfied",
			genericTags: map[string]bool{"a": true},
			content:     "// +build a,b\n\npackage foo",
			want:        false,
		}, {
			desc:    "tag unsatisfied negated",
			content: "// +build !a\n\npackage foo",
			want:    true,
		}, {
			desc:        "tag satisfied negated",
			genericTags: map[string]bool{"a": true},
			content:     "// +build !a\n\npackage foo",
			want:        false,
		}, {
			desc:    "tag double negative",
			content: "// +build !!a\n\npackage foo",
			want:    false,
		}, {
			desc:        "tag group and satisfied",
			genericTags: map[string]bool{"foo": true, "bar": true},
			content:     "// +build foo,bar\n\npackage foo",
			want:        true,
		}, {
			desc:        "tag group and unsatisfied",
			genericTags: map[string]bool{"foo": true},
			content:     "// +build foo,bar\n\npackage foo",
			want:        false,
		}, {
			desc:        "tag line or satisfied",
			genericTags: map[string]bool{"foo": true},
			content:     "// +build foo bar\n\npackage foo",
			want:        true,
		}, {
			desc:        "tag line or unsatisfied",
			genericTags: map[string]bool{"foo": true},
			content:     "// +build !foo bar\n\npackage foo",
			want:        false,
		}, {
			desc:        "tag lines and satisfied",
			genericTags: map[string]bool{"foo": true, "bar": true},
			content: `
// +build foo
// +build bar

package foo`,
			want: true,
		}, {
			desc:        "tag lines and unsatisfied",
			genericTags: map[string]bool{"foo": true},
			content: `
// +build foo
// +build bar

package foo`,
			want: false,
		}, {
			desc:        "cgo tags satisfied",
			os:          "linux",
			genericTags: map[string]bool{"foo": true},
			content: `
// +build foo

package foo

/*
#cgo linux CFLAGS: -Ilinux
*/
import "C"
`,
			want: true,
		}, {
			desc: "cgo tags unsatisfied",
			os:   "linux",
			content: `
package foo

/*
#cgo !linux CFLAGS: -Inotlinux
*/
import "C"
`,
			want: false,
		}, {
			desc:    "release tags",
			content: "// +build go1.7,go1.8,go1.9,go1.91,go2.0\n\npackage foo",
			want:    true,
		}, {
			desc:    "release tag negated",
			content: "// +build !go1.8\n\npackage foo",
			want:    true,
		}, {
			desc:    "cgo tag",
			content: "// +build cgo",
			want:    true,
		}, {
			desc:    "cgo tag negated",
			content: "// +build !cgo",
			want:    true,
		}, {
			desc:    "race msan tags",
			content: "// +build msan race",
			want:    true,
		}, {
			desc:    "race msan tags negated",
			content: "//+ build !msan,!race",
			want:    true,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			c, _, _ := testConfig(t)
			gc := getGoConfig(c)
			gc.genericTags = tc.genericTags
			if gc.genericTags == nil {
				gc.genericTags = map[string]bool{"gc": true}
			}
			filename := tc.filename
			if filename == "" {
				filename = tc.desc + ".go"
			}
			content := []byte(tc.content)
			if len(content) == 0 {
				content = []byte(`package foo`)
			}

			path := filepath.Join(dir, filename)
			if err := ioutil.WriteFile(path, []byte(content), 0666); err != nil {
				t.Fatal(err)
			}

			fi := goFileInfo(path, "")
			var cgoTags tagLine
			if len(fi.copts) > 0 {
				cgoTags = fi.copts[0].tags
			}

			got := checkConstraints(c, tc.os, tc.arch, fi.goos, fi.goarch, fi.tags, cgoTags)
			if got != tc.want {
				t.Errorf("got %v ; want %v", got, tc.want)
			}
		})
	}
}
