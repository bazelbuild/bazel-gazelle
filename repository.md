<!-- Generated with Stardoc: http://skydoc.bazel.build -->


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


<a id="git_repository"></a>

## git_repository

<pre>
git_repository(<a href="#git_repository-name">name</a>, <a href="#git_repository-commit">commit</a>, <a href="#git_repository-overlay">overlay</a>, <a href="#git_repository-remote">remote</a>, <a href="#git_repository-repo_mapping">repo_mapping</a>, <a href="#git_repository-tag">tag</a>)
</pre>


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


**ATTRIBUTES**


| Name  | Description | Type | Mandatory | Default |
| :------------- | :------------- | :------------- | :------------- | :------------- |
| <a id="git_repository-name"></a>name |  A unique name for this repository.   | <a href="https://bazel.build/concepts/labels#target-names">Name</a> | required |  |
| <a id="git_repository-commit"></a>commit |  The git commit to check out. Either <code>commit</code> or <code>tag</code> may be specified.   | String | optional | <code>""</code> |
| <a id="git_repository-overlay"></a>overlay |  A set of files to copy into the downloaded repository. The keys in this dictionary are Bazel labels that point to the files to copy. These must be fully qualified labels (i.e., <code>@repo//pkg:name</code>) because relative labels are interpreted in the checked out repository, not the repository containing the WORKSPACE file. The values in this dictionary are root-relative paths where the overlay files should be written.<br><br>It's convenient to store the overlay dictionaries for all repositories in a separate .bzl file. See Gazelle's <code>manifest.bzl</code>_ for an example.   | <a href="https://bazel.build/rules/lib/dict">Dictionary: Label -> String</a> | optional | <code>{}</code> |
| <a id="git_repository-remote"></a>remote |  The remote repository to download.   | String | required |  |
| <a id="git_repository-repo_mapping"></a>repo_mapping |  A dictionary from local repository name to global repository name. This allows controls over workspace dependency resolution for dependencies of this repository.&lt;p&gt;For example, an entry <code>"@foo": "@bar"</code> declares that, for any time this repository depends on <code>@foo</code> (such as a dependency on <code>@foo//some:target</code>, it should actually resolve that dependency within globally-declared <code>@bar</code> (<code>@bar//some:target</code>).   | <a href="https://bazel.build/rules/lib/dict">Dictionary: String -> String</a> | required |  |
| <a id="git_repository-tag"></a>tag |  The git tag to check out. Either <code>commit</code> or <code>tag</code> may be specified.   | String | optional | <code>""</code> |


<a id="go_repository"></a>

## go_repository

<pre>
go_repository(<a href="#go_repository-name">name</a>, <a href="#go_repository-auth_patterns">auth_patterns</a>, <a href="#go_repository-build_config">build_config</a>, <a href="#go_repository-build_directives">build_directives</a>, <a href="#go_repository-build_external">build_external</a>, <a href="#go_repository-build_extra_args">build_extra_args</a>,
              <a href="#go_repository-build_file_generation">build_file_generation</a>, <a href="#go_repository-build_file_name">build_file_name</a>, <a href="#go_repository-build_file_proto_mode">build_file_proto_mode</a>, <a href="#go_repository-build_naming_convention">build_naming_convention</a>,
              <a href="#go_repository-build_tags">build_tags</a>, <a href="#go_repository-canonical_id">canonical_id</a>, <a href="#go_repository-commit">commit</a>, <a href="#go_repository-debug_mode">debug_mode</a>, <a href="#go_repository-importpath">importpath</a>, <a href="#go_repository-patch_args">patch_args</a>, <a href="#go_repository-patch_cmds">patch_cmds</a>,
              <a href="#go_repository-patch_tool">patch_tool</a>, <a href="#go_repository-patches">patches</a>, <a href="#go_repository-remote">remote</a>, <a href="#go_repository-replace">replace</a>, <a href="#go_repository-repo_mapping">repo_mapping</a>, <a href="#go_repository-sha256">sha256</a>, <a href="#go_repository-strip_prefix">strip_prefix</a>, <a href="#go_repository-sum">sum</a>, <a href="#go_repository-tag">tag</a>,
              <a href="#go_repository-type">type</a>, <a href="#go_repository-urls">urls</a>, <a href="#go_repository-vcs">vcs</a>, <a href="#go_repository-version">version</a>)
</pre>


`go_repository` downloads a Go project and generates build files with Gazelle
if they are not already present. This is the simplest way to depend on
external Go projects.

When `go_repository` is in module mode, it saves downloaded modules in a shared,
internal cache within Bazel's cache. It may be cleared with `bazel clean --expunge`.
By setting the environment variable `GO_REPOSITORY_USE_HOST_CACHE=1`, you can
force `go_repository` to use the module cache on the host system in the location
returned by `go env GOPATH`.

**Example**

```starlark
load("@bazel_gazelle//:deps.bzl", "go_repository")

# Download using "go mod download"
go_repository(
    name = "com_github_pkg_errors",
    importpath = "github.com/pkg/errors",
    sum = "h1:iURUrRGxPUNPdy5/HRSm+Yj6okJ6UtLINN0Q9M4+h3I=",
    version = "v0.8.1",
)

# Download automatically via git
go_repository(
    name = "com_github_pkg_errors",
    commit = "816c9085562cd7ee03e7f8188a1cfd942858cded",
    importpath = "github.com/pkg/errors",
)

# Download from git fork
go_repository(
    name = "com_github_pkg_errors",
    commit = "816c9085562cd7ee03e7f8188a1cfd942858cded",
    importpath = "github.com/pkg/errors",
    remote = "https://example.com/fork/github.com/pkg/errors",
    vcs = "git",
)

# Download via HTTP
go_repository(
    name = "com_github_pkg_errors",
    importpath = "github.com/pkg/errors",
    urls = ["https://codeload.github.com/pkg/errors/zip/816c9085562cd7ee03e7f8188a1cfd942858cded"],
    strip_prefix = "errors-816c9085562cd7ee03e7f8188a1cfd942858cded",
    type = "zip",
)

# Download major version suffixed via git
go_repository(
    name = "com_github_thediveo_enumflag_v2",
    commit = "0217df583bf3d37b92798602e5061b36556bcd38",
    importpath = "github.com/thediveo/enumflag/v2",
    remote = "https://github.com/thediveo/enumflag",
    vcs = "git",
)
```



**ATTRIBUTES**


| Name  | Description | Type | Mandatory | Default |
| :------------- | :------------- | :------------- | :------------- | :------------- |
| <a id="go_repository-name"></a>name |  A unique name for this repository.   | <a href="https://bazel.build/concepts/labels#target-names">Name</a> | required |  |
| <a id="go_repository-auth_patterns"></a>auth_patterns |  An optional dict mapping host names to custom authorization patterns.<br><br>If a URL's host name is present in this dict the value will be used as a pattern when generating the authorization header for the http request. This enables the use of custom authorization schemes used in a lot of common cloud storage providers.<br><br>The pattern currently supports 2 tokens: &lt;code&gt;&lt;login&gt;&lt;/code&gt; and &lt;code&gt;&lt;password&gt;&lt;/code&gt;, which are replaced with their equivalent value in the netrc file for the same host name. After formatting, the result is set as the value for the &lt;code&gt;Authorization&lt;/code&gt; field of the HTTP request.<br><br>Example attribute and netrc for a http download to an oauth2 enabled API using a bearer token:<br><br>&lt;pre&gt; auth_patterns = {     "storage.cloudprovider.com": "Bearer &lt;password&gt;" } &lt;/pre&gt;<br><br>netrc:<br><br>&lt;pre&gt; machine storage.cloudprovider.com         password RANDOM-TOKEN &lt;/pre&gt;<br><br>The final HTTP request would have the following header:<br><br>&lt;pre&gt; Authorization: Bearer RANDOM-TOKEN &lt;/pre&gt;   | <a href="https://bazel.build/rules/lib/dict">Dictionary: String -> String</a> | optional | <code>{}</code> |
| <a id="go_repository-build_config"></a>build_config |  A file that Gazelle should read to learn about external repositories before             generating build files. This is useful for dependency resolution. For example,             a <code>go_repository</code> rule in this file establishes a mapping between a             repository name like <code>golang.org/x/tools</code> and a workspace name like             <code>org_golang_x_tools</code>. Workspace directives like             <code># gazelle:repository_macro</code> are recognized.<br><br>            <code>go_repository</code> rules will be re-evaluated when parts of WORKSPACE related             to Gazelle's configuration are changed, including Gazelle directives and             <code>go_repository</code> <code>name</code> and <code>importpath</code> attributes.             Their content should still be fetched from a local cache, but build files             will be regenerated. If this is not desirable, <code>build_config</code> may be set             to a less frequently updated file or <code>None</code> to disable this functionality.   | <a href="https://bazel.build/concepts/labels">Label</a> | optional | <code>@bazel_gazelle_go_repository_config//:WORKSPACE</code> |
| <a id="go_repository-build_directives"></a>build_directives |  A list of directives to be written to the root level build file before             Calling Gazelle to generate build files. Each string in the list will be             prefixed with <code>#</code> automatically. A common use case is to pass a list of             Gazelle directives.   | List of strings | optional | <code>[]</code> |
| <a id="go_repository-build_external"></a>build_external |  One of <code>"external"</code>, <code>"static"</code> or <code>"vendored"</code>.<br><br>            This sets Gazelle's <code>-external</code> command line flag. In <code>"static"</code> mode,             Gazelle will not call out to the network to resolve imports.<br><br>            **NOTE:** This cannot be used to ignore the <code>vendor</code> directory in a             repository. The <code>-external</code> flag only controls how Gazelle resolves             imports which are not present in the repository. Use             <code>build_extra_args = ["-exclude=vendor"]</code> instead.   | String | optional | <code>"static"</code> |
| <a id="go_repository-build_extra_args"></a>build_extra_args |  A list of additional command line arguments to pass to Gazelle when generating build files.   | List of strings | optional | <code>[]</code> |
| <a id="go_repository-build_file_generation"></a>build_file_generation |  One of <code>"auto"</code>, <code>"on"</code>, <code>"off"</code>.<br><br>            Whether Gazelle should generate build files in the repository. In <code>"auto"</code>             mode, Gazelle will run if there is no build file in the repository root             directory.   | String | optional | <code>"auto"</code> |
| <a id="go_repository-build_file_name"></a>build_file_name |  Comma-separated list of names Gazelle will consider to be build files.             If a repository contains files named <code>build</code> that aren't related to Bazel,             it may help to set this to <code>"BUILD.bazel"</code>, especially on case-insensitive             file systems.   | String | optional | <code>"BUILD.bazel,BUILD"</code> |
| <a id="go_repository-build_file_proto_mode"></a>build_file_proto_mode |  One of <code>"default"</code>, <code>"legacy"</code>, <code>"disable"</code>, <code>"disable_global"</code> or <code>"package"</code>.<br><br>            This sets Gazelle's <code>-proto</code> command line flag. See [Directives] for more             information on each mode.   | String | optional | <code>""</code> |
| <a id="go_repository-build_naming_convention"></a>build_naming_convention |  Sets the library naming convention to use when resolving dependencies against this external             repository. If unset, the convention from the external workspace is used.             Legal values are <code>go_default_library</code>, <code>import</code>, and <code>import_alias</code>.<br><br>            See the <code>gazelle:go_naming_convention</code> directive in [Directives] for more information.   | String | optional | <code>"import_alias"</code> |
| <a id="go_repository-build_tags"></a>build_tags |  This sets Gazelle's <code>-build_tags</code> command line flag.   | List of strings | optional | <code>[]</code> |
| <a id="go_repository-canonical_id"></a>canonical_id |  If the repository is downloaded via HTTP (<code>urls</code> is set) and this is set, restrict cache hits to those cases where the             repository was added to the cache with the same canonical id.   | String | optional | <code>""</code> |
| <a id="go_repository-commit"></a>commit |  If the repository is downloaded using a version control tool, this is the             commit or revision to check out. With git, this would be a sha1 commit id.             <code>commit</code> and <code>tag</code> may not both be set.   | String | optional | <code>""</code> |
| <a id="go_repository-debug_mode"></a>debug_mode |  Enables logging of fetch_repo and Gazelle output during succcesful runs. Gazelle can be noisy             so this defaults to <code>False</code>. However, setting to <code>True</code> can be useful for debugging build failures and             unexpected behavior for the given rule.   | Boolean | optional | <code>False</code> |
| <a id="go_repository-importpath"></a>importpath |  The Go import path that matches the root directory of this repository.<br><br>            In module mode (when <code>version</code> is set), this must be the module path. If             neither <code>urls</code> nor <code>remote</code> is specified, <code>go_repository</code> will             automatically find the true path of the module, applying import path             redirection.<br><br>            If build files are generated for this repository, libraries will have their             <code>importpath</code> attributes prefixed with this <code>importpath</code> string.   | String | required |  |
| <a id="go_repository-patch_args"></a>patch_args |  Arguments passed to the patch tool when applying patches.   | List of strings | optional | <code>["-p0"]</code> |
| <a id="go_repository-patch_cmds"></a>patch_cmds |  Commands to run in the repository after patches are applied.   | List of strings | optional | <code>[]</code> |
| <a id="go_repository-patch_tool"></a>patch_tool |  The patch tool used to apply <code>patches</code>. If this is specified, Bazel will             use the specifed patch tool instead of the Bazel-native patch implementation.   | String | optional | <code>""</code> |
| <a id="go_repository-patches"></a>patches |  A list of patches to apply to the repository after gazelle runs.   | <a href="https://bazel.build/concepts/labels">List of labels</a> | optional | <code>[]</code> |
| <a id="go_repository-remote"></a>remote |  The VCS location where the repository should be downloaded from. This is             usually inferred from <code>importpath</code>, but you can set <code>remote</code> to download             from a private repository or a fork.   | String | optional | <code>""</code> |
| <a id="go_repository-replace"></a>replace |  A replacement for the module named by <code>importpath</code>. The module named by             <code>replace</code> will be downloaded at <code>version</code> and verified with <code>sum</code>.<br><br>            NOTE: There is no <code>go_repository</code> equivalent to file path <code>replace</code>             directives. Use <code>local_repository</code> instead.   | String | optional | <code>""</code> |
| <a id="go_repository-repo_mapping"></a>repo_mapping |  A dictionary from local repository name to global repository name. This allows controls over workspace dependency resolution for dependencies of this repository.&lt;p&gt;For example, an entry <code>"@foo": "@bar"</code> declares that, for any time this repository depends on <code>@foo</code> (such as a dependency on <code>@foo//some:target</code>, it should actually resolve that dependency within globally-declared <code>@bar</code> (<code>@bar//some:target</code>).   | <a href="https://bazel.build/rules/lib/dict">Dictionary: String -> String</a> | required |  |
| <a id="go_repository-sha256"></a>sha256 |  If the repository is downloaded via HTTP (<code>urls</code> is set), this is the             SHA-256 sum of the downloaded archive. When set, Bazel will verify the archive             against this sum before extracting it.<br><br>            **CAUTION:** Do not use this with services that prepare source archives on             demand, such as codeload.github.com. Any minor change in the server software             can cause differences in file order, alignment, and compression that break             SHA-256 sums.   | String | optional | <code>""</code> |
| <a id="go_repository-strip_prefix"></a>strip_prefix |  If the repository is downloaded via HTTP (<code>urls</code> is set), this is a             directory prefix to strip. See [<code>http_archive.strip_prefix</code>].   | String | optional | <code>""</code> |
| <a id="go_repository-sum"></a>sum |  A hash of the module contents. In module mode, <code>go_repository</code> will verify             the downloaded module matches this sum. May only be set when <code>version</code>             is also set.<br><br>            A value for <code>sum</code> may be found in the <code>go.sum</code> file or by running             <code>go mod download -json &lt;module&gt;@&lt;version&gt;</code>.   | String | optional | <code>""</code> |
| <a id="go_repository-tag"></a>tag |  If the repository is downloaded using a version control tool, this is the             named revision to check out. <code>commit</code> and <code>tag</code> may not both be set.   | String | optional | <code>""</code> |
| <a id="go_repository-type"></a>type |  One of <code>"zip"</code>, <code>"tar.gz"</code>, <code>"tgz"</code>, <code>"tar.bz2"</code>, <code>"tar.xz"</code>.<br><br>            If the repository is downloaded via HTTP (<code>urls</code> is set), this is the             file format of the repository archive. This is normally inferred from the             downloaded file name.   | String | optional | <code>""</code> |
| <a id="go_repository-urls"></a>urls |  A list of HTTP(S) URLs where an archive containing the project can be             downloaded. Bazel will attempt to download from the first URL; the others             are mirrors.   | List of strings | optional | <code>[]</code> |
| <a id="go_repository-vcs"></a>vcs |  One of <code>"git"</code>, <code>"hg"</code>, <code>"svn"</code>, <code>"bzr"</code>.<br><br>            The version control system to use. This is usually determined automatically,             but it may be necessary to set this when <code>remote</code> is set and the VCS cannot             be inferred. You must have the corresponding tool installed on your host.   | String | optional | <code>""</code> |
| <a id="go_repository-version"></a>version |  If specified, <code>go_repository</code> will download the module at this version             using <code>go mod download</code>. <code>sum</code> must also be set. <code>commit</code>, <code>tag</code>,             and <code>urls</code> may not be set.   | String | optional | <code>""</code> |


<a id="http_archive"></a>

## http_archive

<pre>
http_archive(<a href="#http_archive-name">name</a>, <a href="#http_archive-overlay">overlay</a>, <a href="#http_archive-repo_mapping">repo_mapping</a>, <a href="#http_archive-sha256">sha256</a>, <a href="#http_archive-strip_prefix">strip_prefix</a>, <a href="#http_archive-type">type</a>, <a href="#http_archive-urls">urls</a>)
</pre>


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


**ATTRIBUTES**


| Name  | Description | Type | Mandatory | Default |
| :------------- | :------------- | :------------- | :------------- | :------------- |
| <a id="http_archive-name"></a>name |  A unique name for this repository.   | <a href="https://bazel.build/concepts/labels#target-names">Name</a> | required |  |
| <a id="http_archive-overlay"></a>overlay |  A set of files to copy into the downloaded repository. The keys in this             dictionary are Bazel labels that point to the files to copy. These must be             fully qualified labels (i.e., <code>@repo//pkg:name</code>) because relative labels             are interpreted in the checked out repository, not the repository containing             the WORKSPACE file. The values in this dictionary are root-relative paths             where the overlay files should be written.<br><br>            It's convenient to store the overlay dictionaries for all repositories in             a separate .bzl file. See Gazelle's <code>manifest.bzl</code>_ for an example.   | <a href="https://bazel.build/rules/lib/dict">Dictionary: Label -> String</a> | optional | <code>{}</code> |
| <a id="http_archive-repo_mapping"></a>repo_mapping |  A dictionary from local repository name to global repository name. This allows controls over workspace dependency resolution for dependencies of this repository.&lt;p&gt;For example, an entry <code>"@foo": "@bar"</code> declares that, for any time this repository depends on <code>@foo</code> (such as a dependency on <code>@foo//some:target</code>, it should actually resolve that dependency within globally-declared <code>@bar</code> (<code>@bar//some:target</code>).   | <a href="https://bazel.build/rules/lib/dict">Dictionary: String -> String</a> | required |  |
| <a id="http_archive-sha256"></a>sha256 |  The SHA-256 sum of the downloaded archive. When set, Bazel will verify the             archive against this sum before extracting it.<br><br>            **CAUTION:** Do not use this with services that prepare source archives on             demand, such as codeload.github.com. Any minor change in the server software             can cause differences in file order, alignment, and compression that break             SHA-256 sums.   | String | optional | <code>""</code> |
| <a id="http_archive-strip_prefix"></a>strip_prefix |  A directory prefix to strip. See [http_archive.strip_prefix].   | String | optional | <code>""</code> |
| <a id="http_archive-type"></a>type |  One of <code>"zip"</code>, <code>"tar.gz"</code>, <code>"tgz"</code>, <code>"tar.bz2"</code>, <code>"tar.xz"</code>.<br><br>            The file format of the repository archive. This is normally inferred from             the downloaded file name.   | String | optional | <code>""</code> |
| <a id="http_archive-urls"></a>urls |  A list of HTTP(S) URLs where the project can be downloaded. Bazel will             attempt to download the first URL; the others are mirrors.   | List of strings | optional | <code>[]</code> |


