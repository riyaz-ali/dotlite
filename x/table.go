// Package x provides common extension functionality, namely, a helper to parse table schema and more.
package x

import (
	"errors"
	"unsafe"
)

// #include "sql3parse_table.h"
import "C"

// Column represents an individual column in the table
type Column struct {
	Name           string `json:"name"`                      // column's name
	Type           string `json:"type,omitempty"`            // column's defined data type
	Length         string `json:"length,omitempty"`          // (optional) length of the specified data type (ex. length for varchar fields)
	Comment        string `json:"comment,omitempty"`         // any user provided comment
	ConstraintName string `json:"constraint_name,omitempty"` // name of external constraint

	// meta about the column
	Properties struct {
		PrimaryKey    bool `json:"primary_key"`    // is there an inline primary key constraint defined?
		AutoIncrement bool `json:"auto_increment"` // is this an auto-incrementing column?
		NotNull       bool `json:"not_null"`       // is there a NOT NULL constraint?
		Unique        bool `json:"unique"`         // is there an inline UNIQUE constraint defined?
	} `json:"properties"`

	PrimaryKeyOrder    Ordering   `json:"primary_key_order"`
	PrimaryKeyConflict OnConflict `json:"primary_key_conflict"` // conflict resolution used when there's a conflict in PK constraint
	NotNullConflict    OnConflict `json:"not_null_conflict"`    // conflict resolution used when there's a conflict in NOT NULL constraint
	UniqueConflict     OnConflict `json:"unique_conflict"`      // conflict resolution used when there's a conflict in Unique constraint

	CheckExpr   string `json:"check_expr,omitempty"`   // any user defined CHECK(...) expression
	DefaultExpr string `json:"default_expr,omitempty"` // DEFAULT(...) expression used to generate a default value
	Collate     string `json:"collate,omitempty"`      // collation sequence used by this column

	ForeignKey *ForeignKey `json:"foreign_key,omitempty"` // any inline Foreign Key constraint defined on the column
}

// TableConstraint represents a constraint defined on the table itself
type TableConstraint struct {
	Name       string         `json:"name"`
	Type       ConstraintType `json:"type"`
	CheckExpr  string         `json:"check_expr,omitempty"`
	OnConflict OnConflict     `json:"on_conflict"`

	ForeignKey     *ForeignKey      `json:"foreign_key,omitempty"`
	IndexedColumns []*IndexedColumn `json:"indexed_columns,omitempty"`
}

// IndexedColumn represents a column used in a unique or primary key constraint / index
type IndexedColumn struct {
	Name    string   `json:"name"`    // name of the column
	Collate string   `json:"collate"` // collation function used
	Order   Ordering `json:"order"`   // order of the column in the index
}

// ForeignKey represents a foreign key constraint defined on either the column or the table
type ForeignKey struct {
	Columns           []string             `json:"columns"`            // columns in this table that are part of the FK
	ReferencedTable   string               `json:"referenced_table"`   // the referenced table's name
	ReferencedColumns []string             `json:"referenced_columns"` // columns in the referenced table
	Match             string               `json:"match"`
	OnDelete          ForeignKeyAction     `json:"on_delete"`  // ON DELETE action of the foreign key
	OnUpdate          ForeignKeyAction     `json:"on_update"`  // ON UPDATE action of the foreign key
	Deferrable        ForeignKeyDeferrable `json:"deferrable"` // Deferrable status of the foreign key constraint
}

// Table represents a sqlite3 table parsed from the provided schema
type Table struct {
	Schema  string `json:"schema,omitempty"`  // table's (optional) schema
	Name    string `json:"name"`              // table's name
	Comment string `json:"comment,omitempty"` // (optional) any user-provided comment

	// meta about the table itself
	Properties struct {
		Temporary    bool `json:"is_temporary"`     // is it a temporary table?
		IfNotExists  bool `json:"is_if_not_exists"` // does it have "IF NOT EXISTS" clause
		WithoutRowid bool `json:"is_without_rowid"` // is it a WITHOUT ROWID table?
		Strict       bool `json:"is_strict"`        // is it a STRICT table?
	} `json:"properties"`

	Columns     []*Column          `json:"columns"`     // all columns defined in the table
	Constraints []*TableConstraint `json:"constraints"` // all table-level constraints
}

// ParseSchema parses the given schema and constructs a table instance
func ParseSchema(schema string) (_ *Table, err error) {
	var cstr = C.CString(schema)
	defer C.free(unsafe.Pointer(cstr))

	var ec C.sql3error_code
	var tab *C.sql3table = C.sql3parse_table(cstr, C.size_t(len(schema)), &ec)
	if ec != C.SQL3ERROR_NONE || tab == nil {
		if ec == C.SQL3ERROR_SYNTAX {
			return nil, errors.New("failed to parse schema: syntax error")
		} else if ec == C.SQL3ERROR_UNSUPPORTEDSQL {
			return nil, errors.New("failed to parse schema: unsupported sql")
		}

		return nil, errors.New("failed to parse schema")
	}
	defer C.sql3table_free(tab)

	// create a new tab and fill in with details from the schema
	var table = &Table{Schema: str(C.sql3table_schema(tab)), Name: str(C.sql3table_name(tab)), Comment: str(C.sql3table_comment(tab))}
	table.Properties.Temporary = boolean(C.sql3table_is_temporary(tab))
	table.Properties.Temporary = boolean(C.sql3table_is_temporary(tab))
	table.Properties.IfNotExists = boolean(C.sql3table_is_ifnotexists(tab))
	table.Properties.WithoutRowid = boolean(C.sql3table_is_withoutrowid(tab))
	table.Properties.Strict = boolean(C.sql3table_is_strict(tab))

	// read columns information from the schema
	var columns = make([]*Column, int(C.sql3table_num_columns(tab)))
	for i, n := 0, len(columns); i < n; i++ {
		var col = C.sql3table_get_column(tab, C.size_t(i))
		columns[i] = &Column{
			Name:           str(C.sql3column_name(col)),
			Type:           str(C.sql3column_type(col)),
			Length:         str(C.sql3column_length(col)),
			Comment:        str(C.sql3column_comment(col)),
			ConstraintName: str(C.sql3column_constraint_name(col)),
		}

		columns[i].Properties.PrimaryKey = boolean(C.sql3column_is_primarykey(col))
		columns[i].Properties.AutoIncrement = boolean(C.sql3column_is_autoincrement(col))
		columns[i].Properties.NotNull = boolean(C.sql3column_is_notnull(col))
		columns[i].Properties.Unique = boolean(C.sql3column_is_unique(col))

		columns[i].PrimaryKeyOrder = Ordering(C.sql3column_pk_order(col))
		columns[i].PrimaryKeyConflict = OnConflict(C.sql3column_pk_conflictclause(col))
		columns[i].NotNullConflict = OnConflict(C.sql3column_notnull_conflictclause(col))
		columns[i].UniqueConflict = OnConflict(C.sql3column_unique_conflictclause(col))

		columns[i].CheckExpr = str(C.sql3column_check_expr(col))
		columns[i].DefaultExpr = str(C.sql3column_default_expr(col))
		columns[i].Collate = str(C.sql3column_collate_name(col))

		if fk := C.sql3column_foreignkey_clause(col); fk != nil {
			columns[i].ForeignKey = foreignKey(fk, []string{columns[i].Name})
		}
	}
	table.Columns = columns

	// read all constraints defined on the table-level
	var constraints = make([]*TableConstraint, int(C.sql3table_num_constraints(tab)))
	for i, n := 0, len(constraints); i < n; i++ {
		var cons = C.sql3table_get_constraint(tab, C.size_t(i))
		constraints[i] = &TableConstraint{
			Name:       str(C.sql3table_constraint_name(cons)),
			Type:       ConstraintType(C.sql3table_constraint_type(cons)),
			CheckExpr:  str(C.sql3table_constraint_check_expr(cons)),
			OnConflict: OnConflict(C.sql3table_constraint_conflict_clause(cons)),
		}

		if fk := C.sql3table_constraint_foreignkey_clause(cons); fk != nil {
			var cols = make([]string, int(C.sql3table_constraint_num_fkcolumns(cons)))
			for j := 0; j < len(cols); j++ {
				cols[j] = str(C.sql3table_constraint_get_fkcolumn(cons, C.size_t(j)))
			}

			constraints[i].ForeignKey = foreignKey(fk, cols)
		}

		if idx := int(C.sql3table_constraint_num_idxcolumns(cons)); idx > 0 {
			constraints[i].IndexedColumns = make([]*IndexedColumn, idx)

			for j := 0; j < idx; j++ {
				var idxc = C.sql3table_constraint_get_idxcolumn(cons, C.size_t(j))
				constraints[i].IndexedColumns[j] = &IndexedColumn{
					Name: str(C.sql3idxcolumn_name(idxc)), Collate: str(C.sql3idxcolumn_collate(idxc)), Order: Ordering(C.sql3idxcolumn_order(idxc)),
				}
			}
		}
	}
	table.Constraints = constraints

	return table, nil
}

// converts sql3string to a Golang string
func str(s *C.sql3string) string {
	if s == nil {
		return ""
	}

	var l C.size_t
	var ptr = C.sql3string_ptr(s, &l)
	return C.GoStringN(ptr, C.int(l))
}

// converts C's bool to Golang bool
func boolean(b C.bool) bool { return b == true }

// reads all information about the foreign key
func foreignKey(f *C.sql3foreignkey, cols []string) *ForeignKey {
	var fk = &ForeignKey{
		Columns:         cols,
		ReferencedTable: str(C.sql3foreignkey_table(f)),
		Match:           str(C.sql3foreignkey_match(f)),
		OnDelete:        ForeignKeyAction(C.sql3foreignkey_ondelete_action(f)),
		OnUpdate:        ForeignKeyAction(C.sql3foreignkey_onupdate_action(f)),
		Deferrable:      ForeignKeyDeferrable(C.sql3foreignkey_deferrable(f)),
	}

	fk.ReferencedColumns = make([]string, int(C.sql3foreignkey_num_columns(f)))
	for k := 0; k < len(fk.ReferencedColumns); k++ {
		fk.ReferencedColumns[k] = str(C.sql3foreignkey_get_column(f, C.size_t(k)))
	}

	return fk
}
