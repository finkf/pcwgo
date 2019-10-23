package db

import (
	"database/sql"
	"fmt"
)

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
	"profiled BOOLEAN DEFAULT(false) NOT NULL," +
	"extendedlexicon BOOLEAN DEFAULT(false) NOT NULL," +
	"postcorrected BOOLEAN DEFAULT(false) NOT NULL," +
	"pooled BOOLEAN DEFAULT(false) NOT NULL," +
	"PRIMARY KEY (BookID)" +
	");"

type Book struct {
	BookID, Year                             int
	Status                                   map[string]bool
	Author, Title, Description, HistPatterns string
	URI, ProfilerURL, Directory, Lang        string
	Pooled                                   bool
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
		"(BookID,Author,Title,Year,Description,URI,ProfilerURL,Directory,Lang," +
		"profiled,extendedlexicon,postcorrected,pooled)" +
		"VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)"
	_, err := Exec(db, stmt, book.BookID, book.Author, book.Title,
		book.Year, book.Description,
		book.URI, book.ProfilerURL, book.Directory, book.Lang,
		book.Status["profiled"], book.Status["extended-lexicon"],
		book.Status["post-corrected"], book.Pooled)
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

type BookWithContent struct {
	Book
	Pages map[int]*PageWithContent
}

type PageWithContent struct {
	Page
	Lines []Line
}

func LoadBookWithContent(db DB, bid, pid, lid int) (*BookWithContent, bool, error) {
	var book BookWithContent
	const stmnt = "SELECT BookID,Year,Author,Title,Description,URI," +
		"COALESCE(ProfilerURL, '') as ProfilerURL,Directory,Lang FROM " +
		BooksTableName + " WHERE BookID=?"
	rows, err := Query(db, stmnt, bid)
	if err != nil {
		return nil, false, fmt.Errorf("cannot load book: %v", err)
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, false, nil
	}
	if err := scanBook(rows, &book.Book); err != nil {
		return nil, false, fmt.Errorf("cannot load book: %v", err)
	}
	where := bookWhere(bid, pid, 0)
	if err := loadPages(db, &book, where); err != nil {
		return nil, false, fmt.Errorf("cannot load book: %v", err)
	}
	where = bookWhere(bid, pid, lid)
	if err := loadLines(db, &book, where); err != nil {
		return nil, false, fmt.Errorf("cannot load book: %v", err)
	}
	if err := loadContents(db, &book, where); err != nil {
		return nil, false, fmt.Errorf("cannot load book: %v", err)
	}
	return &book, true, nil
}

func loadPages(db DB, book *BookWithContent, where string) error {
	stmnt := "SELECT bookid,pageid,imagepath,ocrpath,pleft,ptop,pright,pbottom " +
		"FROM " + PagesTableName + " " + where
	rows, err := Query(db, stmnt)
	if err != nil {
		return fmt.Errorf("cannot load pages: %v", err)
	}
	defer rows.Close()
	book.Pages = make(map[int]*PageWithContent)
	for rows.Next() {
		var tmp PageWithContent
		if err := rows.Scan(&tmp.Page.BookID, &tmp.Page.PageID,
			&tmp.Page.ImagePath, &tmp.Page.OCRPath,
			&tmp.Page.Left, &tmp.Page.Top, &tmp.Page.Right, &tmp.Page.Bottom); err != nil {
			return fmt.Errorf("cannot load pages: %v", err)
		}
		book.Pages[tmp.PageID] = &tmp
	}
	return nil
}

func loadLines(db DB, book *BookWithContent, where string) error {
	stmnt := "SELECT bookid,pageid,lineid,imagepath,lleft,ltop,lright,lbottom " +
		"FROM " + TextLinesTableName + " " + where
	rows, err := Query(db, stmnt)
	if err != nil {
		return fmt.Errorf("cannot load lines: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var tmp Line
		if err := rows.Scan(&tmp.BookID, &tmp.PageID, &tmp.LineID, &tmp.ImagePath,
			&tmp.Left, &tmp.Top, &tmp.Right, &tmp.Bottom); err != nil {
			return fmt.Errorf("cannot load lines: %v", err)
		}
		book.Pages[tmp.PageID].Lines = append(book.Pages[tmp.PageID].Lines, tmp)
	}
	return nil
}

func loadContents(db DB, book *BookWithContent, where string) error {
	stmnt := "SELECT bookid,pageid,lineid,ocr,cor,cut,conf " +
		"FROM " + ContentsTableName + " " + where
	rows, err := Query(db, stmnt)
	if err != nil {
		return fmt.Errorf("cannot load contents: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var bid, pid, lid int
		var c Char
		if err := rows.Scan(&bid, &pid, &lid, &c.OCR, &c.Cor, &c.Cut, &c.Conf); err != nil {
			return fmt.Errorf("cannot load contents: %v", err)
		}
		if lid == 0 || book.Pages[pid].Lines[lid-1].LineID != lid {
			return fmt.Errorf("cannot load contents: invalid line id: %d", lid)
		}
		book.Pages[pid].Lines[lid-1].Chars = append(
			book.Pages[pid].Lines[lid-1].Chars, c)
	}
	return nil
}

func bookWhere(bid, pid, lid int) string {
	res := fmt.Sprintf("WHERE bookid=%d", bid)
	if pid != 0 {
		res += fmt.Sprintf(" AND pageid=%d", pid)
	}
	if lid != 0 {
		res += fmt.Sprintf(" AND lineid=%d", lid)
	}
	return res
}
