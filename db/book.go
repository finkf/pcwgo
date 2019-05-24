package db

import "database/sql"

const BooksTableName = "books"

const booksTable = BooksTableName + "(" +
	"BookID INT NOT NULL UNIQUE REFERENCES Projects(ProjectID)," +
	"year INT," +
	"Author VARCHAR(100)," +
	"Title VARCHAR(100)," +
	"Description VARCHAR(255)," +
	"URI VARCHAR(255)," +
	"ProfilerURL VARCHAR(255)," +
	"Directory VARCHAR(255) NOT NULL," +
	"Lang VARCHAR(50) NOT NULL," +
	"Status INT REFERENCES status(ID)," +
	"PRIMARY KEY (BookID)" +
	");"

type Book struct {
	BookID, Year                       int
	Author, Title, Description, Status string
	URI, ProfilerURL, Directory, Lang  string
}

// CreateTableBooks the database table books if it does not already
// exist.  This function will fail, if the projects table does not
// exist.
func CreateTableBooks(db DB) error {
	_, err := Exec(db, "CREATE TABLE IF NOT EXISTS "+booksTable)
	return err
}

func InsertBook(db DB, book *Book) error {
	const stmt = "INSERT INTO " + BooksTableName +
		"(BookID,Author,Title,Year,Description,URI,ProfilerURL,Directory,Lang,Status)" +
		"VALUES(?,?,?,?,?,?,?,?,?,?)"
	_, err := Exec(db, stmt, book.BookID, book.Author, book.Title,
		book.Year, book.Description,
		book.URI, book.ProfilerURL, book.Directory, book.Lang, StatusIDEmpty)
	return err
}

// FindBookByID loads the book from the database that is identified by
// the given ID.
func FindBookByID(db DB, id int) (*Book, bool, error) {
	const stmt = "SELECT BookID,Year,Author,Title,Description,URI," +
		"COALESCE(ProfilerURL, '') as ProfilerURL,Directory,Lang FROM " +
		BooksTableName + " WHERE BookID=?"
	rows, err := Query(db, stmt, id)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, false, nil
	}
	var book Book
	if err := scanBook(rows, &book); err != nil {
		return nil, false, err
	}
	return &book, true, nil
}

// FindBookByProjectID loads the book from the database that is
// identified by the given project ID.
func FindBookByProjectID(db DB, id int) (*Book, bool, error) {
	const stmt = "SELECT b.BookID,b.Year,b.Author,b.Title,b.Description,b.URI," +
		"COALESCE(b.ProfilerURL, '') as ProfilerURL,b.Directory,b.Lang FROM " +
		BooksTableName + " b JOIN " + ProjectsTableName + " p ON p.Origin=b.BookID WHERE p.ID=?"
	rows, err := Query(db, stmt, id)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, false, nil
	}
	var book Book
	if err := scanBook(rows, &book); err != nil {
		return nil, false, err
	}
	return &book, true, nil
}

func scanBook(rows *sql.Rows, book *Book) error {
	return rows.Scan(&book.BookID, &book.Year, &book.Author, &book.Title,
		&book.Description, &book.URI, &book.ProfilerURL, &book.Directory,
		&book.Lang)
}
