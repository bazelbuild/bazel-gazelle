workspace(name = "bazel_gazelle")

git_repository(
    name = "io_bazel_rules_go",
    commit = "e7249a61c3a244513601d998a13df1fa835433eb", # master on 2018-01-04
    remote = "https://github.com/bazelbuild/rules_go",
)

load("@io_bazel_rules_go//go:def.bzl", "go_rules_dependencies", "go_register_toolchains")

go_rules_dependencies()

go_register_toolchains()

load("//:deps.bzl", "gazelle_dependencies")

gazelle_dependencies()
