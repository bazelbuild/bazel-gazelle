workspace(name = "bazel_gazelle")

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

http_archive(
    name = "io_bazel_rules_go",
    sha256 = "d1ffd055969c8f8d431e2d439813e42326961d0942bdf734d2c95dc30c369566",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.24.5/rules_go-v0.24.5.tar.gz",
        "https://github.com/bazelbuild/rules_go/releases/download/v0.24.5/rules_go-v0.24.5.tar.gz",
    ],
)

load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")

go_rules_dependencies()

go_register_toolchains()

go_register_toolchains(nogo = "@bazel_gazelle//:nogo")

load("//:deps.bzl", "gazelle_dependencies")

gazelle_dependencies()

# gazelle:repository go_repository name=org_golang_x_sync importpath=golang.org/x/sync
# gazelle:repository go_repository name=org_golang_x_sys importpath=golang.org/x/sys
# gazelle:repository go_repository name=org_golang_x_xerrors importpath=golang.org/x/xerrors
# gazelle:repository go_repository name=com_github_bazelbuild_buildtools importpath=github.com/bazelbuild/buildtools build_naming_convention=go_default_library
# gazelle:repository go_repository name=com_github_fsnotify_fsnotify importpath=github.com/fsnotify/fsnotify
# gazelle:repository go_repository name=com_github_pelletier_go_toml importpath=github.com/pelletier/go-toml
# gazelle:repository go_repository name=com_github_pmezard_go_difflib importpath=github.com/pmezard/go-difflib
# gazelle:repository go_repository name=com_github_bmatcuk_doublestar importpath=github.com/bmatcuk/doublestar
# gazelle:repository go_repository name=com_github_google_go_cmp importpath=github.com/google/go-cmp
