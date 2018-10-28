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
    "go_repository",
    _go_repository_tools = "go_repository_tools",
)
load(
    "@bazel_gazelle//internal:overlay_repository.bzl",
    "git_repository",
    "http_archive",
)
load(
    "@bazel_gazelle//third_party:manifest.bzl",
    _manifest = "manifest",
)

def gazelle_dependencies(go_sdk = "@go_sdk//:ROOT"):
    _go_repository_tools(
        name = "bazel_gazelle_go_repository_tools",
        go_sdk = go_sdk,
    )

    _maybe(
        git_repository,
        name = "bazel_skylib",
        remote = "https://github.com/bazelbuild/bazel-skylib",
        commit = "f3dd8fd95a7d078cb10fd7fb475b22c3cdbcb307",  # 0.2.0 as of 2017-12-04
    )

    _maybe(
        go_repository,
        name = "errors_go",
        importpath = "github.com/pkg/errors",
        commit = "30136e27e2ac8d167177e8a583aa4c3fea5be833",
        patches = ["@bazel_gazelle//internal:repository_rules_test_errors.patch"],
        patch_args = ["-p1"],
    )

def _maybe(repo_rule, name, **kwargs):
    if name not in native.existing_rules():
        repo_rule(name = name, **kwargs)
