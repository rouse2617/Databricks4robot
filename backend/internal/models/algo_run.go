package models

import "time"

// Algo run status values (algo_runs.status).
const (
	AlgoRunStatusPending   = "pending"
	AlgoRunStatusRunning   = "running"
	AlgoRunStatusOK        = "ok"
	AlgoRunStatusFailed    = "failed"
	AlgoRunStatusCancelled = "cancelled"
)

// AlgoRun is a first-class execution-event row (CYB-1018).
type AlgoRun struct {
	RunID           string                 `json:"run_id"`
	AlgoName        string                 `json:"algo_name"`
	AlgoVersion     string                 `json:"algo_version"`
	AlgoKind        string                 `json:"algo_kind"`
	TriggeredBy     string                 `json:"triggered_by"`
	Status          string                 `json:"status"`
	StartedAt       *time.Time             `json:"started_at,omitempty"`
	FinishedAt      *time.Time             `json:"finished_at,omitempty"`
	DurationNs      *int64                 `json:"duration_ns,omitempty"`
	InputFilter     map[string]interface{} `json:"input_filter,omitempty"`
	InputAssetIDs   []string               `json:"input_asset_ids,omitempty"`
	Params          map[string]interface{} `json:"params,omitempty"`
	CodeCommit      string                 `json:"code_commit,omitempty"`
	ImageDigest     string                 `json:"image_digest,omitempty"`
	PipelineName    string                 `json:"pipeline_name,omitempty"`
	PipelineVersion string                 `json:"pipeline_version,omitempty"`
	AssetsProcessed *int                   `json:"assets_processed,omitempty"`
	AssetsSucceeded *int                   `json:"assets_succeeded,omitempty"`
	AssetsFailed    *int                   `json:"assets_failed,omitempty"`
	ActionsCreated  *int                   `json:"actions_created,omitempty"`
	MetricsWritten  *int                   `json:"metrics_written,omitempty"`
	Outputs         map[string]interface{} `json:"outputs,omitempty"`
	CPUSeconds      *int64                 `json:"cpu_seconds,omitempty"`
	GPUSeconds      *int64                 `json:"gpu_seconds,omitempty"`
	CostUSDMicros   *int64                 `json:"cost_usd_micros,omitempty"`
	ErrorClass      string                 `json:"error_class,omitempty"`
	ErrorMessage    string                 `json:"error_message,omitempty"`
	TenantID        string                 `json:"tenant_id,omitempty"`
	ProjectID       string                 `json:"project_id,omitempty"`
	ExternalRuntime *string                `json:"external_runtime,omitempty"`
	ExternalUrl     *string                `json:"external_url,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	RowVersion      int64                  `json:"row_version"`
}
