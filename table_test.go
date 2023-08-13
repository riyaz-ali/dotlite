package dotlite

import "testing"

func TestTable(t *testing.T) {
	var file = open(t, "testdata/all-kinds.db") // well technically most 😅
	defer file.Close()

	table, err := file.Table("x")
	if err != nil {
		t.Error(err)
	}

	err = table.ForEach(func(values []any) error { t.Logf("value: %v", values); return nil })
	if err != nil {
		t.Error(err)
	}
}
