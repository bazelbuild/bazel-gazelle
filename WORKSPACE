workspace(name = "bazel_gazelle")

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

http_archive(
    name = "io_bazel_rules_go",
    sha256 = "8663604808d2738dc615a2c3eb70eba54a9a982089dd09f6ffe5d0e75771bc4f",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.23.6/rules_go-v0.23.6.tar.gz",
        "https://github.com/bazelbuild/rules_go/releases/download/v0.23.6/rules_go-v0.23.6.tar.gz",
    ],
)

load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")

go_rules_dependencies()

go_register_toolchains()

go_register_toolchains(nogo = "@bazel_gazelle//:nogo")

load("//:deps.bzl", "gazelle_dependencies")

gazelle_dependencies()
