package user

import (
	"database/sql"
	"testing"

	"github.com/finkf/pcwgo/database/sqlite"
)

func withTableUsers(t *testing.T, f func(*sql.DB)) {
	sqlite.With("users.sqlite", func(db *sql.DB) {
		if err := CreateTable(db); err != nil {
			t.Fatalf("got error: %v", err)
		}
		f(db)
	})
}

func withTestUser(t *testing.T, u *User, f func(*sql.DB)) {
	withTableUsers(t, func(db *sql.DB) {
		var err error
		*u, err = New(db, *u)
		if err != nil {
			t.Fatalf("got error: %v", err)
		}
		f(db)
	})
}

func TestNewUser(t *testing.T) {
	withTableUsers(t, func(db *sql.DB) {
		want := User{Name: "test", Email: "test@example.com"}
		got, err := New(db, want)
		if err != nil {
			t.Fatalf("cannot create user: %v", err)
		}
		want.ID = got.ID // ignore ID
		if got != want {
			t.Fatalf("expected user: %s; got %s", want, got)
		}
	})
}

func TestUserPassword(t *testing.T) {
	want := User{Name: "test", Email: "test@example.com"}
	withTestUser(t, &want, func(db *sql.DB) {
		if err := SetUserPassword(db, want, "test-passwd"); err != nil {
			t.Fatalf("got error: %v", err)
		}
		if err := AuthenticateUser(db, want, "test-passwd"); err != nil {
			t.Fatalf("got error: %v", err)
		}
		if err := AuthenticateUser(db, want, "wrong-passwd"); err == nil {
			t.Fatalf("authentification does not work")
		}
	})
}

func TestUpdateUser(t *testing.T) {
	want := User{Name: "test", Email: "test@example.com"}
	withTestUser(t, &want, func(db *sql.DB) {
		want.Institute = "test institute"
		if err := UpdateUser(db, want); err != nil {
			t.Fatalf("got error: %v", err)
		}
		got, found, err := FindByID(db, want.ID)
		if err != nil {
			t.Fatalf("got error: %v", err)
		}
		if !found {
			t.Fatalf("cannot find user: %s", want)
		}
		if got != want {
			t.Fatalf("expected user: %s; got %s", want, got)
		}
	})
}

func TestDeleteUser(t *testing.T) {
	want := User{Name: "test", Email: "test@example.com"}
	withTestUser(t, &want, func(db *sql.DB) {
		if err := DeleteUserByID(db, want.ID); err != nil {
			t.Fatalf("got error: %v", err)
		}
		_, found, err := FindByID(db, want.ID)
		if err != nil {
			t.Fatalf("got error: %v", err)
		}
		if found {
			t.Fatalf("cannot delete user: %s", want)
		}
	})
}
