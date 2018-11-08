package project

import (
	"database/sql"
	"reflect"
	"strconv"
	"testing"

	"github.com/finkf/pcwgo/database/sqlite"
	"github.com/finkf/pcwgo/database/user"
)

var (
	u1, u2, u3 user.User
	p1, p2, p3 Project
)

func mustNewUser(db *sql.DB, u user.User) user.User {
	nu, err := user.New(db, u)
	if err != nil {
		panic(err)
	}
	return nu
}

func mustNewProject(db *sql.DB, p Project) Project {
	np, err := New(db, p)
	if err != nil {
		panic(err)
	}
	return np
}

func withProjectDB(f func(*sql.DB)) {
	sqlite.With("projects.sqlite", func(db *sql.DB) {
		if err := user.CreateTable(db); err != nil {
			panic(err)
		}
		if err := CreateTable(db); err != nil {
			panic(err)
		}
		u1 = mustNewUser(db, user.User{Name: "test1", Email: "email1"})
		u2 = mustNewUser(db, user.User{Name: "test2", Email: "email2"})
		u3 = mustNewUser(db, user.User{Name: "test3", Email: "email3"})
		p1 = mustNewProject(db, Project{Pages: 1, Origin: 1, Owner: u1})
		p2 = mustNewProject(db, Project{Pages: 2, Origin: 2, Owner: u1})
		p3 = mustNewProject(db, Project{Pages: 3, Origin: 3, Owner: u2})
		f(db)
	})
}

func TestFindByID(t *testing.T) {
	withProjectDB(func(db *sql.DB) {
		tests := []struct {
			id    int64
			want  Project
			found bool
		}{
			{p1.ID, p1, true},
			{p2.ID, p2, true},
			{p3.ID, p3, true},
			{p3.ID + 1, Project{}, false},
		}
		for _, tc := range tests {
			t.Run(strconv.Itoa(int(tc.id)), func(t *testing.T) {
				got, found, err := FindByID(db, tc.id)
				if err != nil {
					t.Fatalf("got error: %v", err)
				}
				if found != tc.found {
					t.Fatalf("expected found: %t; got %t", tc.found, found)
				}
				if got.String() != tc.want.String() {
					t.Fatalf("epected project: %s; got %s", got, tc.want)
				}
			})
		}
	})
}

func TestFindByUser(t *testing.T) {
	withProjectDB(func(db *sql.DB) {
		tests := []struct {
			u    user.User
			want []Project
		}{
			{u1, []Project{p1, p2}},
			{u2, []Project{p3}},
			{u3, nil},
		}
		for _, tc := range tests {
			t.Run(strconv.Itoa(int(tc.u.ID)), func(t *testing.T) {
				ps, err := FindByUser(db, tc.u)
				if err != nil {
					t.Fatalf("got error: %s", err)
				}
				if !reflect.DeepEqual(ps, tc.want) {
					t.Fatalf("expected projects: %v; got %v", tc.want, ps)
				}
			})
		}
	})
}
