package db

import (
	"database/sql"
	"reflect"
	"strconv"
	"testing"

	"github.com/finkf/pcwgo/api"
	"github.com/finkf/pcwgo/db/sqlite"
)

var (
	u1, u2, u3 *api.User
	p1, p2, p3 *Project
)

func newTestProject(t *testing.T, db DB, id int, book *Book, user *api.User) *Project {
	if user == nil {
		user = newTestUser(t, db, id)
	}
	project := &Project{
		Owner: *user,
		Book:  *book,
		Pages: 1,
	}
	if err := InsertProject(db, project); err != nil {
		t.Fatalf("got error: %v", err)
	}
	return project
}

func withProjectDB(t *testing.T, f func(*sql.DB)) {
	sqlite.With("projects.sqlite", func(db *sql.DB) {
		if err := CreateTableJobs(db); err != nil {
			t.Fatalf("got error: %v", err)
		}
		if err := CreateTableUsers(db); err != nil {
			t.Fatalf("got error: %v", err)
		}
		if err := CreateTableProjects(db); err != nil {
			t.Fatalf("got error: %v", err)
		}
		if err := CreateTableBooks(db); err != nil {
			t.Fatalf("got error: %v", err)
		}
		b := newTestBook(t, db, 1)
		u1 = newTestUser(t, db, 1)
		u2 = newTestUser(t, db, 2)
		u3 = newTestUser(t, db, 3)
		p1 = newTestProject(t, db, 1, b, u1)
		p2 = newTestProject(t, db, 2, b, u1)
		p3 = newTestProject(t, db, 3, b, u2)
		f(db)
	})
}

func TestFindProjectByID(t *testing.T) {
	withProjectDB(t, func(db *sql.DB) {
		tests := []struct {
			id    int
			want  *Project
			found bool
		}{
			{p1.ProjectID, p1, true},
			{p2.ProjectID, p2, true},
			{p3.ProjectID, p3, true},
			{p3.ProjectID + 1, &Project{}, false},
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
					t.Fatalf("epected project: %s; got %s",
						got.String(), tc.want.String())
				}
			})
		}
	})
}

func TestFindProjectByUser(t *testing.T) {
	withProjectDB(t, func(db *sql.DB) {
		tests := []struct {
			u    *api.User
			want []Project
		}{
			{u1, []Project{*p1, *p2}},
			{u2, []Project{*p3}},
			{u3, nil},
		}
		for _, tc := range tests {
			t.Run(strconv.Itoa(int(tc.u.ID)), func(t *testing.T) {
				ps, err := FindProjectByOwner(db, tc.u.ID)
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
