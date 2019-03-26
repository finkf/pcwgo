package db

import (
	"database/sql"
	"testing"

	"github.com/finkf/pcwgo/db/sqlite"
)

func withTableSessions(f func(*sql.DB)) {
	sqlite.With("sessions.sqlite", func(db *sql.DB) {
		if err := CreateTableUsers(db); err != nil {
			panic(err)
		}
		if err := CreateTableSessions(db); err != nil {
			panic(err)
		}
		f(db)
	})
}

func TestInsertSession(t *testing.T) {
	withTableSessions(func(db *sql.DB) {
		u, err := InsertUser(db, User{
			Name:      "test",
			Email:     "test@example.com",
			Institute: "test institute",
		})
		if err != nil {
			t.Fatalf("got error: %v", err)
		}
		s, err := InsertSession(db, u)
		if err != nil {
			t.Fatalf("got error: %v", err)
		}
		if s.User != u {
			t.Fatalf("expected %v; got %v", u, s.User)
		}
		if len(s.Auth) != IDLength {
			t.Fatalf("invalid session Auth: %s", s.Auth)
		}
		got, found, err := FindSessionByID(db, s.Auth)
		if err != nil {
			t.Fatalf("got error: %v", err)
		}
		if !found {
			t.Fatalf("cannot find session Auth: %s", s.Auth)
		}
		if got != s {
			t.Fatalf("expected %v; got %v", s, got)
		}
		if err := DeleteSessionByUserID(db, u.ID); err != nil {
			t.Fatalf("got error: %v", err)
		}
	})
}
