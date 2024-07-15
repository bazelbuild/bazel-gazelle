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
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/language"
	"github.com/bazelbuild/bazel-gazelle/repo"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/bazelbuild/bazel-gazelle/testtools"
)

func TestImports(t *testing.T) {
	for _, tc := range []struct {
		desc, want        string
		wantErr           string
		stubGoModDownload func(string, []string) ([]byte, error)
		stubGoListModules func(string) ([]byte, error)
		files             []testtools.FileSpec
	}{
		{
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
	github.com/fork/go-toml v0.0.0-20190116191733-b6c0e53d7304
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
github.com/fork/go-toml v0.0.0-20190116191733-b6c0e53d7304 h1:5+8j8FTpnFV4nEImW/ofkzEt8VoOiLXxdYIDsB73T38=
github.com/fork/go-toml v0.0.0-20190116191733-b6c0e53d7304/go.mod h1:ZiWeW+zYFKm7srdB9IoDzzZXaJaI5eL9QjNiN/DMA2s=
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
    version = "v0.0.0-20190202002759-027686e28d67",
)

go_repository(
    name = "com_github_burntsushi_toml",
    importpath = "github.com/BurntSushi/toml",
    sum = "h1:WXkYYl6Yr3qBf1K79EBnL4mak0OimBfB0XUf9Vl28OQ=",
    version = "v0.3.1",
)

go_repository(
    name = "com_github_davecgh_go_spew",
    importpath = "github.com/davecgh/go-spew",
    sum = "h1:vj9j/u1bqnvCEfJOwUhtlOARqs3+rkHYY13jYWTU97c=",
    version = "v1.1.1",
)

go_repository(
    name = "com_github_fork_go_toml",
    importpath = "github.com/fork/go-toml",
    sum = "h1:5+8j8FTpnFV4nEImW/ofkzEt8VoOiLXxdYIDsB73T38=",
    version = "v0.0.0-20190116191733-b6c0e53d7304",
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
    version = "v1.0.0-20180628173108-788fd7840127",
)

go_repository(
    name = "in_gopkg_yaml_v2",
    importpath = "gopkg.in/yaml.v2",
    sum = "h1:ZCJp+EgiOT7lHqUV2J862kp8Qj64Jo6az82+3Td9dZw=",
    version = "v2.2.2",
)

go_repository(
    name = "org_golang_x_tools",
    importpath = "golang.org/x/tools",
    sum = "h1:FkAkwuYWQw+IArrnmhGlisKHQF4MsZ2Nu/fX4ttW55o=",
    version = "v0.0.0-20190122202912-9c309ee22fab",
)
`,
			wantErr:           "",
			stubGoModDownload: nil,
		},
		{
			desc: "modules-download-error",
			files: []testtools.FileSpec{
				{
					Path: "go.mod",
					Content: `
module github.com/bazelbuild/bazel-gazelle

require (
	definitely.doesnotexist/ever v0.1.0
)
`,
				}, {
					Path: "go.sum",
					Content: `
definitely.doesnotexist/ever v0.1.0/go.mod h1:HI93XBmqTisBFMUTm0b8Fm+jr3Dg1NNxqwp+5A1VGuJ=
`,
				},
			},
			want:    "",
			wantErr: "finding module sums: error from go mod download: failed to download\nError downloading definitely.doesnotexist/ever: Did not exist",
			stubGoModDownload: func(dir string, args []string) ([]byte, error) {
				return []byte(`{
"Path": "definitely.doesnotexist/ever",
"Version": "0.1.0",
"Error": "Did not exist"
}`), fmt.Errorf("failed to download")
			},
		},
		{
			desc: "modules-download-bad-json",
			files: []testtools.FileSpec{
				{
					Path: "go.mod",
					Content: `
module github.com/bazelbuild/bazel-gazelle

require (
	definitely.doesnotexist/ever v0.1.0
)
`,
				}, {
					Path: "go.sum",
					Content: `
definitely.doesnotexist/ever v0.1.0/go.mod h1:HI93XBmqTisBFMUTm0b8Fm+jr3Dg1NNxqwp+5A1VGuJ=
`,
				},
			},
			want:    "",
			wantErr: "finding module sums: error from go mod download: failed to download\nError parsing module for more error information: invalid character 'o' in literal null (expecting 'u')",
			stubGoModDownload: func(dir string, args []string) ([]byte, error) {
				return []byte(`{
"Path": "definitely.doesnotexist/ever",
"Version": "0.1.0",
"Error": {
   "Err": not valid json
}
}`), fmt.Errorf("failed to download")
			},
		},
		{
			desc: "list-modules-error",
			files: []testtools.FileSpec{
				{
					Path: "go.mod",
					Content: `
module github.com/bazelbuild/bazel-gazelle

require (
	definitely.doesnotexist/ever v0.1.0
)
`,
				}, {
					Path: "go.sum",
					Content: `
definitely.doesnotexist/ever v0.1.0/go.mod h1:HI93XBmqTisBFMUTm0b8Fm+jr3Dg1NNxqwp+5A1VGuJ=
`,
				},
			},
			want:    "",
			wantErr: "error from go list: failed to download\nError listing definitely.doesnotexist/ever: Did not exist",
			stubGoListModules: func(dir string) ([]byte, error) {
				return []byte(`{
"Path": "definitely.doesnotexist/ever",
"Version": "0.1.0",
"Error": {"Err": "Did not exist"}
}`), fmt.Errorf("failed to download")
			},
		},
		{
			desc: "list-modules-bad-json",
			files: []testtools.FileSpec{
				{
					Path: "go.mod",
					Content: `
module github.com/bazelbuild/bazel-gazelle

require (
	definitely.doesnotexist/ever v0.1.0
)
`,
				}, {
					Path: "go.sum",
					Content: `
definitely.doesnotexist/ever v0.1.0/go.mod h1:HI93XBmqTisBFMUTm0b8Fm+jr3Dg1NNxqwp+5A1VGuJ=
`,
				},
			},
			want:    "",
			wantErr: "error from go list: failed to download\nError parsing module for more error information: invalid character 'n' after object key",
			stubGoListModules: func(dir string) ([]byte, error) {
				return []byte(`{
    "Path": "definitely.doesnotexist/ever",
    "Version": "0.1.0",
    "Error" not valid json
}`), fmt.Errorf("failed to download")
			},
		},
		{
			desc: "work",
			files: []testtools.FileSpec{
				{
					Path: "go.work",
					Content: `
go 1.18

use (
	./project1
	./project2
	./project3
)
`,
				},
				{
					Path: "project1/go.mod",
					Content: `
module project1

go 1.18

require github.com/vmware/govmomi v0.21.0 // indirect
`,
				},
				{
					Path: "project1/go.sum",
					Content: `
github.com/davecgh/go-xdr v0.0.0-20161123171359-e6a2ba005892/go.mod h1:CTDl0pzVzE5DEzZhPfvhY/9sPFMQIxaJ9VAMs9AagrE=
github.com/google/uuid v0.0.0-20170306145142-6a5e28554805/go.mod h1:TIyPZe4MgqvfeYDBFedMoGGpEw/LqOeaOT+nhxU+yHo=
github.com/kr/pretty v0.1.0/go.mod h1:dAy3ld7l9f0ibDNOQOHHMYYIIbhfbHSm3C4ZsoJORNo=
github.com/kr/pty v1.1.1/go.mod h1:pFQYn66WHrOpPYNljwOMqo10TkYh1fy3cYio2l3bCsQ=
github.com/kr/text v0.1.0/go.mod h1:4Jbv+DJW3UT/LiOwJeYQe1efqtUx/iVham/4vfdArNI=
github.com/vmware/govmomi v0.21.0 h1:jc8uMuxpcV2xMAA/cnEDlnsIjvqcMra5Y8onh/U3VuY=
github.com/vmware/govmomi v0.21.0/go.mod h1:zbnFoBQ9GIjs2RVETy8CNEpb+L+Lwkjs3XZUL0B3/m0=
github.com/vmware/vmw-guestinfo v0.0.0-20170707015358-25eff159a728/go.mod h1:x9oS4Wk2s2u4tS29nEaDLdzvuHdB19CvSGJjPgkZJNk=
`,
				},
				{
					Path: "project2/go.mod",
					Content: `
module project2

go 1.18

require github.com/vmware/govmomi v0.24.0 // indirect
`,
				},
				{
					Path: "project2/go.sum",
					Content: `
github.com/davecgh/go-xdr v0.0.0-20161123171359-e6a2ba005892/go.mod h1:CTDl0pzVzE5DEzZhPfvhY/9sPFMQIxaJ9VAMs9AagrE=
github.com/google/uuid v0.0.0-20170306145142-6a5e28554805/go.mod h1:TIyPZe4MgqvfeYDBFedMoGGpEw/LqOeaOT+nhxU+yHo=
github.com/kr/pretty v0.1.0/go.mod h1:dAy3ld7l9f0ibDNOQOHHMYYIIbhfbHSm3C4ZsoJORNo=
github.com/kr/pty v1.1.1/go.mod h1:pFQYn66WHrOpPYNljwOMqo10TkYh1fy3cYio2l3bCsQ=
github.com/kr/text v0.1.0/go.mod h1:4Jbv+DJW3UT/LiOwJeYQe1efqtUx/iVham/4vfdArNI=
github.com/vmware/govmomi v0.24.0 h1:G7YFF6unMTG3OY25Dh278fsomVTKs46m2ENlEFSbmbs=
github.com/vmware/govmomi v0.24.0/go.mod h1:Y+Wq4lst78L85Ge/F8+ORXIWiKYqaro1vhAulACy9Lc=
github.com/vmware/vmw-guestinfo v0.0.0-20170707015358-25eff159a728/go.mod h1:x9oS4Wk2s2u4tS29nEaDLdzvuHdB19CvSGJjPgkZJNk=
`,
				},
				{
					Path: "project3/go.mod",
					Content: `
module project3

go 1.18

require github.com/vmware/govmomi v0.27.0 // indirect
`,
				},
				{
					Path: "project3/go.sum",
					Content: `
github.com/a8m/tree v0.0.0-20210115125333-10a5fd5b637d/go.mod h1:FSdwKX97koS5efgm8WevNf7XS3PqtyFkKDDXrz778cg=
github.com/davecgh/go-xdr v0.0.0-20161123171359-e6a2ba005892/go.mod h1:CTDl0pzVzE5DEzZhPfvhY/9sPFMQIxaJ9VAMs9AagrE=
github.com/google/uuid v1.2.0/go.mod h1:TIyPZe4MgqvfeYDBFedMoGGpEw/LqOeaOT+nhxU+yHo=
github.com/kr/pretty v0.1.0/go.mod h1:dAy3ld7l9f0ibDNOQOHHMYYIIbhfbHSm3C4ZsoJORNo=
github.com/kr/pty v1.1.1/go.mod h1:pFQYn66WHrOpPYNljwOMqo10TkYh1fy3cYio2l3bCsQ=
github.com/kr/text v0.1.0/go.mod h1:4Jbv+DJW3UT/LiOwJeYQe1efqtUx/iVham/4vfdArNI=
github.com/vmware/govmomi v0.27.0 h1:KoQ8IsLAa7V78s5d7dgpZA8d039GBM83cVxgAq9uWuw=
github.com/vmware/govmomi v0.27.0/go.mod h1:daTuJEcQosNMXYJOeku0qdBJP9SOLLWB3Mqz8THtv6o=
github.com/vmware/vmw-guestinfo v0.0.0-20170707015358-25eff159a728/go.mod h1:x9oS4Wk2s2u4tS29nEaDLdzvuHdB19CvSGJjPgkZJNk=
`,
				},
			},
			want: `
go_repository(
    name = "com_github_a8m_tree",
    importpath = "github.com/a8m/tree",
    sum = "h1:4E8RufAN3UQ/weB6AnQ4y5miZCO0Yco8ZdGId41WuQs=",
    version = "v0.0.0-20210115125333-10a5fd5b637d",
)

go_repository(
    name = "com_github_davecgh_go_xdr",
    importpath = "github.com/davecgh/go-xdr",
    sum = "h1:qg9VbHo1TlL0KDM0vYvBG9EY0X0Yku5WYIPoFWt8f6o=",
    version = "v0.0.0-20161123171359-e6a2ba005892",
)

go_repository(
    name = "com_github_google_uuid",
    importpath = "github.com/google/uuid",
    sum = "h1:qJYtXnJRWmpe7m/3XlyhrsLrEURqHRM2kxzoxXqyUDs=",
    version = "v1.2.0",
)

go_repository(
    name = "com_github_kr_pretty",
    importpath = "github.com/kr/pretty",
    sum = "h1:L/CwN0zerZDmRFUapSPitk6f+Q3+0za1rQkzVuMiMFI=",
    version = "v0.1.0",
)

go_repository(
    name = "com_github_kr_pty",
    importpath = "github.com/kr/pty",
    sum = "h1:VkoXIwSboBpnk99O/KFauAEILuNHv5DVFKZMBN/gUgw=",
    version = "v1.1.1",
)

go_repository(
    name = "com_github_kr_text",
    importpath = "github.com/kr/text",
    sum = "h1:45sCR5RtlFHMR4UwH9sdQ5TC8v0qDQCHnXt+kaKSTVE=",
    version = "v0.1.0",
)

go_repository(
    name = "com_github_vmware_govmomi",
    importpath = "github.com/vmware/govmomi",
    sum = "h1:KoQ8IsLAa7V78s5d7dgpZA8d039GBM83cVxgAq9uWuw=",
    version = "v0.27.0",
)

go_repository(
    name = "com_github_vmware_vmw_guestinfo",
    importpath = "github.com/vmware/vmw-guestinfo",
    sum = "h1:sH9mEk+flyDxiUa5BuPiuhDETMbzrt9A20I2wktMvRQ=",
    version = "v0.0.0-20170707015358-25eff159a728",
)
`,
			wantErr: "",
			stubGoModDownload: func(s string, i []string) ([]byte, error) {
				return []byte(`
{
	"Path": "github.com/a8m/tree",
	"Version": "v0.0.0-20210115125333-10a5fd5b637d",
	"Info": "/Users/hhalil/go/pkg/mod/cache/download/github.com/a8m/tree/@v/v0.0.0-20210115125333-10a5fd5b637d.info",
	"GoMod": "/Users/hhalil/go/pkg/mod/cache/download/github.com/a8m/tree/@v/v0.0.0-20210115125333-10a5fd5b637d.mod",
	"Zip": "/Users/hhalil/go/pkg/mod/cache/download/github.com/a8m/tree/@v/v0.0.0-20210115125333-10a5fd5b637d.zip",
	"Dir": "/Users/hhalil/go/pkg/mod/github.com/a8m/tree@v0.0.0-20210115125333-10a5fd5b637d",
	"Sum": "h1:4E8RufAN3UQ/weB6AnQ4y5miZCO0Yco8ZdGId41WuQs=",
	"GoModSum": "h1:FSdwKX97koS5efgm8WevNf7XS3PqtyFkKDDXrz778cg="
}
{
	"Path": "github.com/davecgh/go-xdr",
	"Version": "v0.0.0-20161123171359-e6a2ba005892",
	"Info": "/Users/hhalil/go/pkg/mod/cache/download/github.com/davecgh/go-xdr/@v/v0.0.0-20161123171359-e6a2ba005892.info",
	"GoMod": "/Users/hhalil/go/pkg/mod/cache/download/github.com/davecgh/go-xdr/@v/v0.0.0-20161123171359-e6a2ba005892.mod",
	"Zip": "/Users/hhalil/go/pkg/mod/cache/download/github.com/davecgh/go-xdr/@v/v0.0.0-20161123171359-e6a2ba005892.zip",
	"Dir": "/Users/hhalil/go/pkg/mod/github.com/davecgh/go-xdr@v0.0.0-20161123171359-e6a2ba005892",
	"Sum": "h1:qg9VbHo1TlL0KDM0vYvBG9EY0X0Yku5WYIPoFWt8f6o=",
	"GoModSum": "h1:CTDl0pzVzE5DEzZhPfvhY/9sPFMQIxaJ9VAMs9AagrE="
}
{
	"Path": "github.com/google/uuid",
	"Version": "v1.2.0",
	"Info": "/Users/hhalil/go/pkg/mod/cache/download/github.com/google/uuid/@v/v1.2.0.info",
	"GoMod": "/Users/hhalil/go/pkg/mod/cache/download/github.com/google/uuid/@v/v1.2.0.mod",
	"Zip": "/Users/hhalil/go/pkg/mod/cache/download/github.com/google/uuid/@v/v1.2.0.zip",
	"Dir": "/Users/hhalil/go/pkg/mod/github.com/google/uuid@v1.2.0",
	"Sum": "h1:qJYtXnJRWmpe7m/3XlyhrsLrEURqHRM2kxzoxXqyUDs=",
	"GoModSum": "h1:TIyPZe4MgqvfeYDBFedMoGGpEw/LqOeaOT+nhxU+yHo="
}
{
	"Path": "github.com/kr/pretty",
	"Version": "v0.1.0",
	"Info": "/Users/hhalil/go/pkg/mod/cache/download/github.com/kr/pretty/@v/v0.1.0.info",
	"GoMod": "/Users/hhalil/go/pkg/mod/cache/download/github.com/kr/pretty/@v/v0.1.0.mod",
	"Zip": "/Users/hhalil/go/pkg/mod/cache/download/github.com/kr/pretty/@v/v0.1.0.zip",
	"Dir": "/Users/hhalil/go/pkg/mod/github.com/kr/pretty@v0.1.0",
	"Sum": "h1:L/CwN0zerZDmRFUapSPitk6f+Q3+0za1rQkzVuMiMFI=",
	"GoModSum": "h1:dAy3ld7l9f0ibDNOQOHHMYYIIbhfbHSm3C4ZsoJORNo="
}
{
	"Path": "github.com/kr/pty",
	"Version": "v1.1.1",
	"Info": "/Users/hhalil/go/pkg/mod/cache/download/github.com/kr/pty/@v/v1.1.1.info",
	"GoMod": "/Users/hhalil/go/pkg/mod/cache/download/github.com/kr/pty/@v/v1.1.1.mod",
	"Zip": "/Users/hhalil/go/pkg/mod/cache/download/github.com/kr/pty/@v/v1.1.1.zip",
	"Dir": "/Users/hhalil/go/pkg/mod/github.com/kr/pty@v1.1.1",
	"Sum": "h1:VkoXIwSboBpnk99O/KFauAEILuNHv5DVFKZMBN/gUgw=",
	"GoModSum": "h1:pFQYn66WHrOpPYNljwOMqo10TkYh1fy3cYio2l3bCsQ="
}
{
	"Path": "github.com/kr/text",
	"Version": "v0.1.0",
	"Info": "/Users/hhalil/go/pkg/mod/cache/download/github.com/kr/text/@v/v0.1.0.info",
	"GoMod": "/Users/hhalil/go/pkg/mod/cache/download/github.com/kr/text/@v/v0.1.0.mod",
	"Zip": "/Users/hhalil/go/pkg/mod/cache/download/github.com/kr/text/@v/v0.1.0.zip",
	"Dir": "/Users/hhalil/go/pkg/mod/github.com/kr/text@v0.1.0",
	"Sum": "h1:45sCR5RtlFHMR4UwH9sdQ5TC8v0qDQCHnXt+kaKSTVE=",
	"GoModSum": "h1:4Jbv+DJW3UT/LiOwJeYQe1efqtUx/iVham/4vfdArNI="
}
{
	"Path": "github.com/vmware/govmomi",
	"Version": "v0.27.0",
	"Info": "/Users/hhalil/go/pkg/mod/cache/download/github.com/vmware/govmomi/@v/v0.27.0.info",
	"GoMod": "/Users/hhalil/go/pkg/mod/cache/download/github.com/vmware/govmomi/@v/v0.27.0.mod",
	"Zip": "/Users/hhalil/go/pkg/mod/cache/download/github.com/vmware/govmomi/@v/v0.27.0.zip",
	"Dir": "/Users/hhalil/go/pkg/mod/github.com/vmware/govmomi@v0.27.0",
	"Sum": "h1:KoQ8IsLAa7V78s5d7dgpZA8d039GBM83cVxgAq9uWuw=",
	"GoModSum": "h1:daTuJEcQosNMXYJOeku0qdBJP9SOLLWB3Mqz8THtv6o="
}
{
	"Path": "github.com/vmware/vmw-guestinfo",
	"Version": "v0.0.0-20170707015358-25eff159a728",
	"Info": "/Users/hhalil/go/pkg/mod/cache/download/github.com/vmware/vmw-guestinfo/@v/v0.0.0-20170707015358-25eff159a728.info",
	"GoMod": "/Users/hhalil/go/pkg/mod/cache/download/github.com/vmware/vmw-guestinfo/@v/v0.0.0-20170707015358-25eff159a728.mod",
	"Zip": "/Users/hhalil/go/pkg/mod/cache/download/github.com/vmware/vmw-guestinfo/@v/v0.0.0-20170707015358-25eff159a728.zip",
	"Dir": "/Users/hhalil/go/pkg/mod/github.com/vmware/vmw-guestinfo@v0.0.0-20170707015358-25eff159a728",
	"Sum": "h1:sH9mEk+flyDxiUa5BuPiuhDETMbzrt9A20I2wktMvRQ=",
	"GoModSum": "h1:x9oS4Wk2s2u4tS29nEaDLdzvuHdB19CvSGJjPgkZJNk="
}
`), nil
			},
			stubGoListModules: func(dir string) ([]byte, error) {
				return []byte(`
{
        "Path": "project1",
        "Main": true,
        "Dir": "/Users/hhalil/Documents/VMware/mirrors_github_bazel-gazelle/hakan/project1",
        "GoMod": "/Users/hhalil/Documents/VMware/mirrors_github_bazel-gazelle/hakan/project1/go.mod",
        "GoVersion": "1.18"
}
{
        "Path": "project2",
        "Main": true,
        "Dir": "/Users/hhalil/Documents/VMware/mirrors_github_bazel-gazelle/hakan/project2",
        "GoMod": "/Users/hhalil/Documents/VMware/mirrors_github_bazel-gazelle/hakan/project2/go.mod",
        "GoVersion": "1.18"
}
{
        "Path": "project3",
        "Main": true,
        "Dir": "/Users/hhalil/Documents/VMware/mirrors_github_bazel-gazelle/hakan/project3",
        "GoMod": "/Users/hhalil/Documents/VMware/mirrors_github_bazel-gazelle/hakan/project3/go.mod",
        "GoVersion": "1.18"
}
{
        "Path": "github.com/a8m/tree",
        "Version": "v0.0.0-20210115125333-10a5fd5b637d",
        "Time": "2021-01-15T12:53:33Z",
        "Indirect": true,
        "GoMod": "/Users/hhalil/go/pkg/mod/cache/download/github.com/a8m/tree/@v/v0.0.0-20210115125333-10a5fd5b637d.mod"
}
{
        "Path": "github.com/davecgh/go-xdr",
        "Version": "v0.0.0-20161123171359-e6a2ba005892",
        "Time": "2016-11-23T17:13:59Z",
        "Indirect": true,
        "GoMod": "/Users/hhalil/go/pkg/mod/cache/download/github.com/davecgh/go-xdr/@v/v0.0.0-20161123171359-e6a2ba005892.mod"
}
{
        "Path": "github.com/google/uuid",
        "Version": "v1.2.0",
        "Time": "2021-01-22T18:20:15Z",
        "Indirect": true,
        "GoMod": "/Users/hhalil/go/pkg/mod/cache/download/github.com/google/uuid/@v/v1.2.0.mod"
}
{
        "Path": "github.com/kr/pretty",
        "Version": "v0.1.0",
        "Time": "2018-05-06T08:33:45Z",
        "Indirect": true,
        "GoMod": "/Users/hhalil/go/pkg/mod/cache/download/github.com/kr/pretty/@v/v0.1.0.mod"
}
{
        "Path": "github.com/kr/pty",
        "Version": "v1.1.1",
        "Time": "2018-01-13T18:08:13Z",
        "Indirect": true,
        "GoMod": "/Users/hhalil/go/pkg/mod/cache/download/github.com/kr/pty/@v/v1.1.1.mod"
}
{
        "Path": "github.com/kr/text",
        "Version": "v0.1.0",
        "Time": "2018-05-06T08:24:08Z",
        "Indirect": true,
        "GoMod": "/Users/hhalil/go/pkg/mod/cache/download/github.com/kr/text/@v/v0.1.0.mod"
}
{
        "Path": "github.com/vmware/govmomi",
        "Version": "v0.27.0",
        "Time": "2021-10-14T20:30:09Z",
        "Indirect": true,
        "Dir": "/Users/hhalil/go/pkg/mod/github.com/vmware/govmomi@v0.27.0",
        "GoMod": "/Users/hhalil/go/pkg/mod/cache/download/github.com/vmware/govmomi/@v/v0.27.0.mod",
        "GoVersion": "1.14"
}
{
        "Path": "github.com/vmware/vmw-guestinfo",
        "Version": "v0.0.0-20170707015358-25eff159a728",
        "Time": "2017-07-07T01:53:58Z",
        "Indirect": true,
        "GoMod": "/Users/hhalil/go/pkg/mod/cache/download/github.com/vmware/vmw-guestinfo/@v/v0.0.0-20170707015358-25eff159a728.mod"
}
`), nil
			},
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.stubGoModDownload != nil {
				previousGoModDownload := goModDownload
				goModDownload = tc.stubGoModDownload
				defer func() { goModDownload = previousGoModDownload }()
			}
			if tc.stubGoListModules != nil {
				previousGoListModules := goListModules
				goListModules = tc.stubGoListModules
				defer func() { goListModules = previousGoListModules }()
			}
			dir, cleanup := testtools.CreateFiles(t, tc.files)
			defer cleanup()

			filename := filepath.Join(dir, tc.files[0].Path)
			c := &config.Config{Exts: map[string]interface{}{}}
			rc, rcCleanup := repo.NewRemoteCache(nil)
			defer func() {
				if err := rcCleanup(); err != nil {
					t.Fatal(err)
				}
			}()
			gl := NewLanguage()
			gl.Configure(c, "", nil)
			importer := gl.(language.RepoImporter)
			result := importer.ImportRepos(language.ImportReposArgs{
				Config: c,
				Path:   filename,
				Cache:  rc,
			})
			if tc.wantErr != "" {
				if result.Error == nil {
					t.Fatalf("Want error %v but got %v", tc.wantErr, result)
				}
				if result.Error.Error() != tc.wantErr {
					t.Fatalf("Want error %v but got %v", tc.wantErr, result.Error)
				}
				return
			} else {
				if result.Error != nil {
					t.Fatal(result.Error)
				}
			}
			f := rule.EmptyFile("test", "")
			for _, r := range result.Gen {
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

func TestImportsParseOnly(t *testing.T) {
	for _, tc := range []struct {
		desc    string
		want    string
		wantErr string
		files   []testtools.FileSpec
	}{
		{
			desc: "base_case",
			files: []testtools.FileSpec{
				{
					Path: "go.mod",
					Content: `
module foo

go 1.22.5

require github.com/BurntSushi/toml v0.3.1`,
				},
				{
					Path: "go.sum",
					Content: `
github.com/BurntSushi/toml v0.3.1 h1:WXkYYl6Yr3qBf1K79EBnL4mak0OimBfB0XUf9Vl28OQ=
github.com/BurntSushi/toml v0.3.1/go.mod h1:xHWCNGjB5oqiDr8zfno3MHue2Ht5sIBksp03qcyfWMU=`,
				},
				{
					Path: "main.go",
					Content: `
package main

import (
	_ "github.com/BurntSushi/toml"
)`,
				},
			},
			want: `
go_repository(
    name = "com_github_burntsushi_toml",
    importpath = "github.com/BurntSushi/toml",
    sum = "h1:WXkYYl6Yr3qBf1K79EBnL4mak0OimBfB0XUf9Vl28OQ=",
    version = "v0.3.1",
)`,
		},
		{
			desc: "with_replace",
			files: []testtools.FileSpec{
				{
					Path: "go.mod",
					Content: `
module foo

go 1.22.5

replace github.com/throttled/throttled/v2 => github.com/buildbuddy-io/throttled/v2 v2.9.1-rc2

require github.com/throttled/throttled/v2 v2.12.0

require github.com/stretchr/testify v1.8.0 // indirect`,
				},
				{
					Path: "go.sum",
					Content: `
github.com/buildbuddy-io/throttled/v2 v2.9.1-rc2 h1:l9PGL9DJwcCgQcVt/zFzVjJbSzb+BmpU8NBeo6leHKU=
github.com/buildbuddy-io/throttled/v2 v2.9.1-rc2/go.mod h1:LSVJkC18NVPon/lADUB4AabAylh2dnsZbk4SkMCk4KQ=
github.com/cespare/xxhash/v2 v2.1.1/go.mod h1:VGX0DQ3Q6kWi7AoAeZDth3/j3BFtOZR5XLFGgcrjCOs=
github.com/davecgh/go-spew v1.1.0/go.mod h1:J7Y8YcW2NihsgmVo/mv3lAwl/skON4iLHjSsI+c5H38=
github.com/davecgh/go-spew v1.1.1 h1:vj9j/u1bqnvCEfJOwUhtlOARqs3+rkHYY13jYWTU97c=
github.com/davecgh/go-spew v1.1.1/go.mod h1:J7Y8YcW2NihsgmVo/mv3lAwl/skON4iLHjSsI+c5H38=
github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f/go.mod h1:cuUVRXasLTGF7a8hSLbxyZXjz+1KgoB3wDUb6vlszIc=
github.com/fsnotify/fsnotify v1.4.7/go.mod h1:jwhsz4b93w/PPRr/qN1Yymfu8t87LnFCMoQvtojpjFo=
github.com/fsnotify/fsnotify v1.4.9/go.mod h1:znqG4EE+3YCdAaPaxE2ZRY/06pZUdp0tY4IgpuI1SZQ=
github.com/go-redis/redis v6.15.8+incompatible/go.mod h1:NAIEuMOZ/fxfXJIrKDQDz8wamY7mA7PouImQ2Jvg6kA=
github.com/go-redis/redis/v8 v8.4.2/go.mod h1:A1tbYoHSa1fXwN+//ljcCYYJeLmVrwL9hbQN45Jdy0M=
github.com/golang/protobuf v1.2.0/go.mod h1:6lQm79b+lXiMfvg/cZm0SGofjICqVBUtrP5yJMmIC1U=
github.com/golang/protobuf v1.4.0-rc.1/go.mod h1:ceaxUfeHdC40wWswd/P6IGgMaK3YpKi5j83Wpe3EHw8=
github.com/golang/protobuf v1.4.0-rc.1.0.20200221234624-67d41d38c208/go.mod h1:xKAWHe0F5eneWXFV3EuXVDTCmh+JuBKY0li0aMyXATA=
github.com/golang/protobuf v1.4.0-rc.2/go.mod h1:LlEzMj4AhA7rCAGe4KMBDvJI+AwstrUpVNzEA03Pprs=
github.com/golang/protobuf v1.4.0-rc.4.0.20200313231945-b860323f09d0/go.mod h1:WU3c8KckQ9AFe+yFwt9sWVRKCVIyN9cPHBJSNnbL67w=
github.com/golang/protobuf v1.4.0/go.mod h1:jodUvKwWbYaEsadDk5Fwe5c77LiNKVO9IDvqG2KuDX0=
github.com/golang/protobuf v1.4.2/go.mod h1:oDoupMAO8OvCJWAcko0GGGIgR6R6ocIYbsSw735rRwI=
github.com/gomodule/redigo v2.0.0+incompatible h1:K/R+8tc58AaqLkqG2Ol3Qk+DR/TlNuhuh457pBFPtt0=
github.com/gomodule/redigo v2.0.0+incompatible/go.mod h1:B4C85qUVwatsJoIUNIfCRsp7qO0iAmpGFZ4EELWSbC4=
github.com/google/go-cmp v0.3.0/go.mod h1:8QqcDgzrUqlUb/G2PQTWiueGozuR1884gddMywk6iLU=
github.com/google/go-cmp v0.3.1/go.mod h1:8QqcDgzrUqlUb/G2PQTWiueGozuR1884gddMywk6iLU=
github.com/google/go-cmp v0.4.0/go.mod h1:v8dTdLbMG2kIc/vJvl+f65V22dbkXbowE6jgT/gNBxE=
github.com/google/go-cmp v0.5.3/go.mod h1:v8dTdLbMG2kIc/vJvl+f65V22dbkXbowE6jgT/gNBxE=
github.com/hashicorp/golang-lru v0.5.4 h1:YDjusn29QI/Das2iO9M0BHnIbxPeyuCHsjMW+lJfyTc=
github.com/hashicorp/golang-lru v0.5.4/go.mod h1:iADmTwqILo4mZ8BN3D2Q6+9jd8WM5uGBxy+E8yxSoD4=
github.com/hpcloud/tail v1.0.0/go.mod h1:ab1qPbhIpdTxEkNHXyeSf5vhxWSCs/tWer42PpOxQnU=
github.com/kr/pretty v0.1.0/go.mod h1:dAy3ld7l9f0ibDNOQOHHMYYIIbhfbHSm3C4ZsoJORNo=
github.com/kr/pty v1.1.1/go.mod h1:pFQYn66WHrOpPYNljwOMqo10TkYh1fy3cYio2l3bCsQ=
github.com/kr/text v0.1.0/go.mod h1:4Jbv+DJW3UT/LiOwJeYQe1efqtUx/iVham/4vfdArNI=
github.com/nxadm/tail v1.4.4/go.mod h1:kenIhsEOeOJmVchQTgglprH7qJGnHDVpk1VPCcaMI8A=
github.com/onsi/ginkgo v1.6.0/go.mod h1:lLunBs/Ym6LB5Z9jYTR76FiuTmxDTDusOGeTQH+WWjE=
github.com/onsi/ginkgo v1.12.1/go.mod h1:zj2OWP4+oCPe1qIXoGWkgMRwljMUYCdkwsT2108oapk=
github.com/onsi/ginkgo v1.14.2/go.mod h1:iSB4RoI2tjJc9BBv4NKIKWKya62Rps+oPG/Lv9klQyY=
github.com/onsi/gomega v1.7.1/go.mod h1:XdKZgCCFLUoM/7CFJVPcG8C1xQ1AJ0vpAezJrB7JYyY=
github.com/onsi/gomega v1.10.1/go.mod h1:iN09h71vgCQne3DLsj+A5owkum+a2tYe+TOCB1ybHNo=
github.com/onsi/gomega v1.10.3/go.mod h1:V9xEwhxec5O8UDM77eCW8vLymOMltsqPVYWrpDsH8xc=
github.com/pmezard/go-difflib v1.0.0 h1:4DBwDE0NGyQoBHbLQYPwSUPoCMWR5BEzIk/f1lZbAQM=
github.com/pmezard/go-difflib v1.0.0/go.mod h1:iKH77koFhYxTK1pcRnkKkqfTogsbg7gZNVY4sRDYZ/4=
github.com/stretchr/objx v0.1.0/go.mod h1:HFkY916IF+rwdDfMAkV7OtwuqBVzrE8GR6GFx+wExME=
github.com/stretchr/objx v0.4.0/go.mod h1:YvHI0jy2hoMjB+UWwv71VJQ9isScKT/TqJzVSSt89Yw=
github.com/stretchr/testify v1.6.1/go.mod h1:6Fq8oRcR53rry900zMqJjRRixrwX3KX962/h/Wwjteg=
github.com/stretchr/testify v1.7.1/go.mod h1:6Fq8oRcR53rry900zMqJjRRixrwX3KX962/h/Wwjteg=
github.com/stretchr/testify v1.8.0 h1:pSgiaMZlXftHpm5L7V1+rVB+AZJydKsMxsQBIJw4PKk=
github.com/stretchr/testify v1.8.0/go.mod h1:yNjHg4UonilssWZ8iaSj1OCr/vHnekPRkoO+kdMU+MU=
go.opentelemetry.io/otel v0.14.0/go.mod h1:vH5xEuwy7Rts0GNtsCW3HYQoZDY+OmBJ6t1bFGGlxgw=
golang.org/x/crypto v0.0.0-20190308221718-c2843e01d9a2/go.mod h1:djNgcEr1/C05ACkg1iLfiJU5Ep61QUkGW8qpdssI0+w=
golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9/go.mod h1:LzIPMQfyMNhhGPhUkYOs5KpL4U8rLKemX1yGLhDgUto=
golang.org/x/net v0.0.0-20180906233101-161cd47e91fd/go.mod h1:mL1N/T3taQHkDXs73rZJwtUhF3w3ftmwwsq0BUmARs4=
golang.org/x/net v0.0.0-20190404232315-eb5bcb51f2a3/go.mod h1:t9HGtf8HONx5eT2rtn7q6eTqICYqUVnKs3thJo3Qplg=
golang.org/x/net v0.0.0-20200520004742-59133d7f0dd7/go.mod h1:qpuaurCH72eLCgpAm/N6yyVIVM9cpaDIP3A8BGJEC5A=
golang.org/x/net v0.0.0-20201006153459-a7d1128ccaa0/go.mod h1:sp8m0HH+o8qH0wwXwYZr8TS3Oi6o0r6Gce1SSxlDquU=
golang.org/x/sync v0.0.0-20180314180146-1d60e4601c6f/go.mod h1:RxMgew5VJxzue5/jJTE5uejpjVlOe/izrB70Jof72aM=
golang.org/x/sys v0.0.0-20180909124046-d0be0721c37e/go.mod h1:STP8DvDyc/dI5b8T5hshtkjS+E42TnysNCUPdjciGhY=
golang.org/x/sys v0.0.0-20190215142949-d0b11bdaac8a/go.mod h1:STP8DvDyc/dI5b8T5hshtkjS+E42TnysNCUPdjciGhY=
golang.org/x/sys v0.0.0-20190412213103-97732733099d/go.mod h1:h1NjWce9XRLGQEsW7wpKNCjG9DtNlClVuFLEZdDNbEs=
golang.org/x/sys v0.0.0-20190904154756-749cb33beabd/go.mod h1:h1NjWce9XRLGQEsW7wpKNCjG9DtNlClVuFLEZdDNbEs=
golang.org/x/sys v0.0.0-20191005200804-aed5e4c7ecf9/go.mod h1:h1NjWce9XRLGQEsW7wpKNCjG9DtNlClVuFLEZdDNbEs=
golang.org/x/sys v0.0.0-20191120155948-bd437916bb0e/go.mod h1:h1NjWce9XRLGQEsW7wpKNCjG9DtNlClVuFLEZdDNbEs=
golang.org/x/sys v0.0.0-20200323222414-85ca7c5b95cd/go.mod h1:h1NjWce9XRLGQEsW7wpKNCjG9DtNlClVuFLEZdDNbEs=
golang.org/x/sys v0.0.0-20200519105757-fe76b779f299/go.mod h1:h1NjWce9XRLGQEsW7wpKNCjG9DtNlClVuFLEZdDNbEs=
golang.org/x/sys v0.0.0-20200930185726-fdedc70b468f/go.mod h1:h1NjWce9XRLGQEsW7wpKNCjG9DtNlClVuFLEZdDNbEs=
golang.org/x/text v0.3.0/go.mod h1:NqM8EUOU14njkJ3fqMW+pc6Ldnwhi/IjpwHt7yyuwOQ=
golang.org/x/text v0.3.2/go.mod h1:bEr9sfX3Q8Zfm5fL9x+3itogRgK3+ptLWKqgva+5dAk=
golang.org/x/text v0.3.3/go.mod h1:5Zoc/QRtKVWzQhOtBMvqHzDpF6irO9z98xDceosuGiQ=
golang.org/x/text v0.3.7/go.mod h1:u+2+/6zg+i71rQMx5EYifcz6MCKuco9NR6JIITiCfzQ=
golang.org/x/tools v0.0.0-20180917221912-90fa682c2a6e/go.mod h1:n7NCudcB/nEzxVGmLbDWY5pfWTLqBcC2KZ6jyYvM4mQ=
golang.org/x/xerrors v0.0.0-20191204190536-9bdfabe68543/go.mod h1:I/5z698sn9Ka8TeJc9MKroUUfqBBauWjQqLJ2OPfmY0=
google.golang.org/protobuf v0.0.0-20200109180630-ec00e32a8dfd/go.mod h1:DFci5gLYBciE7Vtevhsrf46CRTquxDuWsQurQQe4oz8=
google.golang.org/protobuf v0.0.0-20200221191635-4d8936d0db64/go.mod h1:kwYJMbMJ01Woi6D6+Kah6886xMZcty6N08ah7+eCXa0=
google.golang.org/protobuf v0.0.0-20200228230310-ab0ca4ff8a60/go.mod h1:cfTl7dwQJ+fmap5saPgwCLgHXTUD7jkjRqWcaiX5VyM=
google.golang.org/protobuf v1.20.1-0.20200309200217-e05f789c0967/go.mod h1:A+miEFZTKqfCUM6K7xSMQL9OKL/b6hQv+e19PK+JZNE=
google.golang.org/protobuf v1.21.0/go.mod h1:47Nbq4nVaFHyn7ilMalzfO3qCViNmqZ2kzikPIcrTAo=
google.golang.org/protobuf v1.23.0/go.mod h1:EGpADcykh3NcUnDUJcl1+ZksZNG86OlYog2l/sGQquU=
gopkg.in/check.v1 v0.0.0-20161208181325-20d25e280405/go.mod h1:Co6ibVJAznAaIkqp8huTwlJQCZ016jof/cbN4VW5Yz0=
gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15/go.mod h1:Co6ibVJAznAaIkqp8huTwlJQCZ016jof/cbN4VW5Yz0=
gopkg.in/fsnotify.v1 v1.4.7/go.mod h1:Tz8NjZHkW78fSQdbUxIjBTcgA1z1m8ZHf0WmKUhAMys=
gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7/go.mod h1:dt/ZhP58zS4L8KSrWDmTeBkI65Dw0HsyUHuEVlX15mw=
gopkg.in/yaml.v2 v2.2.4/go.mod h1:hI93XBmqTisBFMUTm0b8Fm+jr3Dg1NNxqwp+5A1VGuI=
gopkg.in/yaml.v2 v2.3.0/go.mod h1:hI93XBmqTisBFMUTm0b8Fm+jr3Dg1NNxqwp+5A1VGuI=
gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c/go.mod h1:K4uyk7z7BCEPqu6E+C64Yfv1cQ7kz7rIZviUmN+EgEM=
gopkg.in/yaml.v3 v3.0.1 h1:fxVm/GzAzEWqLHuvctI91KS9hhNmmWOoWu0XTYJS7CA=
gopkg.in/yaml.v3 v3.0.1/go.mod h1:K4uyk7z7BCEPqu6E+C64Yfv1cQ7kz7rIZviUmN+EgEM=`,
				},
				{
					Path: "main.go",
					Content: `
package main

import (
	_ "github.com/throttled/throttled/v2"
)`,
				},
			},
			want: `
go_repository(
    name = "com_github_stretchr_testify",
    importpath = "github.com/stretchr/testify",
    sum = "h1:pSgiaMZlXftHpm5L7V1+rVB+AZJydKsMxsQBIJw4PKk=",
    version = "v1.8.0",
)

go_repository(
    name = "com_github_throttled_throttled_v2",
    importpath = "github.com/throttled/throttled/v2",
    replace = "github.com/buildbuddy-io/throttled/v2",
    sum = "h1:l9PGL9DJwcCgQcVt/zFzVjJbSzb+BmpU8NBeo6leHKU=",
    version = "v2.9.1-rc2",
)`,
		},
		{
			desc: "with_version_in_replace",
			files: []testtools.FileSpec{
				{
					Path: "go.mod",
					Content: `
module foo

go 1.22.5

replace github.com/crewjam/saml v0.4.14 => github.com/grafana/saml v0.4.15-0.20240523142256-cc370b98af7c

require github.com/crewjam/saml v0.4.14

require (
	github.com/beevik/etree v1.2.0 // indirect
	github.com/jonboulle/clockwork v0.2.2 // indirect
	github.com/mattermost/xml-roundtrip-validator v0.1.0 // indirect
	github.com/russellhaering/goxmldsig v1.4.0 // indirect
	golang.org/x/crypto v0.14.0 // indirect
)`,
				},
				{
					Path: "go.sum",
					Content: `
github.com/beevik/etree v1.1.0/go.mod h1:r8Aw8JqVegEf0w2fDnATrX9VpkMcyFeM0FhwO62wh+A=
github.com/beevik/etree v1.2.0 h1:l7WETslUG/T+xOPs47dtd6jov2Ii/8/OjCldk5fYfQw=
github.com/beevik/etree v1.2.0/go.mod h1:aiPf89g/1k3AShMVAzriilpcE4R/Vuor90y83zVZWFc=
github.com/creack/pty v1.1.9/go.mod h1:oKZEueFk5CKHvIhNR5MUki03XCEU+Q6VDXinZuGJ33E=
github.com/davecgh/go-spew v1.1.0/go.mod h1:J7Y8YcW2NihsgmVo/mv3lAwl/skON4iLHjSsI+c5H38=
github.com/davecgh/go-spew v1.1.1 h1:vj9j/u1bqnvCEfJOwUhtlOARqs3+rkHYY13jYWTU97c=
github.com/davecgh/go-spew v1.1.1/go.mod h1:J7Y8YcW2NihsgmVo/mv3lAwl/skON4iLHjSsI+c5H38=
github.com/golang-jwt/jwt/v4 v4.5.0 h1:7cYmW1XlMY7h7ii7UhUyChSgS5wUJEnm9uZVTGqOWzg=
github.com/golang-jwt/jwt/v4 v4.5.0/go.mod h1:m21LjoU+eqJr34lmDMbreY2eSTRJ1cv77w39/MY0Ch0=
github.com/google/go-cmp v0.6.0 h1:ofyhxvXcZhMsU5ulbFiLKl/XBFqE1GSq7atu8tAmTRI=
github.com/google/go-cmp v0.6.0/go.mod h1:17dUlkBOakJ0+DkrSSNjCkIjxS6bF9zb3elmeNGIjoY=
github.com/grafana/saml v0.4.15-0.20240523142256-cc370b98af7c h1:SWmG1QLZ36Ay0htq4Wt3dzlNIhWvQ3GUf7mk19dR8nI=
github.com/grafana/saml v0.4.15-0.20240523142256-cc370b98af7c/go.mod h1:S4+611dxnKt8z/ulbvaJzcgSHsuhjVc1QHNTcr1R7Fw=
github.com/jonboulle/clockwork v0.2.2 h1:UOGuzwb1PwsrDAObMuhUnj0p5ULPj8V/xJ7Kx9qUBdQ=
github.com/jonboulle/clockwork v0.2.2/go.mod h1:Pkfl5aHPm1nk2H9h0bjmnJD/BcgbGXUBGnn1kMkgxc8=
github.com/kr/pretty v0.1.0/go.mod h1:dAy3ld7l9f0ibDNOQOHHMYYIIbhfbHSm3C4ZsoJORNo=
github.com/kr/pretty v0.2.1/go.mod h1:ipq/a2n7PKx3OHsz4KJII5eveXtPO4qwEXGdVfWzfnI=
github.com/kr/pretty v0.3.0/go.mod h1:640gp4NfQd8pI5XOwp5fnNeVWj67G7CFk/SaSQn7NBk=
github.com/kr/pty v1.1.1/go.mod h1:pFQYn66WHrOpPYNljwOMqo10TkYh1fy3cYio2l3bCsQ=
github.com/kr/text v0.1.0/go.mod h1:4Jbv+DJW3UT/LiOwJeYQe1efqtUx/iVham/4vfdArNI=
github.com/kr/text v0.2.0/go.mod h1:eLer722TekiGuMkidMxC/pM04lWEeraHUUmBw8l2grE=
github.com/mattermost/xml-roundtrip-validator v0.1.0 h1:RXbVD2UAl7A7nOTR4u7E3ILa4IbtvKBHw64LDsmu9hU=
github.com/mattermost/xml-roundtrip-validator v0.1.0/go.mod h1:qccnGMcpgwcNaBnxqpJpWWUiPNr5H3O8eDgGV9gT5To=
github.com/pkg/diff v0.0.0-20210226163009-20ebb0f2a09e/go.mod h1:pJLUxLENpZxwdsKMEsNbx1VGcRFpLqf3715MtcvvzbA=
github.com/pkg/errors v0.9.1 h1:FEBLx1zS214owpjy7qsBeixbURkuhQAwrK5UwLGTwt4=
github.com/pkg/errors v0.9.1/go.mod h1:bwawxfHBFNV+L2hUp1rHADufV3IMtnDRdf1r5NINEl0=
github.com/pmezard/go-difflib v1.0.0 h1:4DBwDE0NGyQoBHbLQYPwSUPoCMWR5BEzIk/f1lZbAQM=
github.com/pmezard/go-difflib v1.0.0/go.mod h1:iKH77koFhYxTK1pcRnkKkqfTogsbg7gZNVY4sRDYZ/4=
github.com/rogpeppe/go-internal v1.6.1/go.mod h1:xXDCJY+GAPziupqXw64V24skbSoqbTEfhy4qGm1nDQc=
github.com/rogpeppe/go-internal v1.8.0/go.mod h1:WmiCO8CzOY8rg0OYDC4/i/2WRWAB6poM+XZ2dLUbcbE=
github.com/russellhaering/goxmldsig v1.4.0 h1:8UcDh/xGyQiyrW+Fq5t8f+l2DLB1+zlhYzkPUJ7Qhys=
github.com/russellhaering/goxmldsig v1.4.0/go.mod h1:gM4MDENBQf7M+V824SGfyIUVFWydB7n0KkEubVJl+Tw=
github.com/stretchr/objx v0.1.0/go.mod h1:HFkY916IF+rwdDfMAkV7OtwuqBVzrE8GR6GFx+wExME=
github.com/stretchr/testify v1.6.1/go.mod h1:6Fq8oRcR53rry900zMqJjRRixrwX3KX962/h/Wwjteg=
github.com/stretchr/testify v1.8.4 h1:CcVxjf3Q8PM0mHUKJCdn+eZZtm5yQwehR5yeSVQQcUk=
github.com/stretchr/testify v1.8.4/go.mod h1:sz/lmYIOXD/1dqDmKjjqLyZ2RngseejIcXlSw2iwfAo=
golang.org/x/crypto v0.14.0 h1:wBqGXzWJW6m1XrIKlAH0Hs1JJ7+9KBwnIO8v66Q9cHc=
golang.org/x/crypto v0.14.0/go.mod h1:MVFd36DqK4CsrnJYDkBA3VC4m2GkXAM0PvzMCn4JQf4=
gopkg.in/check.v1 v0.0.0-20161208181325-20d25e280405/go.mod h1:Co6ibVJAznAaIkqp8huTwlJQCZ016jof/cbN4VW5Yz0=
gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127/go.mod h1:Co6ibVJAznAaIkqp8huTwlJQCZ016jof/cbN4VW5Yz0=
gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c/go.mod h1:JHkPIbrfpd72SG/EVd6muEfDQjcINNoR0C8j2r3qZ4Q=
gopkg.in/errgo.v2 v2.1.0/go.mod h1:hNsd1EY+bozCKY1Ytp96fpM3vjJbqLJn88ws8XvfDNI=
gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c/go.mod h1:K4uyk7z7BCEPqu6E+C64Yfv1cQ7kz7rIZviUmN+EgEM=
gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b/go.mod h1:K4uyk7z7BCEPqu6E+C64Yfv1cQ7kz7rIZviUmN+EgEM=
gopkg.in/yaml.v3 v3.0.1 h1:fxVm/GzAzEWqLHuvctI91KS9hhNmmWOoWu0XTYJS7CA=
gopkg.in/yaml.v3 v3.0.1/go.mod h1:K4uyk7z7BCEPqu6E+C64Yfv1cQ7kz7rIZviUmN+EgEM=
gotest.tools v2.2.0+incompatible h1:VsBPFP1AI068pPrMxtb/S8Zkgf9xEmTLJjfM+P5UIEo=
gotest.tools v2.2.0+incompatible/go.mod h1:DsYFclhRJ6vuDpmuTbkuFWG+y2sxOXAzmJt81HFBacw=`,
				},
				{
					Path: "main.go",
					Content: `
package main

import (
	_ "github.com/crewjam/saml"
)`,
				},
			},
			want: `
go_repository(
    name = "com_github_beevik_etree",
    importpath = "github.com/beevik/etree",
    sum = "h1:l7WETslUG/T+xOPs47dtd6jov2Ii/8/OjCldk5fYfQw=",
    version = "v1.2.0",
)

go_repository(
    name = "com_github_crewjam_saml",
    importpath = "github.com/crewjam/saml",
    replace = "github.com/grafana/saml",
    sum = "h1:SWmG1QLZ36Ay0htq4Wt3dzlNIhWvQ3GUf7mk19dR8nI=",
    version = "v0.4.15-0.20240523142256-cc370b98af7c",
)

go_repository(
    name = "com_github_jonboulle_clockwork",
    importpath = "github.com/jonboulle/clockwork",
    sum = "h1:UOGuzwb1PwsrDAObMuhUnj0p5ULPj8V/xJ7Kx9qUBdQ=",
    version = "v0.2.2",
)

go_repository(
    name = "com_github_mattermost_xml_roundtrip_validator",
    importpath = "github.com/mattermost/xml-roundtrip-validator",
    sum = "h1:RXbVD2UAl7A7nOTR4u7E3ILa4IbtvKBHw64LDsmu9hU=",
    version = "v0.1.0",
)

go_repository(
    name = "com_github_russellhaering_goxmldsig",
    importpath = "github.com/russellhaering/goxmldsig",
    sum = "h1:8UcDh/xGyQiyrW+Fq5t8f+l2DLB1+zlhYzkPUJ7Qhys=",
    version = "v1.4.0",
)

go_repository(
    name = "org_golang_x_crypto",
    importpath = "golang.org/x/crypto",
    sum = "h1:wBqGXzWJW6m1XrIKlAH0Hs1JJ7+9KBwnIO8v66Q9cHc=",
    version = "v0.14.0",
)`,
		},
		{
			desc: "missing_sum",
			files: []testtools.FileSpec{
				{
					Path: "go.mod",
					Content: `
module foo

go 1.22.5

require github.com/BurntSushi/toml v0.3.1`,
				},
				{
					Path:    "go.sum",
					Content: ``,
				},
				{
					Path: "main.go",
					Content: `
package main

import (
	_ "github.com/BurntSushi/toml"
)`,
				},
			},
			wantErr: "module github.com/BurntSushi/toml@v0.3.1 is missing from go.sum. Run 'go mod tidy' to fix.",
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			dir, cleanup := testtools.CreateFiles(t, tc.files)
			defer cleanup()
			filename := filepath.Join(dir, tc.files[0].Path)
			c := &config.Config{Exts: map[string]interface{}{}}
			rc, rcCleanup := repo.NewRemoteCache(nil)
			defer func() {
				if err := rcCleanup(); err != nil {
					t.Fatal(err)
				}
			}()
			gl := NewLanguage()
			gl.Configure(c, "", nil)
			importer := gl.(language.RepoImporter)
			result := importer.ImportRepos(language.ImportReposArgs{
				Config:    c,
				Path:      filename,
				ParseOnly: true,
				Cache:     rc,
			})
			if tc.wantErr != "" {
				if result.Error == nil {
					t.Fatalf("Want error %v but got %v", tc.wantErr, result)
				}
				if result.Error.Error() != tc.wantErr {
					t.Fatalf("Want error %v but got %v", tc.wantErr, result.Error)
				}
				return
			}
			if result.Error != nil {
				t.Fatal(result.Error)
			}
			f := rule.EmptyFile("test", "")
			for _, r := range result.Gen {
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
