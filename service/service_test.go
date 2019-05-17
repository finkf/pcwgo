package service

import (
	"regexp"
	"testing"
)

func TestParseIDs(t *testing.T) {
	tests := []struct {
		test string
		re   *regexp.Regexp
		want []int
		ok   bool
	}{
		{
			"/books/1/pages/2/lines/3",
			regexp.MustCompile(`/books/(\d+)/pages/(\d+)/lines/(\d+)$`),
			[]int{1, 2, 3}, true,
		},
		{
			"/books/1/pages/2",
			regexp.MustCompile(`/books/(\d+)/pages/(\d+)/lines/(\d+)$`),
			nil, false,
		},
		{
			"/books/1000000000000000000000000000000000000000000000000/pages/2",
			regexp.MustCompile(`/books/(\d+)/pages/(\d+)/lines/(\d+)$`),
			nil, false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.test, func(t *testing.T) {
			ids := make([]*int, len(tc.want))
			for i := range tc.want {
				ids[i] = new(int)
			}
			err := ParseIDs(tc.test, tc.re, ids...)
			if tc.ok && err != nil {
				t.Fatalf("got error: %v", err)
			}
			if !tc.ok && err == nil {
				t.Fatalf("expected an error")
			}
			if !tc.ok {
				return
			}
			for i, id := range tc.want {
				if *ids[i] != id {
					t.Fatalf("expected ids[%d]=%d; got %d", i, id, *ids[i])
				}
			}
		})
	}
}
