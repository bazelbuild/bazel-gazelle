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

package proto

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestProtoRegexpGroupNames(t *testing.T) {
	names := protoRe.SubexpNames()
	nameMap := map[string]int{
		"import":  importSubexpIndex,
		"package": packageSubexpIndex,
		"optkey":  optkeySubexpIndex,
		"optval":  optvalSubexpIndex,
		"service": serviceSubexpIndex,
	}
	for name, index := range nameMap {
		if names[index] != name {
			t.Errorf("proto regexp subexp %d is %s ; want %s", index, names[index], name)
		}
	}
	if len(names)-1 != len(nameMap) {
		t.Errorf("proto regexp has %d groups ; want %d", len(names), len(nameMap))
	}
}

func TestProtoFileInfo(t *testing.T) {
	for _, tc := range []struct {
		desc, name, proto string
		want              FileInfo
	}{
		{
			desc:  "empty",
			name:  "empty^file.proto",
			proto: "",
			want:  FileInfo{},
		}, {
			desc:  "simple package",
			name:  "package.proto",
			proto: "package foo;",
			want: FileInfo{
				PackageName: "foo",
			},
		}, {
			desc:  "full package",
			name:  "full.proto",
			proto: "package foo.bar.baz;",
			want: FileInfo{
				PackageName: "foo.bar.baz",
			},
		}, {
			desc: "import simple",
			name: "imp.proto",
			proto: `import 'single.proto';
import "double.proto";`,
			want: FileInfo{
				Imports: []string{"double.proto", "single.proto"},
			},
		}, {
			desc: "import quote",
			name: "quote.proto",
			proto: `import '""\".proto"';
import "'.proto";`,
			want: FileInfo{
				Imports: []string{"\"\"\".proto\"", "'.proto"},
			},
		}, {
			desc:  "import escape",
			name:  "escape.proto",
			proto: `import '\n\012\x0a.proto';`,
			want: FileInfo{
				Imports: []string{"\n\n\n.proto"},
			},
		}, {
			desc: "import two",
			name: "two.proto",
			proto: `import "first.proto";
import "second.proto";`,
			want: FileInfo{
				Imports: []string{"first.proto", "second.proto"},
			},
		}, {
			desc:  "go_package",
			name:  "gopkg.proto",
			proto: `option go_package = "github.com/example/project;projectpb";`,
			want: FileInfo{
				Options: []Option{{Key: "go_package", Value: "github.com/example/project;projectpb"}},
			},
		}, {
			desc:  "service",
			name:  "service.proto",
			proto: `service ChatService {}`,
			want: FileInfo{
				HasServices: true,
			},
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			dir, err := ioutil.TempDir(os.Getenv("TEST_TEMPDIR"), "TestProtoFileinfo")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(dir)
			if err := ioutil.WriteFile(filepath.Join(dir, tc.name), []byte(tc.proto), 0600); err != nil {
				t.Fatal(err)
			}

			got := protoFileInfo(dir, tc.name)

			// Clear fields we don't care about for testing.
			got = FileInfo{
				PackageName: got.PackageName,
				Imports:     got.Imports,
				Options:     got.Options,
				HasServices: got.HasServices,
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %#v; want %#v", got, tc.want)
			}
		})
	}
}
