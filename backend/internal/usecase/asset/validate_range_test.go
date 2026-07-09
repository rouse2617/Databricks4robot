package asset

import (
	"errors"
	"testing"
)

// CYB-3226: shared write-time time-range validation.
func TestValidateAssetTimeRange(t *testing.T) {
	const ms = int64(1_000_000) // 1ms in ns
	cases := []struct {
		name       string
		start, end int64
		want       error
	}{
		{"valid 1s span", 0, 1_000_000_000, nil},
		{"valid exactly 1ms", 0, ms, nil},
		{"inverted end<start", 100, 50, ErrInvalidRange},
		{"empty end==start", 100, 100, ErrInvalidRange},
		{"sub-ms span rounds to 0", 0, ms - 1, ErrDurationTooSmall},
		{"one ns span", 5, 6, ErrDurationTooSmall},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := validateAssetTimeRange(tc.start, tc.end)
			if !errors.Is(got, tc.want) {
				t.Fatalf("validateAssetTimeRange(%d,%d) = %v, want %v", tc.start, tc.end, got, tc.want)
			}
		})
	}
}
