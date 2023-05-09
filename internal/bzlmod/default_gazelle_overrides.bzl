visibility("private")

DEFAULT_BUILD_FILE_GENERATION_BY_PATH = {
    "github.com/google/safetext": "on",
}

DEFAULT_DIRECTIVES_BY_PATH = {
    "github.com/google/safetext": [
        "gazelle:build_file_name BUILD.bazel",
    ],
}
