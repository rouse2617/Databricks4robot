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
//
// Note: API-only / derived fields like `duration_sec`, `seg_type`,
// `env`, `task`, `algo_results`, `tags`, `lifecycle_meta`, `files_json`
// exist on the business type (internal/models/asset.go) but are NOT
// real SQL columns. They are NOT included here; if migrated to GORM
// they should live in a separate "view" type.
type Asset struct {
	AssetID            string     `gorm:"column:asset_id;type:text;primaryKey" json:"asset_id"`
	McapFileID         string     `gorm:"column:mcap_file_id;type:text" json:"mcap_file_id"`
	SegmentLocator     string     `gorm:"column:segment_locator;type:char(40)" json:"segment_locator,omitempty"`
	StartTimestampNs   int64      `gorm:"column:start_timestamp_ns;type:bigint;not null" json:"start_timestamp_ns"`
	EndTimestampNs     int64      `gorm:"column:end_timestamp_ns;type:bigint;not null" json:"end_timestamp_ns"`
	Reviewer           string     `gorm:"column:reviewer;type:text;default:'';not null" json:"reviewer"`
	Owner              string     `gorm:"column:owner;type:text;default:'';not null" json:"owner"`
	LastDeliveredAt    *time.Time `gorm:"column:last_delivered_at;type:timestamptz" json:"last_delivered_at,omitempty"`
	LastDeliveredTo    string     `gorm:"column:last_delivered_to;type:text;default:'';not null" json:"last_delivered_to,omitempty"`
	DeliveryCount      int        `gorm:"column:delivery_count;type:integer;default:0;not null" json:"delivery_count"`
	Files              string     `gorm:"column:files;type:jsonb;default:'{}';not null" json:"files,omitempty"`
	AssetType          string     `gorm:"column:asset_type;type:text;default:'segment';not null" json:"asset_type"`
	LifecycleState     string     `gorm:"column:lifecycle_state;type:text;default:'created';not null" json:"lifecycle_state"`
	IsDeleted          bool       `gorm:"column:is_deleted;type:boolean;default:false" json:"is_deleted"`
	DurationMs         int64      `gorm:"column:duration_ms;type:bigint;default:0;not null" json:"duration_ms"`
	StorageURI         string     `gorm:"column:storage_uri;type:text;default:'';not null" json:"storage_uri,omitempty"`
	ThumbURI           string     `gorm:"column:thumb_uri;type:text;default:'';not null" json:"thumb_uri,omitempty"`
	RetentionTier      string     `gorm:"column:retention_tier;type:text;default:'';not null" json:"retention_tier,omitempty"`
	ExpireAt           *time.Time `gorm:"column:expire_at;type:timestamptz" json:"expire_at,omitempty"`
	AssetLevel         int        `gorm:"column:asset_level;type:integer;default:0;not null" json:"asset_level"`
	ParentAssetID      string     `gorm:"column:parent_asset_id;type:text" json:"parent_asset_id,omitempty"`
	RootAssetID        string     `gorm:"column:root_asset_id;type:text" json:"root_asset_id,omitempty"`
	SegmentIndex       *int       `gorm:"column:segment_index;type:integer" json:"segment_index,omitempty"`
	ParentStartOffsetMs *int64    `gorm:"column:parent_start_offset_ms;type:bigint" json:"parent_start_offset_ms,omitempty"`
	ParentEndOffsetMs   *int64    `gorm:"column:parent_end_offset_ms;type:bigint" json:"parent_end_offset_ms,omitempty"`
	SplitMethod         string    `gorm:"column:split_method;type:text" json:"split_method,omitempty"`
	SplitAlgoName       string    `gorm:"column:split_algo_name;type:text" json:"split_algo_name,omitempty"`
	SplitAlgoVersion    string    `gorm:"column:split_algo_version;type:text" json:"split_algo_version,omitempty"`
	SplitRunID          string    `gorm:"column:split_run_id;type:text" json:"split_run_id,omitempty"`
	SplitReason         string    `gorm:"column:split_reason;type:text" json:"split_reason,omitempty"`
	Metadata            string    `gorm:"column:metadata;type:jsonb;default:'{}';not null" json:"metadata,omitempty"`
	AlgoInputsURIs      string    `gorm:"column:algo_inputs_uris;type:jsonb;default:'{}';not null" json:"algo_inputs_uris,omitempty"`
	AnnotInputsURIs     string    `gorm:"column:annot_inputs_uris;type:jsonb;default:'{}';not null" json:"annot_inputs_uris,omitempty"`
	TenantID            string    `gorm:"column:tenant_id;type:text" json:"tenant_id,omitempty"`
	ProjectID           string    `gorm:"column:project_id;type:text" json:"project_id,omitempty"`
	LogicalAssetID      string    `gorm:"column:logical_asset_id;type:text" json:"logical_asset_id,omitempty"`
	Revision            *int64    `gorm:"column:revision;type:bigint" json:"revision,omitempty"`
	IsCurrent           *bool     `gorm:"column:is_current;type:boolean" json:"is_current,omitempty"`
	CreatedAt           time.Time `gorm:"column:created_at;type:timestamptz;not null" json:"created_at"`
	UpdatedAt           time.Time `gorm:"column:updated_at;type:timestamptz;not null" json:"updated_at"`
	Version             int64     `gorm:"column:version;type:bigint;default:1" json:"version"`
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
	Status          string    `gorm:"column:status;type:text;not null;default:'active'" json:"status"`
	CurrentRevision int64     `gorm:"column:current_revision;type:bigint;not null;default:1" json:"current_revision"`
	TotalRevisions  int64     `gorm:"column:total_revisions;type:bigint;not null;default:1" json:"total_revisions"`     // drift fix: was default:0
	Metadata        string    `gorm:"column:metadata;type:jsonb;not null;default:'{}'" json:"metadata,omitempty"`              // drift fix: was no not null
	Extra           string    `gorm:"column:extra;type:jsonb;not null;default:'{}'" json:"extra,omitempty"`                  // drift fix: was missing
	CreatedAt       time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`        // drift fix
	UpdatedAt       time.Time `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`        // drift fix
}

func (LogicalAsset) TableName() string { return "logical_assets" }

// Note: `asset_events` and `asset_events_default` are NOT modeled here.
// GORM does not support Postgres table partitioning, so the parent
// partition + DEFAULT partition are managed entirely by SQL migrations.

// AssetTag — `asset_tags` (CYB-1015 multi-source tag assertions).
// Composite PK (asset_id, tag_key). Has a sequence-backed `id` bigint
// column (not part of PK) and a generated `source_version_norm` column.
type AssetTag struct {
	AssetID    string    `gorm:"column:asset_id;type:text;primaryKey" json:"asset_id"`
	TagKey     string    `gorm:"column:tag_key;type:text;primaryKey" json:"tag_key"`
	TagValue   string    `gorm:"column:tag_value;type:text;not null" json:"tag_value"`
	TagValueNum *float64 `gorm:"column:tag_value_num;type:double precision" json:"tag_value_num,omitempty"`
	TagValueBool *bool   `gorm:"column:tag_value_bool;type:boolean" json:"tag_value_bool,omitempty"`
	TagType    string    `gorm:"column:tag_type;type:text;not null;default:'string'" json:"tag_type"`
	SourceType string    `gorm:"column:source_type;type:text;not null;default:'human'" json:"source_type"`
	SourceName string    `gorm:"column:source_name;type:text" json:"source_name,omitempty"`
	SourceVersion string  `gorm:"column:source_version;type:text" json:"source_version,omitempty"`
	RunID      string    `gorm:"column:run_id;type:text" json:"run_id,omitempty"`
	Confidence *float64  `gorm:"column:confidence;type:double precision" json:"confidence,omitempty"`
	TenantID   string    `gorm:"column:tenant_id;type:text" json:"tenant_id,omitempty"`
	ProjectID  string    `gorm:"column:project_id;type:text" json:"project_id,omitempty"`
	ID         int64     `gorm:"column:id;type:bigint;autoIncrement" json:"id"`
	AppliedAt  time.Time `gorm:"column:applied_at;type:timestamptz;not null;default:now()" json:"applied_at"`
	CreatedAt  time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
	// SourceVersionNorm is a GENERATED column; GORM can't model it in
	// tags, but the column exists in SQL. Atlas will report drift; the
	// team accepts this as a known gap.
}

func (AssetTag) TableName() string { return "asset_tags" }

// AssetAlgoLatest — `asset_algo_latest` (one row per asset+algo with
// latest result). Composite PK (asset_id, algo_name).
type AssetAlgoLatest struct {
	AssetID       string     `gorm:"column:asset_id;type:text;primaryKey" json:"asset_id"`
	AlgoName      string     `gorm:"column:algo_name;type:text;primaryKey" json:"algo_name"`
	AlgoVersion   string     `gorm:"column:algo_version;type:text;not null" json:"algo_version"`
	Status        string     `gorm:"column:status;type:text;not null" json:"status"`
	ResultTag     string     `gorm:"column:result_tag;type:text" json:"result_tag,omitempty"`
	ResultScore   *float64   `gorm:"column:result_score;type:double precision" json:"result_score,omitempty"`
	ResultSummary string     `gorm:"column:result_summary;type:jsonb;not null;default:'{}'" json:"result_summary,omitempty"`
	RunID         string     `gorm:"column:run_id;type:text" json:"run_id,omitempty"`
	Method        string     `gorm:"column:method;type:text" json:"method,omitempty"`
	ModelURI      string     `gorm:"column:model_uri;type:text" json:"model_uri,omitempty"`
	OutputURI     string     `gorm:"column:output_uri;type:text" json:"output_uri,omitempty"`
	ErrorCode     string     `gorm:"column:error_code;type:text" json:"error_code,omitempty"`
	ErrorMessage  string     `gorm:"column:error_message;type:text" json:"error_message,omitempty"`
	StartedAt     *time.Time `gorm:"column:started_at;type:timestamptz" json:"started_at,omitempty"`
	FinishedAt    *time.Time `gorm:"column:finished_at;type:timestamptz" json:"finished_at,omitempty"`
	TenantID      string     `gorm:"column:tenant_id;type:text" json:"tenant_id,omitempty"`
	ProjectID     string     `gorm:"column:project_id;type:text" json:"project_id,omitempty"`
	UpdatedAt     time.Time  `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
}

func (AssetAlgoLatest) TableName() string { return "asset_algo_latest" }

// AssetMetric — `asset_metrics` (time-series metrics per asset).
// Composite PK (asset_id, target_type, target_id, metric_key, eval_name,
// eval_version, recorded_at). Many nullable value columns (one of the
// value_* columns holds the actual measurement depending on metric_type).
type AssetMetric struct {
	AssetID         string    `gorm:"column:asset_id;type:text;primaryKey" json:"asset_id"`
	TargetType      string    `gorm:"column:target_type;type:text;not null;default:'segment';primaryKey" json:"target_type"`
	TargetID        string    `gorm:"column:target_id;type:text;not null;default:'';primaryKey" json:"target_id"`
	MetricKey       string    `gorm:"column:metric_key;type:text;not null;primaryKey" json:"metric_key"`
	MetricType      string    `gorm:"column:metric_type;type:text;not null;default:'float'" json:"metric_type"`
	MetricUnit      string    `gorm:"column:metric_unit;type:text" json:"metric_unit,omitempty"`
	MetricValue     *float64  `gorm:"column:metric_value;type:double precision" json:"metric_value,omitempty"`
	MetricValueInt  *int64    `gorm:"column:metric_value_int;type:bigint" json:"metric_value_int,omitempty"`
	MetricValueText string    `gorm:"column:metric_value_text;type:text" json:"metric_value_text,omitempty"`
	MetricValueBool *bool     `gorm:"column:metric_value_bool;type:boolean" json:"metric_value_bool,omitempty"`
	EvalName        string    `gorm:"column:eval_name;type:text;not null;primaryKey" json:"eval_name"`
	EvalVersion     string    `gorm:"column:eval_version;type:text;not null;primaryKey" json:"eval_version"`
	ParameterVersion string   `gorm:"column:parameter_version;type:text" json:"parameter_version,omitempty"`
	RunID           string    `gorm:"column:run_id;type:text" json:"run_id,omitempty"`
	SourceType      string    `gorm:"column:source_type;type:text;not null;default:'algo'" json:"source_type"`
	SourceName      string    `gorm:"column:source_name;type:text" json:"source_name,omitempty"`
	Confidence      *float64  `gorm:"column:confidence;type:double precision" json:"confidence,omitempty"`
	RecordedAt      time.Time `gorm:"column:recorded_at;type:timestamptz;not null;default:now();primaryKey" json:"recorded_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
}

func (AssetMetric) TableName() string { return "asset_metrics" }

// AssetRelation — `asset_relations` (asset graph edges). Modified by
// 044/045 to widen relation_type enum + add metadata column. Composite
// PK (parent_asset_id, child_asset_id, relation_type).
type AssetRelation struct {
	ParentAssetID      string    `gorm:"column:parent_asset_id;type:text;primaryKey" json:"parent_asset_id"`
	ChildAssetID       string    `gorm:"column:child_asset_id;type:text;primaryKey" json:"child_asset_id"`
	RelationType       string    `gorm:"column:relation_type;type:text;primaryKey" json:"relation_type"`
	Method             string    `gorm:"column:method;type:text" json:"method,omitempty"`
	AlgoName           string    `gorm:"column:algo_name;type:text" json:"algo_name,omitempty"`
	AlgoVersion        string    `gorm:"column:algo_version;type:text" json:"algo_version,omitempty"`
	RunID              string    `gorm:"column:run_id;type:text" json:"run_id,omitempty"`
	ParentStartOffsetMs *int64   `gorm:"column:parent_start_offset_ms;type:bigint" json:"parent_start_offset_ms,omitempty"`
	ParentEndOffsetMs   *int64   `gorm:"column:parent_end_offset_ms;type:bigint" json:"parent_end_offset_ms,omitempty"`
	CreatedAt          time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	// Metadata column added in 045 (drift fix below).
	// RelationID bigserial added in 045 — Atlas reports drift; the team
	// accepts this (GORM can model bigserial but the team hasn't
	// migrated PKs to use it yet).
}

func (AssetRelation) TableName() string { return "asset_relations" }

// AssetEvalResult — `asset_eval_results` (evaluation results for assets).
// PK is the auto-generated uuid `eval_result_id` (default gen_random_uuid()).
type AssetEvalResult struct {
	EvalResultID     string     `gorm:"column:eval_result_id;type:uuid;primaryKey;default:gen_random_uuid()" json:"eval_result_id"`
	AssetID          string     `gorm:"column:asset_id;type:text;not null" json:"asset_id"`
	McapFileID       string     `gorm:"column:mcap_file_id;type:text" json:"mcap_file_id,omitempty"`
	TargetType       string     `gorm:"column:target_type;type:text;not null;default:'segment'" json:"target_type"`
	TargetID         string     `gorm:"column:target_id;type:text;not null;default:''" json:"target_id"`
	EvalName         string     `gorm:"column:eval_name;type:text;not null" json:"eval_name"`
	EvalVersion      string     `gorm:"column:eval_version;type:text;not null" json:"eval_version"`
	ParameterVersion string     `gorm:"column:parameter_version;type:text" json:"parameter_version,omitempty"`
	RunID            string     `gorm:"column:run_id;type:text" json:"run_id,omitempty"`
	Status           string     `gorm:"column:status;type:text;not null;default:'ok'" json:"status"`
	ResultPayload    string     `gorm:"column:result_payload;type:jsonb;not null;default:'{}'" json:"result_payload,omitempty"`
	OutputURI        string     `gorm:"column:output_uri;type:text" json:"output_uri,omitempty"`
	SummaryURI       string     `gorm:"column:summary_uri;type:text" json:"summary_uri,omitempty"`
	SourceType       string     `gorm:"column:source_type;type:text;not null;default:'algo'" json:"source_type"`
	SourceName       string     `gorm:"column:source_name;type:text" json:"source_name,omitempty"`
	SourceVersion    string     `gorm:"column:source_version;type:text" json:"source_version,omitempty"`
	StartedAt        *time.Time `gorm:"column:started_at;type:timestamptz" json:"started_at,omitempty"`
	FinishedAt       *time.Time `gorm:"column:finished_at;type:timestamptz" json:"finished_at,omitempty"`
	CreatedAt        time.Time  `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt        time.Time  `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
}

func (AssetEvalResult) TableName() string { return "asset_eval_results" }

// AssetUsageStat — `asset_usage_stats` (per-asset counters). PK on
// `id` (sequence-backed bigint, not asset_id).
type AssetUsageStat struct {
	ID            int64      `gorm:"column:id;type:bigint;primaryKey;autoIncrement" json:"id"`
	AssetID       string     `gorm:"column:asset_id;type:text;not null" json:"asset_id"`
	LogicalAssetID string    `gorm:"column:logical_asset_id;type:text" json:"logical_asset_id,omitempty"`
	ViewCount     int        `gorm:"column:view_count;type:integer;not null;default:0" json:"view_count"`
	LastViewedAt  *time.Time `gorm:"column:last_viewed_at;type:timestamptz" json:"last_viewed_at,omitempty"`
	FavoriteCount int        `gorm:"column:favorite_count;type:integer;not null;default:0" json:"favorite_count"`
	CreatedAt     time.Time  `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
}

func (AssetUsageStat) TableName() string { return "asset_usage_stats" }

// McapFile — `mcap_files` table. Note: actual SQL column is `mcap_uri`
// (not `gcs_path` as the business type had it). Has summary_index
// columns added in 041.
type McapFile struct {
	McapFileID          string     `gorm:"column:mcap_file_id;type:text;primaryKey" json:"mcap_file_id"`
	RawHashMD5          string     `gorm:"column:raw_hash_md5;type:varchar(32)" json:"raw_hash_md5,omitempty"` // nullable!
	IsDeleted           bool       `gorm:"column:is_deleted;type:boolean;default:false" json:"is_deleted"`
	McapURI             string     `gorm:"column:mcap_uri;type:text;not null;default:''" json:"mcap_uri,omitempty"` // drift fix: was gcs_path
	SizeBytes           int64      `gorm:"column:size_bytes;type:bigint;not null;default:0" json:"size_bytes"`
	FileDurationMs      int64      `gorm:"column:file_duration_ms;type:bigint;not null;default:0" json:"file_duration_ms"`
	StartTimestampNs    int64      `gorm:"column:start_timestamp_ns;type:bigint;not null;default:0" json:"start_timestamp_ns"`
	EndTimestampNs      int64      `gorm:"column:end_timestamp_ns;type:bigint;not null;default:0" json:"end_timestamp_ns"`
	ChannelCount        int        `gorm:"column:channel_count;type:integer;not null;default:0" json:"channel_count"`
	ChunkCount          int        `gorm:"column:chunk_count;type:integer;not null;default:0" json:"chunk_count"`
	IngestState         string     `gorm:"column:ingest_state;type:text;not null;default:'pending'" json:"ingest_state"`
	VendorID            string     `gorm:"column:vendor_id;type:text" json:"vendor_id,omitempty"`
	CollectorID         string     `gorm:"column:collector_id;type:text" json:"collector_id,omitempty"`
	TaskID              string     `gorm:"column:task_id;type:text" json:"task_id,omitempty"`
	DeviceID            string     `gorm:"column:device_id;type:text" json:"device_id,omitempty"`
	CameraModel         string     `gorm:"column:camera_model;type:text" json:"camera_model,omitempty"`
	DataSource          string     `gorm:"column:data_source;type:text" json:"data_source,omitempty"`
	LocationID          string     `gorm:"column:location_id;type:text" json:"location_id,omitempty"`
	SceneID             string     `gorm:"column:scene_id;type:text" json:"scene_id,omitempty"`
	EnvironmentID       string     `gorm:"column:environment_id;type:text" json:"environment_id,omitempty"`
	CollectionMethod    string     `gorm:"column:collection_method;type:text" json:"collection_method,omitempty"`
	Owner               string     `gorm:"column:owner;type:text;not null;default:''" json:"owner"`
	RetentionTier       string     `gorm:"column:retention_tier;type:text" json:"retention_tier,omitempty"`
	ExpireAt            *time.Time `gorm:"column:expire_at;type:timestamptz" json:"expire_at,omitempty"`
	Metadata            string     `gorm:"column:metadata;type:jsonb;not null;default:'{}'" json:"metadata,omitempty"`
	ProcessState        string     `gorm:"column:process_state;type:jsonb;not null;default:'{}'" json:"process_state,omitempty"`
	RawHashSHA256       string     `gorm:"column:raw_hash_sha256;type:text" json:"raw_hash_sha256,omitempty"`
	TenantID            string     `gorm:"column:tenant_id;type:text" json:"tenant_id,omitempty"`
	ProjectID           string     `gorm:"column:project_id;type:text" json:"project_id,omitempty"`
	CreatedAt           time.Time  `gorm:"column:created_at;type:timestamptz;not null" json:"created_at"`
	UpdatedAt           time.Time  `gorm:"column:updated_at;type:timestamptz;not null" json:"updated_at"`
	Version             int64      `gorm:"column:version;type:bigint;default:1" json:"version"`
	SummaryIndexState   string     `gorm:"column:summary_index_state;type:text;not null;default:'pending'" json:"summary_index_state"`
	SummaryIndexVersion string     `gorm:"column:summary_index_version;type:text" json:"summary_index_version,omitempty"`
	SummaryIndexedAt    *time.Time `gorm:"column:summary_indexed_at;type:timestamptz" json:"summary_indexed_at,omitempty"`
	SummaryIndexAttempts int       `gorm:"column:summary_index_attempts;type:integer;not null;default:0" json:"summary_index_attempts"`
	SummaryIndexError   string     `gorm:"column:summary_index_error;type:text" json:"summary_index_error,omitempty"`
}

func (McapFile) TableName() string { return "mcap_files" }

// keep gorm import used
var _ = gorm.Model{}