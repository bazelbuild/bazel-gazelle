# Copyright 2014 The Bazel Authors. All rights reserved.
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

# We can't disable timeouts on Bazel, but we can set them to large values.
_GO_REPOSITORY_TIMEOUT = 86400

def _go_repository_impl(ctx):
    if ctx.attr.urls:
        # download from explicit source url
        for key in ("commit", "tag", "vcs", "remote"):
            if getattr(ctx.attr, key):
                fail("cannot specifiy both urls and %s" % key, key)
        ctx.download_and_extract(
            url = ctx.attr.urls,
            sha256 = ctx.attr.sha256,
            stripPrefix = ctx.attr.strip_prefix,
            type = ctx.attr.type,
        )
    else:
        # checkout from vcs
        if ctx.attr.commit and ctx.attr.tag:
            fail("cannot specify both of commit and tag", "commit")
        if ctx.attr.commit:
            rev = ctx.attr.commit
            rev_key = "commit"
        elif ctx.attr.tag:
            rev = ctx.attr.tag
            rev_key = "tag"
        else:
            fail("neither commit or tag is specified", "commit")
        for key in ("urls", "strip_prefix", "type", "sha256"):
            if getattr(ctx.attr, key):
                fail("cannot specify both %s and %s" % (rev_key, key), key)

        # Using fetch repo
        if ctx.attr.vcs and not ctx.attr.remote:
            fail("if vcs is specified, remote must also be")

        env_keys = [
            "PATH",
            "HOME",
            "SSH_AUTH_SOCK",
            "HTTP_PROXY",
            "HTTPS_PROXY",
            "NO_PROXY",
            "GIT_SSL_CAINFO",
        ]
        fetch_repo_env = {k: ctx.os.environ[k] for k in env_keys if k in ctx.os.environ}

        _fetch_repo = "@bazel_gazelle_go_repository_tools//:bin/fetch_repo{}".format(executable_extension(ctx))
        args = [
            ctx.path(Label(_fetch_repo)),
            "--dest",
            ctx.path(""),
        ]
        if ctx.attr.remote:
            args.extend(["--remote", ctx.attr.remote])
        if rev:
            args.extend(["--rev", rev])
        if ctx.attr.vcs:
            args.extend(["--vcs", ctx.attr.vcs])
        if ctx.attr.importpath:
            args.extend(["--importpath", ctx.attr.importpath])
        result = env_execute(ctx, args, environment = fetch_repo_env, timeout = _GO_REPOSITORY_TIMEOUT)
        if result.return_code:
            fail("failed to fetch %s: %s" % (ctx.name, result.stderr))

    generate = ctx.attr.build_file_generation == "on"
    if ctx.attr.build_file_generation == "auto":
        generate = True
        for name in ["BUILD", "BUILD.bazel", ctx.attr.build_file_name]:
            path = ctx.path(name)
            if path.exists and not env_execute(ctx, ["test", "-f", path]).return_code:
                generate = False
                break
    if generate:
        # Build file generation is needed
        _gazelle = "@bazel_gazelle_go_repository_tools//:bin/gazelle{}".format(executable_extension(ctx))
        gazelle = ctx.path(Label(_gazelle))
        cmd = [
            gazelle,
            "--go_prefix",
            ctx.attr.importpath,
            "--mode",
            "fix",
            "--repo_root",
            ctx.path(""),
        ]
        if ctx.attr.build_file_name:
            cmd.extend(["--build_file_name", ctx.attr.build_file_name])
        if ctx.attr.build_tags:
            cmd.extend(["--build_tags", ",".join(ctx.attr.build_tags)])
        if ctx.attr.build_external:
            cmd.extend(["--external", ctx.attr.build_external])
        if ctx.attr.build_file_proto_mode:
            cmd.extend(["--proto", ctx.attr.build_file_proto_mode])
        cmd.extend(ctx.attr.build_extra_args)
        cmd.append(ctx.path(""))
        result = env_execute(ctx, cmd)
        if result.return_code:
            fail("failed to generate BUILD files for %s: %s" % (
                ctx.attr.importpath,
                result.stderr,
            ))

    # Apply patches if necessary.
    patch(ctx)

go_repository = repository_rule(
    implementation = _go_repository_impl,
    attrs = {
        # Fundamental attributes of a go repository
        "importpath": attr.string(mandatory = True),

        # Attributes for a repository that should be checked out from VCS
        "commit": attr.string(),
        "tag": attr.string(),
        "vcs": attr.string(
            default = "",
            values = [
                "",
                "git",
                "hg",
                "svn",
                "bzr",
            ],
        ),
        "remote": attr.string(),

        # Attributes for a repository that comes from a source blob not a vcs
        "urls": attr.string_list(),
        "strip_prefix": attr.string(),
        "type": attr.string(),
        "sha256": attr.string(),

        # Attributes for a repository that needs automatic build file generation
        "build_external": attr.string(
            values = [
                "",
                "external",
                "vendored",
            ],
        ),
        "build_file_name": attr.string(default = "BUILD.bazel,BUILD"),
        "build_file_generation": attr.string(
            default = "auto",
            values = [
                "on",
                "auto",
                "off",
            ],
        ),
        "build_tags": attr.string_list(),
        "build_file_proto_mode": attr.string(
            values = [
                "",
                "default",
                "package",
                "disable",
                "disable_global",
                "legacy",
            ],
        ),
        "build_extra_args": attr.string_list(),

        # Patches to apply after running gazelle.
        "patches": attr.label_list(),
        "patch_tool": attr.string(default = "patch"),
        "patch_args": attr.string_list(default = ["-p0"]),
        "patch_cmds": attr.string_list(default = []),
    },
)
"""See repository.rst#go-repository for full documentation."""

# Copied from @bazel_tools//tools/build_defs/repo:utils.bzl
def patch(ctx):
    """Implementation of patching an already extracted repository"""
    bash_exe = ctx.os.environ["BAZEL_SH"] if "BAZEL_SH" in ctx.os.environ else "bash"
    for patchfile in ctx.attr.patches:
        command = "{patchtool} {patch_args} < {patchfile}".format(
            patchtool = ctx.attr.patch_tool,
            patchfile = ctx.path(patchfile),
            patch_args = " ".join([
                "'%s'" % arg
                for arg in ctx.attr.patch_args
            ]),
        )
        st = ctx.execute([bash_exe, "-c", command])
        if st.return_code:
            fail("Error applying patch %s:\n%s%s" %
                 (str(patchfile), st.stderr, st.stdout))
    for cmd in ctx.attr.patch_cmds:
        st = ctx.execute([bash_exe, "-c", cmd])
        if st.return_code:
            fail("Error applying patch command %s:\n%s%s" %
                 (cmd, st.stdout, st.stderr))

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
"""

def _go_repository_tools_impl(ctx):
    extension = executable_extension(ctx)
    go_root = ctx.path(ctx.attr.go_sdk).dirname
    go_tool = str(go_root) + "/bin/go" + extension

    for root_file, prefix in ctx.attr._deps.items():
        ctx.symlink(ctx.path(root_file).dirname, "src/" + prefix)

    env = {
        "GOROOT": str(go_root),
        "GOPATH": str(ctx.path(".")),
        "GO111MODULE": "off",
        # GOCACHE is required starting in Go 1.12
        "GOCACHE": str(ctx.path(".")) + "/.gocache",
        # workaround: to find gcc for go link tool on Arm platform
        "PATH": ctx.os.environ["PATH"],
        # workaround: avoid the Go SDK paths from leaking into the binary
        "GOROOT_FINAL": "GOROOT",
        # workaround: avoid cgo paths in /tmp leaking into binary
        "CGO_ENABLED": "0",
    }

    for tool in ("fetch_repo", "gazelle"):
        args = [
            go_tool,
            "install",
            "-a",
            "-ldflags",
            "-w -s",
            "-gcflags",
            "all=-trimpath=" + env["GOPATH"],
            "-asmflags",
            "all=-trimpath=" + env["GOPATH"],
            "github.com/bazelbuild/bazel-gazelle/cmd/{}".format(tool),
        ]
        result = env_execute(ctx, args, environment = env)
        if result.return_code:
            fail("failed to build {}: {}".format(tool, result.stderr))

    result = ctx.execute(["rm", "-rf", ".gocache"])
    if result.return_code:
        print("failed to remove GOCACHE: {}".format(result.stderr))

    # add a build file to export the tools
    ctx.file(
        "BUILD.bazel",
        _GO_REPOSITORY_TOOLS_BUILD_FILE.format(extension = executable_extension(ctx)),
        False,
    )

go_repository_tools = repository_rule(
    _go_repository_tools_impl,
    attrs = {
        "go_sdk": attr.label(
            default = "@go_sdk//:ROOT",
            allow_single_file = True,
        ),
        "_deps": attr.label_keyed_string_dict(
            default = {
                "@bazel_gazelle//:WORKSPACE": "github.com/bazelbuild/bazel-gazelle",
            },
        ),
    },
    environ = ["TMP"],
)
"""go_repository_tools is a synthetic repository used by go_repository.

go_repository depends on two Go binaries: fetch_repo and gazelle. We can't
build these with Bazel inside a repository rule, and we don't want to manage
prebuilt binaries, so we build them in here with go build, using whichever
SDK rules_go is using.
"""
