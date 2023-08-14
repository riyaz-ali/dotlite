package dotlite

// Object represents either a table or an index stored in the database file
type Object struct {
	name string // name of the object
	sql  string // raw sql to containing the object's schema
	tree *Tree  // tree holding the object
}

func NewObject(name, sql string, tree *Tree) *Object {
	return &Object{name: name, sql: sql, tree: tree}
}

// Name returns the table's name
func (obj *Object) Name() string { return obj.name }

// SQL returns the table's raw sql schema
func (obj *Object) SQL() string { return obj.sql }

// ForEach iterates over each row in the table in order, invoking callback.
func (obj *Object) ForEach(fn func(*Record) error) error {
	return obj.tree.Walk(func(cell *Cell) (err error) {
		var rec *Record
		if rec, err = NewRecord(obj.tree.file.Encoding(), cell); err != nil {
			return err
		}

		return fn(rec)
	})
}
