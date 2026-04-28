package models

import (
	"encoding/json"
	"time"
)

// AssetTag represents a row in the asset_tags projection table.
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
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// AssetAlgoLatest represents a row in the asset_algo_latest projection table.
type AssetAlgoLatest struct {
	AssetID     string    `json:"asset_id"`
	AlgoName    string    `json:"algo_name"`
	AlgoVersion string    `json:"algo_version"`
	Status      string    `json:"status"`
	RunID       string    `json:"run_id,omitempty"`
	TenantID    string    `json:"tenant_id,omitempty"`
	ProjectID   string    `json:"project_id,omitempty"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AssetEvent represents a row in the asset_events outbox table.
type AssetEvent struct {
	EventID              string          `json:"event_id"`
	EventSeq             int64           `json:"event_seq"`
	EventType            string          `json:"event_type"`
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
