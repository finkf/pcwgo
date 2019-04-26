package db

const PagesTableName = "pages"

const pagesTable = PagesTableName + "(" +
	"BookID INT REFERENCES " + BooksTableName + "(BookID)," +
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

// CreateTablePages creates the databases table pages if it does not
// already exist.  This function will fail if the table books does not
// exist.
func CreateTablePages(db DB) error {
	_, err := Exec(db, "CREATE TABLE IF NOT EXISTS "+pagesTable)
	return err
}

func InsertPage(db DB, page *Page) error {
	const stmt = "INSERT INTO " + PagesTableName +
		"(BookID,PageID,ImagePath,PLeft,PRight,PTop,PBottom)" +
		"VALUES(?,?,?,?,?,?,?)"
	_, err := Exec(db, stmt, page.BookID, page.PageID, page.ImagePath,
		page.Left, page.Right, page.Top, page.Bottom)
	return err
}
