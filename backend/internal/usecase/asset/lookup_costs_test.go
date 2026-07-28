package asset

// CYB-4306: batch asset cost lookup — usecase tests.
//
// These tests exercise the resolve / aggregate / stats layer on top of the
// in-memory mockAssetRepo (algo_usecase_test.go). Repo-level SQL is covered
// separately in internal/postgres/repos_test.go.

import (
	"context"
	"testing"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

func TestLookupCostsDedupsResolvesAndComputesStats(t *testing.T) {
	repo := newMockAssetRepo()
	must := func(a *models.Asset) {
		if err := repo.Set(context.Background(), a); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	must(&models.Asset{AssetID: "aaaaaaaa"})
	must(&models.Asset{AssetID: "bbbbbbbb"})
	must(&models.Asset{
		AssetID:      "cccccccc",
		GraceVideoID: "019f8319-0ef3-7da0-815c-30f23d21e7f1",
	})
	must(&models.Asset{AssetID: "dddddddd"}) // exists but no cost rows

	// Repo returns rows for aaaaaaaa (2 runs), cccccccc (1 run), bbbbbbbb (1 run).
	// dddddddd resolves but has no cost row → filtered_out.
	// missing1 does not resolve → missing_ids.
	repo.lookupCostsFn = func(ctx context.Context, assetIDs []string, startAt, endAt time.Time, byAlgo bool) ([]repository.AssetCostRow, error) {
		want := map[string]struct{}{}
		for _, a := range assetIDs {
			want[a] = struct{}{}
		}
		out := []repository.AssetCostRow{}
		if _, ok := want["aaaaaaaa"]; ok {
			out = append(out, repository.AssetCostRow{
				AssetID: "aaaaaaaa", TotalCostUSD: 2.5, GPUSec: 120, CPUSec: 60, RunCount: 2,
			})
		}
		if _, ok := want["bbbbbbbb"]; ok {
			out = append(out, repository.AssetCostRow{
				AssetID: "bbbbbbbb", TotalCostUSD: 0.10, GPUSec: 0, CPUSec: 20, RunCount: 1,
			})
		}
		if _, ok := want["cccccccc"]; ok {
			out = append(out, repository.AssetCostRow{
				AssetID: "cccccccc", TotalCostUSD: 1.20, GPUSec: 60, CPUSec: 30, RunCount: 1,
			})
		}
		return out, nil
	}

	uc := New(repo)
	start := time.Date(2026, 7, 21, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC)
	req := models.AssetCostsRequest{
		IDs: []string{
			"aaaaaaaa",
			"aaaaaaaa", // dup — should collapse
			" ",        // empty — should drop
			"019f8319-0ef3-7da0-815c-30f23d21e7f1",
			"missing1",
			"bbbbbbbb",
			"dddddddd", // resolves but no cost rows → filtered_out
		},
		StartAt: start,
		EndAt:   end,
	}
	resp, err := uc.LookupCosts(context.Background(), req)
	if err != nil {
		t.Fatalf("LookupCosts err: %v", err)
	}

	if got := len(resp.Items); got != 3 {
		t.Fatalf("items len = %d, want 3: %+v", got, resp.Items)
	}
	// Items order matches input order.
	if resp.Items[0].InputID != "aaaaaaaa" || resp.Items[0].AssetID != "aaaaaaaa" || resp.Items[0].TotalCostUSD != 2.5 {
		t.Fatalf("items[0] = %+v", resp.Items[0])
	}
	if resp.Items[0].GPUMin != 2 || resp.Items[0].CPUMin != 1 {
		t.Fatalf("items[0] gpu_min/cpu_min = %v/%v, want 2/1", resp.Items[0].GPUMin, resp.Items[0].CPUMin)
	}
	if resp.Items[0].ByAlgo != nil {
		t.Fatalf("items[0].ByAlgo should be nil in asset mode, got %+v", resp.Items[0].ByAlgo)
	}
	if resp.Items[1].InputID != "019f8319-0ef3-7da0-815c-30f23d21e7f1" || resp.Items[1].AssetID != "cccccccc" {
		t.Fatalf("items[1] = %+v", resp.Items[1])
	}
	if resp.Items[2].InputID != "bbbbbbbb" || resp.Items[2].TotalCostUSD != 0.10 {
		t.Fatalf("items[2] = %+v", resp.Items[2])
	}

	if got := resp.MissingIDs; len(got) != 1 || got[0] != "missing1" {
		t.Fatalf("missing_ids = %+v", got)
	}
	if got := resp.FilteredOutIDs; len(got) != 1 || got[0] != "dddddddd" {
		t.Fatalf("filtered_out_ids = %+v", got)
	}

	// Stats over sorted costs [0.10, 1.20, 2.50]: total=3.80, mean≈1.266..., p50=1.20, p90=2.24.
	s := resp.Stats
	if s.MatchedCount != 3 || s.MissingCount != 1 || s.FilteredOutCount != 1 {
		t.Fatalf("counts = %d/%d/%d", s.MatchedCount, s.MissingCount, s.FilteredOutCount)
	}
	if got, want := round(s.TotalCostUSD, 4), 3.80; got != want {
		t.Fatalf("total_cost = %v, want %v", got, want)
	}
	if got := round(s.MeanCostUSD, 4); got != 1.2667 {
		t.Fatalf("mean_cost = %v, want 1.2667", got)
	}
	if got := round(s.P50CostUSD, 4); got != 1.20 {
		t.Fatalf("p50_cost = %v, want 1.20", got)
	}
	if got := round(s.P90CostUSD, 4); got != 2.24 {
		// p90 = sorted[1] + 0.8*(sorted[2]-sorted[1]) = 1.20 + 0.8*(2.50-1.20) = 2.24
		t.Fatalf("p90_cost = %v, want 2.24", got)
	}
	if s.TotalGPUSec != 180 {
		t.Fatalf("total_gpu_sec = %v, want 180", s.TotalGPUSec)
	}
	if s.TotalCPUSec != 110 {
		t.Fatalf("total_cpu_sec = %v, want 110", s.TotalCPUSec)
	}
	if s.TotalRunCount != 4 {
		t.Fatalf("total_run_count = %v, want 4", s.TotalRunCount)
	}
}

func TestLookupCostsEmptyRequestReturnsEmptyResponse(t *testing.T) {
	uc := New(newMockAssetRepo())
	resp, err := uc.LookupCosts(context.Background(), models.AssetCostsRequest{
		IDs:     nil,
		StartAt: time.Now().Add(-24 * time.Hour),
		EndAt:   time.Now(),
	})
	if err != nil {
		t.Fatalf("empty err: %v", err)
	}
	if len(resp.Items) != 0 || len(resp.MissingIDs) != 0 || len(resp.FilteredOutIDs) != 0 {
		t.Fatalf("empty response non-empty: %+v", resp)
	}
	// All percentile / total fields must be 0 (not NaN) so JSON marshals cleanly.
	if resp.Stats.MatchedCount != 0 || resp.Stats.TotalCostUSD != 0 || resp.Stats.P50CostUSD != 0 || resp.Stats.P90CostUSD != 0 {
		t.Fatalf("empty response stats non-zero: %+v", resp.Stats)
	}
}

func TestLookupCostsByAlgoGroupsBreakdown(t *testing.T) {
	repo := newMockAssetRepo()
	if err := repo.Set(context.Background(), &models.Asset{AssetID: "aaaaaaaa"}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	repo.lookupCostsFn = func(_ context.Context, _ []string, _ time.Time, _ time.Time, byAlgo bool) ([]repository.AssetCostRow, error) {
		if !byAlgo {
			t.Fatalf("repo LookupCosts was called with byAlgo=false, want true")
		}
		return []repository.AssetCostRow{
			// Unordered on purpose — usecase must sort by cost desc so the UI
			// gets "the expensive step" first.
			{AssetID: "aaaaaaaa", AlgoKey: "encode-video", TotalCostUSD: 0.30, GPUSec: 30, CPUSec: 10, RunCount: 1},
			{AssetID: "aaaaaaaa", AlgoKey: "extract-frames", TotalCostUSD: 0.90, GPUSec: 90, CPUSec: 20, RunCount: 2},
		}, nil
	}
	uc := New(repo)
	resp, err := uc.LookupCosts(context.Background(), models.AssetCostsRequest{
		IDs:     []string{"aaaaaaaa"},
		StartAt: time.Date(2026, 7, 21, 0, 0, 0, 0, time.UTC),
		EndAt:   time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC),
		GroupBy: "asset_algo",
	})
	if err != nil {
		t.Fatalf("LookupCosts err: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("items = %+v, want 1", resp.Items)
	}
	item := resp.Items[0]
	if item.TotalCostUSD != 1.20 || item.RunCount != 3 || item.GPUSec != 120 || item.CPUSec != 30 {
		t.Fatalf("rolled-up totals wrong: %+v", item)
	}
	if item.ByAlgo == nil || len(*item.ByAlgo) != 2 {
		t.Fatalf("by_algo missing or wrong len: %+v", item.ByAlgo)
	}
	byAlgo := *item.ByAlgo
	if byAlgo[0].AlgoKey != "extract-frames" || byAlgo[0].CostUSD != 0.90 {
		t.Fatalf("by_algo[0] should be the expensive step first: %+v", byAlgo[0])
	}
	if byAlgo[1].AlgoKey != "encode-video" || byAlgo[1].CostUSD != 0.30 {
		t.Fatalf("by_algo[1] = %+v", byAlgo[1])
	}
}

// A resolved asset that has no rows in the window must land in
// filtered_out_ids (NOT missing_ids), and its input_id must be preserved.
func TestLookupCostsFilteredOutDistinctFromMissing(t *testing.T) {
	repo := newMockAssetRepo()
	if err := repo.Set(context.Background(), &models.Asset{AssetID: "aaaaaaaa"}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	repo.lookupCostsFn = func(context.Context, []string, time.Time, time.Time, bool) ([]repository.AssetCostRow, error) {
		return nil, nil
	}
	uc := New(repo)
	resp, err := uc.LookupCosts(context.Background(), models.AssetCostsRequest{
		IDs:     []string{"aaaaaaaa", "unknownid"},
		StartAt: time.Date(2026, 7, 21, 0, 0, 0, 0, time.UTC),
		EndAt:   time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(resp.Items) != 0 {
		t.Fatalf("items should be empty when no cost rows: %+v", resp.Items)
	}
	if len(resp.FilteredOutIDs) != 1 || resp.FilteredOutIDs[0] != "aaaaaaaa" {
		t.Fatalf("filtered_out = %+v, want [aaaaaaaa]", resp.FilteredOutIDs)
	}
	if len(resp.MissingIDs) != 1 || resp.MissingIDs[0] != "unknownid" {
		t.Fatalf("missing = %+v, want [unknownid]", resp.MissingIDs)
	}
}

// round to `places` decimals for stat comparison — the linear interpolation
// on floats otherwise yields values like 1.2666666666666666 that are ugly to
// spell in the test.
func round(v float64, places int) float64 {
	mul := 1.0
	for i := 0; i < places; i++ {
		mul *= 10
	}
	if v >= 0 {
		return float64(int64(v*mul+0.5)) / mul
	}
	return float64(int64(v*mul-0.5)) / mul
}
