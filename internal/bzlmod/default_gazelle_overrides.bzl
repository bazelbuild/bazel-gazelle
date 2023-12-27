# Copyright 2023 The Bazel Authors. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

visibility("private")

DEFAULT_BUILD_FILE_GENERATION_BY_PATH = {
    "github.com/envoyproxy/protoc-gen-validate": "on",
    "github.com/google/safetext": "on",
    "github.com/grpc-ecosystem/grpc-gateway/v2": "on",
    "google.golang.org/grpc": "on",
}

DEFAULT_DIRECTIVES_BY_PATH = {
    "github.com/census-instrumentation/opencensus-proto": [
        "gazelle:proto disable",
    ],
    "github.com/envoyproxy/protoc-gen-validate": [
        "gazelle:build_file_name BUILD.bazel",
    ],
    "github.com/gogo/googleapis": [
        "gazelle:go_generate_proto false",
    ],
    "github.com/gogo/protobuf": [
        "gazelle:proto disable",
    ],
    "github.com/google/gnostic": [
        "gazelle:proto disable",
    ],
    "github.com/google/gnostic-models": [
        "gazelle:proto disable",
    ],
    "github.com/google/safetext": [
        "gazelle:build_file_name BUILD.bazel",
        "gazelle:build_file_proto_mode disable_global",
    ],
    "github.com/googleapis/gax-go/v2": [
        "gazelle:proto disable",
    ],
    "github.com/googleapis/gnostic": [
        "gazelle:proto disable",
    ],
    "github.com/pseudomuto/protoc-gen-doc": [
        "gazelle:resolve go github.com/mwitkow/go-proto-validators @com_github_mwitkow_go_proto_validators//:validators_gogo",
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
        "gazelle:go_generate_proto false",
        "gazelle:proto_import_prefix k8s.io/apimachinery",
        "gazelle:resolve proto k8s.io/apimachinery/pkg/runtime/generated.proto @@io_k8s_apimachinery//pkg/runtime:runtime_proto",
        "gazelle:resolve proto k8s.io/apimachinery/pkg/runtime/schema/generated.proto @@io_k8s_apimachinery//pkg/runtime/schema:schema_proto",
    ],
}

DEFAULT_BUILD_EXTRA_ARGS_BY_PATH = {
    "github.com/census-instrumentation/opencensus-proto": [
        "-exclude=src",
    ],
}
