package db

import (
	"fmt"
	"testing"
)

func newTestBook(t *testing.T, db DB, id int) *Book {
	if err := CreateTableBooks(db); err != nil {
		t.Fatalf("got error: %v", err)
	}
	p := newTestProject(t, db, id)
	book := &Book{
		BookID:      int(p.ID),
		Author:      fmt.Sprintf("book_author_%d", id),
		Title:       fmt.Sprintf("book_title_%d", id),
		Year:        1800 + id,
		Description: fmt.Sprintf("book_descriptions_%d", id),
		URI:         fmt.Sprintf("book_uri_%d", id),
		ProfilerURL: fmt.Sprintf("book_profiler_url_%d", id),
		Directory:   fmt.Sprintf("book_directory_%d", id),
		Lang:        fmt.Sprintf("book_lang_%d", id),
	}
	err := NewBook(db, book)
	if err != nil {
		t.Fatalf("got error: %v", err)
	}
	return book
}
