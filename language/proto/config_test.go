package proto

import "testing"

func TestCheckStripImportPrefix(t *testing.T) {
	e := checkStripImportPrefix("/example.com/idl", "example.com")
	wantErr := "invalid proto_strip_import_prefix \"/example.com/idl\" at example.com"
	if e == nil || e.Error() != wantErr {
		t.Errorf("got:\n%v\n\nwant:\n%s\n", e, wantErr)
	}
}