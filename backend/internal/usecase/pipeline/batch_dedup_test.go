package pipeline

import (
	"reflect"
	"testing"
)

// G1 load-test finding: duplicate asset ids must collapse to one item each.
func TestDedupPreservingOrder(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{"dups collapse, order kept", []string{"a", "b", "a", "c", "b"}, []string{"a", "b", "c"}},
		{"blanks and spaces dropped", []string{" a ", "", "  ", "a"}, []string{"a"}},
		{"already unique untouched", []string{"x", "y"}, []string{"x", "y"}},
		{"empty in empty out", nil, []string{}},
	}
	for _, tc := range cases {
		if got := dedupPreservingOrder(tc.in); !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}
