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
    sha256 = "2b1641428dff9018f9e85c0384f03ec6c10660d935b750e3fa1492a281a53b0f",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.29.0/rules_go-v0.29.0.zip",
        "https://github.com/bazelbuild/rules_go/releases/download/v0.29.0/rules_go-v0.29.0.zip",
    ],
)

load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")

go_rules_dependencies()

go_register_toolchains(
    nogo = "@bazel_gazelle//:nogo",
    version = "1.17.2",
)

load("//:deps.bzl", "gazelle_dependencies")

gazelle_dependencies()

# gazelle:repository go_repository name=co_honnef_go_tools importpath=honnef.co/go/tools
# gazelle:repository go_repository name=com_github_bazelbuild_buildtools importpath=github.com/bazelbuild/buildtools build_naming_convention=go_default_library
# gazelle:repository go_repository name=com_github_bazelbuild_rules_go importpath=github.com/bazelbuild/rules_go
# gazelle:repository go_repository name=com_github_bmatcuk_doublestar importpath=github.com/bmatcuk/doublestar
# gazelle:repository go_repository name=com_github_burntsushi_toml importpath=github.com/BurntSushi/toml
# gazelle:repository go_repository name=com_github_census_instrumentation_opencensus_proto importpath=github.com/census-instrumentation/opencensus-proto
# gazelle:repository go_repository name=com_github_chzyer_logex importpath=github.com/chzyer/logex
# gazelle:repository go_repository name=com_github_chzyer_readline importpath=github.com/chzyer/readline
# gazelle:repository go_repository name=com_github_chzyer_test importpath=github.com/chzyer/test
# gazelle:repository go_repository name=com_github_client9_misspell importpath=github.com/client9/misspell
# gazelle:repository go_repository name=com_github_davecgh_go_spew importpath=github.com/davecgh/go-spew
# gazelle:repository go_repository name=com_github_envoyproxy_go_control_plane importpath=github.com/envoyproxy/go-control-plane
# gazelle:repository go_repository name=com_github_envoyproxy_protoc_gen_validate importpath=github.com/envoyproxy/protoc-gen-validate
# gazelle:repository go_repository name=com_github_fsnotify_fsnotify importpath=github.com/fsnotify/fsnotify
# gazelle:repository go_repository name=com_github_golang_glog importpath=github.com/golang/glog
# gazelle:repository go_repository name=com_github_golang_mock importpath=github.com/golang/mock
# gazelle:repository go_repository name=com_github_golang_protobuf importpath=github.com/golang/protobuf
# gazelle:repository go_repository name=com_github_google_go_cmp importpath=github.com/google/go-cmp
# gazelle:repository go_repository name=com_github_pelletier_go_toml importpath=github.com/pelletier/go-toml
# gazelle:repository go_repository name=com_github_pmezard_go_difflib importpath=github.com/pmezard/go-difflib
# gazelle:repository go_repository name=com_github_prometheus_client_model importpath=github.com/prometheus/client_model
# gazelle:repository go_repository name=com_github_yuin_goldmark importpath=github.com/yuin/goldmark
# gazelle:repository go_repository name=com_google_cloud_go importpath=cloud.google.com/go
# gazelle:repository go_repository name=in_gopkg_check_v1 importpath=gopkg.in/check.v1
# gazelle:repository go_repository name=in_gopkg_yaml_v2 importpath=gopkg.in/yaml.v2
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
# gazelle:repository go_repository name=org_golang_x_xerrors importpath=golang.org/x/xerrors
