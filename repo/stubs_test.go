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

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/bazelbuild/bazel-gazelle/pathtools"

	"golang.org/x/tools/go/vcs"
)

// InstallTestStubs replaces some functions with test stubs. This is useful
// for avoiding a dependency on the go command in Bazel.
func InstallTestStubs() {
	goListModules = goListModulesStub
	goModDownload = goModDownloadStub
}

func goListModulesStub(dir string) ([]byte, error) {
	goModContent, err := ioutil.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return nil, err
	}
	switch {
	case bytes.Contains(goModContent, []byte("module github.com/bazelbuild/bazel-gazelle")):
		return []byte(`{
	"Path": "github.com/bazelbuild/bazel-gazelle",
	"Main": true,
	"Dir": "/tmp/tmp.XxZ9HCw1Mq",
	"GoMod": "/tmp/tmp.XxZ9HCw1Mq/go.mod"
}
{
	"Path": "github.com/BurntSushi/toml",
	"Version": "v0.3.1",
	"Time": "2017-03-28T06:15:53Z",
	"Indirect": true,
	"Dir": "/usr/local/google/home/jayconrod/go/pkg/mod/github.com/!burnt!sushi/toml@v0.3.1",
	"GoMod": "/usr/local/google/home/jayconrod/go/pkg/mod/cache/download/github.com/!burnt!sushi/toml/@v/v0.3.1.mod"
}
{
	"Path": "github.com/bazelbuild/buildtools",
	"Version": "v0.0.0-20190202002759-027686e28d67",
	"Time": "2018-02-26T16:48:55Z",
	"Dir": "/usr/local/google/home/jayconrod/go/pkg/mod/github.com/bazelbuild/buildtools@v0.0.0-20190202002759-027686e28d67",
	"GoMod": "/usr/local/google/home/jayconrod/go/pkg/mod/cache/download/github.com/bazelbuild/buildtools/@v/v0.0.0-20190202002759-027686e28d67.mod"
}
{
	"Path": "github.com/davecgh/go-spew",
	"Version": "v1.1.1",
	"Time": "2016-10-29T20:57:26Z",
	"Indirect": true,
	"Dir": "/usr/local/google/home/jayconrod/go/pkg/mod/github.com/davecgh/go-spew@v1.1.1",
	"GoMod": "/usr/local/google/home/jayconrod/go/pkg/mod/cache/download/github.com/davecgh/go-spew/@v/v1.1.1.mod"
}
{
	"Path": "github.com/fork/go-toml",
	"Version": "v0.0.0-20190116191733-b6c0e53d7304",
	"Time": "2016-10-29T20:57:26Z",
	"Dir": "/usr/local/google/home/jayconrod/go/pkg/mod/github.com/fork/!go-toml@v1.0.0",
	"GoMod": "/usr/local/google/home/jayconrod/go/pkg/mod/cache/download/github.com/fork/go-toml@v/v1.0.0.mod"
}
{
	"Path": "github.com/pelletier/go-toml",
	"Version": "v1.0.1",
	"Time": "2017-09-24T18:42:18Z",
	"Dir": "/usr/local/google/home/jayconrod/go/pkg/mod/github.com/pelletier/go-toml@v1.0.1",
	"Replace": {
		"Path": "github.com/fork/go-toml",
		"Version": "v0.0.0-20190425002759-70bc0436ed16",
		"Time": "2017-04-06T11:16:28Z",
		"Dir": "/usr/local/google/home/jayconrod/go/pkg/mod/github.com/fork/!go-toml@v1.0.1",
        "GoMod": "/usr/local/google/home/jayconrod/go/pkg/mod/cache/download/github.com/fork/go-toml@v/v1.0.1.mod"
	},
	"GoMod": "/usr/local/google/home/jayconrod/go/pkg/mod/cache/download/github.com/pelletier/go-toml/@v/v1.0.1.mod"
}
{
	"Path": "golang.org/x/tools",
	"Version": "v0.0.0-20190122202912-9c309ee22fab",
	"Time": "2017-08-24T19:54:20Z",
	"Dir": "/usr/local/google/home/jayconrod/go/pkg/mod/golang.org/x/tools@v0.0.0-20190122202912-9c309ee22fab",
	"GoMod": "/usr/local/google/home/jayconrod/go/pkg/mod/cache/download/golang.org/x/tools/@v/v0.0.0-20190122202912-9c309ee22fab.mod"
}
{
	"Path": "gopkg.in/check.v1",
	"Version": "v1.0.0-20180628173108-788fd7840127",
	"Time": "2016-12-08T18:13:25Z",
	"Indirect": true,
	"Dir": "/usr/local/google/home/jayconrod/go/pkg/mod/gopkg.in/check.v1@v1.0.0-20180628173108-788fd7840127",
	"GoMod": "/usr/local/google/home/jayconrod/go/pkg/mod/cache/download/gopkg.in/check.v1/@v/v1.0.0-20180628173108-788fd7840127.mod"
}
{
	"Path": "gopkg.in/yaml.v2",
	"Version": "v2.2.2",
	"Time": "2018-03-28T19:50:20Z",
	"Indirect": true,
	"Dir": "/usr/local/google/home/jayconrod/go/pkg/mod/gopkg.in/yaml.v2@v2.2.2",
	"GoMod": "/usr/local/google/home/jayconrod/go/pkg/mod/cache/download/gopkg.in/yaml.v2/@v/v2.2.2.mod"
}
`), nil

	case bytes.Contains(goModContent, []byte("module modules_cloud")):
		return []byte(`{
	"Path": "gazelle_bug_test",
	"Main": true,
	"Dir": "/Users/jayconrod/Code/test",
	"GoMod": "/Users/jayconrod/Code/test/go.mod",
	"GoVersion": "1.12"
}
{
	"Path": "cloud.google.com/go",
	"Version": "v0.43.0",
	"Time": "2019-07-18T21:07:03Z",
	"Indirect": true,
	"Dir": "/Users/jayconrod/go/pkg/mod/cloud.google.com/go@v0.43.0",
	"GoMod": "/Users/jayconrod/go/pkg/mod/cache/download/cloud.google.com/go/@v/v0.43.0.mod",
	"GoVersion": "1.9"
}
{
	"Path": "cloud.google.com/go/logging",
	"Version": "v1.0.0",
	"Time": "2019-07-18T21:07:03Z",
	"Dir": "/Users/jayconrod/go/pkg/mod/cloud.google.com/go/logging@v1.0.0",
	"GoMod": "/Users/jayconrod/go/pkg/mod/cache/download/cloud.google.com/go/logging/@v/v1.0.0.mod",
	"GoVersion": "1.9"
}
{
	"Path": "github.com/BurntSushi/toml",
	"Version": "v0.3.1",
	"Time": "2018-08-15T10:47:33Z",
	"Indirect": true,
	"Dir": "/Users/jayconrod/go/pkg/mod/github.com/!burnt!sushi/toml@v0.3.1",
	"GoMod": "/Users/jayconrod/go/pkg/mod/cache/download/github.com/!burnt!sushi/toml/@v/v0.3.1.mod"
}
{
	"Path": "github.com/BurntSushi/xgb",
	"Version": "v0.0.0-20160522181843-27f122750802",
	"Time": "2016-05-22T11:18:43-07:00",
	"Indirect": true,
	"Dir": "/Users/jayconrod/go/pkg/mod/github.com/!burnt!sushi/xgb@v0.0.0-20160522181843-27f122750802",
	"GoMod": "/Users/jayconrod/go/pkg/mod/cache/download/github.com/!burnt!sushi/xgb/@v/v0.0.0-20160522181843-27f122750802.mod"
}
{
	"Path": "github.com/client9/misspell",
	"Version": "v0.3.4",
	"Time": "2018-03-09T01:55:12Z",
	"Indirect": true,
	"Dir": "/Users/jayconrod/go/pkg/mod/github.com/client9/misspell@v0.3.4",
	"GoMod": "/Users/jayconrod/go/pkg/mod/cache/download/github.com/client9/misspell/@v/v0.3.4.mod"
}
{
	"Path": "github.com/golang/glog",
	"Version": "v0.0.0-20160126235308-23def4e6c14b",
	"Time": "2016-01-26T23:53:08Z",
	"Indirect": true,
	"Dir": "/Users/jayconrod/go/pkg/mod/github.com/golang/glog@v0.0.0-20160126235308-23def4e6c14b",
	"GoMod": "/Users/jayconrod/go/pkg/mod/cache/download/github.com/golang/glog/@v/v0.0.0-20160126235308-23def4e6c14b.mod"
}
{
	"Path": "github.com/golang/mock",
	"Version": "v1.3.1",
	"Time": "2019-05-09T10:47:53-07:00",
	"Indirect": true,
	"Dir": "/Users/jayconrod/go/pkg/mod/github.com/golang/mock@v1.3.1",
	"GoMod": "/Users/jayconrod/go/pkg/mod/cache/download/github.com/golang/mock/@v/v1.3.1.mod"
}
{
	"Path": "github.com/golang/protobuf",
	"Version": "v1.3.2",
	"Time": "2019-07-01T18:22:01Z",
	"Indirect": true,
	"Dir": "/Users/jayconrod/go/pkg/mod/github.com/golang/protobuf@v1.3.2",
	"GoMod": "/Users/jayconrod/go/pkg/mod/cache/download/github.com/golang/protobuf/@v/v1.3.2.mod"
}
{
	"Path": "github.com/google/btree",
	"Version": "v1.0.0",
	"Time": "2018-08-13T08:31:12-07:00",
	"Indirect": true,
	"Dir": "/Users/jayconrod/go/pkg/mod/github.com/google/btree@v1.0.0",
	"GoMod": "/Users/jayconrod/go/pkg/mod/cache/download/github.com/google/btree/@v/v1.0.0.mod"
}
{
	"Path": "github.com/google/go-cmp",
	"Version": "v0.3.0",
	"Time": "2019-03-11T20:24:27-07:00",
	"Indirect": true,
	"Dir": "/Users/jayconrod/go/pkg/mod/github.com/google/go-cmp@v0.3.0",
	"GoMod": "/Users/jayconrod/go/pkg/mod/cache/download/github.com/google/go-cmp/@v/v0.3.0.mod",
	"GoVersion": "1.8"
}
{
	"Path": "github.com/google/martian",
	"Version": "v2.1.0+incompatible",
	"Time": "2018-09-28T21:15:21Z",
	"Indirect": true,
	"Dir": "/Users/jayconrod/go/pkg/mod/github.com/google/martian@v2.1.0+incompatible",
	"GoMod": "/Users/jayconrod/go/pkg/mod/cache/download/github.com/google/martian/@v/v2.1.0+incompatible.mod"
}
{
	"Path": "github.com/google/pprof",
	"Version": "v0.0.0-20190515194954-54271f7e092f",
	"Time": "2019-05-15T12:49:54-07:00",
	"Indirect": true,
	"Dir": "/Users/jayconrod/go/pkg/mod/github.com/google/pprof@v0.0.0-20190515194954-54271f7e092f",
	"GoMod": "/Users/jayconrod/go/pkg/mod/cache/download/github.com/google/pprof/@v/v0.0.0-20190515194954-54271f7e092f.mod"
}
{
	"Path": "github.com/googleapis/gax-go/v2",
	"Version": "v2.0.5",
	"Time": "2019-05-13T11:38:25-07:00",
	"Indirect": true,
	"Dir": "/Users/jayconrod/go/pkg/mod/github.com/googleapis/gax-go/v2@v2.0.5",
	"GoMod": "/Users/jayconrod/go/pkg/mod/cache/download/github.com/googleapis/gax-go/v2/@v/v2.0.5.mod"
}
{
	"Path": "github.com/hashicorp/golang-lru",
	"Version": "v0.5.1",
	"Time": "2019-02-27T14:24:58-08:00",
	"Indirect": true,
	"Dir": "/Users/jayconrod/go/pkg/mod/github.com/hashicorp/golang-lru@v0.5.1",
	"GoMod": "/Users/jayconrod/go/pkg/mod/cache/download/github.com/hashicorp/golang-lru/@v/v0.5.1.mod"
}
{
	"Path": "github.com/jstemmer/go-junit-report",
	"Version": "v0.0.0-20190106144839-af01ea7f8024",
	"Time": "2019-01-06T14:48:39Z",
	"Indirect": true,
	"Dir": "/Users/jayconrod/go/pkg/mod/github.com/jstemmer/go-junit-report@v0.0.0-20190106144839-af01ea7f8024",
	"GoMod": "/Users/jayconrod/go/pkg/mod/cache/download/github.com/jstemmer/go-junit-report/@v/v0.0.0-20190106144839-af01ea7f8024.mod"
}
{
	"Path": "go.opencensus.io",
	"Version": "v0.22.0",
	"Time": "2019-05-29T12:10:40-07:00",
	"Indirect": true,
	"Dir": "/Users/jayconrod/go/pkg/mod/go.opencensus.io@v0.22.0",
	"GoMod": "/Users/jayconrod/go/pkg/mod/cache/download/go.opencensus.io/@v/v0.22.0.mod"
}
{
	"Path": "golang.org/x/crypto",
	"Version": "v0.0.0-20190605123033-f99c8df09eb5",
	"Time": "2019-06-05T05:30:33-07:00",
	"Indirect": true,
	"Dir": "/Users/jayconrod/go/pkg/mod/golang.org/x/crypto@v0.0.0-20190605123033-f99c8df09eb5",
	"GoMod": "/Users/jayconrod/go/pkg/mod/cache/download/golang.org/x/crypto/@v/v0.0.0-20190605123033-f99c8df09eb5.mod"
}
{
	"Path": "golang.org/x/exp",
	"Version": "v0.0.0-20190510132918-efd6b22b2522",
	"Time": "2019-05-10T06:29:18-07:00",
	"Indirect": true,
	"Dir": "/Users/jayconrod/go/pkg/mod/golang.org/x/exp@v0.0.0-20190510132918-efd6b22b2522",
	"GoMod": "/Users/jayconrod/go/pkg/mod/cache/download/golang.org/x/exp/@v/v0.0.0-20190510132918-efd6b22b2522.mod",
	"GoVersion": "1.11"
}
{
	"Path": "golang.org/x/image",
	"Version": "v0.0.0-20190227222117-0694c2d4d067",
	"Time": "2019-02-27T14:21:17-08:00",
	"Indirect": true,
	"Dir": "/Users/jayconrod/go/pkg/mod/golang.org/x/image@v0.0.0-20190227222117-0694c2d4d067",
	"GoMod": "/Users/jayconrod/go/pkg/mod/cache/download/golang.org/x/image/@v/v0.0.0-20190227222117-0694c2d4d067.mod"
}
{
	"Path": "golang.org/x/lint",
	"Version": "v0.0.0-20190409202823-959b441ac422",
	"Time": "2019-04-09T13:28:23-07:00",
	"Indirect": true,
	"Dir": "/Users/jayconrod/go/pkg/mod/golang.org/x/lint@v0.0.0-20190409202823-959b441ac422",
	"GoMod": "/Users/jayconrod/go/pkg/mod/cache/download/golang.org/x/lint/@v/v0.0.0-20190409202823-959b441ac422.mod"
}
{
	"Path": "golang.org/x/mobile",
	"Version": "v0.0.0-20190312151609-d3739f865fa6",
	"Time": "2019-03-12T08:16:09-07:00",
	"Indirect": true,
	"Dir": "/Users/jayconrod/go/pkg/mod/golang.org/x/mobile@v0.0.0-20190312151609-d3739f865fa6",
	"GoMod": "/Users/jayconrod/go/pkg/mod/cache/download/golang.org/x/mobile/@v/v0.0.0-20190312151609-d3739f865fa6.mod"
}
{
	"Path": "golang.org/x/net",
	"Version": "v0.0.0-20190620200207-3b0461eec859",
	"Time": "2019-06-20T13:02:07-07:00",
	"Indirect": true,
	"Dir": "/Users/jayconrod/go/pkg/mod/golang.org/x/net@v0.0.0-20190620200207-3b0461eec859",
	"GoMod": "/Users/jayconrod/go/pkg/mod/cache/download/golang.org/x/net/@v/v0.0.0-20190620200207-3b0461eec859.mod",
	"GoVersion": "1.11"
}
{
	"Path": "golang.org/x/oauth2",
	"Version": "v0.0.0-20190604053449-0f29369cfe45",
	"Time": "2019-06-03T22:34:49-07:00",
	"Indirect": true,
	"Dir": "/Users/jayconrod/go/pkg/mod/golang.org/x/oauth2@v0.0.0-20190604053449-0f29369cfe45",
	"GoMod": "/Users/jayconrod/go/pkg/mod/cache/download/golang.org/x/oauth2/@v/v0.0.0-20190604053449-0f29369cfe45.mod",
	"GoVersion": "1.11"
}
{
	"Path": "golang.org/x/sync",
	"Version": "v0.0.0-20190423024810-112230192c58",
	"Time": "2019-04-23T02:48:10Z",
	"Indirect": true,
	"Dir": "/Users/jayconrod/go/pkg/mod/golang.org/x/sync@v0.0.0-20190423024810-112230192c58",
	"GoMod": "/Users/jayconrod/go/pkg/mod/cache/download/golang.org/x/sync/@v/v0.0.0-20190423024810-112230192c58.mod"
}
{
	"Path": "golang.org/x/sys",
	"Version": "v0.0.0-20190624142023-c5567b49c5d0",
	"Time": "2019-06-24T07:20:23-07:00",
	"Indirect": true,
	"Dir": "/Users/jayconrod/go/pkg/mod/golang.org/x/sys@v0.0.0-20190624142023-c5567b49c5d0",
	"GoMod": "/Users/jayconrod/go/pkg/mod/cache/download/golang.org/x/sys/@v/v0.0.0-20190624142023-c5567b49c5d0.mod",
	"GoVersion": "1.12"
}
{
	"Path": "golang.org/x/text",
	"Version": "v0.3.2",
	"Time": "2019-04-25T21:42:06Z",
	"Indirect": true,
	"Dir": "/Users/jayconrod/go/pkg/mod/golang.org/x/text@v0.3.2",
	"GoMod": "/Users/jayconrod/go/pkg/mod/cache/download/golang.org/x/text/@v/v0.3.2.mod"
}
{
	"Path": "golang.org/x/time",
	"Version": "v0.0.0-20190308202827-9d24e82272b4",
	"Time": "2019-03-08T12:28:27-08:00",
	"Indirect": true,
	"Dir": "/Users/jayconrod/go/pkg/mod/golang.org/x/time@v0.0.0-20190308202827-9d24e82272b4",
	"GoMod": "/Users/jayconrod/go/pkg/mod/cache/download/golang.org/x/time/@v/v0.0.0-20190308202827-9d24e82272b4.mod"
}
{
	"Path": "golang.org/x/tools",
	"Version": "v0.0.0-20190628153133-6cdbf07be9d0",
	"Time": "2019-06-28T15:31:33Z",
	"Indirect": true,
	"Dir": "/Users/jayconrod/go/pkg/mod/golang.org/x/tools@v0.0.0-20190628153133-6cdbf07be9d0",
	"GoMod": "/Users/jayconrod/go/pkg/mod/cache/download/golang.org/x/tools/@v/v0.0.0-20190628153133-6cdbf07be9d0.mod",
	"GoVersion": "1.11"
}
{
	"Path": "google.golang.org/api",
	"Version": "v0.7.0",
	"Time": "2019-06-24T12:17:51-07:00",
	"Indirect": true,
	"Dir": "/Users/jayconrod/go/pkg/mod/google.golang.org/api@v0.7.0",
	"GoMod": "/Users/jayconrod/go/pkg/mod/cache/download/google.golang.org/api/@v/v0.7.0.mod"
}
{
	"Path": "google.golang.org/appengine",
	"Version": "v1.6.1",
	"Time": "2019-06-06T10:30:15-07:00",
	"Indirect": true,
	"Dir": "/Users/jayconrod/go/pkg/mod/google.golang.org/appengine@v1.6.1",
	"GoMod": "/Users/jayconrod/go/pkg/mod/cache/download/google.golang.org/appengine/@v/v1.6.1.mod"
}
{
	"Path": "google.golang.org/genproto",
	"Version": "v0.0.0-20190716160619-c506a9f90610",
	"Time": "2019-07-16T16:06:19Z",
	"Indirect": true,
	"Dir": "/Users/jayconrod/go/pkg/mod/google.golang.org/genproto@v0.0.0-20190716160619-c506a9f90610",
	"GoMod": "/Users/jayconrod/go/pkg/mod/cache/download/google.golang.org/genproto/@v/v0.0.0-20190716160619-c506a9f90610.mod"
}
{
	"Path": "google.golang.org/grpc",
	"Version": "v1.21.1",
	"Time": "2019-06-04T14:21:58-07:00",
	"Indirect": true,
	"Dir": "/Users/jayconrod/go/pkg/mod/google.golang.org/grpc@v1.21.1",
	"GoMod": "/Users/jayconrod/go/pkg/mod/cache/download/google.golang.org/grpc/@v/v1.21.1.mod"
}
{
	"Path": "honnef.co/go/tools",
	"Version": "v0.0.0-20190418001031-e561f6794a2a",
	"Time": "2019-04-17T17:10:31-07:00",
	"Indirect": true,
	"Dir": "/Users/jayconrod/go/pkg/mod/honnef.co/go/tools@v0.0.0-20190418001031-e561f6794a2a",
	"GoMod": "/Users/jayconrod/go/pkg/mod/cache/download/honnef.co/go/tools/@v/v0.0.0-20190418001031-e561f6794a2a.mod"
}
{
	"Path": "rsc.io/binaryregexp",
	"Version": "v0.2.0",
	"Time": "2019-05-24T10:11:09-07:00",
	"Indirect": true,
	"Dir": "/Users/jayconrod/go/pkg/mod/rsc.io/binaryregexp@v0.2.0",
	"GoMod": "/Users/jayconrod/go/pkg/mod/cache/download/rsc.io/binaryregexp/@v/v0.2.0.mod",
	"GoVersion": "1.12"
}`), nil

	default:
		return nil, fmt.Errorf("goListModulesStub: unknown module")
	}
}

func goModDownloadStub(dir string, args []string) ([]byte, error) {
	b := &bytes.Buffer{}
	for _, arg := range args {
		var result string
		switch arg {
		case "golang.org/x/tools@v0.0.0-20190122202912-9c309ee22fab":
			result = `{
	"Path": "golang.org/x/tools",
	"Version": "v0.0.0-20190122202912-9c309ee22fab",
	"Info": "/home/jay/go/pkg/mod/cache/download/golang.org/x/tools/@v/v0.0.0-20190122202912-9c309ee22fab.info",
	"GoMod": "/home/jay/go/pkg/mod/cache/download/golang.org/x/tools/@v/v0.0.0-20190122202912-9c309ee22fab.mod",
	"Zip": "/home/jay/go/pkg/mod/cache/download/golang.org/x/tools/@v/v0.0.0-20190122202912-9c309ee22fab.zip",
	"Dir": "/home/jay/go/pkg/mod/golang.org/x/tools@v0.0.0-20190122202912-9c309ee22fab",
	"Sum": "h1:FkAkwuYWQw+IArrnmhGlisKHQF4MsZ2Nu/fX4ttW55o=",
	"GoModSum": "h1:n7NCudcB/nEzxVGmLbDWY5pfWTLqBcC2KZ6jyYvM4mQ="
}
`
		case "golang.org/x/image@v0.0.0-20190227222117-0694c2d4d067":
			result = `{
	"Path": "golang.org/x/image",
	"Version": "v0.0.0-20190227222117-0694c2d4d067",
	"Info": "/Users/jayconrod/go/pkg/mod/cache/download/golang.org/x/image/@v/v0.0.0-20190227222117-0694c2d4d067.info",
	"GoMod": "/Users/jayconrod/go/pkg/mod/cache/download/golang.org/x/image/@v/v0.0.0-20190227222117-0694c2d4d067.mod",
	"Zip": "/Users/jayconrod/go/pkg/mod/cache/download/golang.org/x/image/@v/v0.0.0-20190227222117-0694c2d4d067.zip",
	"Dir": "/Users/jayconrod/go/pkg/mod/golang.org/x/image@v0.0.0-20190227222117-0694c2d4d067",
	"Sum": "h1:KYGJGHOQy8oSi1fDlSpcZF0+juKwk/hEMv5SiwHogR0=",
	"GoModSum": "h1:kZ7UVZpmo3dzQBMxlp+ypCbDeSB+sBbTgSJuh5dn5js="
}`
		case "github.com/google/btree@v1.0.0":
			result = `{
	"Path": "github.com/google/btree",
	"Version": "v1.0.0",
	"Info": "/Users/jayconrod/go/pkg/mod/cache/download/github.com/google/btree/@v/v1.0.0.info",
	"GoMod": "/Users/jayconrod/go/pkg/mod/cache/download/github.com/google/btree/@v/v1.0.0.mod",
	"Zip": "/Users/jayconrod/go/pkg/mod/cache/download/github.com/google/btree/@v/v1.0.0.zip",
	"Dir": "/Users/jayconrod/go/pkg/mod/github.com/google/btree@v1.0.0",
	"Sum": "h1:0udJVsspx3VBr5FwtLhQQtuAsVc79tTq0ocGIPAU6qo=",
	"GoModSum": "h1:lNA+9X1NB3Zf8V7Ke586lFgjr2dZNuvo3lPJSGZ5JPQ=",
	"Latest": true
}`
		case "golang.org/x/crypto@v0.0.0-20190605123033-f99c8df09eb5":
			result = `{
	"Path": "golang.org/x/crypto",
	"Version": "v0.0.0-20190605123033-f99c8df09eb5",
	"Info": "/Users/jayconrod/go/pkg/mod/cache/download/golang.org/x/crypto/@v/v0.0.0-20190605123033-f99c8df09eb5.info",
	"GoMod": "/Users/jayconrod/go/pkg/mod/cache/download/golang.org/x/crypto/@v/v0.0.0-20190605123033-f99c8df09eb5.mod",
	"Zip": "/Users/jayconrod/go/pkg/mod/cache/download/golang.org/x/crypto/@v/v0.0.0-20190605123033-f99c8df09eb5.zip",
	"Dir": "/Users/jayconrod/go/pkg/mod/golang.org/x/crypto@v0.0.0-20190605123033-f99c8df09eb5",
	"Sum": "h1:58fnuSXlxZmFdJyvtTFVmVhcMLU6v5fEb/ok4wyqtNU=",
	"GoModSum": "h1:yigFU9vqHzYiE8UmvKecakEJjdnWj3jj499lnFckfCI="
}`
		case "github.com/google/pprof@v0.0.0-20190515194954-54271f7e092f":
			result = `{
	"Path": "github.com/google/pprof",
	"Version": "v0.0.0-20190515194954-54271f7e092f",
	"Info": "/Users/jayconrod/go/pkg/mod/cache/download/github.com/google/pprof/@v/v0.0.0-20190515194954-54271f7e092f.info",
	"GoMod": "/Users/jayconrod/go/pkg/mod/cache/download/github.com/google/pprof/@v/v0.0.0-20190515194954-54271f7e092f.mod",
	"Zip": "/Users/jayconrod/go/pkg/mod/cache/download/github.com/google/pprof/@v/v0.0.0-20190515194954-54271f7e092f.zip",
	"Dir": "/Users/jayconrod/go/pkg/mod/github.com/google/pprof@v0.0.0-20190515194954-54271f7e092f",
	"Sum": "h1:Jnx61latede7zDD3DiiP4gmNz33uK0U5HDUaF0a/HVQ=",
	"GoModSum": "h1:zfwlbNMJ+OItoe0UupaVj+oy1omPYYDuagoSzA8v9mc="
}`
		case "golang.org/x/tools@v0.0.0-20190628153133-6cdbf07be9d0":
			result = `{
	"Path": "golang.org/x/tools",
	"Version": "v0.0.0-20190628153133-6cdbf07be9d0",
	"Info": "/Users/jayconrod/go/pkg/mod/cache/download/golang.org/x/tools/@v/v0.0.0-20190628153133-6cdbf07be9d0.info",
	"GoMod": "/Users/jayconrod/go/pkg/mod/cache/download/golang.org/x/tools/@v/v0.0.0-20190628153133-6cdbf07be9d0.mod",
	"Zip": "/Users/jayconrod/go/pkg/mod/cache/download/golang.org/x/tools/@v/v0.0.0-20190628153133-6cdbf07be9d0.zip",
	"Dir": "/Users/jayconrod/go/pkg/mod/golang.org/x/tools@v0.0.0-20190628153133-6cdbf07be9d0",
	"Sum": "h1:Dh6fw+p6FyRl5x/FvNswO1ji0lIGzm3KP8Y9VkS9PTE=",
	"GoModSum": "h1:/rFqwRUd4F7ZHNgwSSTFct+R/Kf4OFW1sUzUTQQTgfc="
}`
		case "honnef.co/go/tools@v0.0.0-20190418001031-e561f6794a2a":
			result = `{
	"Path": "honnef.co/go/tools",
	"Version": "v0.0.0-20190418001031-e561f6794a2a",
	"Info": "/Users/jayconrod/go/pkg/mod/cache/download/honnef.co/go/tools/@v/v0.0.0-20190418001031-e561f6794a2a.info",
	"GoMod": "/Users/jayconrod/go/pkg/mod/cache/download/honnef.co/go/tools/@v/v0.0.0-20190418001031-e561f6794a2a.mod",
	"Zip": "/Users/jayconrod/go/pkg/mod/cache/download/honnef.co/go/tools/@v/v0.0.0-20190418001031-e561f6794a2a.zip",
	"Dir": "/Users/jayconrod/go/pkg/mod/honnef.co/go/tools@v0.0.0-20190418001031-e561f6794a2a",
	"Sum": "h1:LJwr7TCTghdatWv40WobzlKXc9c4s8oGa7QKJUtHhWA=",
	"GoModSum": "h1:rf3lG4BRIbNafJWhAfAdb/ePZxsR/4RtNHQocxwk9r4="
}`
		case "github.com/golang/mock@v1.3.1":
			result = `{
	"Path": "github.com/golang/mock",
	"Version": "v1.3.1",
	"Info": "/Users/jayconrod/go/pkg/mod/cache/download/github.com/golang/mock/@v/v1.3.1.info",
	"GoMod": "/Users/jayconrod/go/pkg/mod/cache/download/github.com/golang/mock/@v/v1.3.1.mod",
	"Zip": "/Users/jayconrod/go/pkg/mod/cache/download/github.com/golang/mock/@v/v1.3.1.zip",
	"Dir": "/Users/jayconrod/go/pkg/mod/github.com/golang/mock@v1.3.1",
	"Sum": "h1:qGJ6qTW+x6xX/my+8YUVl4WNpX9B7+/l2tRsHGZ7f2s=",
	"GoModSum": "h1:sBzyDLLjw3U8JLTeZvSv8jJB+tU5PVekmnlKIyFUx0Y=",
	"Latest": true
}`
		case "golang.org/x/time@v0.0.0-20190308202827-9d24e82272b4":
			result = `{
	"Path": "golang.org/x/time",
	"Version": "v0.0.0-20190308202827-9d24e82272b4",
	"Info": "/Users/jayconrod/go/pkg/mod/cache/download/golang.org/x/time/@v/v0.0.0-20190308202827-9d24e82272b4.info",
	"GoMod": "/Users/jayconrod/go/pkg/mod/cache/download/golang.org/x/time/@v/v0.0.0-20190308202827-9d24e82272b4.mod",
	"Zip": "/Users/jayconrod/go/pkg/mod/cache/download/golang.org/x/time/@v/v0.0.0-20190308202827-9d24e82272b4.zip",
	"Dir": "/Users/jayconrod/go/pkg/mod/golang.org/x/time@v0.0.0-20190308202827-9d24e82272b4",
	"Sum": "h1:SvFZT6jyqRaOeXpc5h/JSfZenJ2O330aBsf7JfSUXmQ=",
	"GoModSum": "h1:tRJNPiyCQ0inRvYxbN9jk5I+vvW/OXSQhTDSoE431IQ=",
	"Latest": true
}`
		case "golang.org/x/lint@v0.0.0-20190409202823-959b441ac422":
			result = `{
	"Path": "golang.org/x/lint",
	"Version": "v0.0.0-20190409202823-959b441ac422",
	"Info": "/Users/jayconrod/go/pkg/mod/cache/download/golang.org/x/lint/@v/v0.0.0-20190409202823-959b441ac422.info",
	"GoMod": "/Users/jayconrod/go/pkg/mod/cache/download/golang.org/x/lint/@v/v0.0.0-20190409202823-959b441ac422.mod",
	"Zip": "/Users/jayconrod/go/pkg/mod/cache/download/golang.org/x/lint/@v/v0.0.0-20190409202823-959b441ac422.zip",
	"Dir": "/Users/jayconrod/go/pkg/mod/golang.org/x/lint@v0.0.0-20190409202823-959b441ac422",
	"Sum": "h1:QzoH/1pFpZguR8NrRHLcO6jKqfv2zpuSqZLgdm7ZmjI=",
	"GoModSum": "h1:6SW0HCj/g11FgYtHlgUYUwCkIfeOF89ocIRzGO/8vkc="
}`
		case "github.com/client9/misspell@v0.3.4":
			result = `{
	"Path": "github.com/client9/misspell",
	"Version": "v0.3.4",
	"Info": "/Users/jayconrod/go/pkg/mod/cache/download/github.com/client9/misspell/@v/v0.3.4.info",
	"GoMod": "/Users/jayconrod/go/pkg/mod/cache/download/github.com/client9/misspell/@v/v0.3.4.mod",
	"Zip": "/Users/jayconrod/go/pkg/mod/cache/download/github.com/client9/misspell/@v/v0.3.4.zip",
	"Dir": "/Users/jayconrod/go/pkg/mod/github.com/client9/misspell@v0.3.4",
	"Sum": "h1:ta993UF76GwbvJcIo3Y68y/M3WxlpEHPWIGDkJYwzJI=",
	"GoModSum": "h1:qj6jICC3Q7zFZvVWo7KLAzC3yx5G7kyvSDkc90ppPyw=",
	"Latest": true
}`
		case "github.com/BurntSushi/xgb@v0.0.0-20160522181843-27f122750802":
			result = `{
	"Path": "github.com/BurntSushi/xgb",
	"Version": "v0.0.0-20160522181843-27f122750802",
	"Info": "/Users/jayconrod/go/pkg/mod/cache/download/github.com/!burnt!sushi/xgb/@v/v0.0.0-20160522181843-27f122750802.info",
	"GoMod": "/Users/jayconrod/go/pkg/mod/cache/download/github.com/!burnt!sushi/xgb/@v/v0.0.0-20160522181843-27f122750802.mod",
	"Zip": "/Users/jayconrod/go/pkg/mod/cache/download/github.com/!burnt!sushi/xgb/@v/v0.0.0-20160522181843-27f122750802.zip",
	"Dir": "/Users/jayconrod/go/pkg/mod/github.com/!burnt!sushi/xgb@v0.0.0-20160522181843-27f122750802",
	"Sum": "h1:1BDTz0u9nC3//pOCMdNH+CiXJVYJh5UQNCOBG7jbELc=",
	"GoModSum": "h1:IVnqGOEym/WlBOVXweHU+Q+/VP0lqqI8lqeDx9IjBqo=",
	"Latest": true
}`
		case "golang.org/x/exp@v0.0.0-20190510132918-efd6b22b2522":
			result = `{
	"Path": "golang.org/x/exp",
	"Version": "v0.0.0-20190510132918-efd6b22b2522",
	"Info": "/Users/jayconrod/go/pkg/mod/cache/download/golang.org/x/exp/@v/v0.0.0-20190510132918-efd6b22b2522.info",
	"GoMod": "/Users/jayconrod/go/pkg/mod/cache/download/golang.org/x/exp/@v/v0.0.0-20190510132918-efd6b22b2522.mod",
	"Zip": "/Users/jayconrod/go/pkg/mod/cache/download/golang.org/x/exp/@v/v0.0.0-20190510132918-efd6b22b2522.zip",
	"Dir": "/Users/jayconrod/go/pkg/mod/golang.org/x/exp@v0.0.0-20190510132918-efd6b22b2522",
	"Sum": "h1:OeRHuibLsmZkFj773W4LcfAGsSxJgfPONhr8cmO+eLA=",
	"GoModSum": "h1:ZjyILWgesfNpC6sMxTJOJm9Kp84zZh5NQWvqDGG3Qr8="
}`
		case "golang.org/x/mobile@v0.0.0-20190312151609-d3739f865fa6":
			result = `{
	"Path": "golang.org/x/mobile",
	"Version": "v0.0.0-20190312151609-d3739f865fa6",
	"Info": "/Users/jayconrod/go/pkg/mod/cache/download/golang.org/x/mobile/@v/v0.0.0-20190312151609-d3739f865fa6.info",
	"GoMod": "/Users/jayconrod/go/pkg/mod/cache/download/golang.org/x/mobile/@v/v0.0.0-20190312151609-d3739f865fa6.mod",
	"Zip": "/Users/jayconrod/go/pkg/mod/cache/download/golang.org/x/mobile/@v/v0.0.0-20190312151609-d3739f865fa6.zip",
	"Dir": "/Users/jayconrod/go/pkg/mod/golang.org/x/mobile@v0.0.0-20190312151609-d3739f865fa6",
	"Sum": "h1:Tus/Y4w3V77xDsGwKUC8a/QrV7jScpU557J77lFffNs=",
	"GoModSum": "h1:z+o9i4GpDbdi3rU15maQ/Ox0txvL9dWGYEHz965HBQE="
}`
		case "rsc.io/binaryregexp@v0.2.0":
			result = `{
	"Path": "rsc.io/binaryregexp",
	"Version": "v0.2.0",
	"Info": "/Users/jayconrod/go/pkg/mod/cache/download/rsc.io/binaryregexp/@v/v0.2.0.info",
	"GoMod": "/Users/jayconrod/go/pkg/mod/cache/download/rsc.io/binaryregexp/@v/v0.2.0.mod",
	"Zip": "/Users/jayconrod/go/pkg/mod/cache/download/rsc.io/binaryregexp/@v/v0.2.0.zip",
	"Dir": "/Users/jayconrod/go/pkg/mod/rsc.io/binaryregexp@v0.2.0",
	"Sum": "h1:HfqmD5MEmC0zvwBuF187nq9mdnXjXsSivRiXN7SmRkE=",
	"GoModSum": "h1:qTv7/COck+e2FymRvadv62gMdZztPaShugOCi3I+8D8=",
	"Latest": true
}`
		case "github.com/BurntSushi/toml@v0.3.1":
			result = `{
	"Path": "github.com/BurntSushi/toml",
	"Version": "v0.3.1",
	"Info": "/Users/jayconrod/go/pkg/mod/cache/download/github.com/!burnt!sushi/toml/@v/v0.3.1.info",
	"GoMod": "/Users/jayconrod/go/pkg/mod/cache/download/github.com/!burnt!sushi/toml/@v/v0.3.1.mod",
	"Zip": "/Users/jayconrod/go/pkg/mod/cache/download/github.com/!burnt!sushi/toml/@v/v0.3.1.zip",
	"Dir": "/Users/jayconrod/go/pkg/mod/github.com/!burnt!sushi/toml@v0.3.1",
	"Sum": "h1:WXkYYl6Yr3qBf1K79EBnL4mak0OimBfB0XUf9Vl28OQ=",
	"GoModSum": "h1:xHWCNGjB5oqiDr8zfno3MHue2Ht5sIBksp03qcyfWMU=",
	"Latest": true
}`
		case "github.com/jstemmer/go-junit-report@v0.0.0-20190106144839-af01ea7f8024":
			result = `{
	"Path": "github.com/jstemmer/go-junit-report",
	"Version": "v0.0.0-20190106144839-af01ea7f8024",
	"Info": "/Users/jayconrod/go/pkg/mod/cache/download/github.com/jstemmer/go-junit-report/@v/v0.0.0-20190106144839-af01ea7f8024.info",
	"GoMod": "/Users/jayconrod/go/pkg/mod/cache/download/github.com/jstemmer/go-junit-report/@v/v0.0.0-20190106144839-af01ea7f8024.mod",
	"Zip": "/Users/jayconrod/go/pkg/mod/cache/download/github.com/jstemmer/go-junit-report/@v/v0.0.0-20190106144839-af01ea7f8024.zip",
	"Dir": "/Users/jayconrod/go/pkg/mod/github.com/jstemmer/go-junit-report@v0.0.0-20190106144839-af01ea7f8024",
	"Sum": "h1:rBMNdlhTLzJjJSDIjNEXX1Pz3Hmwmz91v+zycvx9PJc=",
	"GoModSum": "h1:6v2b51hI/fHJwM22ozAgKL4VKDeJcHhJFhtBdhmNjmU=",
	"Latest": true
}`
		default:
			return nil, fmt.Errorf("goModDownloadStub: unknown module version: %s", arg)
		}
		b.WriteString(result)
		b.WriteByte('\n')
	}
	return b.Bytes(), nil
}

func NewStubRemoteCache(rs []Repo) *RemoteCache {
	rc, _ := NewRemoteCache(rs)
	rc.tmpDir = os.DevNull
	rc.tmpErr = errors.New("stub remote cache cannot use temp dir")
	rc.RepoRootForImportPath = stubRepoRootForImportPath
	rc.HeadCmd = stubHeadCmd
	rc.ModInfo = stubModInfo
	rc.ModVersionInfo = stubModVersionInfo
	return rc
}

// stubRepoRootForImportPath is a stub implementation of vcs.RepoRootForImportPath
func stubRepoRootForImportPath(importPath string, verbose bool) (*vcs.RepoRoot, error) {
	if pathtools.HasPrefix(importPath, "example.com/repo.git") {
		return &vcs.RepoRoot{
			VCS:  vcs.ByCmd("git"),
			Repo: "https://example.com/repo.git",
			Root: "example.com/repo.git",
		}, nil
	}

	if pathtools.HasPrefix(importPath, "example.com/repo") {
		return &vcs.RepoRoot{
			VCS:  vcs.ByCmd("git"),
			Repo: "https://example.com/repo.git",
			Root: "example.com/repo",
		}, nil
	}

	if pathtools.HasPrefix(importPath, "example.com") {
		return &vcs.RepoRoot{
			VCS:  vcs.ByCmd("git"),
			Repo: "https://example.com",
			Root: "example.com",
		}, nil
	}

	return nil, fmt.Errorf("could not resolve import path: %q", importPath)
}

func stubHeadCmd(remote, vcs string) (string, error) {
	if vcs == "git" && remote == "https://example.com/repo" {
		return "abcdef", nil
	}
	return "", fmt.Errorf("could not resolve remote: %q", remote)
}

func stubModInfo(importPath string) (string, error) {
	if pathtools.HasPrefix(importPath, "example.com/stub/v2") {
		return "example.com/stub/v2", nil
	}
	if pathtools.HasPrefix(importPath, "example.com/stub") {
		return "example.com/stub", nil
	}
	return "", fmt.Errorf("could not find module path for %s", importPath)
}

func stubModVersionInfo(modPath, query string) (version, sum string, err error) {
	if modPath == "example.com/known" || modPath == "example.com/unknown" {
		return "v1.2.3", "h1:abcdef", nil
	}
	return "", "", fmt.Errorf("no such module: %s", modPath)
}
