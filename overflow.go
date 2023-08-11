package dotlite

import (
	"encoding/binary"
	"io"
)

// io.Reader implementation to hide details for reading from overflow pages
// see: https://www.sqlite.org/fileformat.html#ovflpgs
type overflow struct {
	next  int32 // next page in the chain; 0 if this is the last
	page  *Page // current page we are reading
	pager *Pager

	usable int // configured usable size of the page
	size   int // total size of the overflow content
	left   int // bytes left to read in overflow
}

func newOverflowReader(pager *Pager, page int32, usable, size int) *overflow {
	return &overflow{pager: pager, next: page, usable: usable, size: size, left: size}
}

func (o *overflow) Read(buf []byte) (n int, err error) {
	if (o.page == nil || o.page.Len() == 0 || o.left == 0) && o.next == 0 {
		if o.left != 0 {
			return 0, io.ErrUnexpectedEOF
		}
		return 0, io.EOF
	}

fetch:
	if o.page == nil || o.page.Len() == 0 {
		if o.page, err = o.pager.ReadPage(int(o.next)); err != nil {
			return 0, err
		}

		// next page in the chain
		if err = binary.Read(o.page, binary.BigEndian, &o.next); err != nil {
			return 0, err
		}
	}

	buf = buf[:min(len(buf), o.left, o.usable-4)]
	if n, err = o.page.Read(buf); err != nil {
		if err == io.EOF && o.next == 0 {
			if o.next == 0 && o.left-n != 0 { // we expected more but hit an unexpected EOF
				return n, io.ErrUnexpectedEOF
			}

			buf = buf[n:] // update buffer start position
			goto fetch    // read the next page
		}

		return n, err
	}

	o.left -= n
	return n, nil
}
