package service

import (
	"reflect"
	"regexp"
	"testing"
)

func TestParseIDMap(t *testing.T) {
	tests := []struct {
		url     string
		keys    []string
		want    map[string]int
		wantErr bool
	}{
		{"/a/1/b/c/d/e/2", []string{"a", "e"}, map[string]int{"a": 1, "e": 2}, false},
		{"/jobs/1/books/2/lines/3", []string{"jobs", "books", "lines"},
			map[string]int{"jobs": 1, "books": 2, "lines": 3}, false},
		/* wrong order */
		{"/jobs/1/books/2", []string{"books", "jobs"}, nil, true},
		/* not a valid int */
		{"/jobs/1/books/3foobar/", []string{"jobs", "books"}, nil, true},
	}
	for _, tc := range tests {
		t.Run(tc.url, func(t *testing.T) {
			got, err := parseIDMap(tc.url, tc.keys...)
			if err == nil && tc.wantErr {
				t.Fatalf("expected an error")
			}
			if err != nil && !tc.wantErr {
				t.Fatalf("got error: %v", err)
			}
			if tc.wantErr {
				return
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("expected %v; got %v", tc.want, got)
			}
		})
	}
}

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
