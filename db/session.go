package db

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strconv"
	"time"
)

const (
	// IDLength defines the length of auth tokens.
	IDLength = 10
	// Expires defines the time after a session expires
	Expires = 10 * time.Hour
)

// Name defines the name of the sessions table.
const SessionsTableName = "sessions"

var sessionsTable = "" +
	SessionsTableName + " (" +
	"Auth char(" + strconv.Itoa(IDLength) + ") NOT NULL UNIQUE," +
	"UserID INTEGER NOT NULL REFERENCES " + UsersTableName + "(ID)," +
	"Expires INTEGER NOT NULL" +
	")"

type Session struct {
	User    User   `json:"user"`
	Auth    string `json:"auth"`
	Expires int64  `json:"expires"`
}

func (s Session) Expired() bool {
	if s.Expires < time.Now().Unix() {
		return true
	}
	return false
}

func (s Session) String() string {
	return fmt.Sprintf("%s [%s] expires: %s",
		s.User, s.Auth, time.Unix(s.Expires, 0).Format("2006-01-02:15:04"))
}

// CreateTable creates the sessions table.
func CreateTableSessions(db DB) error {
	stmt := "CREATE TABLE IF NOT EXISTS " + sessionsTable + ";"
	_, err := Exec(db, stmt)
	return err
}

// Insert creates a new unique session for the given user in the database
// and returns the new session.
func InsertSession(db DB, u User) (Session, error) {
	auth, err := genAuth()
	if err != nil {
		return Session{}, err
	}
	expires := time.Now().Add(Expires).Unix()
	// Insert new session for the user.
	const stmt2 = "INSERT INTO " + SessionsTableName + "(Auth,UserID,Expires)values(?,?,?)"
	_, err = Exec(db, stmt2, auth, u.ID, expires)
	if err != nil {
		return Session{}, err
	}
	return Session{Auth: auth, User: u, Expires: expires}, nil
}

func FindSessionByID(db DB, id string) (Session, bool, error) {
	s, found, err := selectSession(db, id)
	if !found || err != nil {
		return s, found, err
	}
	return s, found, err
}

func DeleteSessionByUserID(db DB, id int64) error {
	const stmt = "DELETE FROM " + SessionsTableName + " WHERE UserID=?"
	_, err := Exec(db, stmt, id)
	return err
}

func selectSession(db DB, id string) (Session, bool, error) {
	const stmt = "" +
		"SELECT s.Auth,s.Expires,u.ID,u.Name,u.Email,u.Institute,u.Admin " +
		"FROM " + SessionsTableName + " s JOIN " +
		UsersTableName + " u ON s.UserID=u.ID WHERE s.Auth=?"
	rows, err := Query(db, stmt, id)
	if err != nil {
		return Session{}, false, err
	}
	defer rows.Close()
	if !rows.Next() {
		return Session{}, false, nil
	}
	var s Session
	if err = rows.Scan(&s.Auth, &s.Expires, &s.User.ID, &s.User.Name,
		&s.User.Email, &s.User.Institute, &s.User.Admin); err != nil {
		return Session{}, false, err
	}
	return s, true, nil
}

const sessionIDchars = "" +
	"abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
	"012345678"

func genAuth() (string, error) {
	id := make([]byte, IDLength)
	max := big.NewInt(int64(len(sessionIDchars)))
	for i := 0; i < IDLength; i++ {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		id[i] = sessionIDchars[int(n.Int64())]
	}
	return string(id), nil
}
