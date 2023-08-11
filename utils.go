package dotlite

import (
	"io"
)

func min(a, b int) int {
	if a > b {
		return b
	}
	return a
}

// Varint computes a 64-bit integer value from the given source.
//
// It differs slightly from binary.ReadVarint and follows sqlite's logic for
// computing the integer value from the bytes.
//
// see: https://www.sqlite.org/fileformat.html#b_tree_pages description for more details
func Varint(r io.ByteReader) (_ int64, err error) {
	var b byte
	var val uint64
	for i := 0; i < 8; i++ {
		if b, err = r.ReadByte(); err != nil {
			return 0, err
		}

		val = (val << 7) | uint64(b&0x7f)
		if b < 0x80 {
			return int64(val), nil
		}
	}

	if b, err = r.ReadByte(); err != nil {
		return 0, err
	}

	return int64((val << 8) | uint64(b)), nil
}
