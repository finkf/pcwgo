package db

import (
	"database/sql"
	"strings"
	"unicode"
)

// TextLinesTableName defines the name of the textlines table.
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

// ContentsTableName defines the name of the contents table.
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

// Char defines a character.
type Char struct {
	Cor, OCR rune
	Cut, Seq int
	Conf     float64
}

// IsCorrected returns true if the given character is corrected.
func (c Char) IsCorrected() bool {
	return c.Cor == 0
}

// Chars defines a slice of characters.
type Chars []Char

// AverageConfidence calculates the average confidence of the
// character slice.
func (cs Chars) AverageConfidence() float64 {
	if len(cs) == 0 {
		return 0.0
	}
	sum := 0.0
	for _, c := range cs {
		sum += c.Conf
	}
	return sum / float64(len(cs))
}

// IsFullyCorrected returns true if all characters in the slice have
// been corrected.
func (cs Chars) IsFullyCorrected() bool {
	for _, c := range cs {
		if c.Cor == 0 {
			return false
		}
	}
	return true
}

// IsPartiallyCorrected returns true if a part of the character slice
// contains corrections.
func (cs Chars) IsPartiallyCorrected() bool {
	for _, c := range cs {
		if c.Cor != 0 || c.Cor == -1 {
			return true
		}
	}
	return false
}

// Cor returns the corrected string.
func (cs Chars) Cor() string {
	var b strings.Builder
	for _, c := range cs {
		if c.Cor != 0 && c.Cor != -1 {
			b.WriteRune(c.Cor)
		} else if c.Cor != -1 && c.OCR != 0 {
			b.WriteRune(c.OCR)
		}
	}
	return b.String()
}

// OCR returns the OCR string.
func (cs Chars) OCR() string {
	var b strings.Builder
	for _, c := range cs {
		if c.OCR != 0 && c.OCR != rune(-1) {
			b.WriteRune(c.OCR)
		}
	}
	return b.String()
}

func issep(char Char) bool {
	c := char.Cor
	if c == 0 {
		c = char.OCR
	}
	// a deletion (cor = -1, ocr = char) is not a sep
	return c != -1 && unicode.IsSpace(c)
}

// NextWord returns the next word and the rest in this character
// sequence.  A word/token is any sequence of non whitespace
// characters.  Deletions are ignored.
func (cs Chars) NextWord() (word, rest Chars) {
	for len(cs) > 0 && issep(cs[0]) {
		cs = cs[1:]
	}
	if len(cs) == 0 {
		return nil, nil
	}
	i := 1
	for ; i < len(cs); i++ {
		if issep(cs[i]) {
			break
		}
	}
	return cs[:i], cs[i:]
}

// Line defines the line of a page in a book.
type Line struct {
	ImagePath                string
	Chars                    Chars
	LineID                   int
	PageID                   int
	BookID                   int
	Left, Right, Top, Bottom int
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

// InsertLine inserts the given line into the database.
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

// UpdateLine updates the contents for the given line.
func UpdateLine(db DB, line *Line) error {
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

// FindPageLines returns all line IDs for the page identified by the
// given book and page IDs.
func FindPageLines(db DB, bookID, pageID int) ([]int, error) {
	const stmt = "SELECT LineID FROM " + TextLinesTableName + " WHERE bookID=? AND pageID=?"
	rows, err := Query(db, stmt, bookID, pageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var lineIDs []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		lineIDs = append(lineIDs, id)
	}
	return lineIDs, nil
}

// FindLineByID returns the line identified by the given book, page
// and line ID.
func FindLineByID(db DB, bookID, pageID, lineID int) (*Line, bool, error) {
	const stmt1 = "SELECT ImagePath,LLeft,LRight,LTop,LBottom FROM " +
		TextLinesTableName + " WHERE BookID=? AND PageID=? AND LineID=?"
	const stmt2 = "SELECT OCR,Cor,Cut,Conf,Seq FROM " + ContentsTableName +
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
	if err != nil {
		return nil, false, err
	}
	for rows.Next() {
		line.Chars = append(line.Chars, Char{})
		if err := scanChar(rows, &line.Chars[len(line.Chars)-1]); err != nil {
			return nil, false, err
		}
	}
	return &line, true, nil
}

func scanChar(rows *sql.Rows, char *Char) error {
	return rows.Scan(&char.OCR, &char.Cor, &char.Cut, &char.Conf, &char.Seq)
}

func scanLine(rows *sql.Rows, line *Line) error {
	err := rows.Scan(&line.ImagePath, &line.Left, &line.Right, &line.Top, &line.Bottom)
	if err != nil {
		return err
	}
	return nil
}
