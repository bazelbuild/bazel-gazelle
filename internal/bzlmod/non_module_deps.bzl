load(
    "//internal:go_repository_cache.bzl",
    "go_repository_cache",
)
load(
    "//internal:go_repository_tools.bzl",
    "go_repository_tools",
)
load(
    "@go_host_compatible_sdk_label//:defs.bzl",
    "HOST_COMPATIBLE_SDK",
)

visibility("//")

def _non_module_deps_impl(_):
    go_repository_cache(
        name = "bazel_gazelle_go_repository_cache",
        # Label.workspace_name is always a canonical name, so use a canonical label.
        go_sdk_name = "@" + HOST_COMPATIBLE_SDK.workspace_name,
        go_env = {},
    )
    go_repository_tools(
        name = "bazel_gazelle_go_repository_tools",
        go_cache = Label("@bazel_gazelle_go_repository_cache//:go.env"),
    )

non_module_deps = module_extension(
    _non_module_deps_impl,
)
