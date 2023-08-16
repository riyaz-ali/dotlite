package dotlite

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// Magic is the 16-byte constant magic value used by sqlite3
const Magic = "SQLite format 3\x00"

// TextEncoding represents the database text encoding
type TextEncoding int32

const (
	_ TextEncoding = iota
	UTF8
	UTF16LE
	UTF16BE
)

// Header describes the sqlite3 database header as defined under https://www.sqlite.org/fileformat.html#the_database_header
type Header struct {
	Magic           [16]byte
	PageSize        uint16  // the database page size in bytes
	WriteVersion    byte    // file format write version
	ReadVersion     byte    // file format read version
	PageReserved    byte    // bytes of unused reserved space at the end of each page; usually 0
	MaxEmbeddedFrac byte    // maximum embedded payload fraction. Must be 64
	MinEmbeddedFrac byte    // minimum embedded payload fraction. Must be 32
	LeafFrac        byte    // leaf payload fraction (must be 32)
	ChangeCounter   int32   // file change counter
	Size            int32   // size of the database file in pages
	FreePage        int32   // page number of the first freelist trunk page
	TotalFreePages  int32   // total number of freelist pages
	SchemaCookie    [4]byte // the schema cookie
	SchemaFormat    int32   // the schema format number. Supported schema formats are 1, 2, 3, and 4.
	PageCacheSize   int32   // default page cache size
	AutoVacuum      int32   // page number of the largest root b-tree page when in auto-vacuum or incremental-vacuum modes, or zero otherwise.
	TextEncoding    TextEncoding
	UserVersion     int32 // the "user version" as read and set by the user_version PRAGMA
	IncrVacuum      int32 // True (non-zero) for incremental-vacuum mode. False (zero) otherwise
	ApplicationID   int32 // the "Application ID" set by the PRAGMA application_id

	_ [20]byte // reserved for expansion. Must be zero.

	VersionValid   int32 // the version-valid-for number; see: https://www.sqlite.org/fileformat2.html#validfor
	LibraryVersion int32
}

// Valid validates the header ensuring it is well-formed and correct.
func (h *Header) Valid() error {
	if string(h.Magic[:]) != Magic {
		return fmt.Errorf("invalid header")
	}

	// ensure file can be read
	if h.ReadVersion > 2 {
		return fmt.Errorf("file not readable by current version of library")
	}

	// Ensure reserved space at the end of the page is valid.
	// The documentation states that "the usable size is not allowed to be less than 480 [bytes]"
	if usable := h.PageSize - uint16(h.PageReserved); usable < 480 {
		return fmt.Errorf("invalid file: usable page size is less than allowed limit")
	}

	// ensure payload fraction values are fixed; see: https://www.sqlite.org/fileformat.html#payload_fractions
	if h.MaxEmbeddedFrac != 64 || h.MinEmbeddedFrac != 32 || h.LeafFrac != 32 {
		return fmt.Errorf("invalid payload fractions")
	}

	return nil
}

// File represents a sqlite3 database file
type File struct {
	Header Header // sqlite3 database header; see: https://www.sqlite.org/fileformat.html#the_database_header

	//-  start of internal state
	file   io.ReadSeeker // the underlying file reference
	closer io.Closer
	Pager  *Pager // pager used to fetch pages
}

// Open reads the stream from f as a sqlite database file.
func Open(f io.ReadSeekCloser) (_ *File, err error) {
	var header Header
	if err = binary.Read(f, binary.BigEndian, &header); err != nil {
		return nil, err
	}

	// determine database size (in pages) if any of this condition is met
	// see: https://www.sqlite.org/fileformat.html#in_header_database_size
	if header.Size == 0 || (header.ChangeCounter != header.VersionValid) {
		var size int64
		if size, err = f.Seek(0, io.SeekEnd); err != nil {
			return nil, err
		}

		if _, err = f.Seek(0, io.SeekStart); err != nil { // reset
			return nil, err
		}

		var pages = (size + int64(header.PageSize) - 1) / int64(header.PageSize)
		header.Size = int32(pages)
	}

	if err = header.Valid(); err != nil {
		return nil, err
	}

	if _, err = f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	// pager is used to fetch and read pages of data from the database file
	// other high-level constructs (such as free-list and btree) builds on top of pager
	var pager = &Pager{file: f, size: int(header.PageSize), pages: int(header.Size)}

	var file = &File{Header: header, Pager: pager, file: f, closer: f}
	return file, nil
}

func OpenFile(name string) (_ *File, err error) {
	var file *os.File
	if file, err = os.Open(name); err != nil {
		return nil, err
	}

	return Open(file)
}

// NumPages returns the number of pages in the database
func (f *File) NumPages() int { return int(f.Header.Size) }

// PageSize returns the database page size in bytes
func (f *File) PageSize() int { return int(f.Header.PageSize) }

// Encoding returns the text encoding for this database
func (f *File) Encoding() TextEncoding { return f.Header.TextEncoding }

// Version returns the sqlite version number used to create this database
func (f *File) Version() int { return int(f.Header.LibraryVersion) }

// Close closes the underlying file handle
func (f *File) Close() error { return f.closer.Close() }

// Schema returns a list of all tables and indexes found in the file.
// It parses sqlite_schema table, found at database page 1.
//
// see: https://www.sqlite.org/fileformat.html#storage_of_the_sql_database_schema
func (f *File) Schema() (_ []*Object, err error) {
	var tree = NewTree(f, f.Pager, 1)
	var schemaTable = NewObject("sqlite_schema", "table", "CREATE TABLE sqlite_schema(type,name,tbl_name,rootpage,sql)", tree)

	var objects []*Object
	err = schemaTable.ForEach(func(record *Record) (err error) {
		var typ, _ = record.AsString(0)
		var name, _ = record.AsString(1)
		var root, _ = record.AsInt(3)
		var sql, _ = record.AsString(4)

		if typ == "table" || typ == "index" {
			objects = append(objects, NewObject(name, typ, sql, NewTree(f, f.Pager, root)))
		}

		return nil
	})

	return objects, err
}

func (f *File) Object(name string) (_ *Object, err error) {
	var objects []*Object
	if objects, err = f.Schema(); err != nil {
		return nil, err
	}

	for _, obj := range objects {
		if obj.Name() == name {
			return obj, nil
		}
	}

	return nil, fmt.Errorf("object with name %q not found", name)
}

func (f *File) ForEach(name string, fn func(*Record) error) (err error) {
	var table *Object
	if table, err = f.Object(name); err != nil {
		return err
	}

	return table.ForEach(fn)
}
