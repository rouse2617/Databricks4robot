package models

import (
	"encoding/json"
	"time"
)

// AssetTag represents a row in the asset_tags projection table.
//
// As of CYB-1015 multiple rows may share the same (asset_id, tag_key) as
// long as (source_type, source_version) differs. AppliedAt is the moment
// the source declared this assertion; CreatedAt is the row's first insert.
type AssetTag struct {
	AssetID       string    `json:"asset_id"`
	TagKey        string    `json:"tag_key"`
	TagValue      string    `json:"tag_value"`
	TagType       string    `json:"tag_type"`
	SourceType    string    `json:"source_type"`
	SourceName    string    `json:"source_name,omitempty"`
	SourceVersion string    `json:"source_version,omitempty"`
	RunID         string    `json:"run_id,omitempty"`
	TenantID      string    `json:"tenant_id,omitempty"`
	ProjectID     string    `json:"project_id,omitempty"`
	AppliedAt     time.Time `json:"applied_at"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// AssetAlgoLatest represents a row in the asset_algo_latest projection table.
// Concurrency contract: this table is the source of truth for per-algorithm
// state on an asset. The (asset_id, algo_name) primary key plus a monotonic
// guard on algo_version (`WHERE existing.algo_version <= EXCLUDED.algo_version`)
// makes concurrent algo finishes safe without touching assets.version.
type AssetAlgoLatest struct {
	AssetID       string                 `json:"asset_id"`
	AlgoName      string                 `json:"algo_name"`
	AlgoVersion   string                 `json:"algo_version"`
	Status        string                 `json:"status"`
	ResultTag     string                 `json:"result_tag,omitempty"`
	ResultScore   *float64               `json:"result_score,omitempty"`
	ResultSummary map[string]interface{} `json:"result_summary,omitempty"`
	RunID         string                 `json:"run_id,omitempty"`
	Method        string                 `json:"method,omitempty"`
	ModelURI      string                 `json:"model_uri,omitempty"`
	OutputURI     string                 `json:"output_uri,omitempty"`
	ErrorCode     string                 `json:"error_code,omitempty"`
	ErrorMessage  string                 `json:"error_message,omitempty"`
	StartedAt     *time.Time             `json:"started_at,omitempty"`
	FinishedAt    *time.Time             `json:"finished_at,omitempty"`
	TenantID      string                 `json:"tenant_id,omitempty"`
	ProjectID     string                 `json:"project_id,omitempty"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// AssetEvent represents a row in the asset_events outbox table.
type AssetEvent struct {
	EventID              string          `json:"event_id"`
	EventSeq             int64           `json:"event_seq"`
	EventType            string          `json:"event_type"`
	AggregateType        string          `json:"aggregate_type"`
	PayloadSchemaVersion string          `json:"payload_schema_version"`
	AssetID              string          `json:"asset_id,omitempty"`
	McapFileID           string          `json:"mcap_file_id,omitempty"`
	TenantID             string          `json:"tenant_id,omitempty"`
	ProjectID            string          `json:"project_id,omitempty"`
	EventSource          string          `json:"event_source"`
	PublishState         string          `json:"publish_state"`
	EventPayload         json.RawMessage `json:"event_payload"`
	RetryCount           int             `json:"retry_count"`
	LastError            string          `json:"last_error,omitempty"`
	OccurredAt           time.Time       `json:"occurred_at"`
	CreatedAt            time.Time       `json:"created_at"`
	PublishedAt          *time.Time      `json:"published_at,omitempty"`
}
