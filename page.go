package dotlite

import (
	"bytes"
	"fmt"
	"io"
)

// Page represents a single page in the sqlite database file
type Page struct {
	*bytes.Reader

	ID int // location of the page in the database file
}

// Pager is a service used to fetch pages from the database file
type Pager struct {
	size, pages int
	file        io.ReadSeeker
}

// ReadPage reads a single page, identified by its location / id, from the database file
func (pager *Pager) ReadPage(i int) (_ *Page, err error) {
	if i > pager.pages {
		return nil, fmt.Errorf("page index out of range (%d > %d)", i, pager.pages)
	}

	pos, _ := pager.file.Seek(0, io.SeekCurrent)
	defer pager.file.Seek(pos, io.SeekStart)

	var buf = make([]byte, pager.size)
	var pageOffset = int64((i - 1) * pager.size)
	if _, err = pager.file.Seek(pageOffset, io.SeekStart); err != nil {
		return nil, err
	}

	var n int
	if n, err = pager.file.Read(buf); err != nil {
		return nil, err
	}

	if n < len(buf) {
		return nil, fmt.Errorf("read fewer bytes than page size")
	}

	return &Page{Reader: bytes.NewReader(buf), ID: i}, nil
}
