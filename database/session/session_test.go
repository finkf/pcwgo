package session

import (
	"database/sql"
	"testing"

	"github.com/finkf/pcwgo/database/sqlite"
	"github.com/finkf/pcwgo/database/user"
)

func withTableSessions(f func(*sql.DB)) {
	sqlite.With("sessions.sqlite", func(db *sql.DB) {
		if err := user.CreateTable(db); err != nil {
			panic(err)
		}
		if err := CreateTable(db); err != nil {
			panic(err)
		}
		f(db)
	})
}

func TestNewSession(t *testing.T) {
	withTableSessions(func(db *sql.DB) {
		u, err := user.New(db, user.User{
			Name:      "test",
			Email:     "test@example.com",
			Institute: "test institute",
		})
		if err != nil {
			t.Fatalf("got error: %v", err)
		}
		s, err := New(db, u)
		if err != nil {
			t.Fatalf("got error: %v", err)
		}
		if s.User != u {
			t.Fatalf("expected %v; got %v", u, s.User)
		}
		if len(s.Auth) != IDLength {
			t.Fatalf("invalid session Auth: %s", s.Auth)
		}
		got, found, err := FindByID(db, s.Auth)
		if err != nil {
			t.Fatalf("got error: %v", err)
		}
		if !found {
			t.Fatalf("cannot find session Auth: %s", s.Auth)
		}
		got.Expires = s.Expires
		if got != s {
			t.Fatalf("expected %v; got %v", s, got)
		}
	})
}
