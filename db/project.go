package db

import (
	"database/sql"
	"fmt"
)

// TableName of the the database table.
const ProjectsTableName = "projects"

const projectsTable = "" +
	ProjectsTableName + " (" +
	"ID INTEGER NOT NULL PRIMARY KEY /*!40101 AUTO_INCREMENT */," +
	"Owner INTEGER NOT NULL REFERENCES Users(ID)," +
	"Origin INTEGER NOT NULL," +
	"Pages INTEGER NOT NULL" +
	")"

type Project struct {
	ID, Origin, Pages int64
	Owner             User
}

func (p Project) String() string {
	return fmt.Sprintf("%d/%d/%d %s", p.ID, p.Origin, p.Pages, p.Owner)
}

// CreateTableProjects creates the project table if it does not
// already exist.  This function will fail, if the users table does
// not already exist.
func CreateTableProjects(db DB) error {
	_, err := Exec(db, "CREATE TABLE IF NOT EXISTS "+projectsTable)
	return err
}

// CreateAllTables creates all tables in the right order. The order
// is: users -> projects -> books -> pages -> lines.
func CreateAllTables(db DB) error {
	if err := CreateTableUsers(db); err != nil {
		return fmt.Errorf("cannot create table %s: %v", UsersTableName, err)
	}
	if err := CreateTableProjects(db); err != nil {
		return fmt.Errorf("cannot create table %s: %v", ProjectsTableName, err)
	}
	if err := CreateTableBooks(db); err != nil {
		return fmt.Errorf("cannot create table %s: %v", BooksTableName, err)
	}
	if err := CreateTablePages(db); err != nil {
		return fmt.Errorf("cannot create table %s: %v", PagesTableName, err)
	}
	if err := CreateTableLines(db); err != nil {
		return fmt.Errorf("cannot create tables %s,%s: %v",
			TextLinesTableName, ContentsTableName, err)
	}
	return nil
}

func NewProject(db DB, p *Project) error {
	const stmt = "INSERT INTO " + ProjectsTableName + "(Owner,Origin,Pages) values(?,?,?)"
	res, err := Exec(db, stmt, p.Owner.ID, p.Origin, p.Pages)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	p.ID = id
	return nil
}

func FindProjectByID(db DB, id int64) (*Project, bool, error) {
	const stmt = "" +
		"SELECT p.ID,p.Origin,p.Pages,u.ID,u.Name,u.Email,u.Admin " +
		"FROM " + ProjectsTableName + " p JOIN " + UsersTableName + " u ON p.Owner=u.ID " +
		"WHERE p.ID=?"
	rows, err := Query(db, stmt, id)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, false, nil
	}
	p, err := fromRows(rows)
	if err != nil {
		return nil, false, err
	}
	return p, true, nil
}

func FindProjectByUser(db DB, u User) ([]*Project, error) {
	const stmt = "" +
		"SELECT p.ID,p.Origin,p.Pages,u.ID,u.Name,u.Email,u.Admin " +
		"FROM " + ProjectsTableName + " p JOIN " + UsersTableName + " u on p.Owner=u.ID " +
		"WHERE p.Owner=?"
	rows, err := Query(db, stmt, u.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ps []*Project
	for rows.Next() {
		p, err := fromRows(rows)
		if err != nil {
			return nil, err
		}
		ps = append(ps, p)
	}
	return ps, nil
}

func fromRows(rows *sql.Rows) (*Project, error) {
	var p Project
	if err := rows.Scan(&p.ID, &p.Origin, &p.Pages, &p.Owner.ID, &p.Owner.Name, &p.Owner.Email, &p.Owner.Admin); err != nil {
		return nil, err
	}
	return &p, nil
}
