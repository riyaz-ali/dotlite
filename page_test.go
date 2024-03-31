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

type errStream struct{ AllowRead bool }

func (e *errStream) ReadAt(p []byte, off int64) (n int, err error) {
	//TODO implement me
	panic("implement me")
}

func (e *errStream) Read(_ []byte) (_ int, _ error) {
	if e.AllowRead {
		return 0, io.EOF
	} else {
		return 0, fmt.Errorf("read failed")
	}
}

func TestPage_Read(t *testing.T) {
	var buf = read(t, "testdata/only-pages.bin")
	var reader = bytes.NewReader(buf)
	var pager = &Pager{size: 512, pages: 4, file: reader}

	var sink bytes.Buffer

	var page, _ = pager.ReadPage(1)
	if sz := page.Remaining(); sz != 512 {
		t.Errorf("expected length to be %d; got %d", 512, sz)
	}

	if n, err := io.Copy(&sink, page); err != nil {
		t.Error(err)
	} else if n != 512 {
		t.Errorf("expected to read %d bytes; got %d", 512, n)
	}

	if sz := page.Remaining(); sz != 0 {
		t.Errorf("expected length to be %d; got %d", 0, sz)
	}

	if !bytes.Equal(buf[:512], sink.Bytes()) {
		t.Errorf("content not equal")
	}
}
