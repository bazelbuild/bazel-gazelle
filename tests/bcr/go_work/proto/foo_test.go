package proto

import (
	"testing"

	"github.com/bazelbuild/bazel-gazelle/tests/bcr/proto/foo"
	"google.golang.org/protobuf/types/known/sourcecontextpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/typepb"
)

func TestWellKnownTypes(t *testing.T) {
	var foo foo.Foo
	foo.Name = "foo"
	foo.Type = &typepb.Type{
		Name:          "my_type",
		SourceContext: &sourcecontextpb.SourceContext{},
	}
	foo.LastUpdated = &timestamppb.Timestamp{
		Seconds: 12345,
		Nanos:   67890,
	}
}
