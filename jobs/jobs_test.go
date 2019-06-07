package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/finkf/pcwgo/db/sqlite"
)

func Test(t *testing.T) {
	// log.SetLevel(log.DebugLevel)
	// var done, failed int
	sqlite.With("jobs.sqlite", func(dtb *sql.DB) {
		if err := Init(dtb); err != nil {
			t.Fatalf("cannot initialize: %v", err)
		}
		defer func() {
			if err := Close(); err != nil {
				t.Fatalf("cannot close: %v", err)
			}
		}()
		testStart(t, 1, func(context.Context) error {
			return nil
		})
		testStart(t, 2, func(context.Context) error {
			return fmt.Errorf("error")
		})
		testStart(t, 3, func(ctx context.Context) error {
			for {
				select {
				case <-ctx.Done():
					return fmt.Errorf("canceled")
				default:
				}
			}
		})
	})
}

func testStart(t *testing.T, id int, f Func) {
	t.Helper()
	desc := Descriptor{BookID: id}
	if _, err := Start(context.Background(), desc, f); err != nil {
		t.Fatalf("cannot start: %v", err)
	}
}
