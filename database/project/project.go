package project

import (
	"database/sql"
	"fmt"

	"github.com/finkf/pcwgo/database"
	"github.com/finkf/pcwgo/database/user"
)

// Name of the the database table.
const Name = "projects"

const table = "" +
	Name + " (" +
	"ID INTEGER NOT NULL PRIMARY KEY /*!40101 AUTO_INCREMENT */," +
	"Owner INTEGER NOT NULL REFERENCES Users(ID)," +
	"Origin INTEGER NOT NULL," +
	"Pages INTEGER NOT NULL" +
	")"

type Project struct {
	ID, Origin, Pages int64
	Owner             user.User
}

func (p Project) String() string {
	return fmt.Sprintf("%d/%d/%d %s", p.ID, p.Origin, p.Pages, p.Owner)
}

// CreateTable creates the project table.
func CreateTable(db database.DB) error {
	_, err := database.Exec(db, "CREATE TABLE IF NOT EXISTS "+table)
	return err
}

func New(db database.DB, p Project) (Project, error) {
	const stmt = "INSERT INTO " + Name + "(Owner,Origin,Pages) values(?,?,?)"
	res, err := database.Exec(db, stmt, p.Owner.ID, p.Origin, p.Pages)
	if err != nil {
		return Project{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Project{}, err
	}
	p.ID = id
	return p, nil
}

func FindByID(db database.DB, id int64) (Project, bool, error) {
	const stmt = "" +
		"SELECT p.ID,p.Origin,p.Pages,u.ID,u.Name,u.Email,u.Admin " +
		"FROM " + Name + " p JOIN " + user.Name + " u ON p.Owner=u.ID " +
		"WHERE p.ID=?"
	rows, err := database.Query(db, stmt, id)
	if err != nil {
		return Project{}, false, err
	}
	defer rows.Close()
	if !rows.Next() {
		return Project{}, false, nil
	}
	p, err := fromRows(rows)
	if err != nil {
		return Project{}, false, err
	}
	return p, true, nil
}

func FindByUser(db database.DB, u user.User) ([]Project, error) {
	const stmt = "" +
		"SELECT p.ID,p.Origin,p.Pages,u.ID,u.Name,u.Email,u.Admin " +
		"FROM " + Name + " p JOIN " + user.Name + " u on p.Owner=u.ID " +
		"WHERE p.Owner=?"
	rows, err := database.Query(db, stmt, u.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ps []Project
	for rows.Next() {
		p, err := fromRows(rows)
		if err != nil {
			return nil, err
		}
		ps = append(ps, p)
	}
	return ps, nil
}

func fromRows(rows *sql.Rows) (Project, error) {
	var p Project
	if err := rows.Scan(&p.ID, &p.Origin, &p.Pages, &p.Owner.ID, &p.Owner.Name, &p.Owner.Email, &p.Owner.Admin); err != nil {
		return Project{}, err
	}
	return p, nil
}
