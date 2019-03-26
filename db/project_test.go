package db

import (
	"database/sql"
	"reflect"
	"strconv"
	"testing"

	"github.com/finkf/pcwgo/db/sqlite"
)

var (
	u1, u2, u3 User
	p1, p2, p3 *Project
)

func newTestProject(t *testing.T, db DB, id int) *Project {
	if err := CreateTableBooks(db); err != nil {
		t.Fatalf("got error: %v", err)
	}
	u := newTestUser(t, db, id)
	project := &Project{
		Owner:  u,
		Origin: int64(id),
		Pages:  1,
	}

	if err := InsertProject(db, project); err != nil {
		t.Fatalf("got error: %v", err)
	}
	return project
}

func mustNewUser(db *sql.DB, u User) User {
	err := InsertUser(db, &u)
	if err != nil {
		panic(err)
	}
	return u
}

func mustNewProject(db *sql.DB, p Project) *Project {
	err := InsertProject(db, &p)
	if err != nil {
		panic(err)
	}
	return &p
}

func withProjectDB(f func(*sql.DB)) {
	sqlite.With("projects.sqlite", func(db *sql.DB) {
		if err := CreateTableUsers(db); err != nil {
			panic(err)
		}
		if err := CreateTableProjects(db); err != nil {
			panic(err)
		}
		u1 = mustNewUser(db, User{Name: "test1", Email: "email1"})
		u2 = mustNewUser(db, User{Name: "test2", Email: "email2"})
		u3 = mustNewUser(db, User{Name: "test3", Email: "email3"})
		p1 = mustNewProject(db, Project{Pages: 1, Origin: 1, Owner: u1})
		p2 = mustNewProject(db, Project{Pages: 2, Origin: 2, Owner: u1})
		p3 = mustNewProject(db, Project{Pages: 3, Origin: 3, Owner: u2})
		f(db)
	})
}

func TestFindProjectByID(t *testing.T) {
	withProjectDB(func(db *sql.DB) {
		tests := []struct {
			id    int64
			want  *Project
			found bool
		}{
			{p1.ID, p1, true},
			{p2.ID, p2, true},
			{p3.ID, p3, true},
			{p3.ID + 1, &Project{}, false},
		}
		for _, tc := range tests {
			t.Run(strconv.Itoa(int(tc.id)), func(t *testing.T) {
				got, found, err := FindProjectByID(db, tc.id)
				if err != nil {
					t.Fatalf("got error: %v", err)
				}
				if found != tc.found {
					t.Fatalf("expected found: %t; got %t", tc.found, found)
				}
				if tc.found && got.String() != tc.want.String() {
					t.Fatalf("epected project: %s; got %s", got, tc.want)
				}
			})
		}
	})
}

func TestFindByUser(t *testing.T) {
	withProjectDB(func(db *sql.DB) {
		tests := []struct {
			u    User
			want []Project
		}{
			{u1, []Project{*p1, *p2}},
			{u2, []Project{*p3}},
			{u3, nil},
		}
		for _, tc := range tests {
			t.Run(strconv.Itoa(int(tc.u.ID)), func(t *testing.T) {
				ps, err := FindProjectByUser(db, tc.u)
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
