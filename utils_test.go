package dotlite

import (
	"bytes"
	"testing"
)

func v(t *testing.T, b []byte, c int64) {
	if n, err := Varint(bytes.NewReader(b)); err != nil {
		t.Error(err)
	} else if n != c {
		t.Errorf("expected %d got %d", c, n)
	}
}

func ve(t *testing.T, b []byte) {
	if _, err := Varint(bytes.NewReader(b)); err == nil {
		t.Error("expected error to be non-nil")
	}
}

func TestVarint(t *testing.T) {
	v(t, []byte{0b0000_1000}, 8)
	v(t, []byte{0b1000_1000, 0b0000_0000}, 1024)
	v(t, []byte{0b1000_1000, 0b1000_0000, 0b0000_0011}, 131075)
	v(t, []byte{0b1000_0000, 0b1000_0000, 0b1000_0000, 0b1000_0000, 0b1000_0000, 0b1000_0000, 0b1000_0000, 0b1000_0000, 0b0000_0001}, 1)
	v(t, []byte{0b1000_0000, 0b1000_0000, 0b1000_0000, 0b1000_0000, 0b1000_0000, 0b1000_0000, 0b1000_0000, 0b1000_0000, 0b0000_1010}, 10)

	// error cases
	ve(t, []byte{0b1000_0000})
}
