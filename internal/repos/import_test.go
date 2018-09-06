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
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/internal/rule"
)

func init() {
	goListModulesFn = goListModulesStub
}

func TestImports(t *testing.T) {
	for _, tc := range []struct {
		desc, filename, content, want string
	}{
		{
			desc:     "dep",
			filename: "Gopkg.lock",
			content: `
# This is an abbreviated version of dep's Gopkg.lock
# Retrieved 2017-12-20

[[projects]]
  branch = "parse-constraints-with-dash-in-pre"
  name = "github.com/Masterminds/semver"
  packages = ["."]
  revision = "a93e51b5a57ef416dac8bb02d11407b6f55d8929"
  source = "https://github.com/carolynvs/semver.git"

[[projects]]
  name = "github.com/Masterminds/vcs"
  packages = ["."]
  revision = "3084677c2c188840777bff30054f2b553729d329"
  version = "v1.11.1"

[[projects]]
  branch = "master"
  name = "github.com/armon/go-radix"
  packages = ["."]
  revision = "4239b77079c7b5d1243b7b4736304ce8ddb6f0f2"

[[projects]]
  branch = "master"
  name = "golang.org/x/net"
  packages = ["context"]
  revision = "66aacef3dd8a676686c7ae3716979581e8b03c47"

[solve-meta]
  analyzer-name = "dep"
  analyzer-version = 1
  inputs-digest = "05c1cd69be2c917c0cc4b32942830c2acfa044d8200fdc94716aae48a8083702"
  solver-name = "gps-cdcl"
  solver-version = 1
`,
			want: `
go_repository(
    name = "com_github_armon_go_radix",
    commit = "4239b77079c7b5d1243b7b4736304ce8ddb6f0f2",
    importpath = "github.com/armon/go-radix",
)

go_repository(
    name = "com_github_masterminds_semver",
    commit = "a93e51b5a57ef416dac8bb02d11407b6f55d8929",
    importpath = "github.com/Masterminds/semver",
    remote = "https://github.com/carolynvs/semver.git",
)

go_repository(
    name = "com_github_masterminds_vcs",
    commit = "3084677c2c188840777bff30054f2b553729d329",
    importpath = "github.com/Masterminds/vcs",
)

go_repository(
    name = "org_golang_x_net",
    commit = "66aacef3dd8a676686c7ae3716979581e8b03c47",
    importpath = "golang.org/x/net",
)
`,
		}, {
			desc:     "modules",
			filename: "go.mod",
			content: `
module github.com/bazelbuild/bazel-gazelle

require (
	github.com/BurntSushi/toml v0.3.0 // indirect
	github.com/bazelbuild/buildtools v0.0.0-20180226164855-80c7f0d45d7e
	github.com/davecgh/go-spew v1.1.0 // indirect
	github.com/pelletier/go-toml v1.0.1
	golang.org/x/tools v0.0.0-20170824195420-5d2fd3ccab98
	gopkg.in/yaml.v2 v2.2.1 // indirect
)
`,
			want: `
go_repository(
    name = "com_github_bazelbuild_buildtools",
    commit = "80c7f0d45d7e",
    importpath = "github.com/bazelbuild/buildtools",
)

go_repository(
    name = "com_github_burntsushi_toml",
    importpath = "github.com/BurntSushi/toml",
    tag = "v0.3.0",
)

go_repository(
    name = "com_github_davecgh_go_spew",
    importpath = "github.com/davecgh/go-spew",
    tag = "v1.1.0",
)

go_repository(
    name = "com_github_pelletier_go_toml",
    importpath = "github.com/pelletier/go-toml",
    tag = "v1.0.1",
)

go_repository(
    name = "in_gopkg_check_v1",
    commit = "20d25e280405",
    importpath = "gopkg.in/check.v1",
)

go_repository(
    name = "in_gopkg_yaml_v2",
    importpath = "gopkg.in/yaml.v2",
    tag = "v2.2.1",
)

go_repository(
    name = "org_golang_x_tools",
    commit = "5d2fd3ccab98",
    importpath = "golang.org/x/tools",
)
`,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			dir, err := ioutil.TempDir(os.Getenv("TEST_TEMPDIR"), "TestImports")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(dir)
			filename := filepath.Join(dir, tc.filename)
			if err := ioutil.WriteFile(filename, []byte(tc.content), 0666); err != nil {
				t.Fatal(err)
			}

			rules, err := ImportRepoRules(filename)
			if err != nil {
				t.Fatal(err)
			}
			f := rule.EmptyFile("test", "")
			for _, r := range rules {
				r.Insert(f)
			}
			got := strings.TrimSpace(string(f.Format()))
			want := strings.TrimSpace(tc.want)
			if got != want {
				t.Errorf("got:\n%s\n\nwant:\n%s\n", got, want)
			}
		})
	}
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
	"Version": "v0.3.0",
	"Time": "2017-03-28T06:15:53Z",
	"Indirect": true,
	"Dir": "/usr/local/google/home/jayconrod/go/pkg/mod/github.com/!burnt!sushi/toml@v0.3.0",
	"GoMod": "/usr/local/google/home/jayconrod/go/pkg/mod/cache/download/github.com/!burnt!sushi/toml/@v/v0.3.0.mod"
}
{
	"Path": "github.com/bazelbuild/buildtools",
	"Version": "v0.0.0-20180226164855-80c7f0d45d7e",
	"Time": "2018-02-26T16:48:55Z",
	"Dir": "/usr/local/google/home/jayconrod/go/pkg/mod/github.com/bazelbuild/buildtools@v0.0.0-20180226164855-80c7f0d45d7e",
	"GoMod": "/usr/local/google/home/jayconrod/go/pkg/mod/cache/download/github.com/bazelbuild/buildtools/@v/v0.0.0-20180226164855-80c7f0d45d7e.mod"
}
{
	"Path": "github.com/davecgh/go-spew",
	"Version": "v1.1.0",
	"Time": "2016-10-29T20:57:26Z",
	"Indirect": true,
	"Dir": "/usr/local/google/home/jayconrod/go/pkg/mod/github.com/davecgh/go-spew@v1.1.0",
	"GoMod": "/usr/local/google/home/jayconrod/go/pkg/mod/cache/download/github.com/davecgh/go-spew/@v/v1.1.0.mod"
}
{
	"Path": "github.com/pelletier/go-toml",
	"Version": "v1.0.1",
	"Time": "2017-09-24T18:42:18Z",
	"Dir": "/usr/local/google/home/jayconrod/go/pkg/mod/github.com/pelletier/go-toml@v1.0.1",
	"GoMod": "/usr/local/google/home/jayconrod/go/pkg/mod/cache/download/github.com/pelletier/go-toml/@v/v1.0.1.mod"
}
{
	"Path": "golang.org/x/tools",
	"Version": "v0.0.0-20170824195420-5d2fd3ccab98",
	"Time": "2017-08-24T19:54:20Z",
	"Dir": "/usr/local/google/home/jayconrod/go/pkg/mod/golang.org/x/tools@v0.0.0-20170824195420-5d2fd3ccab98",
	"GoMod": "/usr/local/google/home/jayconrod/go/pkg/mod/cache/download/golang.org/x/tools/@v/v0.0.0-20170824195420-5d2fd3ccab98.mod"
}
{
	"Path": "gopkg.in/check.v1",
	"Version": "v0.0.0-20161208181325-20d25e280405",
	"Time": "2016-12-08T18:13:25Z",
	"Indirect": true,
	"Dir": "/usr/local/google/home/jayconrod/go/pkg/mod/gopkg.in/check.v1@v0.0.0-20161208181325-20d25e280405",
	"GoMod": "/usr/local/google/home/jayconrod/go/pkg/mod/cache/download/gopkg.in/check.v1/@v/v0.0.0-20161208181325-20d25e280405.mod"
}
{
	"Path": "gopkg.in/yaml.v2",
	"Version": "v2.2.1",
	"Time": "2018-03-28T19:50:20Z",
	"Indirect": true,
	"Dir": "/usr/local/google/home/jayconrod/go/pkg/mod/gopkg.in/yaml.v2@v2.2.1",
	"GoMod": "/usr/local/google/home/jayconrod/go/pkg/mod/cache/download/gopkg.in/yaml.v2/@v/v2.2.1.mod"
}
`), nil
}
