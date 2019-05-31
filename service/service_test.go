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
		n    int
	}{
		{
			"/books/1/pages/2/lines/3",
			regexp.MustCompile(`/books/(\d+)/pages/(\d+)/lines/(\d+)$`),
			[]int{1, 2, 3}, 3,
		},
		{
			"/books/1/pages/2/lines/3",
			regexp.MustCompile(`/books/(\d+)(?:/pages/(\d+)(?:/lines/(\d+))?)?$`),
			[]int{1, 2, 3}, 3,
		},
		{
			"/books/1/pages/2",
			regexp.MustCompile(`/books/(\d+)(?:/pages/(\d+)(?:/lines/(\d+))?)?$`),
			[]int{1, 2}, 2,
		},
		{
			"/books/1",
			regexp.MustCompile(`/books/(\d+)(?:/pages/(\d+)(?:/lines/(\d+))?)?$`),
			[]int{1}, 1,
		},
		{
			"/books/1/pages/2",
			regexp.MustCompile(`/books/(\d+)/pages/(\d+)/lines/(\d+)$`),
			nil, 0,
		},
		{
			"/books/1000000000000000000000000000000000000000000000000/pages/2",
			regexp.MustCompile(`/books/(\d+)/pages/(\d+)/lines/(\d+)$`),
			nil, 0,
		},
	}
	for _, tc := range tests {
		t.Run(tc.test, func(t *testing.T) {
			ids := make([]*int, len(tc.want))
			for i := range tc.want {
				ids[i] = new(int)
			}
			n := ParseIDs(tc.test, tc.re, ids...)
			if n != tc.n {
				t.Fatalf("expected %d; got %d", tc.n, n)
			}
			if tc.n == 0 {
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
