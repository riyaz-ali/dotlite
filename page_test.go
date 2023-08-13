package dotlite

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"
)

func read(t *testing.T, name string) []byte {
	file, err := os.Open(name)
	if err != nil {
		t.Errorf("failed to open file: %v", err)
	}

	var buf bytes.Buffer
	if _, err = io.Copy(&buf, file); err != nil {
		t.Errorf("failed to read file: %v", err)
	}

	_ = file.Close()
	return buf.Bytes()
}

func TestPager(t *testing.T) {
	var buf = read(t, "testdata/only-pages.bin")
	var reader = bytes.NewReader(buf)
	var pager = &Pager{size: 512, pages: 4, file: reader}

	if _, err := pager.ReadPage(5); err == nil {
		t.Errorf("expected index out of range; got nothing")
	}

	if page, err := pager.ReadPage(1); err != nil || page == nil {
		t.Errorf("failed to read page #1")
	}
}

type errStream struct{ AllowSeek, AllowRead bool }

func (e *errStream) Seek(offset int64, _ int) (int64, error) {
	if e.AllowSeek {
		return offset, nil
	} else {
		return 0, fmt.Errorf("seek failed")
	}
}

func (e *errStream) Read(_ []byte) (_ int, _ error) {
	if e.AllowRead {
		return 0, io.EOF
	} else {
		return 0, fmt.Errorf("read failed")
	}
}

func TestPager_error_when_reading(t *testing.T) {
	var reader errStream
	var pager = &Pager{size: 512, pages: 4, file: &reader}

	if _, err := pager.ReadPage(1); err == nil {
		t.Errorf("expected error when seeking; got nothing")
	}

	reader.AllowSeek = true
	if _, err := pager.ReadPage(1); err == nil {
		t.Errorf("expected error when reading; got nothing")
	}

	reader.AllowRead = true
	if _, err := pager.ReadPage(1); err == nil {
		t.Errorf("expected short read error; got nothing")
	}
}

func TestPage_Read(t *testing.T) {
	var buf = read(t, "testdata/only-pages.bin")
	var reader = bytes.NewReader(buf)
	var pager = &Pager{size: 512, pages: 4, file: reader}

	var sink bytes.Buffer

	var page, _ = pager.ReadPage(1)
	if sz := page.Len(); sz != 512 {
		t.Errorf("expected length to be %d; got %d", 512, sz)
	}

	if n, err := io.Copy(&sink, page); err != nil {
		t.Error(err)
	} else if n != 512 {
		t.Errorf("expected to read %d bytes; got %d", 512, n)
	}

	if sz := page.Len(); sz != 0 {
		t.Errorf("expected length to be %d; got %d", 0, sz)
	}

	if !bytes.Equal(buf[:512], sink.Bytes()) {
		t.Errorf("content not equal")
	}
}

func TestPage_corrupt_page(t *testing.T) {
	var buf = read(t, "testdata/corrupt-page.bin")
	var reader = bytes.NewReader(buf)
	var pager = &Pager{size: 512, pages: 1, file: reader}

	if _, err := pager.ReadPage(1); err == nil {
		t.Errorf("expected error to be non-nil")
	}
}
