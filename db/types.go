package db

import (
	"database/sql"
	"fmt"
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
	stmt1 := "SELECT ID FROM " + TypesTableName +
		" WHERE " + TypesTableType + "=?"
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
	stmt2 := "INSERT INTO " + TypesTableName +
		" (" + TypesTableType + ") " +
		"VALUES (?)"
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

const (
	SuggestionsTableName             = "suggestions"
	SuggestionsTableID               = "id"
	SuggestionsTableBookID           = "bookid"
	SuggestionsTableTokenTypeID      = "tokentypid"
	SuggestionsTableSuggestionTypeID = "suggestiontypid"
	SuggestionsTableModernTypeID     = "moderntypid"
	SuggestionsTableDict             = "dict"
	SuggestionsTableWeight           = "weight"
	SuggestionsTableTopSuggestion    = "topsuggestion"
	SuggestionsTableDistance         = "distance"
	SuggestionsTableHistPatterns     = "histpatterns"
	SuggestionsTableOCRPatterns      = "ocrpatterns"
)

var suggestionsTable = SuggestionsTableName + "(" +
	SuggestionsTableID + " int not null unique primary key auto_increment," +
	SuggestionsTableBookID + " int references books(bookid)," +
	SuggestionsTableTokenTypeID + " int references types(id)," +
	SuggestionsTableSuggestionTypeID + " int references types(id)," +
	SuggestionsTableModernTypeID + " int references types(id)," +
	SuggestionsTableDict + " varchar(50) not null," +
	SuggestionsTableOCRPatterns + " varchar(50) not null," +
	SuggestionsTableHistPatterns + " varchar(50) not null," +
	SuggestionsTableWeight + " double not null," +
	SuggestionsTableDistance + " int not null," +
	SuggestionsTableTopSuggestion + " boolean not null," +
	");"

// CreateTableSuggestions creates the suggestion table.
func CreateTableSuggestions(db DB) error {
	_, err := Exec(db, "CREATE TABLE IF NOT EXISTS "+suggestionsTable)
	return err
}

// TypeInserter helps handle types with their according type IDs.
type TypeInserter struct {
	sel, ins *sql.Stmt
	ids      map[string]int
}

// NewTypeInserter constructs a new TypeInserter instance.
func NewTypeInserter(db sql.DB) (*TypeInserter, error) {
	sel, err := db.Prepare("SELECT id FROM types WHERE  typ=?")
	if err != nil {
		return nil, fmt.Errorf("cannot prepare type select statement: %v", err)
	}
	ins, err := db.Prepare("INSERT into types (typ) VALUES (?)")
	if err != nil {
		sel.Close() // sel must be closed; ignore error
		return nil, fmt.Errorf("cannot prepare type insert statment: %v", err)
	}
	return &TypeInserter{
		sel: sel,
		ins: ins,
		ids: make(map[string]int),
	}, nil
}

// ID returns the id of a given type.  If the type does not yet exist,
// it is inserted into the database.
func (t *TypeInserter) ID(typ string) (int, error) {
	// Cached?
	if id, ok := t.ids[typ]; ok {
		return id, nil
	}
	rows, err := t.sel.Query(typ)
	if err != nil {
		return 0, fmt.Errorf("cannot query type %s: %v", typ, err)
	}
	defer rows.Close()
	// Lookup type in database
	if rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return 0, fmt.Errorf("cannot scan id of type %s: %v", typ, err)
		}
		t.ids[typ] = id
		return id, nil
	}
	// New type: insert into database
	res, err := t.ins.Exec(typ)
	if err != nil {
		return 0, fmt.Errorf("cannot insert type %s: %v", typ, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("cannot insert type %s: %v", typ, err)
	}
	t.ids[typ] = int(id)
	return int(id), nil
}

// Close closes the database connections.
func (t *TypeInserter) Close() (err error) {
	defer func() {
		e := t.ins.Close()
		if err == nil {
			err = e
		}
	}()
	defer func() {
		e := t.sel.Close()
		if err == nil {
			err = e
		}
	}()
	return nil
}
