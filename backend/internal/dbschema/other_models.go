package dbschema

import "time"

// Action — `actions` table (annotation/label rows).
type Action struct {
	ActionID     string    `gorm:"column:action_id;type:text;primaryKey" json:"action_id"`
	AssetID      string    `gorm:"column:asset_id;type:text;not null" json:"asset_id"`
	StartNs      int64     `gorm:"column:start_ns;type:bigint;not null" json:"start_ns"`
	EndNs        int64     `gorm:"column:end_ns;type:bigint;not null" json:"end_ns"`
	ActionIndex  *int      `gorm:"column:action_index;type:integer" json:"action_index,omitempty"`
	PrimaryLabel string    `gorm:"column:primary_label;type:text" json:"primary_label,omitempty"`
	Labels       string    `gorm:"column:labels;type:text[];not null;default:'{}'" json:"labels"`
	Description  string    `gorm:"column:description;type:text" json:"description,omitempty"`
	Attrs        string    `gorm:"column:attrs;type:jsonb;not null;default:'{}'" json:"attrs"`
	SourceType   string    `gorm:"column:source_type;type:text;not null;default:'human'" json:"source_type"`
	SourceName   string    `gorm:"column:source_name;type:text" json:"source_name,omitempty"`
	SourceVersion string   `gorm:"column:source_version;type:text" json:"source_version,omitempty"`
	RunID        string    `gorm:"column:run_id;type:text" json:"run_id,omitempty"`
	Confidence   *float64  `gorm:"column:confidence;type:double precision" json:"confidence,omitempty"`
	IsDeleted    bool      `gorm:"column:is_deleted;type:boolean;not null;default:false" json:"is_deleted"`
	ExternalID   string    `gorm:"column:external_id;type:text" json:"external_id,omitempty"`
	TenantID     string    `gorm:"column:tenant_id;type:text" json:"tenant_id,omitempty"`
	ProjectID    string    `gorm:"column:project_id;type:text" json:"project_id,omitempty"`
	Version      int64     `gorm:"column:version;type:bigint;not null;default:1" json:"version"`
	CreatedAt    time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
	TaskID       string    `gorm:"column:task_id;type:text" json:"task_id,omitempty"`
}

func (Action) TableName() string { return "actions" }

// AlgoRun — `algo_runs` table.
type AlgoRun struct {
	RunID             string     `gorm:"column:run_id;type:text;primaryKey" json:"run_id"`
	AlgoName          string     `gorm:"column:algo_name;type:text;not null" json:"algo_name"`
	AlgoVersion       string     `gorm:"column:algo_version;type:text;not null" json:"algo_version"`
	AlgoKind          string     `gorm:"column:algo_kind;type:text;not null" json:"algo_kind"`
	TriggeredBy       string     `gorm:"column:triggered_by;type:text;not null" json:"triggered_by"`
	Status            string     `gorm:"column:status;type:text;not null;default:'pending'" json:"status"`
	StartedAt         *time.Time `gorm:"column:started_at;type:timestamptz" json:"started_at,omitempty"`
	FinishedAt        *time.Time `gorm:"column:finished_at;type:timestamptz" json:"finished_at,omitempty"`
	DurationNs        *int64     `gorm:"column:duration_ns;type:bigint" json:"duration_ns,omitempty"` // GENERATED
	InputFilter       string     `gorm:"column:input_filter;type:jsonb;not null;default:'{}'" json:"input_filter"`
	InputAssetIDs     string     `gorm:"column:input_asset_ids;type:text[]" json:"input_asset_ids,omitempty"`
	Params            string     `gorm:"column:params;type:jsonb;not null;default:'{}'" json:"params"`
	CodeCommit        string     `gorm:"column:code_commit;type:text" json:"code_commit,omitempty"`
	ImageDigest       string     `gorm:"column:image_digest;type:text" json:"image_digest,omitempty"`
	PipelineName      string     `gorm:"column:pipeline_name;type:text" json:"pipeline_name,omitempty"`
	PipelineVersion   string     `gorm:"column:pipeline_version;type:text" json:"pipeline_version,omitempty"`
	AssetsProcessed   *int       `gorm:"column:assets_processed;type:integer" json:"assets_processed,omitempty"`
	AssetsSucceeded   *int       `gorm:"column:assets_succeeded;type:integer" json:"assets_succeeded,omitempty"`
	AssetsFailed      *int       `gorm:"column:assets_failed;type:integer" json:"assets_failed,omitempty"`
	ActionsCreated    *int       `gorm:"column:actions_created;type:integer" json:"actions_created,omitempty"`
	MetricsWritten    *int       `gorm:"column:metrics_written;type:integer" json:"metrics_written,omitempty"`
	Outputs           string     `gorm:"column:outputs;type:jsonb;not null;default:'{}'" json:"outputs"`
	CPUSeconds        *int64     `gorm:"column:cpu_seconds;type:bigint" json:"cpu_seconds,omitempty"`
	GPUSeconds        *int64     `gorm:"column:gpu_seconds;type:bigint" json:"gpu_seconds,omitempty"`
	CostUsdMicros     *int64     `gorm:"column:cost_usd_micros;type:bigint" json:"cost_usd_micros,omitempty"`
	ErrorClass        string     `gorm:"column:error_class;type:text" json:"error_class,omitempty"`
	ErrorMessage      string     `gorm:"column:error_message;type:text" json:"error_message,omitempty"`
	TenantID          string     `gorm:"column:tenant_id;type:text" json:"tenant_id,omitempty"`
	ProjectID         string     `gorm:"column:project_id;type:text" json:"project_id,omitempty"`
	ExternalRuntime   string     `gorm:"column:external_runtime;type:text" json:"external_runtime,omitempty"`
	ExternalURL       string     `gorm:"column:external_url;type:text" json:"external_url,omitempty"`
	CreatedAt         time.Time  `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt         time.Time  `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
}

func (AlgoRun) TableName() string { return "algo_runs" }

// AuditEvent — `audit_events` table.
type AuditEvent struct {
	EventID        string    `gorm:"column:event_id;type:uuid;primaryKey;default:gen_random_uuid()" json:"event_id"`
	Actor          string    `gorm:"column:actor;type:text;not null" json:"actor"`
	Action         string    `gorm:"column:action;type:text;not null" json:"action"`
	ResourceType   string    `gorm:"column:resource_type;type:text;not null" json:"resource_type"`
	ResourceIDs    string    `gorm:"column:resource_ids;type:text[];not null" json:"resource_ids"`
	RequestSummary string    `gorm:"column:request_summary;type:jsonb;not null;default:'{}'" json:"request_summary"` // drift fix: was not null
	RequestID      string    `gorm:"column:request_id;type:text" json:"request_id,omitempty"`
	CreatedAt      time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
}

func (AuditEvent) TableName() string { return "audit_events" }

// ESSyncCheckpoint — `es_sync_checkpoint` table.
type ESSyncCheckpoint struct {
	ShardID    int64     `gorm:"column:shard_id;type:bigint;primaryKey;autoIncrement" json:"shard_id"` // drift fix: was integer
	AppliedSeq int64     `gorm:"column:applied_seq;type:bigint;not null" json:"applied_seq"`
	UpdatedAt  time.Time `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
}

func (ESSyncCheckpoint) TableName() string { return "es_sync_checkpoint" }

// IdempotencyKey — `idempotency_keys` table.
type IdempotencyKey struct {
	Scope        string     `gorm:"column:scope;type:varchar(64);primaryKey" json:"scope"`
	IdemKey      string     `gorm:"column:idem_key;type:varchar(128);primaryKey" json:"idem_key"`
	RequestHash  string     `gorm:"column:request_hash;type:varchar(64)" json:"request_hash,omitempty"`
	StatusCode   *int       `gorm:"column:status_code;type:integer" json:"status_code,omitempty"`
	ResponseJSON string     `gorm:"column:response_json;type:jsonb" json:"response_json,omitempty"`
	CreatedAt    *time.Time `gorm:"column:created_at;type:timestamptz" json:"created_at,omitempty"`
}

func (IdempotencyKey) TableName() string { return "idempotency_keys" }

// LakehouseBronzeCheckpoint — `lakehouse_bronze_checkpoint` table.
type LakehouseBronzeCheckpoint struct {
	ID          int       `gorm:"column:id;type:integer;primaryKey" json:"id"`
	AppliedSeq  int64     `gorm:"column:applied_seq;type:bigint;not null" json:"applied_seq"`
	IngestedAt  time.Time `gorm:"column:ingested_at;type:timestamptz;not null" json:"ingested_at"`
	RunID       string    `gorm:"column:run_id;type:text" json:"run_id,omitempty"`
	UpdatedAt   time.Time `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
}

func (LakehouseBronzeCheckpoint) TableName() string { return "lakehouse_bronze_checkpoint" }

// OutboxDLQ — `outbox_dlq` table.
type OutboxDLQ struct {
	DLQID              int64     `gorm:"column:dlq_id;type:bigint;primaryKey" json:"dlq_id"`
	EventID            string    `gorm:"column:event_id;type:uuid;not null" json:"event_id"`
	EventSeq           int64     `gorm:"column:event_seq;type:bigint;not null" json:"event_seq"`
	EventType          string    `gorm:"column:event_type;type:text;not null" json:"event_type"`
	AggregateType      string    `gorm:"column:aggregate_type;type:text;not null;default:'asset'" json:"aggregate_type"`
	AssetID            string    `gorm:"column:asset_id;type:text" json:"asset_id,omitempty"`
	McapFileID         string    `gorm:"column:mcap_file_id;type:text" json:"mcap_file_id,omitempty"`
	EventSource        string    `gorm:"column:event_source;type:text;not null;default:'backend'" json:"event_source"`
	EventPayload       string    `gorm:"column:event_payload;type:jsonb;not null;default:'{}'" json:"event_payload"`
	RetryCount         int       `gorm:"column:retry_count;type:integer;not null;default:0" json:"retry_count"`
	LastError          string    `gorm:"column:last_error;type:text" json:"last_error,omitempty"`
	OriginalCreatedAt  time.Time `gorm:"column:original_created_at;type:timestamptz;not null" json:"original_created_at"`
	MovedAt            time.Time `gorm:"column:moved_at;type:timestamptz;not null;default:now()" json:"moved_at"`
	ResolvedAt         *time.Time `gorm:"column:resolved_at;type:timestamptz" json:"resolved_at,omitempty"`
	Resolution         string    `gorm:"column:resolution;type:text" json:"resolution,omitempty"`
}

func (OutboxDLQ) TableName() string { return "outbox_dlq" }

// SavedQuery — `saved_queries` table.
type SavedQuery struct {
	SavedQueryID   string    `gorm:"column:saved_query_id;type:uuid;primaryKey;default:gen_random_uuid()" json:"saved_query_id"`
	Name           string    `gorm:"column:name;type:text;not null" json:"name"`
	Description    string    `gorm:"column:description;type:text" json:"description,omitempty"`
	Resource       string    `gorm:"column:resource;type:text;not null;default:'assets'" json:"resource"`
	SchemaVersion  string    `gorm:"column:schema_version;type:text;not null;default:'v1'" json:"schema_version"`
	QueryIRJSON    string    `gorm:"column:query_ir_json;type:jsonb;not null" json:"query_ir_json"`
	Owner          string    `gorm:"column:owner;type:text" json:"owner,omitempty"`
	CreatedAt      time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
}

func (SavedQuery) TableName() string { return "saved_queries" }

// SearchReindexJob — `search_reindex_jobs` table.
type SearchReindexJob struct {
	ID                  string     `gorm:"column:id;type:text;primaryKey" json:"id"`
	Status              string     `gorm:"column:status;type:text;not null" json:"status"`
	DryRun              bool       `gorm:"column:dry_run;type:boolean;not null;default:false" json:"dry_run"`
	PageSize            int        `gorm:"column:page_size;type:integer;not null;default:200" json:"page_size"`
	NextPage            int        `gorm:"column:next_page;type:integer;not null;default:1" json:"next_page"`
	StopRequested       bool       `gorm:"column:stop_requested;type:boolean;not null;default:false" json:"stop_requested"`
	TotalAssets         int64      `gorm:"column:total_assets;type:bigint;not null;default:0" json:"total_assets"`
	AssetsScanned       int64      `gorm:"column:assets_scanned;type:bigint;not null;default:0" json:"assets_scanned"`
	DocumentsIndexed    int64      `gorm:"column:documents_indexed;type:bigint;not null;default:0" json:"documents_indexed"`
	Failed              int64      `gorm:"column:failed;type:bigint;not null;default:0" json:"failed"`
	Error               string     `gorm:"column:error;type:text;not null;default:''" json:"error"`
	ErrorSamples        string     `gorm:"column:error_samples;type:jsonb;not null;default:'[]'" json:"error_samples"`
	ElasticsearchDocCount *int64   `gorm:"column:elasticsearch_doc_count;type:bigint" json:"elasticsearch_doc_count,omitempty"`
	IndexCleared        bool       `gorm:"column:index_cleared;type:boolean;not null;default:false" json:"index_cleared"`
	CreatedAt           time.Time  `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt           time.Time  `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
	StartedAt           *time.Time `gorm:"column:started_at;type:timestamptz" json:"started_at,omitempty"`
	FinishedAt          *time.Time `gorm:"column:finished_at;type:timestamptz" json:"finished_at,omitempty"`
}

func (SearchReindexJob) TableName() string { return "search_reindex_jobs" }

// SyncWatermark — `sync_watermarks` table. Field name `Table` (not
// `TableName`) to avoid conflict with the GORM TableName() method.
type SyncWatermark struct {
	Table        string    `gorm:"column:table_name;type:text;primaryKey" json:"table_name"`
	Watermark    time.Time `gorm:"column:watermark;type:timestamptz;not null" json:"watermark"`
	DagsterRunID string    `gorm:"column:dagster_run_id;type:text" json:"dagster_run_id,omitempty"`
	UpdatedAt    time.Time `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
}

func (SyncWatermark) TableName() string { return "sync_watermarks" }

// VideoDuration — `video_durations` table (063).
type VideoDuration struct {
	VideoID     string    `gorm:"column:video_id;type:text;primaryKey" json:"video_id"`
	DurationSec float64   `gorm:"column:duration_sec;type:double precision;not null" json:"duration_sec"`
	UpdatedAt   time.Time `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
}

func (VideoDuration) TableName() string { return "video_durations" }