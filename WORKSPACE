workspace(name = "bazel_gazelle")

git_repository(
    name = "io_bazel_rules_go",
    remote = "https://github.com/bazelbuild/rules_go",
    commit = "d36234e86678457e769ab62e597f90eef025bf8d",
)

load("@io_bazel_rules_go//go:def.bzl", "go_rules_dependencies", "go_register_toolchains")

go_rules_dependencies()

go_register_toolchains()

load("//:deps.bzl", "gazelle_dependencies")

gazelle_dependencies()

load("@io_bazel_rules_go//tests:bazel_tests.bzl", "test_environment")

test_environment()
