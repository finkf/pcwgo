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
			"/a/1/b/2/c/3",
			regexp.MustCompile(`/a/(\d+)/b/(\d+)/c/(\d+)$`),
			[]int{1, 2, 3}, 3,
		},
		{
			"/a/1/b/2/c/3",
			regexp.MustCompile(`/a/(\d+)(?:/b/(\d+)(?:/c/(\d+))?)?$`),
			[]int{1, 2, 3}, 3,
		},
		{
			"/a/1/b/2",
			regexp.MustCompile(`/a/(\d+)(?:/b/(\d+)(?:/c/(\d+))?)?$`),
			[]int{1, 2}, 2,
		},
		{
			"/a/1",
			regexp.MustCompile(`/a/(\d+)(?:/b/(\d+)(?:/c/(\d+))?)?$`),
			[]int{1}, 1,
		},
		{
			"/a/1/b/2",
			regexp.MustCompile(`/a/(\d+)/b/(\d+)/c/(\d+)$`),
			nil, 0,
		},
		{
			"/a/1000000000000000000000000000000000000000000000000/b/2",
			regexp.MustCompile(`/a/(\d+)/b/(\d+)`),
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
