# Copyright 2018 The Bazel Authors. All rights reserved.
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

def _http_archive_impl(ctx):
    print("http_archive is deprecated. Use http_archive from @bazel_tools//tools/build_defs/repo:http.bzl instead.")
    overlay = _resolve_overlay(ctx, ctx.attr.overlay)
    ctx.download_and_extract(
        url = ctx.attr.urls,
        sha256 = ctx.attr.sha256,
        type = ctx.attr.type,
        stripPrefix = ctx.attr.strip_prefix,
    )
    _apply_overlay(ctx, overlay)

http_archive = repository_rule(
    implementation = _http_archive_impl,
    doc = """
**NOTE:** `http_archive` is deprecated in favor of the rule of the same name
in [@bazel_tools//tools/build_defs/repo:http.bzl].

`http_archive` downloads a project over HTTP(S). It has the same features as
the [native http_archive rule], but it also allows you to copy a set of files
into the repository after download. This is particularly useful for placing
pre-generated build files.

**Example**

```starlark
load("@bazel_gazelle//:deps.bzl", "http_archive")

http_archive(
    name = "com_github_pkg_errors",
    urls = ["https://codeload.github.com/pkg/errors/zip/816c9085562cd7ee03e7f8188a1cfd942858cded"],
    strip_prefix = "errors-816c9085562cd7ee03e7f8188a1cfd942858cded",
    type = "zip",
    overlay = {
        "@my_repo//third_party:com_github_pkg_errors/BUILD.bazel.in" : "BUILD.bazel",
    },
)
```
""",
    attrs = {
        "urls": attr.string_list(
            doc = """A list of HTTP(S) URLs where the project can be downloaded. Bazel will
            attempt to download the first URL; the others are mirrors.""",
        ),
        "sha256": attr.string(
            doc = """The SHA-256 sum of the downloaded archive. When set, Bazel will verify the
            archive against this sum before extracting it.

            **CAUTION:** Do not use this with services that prepare source archives on
            demand, such as codeload.github.com. Any minor change in the server software
            can cause differences in file order, alignment, and compression that break
            SHA-256 sums.""",
        ),
        "strip_prefix": attr.string(
            doc = "A directory prefix to strip. See [http_archive.strip_prefix].",
        ),
        "type": attr.string(
            doc = """One of `"zip"`, `"tar.gz"`, `"tgz"`, `"tar.bz2"`, `"tar.xz"`.

            The file format of the repository archive. This is normally inferred from
            the downloaded file name.""",
        ),
        "overlay": attr.label_keyed_string_dict(
            allow_files = True,
            doc = """A set of files to copy into the downloaded repository. The keys in this
            dictionary are Bazel labels that point to the files to copy. These must be
            fully qualified labels (i.e., `@repo//pkg:name`) because relative labels
            are interpreted in the checked out repository, not the repository containing
            the WORKSPACE file. The values in this dictionary are root-relative paths
            where the overlay files should be written.

            It's convenient to store the overlay dictionaries for all repositories in
            a separate .bzl file. See Gazelle's `manifest.bzl`_ for an example.""",
        ),
    },
)
# TODO(jayconrod): add strip_count to remove a number of unnamed
# parent directories.
# TODO(jayconrod): add sha256_contents to check sha256sum on files extracted
# from the archive instead of on the archive itself.

def _git_repository_impl(ctx):
    print("git_repository is deprecated. Use git_repository from @bazel_tools//tools/build_defs/repo:git.bzl instead.")
    if not ctx.attr.commit and not ctx.attr.tag:
        fail("either 'commit' or 'tag' must be specified")
    if ctx.attr.commit and ctx.attr.tag:
        fail("'commit' and 'tag' may not both be specified")

    overlay = _resolve_overlay(ctx, ctx.attr.overlay)

    # TODO(jayconrod): sanitize inputs passed to git.
    revision = ctx.attr.commit if ctx.attr.commit else ctx.attr.tag
    _check_execute(ctx, ["git", "clone", "-n", ctx.attr.remote, "."], "failed to clone %s" % ctx.attr.remote)
    _check_execute(ctx, ["git", "checkout", revision], "failed to checkout revision %s in remote %s" % (revision, ctx.attr.remote))

    _apply_overlay(ctx, overlay)

git_repository = repository_rule(
    implementation = _git_repository_impl,
    doc = """
**NOTE:** `git_repository` is deprecated in favor of the rule of the same name
in [@bazel_tools//tools/build_defs/repo:git.bzl].

`git_repository` downloads a project with git. It has the same features as the
[native git_repository rule], but it also allows you to copy a set of files
into the repository after download. This is particularly useful for placing
pre-generated build files.

**Example**

```starlark
load("@bazel_gazelle//:deps.bzl", "git_repository")

git_repository(
    name = "com_github_pkg_errors",
    remote = "https://github.com/pkg/errors",
    commit = "816c9085562cd7ee03e7f8188a1cfd942858cded",
    overlay = {
        "@my_repo//third_party:com_github_pkg_errors/BUILD.bazel.in" : "BUILD.bazel",
    },
)
```
""",
    attrs = {
        "commit": attr.string(
            doc = "The git commit to check out. Either `commit` or `tag` may be specified.",
        ),
        "remote": attr.string(
            doc = "The remote repository to download.",
            mandatory = True,
        ),
        "tag": attr.string(
            doc = "The git tag to check out. Either `commit` or `tag` may be specified.",
        ),
        "overlay": attr.label_keyed_string_dict(
            allow_files = True,
            doc = """A set of files to copy into the downloaded repository. The keys in this
dictionary are Bazel labels that point to the files to copy. These must be
fully qualified labels (i.e., `@repo//pkg:name`) because relative labels
are interpreted in the checked out repository, not the repository containing
the WORKSPACE file. The values in this dictionary are root-relative paths
where the overlay files should be written.

It's convenient to store the overlay dictionaries for all repositories in
a separate .bzl file. See Gazelle's `manifest.bzl`_ for an example.""",
        ),
    },
)

def _resolve_overlay(ctx, overlay):
    """Resolve overlay labels to paths.

    This should be done before downloading the repository, since it may
    trigger restarts.
    """
    return [(ctx.path(src_label), dst_rel) for src_label, dst_rel in overlay.items()]

def _apply_overlay(ctx, overlay):
    """Copies overlay files into the repository.

    This should be done after downloading the repository, since it may replace
    downloaded files.
    """

    # TODO(jayconrod): sanitize destination paths.
    for src_path, dst_rel in overlay:
        ctx.template(dst_rel, src_path)

def _check_execute(ctx, arguments, message):
    res = ctx.execute(arguments)
    if res.return_code != 0:
        fail(message + "\n" + res.stdout + res.stderr)
