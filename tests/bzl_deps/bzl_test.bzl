"""Macro for Ensuring Starlark Dependencies are Specified Properly"""

load("@bazel_skylib//:bzl_library.bzl", "bzl_library")
load("@io_bazel_stardoc//stardoc:stardoc.bzl", "stardoc")

def bzl_test(name, src, deps):
    """Provides build-time assurances that `bzl_library` declarations exist \
    and are referenced properly.

    This macro relies upon Stardoc's ability to traverse `bzl_library`
    dependencies. If a Starlark dependency is loaded, but not specified as a
    dependency, the Stardoc utility will fail with a reasonably helpful error
    message. Interestingly, the Stardoc utility does not apply the same rigor
    to files that are directly specifed to it.

    Another point worth metioning is that the `src` file cannot be generated
    for this macro to work. If one tries to use a generated file, the `input`
    for the `stardoc` rule will resolve to the label for the generated file
    which will cause the Stardoc utility to not find the file. Specifying the
    input in different ways (i.e. filename vs target name) did not seem to
    affect this behavior.

    Args:
        name: The name of the build target.
        src: A non-generated Starlark file that loads the `bzl_library` that is
             being checked.
        deps: A `list` of deps for the Starlark file.

    Returns:
    """
    macro_lib_name = name + "_macro_lib"
    bzl_library(
        name = macro_lib_name,
        srcs = [src],
        deps = deps,
    )

    stardoc(
        name = name,
        out = macro_lib_name + ".md_",
        input = src,
        deps = [macro_lib_name],
    )
