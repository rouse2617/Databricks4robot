package dbschema

import (
	"time"

	"gorm.io/gorm"
)

// Asset is the GORM model for the `assets` table (see 000_initial.sql,
// modified by 044/045/060 — all changes are CHECK constraints only;
// columns themselves are unchanged).
//
// Check constraints (lifecycle_state enum, asset_id regex, mcap_file_id
// regex, etc.) and indexes are NOT expressible in GORM tags and are
// expected to remain in SQL migrations. Atlas `migrate diff` will
// report these as drift; the team accepts this known gap.
type Asset struct {
	AssetID         string     `gorm:"column:asset_id;type:text;primaryKey" json:"asset_id"`
	McapFileID      string     `gorm:"column:mcap_file_id;type:text" json:"mcap_file_id"`
	SegmentLocator  string     `gorm:"column:segment_locator;type:char(40)" json:"segment_locator,omitempty"`
	StartTimestampNs int64      `gorm:"column:start_timestamp_ns;type:bigint;not null" json:"start_timestamp_ns"`
	EndTimestampNs   int64      `gorm:"column:end_timestamp_ns;type:bigint;not null" json:"end_timestamp_ns"`
	DurationSec      float64    `gorm:"column:duration_sec;type:double precision" json:"duration_sec"`
	Reviewer         string     `gorm:"column:reviewer;type:text" json:"reviewer"`
	Owner            string     `gorm:"column:owner;type:text" json:"owner"`
	SegType          string     `gorm:"column:seg_type;type:text" json:"type,omitempty"`
	Env              string     `gorm:"column:env;type:text" json:"env,omitempty"`
	Task             string     `gorm:"column:task;type:text" json:"task,omitempty"`
	LastDeliveredAt  *time.Time `gorm:"column:last_delivered_at;type:timestamptz" json:"last_delivered_at,omitempty"`
	LastDeliveredTo  string     `gorm:"column:last_delivered_to;type:text" json:"last_delivered_to,omitempty"`
	DeliveryCount    int        `gorm:"column:delivery_count;type:integer" json:"delivery_count"`
	AlgoResults      string     `gorm:"column:algo_results;type:jsonb" json:"algo_results,omitempty"`
	Tags             string     `gorm:"column:tags;type:jsonb" json:"tags,omitempty"`
	Files            string     `gorm:"column:files;type:jsonb" json:"files,omitempty"`
	LifecycleMeta    string     `gorm:"column:lifecycle_meta;type:jsonb" json:"lifecycle_meta,omitempty"`
	AssetType        string     `gorm:"column:asset_type;type:text;default:'segment';not null" json:"asset_type"`
	LifecycleState   string     `gorm:"column:lifecycle_state;type:text;default:'created';not null" json:"lifecycle_state"`
	IsDeleted        bool       `gorm:"column:is_deleted;type:boolean;default:false" json:"is_deleted"`
	DurationMs       int64      `gorm:"column:duration_ms;type:bigint;default:0;not null" json:"duration_ms"`
	StorageURI       string     `gorm:"column:storage_uri;type:text;default:'';not null" json:"storage_uri,omitempty"`
	ThumbURI         string     `gorm:"column:thumb_uri;type:text;default:'';not null" json:"thumb_uri,omitempty"`
	RetentionTier    string     `gorm:"column:retention_tier;type:text;default:'';not null" json:"retention_tier,omitempty"`
	ExpireAt         *time.Time `gorm:"column:expire_at;type:timestamptz" json:"expire_at,omitempty"`
	AssetLevel       int        `gorm:"column:asset_level;type:integer;default:0;not null" json:"asset_level"`
	ParentAssetID    string     `gorm:"column:parent_asset_id;type:text" json:"parent_asset_id,omitempty"`
	RootAssetID      string     `gorm:"column:root_asset_id;type:text" json:"root_asset_id,omitempty"`
	SegmentIndex     *int       `gorm:"column:segment_index;type:integer" json:"segment_index,omitempty"`
	ParentStartOffsetMs *int64  `gorm:"column:parent_start_offset_ms;type:bigint" json:"parent_start_offset_ms,omitempty"`
	ParentEndOffsetMs   *int64  `gorm:"column:parent_end_offset_ms;type:bigint" json:"parent_end_offset_ms,omitempty"`
	SplitMethod         string  `gorm:"column:split_method;type:text" json:"split_method,omitempty"`
	SplitAlgoName       string  `gorm:"column:split_algo_name;type:text" json:"split_algo_name,omitempty"`
	SplitAlgoVersion    string  `gorm:"column:split_algo_version;type:text" json:"split_algo_version,omitempty"`
	SplitRunID          string  `gorm:"column:split_run_id;type:text" json:"split_run_id,omitempty"`
	SplitReason         string  `gorm:"column:split_reason;type:text" json:"split_reason,omitempty"`
	Metadata            string  `gorm:"column:metadata;type:jsonb;default:'{}';not null" json:"metadata,omitempty"`
	FilesJSON           string  `gorm:"column:files_json;type:jsonb" json:"files_json,omitempty"`
	AlgoInputsURIs      string  `gorm:"column:algo_inputs_uris;type:jsonb;default:'{}';not null" json:"algo_inputs_uris,omitempty"`
	AnnotInputsURIs     string  `gorm:"column:annot_inputs_uris;type:jsonb;default:'{}';not null" json:"annot_inputs_uris,omitempty"`
	TenantID            string  `gorm:"column:tenant_id;type:text" json:"tenant_id,omitempty"`
	ProjectID           string  `gorm:"column:project_id;type:text" json:"project_id,omitempty"`
	LogicalAssetID      string  `gorm:"column:logical_asset_id;type:text" json:"logical_asset_id,omitempty"`
	Revision            *int64  `gorm:"column:revision;type:bigint" json:"revision,omitempty"`
	IsCurrent           *bool   `gorm:"column:is_current;type:boolean" json:"is_current,omitempty"`
	CreatedAt           time.Time `gorm:"column:created_at;type:timestamptz;not null" json:"created_at"`
	UpdatedAt           time.Time `gorm:"column:updated_at;type:timestamptz;not null" json:"updated_at"`
	Version             int64   `gorm:"column:version;type:bigint;default:1" json:"version"`
}

func (Asset) TableName() string { return "assets" }

// LogicalAsset — `logical_assets` table (multi-version identity coordinator).
// Modified by 060 to widen logical_asset_id regex (UUID format support).
type LogicalAsset struct {
	LogicalAssetID  string    `gorm:"column:logical_asset_id;type:text;primaryKey" json:"logical_asset_id"`
	AssetType       string    `gorm:"column:asset_type;type:text;not null" json:"asset_type"`
	DisplayName     string    `gorm:"column:display_name;type:text" json:"display_name,omitempty"`
	Description     string    `gorm:"column:description;type:text" json:"description,omitempty"`
	Owner           string    `gorm:"column:owner;type:text" json:"owner,omitempty"`
	Status          string    `gorm:"column:status;type:text;default:'active';not null" json:"status"`
	CurrentRevision int64     `gorm:"column:current_revision;type:bigint;default:1;not null" json:"current_revision"`
	TotalRevisions  int64     `gorm:"column:total_revisions;type:bigint;default:0" json:"total_revisions"`
	Metadata        string    `gorm:"column:metadata;type:jsonb" json:"metadata,omitempty"`
	CreatedAt       time.Time `gorm:"column:created_at;type:timestamptz" json:"created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at;type:timestamptz" json:"updated_at"`
}

func (LogicalAsset) TableName() string { return "logical_assets" }

// Note: `asset_events` and `asset_events_default` are NOT modeled here.
// GORM does not support Postgres table partitioning, so the parent
// partition + DEFAULT partition are managed entirely by SQL migrations.

// AssetTag — `asset_tags` (CYB-1015 multi-source tag assertions).
type AssetTag struct {
	AssetID    string    `gorm:"column:asset_id;type:text;primaryKey" json:"asset_id"`
	Source     string    `gorm:"column:source;type:text;primaryKey" json:"source"`
	Key        string    `gorm:"column:key;type:text;primaryKey" json:"key"`
	Value      string    `gorm:"column:value;type:text;not null" json:"value"`
	Confidence float64   `gorm:"column:confidence;type:double precision" json:"confidence,omitempty"`
	ProducerID string    `gorm:"column:producer_id;type:text" json:"producer_id,omitempty"`
	CreatedAt  time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
}

func (AssetTag) TableName() string { return "asset_tags" }

// AssetAlgoLatest — `asset_algo_latest` (one row per asset+algo with latest result).
type AssetAlgoLatest struct {
	AssetID         string    `gorm:"column:asset_id;type:text;primaryKey" json:"asset_id"`
	Algo            string    `gorm:"column:algo;type:text;primaryKey" json:"algo"`
	AlgoVersion     string    `gorm:"column:algo_version;type:text;not null" json:"algo_version"`
	Status          string    `gorm:"column:status;type:text;not null" json:"status"`
	Score           *float64  `gorm:"column:score;type:double precision" json:"score,omitempty"`
	Result          string    `gorm:"column:result;type:jsonb" json:"result,omitempty"`
	StartedAt       time.Time `gorm:"column:started_at;type:timestamptz;not null" json:"started_at"`
	CompletedAt     time.Time `gorm:"column:completed_at;type:timestamptz;not null" json:"completed_at"`
	RecordedAt      time.Time `gorm:"column:recorded_at;type:timestamptz;not null;default:now()" json:"recorded_at"`
}

func (AssetAlgoLatest) TableName() string { return "asset_algo_latest" }

// AssetMetric — `asset_metrics` (time-series metrics per asset).
type AssetMetric struct {
	AssetID     string    `gorm:"column:asset_id;type:text;primaryKey" json:"asset_id"`
	MetricName  string    `gorm:"column:metric_name;type:text;primaryKey" json:"metric_name"`
	BucketStart time.Time `gorm:"column:bucket_start;type:timestamptz;primaryKey" json:"bucket_start"`
	Value       float64   `gorm:"column:value;type:double precision;not null" json:"value"`
	UpdatedAt   time.Time `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
}

func (AssetMetric) TableName() string { return "asset_metrics" }

// AssetRelation — `asset_relations` (asset graph edges). Modified by 044/045
// to widen relation_type enum + add metadata column.
type AssetRelation struct {
	RelationID    int64     `gorm:"column:relation_id;type:bigint;primaryKey;autoIncrement" json:"relation_id"`
	FromAssetID   string    `gorm:"column:from_asset_id;type:text;not null" json:"from_asset_id"`
	ToAssetID     string    `gorm:"column:to_asset_id;type:text;not null" json:"to_asset_id"`
	RelationType  string    `gorm:"column:relation_type;type:text;not null" json:"relation_type"`
	AlgoName      string    `gorm:"column:algo_name;type:text" json:"algo_name,omitempty"`
	AlgoVersion   string    `gorm:"column:algo_version;type:text" json:"algo_version,omitempty"`
	Confidence    *float64  `gorm:"column:confidence;type:double precision" json:"confidence,omitempty"`
	Metadata      string    `gorm:"column:metadata;type:jsonb;default:'{}';not null" json:"metadata,omitempty"`
	CreatedAt     time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
}

func (AssetRelation) TableName() string { return "asset_relations" }

// AssetEvalResult — `asset_eval_results` (evaluation results for ML model assets).
type AssetEvalResult struct {
	EvalID        int64     `gorm:"column:eval_id;type:bigint;primaryKey;autoIncrement" json:"eval_id"`
	AssetID       string    `gorm:"column:asset_id;type:text;not null" json:"asset_id"`
	EvalType      string    `gorm:"column:eval_type;type:text;not null" json:"eval_type"`
	Score         *float64  `gorm:"column:score;type:double precision" json:"score,omitempty"`
	Threshold     *float64  `gorm:"column:threshold;type:double precision" json:"threshold,omitempty"`
	Passed        *bool     `gorm:"column:passed;type:boolean" json:"passed,omitempty"`
	Metrics       string    `gorm:"column:metrics;type:jsonb" json:"metrics,omitempty"`
	StartedAt     time.Time `gorm:"column:started_at;type:timestamptz" json:"started_at"`
	CompletedAt   time.Time `gorm:"column:completed_at;type:timestamptz" json:"completed_at,omitempty"`
	CreatedAt     time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
}

func (AssetEvalResult) TableName() string { return "asset_eval_results" }

// AssetUsageStat — `asset_usage_stats` (aggregated usage counters per asset).
type AssetUsageStat struct {
	AssetID      string    `gorm:"column:asset_id;type:text;primaryKey" json:"asset_id"`
	ViewCount    int64     `gorm:"column:view_count;type:bigint;default:0;not null" json:"view_count"`
	DownloadCount int64    `gorm:"column:download_count;type:bigint;default:0;not null" json:"download_count"`
	QueryCount   int64     `gorm:"column:query_count;type:bigint;default:0;not null" json:"query_count"`
	DeliveryCount int64    `gorm:"column:delivery_count;type:bigint;default:0;not null" json:"delivery_count"`
	LastUsedAt   *time.Time `gorm:"column:last_used_at;type:timestamptz" json:"last_used_at,omitempty"`
	UpdatedAt    time.Time `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
}

func (AssetUsageStat) TableName() string { return "asset_usage_stats" }

// McapFile — `mcap_files` table.
type McapFile struct {
	McapFileID     string    `gorm:"column:mcap_file_id;type:text;primaryKey" json:"mcap_file_id"`
	GCSPath        string    `gorm:"column:gcs_path;type:text;not null" json:"gcs_path"`
	SizeBytes      int64     `gorm:"column:size_bytes;type:bigint;not null" json:"size_bytes"`
	RawHashMD5     string    `gorm:"column:raw_hash_md5;type:text;not null" json:"raw_hash_md5"`
	RawHashSHA256  string    `gorm:"column:raw_hash_sha256;type:text" json:"raw_hash_sha256,omitempty"`
	IngestState    string    `gorm:"column:ingest_state;type:text;not null;default:'pending'" json:"ingest_state"`
	FileDurationMs int64     `gorm:"column:file_duration_ms;type:bigint" json:"file_duration_ms,omitempty"`
	StartTimestampNs int64   `gorm:"column:start_timestamp_ns;type:bigint" json:"start_timestamp_ns,omitempty"`
	EndTimestampNs   int64   `gorm:"column:end_timestamp_ns;type:bigint" json:"end_timestamp_ns,omitempty"`
	ChannelCount   int       `gorm:"column:channel_count;type:integer" json:"channel_count,omitempty"`
	ChunkCount     int       `gorm:"column:chunk_count;type:integer" json:"chunk_count,omitempty"`
	VendorID       string    `gorm:"column:vendor_id;type:text" json:"vendor_id,omitempty"`
	CollectorID    string    `gorm:"column:collector_id;type:text" json:"collector_id,omitempty"`
	TaskID         string    `gorm:"column:task_id;type:text" json:"task_id,omitempty"`
	DeviceID       string    `gorm:"column:device_id;type:text" json:"device_id,omitempty"`
	CameraModel    string    `gorm:"column:camera_model;type:text" json:"camera_model,omitempty"`
	DataSource     string    `gorm:"column:data_source;type:text" json:"data_source,omitempty"`
	LocationID     string    `gorm:"column:location_id;type:text" json:"location_id,omitempty"`
	SceneID        string    `gorm:"column:scene_id;type:text" json:"scene_id,omitempty"`
	EnvironmentID  string    `gorm:"column:environment_id;type:text" json:"environment_id,omitempty"`
	CollectionMethod string  `gorm:"column:collection_method;type:text" json:"collection_method,omitempty"`
	Owner          string    `gorm:"column:owner;type:text" json:"owner"`
	RetentionTier  string    `gorm:"column:retention_tier;type:text" json:"retention_tier,omitempty"`
	ExpireAt       *time.Time `gorm:"column:expire_at;type:timestamptz" json:"expire_at,omitempty"`
	TenantID       string    `gorm:"column:tenant_id;type:text" json:"tenant_id,omitempty"`
	ProjectID      string    `gorm:"column:project_id;type:text" json:"project_id,omitempty"`
	Metadata       string    `gorm:"column:metadata;type:jsonb" json:"metadata,omitempty"`
	ProcessState   string    `gorm:"column:process_state;type:jsonb" json:"process_state,omitempty"`
	CreatedAt      time.Time `gorm:"column:created_at;type:timestamptz;not null" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at;type:timestamptz;not null" json:"updated_at"`
	Version        int64     `gorm:"column:version;type:bigint;default:1" json:"version"`
}

func (McapFile) TableName() string { return "mcap_files" }

// keep gorm import used
var _ = gorm.Model{}