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
load("@bazel_gazelle//internal:go_repository_cache.bzl", "read_cache_env")

def _go_repository_config_(ctx):
    # Locate and resolve configuration files. Gazelle reads directives and
    # known repositories from these files. Resolving them here forces
    # go_repository rules to be invalidated when they change. Gazelle's cache
    # should NOT be invalidated, so we shouldn't need to download these again.
    # TODO(#549): vcs repositories are not cached and still need to be fetched.
    config_path = None
    if ctx.attr.config:
        config_path = ctx.path(ctx.attr.config)
        for label in _find_macro_file_labels(ctx, ctx.attr.config):
            ctx.path(label)

    env = read_cache_env(ctx, str(ctx.path(Label("@bazel_gazelle_go_repository_cache//:go.env"))))

    generate_repo_config = str(ctx.path(Label("@bazel_gazelle_go_repository_tools//:bin/generate_repo_config{}".format(executable_extension(ctx)))))
    list_repos_args = [
        "-dest=" + str(ctx.path("")),
        "-config_source=" + str(config_path),
    ]
    result = env_execute(
        ctx,
        [generate_repo_config] + list_repos_args,
        environment = env,
    )
    if result.return_code:
        print("generate_repo_config: " + result.stderr)

    # add an empty build file so Bazel recognizes the config
    ctx.file(
        "BUILD.bazel",
        "",
        False,
    )


go_repository_config = repository_rule(
    implementation = _go_repository_config_,
    attrs = {
        "config": attr.label(default = "@//:WORKSPACE"),
    },
)

def _find_macro_file_labels(ctx, label):
    """Returns a list of labels for configuration files that Gazelle may read.

    The list is gathered by reading '# gazelle:repository_macro' directives
    from the file named by label (which is not included in the returned list).
    """
    seen = {}
    files = []

    result = ctx.execute(["cat", str(ctx.path(label))])
    if result.return_code == 0:
        content = result.stdout
    else:
        # TODO(jayconrod): "type" might work on Windows, but I think
        # it's a shell builtin, and I'm not sure if ctx.execute will work.
        content = ""

    lines = content.split("\n")
    for line in lines:
        i = line.find("#")
        if i < 0:
            continue
        line = line[i + len("#"):]
        i = line.find("gazelle:")
        if i < 0 or not line[:i].isspace():
            continue
        line = line[i + len("gazelle:"):]
        i = line.find("repository_macro")
        if i < 0 or (i > 0 and not line[:i].isspace()):
            continue
        line = line[i + len("repository_macro"):]
        if len(line) == 0 or not line[0].isspace():
            continue
        i = line.rfind("%")
        if i < 0:
            continue
        line = line[:i].lstrip()
        macro_label = Label("@" + label.workspace_name + "//:" + line)
        if macro_label not in seen:
            seen[macro_label] = None
            files.append(macro_label)

    return files
