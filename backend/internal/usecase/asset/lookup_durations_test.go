package asset

// CYB-4294: batch asset-duration lookup — usecase tests.
//
// These tests exercise the response-shaping and stats layer on top of a
// synthetic in-memory mock repo (algo_usecase_test.go). The real SQL is
// covered separately in internal/postgres/repos_test.go.

import (
	"context"
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

func TestLookupDurationsDedupsAndReportsMissingAndFilteredOut(t *testing.T) {
	repo := newMockAssetRepo()
	// Three real assets. "aaaaaaaa" and "bbbbbbbb" have direct asset_ids;
	// "cccccccc" is addressable by its grace_video_id.
	must := func(a *models.Asset) {
		if err := repo.Set(context.Background(), a); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	must(&models.Asset{AssetID: "aaaaaaaa", DurationMs: 5_000})
	must(&models.Asset{AssetID: "bbbbbbbb", DurationMs: 900_000})
	must(&models.Asset{
		AssetID:      "cccccccc",
		GraceVideoID: "019f8319-0ef3-7da0-815c-30f23d21e7f1",
		DurationMs:   120_000,
	})

	uc := New(repo)

	// Include a duplicate "aaaaaaaa" (dedup), a whitespace-only entry
	// (dropped), a not-in-DB id (missing), a matching grace_video_id, and
	// bbbbbbbb which will be filtered out by max=600s.
	req := models.AssetDurationsRequest{
		IDs: []string{
			"aaaaaaaa",
			"aaaaaaaa", // dup — should collapse
			" ",        // empty — should drop
			"019f8319-0ef3-7da0-815c-30f23d21e7f1",
			"missing1",
			"bbbbbbbb",
		},
		MinDurationMs: 1_000,
		MaxDurationMs: 600_000,
	}

	resp, err := uc.LookupDurations(context.Background(), req)
	if err != nil {
		t.Fatalf("LookupDurations err: %v", err)
	}

	if got, want := len(resp.Items), 2; got != want {
		t.Fatalf("items len = %d, want %d: %+v", got, want, resp.Items)
	}
	// Items order matches input order (a first, then grace id).
	if resp.Items[0].InputID != "aaaaaaaa" || resp.Items[0].AssetID != "aaaaaaaa" || resp.Items[0].DurationMs != 5_000 {
		t.Fatalf("items[0] = %+v", resp.Items[0])
	}
	if resp.Items[0].Formatted != "5s" {
		t.Fatalf("items[0].Formatted = %q, want 5s", resp.Items[0].Formatted)
	}
	if resp.Items[1].InputID != "019f8319-0ef3-7da0-815c-30f23d21e7f1" || resp.Items[1].AssetID != "cccccccc" {
		t.Fatalf("items[1] = %+v", resp.Items[1])
	}
	if resp.Items[1].DurationSec != 120 {
		t.Fatalf("items[1].DurationSec = %v, want 120", resp.Items[1].DurationSec)
	}
	if resp.Items[1].Formatted != "2m 0s" {
		t.Fatalf("items[1].Formatted = %q, want 2m 0s", resp.Items[1].Formatted)
	}

	if got := resp.MissingIDs; len(got) != 1 || got[0] != "missing1" {
		t.Fatalf("missing_ids = %+v, want [missing1]", got)
	}
	if got := resp.FilteredOutIDs; len(got) != 1 || got[0] != "bbbbbbbb" {
		t.Fatalf("filtered_out_ids = %+v, want [bbbbbbbb]", got)
	}

	// Stats over the 2 items in range: 5000 and 120000.
	if got := resp.Stats.MatchedCount; got != 2 {
		t.Fatalf("MatchedCount = %d, want 2", got)
	}
	if got := resp.Stats.MissingCount; got != 1 {
		t.Fatalf("MissingCount = %d, want 1", got)
	}
	if got := resp.Stats.FilteredOutCount; got != 1 {
		t.Fatalf("FilteredOutCount = %d, want 1", got)
	}
	if got := resp.Stats.TotalMs; got != 125_000 {
		t.Fatalf("TotalMs = %d, want 125000", got)
	}
	if got := resp.Stats.MeanMs; got != 62_500 {
		t.Fatalf("MeanMs = %d, want 62500", got)
	}
	if got := resp.Stats.MinMs; got != 5_000 {
		t.Fatalf("MinMs = %d, want 5000", got)
	}
	if got := resp.Stats.MaxMs; got != 120_000 {
		t.Fatalf("MaxMs = %d, want 120000", got)
	}
	// Two-element percentile: p50 = 5000 + 0.5*(120000-5000) = 62500
	if got := resp.Stats.P50Ms; got != 62_500 {
		t.Fatalf("P50Ms = %d, want 62500", got)
	}
	// p90 = 5000 + 0.9*(120000-5000) = 108500
	if got := resp.Stats.P90Ms; got != 108_500 {
		t.Fatalf("P90Ms = %d, want 108500", got)
	}
}

func TestLookupDurationsEmptyRequestReturnsEmptyResponse(t *testing.T) {
	uc := New(newMockAssetRepo())
	resp, err := uc.LookupDurations(context.Background(), models.AssetDurationsRequest{IDs: nil})
	if err != nil {
		t.Fatalf("empty err: %v", err)
	}
	if len(resp.Items) != 0 || len(resp.MissingIDs) != 0 || len(resp.FilteredOutIDs) != 0 {
		t.Fatalf("empty response non-empty: %+v", resp)
	}
	if resp.Stats.MatchedCount != 0 || resp.Stats.TotalMs != 0 {
		t.Fatalf("empty response stats non-zero: %+v", resp.Stats)
	}
}

func TestLookupDurationsPercentileLinear(t *testing.T) {
	// Standard 5-element percentile check: [0, 25, 50, 75, 100]. p50 exact.
	// p90 = 90th between sorted[3]=75 and sorted[4]=100 with frac 0.6 → 90.
	cases := []struct {
		name    string
		p       float64
		want    int64
		sorted  []int64
	}{
		{"p0", 0.0, 0, []int64{0, 25, 50, 75, 100}},
		{"p50", 0.5, 50, []int64{0, 25, 50, 75, 100}},
		{"p90", 0.9, 90, []int64{0, 25, 50, 75, 100}},
		{"p100", 1.0, 100, []int64{0, 25, 50, 75, 100}},
		{"single", 0.5, 42, []int64{42}},
		{"two", 0.5, 50, []int64{0, 100}}, // exactly halfway
	}
	for _, c := range cases {
		if got := percentileLinear(c.sorted, c.p); got != c.want {
			t.Fatalf("%s: got %d, want %d", c.name, got, c.want)
		}
	}
}

func TestFormatDurationMs(t *testing.T) {
	cases := []struct {
		ms   int64
		want string
	}{
		{0, "0s"},
		{-1, "0s"},
		{500, "0s"},        // rounds down to 0s
		{1_000, "1s"},
		{45_000, "45s"},
		{60_000, "1m 0s"},
		{90_000, "1m 30s"}, // 1m 30s
		{3_599_000, "59m 59s"},
		{3_600_000, "1h 0m"},
		{5_400_000, "1h 30m"}, // 1h 30m
	}
	for _, c := range cases {
		if got := FormatDurationMs(c.ms); got != c.want {
			t.Fatalf("FormatDurationMs(%d) = %q, want %q", c.ms, got, c.want)
		}
	}
}

// Regression: a row addressable by both its asset_id AND its grace_video_id
// in the same request must collapse to a single item. The winning input_id
// is whichever appeared first in the request.
func TestLookupDurationsCollapsesDoubleAddressing(t *testing.T) {
	repo := newMockAssetRepo()
	if err := repo.Set(context.Background(), &models.Asset{
		AssetID:      "aaaaaaaa",
		GraceVideoID: "019f8319-0ef3-7da0-815c-30f23d21e7f1",
		DurationMs:   30_000,
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	uc := New(repo)
	resp, err := uc.LookupDurations(context.Background(), models.AssetDurationsRequest{
		IDs: []string{"019f8319-0ef3-7da0-815c-30f23d21e7f1", "aaaaaaaa"},
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("items = %+v, want 1 collapsed item", resp.Items)
	}
	if resp.Items[0].InputID != "019f8319-0ef3-7da0-815c-30f23d21e7f1" {
		t.Fatalf("InputID = %q, want the first-appearing input", resp.Items[0].InputID)
	}
	if len(resp.MissingIDs) != 0 {
		t.Fatalf("missing = %+v, want empty (second input hit same row)", resp.MissingIDs)
	}
}
