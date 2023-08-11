package dotlite

import (
	"encoding/binary"
	"fmt"
	"strings"
)

// Affinity of a column is the recommended type for data stored in that column.
// see: https://www.sqlite.org/datatype3.html#affinity
type Affinity int

const (
	_ = iota
	TEXT
	NUMERIC
	INTEGER
	REAL
	BLOB
)

// Column represents meta-information about a given column in a given table.
type Column struct {
	Name     string   // column's name defined by the user
	Type     string   // user defined type for the column
	Affinity Affinity // assigned type affinity for the column
}

// Table represents an ordinary (w/ ROWID) table defined in the sqlite database file
type Table struct {
	tree    *Tree
	name    string
	columns []*Column

	Options struct {
		WithoutRowid bool
		Strict       bool
	}
}

func parseTable(name string, tree *Tree, sql string) (_ *Table, err error) {
	sql = strings.ReplaceAll(sql, "\n", "") // remove all new line characters
	sql = strings.TrimSpace(sql)            // remove all spaces

	var columns []*Column
	{
		var fbrack, lbrack = strings.Index(sql, "(") + 1, strings.LastIndex(sql, ")")

		var def = strings.TrimSpace(sql[fbrack:lbrack])
		var parts = strings.Split(def, ",")

		for _, part := range parts {
			var s = strings.Split(part, " ")
			var c = &Column{Name: strings.TrimSpace(s[0])}

			if len(s) >= 2 {
				c.Type = strings.TrimSpace(s[1])
			}

			switch {
			case strings.Contains(c.Type, "INT"):
				c.Affinity = INTEGER
			case strings.Contains(c.Type, "CHAR") || strings.Contains(c.Type, "CLOB") || strings.Contains(c.Type, "TEXT"):
				c.Affinity = TEXT
			case strings.Contains(c.Type, "REAL") || strings.Contains(c.Type, "FLOA") || strings.Contains(c.Type, "DOUB"):
				c.Affinity = REAL
			case strings.Contains(c.Type, "BLOB") || c.Type == "":
				c.Affinity = BLOB
			default:
				c.Affinity = NUMERIC
			}

			columns = append(columns, c)
		}
	}

	return NewTable(name, tree, columns), nil
}

func NewTable(name string, tree *Tree, cols []*Column) *Table {
	return &Table{name: name, tree: tree, columns: cols}
}

// Name returns the table's name
func (table *Table) Name() string { return table.name }

// Columns return a list of all associated columns for the table
func (table *Table) Columns() []*Column { return table.columns }

// ForEach iterates over each row in the table in order, invoking callback.
func (table *Table) ForEach(fn func([]any) error) error {
	return table.tree.Walk(func(cell *Cell) (err error) {
		var types []int
		{ // read record header and determine serial types of all contained values
			var v int64 // reusable varint holder

			var n = cell.Len()
			if v, err = Varint(cell); err != nil {
				return err
			}

			var headerSize = int(v) - (n - cell.Len())

			for i := 0; i < headerSize; {
				n = cell.Len()
				if v, err = Varint(cell); err != nil {
					return err
				}
				i += n - cell.Len()
				types = append(types, int(v))
			}
		}

		var values = make([]any, 1, len(types)+1)
		values[0] = cell.Rowid

		for _, t := range types {
			switch t {
			case 0x01: // 8-bit twos-complement integer
				var data int8
				if err = binary.Read(cell, binary.BigEndian, &data); err != nil {
					return err
				}
				values = append(values, int64(data))

			case 0x02: // 16-bit twos-complement integer
				var data int16
				if err = binary.Read(cell, binary.BigEndian, &data); err != nil {
					return err
				}
				values = append(values, int64(data))

			case 0x03: // 24-bit twos-complement integer
				var bs = make([]byte, 4)
				if n, _ := cell.Read(bs[1:]); n != 3 {
					return fmt.Errorf("failed to decode 24-bit integer value")
				}

				if bs[1]&0x80 > 0 {
					bs[0] = 0xff
				}
				values = append(values, int64(binary.BigEndian.Uint32(bs)))

			case 0x04: // 32-bit twos-complement integer
				var data int32
				if err = binary.Read(cell, binary.BigEndian, &data); err != nil {
					return err
				}
				values = append(values, int64(data))

			case 0x05: // 48-bit twos-complement integer
				var bs = make([]byte, 8)
				if n, _ := cell.Read(bs[2:]); n != 6 {
					return fmt.Errorf("failed to decode 48-bit integer value")
				}

				if bs[2]&0x80 > 0 {
					bs[0] = 0xff
				}
				values = append(values, int64(binary.BigEndian.Uint64(bs)))

			case 0x06: // 64-bit twos-complement integer
				var data int64
				if err = binary.Read(cell, binary.BigEndian, &data); err != nil {
					return err
				}
				values = append(values, data)

			case 0x07: // IEEE 754-2008 64-bit floating point number
				var data float64
				if err = binary.Read(cell, binary.BigEndian, &data); err != nil {
					return err
				}
				values = append(values, data)

			case 0x08: // Value is the integer 0.
				values = append(values, int64(0))

			case 0x09: // Value is the integer 1.
				values = append(values, int64(1))

			default:
				// if the type is BLOB
				if t >= 12 && t%2 == 0 {
					var buf = cell.Region(int64((t - 12) / 2))
					values = append(values, buf)
				}

				// if the type is TEXT
				if t >= 13 && t%2 != 0 {
					var buf = cell.Region(int64((t - 13) / 2))

					if table.tree.file.Header.TextEncoding == UTF8 {
						var s = string(buf)
						if idx := strings.Index(s, "\x00"); idx >= 0 {
							s = s[:idx]
						}

						values = append(values, s)
					} else {
						return fmt.Errorf("UTF-16 is not supported")
					}
				}
			}
		}

		return fn(values)
	})
}
