visibility("private")

DEFAULT_DIRECTIVES_BY_PATH = {
    "github.com/census-instrumentation/opencensus-proto": [
        "gazelle:proto disable",
    ],
    "github.com/gogo/protobuf": [
        "gazelle:proto disable",
    ],
    "github.com/google/gnostic": [
        "gazelle:proto disable",
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
    "k8s.io/apimachinery": [
        "gazelle:proto disable",
    ],
}
