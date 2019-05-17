package db

import (
	"strconv"
	"strings"
)

const (
	TypesTableName = "types" // Name of the types table.
	TypesTableID   = "ID"    // ID field
	TypesTableType = "typ"   // type field
	MaxType        = 50      // Max length of type strings
)

var typesTable = TypesTableName + "(" +
	TypesTableID + " int not null unique primary key auto_increment," +
	TypesTableType + " varchar(" + strconv.Itoa(MaxType) + ") not null unique" +
	");"

// CreateTableTypes creates the types table.
func CreateTableTypes(db DB) error {
	_, err := Exec(db, "CREATE TABLE IF NOT EXISTS "+typesTable)
	return err
}

// NewType inserts a new (string-) type into the types tables and
// returns its id.  If the type does already exist in the table, the
// id of the existing string is returned and nothing is inserted into
// the table.  All types are converted to lowercase.
//
// An additional map can be supplied to speed up the creation of
// types.  A nil map can be supplied.
func NewType(db DB, str string, ids map[string]int) (int, error) {
	// convert type to lower case
	str = strings.ToLower(str)
	// check if id is in the map already
	if id, ok := ids[str]; ok {
		return id, nil
	}
	// check if type is already in the database and return it
	stmt1 := "SELECT ID FROM " + TypesTableName + " WHERE " + TypesTableType + "=?"
	rows, err := Query(db, stmt1, str)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	if rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return 0, err
		}
		if ids != nil {
			ids[str] = id
		}
		return id, nil
	}
	// insert new type into the database
	stmt2 := "INSERT INTO " + TypesTableName + "(" + TypesTableType + ") VALUES (?)"
	res, err := Exec(db, stmt2, str)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if ids != nil {
		ids[str] = int(id)
	}
	return int(id), err
}
