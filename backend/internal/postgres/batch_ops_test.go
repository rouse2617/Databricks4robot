package postgres

import "testing"

func TestChunkStrings(t *testing.T) {
	cases := []struct {
		name   string
		input  []string
		size   int
		expect [][]string
	}{
		{
			name:   "empty input",
			input:  nil,
			size:   100,
			expect: [][]string{nil},
		},
		{
			name:   "single chunk when size >= len",
			input:  []string{"a", "b", "c"},
			size:   10,
			expect: [][]string{{"a", "b", "c"}},
		},
		{
			name:   "evenly divides",
			input:  []string{"a", "b", "c", "d"},
			size:   2,
			expect: [][]string{{"a", "b"}, {"c", "d"}},
		},
		{
			name:   "trailing partial chunk",
			input:  []string{"a", "b", "c", "d", "e"},
			size:   2,
			expect: [][]string{{"a", "b"}, {"c", "d"}, {"e"}},
		},
		{
			name:   "non-positive size collapses to single chunk",
			input:  []string{"a", "b"},
			size:   0,
			expect: [][]string{{"a", "b"}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := chunkStrings(tc.input, tc.size)
			if len(got) != len(tc.expect) {
				t.Fatalf("chunk count: got %d want %d (%v)", len(got), len(tc.expect), got)
			}
			for i := range got {
				if len(got[i]) != len(tc.expect[i]) {
					t.Fatalf("chunk %d size: got %d want %d", i, len(got[i]), len(tc.expect[i]))
				}
				for j := range got[i] {
					if got[i][j] != tc.expect[i][j] {
						t.Fatalf("chunk %d[%d]: got %q want %q", i, j, got[i][j], tc.expect[i][j])
					}
				}
			}
		})
	}
}
