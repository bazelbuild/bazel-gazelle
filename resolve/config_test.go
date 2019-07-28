package resolve

import (
	"github.com/bazelbuild/bazel-gazelle/label"
	"reflect"
	"testing"
)

func TestOverrideSpecFromString(t *testing.T)  {
	s := "proto|mesos/isolator.proto|@apache//mesos:isolator_proto"
	o, err := overrideSpecFromString(s, "|")
	if err != nil {
		t.Errorf("failed to parse resolve string %s: %v", s, err)
	}
	wantedSpec := overrideSpec{
		imp:ImportSpec{
			Imp:"mesos/isolator.proto",
			Lang:"proto",
		},
		dep:label.Label{
			Repo:"apache",
			Pkg:"mesos",
			Name:"isolator_proto",
		},
	}
	if !reflect.DeepEqual(o, wantedSpec) {
		t.Errorf("got %#v; want %#v", o, wantedSpec)
	}
}
