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

load("@io_bazel_rules_go//go:def.bzl", "go_repository")

def gazelle_dependencies():
  _maybe(native.git_repository,
      name = "bazel_skylib",
      remote = "https://github.com/bazelbuild/bazel-skylib",
      commit = "f3dd8fd95a7d078cb10fd7fb475b22c3cdbcb307", # 0.2.0 as of 2017-12-04
  )

  # TODO(jayconrod): restore when gazelle is no longer built for go_repository.
  # _maybe(go_repository,
  #     name = "com_github_pelletier_go_toml",
  #     importpath = "github.com/pelletier/go-toml",
  #     commit = "16398bac157da96aa88f98a2df640c7f32af1da2", # v1.0.1 as of 2017-12-19
  # )

def _maybe(repo_rule, name, **kwargs):
  if name not in native.existing_rules():
    repo_rule(name=name, **kwargs)
