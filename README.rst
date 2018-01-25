Gazelle build file generator
============================

.. All external links are here
.. _go_repository: https://github.com/bazelbuild/rules_go/blob/master/go/workspace.rst#go-repository
.. _Gazelle in rules_go: https://github.com/bazelbuild/rules_go/tree/master/go/tools/gazelle
.. _fix: #fix-and-update
.. _update: #fix-and-update

.. role:: cmd(code)
.. role:: flag(code)
.. role:: param(kbd)
.. role:: type(emphasis)
.. role:: value(code)
.. |mandatory| replace:: **mandatory value**
.. End of directives

Gazelle is a build file generator for Go projects. It can create new
BUILD.bazel files for a project that follows "go build" conventions, and it
can update existing build files to include new files and options. Gazelle can
be invoked directly in a project workspace, or it can be run on an external
repository during the build as part of the go_repository_ rule.

*Gazelle is under active development. Its interface and the rules it generates
may change. Gazelle is not an official Google product.*

.. contents:: **Contents** 
  :depth: 2

Setup
-----

Running Gazelle with Bazel
~~~~~~~~~~~~~~~~~~~~~~~~~~

To use Gazelle in a new project, add the ``bazel_gazelle`` repository and its
dependencies to your WORKSPACE file before ``go_rules_dependencies`` is called.
It should look like this:

.. code:: bzl

    http_archive(
        name = "io_bazel_rules_go",
        url = "https://github.com/bazelbuild/rules_go/releases/download/0.9.0/rules_go-0.9.0.tar.gz",
        sha256 = "4d8d6244320dd751590f9100cf39fd7a4b75cd901e1f3ffdfd6f048328883695",
    )
    http_archive(
        name = "bazel_gazelle",
        url = "https://github.com/bazelbuild/bazel-gazelle/releases/download/0.9/bazel-gazelle-0.9.tar.gz",
        sha256 = "0103991d994db55b3b5d7b06336f8ae355739635e0c2379dea16b8213ea5a223",
    )
    load("@io_bazel_rules_go//go:def.bzl", "go_rules_dependencies", "go_register_toolchains")
    go_rules_dependencies()
    go_register_toolchains()
    load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies")
    gazelle_dependencies()
      
Add the code below to the BUILD or BUILD.bazel file in the root directory
of your repository. Replace the string in ``prefix`` with the portion of
your import path that corresponds to your repository.

.. code:: bzl
  
  load("@bazel_gazelle//:def.bzl", "gazelle")

  gazelle(
      name = "gazelle",
      prefix = "github.com/example/project",
  )

After adding this code, you can run Gazelle with Bazel.

.. code::

  bazel run //:gazelle

This will generate new BUILD.bazel files for your project. You can run the same
command in the future to update existing BUILD.bazel files to include new source
files or options.

Running Gazelle with Go
~~~~~~~~~~~~~~~~~~~~~~~

If you have a Go SDK installed, you can install Gazelle in your ``GOPATH`` with
the command below:

.. code::

  go get -u github.com/bazelbuild/bazel-gazelle/cmd/gazelle

Make sure to re-run this command to upgrade Gazelle whenever you upgrade
rules_go in your repository.

To generate BUILD.bazel files in a new project, run the command below, replacing
the prefix with the portion of your import path that corresponds to your
repository.

.. code::

  gazelle -go_prefix github.com/my/project

The prefix only needs to be specified the first time you run Gazelle. To update
existing BUILD.bazel files, you can just run ``gazelle`` without arguments.

Compatibility
-------------

Gazelle generates build files that require a minimum version of ``rules_go``
to build. Check the table below to ensure that you're using compatible versions.

+---------------------+------------------------------+------------------------------+
| **Gazelle version** | **Minimum rules_go version** | **Maximum rules_go version** |
+=====================+==============================+==============================+
| 0.8                 | 0.8.0                        | n/a                          |
+---------------------+------------------------------+------------------------------+
| 0.9                 | 0.9.0                        | n/a                          |
+---------------------+------------------------------+------------------------------+

Usage
-----

Command line
~~~~~~~~~~~~

.. code::

  gazelle <command> [flags...] [package-dirs...]

The first argument to Gazelle may be one of the commands below. If no command
is specified, ``update`` is assumed. The remaining arguments are specific
to each command and are documented below.

update_
  Scans sources files, then generates and updates build files.

fix_
  Same as the ``update`` command, but it also fixes deprecated usage of rules.

update-repos_
  Updates repository rules in the WORKSPACE file.

Bazel rule
~~~~~~~~~~

Gazelle may be run via a rule. See `Running Gazelle with Bazel`_ for setup
instructions. This rule builds Gazelle and generates a wrapper script that
executes Gazelle with baked-in set of arguments. You can run this script
with ``bazel run``, or you can copy it into your workspace and run it directly.

The following attributes are available on the ``gazelle`` rule.

+----------------------+---------------------+--------------------------------------+
| **Name**             | **Type**            | **Default value**                    |
+======================+=====================+======================================+
| :param:`gazelle`     | :type:`label`       | :value:`@bazel_gazelle//cmd/gazelle` |
+----------------------+---------------------+--------------------------------------+
| The ``go_binary`` rule that builds Gazelle. You can substitute a modified         |
| version of Gazelle with this.                                                     |
+----------------------+---------------------+--------------------------------------+
| :param:`external`    | :type:`string`      | :value:`external`                    |
+----------------------+---------------------+--------------------------------------+
| The method for resolving unknown imports to Bazel dependencies. May be            |
| :value:`external` or :value:`vendored`.                                           |
+----------------------+---------------------+--------------------------------------+
| :param:`build_tags`  | :type:`string_list` | :value:`[]`                          |
+----------------------+---------------------+--------------------------------------+
| The last of Go build tags that Gazelle should consider to always be true.         |
+----------------------+---------------------+--------------------------------------+
| :param:`prefix`      | :type:`string`      | |mandatory|                          |
+----------------------+---------------------+--------------------------------------+
| The import path that corresponds to the repository root directory.                |
| TODO(#26): this should be optional.                                               |
+----------------------+---------------------+--------------------------------------+
| :param:`extra_args`  | :type:`string_list` | :value:`[]`                          |
+----------------------+---------------------+--------------------------------------+
| A list of extra command line arguments passed to Gazelle.                         |
+----------------------+---------------------+--------------------------------------+
| :param:`command`     | :type:`string`      | :value:`update`                      |
+----------------------+---------------------+--------------------------------------+
| The Gazelle command to use. May be :value:`fix` or :value:`update`. To run        |
| a different command, e.g., :value:`update-repos`, you'll need to copy the         |
| invoke the generated wrapper script directly with explicit arguments.             |
+----------------------+---------------------+--------------------------------------+

``fix`` and ``update``
~~~~~~~~~~~~~~~~~~~~~~

The ``update`` command is the most common way of running Gazelle. Gazelle will
scan sources in directories throughout the repository, then create and update
build files.

The ``fix`` command does everything ``update`` does, but it also fixes
deprecated usage of rules, analogous to ``go fix``. For example, ``cgo_library``
will be consolidated with ``go_library``. This command may delete or rename
rules, so it's not on by default. See `Fix command transformations`_
for details.

Both commands accept a list of directories to process as positional arguments.
If no directories are specified, Gazelle will process the current directory.
Subdirectories will be processed recursively.

The following flags are accepted:

+------------------------------------------+-----------------------------------+
| **Name**                                 | **Default value**                 |
+==========================================+===================================+
| :flag:`-build_file_name file1,file2,...` | :value:`BUILD.bazel,BUILD`        |
+------------------------------------------+-----------------------------------+
| Comma-separated list of file names. Gazelle recognizes these files as Bazel  |
| build files. New files will use the first name in this list. Use this if     |
| your project contains non-Bazel files named ``BUILD`` (or ``build`` on       |
| case-insensitive file systems).                                              |
+------------------------------------------+-----------------------------------+
| :flag:`-build_tags tag1,tag2`            |                                   |
+------------------------------------------+-----------------------------------+
| List of Go build tags Gazelle will consider to be true. Gazelle applies      |
| constraints when generating Go rules. It assumes certain tags are true on    |
| certain platforms (for example, ``amd64,linux``). It assumes all Go release  |
| tags are true (for example, ``go1.8``). It considers other tags to be false  |
| (for example, ``ignore``). This flag overrides that behavior.                |
+------------------------------------------+-----------------------------------+
| :flag:`-external external|vendored`      | :value:`external`                 |
+------------------------------------------+-----------------------------------+
| Determines how Gazelle resolves import paths. May be :value:`external` or    |
| :value:`vendored`. Gazelle translates Go import paths to Bazel labels when   |
| resolving library dependencies. Import paths that start with the             |
| ``go_prefix`` are resolved to local labels, but other imports                |
| are resolved based on this mode. In :value:`external` mode, paths are        |
| resolved using an external dependency in the WORKSPACE file (Gazelle does    |
| not create or maintain these dependencies yet). In :value:`vendored` mode,   |
| paths are resolved to a library in the vendor directory.                     |
+------------------------------------------+-----------------------------------+
| :flag:`-go_prefix example.com/repo`      |                                   |
+------------------------------------------+-----------------------------------+
| A prefix of import paths for libraries in the repository that corresponds to |
| the repository root. Gazelle infers this from the ``go_prefix`` rule in the  |
| root BUILD.bazel file, if it exists. If not, this option is mandatory.       |
|                                                                              |
| This prefix is used to determine whether an import path refers to a library  |
| in the current repository or an external dependency.                         |
+------------------------------------------+-----------------------------------+
| :flag:`-known_import example.com`        |                                   |
+------------------------------------------+-----------------------------------+
| Skips import path resolution for a known domain. May be repeated.            |
|                                                                              |
| When Gazelle resolves an import path to an external dependency, it attempts  |
| to discover the remote repository root over HTTP. Gazelle skips this         |
| discovery step for a few well-known domains with predictable structure, like |
| golang.org and github.com. This flag specifies additional domains to skip,   |
| which is useful in situations where the lookup would fail for some reason.   |
+------------------------------------------+-----------------------------------+
| :flag:`-mode fix|print|diff`             | :value:`fix`                      |
+------------------------------------------+-----------------------------------+
| Method for emitting merged build files.                                      |
|                                                                              |
| In ``fix`` mode, Gazelle writes generated and merged files to disk. In       |
| ``print`` mode, it prints them to stdout. In ``diff`` mode, it prints a      |
| unified diff.                                                                |
+------------------------------------------+-----------------------------------+
| :flag:`-proto default|legacy|disable`    | :value:`default`                  |
+------------------------------------------+-----------------------------------+
| Determines how Gazelle should generate rules for .proto files. See details   |
| in `Directives`_ below.                                                      |
+------------------------------------------+-----------------------------------+
| :flag:`-repo_root dir`                   |                                   |
+------------------------------------------+-----------------------------------+
| The root directory of the repository. Gazelle normally infers this to be the |
| directory containing the WORKSPACE file.                                     |
|                                                                              |
| Gazelle will not process packages outside this directory.                    |
+------------------------------------------+-----------------------------------+

``update-repos``
~~~~~~~~~~~~~~~~

The ``update-repos`` command updates repository rules in the WORKSPACE file.
Currently, this can only be used to import repositories from a vendoring tool's
lock file. More functionality will be added in the future.

The following flags are accepted:

+------------------------------+-----------------------------------------------+
| **Name**                     | **Default value**                             |
+==============================+===============================================+
| :flag:`-from_file lock-file` |                                               |
+------------------------------+-----------------------------------------------+
| Import repositories from a vendoring tool's lock file as `go_repository`_    |
| rules. These rules will be added to the bottom of WORKSPACE or merged with   |
| existing rules.                                                              |
|                                                                              |
| The lock file format is inferred from the file's base name. Currently, only  |
| Gopkg.lock is supported.                                                     |
+------------------------------+-----------------------------------------------+
| :flag:`-repo_root dir`       |                                               |
+------------------------------+-----------------------------------------------+
| The root directory of the repository. Gazelle normally infers this to be the |
| directory containing the WORKSPACE file.                                     |
|                                                                              |
| Gazelle will not process packages outside this directory.                    |
+------------------------------+-----------------------------------------------+

Bazel rule
~~~~~~~~~~

When Gazelle is run by Bazel, most of the flags above can be encoded in the
``gazelle`` rule. For example:

.. code:: bzl

  load("@bazel_gazelle//:def.bzl", "gazelle")

  gazelle(
      name = "gazelle",
      command = "fix",
      prefix = "github.com/example/project",
      external = "vendored",
      build_tags = [
          "integration",
          "debug",
      ],
      extra_args = [
          "-build_file_name",
          "BUILD,BUILD.bazel",
      ],
  )

Directives
~~~~~~~~~~

Gazelle supports several directives, written as comments in build files.

* ``# gazelle:ignore``: may be written at the top level of any build file.
  Gazelle will not update files with this comment.
* ``# gazelle:exclude path``: may be written at the top level of
  any build file. If the path refers to a source file, Gazelle won't include
  it in any rules. If the path refers to a directory, Gazelle won't recurse
  into it. The path may refer to something in a subdirectory, for example,
  a testdata directory somewhere in a vendor tree. This directive may be
  repeated to exclude multiple paths, one per line.
* ``# gazelle:proto <mode>``: Tells Gazelle how to generate rules for .proto
  files. Applies to the current directory and subdirectories. Valid values for
  ``mode`` are:

  * ``default``: ``proto_library``, ``go_proto_library``, ``go_grpc_library``,
    and ``go_library`` rules are generated using
    ``@io_bazel_rules_go//proto:def.bzl``. This is the default mode.
  * ``legacy``: ``filegroup`` rules are generated for use by
    ``@io_bazel_rules_go//proto:go_proto_library.bzl``. ``go_proto_library``
    rules must be written by hand. Gazelle will run in this mode automatically
    if ``go_proto_library.bzl`` is loaded to avoid disrupting existing
    projects, but this can be overridden with a directive.
  * ``disable``: .proto files are ignored. Gazelle will run in this mode
    automatically if ``go_proto_library`` is loaded from any other source,
    but this can be overridden with a directive.
* ``# keep``: may be written before a rule to prevent the rule from being
  updated or after a source file, dependency, or flag to prevent it from being
  removed.

Example
^^^^^^^

Suppose you have a library that includes a generated .go file. Gazelle won't
know what imports to resolve, so you may need to add dependencies manually with
``# keep`` comments.

.. code:: bzl

  load("@io_bazel_rules_go//go:def.bzl", "go_library")
  load("@com_github_example_gen//:gen.bzl", "gen_go_file")

  gen_go_file(
      name = "magic",
      srcs = ["magic.go.in"],
      outs = ["magic.go"],
  )

  go_library(
      name = "go_default_library",
      srcs = ["magic.go"],
      visibility = ["//visibility:public"],
      deps = [
          "@com_github_example_gen//:go_default_library",  # keep
      ],
  )

Fix command transformations
---------------------------

When Gazelle is invoked with the ``fix`` command, in addition to updating
source files and dependencies of existing rules, Gazelle will remove deprecated
usage of the Go rules, analogous to ``go fix``. The following transformations
are performed.

**Squash cgo libraries**: Gazelle will remove `cgo_library` rules named
``cgo_default_library`` and merge their attributes with a ``go_library`` rule
in the same package named ``go_default_library``. If no such ``go_library``
rule exists, a new one will be created. Other ``cgo_library`` rules will not
be removed.

.. code:: bzl
  # BEFORE
  go_library(
      name = "go_default_library",
      srcs = ["pure.go"],
      library = ":cgo_default_library",
  )

  cgo_library(
      name = "cgo_default_library",
      srcs = ["cgo.go"],
  )

  # AFTER
  go_library(
      name = "go_default_library",
      srcs = [
          "cgo.go",
          "pure.go",
      ],
      cgo = True,
  )

**Remove legacy protos**: Gazelle will remove usage of ``go_proto_library``
rules loaded from ``@io_bazel_rules_go//proto:go_proto_library.bzl`` and
``filegroup`` rules named ``go_default_library_protos``. Newly generated
proto rules will take their place. Since ``filegroup`` isn't needed anymore
and ``go_proto_library`` has different attributes and was always written by
hand, Gazelle will not attempt to merge anything from these rules with the
newly generated rules.

This transformation is only applied in the default proto mode. Since Gazelle
will run in legacy proto mode if ``go_proto_library.bzl`` is loaded, this
transformation is not usually applied. You can set the proto mode explicitly
using the directive ``# gazelle:proto default``.

.. code:: bzl
  # BEFORE
  # gazelle:proto default
  load("@io_bazel_rules_go//proto:go_proto_library.bzl", "go_proto_library")

  go_proto_library(
      name = "go_default_library",
      srcs = [":go_default_library_protos"],
  )

  filegroup(
      name = "go_default_library_protos",
      srcs = ["foo.proto"],
  )

  # AFTER
  # The above rules are deleted. New proto_library, go_proto_library, and
  # go_library rules will be generated automatically.
