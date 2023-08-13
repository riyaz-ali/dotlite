package dotlite

import (
	"bytes"
	"encoding/hex"
	"io"
	"testing"
)

func TestOverflow(t *testing.T) {
	var buf = read(t, "testdata/overflow-pages.bin")
	var reader = bytes.NewReader(buf)

	var pager = &Pager{size: 16, pages: 6, file: reader}
	var or = newOverflowReader(pager, 1, pager.size, 64 /* size of overflow content */)

	var sink bytes.Buffer

	if n, err := io.Copy(&sink, or); err != nil {
		t.Error(err)
	} else if n != 64 {
		t.Errorf("expected to read %d bytes; got %d", 48, n)
	}

	t.Logf("content: \n%s", hex.Dump(sink.Bytes()))
}
