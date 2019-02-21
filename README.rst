Gazelle build file generator
============================

.. All external links are here
.. _Architecture of Gazelle: Design.rst
.. _Repository rules: repository.rst
.. _go_repository: repository.rst#go_repository
.. _git_repository: repository.rst#git_repository
.. _http_archive: repository.rst#http_archive
.. _Gazelle in rules_go: https://github.com/bazelbuild/rules_go/tree/master/go/tools/gazelle
.. _fix: #fix-and-update
.. _update: #fix-and-update
.. _Avoiding conflicts with proto rules: https://github.com/bazelbuild/rules_go/blob/master/proto/core.rst#avoiding-conflicts
.. _gazelle rule: #bazel-rule
.. _Extending Gazelle: extend.rst
.. _extended: `Extending Gazelle`_
.. _gazelle_binary: extend.rst#gazelle_binary
.. _import_prefix: https://docs.bazel.build/versions/master/be/protocol-buffer.html#proto_library.import_prefix
.. _strip_import_prefix: https://docs.bazel.build/versions/master/be/protocol-buffer.html#proto_library.strip_import_prefix

.. role:: cmd(code)
.. role:: flag(code)
.. role:: direc(code)
.. role:: param(kbd)
.. role:: type(emphasis)
.. role:: value(code)
.. |mandatory| replace:: **mandatory value**
.. End of directives

Gazelle is a build file generator for Go projects. It can create new BUILD.bazel
files for a project that follows "go build" conventions, and it can update
existing build files to include new sources, dependencies, and options. Gazelle
may be run by Bazel using the `gazelle rule`_, or it can be run as a command
line tool. Gazelle can also be run in an external repository as part of the
`go_repository`_ rule. Gazelle may be extended_ to support new languages
and custom rule sets.

*Gazelle is under active development. Its interface and the rules it generates
may change. Gazelle is not an official Google product.*

.. contents:: **Contents**
  :depth: 2

**See also:**

* `Architecture of Gazelle`_
* `Repository rules`_

  * `go_repository`_
  * `git_repository`_ (deprecated)
  * `http_archive`_ (deprecated)

* `Extending Gazelle`_
* `Avoiding conflicts with proto rules`_

Setup
-----

Running Gazelle with Bazel
~~~~~~~~~~~~~~~~~~~~~~~~~~

To use Gazelle in a new project, add the ``bazel_gazelle`` repository and its
dependencies to your WORKSPACE file and call ``gazelle_dependencies``. It
should look like this:

.. code:: bzl

    load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")
    http_archive(
        name = "io_bazel_rules_go",
        urls = ["https://github.com/bazelbuild/rules_go/releases/download/0.17.0/rules_go-0.17.0.tar.gz"],
        sha256 = "492c3ac68ed9dcf527a07e6a1b2dcbf199c6bf8b35517951467ac32e421c06c1",
    )
    http_archive(
        name = "bazel_gazelle",
        urls = ["https://github.com/bazelbuild/bazel-gazelle/releases/download/0.16.0/bazel-gazelle-0.16.0.tar.gz"],
        sha256 = "7949fc6cc17b5b191103e97481cf8889217263acf52e00b560683413af204fcb",
    )
    load("@io_bazel_rules_go//go:deps.bzl", "go_rules_dependencies", "go_register_toolchains")
    go_rules_dependencies()
    go_register_toolchains()
    load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies")
    gazelle_dependencies()

Add the code below to the BUILD or BUILD.bazel file in the root directory
of your repository. Replace the string after ``prefix`` with the portion of
your import path that corresponds to your repository.

.. code:: bzl

  load("@bazel_gazelle//:def.bzl", "gazelle")

  # gazelle:prefix github.com/example/project
  gazelle(name = "gazelle")

After adding this code, you can run Gazelle with Bazel.

.. code::

  $ bazel run //:gazelle

This will generate new BUILD.bazel files for your project. You can run the same
command in the future to update existing BUILD.bazel files to include new source
files or options.

You can pass additional arguments to Gazelle after a ``--`` argument. This
can be used to run alternate commands like ``update-repos`` that the ``gazelle``
rule cannot run directly.

.. code::

  $ bazel run //:gazelle -- update-repos -from_file=go.mod

Running Gazelle with Go
~~~~~~~~~~~~~~~~~~~~~~~

If you have a Go SDK installed, you can install Gazelle with the command below:

.. code::

  go get -u github.com/bazelbuild/bazel-gazelle/cmd/gazelle

Make sure to re-run this command to upgrade Gazelle whenever you upgrade
rules_go in your repository.

To generate BUILD.bazel files in a new project, run the command below, replacing
the prefix with the portion of your import path that corresponds to your
repository.

.. code::

  gazelle -go_prefix github.com/example/project

Most of Gazelle's command-line arguments can be expressed as special comments
in build files. See Directives_ below. You may want to copy this line into
your root build files to avoid having to type ``-go_prefix`` every time.

.. code:: bzl

  # gazelle:prefix github.com/example/project

Compatibility
-------------

Gazelle generates build files that use features in newer versions of
``rules_go``. Newer versions of Gazelle *may* generate build files that work
with older versions of ``rules_go``, but check the table below to ensure
you're using a compatible version.

+---------------------+------------------------------+------------------------------+
| **Gazelle version** | **Minimum rules_go version** | **Maximum rules_go version** |
+=====================+==============================+==============================+
| 0.8                 | 0.8.0                        | n/a                          |
+---------------------+------------------------------+------------------------------+
| 0.9                 | 0.9.0                        | n/a                          |
+---------------------+------------------------------+------------------------------+
| 0.10.0              | 0.9.0                        | 0.11.0                       |
+---------------------+------------------------------+------------------------------+
| 0.11.0              | 0.11.0                       | n/a                          |
+---------------------+------------------------------+------------------------------+
| 0.12.0              | 0.11.0                       | n/a                          |
+---------------------+------------------------------+------------------------------+
| 0.13.0              | 0.13.0                       | n/a                          |
+---------------------+------------------------------+------------------------------+
| 0.14.0              | 0.13.0                       | n/a                          |
+---------------------+------------------------------+------------------------------+
| 0.15.0              | 0.13.0                       | n/a                          |
+---------------------+------------------------------+------------------------------+
| 0.16.0              | 0.13.0                       | n/a                          |
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
  Adds and updates repository rules in the WORKSPACE file.

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
| The `gazelle_binary`_ rule that builds Gazelle. You can substitute a modified     |
| version of Gazelle with this. See `Extending Gazelle`_.                           |
+----------------------+---------------------+--------------------------------------+
| :param:`external`    | :type:`string`      | :value:`external`                    |
+----------------------+---------------------+--------------------------------------+
| The method for resolving unknown imports to Bazel dependencies. May be            |
| :value:`external` or :value:`vendored`. See `Dependency resolution`_.             |
+----------------------+---------------------+--------------------------------------+
| :param:`build_tags`  | :type:`string_list` | :value:`[]`                          |
+----------------------+---------------------+--------------------------------------+
| The last of Go build tags that Gazelle should consider to always be true.         |
+----------------------+---------------------+--------------------------------------+
| :param:`prefix`      | :type:`string`      | :value:`""`                          |
+----------------------+---------------------+--------------------------------------+
| The import path that corresponds to the repository root directory.                |
|                                                                                   |
| Note: It's usually better to write a directive like                               |
| ``# gazelle:prefix example.com/repo`` in your build file instead of setting       |
| this attribute.                                                                   |
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

The ``update`` command is the most common way of running Gazelle. Gazelle
scans sources in directories throughout the repository, then creates and updates
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

+--------------------------------------------------------------+----------------------------------------+
| **Name**                                                     | **Default value**                      |
+==============================================================+========================================+
| :flag:`-build_file_name file1,file2,...`                     | :value:`BUILD.bazel,BUILD`             |
+--------------------------------------------------------------+----------------------------------------+
| Comma-separated list of file names. Gazelle recognizes these files as Bazel                           |
| build files. New files will use the first name in this list. Use this if                              |
| your project contains non-Bazel files named ``BUILD`` (or ``build`` on                                |
| case-insensitive file systems).                                                                       |
+--------------------------------------------------------------+----------------------------------------+
| :flag:`-build_tags tag1,tag2`                                |                                        |
+--------------------------------------------------------------+----------------------------------------+
| List of Go build tags Gazelle will consider to be true. Gazelle applies                               |
| constraints when generating Go rules. It assumes certain tags are true on                             |
| certain platforms (for example, ``amd64,linux``). It assumes all Go release                           |
| tags are true (for example, ``go1.8``). It considers other tags to be false                           |
| (for example, ``ignore``). This flag overrides that behavior.                                         |
|                                                                                                       |
| Bazel may still filter sources with these tags. Use                                                   |
| ``bazel build --features gotags=foo,bar`` to set tags at build time.                                  |
+--------------------------------------------------------------+----------------------------------------+
| :flag:`-exclude path`                                        |                                        |
+--------------------------------------------------------------+----------------------------------------+
| Prevents Gazelle from processing a file or directory. If the path refers to                           |
| a source file, Gazelle won't include it in any rules. If the path refers to                           |
| a directory, Gazelle won't recurse into it.                                                           |
|                                                                                                       |
| This option may be repeated. Paths must be slash-separated, relative to the                           |
| repository root. This is equivalent to the ``# gazelle:exclude path``                                 |
| directive.                                                                                            |
+--------------------------------------------------------------+----------------------------------------+
| :flag:`-external external|vendored`                          | :value:`external`                      |
+--------------------------------------------------------------+----------------------------------------+
| Determines how Gazelle resolves import paths that cannot be resolve in the                            |
| current repository. May be :value:`external` or :value:`vendored`. See                                |
| `Dependency resolution`_.                                                                             |
+--------------------------------------------------------------+----------------------------------------+
| :flag:`-index true|false`                                    | :value:`true`                          |
+--------------------------------------------------------------+----------------------------------------+
| Determines whether Galleze should index the libraries in the current repository and whether it        |
| should use the index to resolve dependencies. If this is switched off, Galleze would rely on          |
| ``# gazelle:prefix`` directive or ``-go_prefix`` flag to resolve dependencies.                        |
+--------------------------------------------------------------+----------------------------------------+
| :flag:`-go_grpc_compiler`                                    | ``@io_bazel_rules_go//proto:go_grpc``  |
+--------------------------------------------------------------+----------------------------------------+
| The protocol buffers compiler to use for building go bindings for gRPC. May be repeated.              |
|                                                                                                       |
| See `Predefined plugins`_ for available options; commonly used options include                        |
| ``@io_bazel_rules_go//proto:gofast_grpc`` and ``@io_bazel_rules_go//proto:gogofaster_grpc``.          |
+--------------------------------------------------------------+----------------------------------------+
| :flag:`-go_prefix example.com/repo`                          |                                        |
+--------------------------------------------------------------+----------------------------------------+
| A prefix of import paths for libraries in the repository that corresponds to                          |
| the repository root. Gazelle infers this from the ``go_prefix`` rule in the                           |
| root BUILD.bazel file, if it exists. If not, this option is mandatory.                                |
|                                                                                                       |
| This prefix is used to determine whether an import path refers to a library                           |
| in the current repository or an external dependency.                                                  |
+--------------------------------------------------------------+----------------------------------------+
| :flag:`-go_proto_compiler`                                   | ``@io_bazel_rules_go//proto:go_proto`` |
+--------------------------------------------------------------+----------------------------------------+
| The protocol buffers compiler to use for building go bindings. May be repeated.                       |
|                                                                                                       |
| See `Predefined plugins`_ for available options; commonly used options include                        |
| ``@io_bazel_rules_go//proto:gofast_proto`` and ``@io_bazel_rules_go//proto:gogofaster_proto``.        |
+--------------------------------------------------------------+----------------------------------------+
| :flag:`-known_import example.com`                            |                                        |
+--------------------------------------------------------------+----------------------------------------+
| Skips import path resolution for a known domain. May be repeated.                                     |
|                                                                                                       |
| When Gazelle resolves an import path to an external dependency, it attempts                           |
| to discover the remote repository root over HTTP. Gazelle skips this                                  |
| discovery step for a few well-known domains with predictable structure, like                          |
| golang.org and github.com. This flag specifies additional domains to skip,                            |
| which is useful in situations where the lookup would fail for some reason.                            |
+--------------------------------------------------------------+----------------------------------------+
| :flag:`-mode fix|print|diff`                                 | :value:`fix`                           |
+--------------------------------------------------------------+----------------------------------------+
| Method for emitting merged build files.                                                               |
|                                                                                                       |
| In ``fix`` mode, Gazelle writes generated and merged files to disk. In                                |
| ``print`` mode, it prints them to stdout. In ``diff`` mode, it prints a                               |
| unified diff.                                                                                         |
+--------------------------------------------------------------+----------------------------------------+
| :flag:`-proto default|package|legacy|disable|disable_global` | :value:`default`                       |
+--------------------------------------------------------------+----------------------------------------+
| Determines how Gazelle should generate rules for .proto files. See details                            |
| in `Directives`_ below.                                                                               |
+--------------------------------------------------------------+----------------------------------------+
| :flag:`-proto_group group`                                   | :value:`""`                            |
+--------------------------------------------------------------+----------------------------------------+
| Determines the proto option Gazelle uses to group .proto files into rules                             |
| when in ``package`` mode. See details in `Directives`_ below.                                         |
+--------------------------------------------------------------+----------------------------------------+
| :flag:`-proto_import_prefix repo`                            |                                        |
+--------------------------------------------------------------+----------------------------------------+
| Sets the `import_prefix`_ attribute of generated ``proto_library`` rules. This is a prefix            |
| to add to import paths of .proto files.                                                               |
+--------------------------------------------------------------+----------------------------------------+
| :flag:`-repo_root dir`                                       |                                        |
+--------------------------------------------------------------+----------------------------------------+
| The root directory of the repository. Gazelle normally infers this to be the                          |
| directory containing the WORKSPACE file.                                                              |
|                                                                                                       |
| Gazelle will not process packages outside this directory.                                             |
+--------------------------------------------------------------+----------------------------------------+
.. _Predefined plugins: https://github.com/bazelbuild/rules_go/blob/master/proto/core.rst#predefined-plugins

``update-repos``
~~~~~~~~~~~~~~~~

The ``update-repos`` command updates repository rules in the WORKSPACE file.
It can be used to add new repository rules or update existing rules to the
latest version. It can also import repository rules from a ``go.mod`` file or
a ``Gopkg.lock`` file.

.. code:: bash

  # Add or update a repository by import path
  $ gazelle update-repos example.com/new/repo

  # Import repositories from go.mod
  $ gazelle update-repos -from_file=go.mod

:Note: ``update-repos`` is not directly supported by the ``gazelle`` rule.
  You can run it through the ``gazelle`` rule by passing extra arguments after
  ``--``. For example:

  .. code::

    $ bazel run //:gazelle -- update-repos example.com/new/repo

The following flags are accepted:

+----------------------------------------------------------------------------------------------------------+----------------------------------------------+
| **Name**                                                                                                 | **Default value**                            |
+==========================================================================================================+==============================================+
| :flag:`-from_file lock-file`                                                                             |                                              |
+----------------------------------------------------------------------------------------------------------+----------------------------------------------+
| Import repositories from a file as `go_repository`_ rules. These rules will be added to the bottom of the WORKSPACE file or merged with existing rules. |
|                                                                                                                                                         |
| The lock file format is inferred from the file name. ``go.mod`` and, ``Gopkg.lock`` (the dep lock format) are both supported.                           |
+----------------------------------------------------------------------------------------------------------+----------------------------------------------+
| :flag:`-repo_root dir`                                                                                   |                                              |
+----------------------------------------------------------------------------------------------------------+----------------------------------------------+
| The root directory of the repository. Gazelle normally infers this to be the directory containing the WORKSPACE file.                                   |
|                                                                                                                                                         |
| Gazelle will not process packages outside this directory.                                                                                               |
+----------------------------------------------------------------------------------------------------------+----------------------------------------------+
| :flag:`-build_file_names file1,file2,...`                                                                |                                              |
+----------------------------------------------------------------------------------------------------------+----------------------------------------------+
| Sets the ``build_file_name`` attribute for the generated `go_repository`_ rule(s).                                                                      |
+----------------------------------------------------------------------------------------------------------+----------------------------------------------+
| :flag:`-build_external external|vendored`                                                                |                                              |
+----------------------------------------------------------------------------------------------------------+----------------------------------------------+
| Sets the ``build_external`` attribute for the generated `go_repository`_ rule(s).                                                                       |
+----------------------------------------------------------------------------------------------------------+----------------------------------------------+
| :flag:`-build_file_generation auto|on|off`                                                               |                                              |
+----------------------------------------------------------------------------------------------------------+----------------------------------------------+
| Sets the ``build_file_generation`` attribute for the generated `go_repository`_ rule(s).                                                                |
+----------------------------------------------------------------------------------------------------------+----------------------------------------------+
| :flag:`-build_tags tag1,tag2,...`                                                                        |                                              |
+----------------------------------------------------------------------------------------------------------+----------------------------------------------+
| Sets the ``build_tags`` attribute for the generated `go_repository`_ rule(s).                                                                           |
+----------------------------------------------------------------------------------------------------------+----------------------------------------------+
| :flag:`-build_file_proto_mode default|package|legacy|disable|disable_global`                             |                                              |
+----------------------------------------------------------------------------------------------------------+----------------------------------------------+
| Sets the ``build_file_proto_mode`` attribute for the generated `go_repository`_ rule(s).                                                                |
+----------------------------------------------------------------------------------------------------------+----------------------------------------------+
| :flag:`-build_extra_args arg1,arg2,...`                                                                  |                                              |
+----------------------------------------------------------------------------------------------------------+----------------------------------------------+
| Sets the ``build_exra_args attribute`` for the generated `go_repository`_ rule(s).                                                                      |
+----------------------------------------------------------------------------------------------------------+----------------------------------------------+

Directives
~~~~~~~~~~

Gazelle can be configured with *directives*, which are written as top-level
comments in build files. Most options that can be set on the command line
can also be set using directives. Some options can only be set with
directives.

Directive comments have the form ``# gazelle:key value``. For example:

.. code:: bzl

  load("@io_bazel_rules_go//go:def.bzl", "go_library")

  # gazelle:prefix github.com/example/project
  # gazelle:build_file_name BUILD,BUILD.bazel

  go_library(
      name = "go_default_library",
      srcs = ["example.go"],
      importpath = "github.com/example/project",
      visibility = ["//visibility:public"],
  )

Directives apply in the directory where they are set *and* in subdirectories.
This means, for example, if you set ``# gazelle:prefix`` in the build file
in your project's root directory, it affects your whole project. If you
set it in a subdirectory, it only affects rules in that subtree.

The following directives are recognized:

+---------------------------------------------------+----------------------------------------+
| **Directive**                                     | **Default value**                      |
+===================================================+========================================+
| :direc:`# gazelle:build_file_name names`          | :value:`BUILD.bazel,BUILD`             |
+---------------------------------------------------+----------------------------------------+
| Comma-separated list of file names. Gazelle recognizes these files as Bazel                |
| build files. New files will use the first name in this list. Use this if                   |
| your project contains non-Bazel files named ``BUILD`` (or ``build`` on                     |
| case-insensitive file systems).                                                            |
+---------------------------------------------------+----------------------------------------+
| :direc:`# gazelle:build_tags foo,bar`             | none                                   |
+---------------------------------------------------+----------------------------------------+
| List of Go build tags Gazelle will consider to be true. Gazelle applies                    |
| constraints when generating Go rules. It assumes certain tags are true on                  |
| certain platforms (for example, ``amd64,linux``). It assumes all Go release                |
| tags are true (for example, ``go1.8``). It considers other tags to be false                |
| (for example, ``ignore``). This flag overrides that behavior.                              |
|                                                                                            |
| Bazel may still filter sources with these tags. Use                                        |
| ``bazel build --features gotags=foo,bar`` to set tags at build time.                       |
+---------------------------------------------------+----------------------------------------+
| :direc:`# gazelle:exclude path`                   | n/a                                    |
+---------------------------------------------------+----------------------------------------+
| Prevents Gazelle from processing a file or directory. If the path refers to                |
| a source file, Gazelle won't include it in any rules. If the path refers to                |
| a directory, Gazelle won't recurse into it. The path may refer to something                |
| withinin a subdirectory, for example, a testdata directory somewhere in a                  |
| vendor tree. This directive may be repeated to exclude multiple paths, one                 |
| per line.                                                                                  |
+---------------------------------------------------+----------------------------------------+
| :direc:`# gazelle:follow path`                    | n/a                                    |
+---------------------------------------------------+----------------------------------------+
| Instructs Gazelle to follow a symbolic link to a directory within the                      |
| repository. Normally, Gazelle does not follow symbolic links unless they                   |
| point outside of the repository root.                                                      |
|                                                                                            |
| Care must be taken to avoid visiting a directory more than once.                           |
| The ``# gazelle:exclude`` directive may be used to prevent Gazelle from                    |
| recursing into a directory.                                                                |
+---------------------------------------------------+----------------------------------------+
| :direc:`# gazelle:go_grpc_compilers`              | ``@io_bazel_rules_go//proto:go_grpc``  |
+---------------------------------------------------+----------------------------------------+
| The protocol buffers compiler(s) to use for building go bindings for gRPC.                 |
| Multiple compilers, separated by commas, may be specified.                                 |
| Omit the directive value to reset ``go_grpc_compilers`` back to the default.               |
|                                                                                            |
| See `Predefined plugins`_ for available options; commonly used options include             |
| ``@io_bazel_rules_go//proto:gofast_grpc`` and                                              |
| ``@io_bazel_rules_go//proto:gogofaster_grpc``.                                             |
+---------------------------------------------------+----------------------------------------+
| :direc:`# gazelle:go_proto_compilers`             | ``@io_bazel_rules_go//proto:go_proto`` |
+---------------------------------------------------+----------------------------------------+
| The protocol buffers compiler(s) to use for building go bindings.                          |
| Multiple compilers, separated by commas, may be specified.                                 |
| Omit the directive value to reset ``go_proto_compilers`` back to the default.              |
|                                                                                            |
| See `Predefined plugins`_ for available options; commonly used options include             |
| ``@io_bazel_rules_go//proto:gofast_proto`` and                                             |
| ``@io_bazel_rules_go//proto:gogofaster_proto``.                                            |
+---------------------------------------------------+----------------------------------------+
| :direc:`# gazelle:ignore`                         | n/a                                    |
+---------------------------------------------------+----------------------------------------+
| Prevents Gazelle from modifying the build file. Gazelle will still read                    |
| rules in the build file and may modify build files in subdirectories.                      |
+---------------------------------------------------+----------------------------------------+
| :direc:`# gazelle:importmap_prefix path`          | See below                              |
+---------------------------------------------------+----------------------------------------+
| A prefix for ``importmap`` attributes in library rules. Gazelle will set                   |
| an ``importmap`` on a ``go_library`` or ``go_proto_library`` by                            |
| concatenating this with the relative path from the directory where the                     |
| prefix is set to the library. For example, if ``importmap_prefix`` is set                  |
| to ``"x/example.com/repo"`` in the build file ``//foo/bar:BUILD.bazel``,                   |
| then a library in ``foo/bar/baz`` will have the ``importmap`` of                           |
| ``"x/example.com/repo/baz"``.                                                              |
|                                                                                            |
| ``importmap`` is not set when it matches ``importpath``.                                   |
|                                                                                            |
| As a special case, when Gazelle enters a directory named ``vendor``, it                    |
| sets ``importmap_prefix`` to a string based on the repository name and the                 |
| location of the vendor directory. If you wish to override this, you'll need                |
| to set ``importmap_prefix`` explicitly in the vendor directory.                            |
+------------------------------------------------------------+-------------------------------+
| :direc:`# gazelle:map_kind from_kind to_kind to_kind_load` | n/a                           |
+------------------------------------------------------------+-------------------------------+
| Customizes the kind of rules generated by Gazelle.                                         |
|                                                                                            |
| As a separate step after generating rules, any new rules of kind ``from_kind`` have their  |
| kind replaced with ``to_kind``. This means that ``to_kind`` must accept the same           |
| parameters and behave similarly.                                                           |
|                                                                                            |
| Most commonly, this would be used to replace the rules provided by ``rules_go`` with       |
| custom macros. For example,                                                                |
| ``gazelle:map_kind go_binary go_deployable //tools/go:def.bzl`` would configure Gazelle to |
| produce rules of kind ``go_deployable`` as loaded from ``//tools/go:def.bzl`` instead of   |
| ``go_binary``, for this directory or within.                                               |
+---------------------------------------------------+----------------------------------------+
| :direc:`# gazelle:prefix path`                    | n/a                                    |
+---------------------------------------------------+----------------------------------------+
| A prefix for ``importpath`` attributes on library rules. Gazelle will set                  |
| an ``importpath`` on a ``go_library`` or ``go_proto_library`` by                           |
| concatenating this with the relative path from the directory where the                     |
| prefix is set to the library. Most commonly, ``prefix`` is set to the                      |
| name of a repository in the root directory of a repository. For example,                   |
| in this repository, ``prefix`` is set in ``//:BUILD.bazel`` to                             |
| ``github.com/bazelbuild/bazel-gazelle``. The ``go_library`` in                             |
| ``//cmd/gazelle`` is assigned the ``importpath``                                           |
| ``"github.com/bazelbuild/bazel-gazelle/cmd/gazelle"``.                                     |
|                                                                                            |
| As a special case, when Gazelle enters a directory named ``vendor``, it sets               |
| ``prefix`` to the empty string. This automatically gives vendored libraries                |
| an intuitive ``importpath``.                                                               |
+---------------------------------------------------+----------------------------------------+
| :direc:`# gazelle:proto mode`                     | :value:`default`                       |
+---------------------------------------------------+----------------------------------------+
| Tells Gazelle how to generate rules for .proto files. Valid values are:                    |
|                                                                                            |
| * ``default``: ``proto_library``, ``go_proto_library``, and ``go_library``                 |
|   rules are generated using ``@io_bazel_rules_go//proto:def.bzl``. Only one                |
|   of each rule may be generated per directory. This is the default mode.                   |
| * ``package``: multiple ``proto_library`` and ``go_proto_library`` rules                   |
|   may be generated in the same directory. .proto files are grouped into                    |
|   rules based on their package name or another option (see ``proto_group``).               |
| * ``legacy``: ``filegroup`` rules are generated for use by                                 |
|   ``@io_bazel_rules_go//proto:go_proto_library.bzl``. ``go_proto_library``                 |
|   rules must be written by hand. Gazelle will run in this mode automatically               |
|   if ``go_proto_library.bzl`` is loaded to avoid disrupting existing                       |
|   projects, but this can be overridden with a directive.                                   |
| * ``disable``: .proto files are ignored. Gazelle will run in this mode                     |
|   automatically if ``go_proto_library`` is loaded from any other source,                   |
|   but this can be overridden with a directive.                                             |
| * ``disable_global``: like ``disable`` mode, but also prevents Gazelle from                |
|   using any special cases in dependency resolution for Well Known Types and                |
|   Google APIs. Useful for avoiding build-time dependencies on protoc.                      |
|                                                                                            |
| This directive applies to the current directory and subdirectories. As a                   |
| special case, when Gazelle enters a directory named ``vendor``, if the proto               |
| mode isn't set explicitly in a parent directory or on the command line,                    |
| Gazelle will run in ``disable`` mode. Additionally, if the file                            |
| ``@io_bazel_rules_go//proto:go_proto_library.bzl`` is loaded, Gazelle                      |
| will run in ``legacy`` mode.                                                               |
+---------------------------------------------------+----------------------------------------+
| :direc:`# gazelle:proto_group option`             | :value:`""`                            |
+---------------------------------------------------+----------------------------------------+
| *This directive is only effective in* ``package`` *mode (see above).*                      |
|                                                                                            |
| Specifies an option that Gazelle can use to group .proto files into rules.                 |
| For example, when set to ``go_package``, .proto files with the same                        |
| ``option go_package`` will be grouped together.                                            |
|                                                                                            |
| When this directive is set to the empty string, Gazelle will group packages                |
| by their proto package statement.                                                          |
|                                                                                            |
| Rule names are generated based on the last run of identifier characters                    |
| in the package name. For example, if the package is ``"foo/bar/baz"``, the                 |
| ``proto_library`` rule will be named ``baz_proto``.                                        |
+---------------------------------------------------+----------------------------------------+
| :direc:`# gazelle:proto_strip_import_prefix path` | n/a                                    |
+---------------------------------------------------+----------------------------------------+
| Sets the `strip_import_prefix`_ attribute of generated ``proto_library`` rules.            |
| This is a prefix to strip from the import paths of .proto files.                           |
+---------------------------------------------------+----------------------------------------+
| :direc:`# gazelle:proto_import_prefix path`       | n/a                                    |
+---------------------------------------------------+----------------------------------------+
| Sets the `import_prefix`_ attribute of generated ``proto_library`` rules.                  |
| This is a prefix to add to import paths of .proto files.                                   |
+---------------------------------------------------+----------------------------------------+
| :direc:`# gazelle:resolve ...`                    | n/a                                    |
+---------------------------------------------------+----------------------------------------+
| Specifies an explicit mapping from an import string to a label for                         |
| `Dependency resolution`_. The format for a resolve directive is:                           |
|                                                                                            |
| ``# gazelle:resolve source-lang import-lang import-string label``                          |
|                                                                                            |
| * ``source-lang`` is the language of the source code being imported.                       |
| * ``import-lang`` is the language importing the library. This is usually                   |
|   the same as ``source-lang`` but may differ with generated code. For                      |
|   example, when resolving dependencies for a ``go_proto_library``,                         |
|   ``source-lang`` would be ``"proto"`` and ``import-lang`` would be ``"go"``.              |
|   ``import-lang`` may be omitted if it is the same as ``source-lang``.                     |
| * ``import-string`` is the string used in source code to import a library.                 |
| * ``label`` is the Bazel label that Gazelle should write in ``deps``.                      |
|                                                                                            |
| For example:                                                                               |
|                                                                                            |
| .. code:: bzl                                                                              |
|                                                                                            |
|   # gazelle:resolve go example.com/foo //foo:go_default_library                            |
|   # gazelle:resolve proto go foo/foo.proto //foo:foo_go_proto                              |
|                                                                                            |
+---------------------------------------------------+----------------------------------------+

Keep comments
~~~~~~~~~~~~~

In addition to directives, Gazelle supports ``# keep`` comments that protect
parts of build files from being modified. ``# keep`` may be written before
a rule, before an attribute, or after a string within a list.

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

Dependency resolution
---------------------

One of Gazelle's most important jobs is resolving library import strings
(like ``import "golang.org/x/sys/unix"``) to Bazel labels (like
``@org_golang_x_sys//unix:go_default_library``). Gazelle follows the rules
below to resolve dependencies:

1. If the import to be resolved is part of a standard library, no explicit
   dependency is written. For example, in Go, you don't need to declare
   that you depend on ``"fmt"``.
2. If a ``# gazelle:resolve`` directive matches the import to be resolved,
   the label at the end of the directive will be used.
3. If proto rule generation is enabled, special rules will be used when
   importing certain libraries. These rules may be disabled by adding
   ``# gazelle:proto disable_global`` to a build file (this will affect
   subdirectories, too) or by passing ``-proto disable_global`` on the
   command line.

   a) Imports of Well Known Types are mapped to rules in
      ``@io_bazel_rules_go//proto/wkt``.
   b) Imports of Google APIs are mapped to ``@go_googleapis``.
   c) Imports of ``github.com/golang/protobuf/ptypes``, ``descriptor``, and
      ``jsonpb`` are mapped to special rules in ``@com_github_golang_protobuf``.
      See `Avoiding conflicts with proto rules`_.

4. If the import to be resolved is in the library index, the import will be resolved
   to that library. If ``-index=true``, Gazelle builds an index of library rules in
   the current repository before starting dependency resolution, and this is how
   most dependencies are resolved.

   a) For Go, the match is based on the ``importpath`` attribute.
   b) For proto, the match is based on the ``srcs`` attribute.

5. If ``-index=false`` and a package is imported that has the current ``go_prefix``
   as a prefix, Gazelle generates a label following a convention. For example, if
   the build file in ``//src`` set the prefix with
   ``# gazelle:prefix example.com/repo/foo``, and you import the library
   ``"example.com/repo/foo/bar``, the dependency will be
   ``"//src/foo/bar:go_default_library"``.
6. Otherwise, Gazelle will use the current ``external`` mode to resolve
   the dependency.

   a) In ``external`` mode (the default), Gazelle will transform the import
      string into an external repository label. For example,
      ``"golang.org/x/sys/unix"`` would be resolved to
      ``"@org_golang_x_sys//unix:go_default_library"``. Gazelle does not confirm
      whether the external repository is actually declared in WORKSPACE,
      but if there *is* a ``go_repository`` in WORKSPACE with a matching
      ``importpath``, Gazelle will use its name. Gazelle does not index
      rules in external repositories, so it's possible the resolved dependency
      does not exist.
   b) In ``vendored`` mode, Gazelle will transform the import string into
      a label in the vendor directory. For example, ``"golang.org/x/sys/unix"``
      would be resolved to
      ``"//vendor/golang.org/x/sys/unix:go_default_library"``. This mode is
      usually not necessary, since vendored libraries will be indexed and
      resolved using rule 4.

Fix command transformations
---------------------------

Gazelle will generate and update build files when invoked with either
``gazelle update`` or ``gazelle fix`` (``update`` is the default). Both commands
perform several transformations to fix deprecated usage of the Go rules.
``update`` performs a safe set of tranformations, while ``fix`` performs some
additional transformations that may delete or rename rules.

The following transformations are performed:

**Migrate library to embed (fix and update):** Gazelle replaces ``library``
attributes with ``embed`` attributes.

**Migrate gRPC compilers (fix and update):** Gazelle converts
``go_grpc_library`` rules to ``go_proto_library`` rules with
``compilers = ["@io_bazel_rules_go//proto:go_grpc"]``.

**Flatten srcs (fix and update):** Gazelle converts ``srcs`` attributes that
use OS and architecture-specific ``select`` expressions to flat lists.
rules_go filters these sources anyway.

**Squash cgo libraries (fix only)**: Gazelle will remove `cgo_library` rules
named ``cgo_default_library`` and merge their attributes with a ``go_library``
rule in the same package named ``go_default_library``. If no such ``go_library``
rule exists, a new one will be created. Other ``cgo_library`` rules will not be
removed.

**Squash external tests (fix only)**: Gazelle will squash ``go_test`` rules
named ``go_default_xtest`` into ``go_default_test``. Earlier versions of
rules_go required internal and external tests to be built separately, but
this is no longer needed.

**Remove legacy protos (fix only)**: Gazelle will remove usage of
``go_proto_library`` rules loaded from
``@io_bazel_rules_go//proto:go_proto_library.bzl`` and ``filegroup`` rules named
``go_default_library_protos``. Newly generated proto rules will take their
place. Since ``filegroup`` isn't needed anymore and ``go_proto_library`` has
different attributes and was always written by hand, Gazelle will not attempt to
merge anything from these rules with the newly generated rules.

This transformation is only applied in the default proto mode. Since Gazelle
will run in legacy proto mode if ``go_proto_library.bzl`` is loaded, this
transformation is not usually applied. You can set the proto mode explicitly
using the directive ``# gazelle:proto default``.

**Update loads of gazelle rule (fix and update)**: Gazelle will remove loads
of ``gazelle`` from ``@io_bazel_rules_go//go:def.bzl``. It will automatically
add a load from ``@bazel_gazelle//:def.bzl`` if ``gazelle`` is not loaded
from another location.
