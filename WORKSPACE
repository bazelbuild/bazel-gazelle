workspace(name = "bazel_gazelle")

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

http_archive(
    name = "io_bazel_rules_go",
    sha256 = "94f90feaa65c9cdc840cd21f67d967870b5943d684966a47569da8073e42063d",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.22.0/rules_go-v0.22.0.tar.gz",
        "https://github.com/bazelbuild/rules_go/releases/download/v0.22.0/rules_go-v0.22.0.tar.gz",
    ],
)

load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")

go_rules_dependencies()

go_register_toolchains()

go_register_toolchains(nogo = "@bazel_gazelle//:nogo")

load("//:deps.bzl", "gazelle_dependencies")

gazelle_dependencies()
