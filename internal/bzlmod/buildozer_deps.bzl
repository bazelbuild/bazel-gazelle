load(
    "@bazel_tools//tools/build_defs/repo:http.bzl",
    "http_file",
)
load(
    "@bazel_tools//tools/build_defs/repo:utils.bzl",
    "maybe",
)

def buildozer_deps():
    """Buildozer dependencies, shared between bzlmod and workspace builds"""

    maybe(
        http_file,
        name = "buildozer_macos_arm",
        urls = ["https://github.com/bazelbuild/buildtools/releases/download/5.0.1/buildozer-darwin-arm64"],
        sha256 = "1d8a3b0e3a702630d36de012fd5cf93342af0fa1b5853643b318667345c84df0",
        downloaded_file_path = "buildozer",
        executable = True,
    )

    maybe(
        http_file,
        name = "buildozer_macos_x86",
        urls = ["https://github.com/bazelbuild/buildtools/releases/download/5.0.1/buildozer-darwin-amd64"],
        sha256 = "17a093596f141ead6ff70ac217a063d7aebc86174faa8ab43620392c17b8ee61",
        downloaded_file_path = "buildozer",
        executable = True,
    )

    maybe(
        http_file,
        name = "buildozer_linux_x86",
        urls = ["https://github.com/bazelbuild/buildtools/releases/download/5.0.1/buildozer-linux-amd64"],
        sha256 = "78204dac0ac6a94db499c57c5334b9c0c409d91de9779032c73ad42f2362e901",
        downloaded_file_path = "buildozer",
        executable = True,
    )
