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

package go_repository_test

import (
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel_testing"
)

const mainWorkspace = `
-- WORKSPACE --
local_repository(
    name = "io_bazel_rules_go",
    path = "../io_bazel_rules_go",
)

load("@io_bazel_rules_go//go:deps.bzl", "go_rules_dependencies", "go_register_toolchains")

go_rules_dependencies()

go_register_toolchains(go_version = "host")

local_repository(
    name = "bazel_gazelle",
    path = "../bazel_gazelle",
)

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")

gazelle_dependencies()

go_repository(
    name = "errors_go_git",
    importpath = "github.com/pkg/errors",
    commit = "30136e27e2ac8d167177e8a583aa4c3fea5be833",
    patches = ["@bazel_gazelle//internal:repository_rules_test_errors.patch"],
    patch_args = ["-p1"],
)

go_repository(
    name = "errors_go_mod",
    importpath = "github.com/pkg/errors",
    version = "v0.8.1",
    sum ="h1:iURUrRGxPUNPdy5/HRSm+Yj6okJ6UtLINN0Q9M4+h3I=",
)

-- BUILD.bazel --
`

func TestMain(m *testing.M) {
	bazel_testing.TestMain(m, bazel_testing.Args{Main: mainWorkspace})
}

func TestBuild(t *testing.T) {
	if err := bazel_testing.RunBazel("build", "@errors_go_git//:errors", "@errors_go_mod//:go_default_library"); err != nil {
		t.Fatal(err)
	}
}
