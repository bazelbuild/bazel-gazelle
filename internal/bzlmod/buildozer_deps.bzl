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
        urls = ["https://github.com/bazelbuild/buildtools/releases/download/5.1.0/buildozer-darwin-arm64"],
        sha256 = "57f8d90fac6b111bd0859b97847d3db2ce71419f44588b0e91250892037cf638",
        downloaded_file_path = "buildozer",
        executable = True,
    )

    maybe(
        http_file,
        name = "buildozer_macos_x86",
        urls = ["https://github.com/bazelbuild/buildtools/releases/download/5.1.0/buildozer-darwin-amd64"],
        sha256 = "294f4d0790f4dba18c9b7617f57563e07c2c7e529a8915bcbc49170dc3c08eb9",
        downloaded_file_path = "buildozer",
        executable = True,
    )

    maybe(
        http_file,
        name = "buildozer_linux_arm",
        urls = ["https://github.com/bazelbuild/buildtools/releases/download/5.1.0/buildozer-linux-arm64"],
        sha256 = "0b08e384709ec4d4f5320bf31510d2cefe8f9e425a6565b31db06b2398ff9dc4",
        downloaded_file_path = "buildozer",
        executable = True,
    )

    maybe(
        http_file,
        name = "buildozer_linux_x86",
        urls = ["https://github.com/bazelbuild/buildtools/releases/download/5.1.0/buildozer-linux-amd64"],
        sha256 = "7346ce1396dfa9344a5183c8e3e6329f067699d71c4391bd28317391228666bf",
        downloaded_file_path = "buildozer",
        executable = True,
    )
