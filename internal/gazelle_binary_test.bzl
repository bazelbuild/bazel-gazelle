load("@bazel_skylib//lib:partial.bzl", "partial")
load("@bazel_skylib//lib:unittest.bzl", "asserts", "unittest")
load(":gazelle_binary.bzl", "format_call", "format_import")

def _format_call_test_impl(ctx):
    env = unittest.begin(ctx)
    asserts.equals(
        env,
        "github_com_bazelbuild_bazel_skylib_gazelle_.NewLanguage()",
        format_call("github.com/bazelbuild/bazel-skylib/gazelle"),
    )
    return unittest.end(env)

def _format_import_test_impl(ctx):
    env = unittest.begin(ctx)
    asserts.equals(
        env,
        "github_com_bazelbuild_bazel_skylib_gazelle_ \"github.com/bazelbuild/bazel-skylib/gazelle\"",
        format_import("github.com/bazelbuild/bazel-skylib/gazelle"),
    )
    return unittest.end(env)

_format_call_test = unittest.make(_format_call_test_impl)
_format_import_test = unittest.make(_format_import_test_impl)

def gazelle_binary_test_suite():
    unittest.suite(
        "gazelle_binary_tests",
        partial.make(_format_call_test, size = "small"),
        partial.make(_format_import_test, size = "small"),
    )
