package dotlite

import (
	"encoding/binary"
	"fmt"
	"io"
	"strings"
)

// RecordVal holds type and offset information about a single value contained in the record
type RecordVal struct {
	Type   int   // serial type of the value
	Offset int64 // offset from start of cell region
}

// Record represents an individual record saved in btree in the Record Format (https://www.sqlite.org/fileformat.html#record_format)
type Record struct {
	encoding TextEncoding // supported text encoding for this file
	cell     *Cell        // cell backing this record
	values   []RecordVal  // slice of meta information about the values contained within the record
}

// NewRecord creates a new record from the given cell
func NewRecord(enc TextEncoding, cell *Cell) (_ *Record, err error) {
	// read record header and determine serial types of all contained values
	var values []RecordVal

	var v int64 // reusable varint holder
	var n = cell.Len()
	if v, err = Varint(cell); err != nil {
		return nil, err
	}

	var headerSize = int(v) - (n - cell.Len())
	var body = v // offset where body starts

	for i := 0; i < headerSize; {
		n = cell.Len()
		if v, err = Varint(cell); err != nil {
			return nil, err
		}
		i += n - cell.Len()

		values = append(values, RecordVal{Type: int(v), Offset: body})
		body += typeSize(v)
	}

	return &Record{encoding: enc, cell: cell, values: values}, nil
}

// Encoding returns the text encoding used by the record
func (rec *Record) Encoding() TextEncoding { return rec.encoding }

// NumValues return the number of values contained within this record
func (rec *Record) NumValues() int { return len(rec.values) }

// ValueAt returns the value at position c as a golang primitive type
func (rec *Record) ValueAt(c int) (any, error) {
	if c > rec.NumValues() {
		return nil, fmt.Errorf("column index %d out of range", c)
	}

	var cell, val = rec.cell, rec.values[c]

	pos, _ := cell.Seek(0, io.SeekCurrent)
	defer cell.Seek(pos, io.SeekStart) // restore to original position

	_, _ = cell.Seek(val.Offset, io.SeekStart) // seek to where the content for c starts

	switch val.Type {
	case 0x00: // sqlite NULL
		return nil, nil

	case 0x01: // 8-bit twos-complement integer
		var data int8
		if err := binary.Read(cell, binary.BigEndian, &data); err != nil {
			return nil, err
		}
		return int64(data), nil

	case 0x02: // 16-bit twos-complement integer
		var data int16
		if err := binary.Read(cell, binary.BigEndian, &data); err != nil {
			return nil, err
		}
		return int64(data), nil

	case 0x03: // 24-bit twos-complement integer
		var bs = make([]byte, 4)
		if n, _ := cell.Read(bs[1:]); n != 3 {
			return nil, fmt.Errorf("failed to decode 24-bit integer value")
		}

		if bs[1]&0x80 > 0 {
			bs[0] = 0xff
		}
		return int64(binary.BigEndian.Uint32(bs)), nil

	case 0x04: // 32-bit twos-complement integer
		var data int32
		if err := binary.Read(cell, binary.BigEndian, &data); err != nil {
			return nil, err
		}
		return int64(data), nil

	case 0x05: // 48-bit twos-complement integer
		var bs = make([]byte, 8)
		if n, _ := cell.Read(bs[2:]); n != 6 {
			return nil, fmt.Errorf("failed to decode 48-bit integer value")
		}

		if bs[2]&0x80 > 0 {
			bs[0] = 0xff
		}
		return int64(binary.BigEndian.Uint64(bs)), nil

	case 0x06: // 64-bit twos-complement integer
		var data int64
		if err := binary.Read(cell, binary.BigEndian, &data); err != nil {
			return nil, err
		}
		return data, nil

	case 0x07: // IEEE 754-2008 64-bit floating point number
		var data float64
		if err := binary.Read(cell, binary.BigEndian, &data); err != nil {
			return nil, err
		}
		return data, nil

	case 0x08: // Value is the integer 0.
		return int64(0), nil

	case 0x09: // Value is the integer 1.
		return int64(1), nil

	default:
		// if the type is BLOB
		if t := val.Type; t >= 12 && t%2 == 0 {
			var buf = make([]byte, (t-12)/2)
			if _, err := io.ReadFull(cell, buf); err != nil {
				return nil, err
			}

			return buf, nil
		}

		// if the type is TEXT
		if t := val.Type; t >= 13 && t%2 != 0 {
			var buf = make([]byte, (t-13)/2)
			if _, err := io.ReadFull(cell, buf); err != nil {
				return nil, err
			}

			if rec.encoding == UTF8 {
				var s = string(buf)
				if idx := strings.Index(s, "\x00"); idx >= 0 {
					s = s[:idx]
				}

				return s, nil
			} else {
				return nil, fmt.Errorf("UTF-16 is not supported")
			}
		}
	}

	return nil, fmt.Errorf("unknown value type %d", rec.values[c].Type)
}

func (rec *Record) AsInt(c int) (_ int, err error) {
	var v int64
	if v, err = rec.AsInt64(c); err != nil {
		return 0, err
	}
	return int(v), nil
}

func (rec *Record) AsInt64(c int) (_ int64, err error) {
	var v any
	if v, err = rec.ValueAt(c); err != nil {
		return 0, err
	} else if n, ok := v.(float64); ok {
		return int64(n), nil
	}
	n, _ := v.(int64)
	return n, nil
}

func (rec *Record) AsFloat64(c int) (_ float64, err error) {
	var v any
	if v, err = rec.ValueAt(c); err != nil {
		return 0, err
	}
	n, _ := v.(float64)
	return n, nil
}

func (rec *Record) AsString(c int) (_ string, err error) {
	var v any
	if v, err = rec.ValueAt(c); err != nil {
		return "", err
	}

	s, _ := v.(string)
	return s, nil
}

func (rec *Record) AsBlob(c int) (_ []byte, err error) {
	var v any
	if v, err = rec.ValueAt(c); err != nil {
		return []byte(nil), err
	}

	b, _ := v.([]byte)
	return b, nil
}
