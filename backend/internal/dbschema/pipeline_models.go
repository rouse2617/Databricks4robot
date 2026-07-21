package dbschema

import (
	"time"
)

// PipelineTemplate — `pipeline_templates` (added in 039; altered by 042 to add
// version column).
type PipelineTemplate struct {
	ID            string    `gorm:"column:id;type:text;primaryKey" json:"id"`
	Name          string    `gorm:"column:name;type:text;not null" json:"name"`
	Version       int       `gorm:"column:version;type:integer;not null;default:1" json:"version"` // 042
	Pipeline      string    `gorm:"column:pipeline;type:jsonb;not null" json:"pipeline"`
	NodeCount     int       `gorm:"column:node_count;type:integer;not null;default:0" json:"node_count"`
	Scope         string    `gorm:"column:scope;type:text;default:'dev';not null" json:"scope"`         // 052
	Owner         string    `gorm:"column:owner;type:text;default:'';not null" json:"owner"`            // 052
	ActiveVersion *int      `gorm:"column:active_version;type:integer" json:"active_version,omitempty"` // 051
	CreatedAt     time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
}

func (PipelineTemplate) TableName() string { return "pipeline_templates" }

// PipelineDeployment — `pipeline_deployments` (039; altered by 052 to add scope/owner).
type PipelineDeployment struct {
	ID           string     `gorm:"column:id;type:text;primaryKey" json:"id"`
	TemplateID   string     `gorm:"column:template_id;type:text" json:"template_id,omitempty"`
	PipelineName string     `gorm:"column:pipeline_name;type:text;not null" json:"pipeline_name"`
	WorkflowName string     `gorm:"column:workflow_name;type:text;not null" json:"workflow_name"`
	Status       string     `gorm:"column:status;type:text;not null;default:'Pending'" json:"status"`
	NodeCount    int        `gorm:"column:node_count;type:integer;not null;default:0" json:"node_count"`
	Manifest     string     `gorm:"column:manifest;type:text" json:"manifest,omitempty"`
	PipelineJSON string     `gorm:"column:pipeline_json;type:jsonb" json:"pipeline_json,omitempty"`
	Scope        string     `gorm:"column:scope;type:text;default:'dev';not null" json:"scope"` // 052
	Owner        string     `gorm:"column:owner;type:text;default:'';not null" json:"owner"`    // 052
	CreatedAt    time.Time  `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt    time.Time  `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
	FinishedAt   *time.Time `gorm:"column:finished_at;type:timestamptz" json:"finished_at,omitempty"`
}

func (PipelineDeployment) TableName() string { return "pipeline_deployments" }

// PipelineComponent — `pipeline_components` (040).
type PipelineComponent struct {
	ID          string    `gorm:"column:id;type:text;primaryKey" json:"id"`
	Name        string    `gorm:"column:name;type:text;not null" json:"name"`
	Description string    `gorm:"column:description;type:text;not null;default:''" json:"description"`
	Image       string    `gorm:"column:image;type:text;not null" json:"image"`
	Tag         string    `gorm:"column:tag;type:text;not null;default:'latest'" json:"tag"`
	Source      string    `gorm:"column:source;type:text;not null;default:'custom'" json:"source"`
	InputPorts  string    `gorm:"column:input_ports;type:jsonb;not null;default:'[]'" json:"input_ports"`
	OutputPorts string    `gorm:"column:output_ports;type:jsonb;not null;default:'[]'" json:"output_ports"`
	Resources   string    `gorm:"column:resources;type:jsonb" json:"resources,omitempty"`
	EnvVars     string    `gorm:"column:env_vars;type:jsonb" json:"env_vars,omitempty"`
	Scope       string    `gorm:"column:scope;type:text;default:'dev';not null" json:"scope"` // 052
	Owner       string    `gorm:"column:owner;type:text;default:'';not null" json:"owner"`    // 052
	CreatedAt   time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
}

func (PipelineComponent) TableName() string { return "pipeline_components" }

// ExecutionTarget — `execution_targets` (041).
type ExecutionTarget struct {
	ID                     string    `gorm:"column:id;type:text;primaryKey" json:"id"`
	Name                   string    `gorm:"column:name;type:text;not null" json:"name"`
	Description            string    `gorm:"column:description;type:text;not null;default:''" json:"description"`
	Cluster                string    `gorm:"column:cluster;type:text;not null" json:"cluster"`
	Namespace              string    `gorm:"column:namespace;type:text;not null" json:"namespace"`
	ServiceAccount         string    `gorm:"column:service_account;type:text;not null;default:''" json:"service_account"`
	ArgoServerURL          string    `gorm:"column:argo_server_url;type:text;not null;default:''" json:"argo_server_url"`
	ArgoAuthSecretRef      string    `gorm:"column:argo_auth_secret_ref;type:text;not null;default:''" json:"argo_auth_secret_ref"`
	ArgoInsecureSkipVerify bool      `gorm:"column:argo_insecure_skip_verify;type:boolean;not null;default:false" json:"argo_insecure_skip_verify"`
	ArgoCACertRef          string    `gorm:"column:argo_ca_cert_ref;type:text;not null;default:''" json:"argo_ca_cert_ref"`
	Enabled                bool      `gorm:"column:enabled;type:boolean;not null;default:true" json:"enabled"`
	Status                 string    `gorm:"column:status;type:text;not null;default:'available'" json:"status"`
	IsDefault              bool      `gorm:"column:is_default;type:boolean;not null;default:false" json:"is_default"`
	ResourceDefaults       string    `gorm:"column:resource_defaults;type:jsonb;not null;default:'{}'" json:"resource_defaults"`
	QuotaPolicy            string    `gorm:"column:quota_policy;type:jsonb;not null;default:'{}'" json:"quota_policy"`
	Labels                 string    `gorm:"column:labels;type:jsonb;not null;default:'{}'" json:"labels"`
	CreatedAt              time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt              time.Time `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
}

func (ExecutionTarget) TableName() string { return "execution_targets" }

// PipelineRun — `pipeline_runs` (041; altered by 050 to add ledger_state,
// 052 to add scope/owner, 053 to add backfill_job_id).
type PipelineRun struct {
	ID                string     `gorm:"column:id;type:text;primaryKey" json:"id"`
	TemplateID        *string    `gorm:"column:template_id;type:text" json:"template_id,omitempty"`
	PipelineName      string     `gorm:"column:pipeline_name;type:text;not null" json:"pipeline_name"`
	TemplateVersion   *int       `gorm:"column:template_version;type:integer" json:"template_version,omitempty"`
	WorkflowName      string     `gorm:"column:workflow_name;type:text;not null;uniqueIndex" json:"workflow_name"`
	ExecutionTargetID string     `gorm:"column:execution_target_id;type:text;not null" json:"execution_target_id"`
	TargetSnapshot    string     `gorm:"column:target_snapshot;type:jsonb;not null;default:'{}'" json:"target_snapshot"`
	Status            string     `gorm:"column:status;type:text;not null;default:'Pending'" json:"status"`
	NodeCount         int        `gorm:"column:node_count;type:integer;not null;default:0" json:"node_count"`
	AssetIDs          string     `gorm:"column:asset_ids;type:text[];not null;default:'{}'" json:"asset_ids"`
	AssetCount        int        `gorm:"column:asset_count;type:integer;not null;default:0" json:"asset_count"`
	NoAssetRun        bool       `gorm:"column:no_asset_run;type:boolean;not null;default:false" json:"no_asset_run"`
	Manifest          string     `gorm:"column:manifest;type:text" json:"manifest,omitempty"`
	PipelineJSON      string     `gorm:"column:pipeline_json;type:jsonb;not null;default:'{}'" json:"pipeline_json"`
	ArgoNamespace     string     `gorm:"column:argo_namespace;type:text;not null" json:"argo_namespace"`
	ArgoWorkflowUID   string     `gorm:"column:argo_workflow_uid;type:text;not null;default:''" json:"argo_workflow_uid"`
	Message           string     `gorm:"column:message;type:text;not null;default:''" json:"message"`
	Progress          string     `gorm:"column:progress;type:text;not null;default:''" json:"progress,omitempty"`      // CYB-3490
	Scope             string     `gorm:"column:scope;type:text;default:'dev';not null" json:"scope"`                   // 052
	Owner             string     `gorm:"column:owner;type:text;default:'';not null" json:"owner"`                      // 052
	LedgerState       string     `gorm:"column:ledger_state;type:text;not null;default:'pending'" json:"ledger_state"` // 050
	BackfillJobID     *string    `gorm:"column:backfill_job_id;type:text" json:"backfill_job_id,omitempty"`            // 053
	CreatedAt         time.Time  `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt         time.Time  `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
	StartedAt         *time.Time `gorm:"column:started_at;type:timestamptz" json:"started_at,omitempty"`
	FinishedAt        *time.Time `gorm:"column:finished_at;type:timestamptz" json:"finished_at,omitempty"`
}

func (PipelineRun) TableName() string { return "pipeline_runs" }

// PipelineRunNode — `pipeline_run_nodes` (041; altered by 046 to add
// estimated_cost_usd).
type PipelineRunNode struct {
	ID                string     `gorm:"column:id;type:text;primaryKey" json:"id"`
	RunID             string     `gorm:"column:run_id;type:text;not null" json:"run_id"`
	PipelineNodeID    string     `gorm:"column:pipeline_node_id;type:text;not null" json:"pipeline_node_id"`
	ArgoNodeID        string     `gorm:"column:argo_node_id;type:text;not null;default:''" json:"argo_node_id"`
	ArgoNodeName      string     `gorm:"column:argo_node_name;type:text;not null;default:''" json:"argo_node_name"`
	DisplayName       string     `gorm:"column:display_name;type:text;not null;default:''" json:"display_name"`
	TemplateName      string     `gorm:"column:template_name;type:text;not null;default:''" json:"template_name"`
	Type              string     `gorm:"column:type;type:text;not null;default:''" json:"type"`
	Phase             string     `gorm:"column:phase;type:text;not null;default:''" json:"phase"`
	Message           string     `gorm:"column:message;type:text;not null;default:''" json:"message"`
	PodName           string     `gorm:"column:pod_name;type:text;not null;default:''" json:"pod_name"`
	HostNodeName      string     `gorm:"column:host_node_name;type:text;not null;default:''" json:"host_node_name"`
	Children          string     `gorm:"column:children;type:text[];not null;default:'{}'" json:"children"`
	Inputs            string     `gorm:"column:inputs;type:jsonb;not null;default:'{}'" json:"inputs"`
	Outputs           string     `gorm:"column:outputs;type:jsonb;not null;default:'{}'" json:"outputs"`
	ResourcesDuration string     `gorm:"column:resources_duration;type:jsonb;not null;default:'{}'" json:"resources_duration"`
	ResourceSummary   string     `gorm:"column:resource_summary;type:jsonb;not null;default:'{}'" json:"resource_summary"`
	LogRef            string     `gorm:"column:log_ref;type:text;not null;default:''" json:"log_ref"`
	EstimatedCostUsd  *float64   `gorm:"column:estimated_cost_usd;type:decimal(16,8)" json:"estimated_cost_usd,omitempty"` // 046
	StartedAt         *time.Time `gorm:"column:started_at;type:timestamptz" json:"started_at,omitempty"`
	FinishedAt        *time.Time `gorm:"column:finished_at;type:timestamptz" json:"finished_at,omitempty"`
	CreatedAt         time.Time  `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt         time.Time  `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
}

func (PipelineRunNode) TableName() string { return "pipeline_run_nodes" }

// PipelineRunEvent — `pipeline_run_events` (047; outbox pattern).
type PipelineRunEvent struct {
	ID             string    `gorm:"column:id;type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	RunID          string    `gorm:"column:run_id;type:text;not null" json:"run_id"`
	WorkflowName   string    `gorm:"column:workflow_name;type:text" json:"workflow_name,omitempty"`
	EventType      string    `gorm:"column:event_type;type:text;not null" json:"event_type"`
	SubjectType    string    `gorm:"column:subject_type;type:text;not null" json:"subject_type"`
	SubjectID      string    `gorm:"column:subject_id;type:text;not null" json:"subject_id"`
	Status         string    `gorm:"column:status;type:text" json:"status,omitempty"`
	Message        string    `gorm:"column:message;type:text" json:"message,omitempty"`
	Reason         string    `gorm:"column:reason;type:text" json:"reason,omitempty"`
	Payload        string    `gorm:"column:payload;type:jsonb;not null;default:'{}'" json:"payload"`
	IdempotencyKey string    `gorm:"column:idempotency_key;type:text;not null" json:"idempotency_key"`
	Sequence       int64     `gorm:"column:sequence;type:bigint;not null;autoIncrement" json:"sequence"`
	OccurredAt     time.Time `gorm:"column:occurred_at;type:timestamptz;not null;default:now()" json:"occurred_at"`
	ObservedAt     time.Time `gorm:"column:observed_at;type:timestamptz;not null;default:now()" json:"observed_at"`
	CreatedAt      time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
}

func (PipelineRunEvent) TableName() string { return "pipeline_run_events" }

// PipelineRunAssetNode — `pipeline_run_asset_nodes` (048; altered by
// cost_source default and id).
type PipelineRunAssetNode struct {
	ID               string     `gorm:"column:id;type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	RunID            string     `gorm:"column:run_id;type:text;not null" json:"run_id"`
	AssetID          string     `gorm:"column:asset_id;type:text;not null" json:"asset_id"`
	PipelineNodeID   string     `gorm:"column:pipeline_node_id;type:text;not null" json:"pipeline_node_id"`
	ArgoNodeID       string     `gorm:"column:argo_node_id;type:text" json:"argo_node_id,omitempty"`
	DisplayName      string     `gorm:"column:display_name;type:text" json:"display_name,omitempty"`
	Status           string     `gorm:"column:status;type:text" json:"status,omitempty"`
	Message          string     `gorm:"column:message;type:text" json:"message,omitempty"`
	PodName          string     `gorm:"column:pod_name;type:text" json:"pod_name,omitempty"`
	LogRef           string     `gorm:"column:log_ref;type:text" json:"log_ref,omitempty"`
	EstimatedCostUsd *float64   `gorm:"column:estimated_cost_usd;type:decimal(16,8)" json:"estimated_cost_usd,omitempty"`
	CostSource       string     `gorm:"column:cost_source;type:text;not null;default:'not_available'" json:"cost_source"`
	StartedAt        *time.Time `gorm:"column:started_at;type:timestamptz" json:"started_at,omitempty"`
	FinishedAt       *time.Time `gorm:"column:finished_at;type:timestamptz" json:"finished_at,omitempty"`
	UpdatedAt        time.Time  `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
}

func (PipelineRunAssetNode) TableName() string { return "pipeline_run_asset_nodes" }

// PipelineRunNotificationCandidate — `pipeline_run_notification_candidates` (048).
type PipelineRunNotificationCandidate struct {
	ID             string    `gorm:"column:id;type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	RunID          string    `gorm:"column:run_id;type:text;not null" json:"run_id"`
	EventID        string    `gorm:"column:event_id;type:text;not null" json:"event_id"`
	EventType      string    `gorm:"column:event_type;type:text;not null" json:"event_type"`
	SubjectType    string    `gorm:"column:subject_type;type:text;not null" json:"subject_type"`
	SubjectID      string    `gorm:"column:subject_id;type:text;not null" json:"subject_id"`
	Status         string    `gorm:"column:status;type:text" json:"status,omitempty"`
	Message        string    `gorm:"column:message;type:text" json:"message,omitempty"`
	SinkType       string    `gorm:"column:sink_type;type:text;not null;default:'candidate'" json:"sink_type"`
	DeliveryStatus string    `gorm:"column:delivery_status;type:text;not null;default:'pending'" json:"delivery_status"`
	IdempotencyKey string    `gorm:"column:idempotency_key;type:text;not null" json:"idempotency_key"`
	CreatedAt      time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
}

func (PipelineRunNotificationCandidate) TableName() string {
	return "pipeline_run_notification_candidates"
}

// PipelineRunWatcherState — `pipeline_run_watcher_state` (048; altered by
// 049 to add health columns).
type PipelineRunWatcherState struct {
	ID              string     `gorm:"column:id;type:text;primaryKey" json:"id"`
	LastSyncedAt    *time.Time `gorm:"column:last_synced_at;type:timestamptz" json:"last_synced_at,omitempty"`
	ActiveScanLimit int        `gorm:"column:active_scan_limit;type:integer;not null;default:100" json:"active_scan_limit"`
	LastError       string     `gorm:"column:last_error;type:text" json:"last_error,omitempty"`
	HealthStatus    string     `gorm:"column:health_status;type:text;default:'healthy'" json:"health_status"`        // 049
	HealthMessage   string     `gorm:"column:health_message;type:text" json:"health_message,omitempty"`              // 049
	HealthCheckedAt *time.Time `gorm:"column:health_checked_at;type:timestamptz" json:"health_checked_at,omitempty"` // 049
	UpdatedAt       time.Time  `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
}

func (PipelineRunWatcherState) TableName() string { return "pipeline_run_watcher_state" }

// PipelineComponentRelease — `pipeline_component_releases` (056).
type PipelineComponentRelease struct {
	ID                string     `gorm:"column:id;type:text;primaryKey" json:"id"`
	ComponentID       string     `gorm:"column:component_id;type:text;not null" json:"component_id"`
	TaskName          string     `gorm:"column:task_name;type:text;not null" json:"task_name"`
	TaskPath          string     `gorm:"column:task_path;type:text;not null;default:''" json:"task_path"`
	DisplayName       string     `gorm:"column:display_name;type:text;not null;default:''" json:"display_name"`
	Owner             string     `gorm:"column:owner;type:text;not null;default:''" json:"owner"`
	ReleaseLabel      string     `gorm:"column:release_label;type:text;not null" json:"release_label"`
	Channel           string     `gorm:"column:channel;type:text;not null;default:'dev'" json:"channel"`
	SourceRepo        string     `gorm:"column:source_repo;type:text;not null;default:''" json:"source_repo"`
	SourceRef         string     `gorm:"column:source_ref;type:text;not null;default:''" json:"source_ref"`
	SourceCommit      string     `gorm:"column:source_commit;type:text;not null;default:''" json:"source_commit"`
	BuildID           string     `gorm:"column:build_id;type:text;not null;default:''" json:"build_id"`
	ImageRepo         string     `gorm:"column:image_repo;type:text;not null;default:''" json:"image_repo"`
	ImageTag          string     `gorm:"column:image_tag;type:text;not null;default:''" json:"image_tag"`
	ImageDigest       string     `gorm:"column:image_digest;type:text;not null;default:''" json:"image_digest"`
	RuntimeImage      string     `gorm:"column:runtime_image;type:text;not null;default:''" json:"runtime_image"`
	Status            string     `gorm:"column:status;type:text;not null;default:'failed'" json:"status"`
	Selectable        bool       `gorm:"column:selectable;type:boolean;not null;default:false" json:"selectable"`
	ValidationStatus  string     `gorm:"column:validation_status;type:text;not null;default:'failed'" json:"validation_status"`
	ValidationErrors  string     `gorm:"column:validation_errors;type:jsonb;not null;default:'[]'" json:"validation_errors"`
	RuntimeSnapshot   string     `gorm:"column:runtime_snapshot;type:jsonb;not null;default:'{}'" json:"runtime_snapshot"`
	TechnicalMetadata string     `gorm:"column:technical_metadata;type:jsonb;not null;default:'{}'" json:"technical_metadata"`
	CreatedAt         time.Time  `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt         time.Time  `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
	LastSyncedAt      *time.Time `gorm:"column:last_synced_at;type:timestamptz" json:"last_synced_at,omitempty"`
}

func (PipelineComponentRelease) TableName() string { return "pipeline_component_releases" }

// PipelineConfig — `pipeline_configs` (057; unique on name added in 059).
type PipelineConfig struct {
	ID             string    `gorm:"column:id;type:text;primaryKey" json:"id"`
	Name           string    `gorm:"column:name;type:text;not null;uniqueIndex" json:"name"` // 059 unique
	Description    string    `gorm:"column:description;type:text;not null;default:''" json:"description"`
	Owner          string    `gorm:"column:owner;type:text;not null;default:''" json:"owner"`
	Scope          string    `gorm:"column:scope;type:varchar(16);not null;default:'dev'" json:"scope"`
	Tags           string    `gorm:"column:tags;type:jsonb;not null;default:'[]'" json:"tags"`
	FileType       string    `gorm:"column:file_type;type:text;not null" json:"file_type"`
	Lifecycle      string    `gorm:"column:lifecycle;type:text;not null;default:'draft'" json:"lifecycle"`
	CurrentVersion int       `gorm:"column:current_version;type:integer;not null;default:1" json:"current_version"`
	CreatedAt      time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
}

func (PipelineConfig) TableName() string { return "pipeline_configs" }

// PipelineConfigVersion — `pipeline_config_versions` (057).
type PipelineConfigVersion struct {
	ID               string    `gorm:"column:id;type:text;primaryKey" json:"id"`
	ConfigID         string    `gorm:"column:config_id;type:text;not null" json:"config_id"`
	Version          int       `gorm:"column:version;type:integer;not null;uniqueIndex:idx_pipeline_config_versions_config_version" json:"version"`
	Status           string    `gorm:"column:status;type:text;not null;default:'draft'" json:"status"`
	Content          string    `gorm:"column:content;type:text;not null" json:"content"`
	ContentSHA256    string    `gorm:"column:content_sha256;type:text;not null" json:"content_sha256"`
	ContentSizeBytes int       `gorm:"column:content_size_bytes;type:integer;not null" json:"content_size_bytes"`
	Summary          string    `gorm:"column:summary;type:text;not null;default:''" json:"summary"`
	Author           string    `gorm:"column:author;type:text;not null;default:''" json:"author"`
	CreatedAt        time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
}

func (PipelineConfigVersion) TableName() string { return "pipeline_config_versions" }

// DatabrewRun — `databrew_runs` (055; unifies pipeline / component / rag runs).
type DatabrewRun struct {
	ID                  string     `gorm:"column:id;type:text;primaryKey" json:"id"`
	Type                string     `gorm:"column:type;type:text;not null" json:"type"`
	Name                string     `gorm:"column:name;type:text;not null" json:"name"`
	Status              string     `gorm:"column:status;type:text;not null;default:'Pending'" json:"status"`
	Runtime             string     `gorm:"column:runtime;type:text;not null;default:'argo'" json:"runtime"`
	RuntimeNamespace    string     `gorm:"column:runtime_namespace;type:text;not null;default:''" json:"runtime_namespace"`
	RuntimeResourceName string     `gorm:"column:runtime_resource_name;type:text;not null" json:"runtime_resource_name"`
	RuntimeUID          string     `gorm:"column:runtime_uid;type:text;not null;default:''" json:"runtime_uid"`
	Owner               string     `gorm:"column:owner;type:text;not null;default:''" json:"owner"`
	CreatedBy           string     `gorm:"column:created_by;type:text;not null;default:''" json:"created_by"`
	Message             string     `gorm:"column:message;type:text;not null;default:''" json:"message"`
	CreatedAt           time.Time  `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	StartedAt           *time.Time `gorm:"column:started_at;type:timestamptz" json:"started_at,omitempty"`
	FinishedAt          *time.Time `gorm:"column:finished_at;type:timestamptz" json:"finished_at,omitempty"`
	UpdatedAt           time.Time  `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
}

func (DatabrewRun) TableName() string { return "databrew_runs" }

// ComponentBuildRun — `component_build_runs` (055).
type ComponentBuildRun struct {
	RunID           string `gorm:"column:run_id;type:text;primaryKey" json:"run_id"`
	ComponentID     string `gorm:"column:component_id;type:text;not null;default:''" json:"component_id"`
	RepoURL         string `gorm:"column:repo_url;type:text;not null;default:''" json:"repo_url"`
	GitRef          string `gorm:"column:git_ref;type:text;not null;default:''" json:"git_ref"`
	CommitSHA       string `gorm:"column:commit_sha;type:text;not null;default:''" json:"commit_sha"`
	Dockerfile      string `gorm:"column:dockerfile;type:text;not null;default:'Dockerfile'" json:"dockerfile"`
	BuildContext    string `gorm:"column:build_context;type:text;not null;default:'.'" json:"build_context"`
	ImageRepository string `gorm:"column:image_repository;type:text;not null;default:''" json:"image_repository"`
	ImageTag        string `gorm:"column:image_tag;type:text;not null;default:''" json:"image_tag"`
	ImageDigest     string `gorm:"column:image_digest;type:text;not null;default:''" json:"image_digest"`
}

func (ComponentBuildRun) TableName() string { return "component_build_runs" }

// RagBuildRun — `rag_build_runs` (055).
type RagBuildRun struct {
	RunID              string `gorm:"column:run_id;type:text;primaryKey" json:"run_id"`
	KnowledgeBaseID    string `gorm:"column:knowledge_base_id;type:text;not null;default:''" json:"knowledge_base_id"`
	DatasourceSnapshot string `gorm:"column:datasource_snapshot;type:jsonb;not null;default:'{}'" json:"datasource_snapshot"`
	EmbeddingModel     string `gorm:"column:embedding_model;type:text;not null;default:''" json:"embedding_model"`
	VectorIndexName    string `gorm:"column:vector_index_name;type:text;not null;default:''" json:"vector_index_name"`
	ReleaseVersion     string `gorm:"column:release_version;type:text;not null;default:''" json:"release_version"`
}

func (RagBuildRun) TableName() string { return "rag_build_runs" }

// ComponentRelease — `component_releases` (055).
type ComponentRelease struct {
	ID           string    `gorm:"column:id;type:text;primaryKey" json:"id"`
	ComponentID  string    `gorm:"column:component_id;type:text;not null" json:"component_id"`
	SourceCommit string    `gorm:"column:source_commit;type:text;not null;default:''" json:"source_commit"`
	Image        string    `gorm:"column:image;type:text;not null;default:''" json:"image"`
	ImageTag     string    `gorm:"column:image_tag;type:text;not null;default:''" json:"image_tag"`
	ImageDigest  string    `gorm:"column:image_digest;type:text;not null;default:''" json:"image_digest"`
	ReleaseLabel string    `gorm:"column:release_label;type:text;not null;default:''" json:"release_label"`
	BuildRunID   *string   `gorm:"column:build_run_id;type:text" json:"build_run_id,omitempty"`
	CreatedAt    time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
}

func (ComponentRelease) TableName() string { return "component_releases" }

// RunRelation — `run_relations` (058).
type RunRelation struct {
	ID           string    `gorm:"column:id;type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ParentRunID  string    `gorm:"column:parent_run_id;type:text;not null" json:"parent_run_id"`
	ChildRunID   string    `gorm:"column:child_run_id;type:text;not null" json:"child_run_id"`
	RelationType string    `gorm:"column:relation_type;type:text;not null" json:"relation_type"`
	AssetID      string    `gorm:"column:asset_id;type:text;not null;default:''" json:"asset_id"`
	Source       string    `gorm:"column:source;type:text;not null;default:''" json:"source"`
	Snapshot     string    `gorm:"column:snapshot;type:jsonb;not null;default:'{}'" json:"snapshot"`
	CreatedAt    time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
}

func (RunRelation) TableName() string { return "run_relations" }

// RunInput — `run_inputs` (058).
type RunInput struct {
	ID             string    `gorm:"column:id;type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	RunID          string    `gorm:"column:run_id;type:text;not null" json:"run_id"`
	NodeID         string    `gorm:"column:node_id;type:text;not null;default:''" json:"node_id"`
	Type           string    `gorm:"column:type;type:text;not null" json:"type"`
	RefID          string    `gorm:"column:ref_id;type:text;not null;default:''" json:"ref_id"`
	RefVersion     string    `gorm:"column:ref_version;type:text;not null;default:''" json:"ref_version"`
	FileName       string    `gorm:"column:file_name;type:text;not null;default:''" json:"file_name"`
	MountPath      string    `gorm:"column:mount_path;type:text;not null;default:''" json:"mount_path"`
	TargetFilename string    `gorm:"column:target_filename;type:text;not null;default:''" json:"target_filename"`
	ContentHash    string    `gorm:"column:content_hash;type:text;not null;default:''" json:"content_hash"`
	ProjectionKey  string    `gorm:"column:projection_key;type:text;not null;default:''" json:"projection_key"`
	Source         string    `gorm:"column:source;type:text;not null;default:''" json:"source"`
	Snapshot       string    `gorm:"column:snapshot;type:jsonb;not null;default:'{}'" json:"snapshot"`
	CreatedAt      time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
}

func (RunInput) TableName() string { return "run_inputs" }
