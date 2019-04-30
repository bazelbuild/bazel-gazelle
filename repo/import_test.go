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

package repo_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/repo"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/bazelbuild/bazel-gazelle/testtools"
)

func init() {
	repo.InstallTestStubs()
}

func TestImports(t *testing.T) {
	for _, tc := range []struct {
		desc, want string
		files      []testtools.FileSpec
	}{
		{
			desc: "dep",
			files: []testtools.FileSpec{{
				Path: "Gopkg.lock",
				Content: `
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
			}},
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
    vcs = "git",
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
			desc: "modules",
			files: []testtools.FileSpec{
				{
					Path: "go.mod",
					Content: `
module github.com/bazelbuild/bazel-gazelle

require (
	github.com/BurntSushi/toml v0.3.1 // indirect
	github.com/bazelbuild/buildtools v0.0.0-20190202002759-027686e28d67
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/fsnotify/fsnotify v1.4.7
	github.com/kr/pretty v0.1.0 // indirect
	github.com/pelletier/go-toml v1.0.1
	github.com/pmezard/go-difflib v1.0.0
	golang.org/x/sys v0.0.0-20190122071731-054c452bb702 // indirect
	golang.org/x/tools v0.0.0-20190122202912-9c309ee22fab
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
	gopkg.in/yaml.v2 v2.2.2 // indirect
)

replace (
	github.com/pelletier/go-toml => github.com/fork/go-toml v0.0.0-20190425002759-70bc0436ed16
)
`,
				}, {
					// Note: the sum for x/tools has been deleted to force a call
					// to goModDownload.
					Path: "go.sum",
					Content: `
github.com/BurntSushi/toml v0.3.1 h1:WXkYYl6Yr3qBf1K79EBnL4mak0OimBfB0XUf9Vl28OQ=
github.com/BurntSushi/toml v0.3.1/go.mod h1:xHWCNGjB5oqiDr8zfno3MHue2Ht5sIBksp03qcyfWMU=
github.com/bazelbuild/buildtools v0.0.0-20190202002759-027686e28d67 h1:zS8p6ZRbNVa7QfK3tpoIRDqGzCA2J0uJffaMTWoneac=
github.com/bazelbuild/buildtools v0.0.0-20190202002759-027686e28d67/go.mod h1:5JP0TXzWDHXv8qvxRC4InIazwdyDseBDbzESUMKk1yU=
github.com/davecgh/go-spew v1.1.1 h1:vj9j/u1bqnvCEfJOwUhtlOARqs3+rkHYY13jYWTU97c=
github.com/davecgh/go-spew v1.1.1/go.mod h1:J7Y8YcW2NihsgmVo/mv3lAwl/skON4iLHjSsI+c5H38=
github.com/fsnotify/fsnotify v1.4.7 h1:IXs+QLmnXW2CcXuY+8Mzv/fWEsPGWxqefPtCP5CnV9I=
github.com/fsnotify/fsnotify v1.4.7/go.mod h1:jwhsz4b93w/PPRr/qN1Yymfu8t87LnFCMoQvtojpjFo=
github.com/kr/pretty v0.1.0 h1:L/CwN0zerZDmRFUapSPitk6f+Q3+0za1rQkzVuMiMFI=
github.com/kr/pretty v0.1.0/go.mod h1:dAy3ld7l9f0ibDNOQOHHMYYIIbhfbHSm3C4ZsoJORNo=
github.com/kr/pty v1.1.1/go.mod h1:pFQYn66WHrOpPYNljwOMqo10TkYh1fy3cYio2l3bCsQ=
github.com/kr/text v0.1.0 h1:45sCR5RtlFHMR4UwH9sdQ5TC8v0qDQCHnXt+kaKSTVE=
github.com/kr/text v0.1.0/go.mod h1:4Jbv+DJW3UT/LiOwJeYQe1efqtUx/iVham/4vfdArNI=
github.com/fork/go-toml v0.0.0-20190425002759-70bc0436ed16 h1:T5zMGML61Wp+FlcbWjRDT7yAxhJNAiPPLOFECq181zc=
github.com/fork/go-toml v0.0.0-20190425002759-70bc0436ed16/go.mod h1:5z9KED0ma1S8pY6P1sdut58dfprrGBbd/94hg7ilaic=
github.com/pmezard/go-difflib v1.0.0 h1:4DBwDE0NGyQoBHbLQYPwSUPoCMWR5BEzIk/f1lZbAQM=
github.com/pmezard/go-difflib v1.0.0/go.mod h1:iKH77koFhYxTK1pcRnkKkqfTogsbg7gZNVY4sRDYZ/4=
golang.org/x/sys v0.0.0-20190122071731-054c452bb702 h1:Lk4tbZFnlyPgV+sLgTw5yGfzrlOn9kx4vSombi2FFlY=
golang.org/x/sys v0.0.0-20190122071731-054c452bb702/go.mod h1:STP8DvDyc/dI5b8T5hshtkjS+E42TnysNCUPdjciGhY=
gopkg.in/check.v1 v0.0.0-20161208181325-20d25e280405 h1:yhCVgyC4o1eVCa2tZl7eS0r+SDo693bJlVdllGtEeKM=
gopkg.in/check.v1 v0.0.0-20161208181325-20d25e280405/go.mod h1:Co6ibVJAznAaIkqp8huTwlJQCZ016jof/cbN4VW5Yz0=
gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 h1:qIbj1fsPNlZgppZ+VLlY7N33q108Sa+fhmuc+sWQYwY=
gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127/go.mod h1:Co6ibVJAznAaIkqp8huTwlJQCZ016jof/cbN4VW5Yz0=
gopkg.in/yaml.v2 v2.2.2 h1:ZCJp+EgiOT7lHqUV2J862kp8Qj64Jo6az82+3Td9dZw=
gopkg.in/yaml.v2 v2.2.2/go.mod h1:hI93XBmqTisBFMUTm0b8Fm+jr3Dg1NNxqwp+5A1VGuI=
`,
				},
			},
			want: `
go_repository(
    name = "com_github_bazelbuild_buildtools",
    importpath = "github.com/bazelbuild/buildtools",
    sum = "h1:zS8p6ZRbNVa7QfK3tpoIRDqGzCA2J0uJffaMTWoneac=",
    version = "v0.0.0-20180226164855-80c7f0d45d7e",
)

go_repository(
    name = "com_github_burntsushi_toml",
    importpath = "github.com/BurntSushi/toml",
    sum = "h1:WXkYYl6Yr3qBf1K79EBnL4mak0OimBfB0XUf9Vl28OQ=",
    version = "v0.3.0",
)

go_repository(
    name = "com_github_davecgh_go_spew",
    importpath = "github.com/davecgh/go-spew",
    sum = "h1:vj9j/u1bqnvCEfJOwUhtlOARqs3+rkHYY13jYWTU97c=",
    version = "v1.1.0",
)

go_repository(
    name = "com_github_pelletier_go_toml",
    importpath = "github.com/pelletier/go-toml",
    replace = "github.com/fork/go-toml",
    sum = "h1:T5zMGML61Wp+FlcbWjRDT7yAxhJNAiPPLOFECq181zc=",
    version = "v0.0.0-20190425002759-70bc0436ed16",
)

go_repository(
    name = "in_gopkg_check_v1",
    importpath = "gopkg.in/check.v1",
    sum = "h1:qIbj1fsPNlZgppZ+VLlY7N33q108Sa+fhmuc+sWQYwY=",
    version = "v0.0.0-20161208181325-20d25e280405",
)

go_repository(
    name = "in_gopkg_yaml_v2",
    importpath = "gopkg.in/yaml.v2",
    sum = "h1:ZCJp+EgiOT7lHqUV2J862kp8Qj64Jo6az82+3Td9dZw=",
    version = "v2.2.1",
)

go_repository(
    name = "org_golang_x_tools",
    importpath = "golang.org/x/tools",
    sum = "h1:FkAkwuYWQw+IArrnmhGlisKHQF4MsZ2Nu/fX4ttW55o=",
    version = "v0.0.0-20170824195420-5d2fd3ccab98",
)
`,
		}, {
			desc: "godep",
			files: []testtools.FileSpec{{
				Path: "Godeps.json",
				Content: `
{
  "ImportPath": "github.com/nordstrom/kubelogin",
  "GoVersion": "go1.8",
  "GodepVersion": "v79",
  "Packages": [
    "./..."
  ],
  "Deps": [
    {
      "ImportPath": "github.com/beorn7/perks/quantile",
      "Rev": "4c0e84591b9aa9e6dcfdf3e020114cd81f89d5f9"
    },
    {
      "ImportPath": "github.com/coreos/go-oidc",
      "Rev": "d68c0e2fef598f5bbf15edd34905f4bf551a54ec"
    },
    {
      "ImportPath": "github.com/go-redis/redis",
      "Comment": "v6.5.6-2-g7a034e1",
      "Rev": "7a034e1609674d5eb847c3885e5058c54e79a1df"
    },
    {
      "ImportPath": "github.com/go-redis/redis/internal",
      "Comment": "v6.5.6-2-g7a034e1",
      "Rev": "7a034e1609674d5eb847c3885e5058c54e79a1df"
    },
    {
      "ImportPath": "github.com/go-redis/redis/internal/consistenthash",
      "Comment": "v6.5.6-2-g7a034e1",
      "Rev": "7a034e1609674d5eb847c3885e5058c54e79a1df"
    },
    {
      "ImportPath": "github.com/go-redis/redis/internal/hashtag",
      "Comment": "v6.5.6-2-g7a034e1",
      "Rev": "7a034e1609674d5eb847c3885e5058c54e79a1df"
    },
    {
      "ImportPath": "github.com/go-redis/redis/internal/pool",
      "Comment": "v6.5.6-2-g7a034e1",
      "Rev": "7a034e1609674d5eb847c3885e5058c54e79a1df"
    },
    {
      "ImportPath": "github.com/go-redis/redis/internal/proto",
      "Comment": "v6.5.6-2-g7a034e1",
      "Rev": "7a034e1609674d5eb847c3885e5058c54e79a1df"
    },
		{
      "ImportPath": "github.com/golang/protobuf/proto",
      "Rev": "748d386b5c1ea99658fd69fe9f03991ce86a90c1"
    }
	]
}
`,
			}},
			want: `
go_repository(
    name = "com_github_beorn7_perks",
    commit = "4c0e84591b9aa9e6dcfdf3e020114cd81f89d5f9",
    importpath = "github.com/beorn7/perks",
)

go_repository(
    name = "com_github_coreos_go_oidc",
    commit = "d68c0e2fef598f5bbf15edd34905f4bf551a54ec",
    importpath = "github.com/coreos/go-oidc",
)

go_repository(
    name = "com_github_go_redis_redis",
    commit = "7a034e1609674d5eb847c3885e5058c54e79a1df",
    importpath = "github.com/go-redis/redis",
)

go_repository(
    name = "com_github_golang_protobuf",
    commit = "748d386b5c1ea99658fd69fe9f03991ce86a90c1",
    importpath = "github.com/golang/protobuf",
)
`,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			dir, cleanup := testtools.CreateFiles(t, tc.files)
			defer cleanup()

			filename := filepath.Join(dir, tc.files[0].Path)
			cache := repo.NewStubRemoteCache(nil)
			rules, err := repo.ImportRepoRules(filename, cache)
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
