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
)
load(
    "@bazel_gazelle//internal:go_repository_cache.bzl",
    "go_repository_cache",
)
load(
    "@bazel_gazelle//internal:go_repository_tools.bzl",
    "go_repository_tools",
)
load(
    "@bazel_gazelle//internal:go_repository_config.bzl",
    "go_repository_config",
)
load(
    "@bazel_tools//tools/build_defs/repo:git.bzl",
    _tools_git_repository = "git_repository",
)

# Re-export go_repository . Users should get it from this file.
go_repository = _go_repository

def gazelle_dependencies(
        go_sdk = "",
        go_repository_default_config = "@//:WORKSPACE",
        go_env = {}):
    if go_sdk:
        go_repository_cache(
            name = "bazel_gazelle_go_repository_cache",
            go_sdk_name = go_sdk,
            go_env = go_env,
        )
    else:
        go_sdk_info = {}
        for name, r in native.existing_rules().items():
            # match internal rule names but don't reference them directly.
            # New rules may be added in the future, and they might be
            # renamed (_go_download_sdk => go_download_sdk).
            if name != "go_sdk" and ("go_" not in r["kind"] or "_sdk" not in r["kind"]):
                continue
            if r.get("goos", "") and r.get("goarch", ""):
                platform = r["goos"] + "_" + r["goarch"]
            else:
                platform = "host"
            go_sdk_info[name] = platform
        go_repository_cache(
            name = "bazel_gazelle_go_repository_cache",
            go_sdk_info = go_sdk_info,
            go_env = go_env,
        )

    go_repository_tools(
        name = "bazel_gazelle_go_repository_tools",
        go_cache = "@bazel_gazelle_go_repository_cache//:go.env",
    )

    go_repository_config(
        name = "bazel_gazelle_go_repository_config",
        config = go_repository_default_config,
    )

    _maybe(
        _tools_git_repository,
        name = "bazel_skylib",
        remote = "https://github.com/bazelbuild/bazel-skylib",
        commit = "3fea8cb680f4a53a129f7ebace1a5a4d1e035914",  # 0.5.0 as of 2018-11-01
    )

    _maybe(
        go_repository,
        name = "com_github_bazelbuild_buildtools",
        importpath = "github.com/bazelbuild/buildtools",
        sum = "h1:Et1IIXrXwhpDvR5wH9REPEZ0sUtzUoJSq19nfmBqzBY=",
        version = "v0.0.0-20200718160251-b1667ff58f71",
    )

    _maybe(
        go_repository,
        name = "com_github_bazelbuild_rules_go",
        importpath = "github.com/bazelbuild/rules_go",
        sum = "h1:wzbawlkLtl2ze9w/312NHZ84c7kpUCtlkD8HgFY27sw=",
        version = "v0.0.0-20190719190356-6dae44dc5cab",
    )

    _maybe(
        go_repository,
        name = "com_github_bmatcuk_doublestar",
        importpath = "github.com/bmatcuk/doublestar",
        sum = "h1:oC24CykoSAB8zd7XgruHo33E0cHJf/WhQA/7BeXj+x0=",
        version = "v1.2.2",
    )

    _maybe(
        go_repository,
        name = "com_github_burntsushi_toml",
        importpath = "github.com/BurntSushi/toml",
        sum = "h1:WXkYYl6Yr3qBf1K79EBnL4mak0OimBfB0XUf9Vl28OQ=",
        version = "v0.3.1",
    )

    _maybe(
        go_repository,
        name = "com_github_davecgh_go_spew",
        importpath = "github.com/davecgh/go-spew",
        sum = "h1:vj9j/u1bqnvCEfJOwUhtlOARqs3+rkHYY13jYWTU97c=",
        version = "v1.1.1",
    )

    _maybe(
        go_repository,
        name = "com_github_fsnotify_fsnotify",
        importpath = "github.com/fsnotify/fsnotify",
        sum = "h1:IXs+QLmnXW2CcXuY+8Mzv/fWEsPGWxqefPtCP5CnV9I=",
        version = "v1.4.7",
    )

    _maybe(
        go_repository,
        name = "com_github_google_go_cmp",
        importpath = "github.com/google/go-cmp",
        sum = "h1:L8R9j+yAqZuZjsqh/z+F1NCffTKKLShY6zXTItVIZ8M=",
        version = "v0.5.4",
    )

    _maybe(
        go_repository,
        name = "com_github_kr_pretty",
        importpath = "github.com/kr/pretty",
        sum = "h1:L/CwN0zerZDmRFUapSPitk6f+Q3+0za1rQkzVuMiMFI=",
        version = "v0.1.0",
    )

    _maybe(
        go_repository,
        name = "com_github_kr_pty",
        importpath = "github.com/kr/pty",
        sum = "h1:VkoXIwSboBpnk99O/KFauAEILuNHv5DVFKZMBN/gUgw=",
        version = "v1.1.1",
    )

    _maybe(
        go_repository,
        name = "com_github_kr_text",
        importpath = "github.com/kr/text",
        sum = "h1:45sCR5RtlFHMR4UwH9sdQ5TC8v0qDQCHnXt+kaKSTVE=",
        version = "v0.1.0",
    )

    _maybe(
        go_repository,
        name = "com_github_pelletier_go_toml",
        importpath = "github.com/pelletier/go-toml",
        sum = "h1:T5zMGML61Wp+FlcbWjRDT7yAxhJNAiPPLOFECq181zc=",
        version = "v1.2.0",
    )

    _maybe(
        go_repository,
        name = "com_github_pmezard_go_difflib",
        importpath = "github.com/pmezard/go-difflib",
        sum = "h1:4DBwDE0NGyQoBHbLQYPwSUPoCMWR5BEzIk/f1lZbAQM=",
        version = "v1.0.0",
    )

    _maybe(
        go_repository,
        name = "in_gopkg_check_v1",
        importpath = "gopkg.in/check.v1",
        sum = "h1:qIbj1fsPNlZgppZ+VLlY7N33q108Sa+fhmuc+sWQYwY=",
        version = "v1.0.0-20180628173108-788fd7840127",
    )

    _maybe(
        go_repository,
        name = "in_gopkg_yaml_v2",
        importpath = "gopkg.in/yaml.v2",
        sum = "h1:ZCJp+EgiOT7lHqUV2J862kp8Qj64Jo6az82+3Td9dZw=",
        version = "v2.2.2",
    )

    _maybe(
        go_repository,
        name = "org_golang_x_crypto",
        importpath = "golang.org/x/crypto",
        sum = "h1:ObdrDkeb4kJdCP557AjRjq69pTHfNouLtWZG7j9rPN8=",
        version = "v0.0.0-20191011191535-87dc89f01550",
    )

    _maybe(
        go_repository,
        name = "org_golang_x_mod",
        importpath = "golang.org/x/mod",
        sum = "h1:Kvvh58BN8Y9/lBi7hTekvtMpm07eUZ0ck5pRHpsMWrY=",
        version = "v0.4.1",
    )

    _maybe(
        go_repository,
        name = "org_golang_x_net",
        importpath = "golang.org/x/net",
        sum = "h1:R/3boaszxrf1GEUWTVDzSKVwLmSJpwZ1yqXm8j0v2QI=",
        version = "v0.0.0-20190620200207-3b0461eec859",
    )

    _maybe(
        go_repository,
        name = "org_golang_x_sync",
        importpath = "golang.org/x/sync",
        sum = "h1:vcxGaoTs7kV8m5Np9uUNQin4BrLOthgV7252N8V+FwY=",
        version = "v0.0.0-20190911185100-cd5d95a43a6e",
    )

    _maybe(
        go_repository,
        name = "org_golang_x_sys",
        importpath = "golang.org/x/sys",
        sum = "h1:+R4KGOnez64A81RvjARKc4UT5/tI9ujCIVX+P5KiHuI=",
        version = "v0.0.0-20190412213103-97732733099d",
    )

    _maybe(
        go_repository,
        name = "org_golang_x_text",
        importpath = "golang.org/x/text",
        sum = "h1:g61tztE5qeGQ89tm6NTjjM9VPIm088od1l6aSorWRWg=",
        version = "v0.3.0",
    )

    _maybe(
        go_repository,
        name = "org_golang_x_tools",
        importpath = "golang.org/x/tools",
        sum = "h1:aZzprAO9/8oim3qStq3wc1Xuxx4QmAGriC4VU4ojemQ=",
        version = "v0.0.0-20191119224855-298f0cb1881e",
    )

    _maybe(
        go_repository,
        name = "org_golang_x_xerrors",
        importpath = "golang.org/x/xerrors",
        sum = "h1:E7g+9GITq07hpfrRu66IVDexMakfv52eLZ2CXBWiKr4=",
        version = "v0.0.0-20191204190536-9bdfabe68543",
    )

def _maybe(repo_rule, name, **kwargs):
    if name not in native.existing_rules():
        repo_rule(name = name, **kwargs)
