package manifest

import (
	"testing"

	"github.com/foxglove/mcap/go/mcap"
)

func TestNormalizeWindow_PicksMillisecondsAgainstStats(t *testing.T) {
	info := testInfoWithStats(1_700_000_000_000_000_000, 1_700_000_100_000_000_000)
	start, end, mode := NormalizeWindow(1_700_000_000_000, 1_700_000_050_000, info)
	if mode != "locator_ms" {
		t.Fatalf("mode=%q want locator_ms", mode)
	}
	if start != 1_700_000_000_000_000_000 || end != 1_700_000_050_000_000_000 {
		t.Fatalf("start/end mismatch: %d %d", start, end)
	}
}

func TestNormalizeWindow_FallsBackWhenNoOverlap(t *testing.T) {
	info := testInfoWithStats(1_000_000_000, 2_000_000_000)
	start, end, mode := NormalizeWindow(10_000, 20_000, info)
	if mode != "stats_fallback_mismatch" {
		t.Fatalf("mode=%q", mode)
	}
	if start != 1_000_000_000 || end != 2_000_000_000 {
		t.Fatalf("start/end mismatch: %d %d", start, end)
	}
}

func TestNormalizeWindow_DoesNotOverflowSecondsCandidate(t *testing.T) {
	// raw values are in milliseconds; seconds conversion would overflow uint64.
	info := testInfoWithStats(1_700_000_000_000_000_000, 1_700_000_100_000_000_000)
	start, end, mode := NormalizeWindow(1_700_000_000_000, 1_700_000_050_000, info)
	if mode != "locator_ms" {
		t.Fatalf("mode=%q want locator_ms", mode)
	}
	if start != 1_700_000_000_000_000_000 || end != 1_700_000_050_000_000_000 {
		t.Fatalf("start/end mismatch: %d %d", start, end)
	}
}

func testInfoWithStats(start, end uint64) *mcap.Info {
	return &mcap.Info{
		Statistics: &mcap.Statistics{
			MessageStartTime: start,
			MessageEndTime:   end,
		},
	}
}

