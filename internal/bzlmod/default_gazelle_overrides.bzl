visibility("private")

DEFAULT_BUILD_FILE_GENERATION_BY_PATH = {
    "github.com/envoyproxy/protoc-gen-validate": "on",
    "github.com/google/safetext": "on",
}

DEFAULT_DIRECTIVES_BY_PATH = {
    "github.com/census-instrumentation/opencensus-proto": [
        "gazelle:proto disable",
    ],
    "github.com/envoyproxy/protoc-gen-validate": [
        "gazelle:build_file_name BUILD.bazel",
    ],
    "github.com/gogo/protobuf": [
        "gazelle:proto disable",
    ],
    "github.com/google/gnostic": [
        "gazelle:proto disable",
    ],
    "github.com/google/safetext": [
        "gazelle:build_file_name BUILD.bazel",
        "gazelle:build_file_proto_mode disable_global",
    ],
    "github.com/googleapis/gnostic": [
        "gazelle:proto disable",
    ],
    "google.golang.org/grpc": [
        "gazelle:proto disable",
    ],
    "k8s.io/api": [
        "gazelle:proto disable",
    ],
    "k8s.io/apiextensions-apiserver": [
        "gazelle:proto disable",
    ],
    "k8s.io/apimachinery": [
        "gazelle:proto disable",
    ],
}
