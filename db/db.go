package db

import (
	"database/sql"
	"fmt"

	log "github.com/sirupsen/logrus"
)

// DB defines a simple interface for database handling.
type DB interface {
	Exec(string, ...interface{}) (sql.Result, error)
	Query(string, ...interface{}) (*sql.Rows, error)
	Begin() (*sql.Tx, error)
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

// Begin calls Begin on the given DB handle and logs the beginning of
// a transaction.
func Begin(db DB) (*sql.Tx, error) {
	log.Debugf("begin transaction")
	return db.Begin()
}

type Transaction struct {
	tx  *sql.Tx
	err error
}

func NewTransaction(tx *sql.Tx, err error) *Transaction {
	if err != nil {
		return &Transaction{err: err} // tx = nil, err != nil
	}
	return &Transaction{tx: tx} // tx != nil, err = nil
}

func (t *Transaction) Exec(stmt string, args ...interface{}) (sql.Result, error) {
	if t.err != nil {
		return nil, fmt.Errorf("cannot exec: transaction error: %v", t.err)
	}
	return t.tx.Exec(stmt, args...)
}

func (t *Transaction) Query(stmt string, args ...interface{}) (*sql.Rows, error) {
	if t.err != nil {
		return nil, fmt.Errorf("cannot query: transaction error: %v", t.err)
	}
	return t.tx.Query(stmt, args...)
}

func (t *Transaction) Begin() (*sql.Tx, error) {
	return nil, fmt.Errorf("cannot call Begin() on Transaction")
}

func (t *Transaction) Do(f func(DB) error) {
	if t.err != nil {
		return
	}
	t.err = f(t)
}

func (t *Transaction) Done() error {
	if t.err == nil { // no error; commit
		log.Debugf("commit transaction")
		if err := t.tx.Commit(); err != nil {
			return fmt.Errorf("cannot commit transaction: %v", err)
		}
		return nil
	}
	// error; rollback
	log.Debugf("rollback transaction")
	if err := t.tx.Rollback(); err != nil {
		return fmt.Errorf("cannot rollback after error: %v: %v", t.err, err)
	}
	return t.err
}
