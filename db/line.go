package db

import (
	"database/sql"
	"fmt"
	"strings"
)

const TextLinesTableName = "textlines"
const tableTextLines = TextLinesTableName + " (" +
	"BookID INT REFERENCES Books(BookID)," +
	"PageID INT REFERENCES Pages(PageID)," +
	"LineID INT NOT NULL," +
	"ImagePath VARCHAR(255)," +
	"LLeft INT," +
	"LTop INT," +
	"LRight INT," +
	"LBottom INT," +
	"PRIMARY KEY (BookID, PageID, LineID)" +
	");"

const ContentsTableName = "contents"
const tableContents = ContentsTableName + " (" +
	"BookID INT REFERENCES Books(BookID)," +
	"PageID INT REFERENCES Pages(PageID)," +
	"LineID INT REFERENCES " + TextLinesTableName + "(LineID)," +
	"Seq INT NOT NULL," +
	"OCR INT NOT NULL," +
	"Cor INT NOT NULL," +
	"Cut INT NOT NULL," +
	"Conf double NOT NULL," +
	"PRIMARY KEY (BookID, PageID, LineID, Seq)" +
	");"

// Char defines a character
type Char struct {
	Cor, OCR rune
	Cut      int
	Conf     float64
}

// Line defines the line of a page in a book.
type Line struct {
	ImagePath                string
	Chars                    []Char
	LineID                   int
	PageID                   int
	BookID                   int
	Left, Right, Top, Bottom int
}

func (l Line) AverageConfidence() float64 {
	sum := 0.0
	for _, char := range l.Chars {
		sum += char.Conf
	}
	return sum / float64(len(l.Chars))
}

func (l Line) IsFullyCorrected() bool {
	for _, char := range l.Chars {
		if char.Cor == 0 {
			return false
		}
	}
	return true
}

func (l Line) IsPartiallyCorrected() bool {
	for _, char := range l.Chars {
		if char.Cor != 0 {
			return true
		}
	}
	return false
}

func (l Line) Cor() string {
	var b strings.Builder
	for _, char := range l.Chars {
		if char.Cor != 0 {
			b.WriteRune(char.Cor)
		}
	}
	return b.String()
}

func (l Line) OCR() string {
	var b strings.Builder
	for _, char := range l.Chars {
		if char.OCR != 0 && char.OCR != rune(-1) {
			b.WriteRune(char.OCR)
		}
	}
	return b.String()
}

// CreateTableTextLines creates the textlines table (and all its
// directly dependent tables).  It does not matter if
// CreateTableTextLines or CreateTableContents is used to create the
// tables for the storage of page lines.
func CreateTableTextLines(db DB) error {
	if err := CreateTablePages(db); err != nil {
		return fmt.Errorf("cannot create table pages: %v", err)
	}
	if err := CreateTableBooks(db); err != nil {
		return fmt.Errorf("cannot create table pages: %v", err)
	}
	if err := CreateTableContents(db); err != nil {
		return fmt.Errorf("cannot create table textlines: %v", err)
	}
	_, err := Exec(db, "CREATE TABLE IF NOT EXISTS "+tableTextLines)
	return err
}

// CreateTableContents creates the contents table (and all its
// directly dependent tables).  It does not matter if
// CreateTableTextLines or CreateTableContents is used to create the
// tables for the storage of page lines.
func CreateTableContents(db DB) error {
	if err := CreateTablePages(db); err != nil {
		return fmt.Errorf("cannot create table pages: %v", err)
	}
	if err := CreateTableBooks(db); err != nil {
		return fmt.Errorf("cannot create table pages: %v", err)
	}
	if err := CreateTableTextLines(db); err != nil {
		return fmt.Errorf("cannot create table textlines: %v", err)
	}
	_, err := Exec(db, "CREATE TABLE IF NOT EXISTS "+tableContents)
	return err
}

func NewLine(db DB, line *Line) error {
	const stmt1 = "INSERT INTO " + TextLinesTableName +
		"(BookID,PageID,LineID,ImagePath,LLeft,LRight,LTop,LBottom) " +
		"VALUES(?,?,?,?,?,?,?,?)"
	const stmt2 = "INSERT INTO " + ContentsTableName +
		"(BookID,PageID,LineID,OCR,Cor,Cut,Conf,Seq) " +
		"VALUES(?,?,?,?,?,?,?,?)"
	t := NewTransaction(Begin(db))
	t.Do(func(db DB) error {
		_, err := Exec(db, stmt1, line.BookID, line.PageID, line.LineID,
			line.ImagePath, line.Left, line.Right, line.Top, line.Bottom)
		return err
	})
	for i, char := range line.Chars {
		t.Do(func(db DB) error {
			_, err := Exec(db, stmt2, line.BookID, line.PageID, line.LineID,
				char.OCR, char.Cor, char.Cut, char.Conf, i+1)
			return err
		})
	}
	return t.Done()
}

func UpdateLine(db DB, line Line) error {
	const stmt1 = "UPDATE " + TextLinesTableName + " SET " +
		"ImagePath=?,LLeft=?,LRight=?,LTop=?,LBottom=? " +
		"WHERE BookID=? AND PageID=? AND LineID=?"
	const stmt2 = "UPDATE " + ContentsTableName + " SET " +
		"OCR=?,Cor=?,Cut=?,Conf=?,Seq=? " +
		"WHERE BookID=? AND PageID=? AND LineID=?"
	t := NewTransaction(Begin(db))
	t.Do(func(db DB) error {
		_, err := Exec(db, stmt1,
			line.ImagePath, line.Left, line.Right, line.Top, line.Bottom,
			line.BookID, line.PageID, line.LineID)
		return err
	})
	for i, char := range line.Chars {
		t.Do(func(db DB) error {
			_, err := Exec(db, stmt2,
				char.OCR, char.Cor, char.Cut, char.Conf, i+1,
				line.BookID, line.PageID, line.LineID)
			return err
		})
	}
	return t.Done()
}

func FindLineByID(db DB, bookID, pageID, lineID int) (*Line, bool, error) {
	const stmt = "SELECT l.ImagePath,l.LLeft,l.LRight,l.LTop,l.LBottom," +
		"c.OCR,c.Cor,c.Cut,c.Conf FROM " + TextLinesTableName +
		" l JOIN " + ContentsTableName + " c " +
		"ON l.BookID=c.BookID AND l.PageID=c.PageID AND l.LineID=c.LineID " +
		"WHERE l.BookID=? AND l.PageID=? AND l.LineID=? ORDER BY c.Seq"
	rows, err := Query(db, stmt, bookID, pageID, lineID)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	line := Line{
		BookID: bookID,
		PageID: pageID,
		LineID: lineID,
	}
	for rows.Next() {
		if err := scanLine(rows, &line); err != nil {
			return nil, false, err
		}
	}
	return &line, len(line.Chars) > 0, nil
}

func scanLine(rows *sql.Rows, line *Line) error {
	var char Char
	err := rows.Scan(&line.ImagePath, &line.Left, &line.Right, &line.Top, &line.Bottom,
		&char.OCR, &char.Cor, &char.Cut, &char.Conf)
	if err != nil {
		return err
	}
	line.Chars = append(line.Chars, char)
	return nil
}
