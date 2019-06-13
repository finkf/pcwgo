package db

import (
	"database/sql"
	"fmt"

	"github.com/finkf/pcwgo/api"
)

// ProjectsTableName of the the database table.
const ProjectsTableName = "projects"

const projectsTable = "" +
	ProjectsTableName + " (" +
	"ID INTEGER NOT NULL PRIMARY KEY /*!40101 AUTO_INCREMENT */," +
	"Owner INTEGER NOT NULL REFERENCES Users(ID)," +
	"Origin INTEGER NOT NULL REFERENCES ID," +
	"Pages INTEGER NOT NULL" +
	")"

const ProjectPagesTableName = "project_pages"

const projectPagesTable = ProjectPagesTableName + " (" +
	"ProjectID INT NOT NULL REFERENCES Projects(id)," +
	"PageID INT NOT NULL REFERENCES Pages(PageID)" +
	")"

// Project wraps a book with project-related information.
type Project struct {
	Book
	ProjectID int
	Pages     int
	Owner     api.User
}

func (p Project) String() string {
	return fmt.Sprintf("%d/%d/%d %s", p.ProjectID, p.BookID, p.Pages, p.Owner)
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
	if err := CreateTableProjectPages(db); err != nil {
		return fmt.Errorf("cannot create table %s: %v", ProjectPagesTableName, err)
	}
	if err := CreateTableLines(db); err != nil {
		return fmt.Errorf("cannot create tables %s,%s: %v",
			TextLinesTableName, ContentsTableName, err)
	}
	return nil
}

func InsertProject(db DB, p *Project) error {
	const stmt = "INSERT INTO " + ProjectsTableName +
		"(Owner,Origin,Pages) VALUES(?,?,?)"
	res, err := Exec(db, stmt, p.Owner.ID, p.BookID, p.Pages)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	p.ProjectID = int(id)
	return nil
}

func FindProjectByID(db DB, id int) (*Project, bool, error) {
	const stmt = "SELECT p.ID,p.Pages," +
		"b.BookID,b.Year,b.Author,b.Title,b.Description,b.URI," +
		"COALESCE(b.ProfilerURL,''),b.Directory,b.Lang,s.text," +
		"u.ID,u.Name,u.Email,u.Institute,u.Admin " +
		"FROM " + ProjectsTableName + " p JOIN " + UsersTableName +
		" u ON p.Owner=u.ID JOIN " + BooksTableName +
		" b ON p.Origin=b.BookID JOIN " + StatusTableName +
		" s ON b.status=s.id " +
		"WHERE p.ID=?"
	rows, err := Query(db, stmt, id)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, false, nil
	}
	var p Project
	if err := scanProject(rows, &p); err != nil {
		return nil, false, err
	}
	return &p, true, nil
}

func FindProjectByOwner(db DB, owner int64) ([]Project, error) {
	const stmt = "SELECT p.ID,p.Pages," +
		"b.BookID,b.Year,b.Author,b.Title,b.Description,b.URI," +
		"COALESCE(b.ProfilerURL,''),b.Directory,b.Lang,s.text," +
		"u.ID,u.Name,u.Email,u.Institute,u.Admin " +
		"FROM " + ProjectsTableName + " p JOIN " + UsersTableName +
		" u ON p.Owner=u.ID JOIN " + BooksTableName +
		" b ON p.Origin=b.BookID JOIN " + StatusTableName +
		" s ON b.status=s.id " +
		"WHERE p.Owner=?"
	rows, err := Query(db, stmt, owner)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ps []Project
	for rows.Next() {
		ps = append(ps, Project{})
		if err := scanProject(rows, &ps[len(ps)-1]); err != nil {
			return nil, err
		}
	}
	return ps, nil
}

func scanProject(rows *sql.Rows, p *Project) error {
	return rows.Scan(&p.ProjectID, &p.Pages,
		&p.BookID, &p.Year, &p.Author, &p.Title, &p.Description, &p.URI,
		&p.ProfilerURL, &p.Directory, &p.Lang, &p.Status,
		&p.Owner.ID, &p.Owner.Name, &p.Owner.Email,
		&p.Owner.Institute, &p.Owner.Admin)
}

func CreateTableProjectPages(db DB) error {
	_, err := Exec(db, "CREATE TABLE IF NOT EXISTS "+projectPagesTable)
	return err
}

// FindBookPages returns the page IDs for the given book.
func FindBookPages(db DB, bookID int) ([]int, error) {
	const stmt = "SELECT PageID FROM " + PagesTableName + " WHERE BookID=?"
	rows, err := Query(db, stmt, bookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return getIDs(rows)
}

// FindProjectPages returns the page IDs for the given project.
func FindProjectPages(db DB, projectID int) ([]int, error) {
	const stmt = "SELECT PageID FROM " + ProjectPagesTableName + " WHERE ProjectID=?"
	rows, err := Query(db, stmt, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return getIDs(rows)
}

func getIDs(rows *sql.Rows) ([]int, error) {
	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}
