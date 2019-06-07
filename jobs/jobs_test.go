package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/finkf/pcwgo/db/sqlite"
)

type runner struct {
	id  int
	run func(context.Context) error
}

func (r runner) BookID() int {
	return r.id
}

func (r runner) Name() string {
	return "runner"
}

func (r runner) Run(ctx context.Context) error {
	return r.run(ctx)
}

func testRunner(id int, run func(context.Context) error) Runner {
	return runner{id: id, run: run}
}

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
		testStart(t, testRunner(1, func(context.Context) error {
			return nil
		}))
		testStart(t, testRunner(2, func(context.Context) error {
			return fmt.Errorf("error")
		}))
		testStart(t, testRunner(3, func(ctx context.Context) error {
			for {
				select {
				case <-ctx.Done():
					return fmt.Errorf("canceled")
				default:
				}
			}
		}))
	})
}

func testStart(t *testing.T, r Runner) {
	t.Helper()
	if _, err := Start(context.Background(), r); err != nil {
		t.Fatalf("cannot start: %v", err)
	}
}
