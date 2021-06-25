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

load(
    "@io_bazel_rules_go//go:def.bzl",
    "GoArchive",
    "go_context",
)

def _gazelle_binary_impl(ctx):
    go = go_context(ctx)

    # Generate a source file with a list of languages. This will get compiled
    # with the rest of the sources in the main package.
    langs_file = go.declare_file(go, "langs.go")
    langs_content_tpl = """
package main

import (
	"github.com/bazelbuild/bazel-gazelle/language"

	{lang_imports}
)

var languages = []language.Language{{
	{lang_calls},
}}
"""
    lang_imports = [_format_import(d[GoArchive].data.importpath) for d in ctx.attr.languages]
    lang_calls = [_format_call(d[GoArchive].data.importpath) for d in ctx.attr.languages]
    langs_content = langs_content_tpl.format(
        lang_imports = "\n\t".join(lang_imports),
        lang_calls = ",\n\t".join(lang_calls),
    )
    go.actions.write(langs_file, langs_content)

    # Build the gazelle binary.
    library = go.new_library(go, is_main = True)
    attr = struct(
        srcs = [struct(files = [langs_file])],
        deps = ctx.attr.languages,
        embed = [ctx.attr._srcs],
    )
    source = go.library_to_source(go, attr, library, ctx.coverage_instrumented())

    archive, executable, runfiles = go.binary(
        go,
        name = ctx.label.name,
        source = source,
        version_file = ctx.version_file,
        info_file = ctx.info_file,
    )

    return [
        library,
        source,
        archive,
        OutputGroupInfo(compilation_outputs = [archive.data.file]),
        DefaultInfo(
            files = depset([executable]),
            runfiles = runfiles,
            executable = executable,
        ),
    ]

_gazelle_binary_kwargs = {
    "implementation": _gazelle_binary_impl,
    "attrs": {
        "languages": attr.label_list(
            doc = "A list of language extensions the Gazelle binary will use",
            providers = [GoArchive],
            mandatory = True,
            allow_empty = False,
        ),
        "_go_context_data": attr.label(default = "@io_bazel_rules_go//:go_context_data"),
        # _stdlib is needed for rules_go versions before v0.23.0. After that,
        # _go_context_data includes a dependency on stdlib.
        "_stdlib": attr.label(default = "@io_bazel_rules_go//:stdlib"),
        "_srcs": attr.label(
            default = "//cmd/gazelle:gazelle_lib",
        ),
    },
    "executable": True,
    "toolchains": ["@io_bazel_rules_go//go:toolchain"],
}

gazelle_binary = rule(**_gazelle_binary_kwargs)

def gazelle_binary_wrapper(**kwargs):
    for key in ("goos", "goarch", "static", "msan", "race", "pure", "strip", "debug", "linkmode", "gotags"):
        if key in kwargs:
            fail("gazelle_binary attribute '%s' is no longer supported (https://github.com/bazelbuild/bazel-gazelle/issues/803)" % key)
    gazelle_binary(**kwargs)

def _format_import(importpath):
    _, _, base = importpath.rpartition("/")
    return '{} "{}"'.format(base + "_", importpath)

def _format_call(importpath):
    _, _, base = importpath.rpartition("/")
    return "{}.NewLanguage()".format(base + "_")
