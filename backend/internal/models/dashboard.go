package models

// CYB-4303: dashboard duration-distribution.
//
// Shape mirrors the OpenSpec response contract exactly. Buckets always
// include all rows (CYB-4338: 10), in fixed order — the repo layer fills
// missing rows with (count=0, total_ms=0) so clients never reason about gaps.

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
// a deterministic order.
//
// CYB-4338: refined 5 → 10 buckets ("前密后疏" plan A). The old coarse buckets
// (1-10min / 10-30min / 30-60min) collapsed the corpus into three slabs and
// hid its real shape — a spike at 1-5min and another right at the 30min edge.
// High-frequency region (1-30min) is now split at 5min steps; the long tail
// (30-60min) at 15min steps. Short labels (`<1m`, `1-5m`, …) keep the widened
// x-axis from crowding. NOTE: this histogram now has finer granularity than
// CYB-4294's batch-lookup 5-bucket view (intentional — the fleet dashboard
// wants the detail; the per-asset lookup keeps the coarse buckets).
var DurationBucketOrder = []DurationBucket{
	{Label: "<1m", LoMs: 0, HiMs: int64Ptr(60_000)},
	{Label: "1-5m", LoMs: 60_000, HiMs: int64Ptr(300_000)},
	{Label: "5-10m", LoMs: 300_000, HiMs: int64Ptr(600_000)},
	{Label: "10-15m", LoMs: 600_000, HiMs: int64Ptr(900_000)},
	{Label: "15-20m", LoMs: 900_000, HiMs: int64Ptr(1_200_000)},
	{Label: "20-25m", LoMs: 1_200_000, HiMs: int64Ptr(1_500_000)},
	{Label: "25-30m", LoMs: 1_500_000, HiMs: int64Ptr(1_800_000)},
	{Label: "30-45m", LoMs: 1_800_000, HiMs: int64Ptr(2_700_000)},
	{Label: "45-60m", LoMs: 2_700_000, HiMs: int64Ptr(3_600_000)},
	{Label: "60m+", LoMs: 3_600_000, HiMs: nil},
}

func int64Ptr(v int64) *int64 { return &v }
