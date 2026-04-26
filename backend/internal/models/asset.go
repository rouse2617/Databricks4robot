package models

import "time"

// AssetStatus mirrors the Bigtable cf:meta.status values.
type AssetStatus string

const (
	AssetStatusApproved   AssetStatus = "approved"
	AssetStatusRejected   AssetStatus = "rejected"
	AssetStatusSuperseded AssetStatus = "superseded"
	AssetStatusArchived   AssetStatus = "archived"
)

// IngestState mirrors cf:process.ingest_state values on McapFile.
type IngestState string

const (
	IngestStatePending    IngestState = "pending"
	IngestStateSummarized IngestState = "summarized"
	IngestStateFailed     IngestState = "failed"
)

// DeliveryStatus mirrors cf:meta.status values on Delivery.
type DeliveryStatus string

const (
	DeliveryStatusPending   DeliveryStatus = "pending"
	DeliveryStatusDelivered DeliveryStatus = "delivered"
	DeliveryStatusAccepted  DeliveryStatus = "accepted"
	DeliveryStatusRejected  DeliveryStatus = "rejected"
	DeliveryStatusRecalled  DeliveryStatus = "recalled"
)

// ──────────────────────────────────────────────────────────────────────────────
// Asset — row in the `assets` Bigtable table.
// Row key: v1#<asset_id>
// CF layout:
//
//	cf:meta  → scalar metadata fields
//	cf:algo  → algorithm results, key pattern: <algo>@<ver>:<field>
//	cf:tag   → free-form tags, key: tag name, value: tag value
//
// ──────────────────────────────────────────────────────────────────────────────
type Asset struct {
	// Identity
	AssetID    string `json:"asset_id"`
	McapFileID string `json:"mcap_file_id"`

	// Segment locator — deterministic SHA-1 of mcap_file_id + start_ns + end_ns.
	SegmentLocator string `json:"segment_locator,omitempty"`

	// cf:meta — immutable fields (written at creation, never updated afterwards)
	// Phase 1 Bigtable: stay in cf:meta, GC policy maxVersions=1, no TTL.
	StartTimestampNs int64       `json:"start_timestamp_ns"`
	EndTimestampNs   int64       `json:"end_timestamp_ns"`
	DurationSec      float64     `json:"duration_sec"`
	Reviewer         string      `json:"reviewer"`
	Status           AssetStatus `json:"status"` // mutable via QA workflow, but low-frequency
	Owner            string      `json:"owner"`
	SegType          string      `json:"type,omitempty"`
	Env              string      `json:"env,omitempty"`
	Task             string      `json:"task,omitempty"`

	// cf:meta — mutable fields (updated by delivery triggers, lifecycle jobs, API calls)
	// Phase 1 Bigtable: consider splitting into cf:state with separate GC policy.
	LastDeliveredAt *time.Time `json:"last_delivered_at,omitempty"`
	LastDeliveredTo string     `json:"last_delivered_to,omitempty"`
	DeliveryCount   int        `json:"delivery_count"`

	// cf:algo — keyed by "<algo>@<ver>:<field>" (arbitrary map)
	AlgoResults map[string]string `json:"algo_results,omitempty"`

	// cf:tag — free-form tags (arbitrary map)
	Tags map[string]string `json:"tags,omitempty"`

	// cf_files — file reference registry (key=logical name, value=GCS URI)
	Files map[string]string `json:"files,omitempty"`

	// LifecycleMeta holds lifecycle governance fields stored in cf_meta JSONB.
	// These are mutable — updated by lifecycle jobs and finish_algo (total_size_bytes).
	// Keys: retention_tier, archive_after_days, delete_after_days, total_size_bytes, last_accessed_at.
	// Phase 1 Bigtable: move to cf:state alongside delivery summary fields.
	LifecycleMeta map[string]interface{} `json:"lifecycle_meta,omitempty"`

	// Timestamps / versioning
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Version   int64     `json:"version"`
}

// ──────────────────────────────────────────────────────────────────────────────
// McapFile — row in the `mcap_files` Bigtable table.
// Row key: v1#<mcap_file_id>
// CF layout:
//
//	cf:meta    → scalar metadata
//	cf:process → per-algorithm processing state
//
// ──────────────────────────────────────────────────────────────────────────────
type McapFile struct {
	McapFileID string `json:"mcap_file_id"`

	// cf:meta
	GCSPath          string      `json:"gcs_path"`
	SizeBytes        int64       `json:"size_bytes"`
	RawHashMD5       string      `json:"raw_hash_md5"`
	IngestState      IngestState `json:"ingest_state"`
	StartTimestampNs int64       `json:"start_timestamp_ns,omitempty"`
	EndTimestampNs   int64       `json:"end_timestamp_ns,omitempty"`
	ChannelCount     int         `json:"channel_count,omitempty"`
	ChunkCount       int         `json:"chunk_count,omitempty"`
	Owner            string      `json:"owner"`

	// cf:process — keyed by algorithm name (arbitrary map)
	ProcessState map[string]string `json:"process_state,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Version   int64     `json:"version"`
}

// ──────────────────────────────────────────────────────────────────────────────
// Delivery — row in the `deliveries` Bigtable table.
// Row key: v1#<delivery_id>
// CF layout: cf:meta
// ──────────────────────────────────────────────────────────────────────────────
type Delivery struct {
	DeliveryID  string         `json:"delivery_id"`
	CustomerID  string         `json:"customer_id"`
	Status      DeliveryStatus `json:"status"`
	DeliveredAt *time.Time     `json:"delivered_at,omitempty"`
	ManifestURI string         `json:"manifest_uri,omitempty"`
	ContractID  string         `json:"contract_id,omitempty"`
	Note        string         `json:"note,omitempty"`
	AssetCount  int            `json:"asset_count"`
	Owner       string         `json:"owner"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Version   int64     `json:"version"`
}

// DeliveryItem is used only in API request/response bodies.
// In Bigtable this is stored as idx_asset_deliveries + idx_customer_deliveries.
type DeliveryItem struct {
	DeliveryID string    `json:"delivery_id"`
	AssetID    string    `json:"asset_id"`
	CreatedAt  time.Time `json:"created_at"`
}
