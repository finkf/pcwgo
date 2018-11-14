package user

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"

	"github.com/finkf/pcwgo/database"
	"golang.org/x/crypto/scrypt"
)

const (
	// HashLength defines the lenght of the password hash.
	HashLength = 64
	// SaltLength defines the length of the salt.
	SaltLength = 32
)

// Name of the table
const Name = "users"

var table = "" +
	Name + "(" +
	"ID INTEGER NOT NULL PRIMARY KEY /*!40101 AUTO_INCREMENT */," +
	"Name VARCHAR(255) NOT NULL," +
	"Institute VARCHAR(255) NOT NULL," +
	"Email VARCHAR(255) NOT NULL UNIQUE," +
	"Hash VARCHAR(" + strconv.Itoa(HashLength*2) + ")," +
	"Salt VARCHAR(" + strconv.Itoa(SaltLength*2) + ")," +
	"Admin BOOLEAN DEFAULT(false) NOT NULL" +
	")"

type User struct {
	Name      string `json:"name"`
	Email     string `json:"email"`
	Institute string `json:"institute"`
	ID        int64  `json:"id"`
	Admin     bool   `json:"admin"`
}

func (u User) String() string {
	adm := ""
	if u.Admin {
		adm = "/admin"
	}
	return fmt.Sprintf("%s/%d%s [%s,%s]", u.Email, u.ID, adm, u.Name, u.Institute)
}

// CreateTable creates the users table.
func CreateTable(db database.DB) error {
	_, err := database.Exec(db, "CREATE TABLE IF NOT EXISTS "+table)
	return err
}

func New(db database.DB, user User) (User, error) {
	const stmt = "INSERT INTO " + Name + "(Name,Email,Institute,Admin) values(?,?,?,?)"
	res, err := database.Exec(db, stmt, user.Name, user.Email, user.Institute, user.Admin)
	if err != nil {
		return User{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return User{}, err
	}
	user.ID = id
	return user, nil
}

func SetUserPassword(db database.DB, user User, password string) error {
	hash, salt, err := genSaltAndHash(password)
	if err != nil {
		return err
	}
	const stmt = "UPDATE " + Name + " SET Hash=?,Salt=? WHERE ID=?;"
	_, err = database.Exec(db, stmt, hash, salt, user.ID)
	return err
}

func AuthenticateUser(db database.DB, user User, password string) error {
	const stmt = "SELECT Hash,Salt FROM " + Name + " WHERE ID=?"
	rows, err := database.Query(db, stmt, user.ID)
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

func UpdateUser(db database.DB, user User) error {
	const stmt = "UPDATE " + Name + " SET Name=?,Email=?,Institute=? WHERE ID=?"
	_, err := database.Exec(db, stmt, user.Name, user.Email, user.Institute, user.ID)
	return err
}

func DeleteUserByID(db database.DB, id int64) error {
	const stmt = "DELETE FROM " + Name + " WHERE ID=?"
	_, err := database.Exec(db, stmt, id)
	return err
}

func FindByID(db database.DB, id int64) (User, bool, error) {
	const stmt = "SELECT ID,Name,Email,Institute,Admin FROM " + Name + " WHERE ID=?"
	return selectUser(db, stmt, id)
}

func FindByEmail(db database.DB, email string) (User, bool, error) {
	const stmt = "SELECT ID,Name,Email,Institute,Admin FROM " + Name + " WHERE Email=?"
	return selectUser(db, stmt, email)
}

// All returns all users in the database.
func All(db database.DB) ([]User, error) {
	const stmt = "SELECT ID,Name,Email,Institute,Admin FROM " + Name
	rows, err := database.Query(db, stmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []User
	for rows.Next() {
		user, err := getUserFromRow(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

func selectUser(db database.DB, q string, args ...interface{}) (User, bool, error) {
	rows, err := database.Query(db, q, args...)
	if err != nil {
		return User{}, false, err
	}
	defer rows.Close()
	if !rows.Next() {
		return User{}, false, nil
	}
	user, err := getUserFromRow(rows)
	if err != nil {
		return User{}, false, err
	}
	return user, true, nil
}

func getUserFromRow(rows *sql.Rows) (User, error) {
	var user User
	if err := rows.Scan(&user.ID, &user.Name, &user.Email, &user.Institute, &user.Admin); err != nil {
		return User{}, err
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
