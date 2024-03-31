package dotlite

import (
	"testing"
)

func open(t *testing.T, name string) *File {
	var file, err = Open(name)
	if err != nil {
		t.Errorf("failed to open file: %v", err)
	}

	return file
}

func TestOpen(t *testing.T) {
	var file = open(t, "testdata/chinook.db")
	defer file.Close()

	if sz := file.PageSize(); sz != 1024 {
		t.Errorf("expected page size to be %d; got %d", 1024, sz)
	}

	if ver := file.Version(); ver != 3041000 {
		t.Errorf("expected library version to be %d; got %d", 3041000, ver)
	}

	if enc := file.Encoding(); enc != UTF8 {
		t.Errorf("expected encoding to be %d; got %d", UTF8, enc)
	}
}

func TestOpen_invalid_magic(t *testing.T) {
	if _, err := Open("testdata/not-a-database.txt"); err == nil {
		t.Errorf("expected invalid magic error")
	}
}

func TestOpen_size_is_computed(t *testing.T) {
	// 4 bytes starting at position 28 are zeroed
	var file = open(t, "testdata/chinook-no-size.db")
	defer file.Close()

	if sz := file.NumPages(); sz != 1042 {
		t.Errorf("expected page count to be %d; got %d", 1042, sz)
	}
}

func TestSchema(t *testing.T) {
	var file = open(t, "testdata/chinook.db")
	defer file.Close()

	objects, err := file.Schema()
	if err != nil {
		t.Errorf("failed to determine schema: %v", err)
	}

	if tl := len(objects); tl != 23 {
		t.Errorf("expected %d objects; got %d", 23, tl)
	}
}

func TestSchema_find_table(t *testing.T) {
	var file = open(t, "testdata/chinook.db")
	defer file.Close()

	if _, err := file.Object("Album"); err != nil {
		t.Error(err)
	}

	if table, err := file.Object("NotExist"); err == nil || table != nil {
		t.Fail()
	}
}

func TestOverflow_database(t *testing.T) {
	var file = open(t, "testdata/overflow.db")
	defer file.Close()

	var err = file.ForEach("x", func(_ *Record) error {
		return nil
	})

	if err != nil {
		t.Error(err)
	}
}
