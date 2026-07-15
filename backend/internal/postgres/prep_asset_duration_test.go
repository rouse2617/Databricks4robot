package postgres

import (
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// CYB-3226: prepAssetForWrite must persist a correct duration_ms regardless of
// whether the caller provided ms, seconds, or neither. The seconds-only case is
// the segment bug: SyncLegacyFields() must not clobber it back to zero.
func TestPrepAssetForWrite_DerivesDurationMs(t *testing.T) {
	const nsPerSec = int64(1_000_000_000)
	cases := []struct {
		name        string
		durationMs  int64
		durationSec float64
		startNs     int64
		endNs       int64
		wantMs      int64
	}{
		{
			name:        "seconds only (segment path)",
			durationSec: 68.5,
			startNs:     1,
			endNs:       68*nsPerSec + 500*int64(1_000_000),
			wantMs:      68500,
		},
		{
			name:    "span only, no duration fields",
			startNs: 1_000_000_000,
			endNs:   1_000_000_000 + 23_000*int64(1_000_000), // +23s
			wantMs:  23_000,
		},
		{
			name:       "duration_ms already set (child path) is kept",
			durationMs: 12500,
			startNs:    0,
			endNs:      999 * nsPerSec, // span disagrees; explicit ms wins
			wantMs:     12500,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := &models.Asset{
				DurationMs:       tc.durationMs,
				DurationSec:      tc.durationSec,
				StartTimestampNs: tc.startNs,
				EndTimestampNs:   tc.endNs,
			}
			prepAssetForWrite(a)
			if a.DurationMs != tc.wantMs {
				t.Fatalf("DurationMs = %d, want %d", a.DurationMs, tc.wantMs)
			}
			// duration_sec must stay consistent with the canonical ms value.
			if wantSec := float64(a.DurationMs) / 1000.0; a.DurationSec != wantSec {
				t.Fatalf("DurationSec = %v, want %v (ms/1000)", a.DurationSec, wantSec)
			}
		})
	}
}
