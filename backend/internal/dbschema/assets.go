// Package dbschema holds the GORM model definitions that are the schema
// source-of-truth for cyber-databrew. Atlas's `migrate diff` reads this
// package via cmd/schemadump to compute schema diffs against the SQL
// migrations in backend/migrations/.
//
// The types here mirror the columns defined in 000_initial.sql (and
// later migrations). GORM tags drive both:
//   - the live database schema (via GORM AutoMigrate / Atlas inspection)
//   - the business type used by repo / handler code (with json tags
//     for API serialization)
//
// During the GORM-adoption rollout, these models will be merged with
// the existing business types in backend/internal/models/ once we
// confirm the pattern (PoC: `assets` table only).
package dbschema

import (
	"time"

	"gorm.io/gorm"
)

// Asset mirrors the `assets` table defined in
// backend/migrations/000_initial.sql. Field-by-field mapping:
//   - json tags drive the business API serialization
//   - gorm tags drive the database schema (column name, type, nullability)
//
// The CHECK constraints (lifecycle_state enum, mcap_file_id regex, etc.)
// are NOT expressible in GORM tags and are added separately via the
// `AfterMigrate` hook on the schema-dump binary.
type Asset struct {
	// Identity
	AssetID    string `gorm:"column:asset_id;type:text;primaryKey" json:"asset_id"`
	McapFileID string `gorm:"column:mcap_file_id;type:text" json:"mcap_file_id"`

	// Segment locator — character(40) (deterministic SHA-1)
	SegmentLocator string `gorm:"column:segment_locator;type:char(40)" json:"segment_locator,omitempty"`

	// Phase 2 typed fields
	AssetType      string `gorm:"column:asset_type;type:text;default:'segment';not null" json:"asset_type"`
	LifecycleState string `gorm:"column:lifecycle_state;type:text;default:'created';not null" json:"lifecycle_state"`
	IsDeleted      bool   `gorm:"column:is_deleted;type:boolean;default:false" json:"is_deleted"`
	DurationMs     int64  `gorm:"column:duration_ms;type:bigint;default:0;not null" json:"duration_ms"`

	// Ownership / metadata
	Owner        string `gorm:"column:owner;type:text;default:'';not null" json:"owner"`
	Reviewer     string `gorm:"column:reviewer;type:text;default:'';not null" json:"reviewer"`
	StorageURI   string `gorm:"column:storage_uri;type:text;default:'';not null" json:"storage_uri,omitempty"`
	ThumbURI     string `gorm:"column:thumb_uri;type:text;default:'';not null" json:"thumb_uri,omitempty"`
	RetentionTier string `gorm:"column:retention_tier;type:text;default:'';not null" json:"retention_tier,omitempty"`
	ExpireAt     *time.Time `gorm:"column:expire_at;type:timestamptz" json:"expire_at,omitempty"`

	// Asset hierarchy
	AssetLevel    int    `gorm:"column:asset_level;type:integer;default:0;not null" json:"asset_level"`
	ParentAssetID string `gorm:"column:parent_asset_id;type:text" json:"parent_asset_id,omitempty"`
	RootAssetID   string `gorm:"column:root_asset_id;type:text" json:"root_asset_id,omitempty"`

	// Delivery tracking
	DeliveryCount   int        `gorm:"column:delivery_count;type:integer;default:0;not null" json:"delivery_count"`
	LastDeliveredAt *time.Time `gorm:"column:last_delivered_at;type:timestamptz" json:"last_delivered_at,omitempty"`
	LastDeliveredTo string     `gorm:"column:last_delivered_to;type:text;default:'';not null" json:"last_delivered_to,omitempty"`

	// Split metadata
	SegmentIndex        *int   `gorm:"column:segment_index;type:integer" json:"segment_index,omitempty"`
	ParentStartOffsetMs *int64 `gorm:"column:parent_start_offset_ms;type:bigint" json:"parent_start_offset_ms,omitempty"`
	ParentEndOffsetMs   *int64 `gorm:"column:parent_end_offset_ms;type:bigint" json:"parent_end_offset_ms,omitempty"`
	SplitMethod         string `gorm:"column:split_method;type:text" json:"split_method,omitempty"`
	SplitAlgoName       string `gorm:"column:split_algo_name;type:text" json:"split_algo_name,omitempty"`
	SplitAlgoVersion    string `gorm:"column:split_algo_version;type:text" json:"split_algo_version,omitempty"`
	SplitRunID          string `gorm:"column:split_run_id;type:text" json:"split_run_id,omitempty"`
	SplitReason         string `gorm:"column:split_reason;type:text" json:"split_reason,omitempty"`

	// JSONB columns
	Metadata        map[string]interface{} `gorm:"column:metadata;type:jsonb;default:'{}';not null;serializer:json" json:"metadata,omitempty"`
	Files           map[string]interface{} `gorm:"column:files;type:jsonb;default:'{}';not null;serializer:json" json:"files,omitempty"`
	AlgoInputsURIs  map[string]string      `gorm:"column:algo_inputs_uris;type:jsonb;default:'{}';not null;serializer:json" json:"algo_inputs_uris,omitempty"`
	AnnotInputsURIs map[string]string      `gorm:"column:annot_inputs_uris;type:jsonb;default:'{}';not null;serializer:json" json:"annot_inputs_uris,omitempty"`

	// Multi-tenant
	TenantID  string `gorm:"column:tenant_id;type:text" json:"tenant_id,omitempty"`
	ProjectID string `gorm:"column:project_id;type:text" json:"project_id,omitempty"`

	// Multi-version identity (CYB-1013)
	LogicalAssetID string `gorm:"column:logical_asset_id;type:text" json:"logical_asset_id,omitempty"`
	Revision       *int64 `gorm:"column:revision;type:bigint" json:"revision,omitempty"`
	IsCurrent      *bool  `gorm:"column:is_current;type:boolean" json:"is_current,omitempty"`

	// Timestamps / row optimistic-lock
	CreatedAt time.Time `gorm:"column:created_at;type:timestamptz;not null" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;type:timestamptz;not null" json:"updated_at"`
	Version   int64     `gorm:"column:version;type:bigint;default:1" json:"version"`
}

// TableName overrides GORM's default pluralization. The table is
// exactly `assets` (plural), which happens to match GORM's default
// plural for "Asset", so this is just defensive.
func (Asset) TableName() string { return "assets" }

// AllModels returns every model that schemadump should register.
// Adding a new model = add it here.
func AllModels() []any {
	return []any{
		&Asset{},
	}
}

// ensure gorm import is "used" even if the type above is the only ref
var _ = gorm.Model{}