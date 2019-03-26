package db

import (
	"database/sql"
	"fmt"
	"reflect"
	"testing"

	"github.com/finkf/pcwgo/db/sqlite"
	log "github.com/sirupsen/logrus"
)

func newTestLine(t *testing.T, db DB, id int) *Line {
	if err := CreateTableTextLines(db); err != nil {
		t.Fatalf("got error: %v", err)
	}
	page := newTestPage(t, db, id)
	line := &Line{
		ImagePath: fmt.Sprintf("line_image_path_%d", id),
		Chars:     newChars(id),
		LineID:    id,
		PageID:    page.PageID,
		BookID:    page.BookID,
		Left:      id * 10,
		Right:     id * 100,
		Top:       id * 1000,
		Bottom:    id * 10000,
	}
	if err := NewLine(db, line); err != nil {
		t.Fatalf("got error: %v", err)
	}
	return line
}

func newChars(id int) []Char {
	ocr := fmt.Sprintf("ocr_%d", id)
	cor := fmt.Sprintf("cor_%d", id)
	chars := make([]Char, len(ocr))
	for i := range ocr {
		chars[i] = Char{
			Cor:  rune(cor[i]),
			OCR:  rune(ocr[i]),
			Cut:  i + id,
			Conf: float64(id) / float64(i+1),
		}
	}
	return chars
}

func TestFindLineByID(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	sqlite.With("lines.sqlite", func(db *sql.DB) {
		tests := []struct {
			test *Line
			id   int
			find bool
		}{
			{newTestLine(t, db, 1), 1, true},
			{newTestLine(t, db, 2), 2, true},
			{newTestLine(t, db, 3), 4, false},
		}
		for _, tc := range tests {
			t.Run(fmt.Sprintf("line_%d", tc.test.LineID), func(t *testing.T) {
				got, found, err := FindLineByID(db, tc.test.BookID, tc.test.PageID, tc.test.LineID)
				if err != nil {
					t.Fatalf("got error: %v", err)
				}
				if found != tc.find {
					t.Fatalf("expected find=%t; got %t", tc.find, found)
				}
				if !reflect.DeepEqual(*got, *tc.test) {
					t.Fatalf("expected line=%v; got %v", *tc.test, *got)
				}
			})
		}
	})
	log.SetLevel(log.DebugLevel)
}

// 	withLineDB(func(db *sql.DB) {
// 		tests := []struct {
// 			ocr, cor      string
// 			bid, pid, lid int
// 		}{
// 			{"abc", "ABC", 1, 4, 1},
// 		}
// 		for _, tc := range tests {
// 			t.Run(fmt.Sprintf("%d-%d-%d", tc.bid, tc.pid, tc.lid), func(t *testing.T) {
// 				line := newLine(tc.bid, tc.pid, tc.lid, tc.cor, tc.ocr)
// 				if err := Insert(db, line); err != nil {
// 					t.Fatalf("got error: %v", err)
// 				}
// 				got, ok, err := FindByID(db, tc.bid, tc.pid, tc.lid)
// 				if err != nil {
// 					t.Fatalf("got error: %v", err)
// 				}
// 				if !ok {
// 					t.Fatalf("cannot find line")
// 				}
// 				if tc.cor != got.Cor() {
// 					t.Fatalf("expected cor = %s; got cor = %s", tc.cor, got.Cor())
// 				}
// 				if tc.ocr != got.OCR() {
// 					t.Fatalf("expected ocr = %s; got ocr = %s", tc.ocr, got.OCR())
// 				}
// 			})
// 		}
// 	})
// }
