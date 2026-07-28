package models

import "time"

// CYB-4306: batch asset cost / gpu-min lookup — request/response types for
// POST /api/v1/assets/costs. See openspec/changes/CYB-4306-costs.
//
// Kept in a dedicated file (not asset.go) so the batch-lookup surface stays
// organized per feature — matches the durations types living beside their
// own doc comment block in asset.go but keeps this new attack surface easy
// to grep by ticket id.

// AssetCostsRequest is the body accepted by POST /api/v1/assets/costs.
//
// `ids` may contain any mix of asset_id (8-char) and grace_video_id (uuid);
// `id_type` is a UI hint — the server always matches both columns. Both
// `start_at` and `end_at` are required (RFC3339); the window is capped at
// 90 days by the handler (see decisions.md).
type AssetCostsRequest struct {
	IDs     []string  `json:"ids"`
	IDType  string    `json:"id_type,omitempty"`
	StartAt time.Time `json:"start_at"`
	EndAt   time.Time `json:"end_at"`
	// GroupBy in {"", "asset", "asset_algo"} — empty defaults to "asset".
	GroupBy string `json:"group_by,omitempty"`
}

// AssetCostByAlgo is one algo-level breakdown row inside an AssetCostItem
// when the request asked for group_by=asset_algo. `AlgoKey` is Argo's
// pipeline_run_nodes.template_name (closest stable identifier that
// aggregates across runs).
type AssetCostByAlgo struct {
	AlgoKey  string  `json:"algo_key"`
	CostUSD  float64 `json:"cost_usd"`
	GPUSec   float64 `json:"gpu_sec"`
	CPUSec   float64 `json:"cpu_sec"`
	RunCount int64   `json:"run_count"`
}

// AssetCostItem is one row of the response `items` array. When
// group_by=asset_algo, the item-level totals still equal the sum of the
// ByAlgo entries — callers can display the summary + drill-down side by
// side without a second call.
//
// ByAlgo is a *pointer* so it is emitted as JSON `null` (not `[]`) when
// group_by=asset — callers can branch with `if item.by_algo`.
type AssetCostItem struct {
	InputID       string             `json:"input_id"`
	AssetID       string             `json:"asset_id"`
	GraceVideoID  string             `json:"grace_video_id,omitempty"`
	TotalCostUSD  float64            `json:"total_cost_usd"`
	GPUSec        float64            `json:"gpu_sec"`
	CPUSec        float64            `json:"cpu_sec"`
	GPUMin        float64            `json:"gpu_min"`
	CPUMin        float64            `json:"cpu_min"`
	RunCount      int64              `json:"run_count"`
	ByAlgo        *[]AssetCostByAlgo `json:"by_algo"`
}

// AssetCostStats are computed server-side over the response `items` array
// (matched assets only). Percentiles use linear interpolation on
// sorted-ascending total_cost_usd. Every field is zero when items empty
// (NOT null / NaN) so callers can render numeric widgets unconditionally.
type AssetCostStats struct {
	MatchedCount     int     `json:"matched_count"`
	MissingCount     int     `json:"missing_count"`
	FilteredOutCount int     `json:"filtered_out_count"`
	TotalCostUSD     float64 `json:"total_cost_usd"`
	MeanCostUSD      float64 `json:"mean_cost_usd"`
	P50CostUSD       float64 `json:"p50_cost_usd"`
	P90CostUSD       float64 `json:"p90_cost_usd"`
	TotalGPUSec      float64 `json:"total_gpu_sec"`
	TotalCPUSec      float64 `json:"total_cpu_sec"`
	TotalRunCount    int64   `json:"total_run_count"`
}

// AssetCostsResponse is the body returned by POST /api/v1/assets/costs.
type AssetCostsResponse struct {
	Items          []AssetCostItem `json:"items"`
	MissingIDs     []string        `json:"missing_ids"`
	FilteredOutIDs []string        `json:"filtered_out_ids"`
	Stats          AssetCostStats  `json:"stats"`
}
