package db

import (
	"fmt"
	"testing"
)

func newTestPage(t *testing.T, db DB, id int) *Page {
	if err := CreateTablePages(db); err != nil {
		t.Fatalf("got error: %v", err)
	}
	book := newTestBook(t, db, id)
	page := &Page{
		ImagePath: fmt.Sprintf("page_image_path_%d", id),
		BookID:    book.BookID,
		PageID:    id,
		Left:      id * 10,
		Right:     id * 100,
		Top:       id * 1000,
		Bottom:    id * 10000,
	}
	err := InsertPage(db, page)
	if err != nil {
		t.Fatalf("got error: %v", err)
	}
	return page
}
