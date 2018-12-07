Extending Gazelle
=================

.. Begin directives
.. _Language: https://godoc.org/github.com/bazelbuild/bazel-gazelle/language#Language
.. _`//internal/gazellebinarytest:go_default_library`: https://github.com/bazelbuild/bazel-gazelle/tree/master/internal/gazellebinarytest
.. _`//language/go:go_default_library`: https://github.com/bazelbuild/bazel-gazelle/tree/master/language/go
.. _`//language/proto:go_default_library`: https://github.com/bazelbuild/bazel-gazelle/tree/master/language/proto
.. _gazelle: https://github.com/bazelbuild/bazel-gazelle#bazel-rule
.. _go_binary: https://github.com/bazelbuild/rules_go/blob/master/go/core.rst#go-binary
.. _go_library: https://github.com/bazelbuild/rules_go/blob/master/go/core.rst#go-library
.. _proto godoc: https://godoc.org/github.com/bazelbuild/bazel-gazelle/language/proto
.. _proto.GetProtoConfig: https://godoc.org/github.com/bazelbuild/bazel-gazelle/language/proto#GetProtoConfig
.. _proto.Package: https://godoc.org/github.com/bazelbuild/bazel-gazelle/language/proto#Package

.. role:: cmd(code)
.. role:: flag(code)
.. role:: direc(code)
.. role:: param(kbd)
.. role:: type(emphasis)
.. role:: value(code)
.. |mandatory| replace:: **mandatory value**
.. End directives

Gazelle started out as a build file generator for Go projects, but it can be
extended to support other languages and custom sets of rules.

To extend Gazelle, you must do three things:

* Write a `go_library`_ with a function named ``NewLanguage`` that provides an
  implementation of the Language_ interface. This interface provides hooks for
  generating rules, parsing configuration directives, and resolving imports
  to Bazel labels.
* Write a `gazelle_binary`_ rule. Include your library in the ``languages``
  list.
* Write a `gazelle`_ rule that points to your ``gazelle_binary``. When you run
  ``bazel run //:gazelle``, your binary will be built and executed instead of
  the default binary.

Example
-------

**TODO:** Add a self-contained, concise, realistic example.

Gazelle itself is built using the model described above, so it may serve as
an example.

`//language/proto:go_default_library`_ and `//language/go:go_default_library`_
both implement the `Language`_
interface. There is also `//internal/gazellebinarytest:go_default_library`_,
a stub implementation used for testing.

``//cmd/gazelle`` is a ``gazelle_binary`` rule that includes both of these
libraries through the ``DEFAULT_LANGUAGES`` list (you may want to use
``DEFAULT_LANGUAGES`` in your own rule). The ``msan``, ``pure``, ``race``,
and ``static`` attributes are optional.

.. code:: bzl

    load("@bazel_gazelle//:def.bzl", "DEFAULT_LANGUAGES", "gazelle_binary")

    gazelle_binary(
        name = "gazelle",
        languages = DEFAULT_LANGUAGES,
        msan = "off",
        pure = "off",
        race = "off",
        static = "off",
        visibility = ["//visibility:public"],
    )

This binary can be invoked using a ``gazelle`` rule like this:

.. code:: bzl

    load("@bazel_gazelle//:def.bzl", "gazelle")

    # gazelle:prefix example.com/project
    gazelle(
        name = "gazelle",
        gazelle = "//:my_gazelle_binary",
    )

You can run this with ``bazel run //:gazelle``.

gazelle_binary
--------------

The ``gazelle_binary`` rule builds a Go binary that incorporates a list of
language extensions. This requires generating a small amount of code that
must be compiled into Gazelle's main package, so the normal `go_binary`_
rule is not used.

When the binary runs, each language extension is run sequentially. This affects
the order that rules appear in generated build files. Metadata may be produced
by an earlier extension and consumed by a later extension. For example, the
proto extension stores metadata in hidden attributes of generated
``proto_library`` rules. The Go extension uses this metadata to generate
``go_proto_library`` rules.

The following attributes are supported on the ``gazelle_binary`` rule.

+----------------------+---------------------+--------------------------------------+
| **Name**             | **Type**            | **Default value**                    |
+======================+=====================+======================================+
| :param:`languages`   | :type:`label_list`  | |mandatory|                          |
+----------------------+---------------------+--------------------------------------+
| A list of language extensions the Gazelle binary will use.                        |
|                                                                                   |
| Each extension must be a `go_library`_ or something compatible. Each extension    |
| must export a function named ``NewLanguage`` with no parameters that returns      |
| a value assignable to `Language`_.                                                |
+----------------------+---------------------+--------------------------------------+
| :param:`pure`        | :type:`string`      | :value:`auto`                        |
+----------------------+---------------------+--------------------------------------+
| Same meaning as `go_binary`_. It may be necessary to set this to avoid            |
| command flags that affect both host and target configurations.                    |
+----------------------+---------------------+--------------------------------------+
| :param:`static`        | :type:`string`      | :value:`auto`                      |
+----------------------+---------------------+--------------------------------------+
| Same meaning as `go_binary`_. It may be necessary to set this to avoid            |
| command flags that affect both host and target configurations.                    |
+----------------------+---------------------+--------------------------------------+
| :param:`race`        | :type:`string`      | :value:`auto`                        |
+----------------------+---------------------+--------------------------------------+
| Same meaning as `go_binary`_. It may be necessary to set this to avoid            |
| command flags that affect both host and target configurations.                    |
+----------------------+---------------------+--------------------------------------+
| :param:`msan`        | :type:`string`      | :value:`auto`                        |
+----------------------+---------------------+--------------------------------------+
| Same meaning as `go_binary`_. It may be necessary to set this to avoid            |
| command flags that affect both host and target configurations.                    |
+----------------------+---------------------+--------------------------------------+
| :param:`goos`        | :type:`string`      | :value:`auto`                        |
+----------------------+---------------------+--------------------------------------+
| Same meaning as `go_binary`_. It may be necessary to set this to avoid            |
| command flags that affect both host and target configurations.                    |
+----------------------+---------------------+--------------------------------------+
| :param:`goarch`        | :type:`string`      | :value:`auto`                      |
+----------------------+---------------------+--------------------------------------+
| Same meaning as `go_binary`_. It may be necessary to set this to avoid            |
| command flags that affect both host and target configurations.                    |
+----------------------+---------------------+--------------------------------------+

Interacting with protos
-----------------------

The proto extension (`//language/proto:go_default_library`_) gathers metadata
from .proto files and generates ``proto_library`` rules based on that metadata.
Extensions that generate language-specific proto rules (e.g.,
``go_proto_library``) may use this metadata.

For API reference, see the `proto godoc`_.

To get proto configuration information, call `proto.GetProtoConfig`_. This is
mainly useful for discovering the current proto mode.

To get information about ``proto_library`` rules, examine the ``OtherGen``
list of rules passed to ``language.GenerateRules``. This is a list of rules
generated by other language extensions, and it will include ``proto_library``
rules in each directory, if there were any. For each of these rules, you can
call ``r.PrivateAttr(proto.PackageKey)`` to get a `proto.Package`_ record. This
includes the proto package name, as well as source names, imports, and options.
