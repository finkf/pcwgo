package db

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"

	"github.com/finkf/pcwgo/api"
	"golang.org/x/crypto/scrypt"
)

const (
	// HashLength defines the lenght of the password hash.
	HashLength = 64
	// SaltLength defines the length of the salt.
	SaltLength = 32
)

// UsersTableName defines the name of the table users.
const UsersTableName = "users"

var usersTable = "" +
	UsersTableName + "(" +
	"ID INTEGER NOT NULL PRIMARY KEY /*!40101 AUTO_INCREMENT */," +
	"Name VARCHAR(255) NOT NULL," +
	"Institute VARCHAR(255) NOT NULL," +
	"Email VARCHAR(255) NOT NULL UNIQUE," +
	"Hash VARCHAR(" + strconv.Itoa(HashLength*2) + ")," +
	"Salt VARCHAR(" + strconv.Itoa(SaltLength*2) + ")," +
	"Admin BOOLEAN DEFAULT(false) NOT NULL" +
	")"

// CreateTableUsers creates the users table if it does not already
// exist.
func CreateTableUsers(db DB) error {
	_, err := Exec(db, "CREATE TABLE IF NOT EXISTS "+usersTable)
	return err
}

// InsertUser inserts a new user into the database.  The user's id is
// adjusted accordingly.
func InsertUser(db DB, user *api.User) error {
	const stmt = "INSERT INTO " + UsersTableName + "(Name,Email,Institute,Admin) values(?,?,?,?)"
	res, err := Exec(db, stmt, user.Name, user.Email, user.Institute, user.Admin)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	user.ID = id
	return nil
}

// SetUserPassword updates the password for the given user.
func SetUserPassword(db DB, user api.User, password string) error {
	hash, salt, err := genSaltAndHash(password)
	if err != nil {
		return err
	}
	const stmt = "UPDATE " + UsersTableName + " SET Hash=?,Salt=? WHERE ID=?;"
	_, err = Exec(db, stmt, hash, salt, user.ID)
	return err
}

// AuthenticateUser authenticates a user.
func AuthenticateUser(db DB, user api.User, password string) error {
	const stmt = "SELECT Hash,Salt FROM " + UsersTableName + " WHERE ID=?"
	rows, err := Query(db, stmt, user.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	if !rows.Next() {
		return fmt.Errorf("invalid user: %v", user)
	}
	var hash, salt string
	if err = rows.Scan(&hash, &salt); err != nil {
		return fmt.Errorf("internal error: cannot scan row")
	}
	saltb, err := hex.DecodeString(salt)
	if err != nil {
		return err
	}
	trueHash, err := genSaltedHash(password, saltb)
	if err != nil {
		return err
	}
	if trueHash != hash {
		return fmt.Errorf("invalid authentification for user: %v", user)
	}
	return nil
}

// UpdateUser updates the data of the given user.
func UpdateUser(db DB, user api.User) error {
	const stmt = "UPDATE " + UsersTableName + " SET Name=?,Email=?,Institute=? WHERE ID=?"
	_, err := Exec(db, stmt, user.Name, user.Email, user.Institute, user.ID)
	return err
}

// DeleteUserByID deletes a user by ID.
func DeleteUserByID(db DB, id int64) error {
	const stmt = "DELETE FROM " + UsersTableName + " WHERE ID=?"
	_, err := Exec(db, stmt, id)
	return err
}

// FindUserByID searches for a user by ID.
func FindUserByID(db DB, id int64) (api.User, bool, error) {
	const stmt = "SELECT ID,Name,Email,Institute,Admin FROM " + UsersTableName + " WHERE ID=?"
	return selectUser(db, stmt, id)
}

// FindUserByEmail searches for a user by its email.
func FindUserByEmail(db DB, email string) (api.User, bool, error) {
	const stmt = "SELECT ID,Name,Email,Institute,Admin FROM " + UsersTableName + " WHERE Email=?"
	return selectUser(db, stmt, email)
}

// FindAllUsers returns all users in the database.
func FindAllUsers(db DB) ([]api.User, error) {
	const stmt = "SELECT ID,Name,Email,Institute,Admin FROM " + UsersTableName
	rows, err := Query(db, stmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []api.User
	for rows.Next() {
		user, err := getUserFromRow(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

func selectUser(db DB, q string, args ...interface{}) (api.User, bool, error) {
	rows, err := Query(db, q, args...)
	if err != nil {
		return api.User{}, false, err
	}
	defer rows.Close()
	if !rows.Next() {
		return api.User{}, false, nil
	}
	user, err := getUserFromRow(rows)
	if err != nil {
		return api.User{}, false, err
	}
	return user, true, nil
}

func getUserFromRow(rows *sql.Rows) (api.User, error) {
	var user api.User
	if err := rows.Scan(&user.ID, &user.Name, &user.Email, &user.Institute, &user.Admin); err != nil {
		return api.User{}, err
	}
	return user, nil
}

func genSaltAndHash(password string) (string, string, error) {
	salt, err := genSalt()
	if err != nil {
		return "", "", err
	}
	hash, err := genSaltedHash(password, salt)
	if err != nil {
		return "", "", err
	}
	return hash, hex.EncodeToString(salt), nil
}

func genSalt() ([]byte, error) {
	salt := make([]byte, SaltLength)
	_, err := io.ReadFull(rand.Reader, salt)
	if err != nil {
		return nil, err
	}
	return salt, nil
}

func genSaltedHash(password string, salt []byte) (string, error) {
	hash, err := scrypt.Key([]byte(password), salt, 1<<14, 8, 1, HashLength)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hash), nil
}
