package proto

import (
	_ "embed"
	"encoding/csv"
	"log"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/label"
)

type protoCSVEntry struct {
	filename            string
	protoLibraryLabel   label.Label
	importPath          string
	goProtoLibraryLabel label.Label
}

var knownImports map[string]*protoCSVEntry
var knownGoProtoLibraryLabelByImportpath map[string]label.Label

// knownImport returns the well-known proto_library label for a given proto
// filename (e.g. google/protobuf/any.proto -> @com_google_protobuf//:any_proto)
func knownImport(filename string) (label.Label, bool) {
	if knownImports == nil {
		mustParseProtoCSV()
	}
	imp, ok := knownImports[filename]
	if !ok {
		return label.NoLabel, false
	}
	return imp.protoLibraryLabel, true
}

// knownProtoImport returns the well-known go_proto_library for a given filename
// (e.g. google/protobuf/any.proto ->
// @io_bazel_rules_go//proto/wkt:any_go_proto)
func knownProtoImport(filename string) (label.Label, bool) {
	if knownImports == nil {
		mustParseProtoCSV()
	}
	imp, ok := knownImports[filename]
	if !ok {
		return label.NoLabel, false
	}
	return imp.goProtoLibraryLabel, true
}

// knownGoProtoImport returns the well-known go_proto_library label for a given
// importpath (e.g. github.com/golang/protobuf/ptypes/any ->
// @io_bazel_rules_go//proto/wkt:any_go_proto)
func knownGoProtoImport(name string) (label.Label, bool) {
	if knownImports == nil {
		mustParseProtoCSV()
	}
	from, ok := knownGoProtoLibraryLabelByImportpath[name]
	return from, ok
}

// mustParseProtoCSV takes the parsed csv file and populates the known import
// maps.
func mustParseProtoCSV() {
	imports, err := parseProtoCSV()
	if err != nil {
		log.Fatalf("failed to read embedded proto.csv: %v", err)
		return
	}

	knownImports = make(map[string]*protoCSVEntry)
	knownGoProtoLibraryLabelByImportpath = make(map[string]label.Label)

	for _, imp := range imports {
		knownImports[imp.filename] = imp
		knownGoProtoLibraryLabelByImportpath[imp.importPath] = imp.goProtoLibraryLabel
	}
}

// parseProtoCSV parses the embedded file 'proto.csv'
func parseProtoCSV() ([]*protoCSVEntry, error) {
	r := csv.NewReader(strings.NewReader(protoCSVRaw))
	r.Comment = '#'
	r.FieldsPerRecord = 4
	records, err := r.ReadAll()
	if err != nil {
		return nil, err
	}

	imports := make([]*protoCSVEntry, len(records))

	for i, rec := range records {
		imports[i] = &protoCSVEntry{
			filename:            rec[0],
			protoLibraryLabel:   mustParseLabel(rec[1]),
			importPath:          rec[2],
			goProtoLibraryLabel: mustParseLabel(rec[3]),
		}
	}

	return imports, nil
}

func mustParseLabel(s string) label.Label {
	lbl, err := label.Parse(s)
	if err != nil {
		log.Fatalf("failed to parse known import label: %v", err)
	}
	return lbl
}
