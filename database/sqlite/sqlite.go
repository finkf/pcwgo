package sqlite

import (
	"database/sql"
	"os"

	_ "rsc.io/sqlite" // register "sqlite3"
)

// With create a temporary sqlite database and executes
// the given function with this new databse.
// After the call to With, the database file is removed.
// With panics if the database could not be created.
func With(file string, f func(*sql.DB)) {
	db, err := sql.Open("sqlite3", file)
	if err != nil {
		panic(err)
	}
	defer os.Remove(file)
	defer db.Close()
	f(db)
}
