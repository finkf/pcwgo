package db

import (
	"database/sql"
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

// CreateTableLines creates the two tables needed for the storing of
// text lines in the right order.  The creation will fail, if the
// books and pages tables do not yet exist.
func CreateTableLines(db DB) error {
	_, err := Exec(db, "CREATE TABLE IF NOT EXISTS "+tableTextLines)
	if err != nil {
		return err
	}
	_, err = Exec(db, "CREATE TABLE IF NOT EXISTS "+tableContents)
	return err
}

func InsertLine(db DB, line *Line) error {
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
	const stmt1 = "SELECT ImagePath,LLeft,LRight,LTop,LBottom FROM " +
		TextLinesTableName + " WHERE BookID=? AND PageID=? AND LineID=?"
	const stmt2 = "SELECT OCR,Cor,Cut,Conf FROM " + ContentsTableName +
		" WHERE BookID=? AND PageID=? AND LineID=? ORDER BY Seq"
	// query for textlines content
	rows, err := Query(db, stmt1, bookID, pageID, lineID)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, false, nil
	}
	line := Line{
		BookID: bookID,
		PageID: pageID,
		LineID: lineID,
	}
	if err := scanLine(rows, &line); err != nil {
		return nil, false, err
	}

	// query for contents
	rows, err = Query(db, stmt2, bookID, pageID, lineID)
	for rows.Next() {
		line.Chars = append(line.Chars, Char{})
		if err := scanChar(rows, &line.Chars[len(line.Chars)-1]); err != nil {
			return nil, false, err
		}
	}
	return &line, true, nil
}

func scanChar(rows *sql.Rows, char *Char) error {
	return rows.Scan(&char.OCR, &char.Cor, &char.Cut, &char.Conf)
}

func scanLine(rows *sql.Rows, line *Line) error {
	err := rows.Scan(&line.ImagePath, &line.Left, &line.Right, &line.Top, &line.Bottom)
	if err != nil {
		return err
	}
	return nil
}
