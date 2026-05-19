package repository

import "context"

// PurgeCounts reports how many rows were deleted from each affected table
// during a hard-delete operation. Tables that were not touched are reported
// as 0 so callers can render a stable schema.
type PurgeCounts struct {
	AssetMetrics       int64 `json:"asset_metrics"`
	AssetEvalResults   int64 `json:"asset_eval_results"`
	AssetAlgoLatest    int64 `json:"asset_algo_latest"`
	AssetTags          int64 `json:"asset_tags"`
	Actions            int64 `json:"actions"`
	DeliveryItems      int64 `json:"delivery_items"`
	AssetRelations     int64 `json:"asset_relations"`
	AssetEventsByAsset int64 `json:"asset_events_by_asset"`
	Assets             int64 `json:"assets"`
	AssetEventsByMcap  int64 `json:"asset_events_by_mcap"`
	EvalResultsByMcap  int64 `json:"asset_eval_results_by_mcap"`
	McapFiles          int64 `json:"mcap_files"`
}

// BatchOpsRepository provides admin-only hard-delete operations that cut
// across multiple tables. Implementations are responsible for honoring the
// foreign-key topology and running deletions in chunked transactions so a
// single delete does not lock huge ranges of rows for too long.
type BatchOpsRepository interface {
	// ListAssetIDsByImportBatch returns all asset_id values where the
	// metadata->>'import_batch' label matches the given batch string. Order
	// is not specified and not guaranteed across calls.
	ListAssetIDsByImportBatch(ctx context.Context, importBatch string) ([]string, error)

	// ListMcapFileIDsByImportBatch returns all mcap_file_id values where the
	// metadata->>'import_batch' label matches the given batch string.
	ListMcapFileIDsByImportBatch(ctx context.Context, importBatch string) ([]string, error)

	// CountAssetReferencesToMcapFiles returns, for each mcap_file_id in the
	// input, how many rows in the assets table still reference it. Used to
	// safeguard mcap-file deletion: callers must ensure all referencing
	// assets are deleted (or excluded from the purge) first.
	CountAssetReferencesToMcapFiles(ctx context.Context, mcapFileIDs []string) (map[string]int64, error)

	// PurgeAssetsDryRun returns the per-table counts that would be deleted
	// for the given assetIDs. No rows are modified.
	PurgeAssetsDryRun(ctx context.Context, assetIDs []string) (PurgeCounts, error)

	// PurgeAssets performs the actual hard delete for the given assetIDs.
	// Deletions are chunked to chunkSize per batch, each chunk in its own
	// transaction so progress is durable. Returns aggregate counts.
	PurgeAssets(ctx context.Context, assetIDs []string, chunkSize int) (PurgeCounts, error)

	// PurgeMcapFilesDryRun reports per-table counts that would be deleted
	// for the given mcapFileIDs (asset_events / asset_eval_results referring
	// to those mcap files, and the mcap_files rows themselves).
	PurgeMcapFilesDryRun(ctx context.Context, mcapFileIDs []string) (PurgeCounts, error)

	// PurgeMcapFiles hard-deletes the mcap_files rows after first deleting
	// dependent asset_events / asset_eval_results rows that referenced them.
	// Returns an error if any input mcap_file_id is still referenced by
	// the assets table; caller is expected to PurgeAssets first or remove
	// those ids from the input.
	PurgeMcapFiles(ctx context.Context, mcapFileIDs []string, chunkSize int) (PurgeCounts, error)
}
