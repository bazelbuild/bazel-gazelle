package proto

import (
	"testing"
)

var protoCSVResult []*protoCSVEntry

func TestMustParseCSV(t *testing.T) {
	mustParseProtoCSV()
}

func BenchmarkParseProtoCSV(b *testing.B) {
	var result []*protoCSVEntry
	var err error
	for n := 0; n < b.N; n++ {
		if result, err = parseProtoCSV(); err != nil {
			b.Fatal(err)
		} else {
			protoCSVResult = result
		}
	}
}
