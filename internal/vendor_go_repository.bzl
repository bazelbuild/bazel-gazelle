
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

def __join_path(path, path_to_join):
    final_path = path
    for repository in path_to_join.split("/"):
        if repository != '':
            final_path = final_path.get_child(repository)
    return final_path


def _vendor_go_repository_impl(ctx):
    fetch_repo_env = {
        "PATH": ctx.os.environ["PATH"],
    }
    for file in __join_path(ctx.path(Label("@//:WORKSPACE")).dirname, "vendor/"+ctx.attr.importpath).readdir():
        result = env_execute(ctx, ["cp", "-RP", file, ctx.path(file.basename)], environment = fetch_repo_env, timeout = _GO_REPOSITORY_TIMEOUT)
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


vendor_go_repository = repository_rule(
    implementation = _vendor_go_repository_impl,
    attrs = {
        # Fundamental attributes of a go repository
        "importpath": attr.string(mandatory = True),

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
                "disable",
                "legacy",
            ],
        ),
        "build_extra_args": attr.string_list(),
    },
)
