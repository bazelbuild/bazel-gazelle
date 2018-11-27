# Copyright 2017 The Bazel Authors. All rights reserved.
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

load(
    "@bazel_gazelle//internal:go_repository.bzl",
    _go_repository = "go_repository",
    _go_repository_tools = "go_repository_tools",
)
load(
    "@bazel_gazelle//internal:overlay_repository.bzl",
    # Load overlay git_repository and http_archive in order to re-export.
    # These may be removed at some point in the future.
    "git_repository",
    "http_archive",
)
load(
    "@bazel_tools//tools/build_defs/repo:git.bzl",
    _tools_git_repository = "git_repository",
)

# Re-export go_repository . Users should get it from this file.
go_repository = _go_repository

def gazelle_dependencies(go_sdk = "@go_sdk//:ROOT"):
    _go_repository_tools(
        name = "bazel_gazelle_go_repository_tools",
        go_sdk = go_sdk,
    )

    _maybe(
        _tools_git_repository,
        name = "bazel_skylib",
        remote = "https://github.com/bazelbuild/bazel-skylib",
        commit = "3fea8cb680f4a53a129f7ebace1a5a4d1e035914",  # 0.5.0 as of 2018-11-01
    )

def _maybe(repo_rule, name, **kwargs):
    if name not in native.existing_rules():
        repo_rule(name = name, **kwargs)
