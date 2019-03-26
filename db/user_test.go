package db

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/finkf/pcwgo/db/sqlite"
)

func newTestUser(t *testing.T, db DB, id int) *User {
	if err := CreateTableUsers(db); err != nil {
		t.Fatalf("got error: %v", err)
	}
	user := User{
		Name:      fmt.Sprintf("user_name_%d", id),
		Email:     fmt.Sprintf("user_email_%d", id),
		Institute: fmt.Sprintf("user_institute_%d", id),
		Admin:     (id % 2) != 0, // odd ids are admins
	}
	err := InsertUser(db, &user)
	if err != nil {
		t.Fatalf("got error: %v", err)
	}
	return &user
}

func withTableUsers(t *testing.T, f func(*sql.DB)) {
	sqlite.With("users.sqlite", func(db *sql.DB) {
		if err := CreateTableUsers(db); err != nil {
			t.Fatalf("got error: %v", err)
		}
		f(db)
	})
}

func withTestUser(t *testing.T, u *User, f func(*sql.DB)) {
	withTableUsers(t, func(db *sql.DB) {
		var err error
		err = InsertUser(db, u)
		if err != nil {
			t.Fatalf("got error: %v", err)
		}
		f(db)
	})
}

func TestInsertUser(t *testing.T) {
	withTableUsers(t, func(db *sql.DB) {
		want := User{Name: "test", Email: "test@example.com"}
		got := want
		err := InsertUser(db, &got)
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
		got, found, err := FindUserByID(db, want.ID)
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
		_, found, err := FindUserByID(db, want.ID)
		if err != nil {
			t.Fatalf("got error: %v", err)
		}
		if found {
			t.Fatalf("cannot delete user: %s", want)
		}
	})
}
