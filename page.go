package dotlite

import (
	"errors"
	"fmt"
	"io"
)

// Page represents a single page in the sqlite database file
type Page struct {
	ID int // location of the page in the database file

	// used to implement io.ReadSeeker and io.ByteReader interfaces
	// taken from bytes.Reader implementation

	s []byte
	i int64 // current reading index
}

func (page *Page) Len() int {
	if page.i >= int64(len(page.s)) {
		return 0
	}
	return int(int64(len(page.s)) - page.i)
}

func (page *Page) Seek(offset int64, whence int) (int64, error) {
	var abs int64
	switch whence {
	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs = page.i + offset
	case io.SeekEnd:
		abs = int64(len(page.s)) + offset
	default:
		return 0, errors.New("Page.Seek: invalid whence")
	}
	if abs < 0 {
		return 0, errors.New("Page.Seek: negative position")
	}
	page.i = abs
	return abs, nil
}

func (page *Page) Read(b []byte) (n int, err error) {
	if page.i >= int64(len(page.s)) {
		return 0, io.EOF
	}

	n = copy(b, page.s[page.i:])
	page.i += int64(n)
	return
}

func (page *Page) ReadByte() (byte, error) {
	if page.i >= int64(len(page.s)) {
		return 0, io.EOF
	}
	b := page.s[page.i]
	page.i++
	return b, nil
}

func (page *Page) Region(i int64) []byte {
	var b = page.s[page.i : page.i+i]
	page.i += i
	return b
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

	if _, err = io.ReadFull(pager.file, buf); err != nil {
		return nil, err
	}

	return &Page{s: buf, i: 0, ID: i}, nil
}
