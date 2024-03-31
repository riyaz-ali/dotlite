package dotlite

import (
	"fmt"
	"io"
)

// Page represents a single page in the sqlite database file
type Page struct {
	*io.SectionReader
	ID int // location of the page in the database file
}

func (page *Page) Remaining() int64 {
	read, _ := page.Seek(0, io.SeekCurrent)
	return page.Size() - read
}

// Pager is a service used to fetch pages from the database file
type Pager struct {
	size, pages int
	file        io.ReaderAt
}

// ReadPage reads a single page, identified by its location / id, from the database file
func (pager *Pager) ReadPage(i int) (_ *Page, err error) {
	if i > pager.pages {
		return nil, fmt.Errorf("page index out of range (%d > %d)", i, pager.pages)
	}

	var pageOffset = int64((i - 1) * pager.size)
	return &Page{ID: i, SectionReader: io.NewSectionReader(pager.file, pageOffset, int64(pager.size))}, nil
}
