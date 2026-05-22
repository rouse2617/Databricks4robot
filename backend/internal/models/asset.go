package models

import (
	"fmt"
	"time"
)

// AssetStatus represents the asset lifecycle status.
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
// Asset — row in the `assets` table (PostgreSQL).
//
// Legacy CF layout (retained for backward compat during dual-write):
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

	// ── Legacy fields (retained for backward compatibility) ──────────────
	StartTimestampNs int64       `json:"start_timestamp_ns"`
	EndTimestampNs   int64       `json:"end_timestamp_ns"`
	DurationSec      float64     `json:"duration_sec"`
	Reviewer         string      `json:"reviewer"`
	Status           AssetStatus `json:"status"` // API-only; derived from lifecycle_state (not stored in PostgreSQL)
	Owner            string      `json:"owner"`
	SegType          string      `json:"type,omitempty"`
	Env              string      `json:"env,omitempty"`
	Task             string      `json:"task,omitempty"`

	LastDeliveredAt *time.Time `json:"last_delivered_at,omitempty"`
	LastDeliveredTo string     `json:"last_delivered_to,omitempty"`
	DeliveryCount   int        `json:"delivery_count"`

	AlgoResults map[string]string `json:"algo_results,omitempty"`
	Tags        map[string]string `json:"tags,omitempty"`
	Files       map[string]string `json:"files,omitempty"`

	LifecycleMeta map[string]interface{} `json:"lifecycle_meta,omitempty"`

	// ── NEW typed fields (Phase 2 — schema evolution) ────────────────────
	AssetType           string                 `json:"asset_type"`
	LifecycleState      string                 `json:"lifecycle_state"`
	DurationMs          int64                  `json:"duration_ms"`
	StorageURI          string                 `json:"storage_uri,omitempty"`
	ThumbURI            string                 `json:"thumb_uri,omitempty"`
	RetentionTier       string                 `json:"retention_tier,omitempty"`
	ExpireAt            *time.Time             `json:"expire_at,omitempty"`
	AssetLevel          int                    `json:"asset_level"`
	ParentAssetID       string                 `json:"parent_asset_id,omitempty"`
	RootAssetID         string                 `json:"root_asset_id,omitempty"`
	SplitMethod         string                 `json:"split_method,omitempty"`
	SplitAlgoName       string                 `json:"split_algo_name,omitempty"`
	SplitAlgoVersion    string                 `json:"split_algo_version,omitempty"`
	SplitRunID          string                 `json:"split_run_id,omitempty"`
	SplitReason         string                 `json:"split_reason,omitempty"`
	SegmentIndex        *int                   `json:"segment_index,omitempty"`
	ParentStartOffsetMs *int64                 `json:"parent_start_offset_ms,omitempty"`
	ParentEndOffsetMs   *int64                 `json:"parent_end_offset_ms,omitempty"`
	Metadata            map[string]interface{} `json:"metadata,omitempty"`
	FilesJSON           map[string]interface{} `json:"files_json,omitempty"`
	AlgoInputsURIs      map[string]string      `json:"algo_inputs_uris,omitempty"`
	AnnotInputsURIs     map[string]string      `json:"annot_inputs_uris,omitempty"`
	TenantID            string                 `json:"tenant_id,omitempty"`
	ProjectID           string                 `json:"project_id,omitempty"`

	// Multi-version identity (CYB-1013). Legacy rows may have empty/zero values.
	LogicalAssetID string `json:"logical_asset_id,omitempty"`
	Revision       int64  `json:"revision,omitempty"`
	IsCurrent      bool   `json:"is_current"`

	// Timestamps / row optimistic-lock version (not content revision)
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Version   int64     `json:"version"`
}

// LogicalAsset is the version coordinator row for a family of asset revisions.
type LogicalAsset struct {
	LogicalAssetID  string                 `json:"logical_asset_id"`
	AssetType       string                 `json:"asset_type"`
	DisplayName     string                 `json:"display_name,omitempty"`
	Description     string                 `json:"description,omitempty"`
	Owner           string                 `json:"owner,omitempty"`
	Status          string                 `json:"status"`
	CurrentRevision int64                  `json:"current_revision"`
	TotalRevisions  int64                  `json:"total_revisions"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// SyncLegacyFields computes backward-compatible legacy fields from the new
// typed fields. Call this after reading from the database to ensure API
// responses include both old and new field names.
//
//   - DurationSec = float64(DurationMs) / 1000.0
//   - SegType mirrors AssetType
//   - Status derived from lifecycle_state (API-only; not stored in PostgreSQL)
func (a *Asset) SyncLegacyFields() {
	if a.LifecycleState != "" {
		a.Status = AssetStatus(LifecycleToStatus(a.LifecycleState))
	} else if a.Status != "" {
		a.LifecycleState = StatusToLifecycle(string(a.Status))
	}
	a.DurationSec = float64(a.DurationMs) / 1000.0
	a.SegType = a.AssetType
	if a.Metadata != nil {
		if a.Env == "" {
			if v, ok := a.Metadata["env"].(string); ok {
				a.Env = v
			}
		}
		if a.Task == "" {
			if v, ok := a.Metadata["task"].(string); ok {
				a.Task = v
			}
		}
	}
	if len(a.Files) == 0 && len(a.FilesJSON) > 0 {
		a.Files = make(map[string]string, len(a.FilesJSON))
		for k, v := range a.FilesJSON {
			switch val := v.(type) {
			case string:
				a.Files[k] = val
			default:
				a.Files[k] = fmt.Sprint(val)
			}
		}
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// McapFile — row in the `mcap_files` PostgreSQL table.
type McapFile struct {
	McapFileID string `json:"mcap_file_id"`

	// Content identity
	GCSPath       string      `json:"gcs_path"`
	SizeBytes     int64       `json:"size_bytes"`
	RawHashMD5    string      `json:"raw_hash_md5"`
	RawHashSHA256 string      `json:"raw_hash_sha256,omitempty"`
	IngestState   IngestState `json:"ingest_state"`

	// Time / structure summary
	FileDurationMs   int64 `json:"file_duration_ms,omitempty"`
	StartTimestampNs int64 `json:"start_timestamp_ns,omitempty"`
	EndTimestampNs   int64 `json:"end_timestamp_ns,omitempty"`
	ChannelCount     int   `json:"channel_count,omitempty"`
	ChunkCount       int   `json:"chunk_count,omitempty"`

	// Provenance
	VendorID         string `json:"vendor_id,omitempty"`
	CollectorID      string `json:"collector_id,omitempty"`
	TaskID           string `json:"task_id,omitempty"`
	DeviceID         string `json:"device_id,omitempty"`
	CameraModel      string `json:"camera_model,omitempty"`
	DataSource       string `json:"data_source,omitempty"`
	LocationID       string `json:"location_id,omitempty"`
	SceneID          string `json:"scene_id,omitempty"`
	EnvironmentID    string `json:"environment_id,omitempty"`
	CollectionMethod string `json:"collection_method,omitempty"`

	// Ownership / lifecycle
	Owner         string     `json:"owner"`
	RetentionTier string     `json:"retention_tier,omitempty"`
	ExpireAt      *time.Time `json:"expire_at,omitempty"`
	TenantID      string     `json:"tenant_id,omitempty"`
	ProjectID     string     `json:"project_id,omitempty"`

	// Extension
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	ProcessState map[string]string      `json:"process_state,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Version   int64     `json:"version"`
}

// ──────────────────────────────────────────────────────────────────────────────
// Delivery — row in the `deliveries` PostgreSQL table.
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

	// NEW typed fields (Phase 2 — schema evolution)
	DeliveryType      string                 `json:"delivery_type"`
	RequestedBy       string                 `json:"requested_by,omitempty"`
	ApprovedBy        string                 `json:"approved_by,omitempty"`
	DeliveredBy       string                 `json:"delivered_by,omitempty"`
	ReplayManifestURI string                 `json:"replay_manifest_uri,omitempty"`
	ItemCount         int64                  `json:"item_count"`
	TotalSizeBytes    *int64                 `json:"total_size_bytes,omitempty"`
	CompletedAt       *time.Time             `json:"completed_at,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
	TenantID          string                 `json:"tenant_id,omitempty"`
	ProjectID         string                 `json:"project_id,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Version   int64     `json:"version"`
}

// DeliveryItem is used only in API request/response bodies.
// Indexed for delivery lookup queries.
type DeliveryItem struct {
	DeliveryID string    `json:"delivery_id"`
	AssetID    string    `json:"asset_id"`
	CreatedAt  time.Time `json:"created_at"`
}

// ── Lifecycle state helpers (shared between postgres and usecase layers) ──

// validLifecycleStates is the canonical set of lifecycle_state values.
// Kept in sync with postgres.AllLifecycleStates and the DB CHECK constraint.
var validLifecycleStates = map[string]bool{
	"created":    true,
	"processing": true,
	"ready":      true,
	"rejected":   true,
	"delivered":  true,
	"archived":   true,
	"superseded": true,
	"failed":     true,
}

// IsValidLifecycleState reports whether ls is a valid lifecycle_state value.
func IsValidLifecycleState(ls string) bool {
	return validLifecycleStates[ls]
}

// lifecycleToStatusMap maps lifecycle_state → legacy status.
var lifecycleToStatusMap = map[string]string{
	"created":    "approved",
	"processing": "approved",
	"ready":      "approved",
	"rejected":   "rejected",
	"delivered":  "approved",
	"archived":   "archived",
	"superseded": "superseded",
	"failed":     "rejected",
}

// statusToLifecycleMap maps legacy status → lifecycle_state.
var statusToLifecycleMap = map[string]string{
	"approved":   "ready",
	"rejected":   "rejected",
	"archived":   "archived",
	"superseded": "superseded",
}

// LifecycleToStatus converts a lifecycle_state to the corresponding legacy status.
// Returns "approved" for unknown inputs.
func LifecycleToStatus(ls string) string {
	if s, ok := lifecycleToStatusMap[ls]; ok {
		return s
	}
	return "approved"
}

// StatusToLifecycle converts a legacy status to the corresponding lifecycle_state.
// Returns "created" for unknown inputs.
func StatusToLifecycle(status string) string {
	if ls, ok := statusToLifecycleMap[status]; ok {
		return ls
	}
	return "created"
}
