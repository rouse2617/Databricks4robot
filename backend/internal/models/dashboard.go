package models

// CYB-4303: dashboard duration-distribution.
//
// Shape mirrors the OpenSpec response contract exactly. Buckets always
// include all 5 rows, in fixed order — the repo layer fills missing rows
// with (count=0, total_ms=0) so clients never have to reason about gaps.

// DurationBucket is one row of the histogram. `HiMs` is nil for the top
// (open-ended) bucket and serializes as `"hi_ms": null`; every other bucket
// has a finite upper bound. Boundaries match CYB-4294 exactly.
type DurationBucket struct {
	Label   string `json:"label"`
	LoMs    int64  `json:"lo_ms"`
	HiMs    *int64 `json:"hi_ms"`
	Count   int64  `json:"count"`
	TotalMs int64  `json:"total_ms"`
}

// DurationDistribution is the response of GET /dashboard/duration-distribution.
// `AssetType` echoes the query param, nil when absent.
type DurationDistribution struct {
	AssetType   *string          `json:"asset_type"`
	Buckets     []DurationBucket `json:"buckets"`
	TotalAssets int64            `json:"total_assets"`
	TotalMs     int64            `json:"total_ms"`
	MeanMs      int64            `json:"mean_ms"`
	MinMs       int64            `json:"min_ms"`
	MaxMs       int64            `json:"max_ms"`
	P50Ms       int64            `json:"p50_ms"`
	P90Ms       int64            `json:"p90_ms"`
}

// DurationBucketOrder is the canonical ordering used by both the repo (when
// filling zero rows) and any client-side layout that wants to render bars in
// a deterministic order. Keep in lockstep with CYB-4294's frontend buckets.
var DurationBucketOrder = []DurationBucket{
	{Label: "<1min", LoMs: 0, HiMs: int64Ptr(60_000)},
	{Label: "1-10min", LoMs: 60_000, HiMs: int64Ptr(600_000)},
	{Label: "10-30min", LoMs: 600_000, HiMs: int64Ptr(1_800_000)},
	{Label: "30-60min", LoMs: 1_800_000, HiMs: int64Ptr(3_600_000)},
	{Label: "60min+", LoMs: 3_600_000, HiMs: nil},
}

func int64Ptr(v int64) *int64 { return &v }
