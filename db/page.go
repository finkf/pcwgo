package db

import "fmt"

const PagesTableName = "pages"

const pagesTable = PagesTableName + "(" +
	"BookID INT REFERENCES Books(BookID)," +
	"PageID INT NOT NULL," +
	"ImagePath VARCHAR(255)," +
	"PLeft INT," +
	"PTop INT," +
	"PRight INT," +
	"PBottom INT," +
	"PRIMARY KEY (BookID, PageID)" +
	");"

type Page struct {
	BookID, PageID           int
	ImagePath                string
	Left, Right, Top, Bottom int
}

func CreateTablePages(db DB) error {
	if err := CreateTableBooks(db); err != nil {
		return fmt.Errorf("cannot create table books: %v", err)
	}
	_, err := Exec(db, "CREATE TABLE IF NOT EXISTS "+pagesTable)
	return err
}

func NewPage(db DB, page *Page) error {
	const stmt = "INSERT INTO " + PagesTableName +
		"(BookID,PageID,ImagePath,PLeft,PRight,PTop,PBottom)" +
		"VALUES(?,?,?,?,?,?,?)"
	_, err := Exec(db, stmt, page.BookID, page.PageID, page.ImagePath,
		page.Left, page.Right, page.Top, page.Bottom)
	return err
}
