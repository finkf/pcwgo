package database

import (
	"database/sql"

	log "github.com/sirupsen/logrus"
)

// DB defines a simple interface for database handling.
type DB interface {
	Exec(string, ...interface{}) (sql.Result, error)
	Query(string, ...interface{}) (*sql.Rows, error)
}

// Exec calls Exec on the given DB handle. The given args are logged.
func Exec(db DB, stmt string, args ...interface{}) (sql.Result, error) {
	log.Debugf("exec: %s %v", stmt, args)
	return db.Exec(stmt, args...)
}

// Query calls Query on the given DB handle. The given args are logged.
func Query(db DB, stmt string, args ...interface{}) (*sql.Rows, error) {
	log.Debugf("query: %s %v", stmt, args)
	return db.Query(stmt, args...)
}
