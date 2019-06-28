workspace(name = "bazel_gazelle")

load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")

git_repository(
    name = "io_bazel_rules_go",
    commit = "8ea79bbd5e6ea09dc611c245d1dc09ef7ab7118a",  # master as of 2019-06-28
    remote = "https://github.com/bazelbuild/rules_go",
    shallow_since = "1561753935 -0400",
)

load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")

go_rules_dependencies()

go_register_toolchains(nogo = "@bazel_gazelle//:nogo")

load("//:deps.bzl", "gazelle_dependencies")

gazelle_dependencies()

load("@io_bazel_rules_go//tests:bazel_tests.bzl", "test_environment")

test_environment()
