package searchindex

import (
	"context"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

// AlgoRunBuilder loads algo_runs rows from PG and produces ES _source-shaped
// maps for the "algo_runs" index.
type AlgoRunBuilder struct {
	Runs repository.AlgoRunRepository
}

// Build returns the Elasticsearch document for an algo run.
// If the run is not found, ok is false (caller should delete the ES doc).
func (b *AlgoRunBuilder) Build(ctx context.Context, runID string) (doc map[string]any, ok bool, err error) {
	run, err := b.Runs.Get(ctx, runID)
	if err != nil {
		return nil, false, err
	}
	if run == nil {
		return nil, false, nil
	}

	doc = map[string]any{
		"run_id":       run.RunID,
		"algo_name":    run.AlgoName,
		"algo_version": run.AlgoVersion,
		"algo_kind":    run.AlgoKind,
		"triggered_by": run.TriggeredBy,
		"status":       run.Status,
		"row_version":  run.RowVersion,
	}

	if !run.CreatedAt.IsZero() {
		doc["created_at"] = run.CreatedAt.UTC().Format(time.RFC3339Nano)
	}
	if !run.UpdatedAt.IsZero() {
		doc["updated_at"] = run.UpdatedAt.UTC().Format(time.RFC3339Nano)
	}
	if run.StartedAt != nil {
		doc["started_at"] = run.StartedAt.UTC().Format(time.RFC3339Nano)
	}
	if run.FinishedAt != nil {
		doc["finished_at"] = run.FinishedAt.UTC().Format(time.RFC3339Nano)
	}
	if run.DurationNs != nil {
		doc["duration_ns"] = *run.DurationNs
	}

	if run.TenantID != "" {
		doc["tenant_id"] = run.TenantID
	}
	if run.ProjectID != "" {
		doc["project_id"] = run.ProjectID
	}
	if run.PipelineName != "" {
		doc["pipeline_name"] = run.PipelineName
	}
	if run.PipelineVersion != "" {
		doc["pipeline_version"] = run.PipelineVersion
	}
	if run.CodeCommit != "" {
		doc["code_commit"] = run.CodeCommit
	}
	if run.ImageDigest != "" {
		doc["image_digest"] = run.ImageDigest
	}

	// Summary counters.
	if run.AssetsProcessed != nil {
		doc["assets_processed"] = *run.AssetsProcessed
	}
	if run.AssetsSucceeded != nil {
		doc["assets_succeeded"] = *run.AssetsSucceeded
	}
	if run.AssetsFailed != nil {
		doc["assets_failed"] = *run.AssetsFailed
	}
	if run.ActionsCreated != nil {
		doc["actions_created"] = *run.ActionsCreated
	}
	if run.MetricsWritten != nil {
		doc["metrics_written"] = *run.MetricsWritten
	}

	// Resource usage.
	if run.CPUSeconds != nil {
		doc["cpu_seconds"] = *run.CPUSeconds
	}
	if run.GPUSeconds != nil {
		doc["gpu_seconds"] = *run.GPUSeconds
	}
	if run.CostUSDMicros != nil {
		doc["cost_usd_micros"] = *run.CostUSDMicros
	}

	// Error fields (only for failed/cancelled runs).
	if run.ErrorClass != "" {
		doc["error_class"] = run.ErrorClass
	}
	if run.ErrorMessage != "" {
		doc["error_message"] = run.ErrorMessage
	}

	// External runtime references.
	if run.ExternalRuntime != nil && *run.ExternalRuntime != "" {
		doc["external_runtime"] = *run.ExternalRuntime
	}
	if run.ExternalUrl != nil && *run.ExternalUrl != "" {
		doc["external_url"] = *run.ExternalUrl
	}

	// Input metadata (flattened for search).
	if run.InputFilter != nil {
		doc["input_filter"] = run.InputFilter
	}
	if len(run.InputAssetIDs) > 0 {
		doc["input_asset_ids"] = run.InputAssetIDs
	}
	if run.Params != nil {
		doc["params"] = run.Params
	}
	if run.Outputs != nil {
		doc["outputs"] = run.Outputs
	}

	return doc, true, nil
}
