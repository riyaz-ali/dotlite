package dotlite

import "testing"

func TestTable(t *testing.T) {
	var file = open(t, "testdata/all-kinds.db") // well technically most 😅
	defer file.Close()

	table, err := file.Object("x")
	if err != nil {
		t.Error(err)
	}

	err = table.ForEach(func(record *Record) error {
		t.Logf("record(%p):", record)
		for i := 0; i < record.NumValues(); i++ {
			var val any
			if val, err = record.ValueAt(i); err != nil {
				return err
			}
			t.Logf("\tval(%d): %+v", i, val)
		}

		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

func TestIndex(t *testing.T) {
	var file = open(t, "testdata/chinook.db")
	defer file.Close()

	index, err := file.Object("IDX_album_title")
	if err != nil {
		t.Error(err)
	}

	err = index.ForEach(func(record *Record) (err error) {
		t.Logf("record(%p):", record)
		for i := 0; i < record.NumValues(); i++ {
			var val any
			if val, err = record.ValueAt(i); err != nil {
				return err
			}
			t.Logf("\tval(%d): %+v", i, val)
		}

		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

func TestIndex_without_rowid(t *testing.T) {
	var file = open(t, "testdata/without-rowid.db")
	defer file.Close()

	index, err := file.Object("wordcount")
	if err != nil {
		t.Error(err)
	}

	err = index.ForEach(func(record *Record) (err error) {
		t.Logf("record(%p):", record)
		for i := 0; i < record.NumValues(); i++ {
			var val any
			if val, err = record.ValueAt(i); err != nil {
				return err
			}
			t.Logf("\tval(%d): %+v", i, val)
		}

		return nil
	})
	if err != nil {
		t.Error(err)
	}
}
