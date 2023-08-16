package x

import (
	"encoding/json"
	. "go.riyazali.net/dotlite"
	"testing"
)

func open(t *testing.T, name string) *File {
	var file, err = OpenFile(name)
	if err != nil {
		t.Errorf("failed to open file: %v", err)
	}

	return file
}

func TestTable(t *testing.T) {
	var file = open(t, "../testdata/chinook.db")
	defer file.Close()

	schema, err := file.Schema()
	if err != nil {
		t.Error(err)
	}

	for _, obj := range schema {
		if obj.Type() == "table" {
			var tab *Table
			if tab, err = ParseSchema(obj.SQL()); err != nil {
				t.Error(err)
			} else {
				var b, _ = json.MarshalIndent(tab, "", "  ")
				t.Logf("table(%s):\n%s", obj.Name(), b)
			}
		}
	}
}
