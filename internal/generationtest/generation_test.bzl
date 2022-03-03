"""
Test for generating rules from gazelle. 
"""

load("@io_bazel_rules_go//go:def.bzl", "go_test")

def gazelle_generation_test(name, gazelle_binary_name, test_data, test_data_dir, gazelle_binary_dir = ""):
    """
    gazelle_generation_test is a macro for testing gazelle against workspaces.

    Args:
        name: the name of the test.
        gazelle_binary_name: the name of the gazelle binary target. For example, if you have a
        gazelle_binary target where you pass the name "my_gazelle", this would be "my_gazelle".
        test_data: a glob of the test data files you will pass to the test.
        test_data_dir: the workspace relative path where the testdata file is.
        gazelle_binary_dir: the workspace relative path where the gazelle_binary target is. For example,
        if "my_gazelle" is defined in a BUILD.bazel file in <wkspace>/my/path, this arg should be
        "my/path".
    """
    native.genrule(
        name = "%s_test_manifest" % name,
        srcs = ["//internal/generationtest:generation_test_manifest.yaml.tpl"],
        outs = ["generation_test_manifest.yaml"],
        cmd = "sed 's|TEMPLATED_gazellebinaryname|%s|g;" % gazelle_binary_name +
              "s|TEMPLATED_gazellebinarydir|%s|g;" % gazelle_binary_dir +
              "s|TEMPLATED_testdatadir|%s|g' " % test_data_dir +
              "$< > $@",
    )
    go_test(
        name = name,
        srcs = ["//internal/generationtest:generation_test.go"],
        deps = [
            "//testtools",
            "@io_bazel_rules_go//go/tools/bazel:go_default_library",
            "@in_gopkg_yaml_v2//:yaml_v2",
        ],
        data = test_data + ["//%s:%s" % (gazelle_binary_dir, gazelle_binary_name)] + [
            ":generation_test_manifest.yaml",
        ],
    )
