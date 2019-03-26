package sqlite

import (
	"database/sql"
	"os"
	"testing"
)

func TestWith(t *testing.T) {
	file := "with.sqlite"
	With(file, func(db *sql.DB) {
		if _, err := db.Exec("CREATE TABLE test(ID INTEGER);"); err != nil {
			t.Fatalf("got error: %v", err)
		}
		if _, err := db.Exec("INSERT INTO test(ID) values(8);"); err != nil {
			t.Fatalf("got error: %v", err)
		}
		rows, err := db.Query("SELECT * FROM test;")
		if err != nil {
			t.Fatalf("got error: %v", err)

		}
		defer rows.Close()
		if !rows.Next() {
			t.Fatalf("cannot read databse")
		}
	})
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Fatalf("%s was not removed: %v", file, err)
	}
}
