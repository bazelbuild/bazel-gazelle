"""
Test for generating rules from gazelle. 
"""

load("@io_bazel_rules_go//go:def.bzl", "go_test")

def gazelle_generation_test(name, gazelle_binary, test_data, build_in_suffix = ".in", build_out_suffix = ".out"):
    """
    gazelle_generation_test is a macro for testing gazelle against workspaces.

    Args:
        name: the name of the test.
        gazelle_binary: the name of the gazelle binary target. For example, //path/to:my_gazelle.
        test_data: a target of the test data files you will pass to the test.
            This can be a https://bazel.build/reference/be/general#filegroup.
        build_in_suffix: the suffix for the input BUILD.bazel files. Defaults to .in.
            By default, will use files named BUILD.in as the BUILD files before running gazelle.
        build_out_suffix: the suffix for the expected BUILD.bazel files after running gazelle. Defaults to .out.
            By default, will use files named check the results of the gazelle run against files named BUILD.out.
    """
    go_test(
        name = name,
        srcs = ["//internal/generationtest:generation_test.go"],
        deps = [
            "//testtools",
            "@io_bazel_rules_go//go/tools/bazel:go_default_library",
            "@in_gopkg_yaml_v2//:yaml_v2",
        ],
        args = [
            "-gazelle_binary_path=$(rootpath %s)" % gazelle_binary,
            "-build_in_suffix=%s" % build_in_suffix,
            "-build_out_suffix=%s" % build_out_suffix,
        ],
        data = [
            test_data,
            gazelle_binary,
        ],
    )
