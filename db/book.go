package db

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
	"PRIMARY KEY (BookID)" +
	");"

type Book struct {
	BookID, Year                      int
	Author, Title, Description        string
	URI, ProfilerURL, Directory, Lang string
}

// CreateTableBooks the database table books if it does not already
// exist.  This function will fail, if the projects table does not
// exist.
func CreateTableBooks(db DB) error {
	_, err := Exec(db, "CREATE TABLE IF NOT EXISTS "+booksTable)
	return err
}

func NewBook(db DB, book *Book) error {
	const stmt = "INSERT INTO " + BooksTableName +
		"(BookID,Author,Title,Year,Description,URI,ProfilerURL,Directory,Lang)" +
		"VALUES(?,?,?,?,?,?,?,?,?)"
	_, err := Exec(db, stmt, book.BookID, book.Author, book.Title,
		book.Year, book.Description,
		book.URI, book.ProfilerURL, book.Directory, book.Lang)
	return err
}
