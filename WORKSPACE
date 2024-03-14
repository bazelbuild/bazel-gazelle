workspace(name = "bazel_gazelle")

load(
    "@bazel_tools//tools/build_defs/repo:git.bzl",
    "git_repository",
)

git_repository(
    name = "bazel_skylib",
    commit = "df3c9e2735f02a7fe8cd80db4db00fec8e13d25f",  # `master` as of 2021-08-19
    remote = "https://github.com/bazelbuild/bazel-skylib",
)

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

http_archive(
    name = "io_bazel_rules_go",
    integrity = "sha256-fHbWI2so/2laoozzX5XeMXqUcv0fsUrHl8m/aE8Js3w=",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.44.2/rules_go-v0.44.2.zip",
        "https://github.com/bazelbuild/rules_go/releases/download/v0.44.2/rules_go-v0.44.2.zip",
    ],
)

load("@io_bazel_rules_go//go:deps.bzl", "go_download_sdk", "go_register_toolchains", "go_rules_dependencies")

go_rules_dependencies()

GO_SDK_VERSION = "1.21.3"

go_download_sdk(
    name = "go_sdk",
    goarch = "amd64",
    goos = "linux",
    version = GO_SDK_VERSION,
)

go_download_sdk(
    name = "go_sdk_linux_arm64",
    goarch = "arm64",
    goos = "linux",
    version = GO_SDK_VERSION,
)

go_download_sdk(
    name = "go_sdk_darwin",
    goarch = "amd64",
    goos = "darwin",
    version = GO_SDK_VERSION,
)

go_download_sdk(
    name = "go_sdk_darwin_arm64",
    goarch = "arm64",
    goos = "darwin",
    version = GO_SDK_VERSION,
)

go_download_sdk(
    name = "go_sdk_windows",
    goarch = "amd64",
    goos = "windows",
    version = GO_SDK_VERSION,
)

go_download_sdk(
    name = "go_sdk_windows_arm64",
    goarch = "arm64",
    goos = "windows",
    version = GO_SDK_VERSION,
)

go_register_toolchains(
    nogo = "@bazel_gazelle//:nogo",
)

load("//:deps.bzl", "gazelle_dependencies")

gazelle_dependencies()

# For API doc generation
# This is a dev dependency, users should not need to install it
# so we declare it in the WORKSPACE
http_archive(
    name = "io_bazel_stardoc",
    sha256 = "62bd2e60216b7a6fec3ac79341aa201e0956477e7c8f6ccc286f279ad1d96432",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/stardoc/releases/download/0.6.2/stardoc-0.6.2.tar.gz",
        "https://github.com/bazelbuild/stardoc/releases/download/0.6.2/stardoc-0.6.2.tar.gz",
    ],
)

# Stardoc pulls in a lot of deps, which we need to declare here.
load("@io_bazel_stardoc//:setup.bzl", "stardoc_repositories")

stardoc_repositories()

load("@rules_jvm_external//:repositories.bzl", "rules_jvm_external_deps")

rules_jvm_external_deps()

load("@rules_jvm_external//:setup.bzl", "rules_jvm_external_setup")

rules_jvm_external_setup()

load("@io_bazel_stardoc//:deps.bzl", "stardoc_external_deps")

stardoc_external_deps()

load("@stardoc_maven//:defs.bzl", stardoc_pinned_maven_install = "pinned_maven_install")

stardoc_pinned_maven_install()

load("@bazel_skylib//lib:unittest.bzl", "register_unittest_toolchains")

register_unittest_toolchains()

# gazelle:repository go_repository name=co_honnef_go_tools importpath=honnef.co/go/tools
# gazelle:repository go_repository name=com_github_bazelbuild_buildtools importpath=github.com/bazelbuild/buildtools
# gazelle:repository go_repository name=com_github_bazelbuild_rules_go importpath=github.com/bazelbuild/rules_go
# gazelle:repository go_repository name=com_github_bmatcuk_doublestar_v4 importpath=github.com/bmatcuk/doublestar/v4
# gazelle:repository go_repository name=com_github_burntsushi_toml importpath=github.com/BurntSushi/toml
# gazelle:repository go_repository name=com_github_census_instrumentation_opencensus_proto importpath=github.com/census-instrumentation/opencensus-proto
# gazelle:repository go_repository name=com_github_chzyer_logex importpath=github.com/chzyer/logex
# gazelle:repository go_repository name=com_github_chzyer_readline importpath=github.com/chzyer/readline
# gazelle:repository go_repository name=com_github_chzyer_test importpath=github.com/chzyer/test
# gazelle:repository go_repository name=com_github_client9_misspell importpath=github.com/client9/misspell
# gazelle:repository go_repository name=com_github_envoyproxy_go_control_plane importpath=github.com/envoyproxy/go-control-plane
# gazelle:repository go_repository name=com_github_envoyproxy_protoc_gen_validate importpath=github.com/envoyproxy/protoc-gen-validate
# gazelle:repository go_repository name=com_github_fsnotify_fsnotify importpath=github.com/fsnotify/fsnotify
# gazelle:repository go_repository name=com_github_golang_glog importpath=github.com/golang/glog
# gazelle:repository go_repository name=com_github_golang_mock importpath=github.com/golang/mock
# gazelle:repository go_repository name=com_github_golang_protobuf importpath=github.com/golang/protobuf
# gazelle:repository go_repository name=com_github_google_go_cmp importpath=github.com/google/go-cmp
# gazelle:repository go_repository name=com_github_pmezard_go_difflib importpath=github.com/pmezard/go-difflib
# gazelle:repository go_repository name=com_github_prometheus_client_model importpath=github.com/prometheus/client_model
# gazelle:repository go_repository name=com_google_cloud_go importpath=cloud.google.com/go
# gazelle:repository go_repository name=net_starlark_go importpath=go.starlark.net
# gazelle:repository go_repository name=org_golang_google_appengine importpath=google.golang.org/appengine
# gazelle:repository go_repository name=org_golang_google_genproto importpath=google.golang.org/genproto
# gazelle:repository go_repository name=org_golang_google_grpc importpath=google.golang.org/grpc
# gazelle:repository go_repository name=org_golang_google_protobuf importpath=google.golang.org/protobuf
# gazelle:repository go_repository name=org_golang_x_crypto importpath=golang.org/x/crypto
# gazelle:repository go_repository name=org_golang_x_exp importpath=golang.org/x/exp
# gazelle:repository go_repository name=org_golang_x_lint importpath=golang.org/x/lint
# gazelle:repository go_repository name=org_golang_x_mod importpath=golang.org/x/mod
# gazelle:repository go_repository name=org_golang_x_net importpath=golang.org/x/net
# gazelle:repository go_repository name=org_golang_x_oauth2 importpath=golang.org/x/oauth2
# gazelle:repository go_repository name=org_golang_x_sync importpath=golang.org/x/sync
# gazelle:repository go_repository name=org_golang_x_sys importpath=golang.org/x/sys
# gazelle:repository go_repository name=org_golang_x_text importpath=golang.org/x/text
# gazelle:repository go_repository name=org_golang_x_tools importpath=golang.org/x/tools
# gazelle:repository go_repository name=org_golang_x_tools_go_vcs importpath=golang.org/x/tools/go/vcs
# gazelle:repository go_repository name=org_golang_x_xerrors importpath=golang.org/x/xerrors

http_archive(
    name = "io_buildbuddy_buildbuddy_toolchain",
    sha256 = "e8ba5cf78c8a6268a08cf563c54d3d23a7edf288a16b39fadc8b8a27b2527155",
    strip_prefix = "buildbuddy-toolchain-f52e991c46e4bb6c71320db3970c20ce088ce951",
    urls = ["https://github.com/buildbuddy-io/buildbuddy-toolchain/archive/f52e991c46e4bb6c71320db3970c20ce088ce951.tar.gz"],
)

load("@io_buildbuddy_buildbuddy_toolchain//:deps.bzl", "buildbuddy_deps")

buildbuddy_deps()

load("@io_buildbuddy_buildbuddy_toolchain//:rules.bzl", "UBUNTU20_04_IMAGE", "buildbuddy")

buildbuddy(
    name = "buildbuddy_toolchain",
    container_image = UBUNTU20_04_IMAGE,
    gcc_version = "9",
)
