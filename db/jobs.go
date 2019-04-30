package db

import (
	"time"

	"github.com/finkf/pcwgo/api"
)

// Status IDs
const (
	StatusIDFailed = iota
	StatusIDRunning
	StatusIDDone
)

// Status names
const (
	StatusFailed  = "failed"
	StatusRunning = "running"
	StatusDone    = "done"
)

// JobsTableName defines the name of the jobs table.
const JobsTableName = "jobs"

const jobsTable = JobsTableName + "(" +
	"JobID INTEGER NOT NULL PRIMARY KEY UNIQUE REFERENCES " + BooksTableName + "(BooksID)," +
	"StatusID INTEGER NOT NULL REFERENCES " + JobsStatusTableName + "(StatusID)," +
	"Timestamp INT(11) NOT NULL" +
	");"

// JobsStatusTableName defines the name of the jobs status table.
const JobsStatusTableName = "jobs_status"

const jobsStatusTable = JobsStatusTableName + "(" +
	"StatusID INTEGER NOT NULL PRIMARY KEY," +
	"StatusName VARCHAR(10) NOT NULL" +
	");"

// CreateTableJobs creates the jobs and jobs status database tables if
// they do not already exist.
func CreateTableJobs(db DB) error {
	_, err := Exec(db, "CREATE TABLE IF NOT EXISTS "+jobsStatusTable)
	if err != nil {
		return err
	}
	const stmnt = "INSERT INTO " + JobsStatusTableName + "(StatusID,StatusName) " +
		"VALUES (?,?),(?,?),(?,?)"
	// ignore any errors if the values do already exist
	Exec(
		db, stmnt,
		StatusIDFailed, StatusFailed,
		StatusIDRunning, StatusRunning,
		StatusIDDone, StatusDone,
	)
	_, err = Exec(db, "CREATE TABLE IF NOT EXISTS "+jobsTable)
	return err
}

// NewJob inserts a new running job into the jobs table and returns
// the new job ID.
func NewJob(db DB, bookID int) (int, error) {
	const stmnt = "INSERT INTO " + JobsTableName + "(JobID,StatusID,Timestamp) VALUES (?,?,?)"
	// ts := time.Now().Unix()
	_, err := Exec(db, stmnt, bookID, StatusIDRunning, time.Now().Unix())
	return bookID, err // book and job IDs are the same
}

// SetJobStatus sets a new status for a job.
func SetJobStatus(db DB, jobID, statusID int) error {
	const stmnt = "UPDATE " + JobsTableName + " SET StatusID=?,Timestamp=? WHERE JobID=?"
	// ts := time.Now().Unix()
	_, err := Exec(db, stmnt, statusID, time.Now().Unix(), jobID)
	return err
}

// FindJobByID returns the given job
func FindJobByID(db DB, jobID int) (*api.Job, bool, error) {
	const stmnt = "SELECT j.JobID,j.Timestamp,j.StatusID,js.StatusName " +
		"FROM " + JobsTableName + " AS j JOIN " + JobsStatusTableName + " js " +
		"ON j.StatusID = js.StatusID WHERE j.JobID=?"
	rows, err := Query(db, stmnt, jobID)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, false, nil
	}
	var j api.Job
	if err := rows.Scan(&j.JobID, &j.Timestamp, &j.StatusID, &j.StatusName); err != nil {
		return nil, false, err
	}
	j.BookID = j.JobID // job and book IDs are the same
	return &j, true, nil
}

// DeleteJobByID delete the given job from the database table.
func DeleteJobByID(db DB, jobID int) error {
	const stmnt = "DELETE FROM " + JobsTableName + " WHERE JobID=?"
	_, err := Exec(db, stmnt, jobID)
	return err
}
