# Copyright 2019 The Bazel Authors. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

load("//internal:common.bzl", "env_execute", "executable_extension")
load("//internal:go_repository_cache.bzl", "read_cache_env")
load("//internal:go_repository_tools_srcs.bzl", "GO_REPOSITORY_TOOLS_SRCS")

_GO_REPOSITORY_TOOLS_BUILD_FILE = """
package(default_visibility = ["//visibility:public"])

filegroup(
    name = "fetch_repo",
    srcs = ["bin/fetch_repo{extension}"],
)

filegroup(
    name = "gazelle",
    srcs = ["bin/gazelle{extension}"],
)

filegroup(
    name = "generate_repo_config",
    srcs = ["bin/generate_repo_config{extension}"],
)

exports_files(["ROOT"])
"""

def _go_repository_tools_impl(ctx):
    # Create a link to the gazelle repo. This will be our GOPATH.
    env = read_cache_env(ctx, str(ctx.path(ctx.attr.go_cache)))
    extension = executable_extension(ctx)
    go_tool = env["GOROOT"] + "/bin/go" + extension

    ctx.symlink(
        ctx.path(Label("//:WORKSPACE")).dirname,
        "src/github.com/bazelbuild/bazel-gazelle",
    )

    env.update({
        "GOPATH": str(ctx.path(".")),
        "GOBIN": "",
        "GO111MODULE": "off",
        # workaround: avoid the Go SDK paths from leaking into the binary
        "GOROOT_FINAL": "GOROOT",
    })

    if "PATH" in ctx.os.environ:
        # workaround: to find gcc for go link tool on Arm platform
        env["PATH"] = ctx.os.environ["PATH"]
    if "GOPROXY" in ctx.os.environ:
        env["GOPROXY"] = ctx.os.environ["GOPROXY"]

    # Make sure the list of source is up to date.
    # We don't want to run the script, then resolve each source file it returns.
    # If many of the sources changed even slightly, Bazel would restart this
    # rule each time. Compiling the script is relatively slow.
    # Don't try this on Windows: bazel does not set up symbolic links.
    if "windows" not in ctx.os.name:
        result = env_execute(
            ctx,
            [
                go_tool,
                "run",
                ctx.path(ctx.attr._list_repository_tools_srcs),
                "-dir=src/github.com/bazelbuild/bazel-gazelle",
                "-check=internal/go_repository_tools_srcs.bzl",
            ],
            environment = env,
        )
        if result.return_code:
            fail("list_repository_tools_srcs: " + result.stderr)

    # Build the tools.
    args = [
        go_tool,
        "install",
        "-ldflags",
        "-w -s",
        "-gcflags",
        "all=-trimpath=" + env["GOPATH"],
        "-asmflags",
        "all=-trimpath=" + env["GOPATH"],
        "github.com/bazelbuild/bazel-gazelle/cmd/gazelle",
        "github.com/bazelbuild/bazel-gazelle/cmd/fetch_repo",
        "github.com/bazelbuild/bazel-gazelle/cmd/generate_repo_config",
    ]
    result = env_execute(ctx, args, environment = env)
    if result.return_code:
        fail("failed to build tools: " + result.stderr)

    # add a build file to export the tools
    ctx.file(
        "BUILD.bazel",
        _GO_REPOSITORY_TOOLS_BUILD_FILE.format(extension = executable_extension(ctx)),
        False,
    )
    ctx.file(
        "ROOT",
        "",
        False,
    )

go_repository_tools = repository_rule(
    _go_repository_tools_impl,
    attrs = {
        "go_cache": attr.label(
            mandatory = True,
            allow_single_file = True,
        ),
        "_go_repository_tools_srcs": attr.label_list(
            default = GO_REPOSITORY_TOOLS_SRCS,
        ),
        "_list_repository_tools_srcs": attr.label(
            default = "//internal:list_repository_tools_srcs.go",
        ),
    },
    environ = [
        "GOCACHE",
        "GOPATH",
        "GO_REPOSITORY_USE_HOST_CACHE",
    ],
)
"""go_repository_tools is a synthetic repository used by go_repository.


go_repository depends on two Go binaries: fetch_repo and gazelle. We can't
build these with Bazel inside a repository rule, and we don't want to manage
prebuilt binaries, so we build them in here with go build, using whichever
SDK rules_go is using.
"""
