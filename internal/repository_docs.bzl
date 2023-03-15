# Module used by stardoc to generate API documentation.
# Not meant for use by bazel-gazelle users.
"""
Repository rules
================

Repository rules are Bazel rules that can be used in WORKSPACE files to import
projects in external repositories. Repository rules may download projects
and transform them by applying patches or generating build files.

The Gazelle repository provides three rules:

* [`go_repository`](#go_repository) downloads a Go project using either `go mod download`, a
  version control tool like `git`, or a direct HTTP download. It understands
  Go import path redirection. If build files are not already present, it can
  generate them with Gazelle.
* [`git_repository`](#git_repository) downloads a project with git. Unlike the native
  `git_repository`, this rule allows you to specify an "overlay": a set of
  files to be copied into the downloaded project. This may be used to add
  pre-generated build files to a project that doesn't have them.
* [`http_archive`](#http_archive) downloads a project via HTTP. It also lets you specify
  overlay files.

**NOTE:** `git_repository` and `http_archive` are deprecated in favor of the
rules of the same name in [@bazel_tools//tools/build_defs/repo:git.bzl] and
[@bazel_tools//tools/build_defs/repo:http.bzl].

Repository rules can be loaded and used in WORKSPACE like this:

```starlark
load("@bazel_gazelle//:deps.bzl", "go_repository")

go_repository(
    name = "com_github_pkg_errors",
    commit = "816c9085562cd7ee03e7f8188a1cfd942858cded",
    importpath = "github.com/pkg/errors",
)
```

Gazelle can add and update some of these rules automatically using the
`update-repos` command. For example, the rule above can be added with:

```shell
$ gazelle update-repos github.com/pkg/errors
```

[http_archive.strip_prefix]: https://docs.bazel.build/versions/master/be/workspace.html#http_archive.strip_prefix
[native git_repository rule]: https://docs.bazel.build/versions/master/be/workspace.html#git_repository
[native http_archive rule]: https://docs.bazel.build/versions/master/be/workspace.html#http_archive
[manifest.bzl]: third_party/manifest.bzl
[Directives]: /README.rst#directives
[@bazel_tools//tools/build_defs/repo:git.bzl]: https://github.com/bazelbuild/bazel/blob/master/tools/build_defs/repo/git.bzl
[@bazel_tools//tools/build_defs/repo:http.bzl]: https://github.com/bazelbuild/bazel/blob/master/tools/build_defs/repo/http.bzl
"""

load("go_repository.bzl", _go_repository = "go_repository")
load(
    "overlay_repository.bzl",
    _git_repository = "git_repository",
    _http_archive = "http_archive",
)

go_repository = _go_repository
http_archive = _http_archive
git_repository = _git_repository
