package db

import (
	"database/sql"
	"testing"

	"github.com/finkf/pcwgo/db/sqlite"
)

func withJobsTable(t *testing.T, f func(DB)) {
	sqlite.With("projects.sqlite", func(db *sql.DB) {
		if err := CreateTableJobs(db); err != nil {
			t.Fatalf("cannot create jobs table: %v", err)
		}
		f(db)
	})
}

func TestNewJobID(t *testing.T) {
	withJobsTable(t, func(db DB) {
		id, err := NewJob(db, 1)
		if err != nil {
			t.Fatalf("got error: %v", err)
		}
		if id == 0 {
			t.Fatalf("id = %d", id)
		}
	})
}

func TestFindJobByID(t *testing.T) {
	withJobsTable(t, func(db DB) {
		id, err := NewJob(db, 1)
		if err != nil {
			t.Fatalf("got error: %v", err)
		}
		job, ok, err := FindJobByID(db, id)
		if err != nil {
			t.Fatalf("got error: %v", err)
		}
		if !ok {
			t.Fatalf("cannot find job id: %d", id)
		}
		if job.BookID != 1 || job.JobID != id {
			t.Fatalf("invalid job: %v", job)
		}
	})
}

func TestSetJobStatus(t *testing.T) {
	withJobsTable(t, func(db DB) {
		id, err := NewJob(db, 1)
		if err != nil {
			t.Fatalf("got error: %v", err)
		}
		if err := SetJobStatus(db, id, StatusIDDone); err != nil {
			t.Fatalf("got error: %v", err)
		}
		job, ok, err := FindJobByID(db, id)
		if err != nil {
			t.Fatalf("got error: %v", err)
		}
		if !ok {
			t.Fatalf("cannot find job id: %d", id)
		}
		if job.StatusID != StatusIDDone || job.StatusName != StatusDone {
			t.Fatalf("invalid job: %v", job)
		}
	})
}

func TestDeleteJobByID(t *testing.T) {
	withJobsTable(t, func(db DB) {
		id, err := NewJob(db, 1)
		if err != nil {
			t.Fatalf("got error: %v", err)
		}
		if err := DeleteJobByID(db, id); err != nil {
			t.Fatalf("got error: %v", err)
		}
		_, ok, err := FindJobByID(db, id)
		if err != nil {
			t.Fatalf("got error: %v", err)
		}
		if ok {
			t.Fatalf("should not able to find job id: %d", id)
		}
	})
}
