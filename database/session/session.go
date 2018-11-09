package session

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/finkf/pcwgo/database"
	"github.com/finkf/pcwgo/database/user"
)

const (
	// IDLength defines the length of auth tokens.
	IDLength = 10
	// Expires defines the time after a session expires
	Expires = 10 * time.Hour
)

// Name defines the name of the sessions table.
const Name = "sessions"

var table = "" +
	Name + " (" +
	"Auth char(" + strconv.Itoa(IDLength) + ") NOT NULL UNIQUE," +
	"UserID INTEGER NOT NULL REFERENCES " + user.Name + "(ID)," +
	"Expires INTEGER NOT NULL" +
	")"

type Session struct {
	User    user.User `json:"user"`
	Auth    string    `json:"auth"`
	Expires int64     `json:"expires"`
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
func CreateTable(db database.DB) error {
	stmt := "CREATE TABLE IF NOT EXISTS " + table + ";"
	_, err := database.Exec(db, stmt)
	return err
}

func New(db database.DB, u user.User) (Session, error) {
	// Delete2 any old user's session.
	const stmt1 = "DELETE FROM " + Name + " WHERE UserID=?"
	_, err := database.Exec(db, stmt1, u.ID)
	if err != nil {
		return Session{}, err
	}
	auth, err := genAuth()
	if err != nil {
		return Session{}, err
	}
	expires := time.Now().Add(Expires).Unix()
	// Insert new session for the user.
	const stmt2 = "INSERT INTO " + Name + "(Auth,UserID,Expires)values(?,?,?)"
	_, err = database.Exec(db, stmt2, auth, u.ID, expires)
	if err != nil {
		return Session{}, err
	}
	return Session{Auth: auth, User: u, Expires: expires}, nil
}

func FindByID(db database.DB, id string) (Session, bool, error) {
	s, found, err := selectSession(db, id)
	if !found || err != nil {
		return s, found, err
	}
	return s, found, err
}

func selectSession(db database.DB, id string) (Session, bool, error) {
	const stmt = "" +
		"SELECT s.Auth,s.Expires,u.ID,u.Name,u.Email,u.Institute,u.Admin " +
		"FROM " + Name + " s JOIN " + user.Name + " u ON s.UserID=u.ID WHERE s.Auth=?"
	rows, err := database.Query(db, stmt, id)
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
