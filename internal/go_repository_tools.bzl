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

load("@io_bazel_rules_go//go/private:common.bzl", "env_execute", "executable_extension")
load("@bazel_gazelle//internal:go_repository_cache.bzl", "read_cache_env")

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

exports_files(["ROOT"])
"""

def _go_repository_tools_impl(ctx):
    # Create a link to the gazelle repo. This will be our GOPATH.
    env = read_cache_env(ctx, str(ctx.path(ctx.attr.go_cache)))
    extension = executable_extension(ctx)
    go_tool = env["GOROOT"] + "/bin/go" + extension

    ctx.symlink(
        ctx.path(Label("@bazel_gazelle//:WORKSPACE")).dirname,
        "src/github.com/bazelbuild/bazel-gazelle",
    )

    # Resolve a label for each source file so this rule will be re-executed
    # when they change.
    list_script = str(ctx.path(Label("@bazel_gazelle//internal:list_repository_tools_srcs.go")))
    result = ctx.execute([go_tool, "run", list_script])
    if result.return_code:
        print("could not resolve gazelle sources: " + result.stderr)
    else:
        for line in result.stdout.split("\n"):
            line = line.strip()
            if line == "":
                continue
            ctx.path(Label(line))

    # Build the tools.
    env.update({
        "GOPATH": str(ctx.path(".")),
        "GO111MODULE": "off",
        # workaround: to find gcc for go link tool on Arm platform
        "PATH": ctx.os.environ["PATH"],
        # workaround: avoid the Go SDK paths from leaking into the binary
        "GOROOT_FINAL": "GOROOT",
        # workaround: avoid cgo paths in /tmp leaking into binary
        "CGO_ENABLED": "0",
    })
    if "GOPROXY" in ctx.os.environ:
        env["GOPROXY"] = ctx.os.environ["GOPROXY"]

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
