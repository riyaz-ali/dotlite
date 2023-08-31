# 🗒️ dotlite

**`dotlite`** is a [`sqlite3` file reader](http://sqlite.org/fileformat.html) written in pure Golang 🍋 

*But why 🤔?* Well, just for fun 🤪 There aren't many practical scenarios where you should need this. You're better off using `sqlite3` directly.
This package exists just an exercise to implement the file format and understand the internals of how the [`btree`](https://en.wikipedia.org/wiki/B-tree) is serialized and stored.

## Usage

Pull the package using `go get -u go.riyazali.net/dotlite`.

Then, to iterate over entries from a table, do:

```golang
// using Album table from testdata/chinook.db
// with schema: CREATE TABLE Album (AlbumId INTEGER NOT NULL, Title TEXT, ArtistId INTEGER);
var file, _ = dotlite.OpenFile("testdata/chinook.db")
defer file.Close()

var err = file.ForEach("Album", func(rec *Record) error {
  log.Printf("record(%p):\n", record)
  for i := 0; i < record.NumValues(); i++ {
    var val any
    if val, err = record.ValueAt(i); err != nil {
      return err
    }

    log.Printf("\tval(%d): %+v\n", i, val)
  }

  return nil
})
```

You can use the same pattern to iterate over entries in an index or a [`WITHOUT ROWID`](https://www.sqlite.org/withoutrowid.html) table as well.

### Wishes (that may never get fulfilled)

- [ ] Support for other page types including `freelist` and `ptrmap`
- [ ] Support for [rollback journal](https://www.sqlite.org/fileformat.html#the_rollback_journal)
- [ ] Support for [Write-Ahead Log](https://www.sqlite.org/fileformat.html#the_write_ahead_log)

## Credits

This package was inspired by (and heavily borrows from) [`github.com/go-sqlite/sqlite3`](https://github.com/go-sqlite/sqlite3). Thanks!

------------------------------------

MIT License Copyright (c) 2023 Riyaz Ali. Refer to [LICENSE](./LICENSE) for full text.
