/*
	Copyright 2018 The Bazel Authors. All rights reserved.

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

func init() {
	// Replace some functions with test stubs. This avoids a dependency on
	// the go command in the actual test, which is sandboxed.
	goListModules = goListModulesStub
	goModDownload = goModDownloadStub
}

func goListModulesStub(dir string) ([]byte, error) {
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
}

func goModDownloadStub(dir string, args []string) ([]byte, error) {
	return []byte(`{
	"Path": "golang.org/x/tools",
	"Version": "v0.0.0-20190122202912-9c309ee22fab",
	"Info": "/home/jay/go/pkg/mod/cache/download/golang.org/x/tools/@v/v0.0.0-20190122202912-9c309ee22fab.info",
	"GoMod": "/home/jay/go/pkg/mod/cache/download/golang.org/x/tools/@v/v0.0.0-20190122202912-9c309ee22fab.mod",
	"Zip": "/home/jay/go/pkg/mod/cache/download/golang.org/x/tools/@v/v0.0.0-20190122202912-9c309ee22fab.zip",
	"Dir": "/home/jay/go/pkg/mod/golang.org/x/tools@v0.0.0-20190122202912-9c309ee22fab",
	"Sum": "h1:FkAkwuYWQw+IArrnmhGlisKHQF4MsZ2Nu/fX4ttW55o=",
	"GoModSum": "h1:n7NCudcB/nEzxVGmLbDWY5pfWTLqBcC2KZ6jyYvM4mQ="
}
`), nil
}
