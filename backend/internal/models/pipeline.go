package models

import "time"

// PipelineTemplate represents a reusable pipeline template stored in the
// pipeline_templates table. The Pipeline field holds the full pipeline
// definition as arbitrary JSON.
type PipelineTemplate struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Version       int                    `json:"version"`
	VersionCount  int                    `json:"versionCount,omitempty"`
	ActiveVersion int                    `json:"activeVersion,omitempty"`
	Scope         string                 `json:"scope,omitempty"`
	Owner         string                 `json:"owner,omitempty"`
	Pipeline      map[string]interface{} `json:"pipeline"`
	NodeCount     int                    `json:"nodeCount"`
	CreatedAt     time.Time              `json:"createdAt"`
	UpdatedAt     time.Time              `json:"updatedAt"`
}

// PipelineDeployment represents a single deployment of a pipeline template
// to a workflow run. Each deployment records the target workflow, current
// status, and optionally the rendered manifest and pipeline JSON snapshot.
type PipelineDeployment struct {
	ID              string                 `json:"id"`
	TemplateID      *string                `json:"templateId,omitempty"`
	TemplateVersion *int                   `json:"templateVersion,omitempty"`
	PipelineName    string                 `json:"pipelineName"`
	WorkflowName    string                 `json:"workflowName"`
	Status          string                 `json:"status"`
	NodeCount       int                    `json:"nodeCount"`
	AssetIDs        []string               `json:"assetIds,omitempty"`
	AssetCount      int                    `json:"assetCount"`
	ExecutionTarget *ExecutionTarget       `json:"executionTarget,omitempty"`
	Scope           string                 `json:"scope,omitempty"`
	Owner           string                 `json:"owner,omitempty"`
	Manifest        *string                `json:"manifest,omitempty"`
	PipelineJSON    map[string]interface{} `json:"pipelineJSON,omitempty"`
	CreatedAt       time.Time              `json:"createdAt"`
	UpdatedAt       time.Time              `json:"updatedAt"`
	FinishedAt      *time.Time             `json:"finishedAt,omitempty"`
}

// ExecutionTarget describes a runtime destination for pipeline workflows.
type ExecutionTarget struct {
	ID                   string                 `json:"id"`
	Name                 string                 `json:"name"`
	Cluster              string                 `json:"cluster"`
	Namespace            string                 `json:"namespace"`
	ServiceAccount       string                 `json:"serviceAccount,omitempty"`
	ArgoServerURL        string                 `json:"argoServerUrl,omitempty"`
	ArgoAuthSecretRef    string                 `json:"argoAuthSecretRef,omitempty"`
	ArgoInsecureSkipTLS  bool                   `json:"argoInsecureSkipTls"`
	ArgoCACertRef        string                 `json:"argoCaCertRef,omitempty"`
	ArgoServerConfigured bool                   `json:"argoServerConfigured"`
	Status               string                 `json:"status"`
	Enabled              bool                   `json:"enabled"`
	IsDefault            bool                   `json:"isDefault"`
	Description          string                 `json:"description,omitempty"`
	ResourceDefaults     map[string]interface{} `json:"resourceDefaults,omitempty"`
	QuotaPolicy          map[string]interface{} `json:"quotaPolicy,omitempty"`
	Labels               map[string]interface{} `json:"labels,omitempty"`
	CreatedAt            time.Time              `json:"createdAt,omitempty"`
	UpdatedAt            time.Time              `json:"updatedAt,omitempty"`
}

// PipelineRun is the first-class execution record for a pipeline run. Legacy
// deployment endpoints can still project this data as PipelineDeployment.
type PipelineRun struct {
	ID                 string                 `json:"id"`
	TemplateID         *string                `json:"templateId,omitempty"`
	PipelineName       string                 `json:"pipelineName"`
	TemplateVersion    *int                   `json:"templateVersion,omitempty"`
	WorkflowName       string                 `json:"workflowName"`
	ExecutionTargetID  string                 `json:"executionTargetId"`
	TargetSnapshot     map[string]interface{} `json:"targetSnapshot,omitempty"`
	Status             string                 `json:"status"`
	NodeCount          int                    `json:"nodeCount"`
	AssetIDs           []string               `json:"assetIds,omitempty"`
	AssetCount         int                    `json:"assetCount"`
	NoAssetRun         bool                   `json:"noAssetRun"`
	Manifest           *string                `json:"manifest,omitempty"`
	PipelineJSON       map[string]interface{} `json:"pipelineJSON,omitempty"`
	ArgoNamespace      string                 `json:"argoNamespace"`
	ArgoWorkflowUID    string                 `json:"argoWorkflowUid,omitempty"`
	Message            string                 `json:"message,omitempty"`
	ExecutionTarget    *ExecutionTarget       `json:"executionTarget,omitempty"`
	Nodes              []PipelineRunNode      `json:"nodes,omitempty"`
	TotalEstimatedCost *float64               `json:"totalEstimatedCost,omitempty"`
	Scope              string                 `json:"scope,omitempty"`
	Owner              string                 `json:"owner,omitempty"`
	BatchJobID         *string                `json:"batchJobId,omitempty"`
	LedgerState        string                 `json:"ledgerState"`
	CreatedAt          time.Time              `json:"createdAt"`
	UpdatedAt          time.Time              `json:"updatedAt"`
	StartedAt          *time.Time             `json:"startedAt,omitempty"`
	FinishedAt         *time.Time             `json:"finishedAt,omitempty"`
}

// PipelineRunNode captures per-node Argo state for a pipeline run.
type PipelineRunNode struct {
	ID                string                 `json:"id"`
	RunID             string                 `json:"runId"`
	PipelineNodeID    string                 `json:"pipelineNodeId"`
	ArgoNodeID        string                 `json:"argoNodeId,omitempty"`
	ArgoNodeName      string                 `json:"argoNodeName,omitempty"`
	DisplayName       string                 `json:"displayName,omitempty"`
	TemplateName      string                 `json:"templateName,omitempty"`
	Type              string                 `json:"type,omitempty"`
	Phase             string                 `json:"phase,omitempty"`
	Message           string                 `json:"message,omitempty"`
	PodName           string                 `json:"podName,omitempty"`
	HostNodeName      string                 `json:"hostNodeName,omitempty"`
	Children          []string               `json:"children,omitempty"`
	Inputs            map[string]interface{} `json:"inputs,omitempty"`
	Outputs           map[string]interface{} `json:"outputs,omitempty"`
	ResourcesDuration map[string]interface{} `json:"resourcesDuration,omitempty"`
	ResourceSummary   map[string]interface{} `json:"resourceSummary,omitempty"`
	LogRef            string                 `json:"logRef,omitempty"`
	EstimatedCostUSD  *float64               `json:"estimatedCostUsd,omitempty"`
	StartedAt         *time.Time             `json:"startedAt,omitempty"`
	FinishedAt        *time.Time             `json:"finishedAt,omitempty"`
	CreatedAt         time.Time              `json:"createdAt"`
	UpdatedAt         time.Time              `json:"updatedAt"`
}

// PipelineRunEvent is a durable, append-only event in a pipeline run timeline.
type PipelineRunEvent struct {
	ID             string                 `json:"id"`
	RunID          string                 `json:"runId"`
	WorkflowName   string                 `json:"workflowName,omitempty"`
	EventType      string                 `json:"eventType"`
	SubjectType    string                 `json:"subjectType"`
	SubjectID      string                 `json:"subjectId"`
	Status         string                 `json:"status,omitempty"`
	Message        string                 `json:"message,omitempty"`
	Reason         string                 `json:"reason,omitempty"`
	Payload        map[string]interface{} `json:"payload,omitempty"`
	IdempotencyKey string                 `json:"idempotencyKey,omitempty"`
	Sequence       int64                  `json:"sequence"`
	OccurredAt     time.Time              `json:"occurredAt"`
	ObservedAt     time.Time              `json:"observedAt"`
	CreatedAt      time.Time              `json:"createdAt"`
}

// PipelineRunEventListOptions controls run event pagination and filtering.
type PipelineRunEventListOptions struct {
	Limit       int
	Cursor      int64
	SubjectType string
	EventType   string
	Status      string
	Query       string
	From        *time.Time
	To          *time.Time
}

// PipelineRunEventListResult is the API response for a run event timeline page.
type PipelineRunEventListResult struct {
	Items      []PipelineRunEvent `json:"items"`
	NextCursor *int64             `json:"nextCursor,omitempty"`
	Total      int                `json:"total"`
}

// PipelineRunAssetNode is a derived execution snapshot for one asset at one
// pipeline node. For no-asset runs AssetID is "no-asset".
type PipelineRunAssetNode struct {
	ID               string     `json:"id"`
	RunID            string     `json:"runId"`
	AssetID          string     `json:"assetId"`
	PipelineNodeID   string     `json:"pipelineNodeId"`
	ArgoNodeID       string     `json:"argoNodeId,omitempty"`
	DisplayName      string     `json:"displayName,omitempty"`
	Status           string     `json:"status,omitempty"`
	Message          string     `json:"message,omitempty"`
	PodName          string     `json:"podName,omitempty"`
	LogRef           string     `json:"logRef,omitempty"`
	EstimatedCostUSD *float64   `json:"estimatedCostUsd,omitempty"`
	CostSource       string     `json:"costSource"`
	StartedAt        *time.Time `json:"startedAt,omitempty"`
	FinishedAt       *time.Time `json:"finishedAt,omitempty"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

// PipelineRunAssetNodeListOptions controls asset-node pagination/filtering.
type PipelineRunAssetNodeListOptions struct {
	Limit    int
	Cursor   string
	AssetID  string
	NodeID   string
	Status   string
	OrderBy  string
	OrderDir string
}

// PipelineRunAssetNodeListResult is the API response for asset-node rows.
type PipelineRunAssetNodeListResult struct {
	Items      []PipelineRunAssetNode      `json:"items"`
	NextCursor *string                     `json:"nextCursor,omitempty"`
	Total      int                         `json:"total"`
	Summary    PipelineRunAssetNodeSummary `json:"summary"`
}

// PipelineRunAssetNodeSummary contains compact matrix totals.
type PipelineRunAssetNodeSummary struct {
	AssetCount            int            `json:"assetCount"`
	NodeCount             int            `json:"nodeCount"`
	Statuses              map[string]int `json:"statuses"`
	TotalEstimatedCostUSD *float64       `json:"totalEstimatedCostUsd,omitempty"`
	CostSource            string         `json:"costSource"`
}

// PipelineRunCostSummary is an estimated cost/audit view for a run.
type PipelineRunCostSummary struct {
	RunID                 string                            `json:"runId"`
	TotalEstimatedCostUSD *float64                          `json:"totalEstimatedCostUsd,omitempty"`
	CostSource            string                            `json:"costSource"`
	NodeSummaries         []PipelineRunNodeCostSummary      `json:"nodeSummaries"`
	AssetNodeSummaries    []PipelineRunAssetNodeCostSummary `json:"assetNodeSummaries"`
	GeneratedAt           time.Time                         `json:"generatedAt"`
}

type PipelineRunNodeCostSummary struct {
	NodeID           string   `json:"nodeId"`
	DisplayName      string   `json:"displayName,omitempty"`
	Status           string   `json:"status,omitempty"`
	PodCount         int      `json:"podCount"`
	EstimatedCostUSD *float64 `json:"estimatedCostUsd,omitempty"`
	CostSource       string   `json:"costSource"`
	DurationSeconds  *int64   `json:"durationSeconds,omitempty"`
}

type PipelineRunAssetNodeCostSummary struct {
	AssetID          string   `json:"assetId"`
	NodeID           string   `json:"nodeId"`
	DisplayName      string   `json:"displayName,omitempty"`
	Status           string   `json:"status,omitempty"`
	EstimatedCostUSD *float64 `json:"estimatedCostUsd,omitempty"`
	CostSource       string   `json:"costSource"`
}

// PipelineRunNotificationCandidate is an idempotent notification work item.
type PipelineRunNotificationCandidate struct {
	ID             string    `json:"id"`
	RunID          string    `json:"runId"`
	EventID        string    `json:"eventId"`
	EventType      string    `json:"eventType"`
	SubjectType    string    `json:"subjectType"`
	SubjectID      string    `json:"subjectId"`
	Status         string    `json:"status,omitempty"`
	Message        string    `json:"message,omitempty"`
	SinkType       string    `json:"sinkType"`
	DeliveryStatus string    `json:"deliveryStatus"`
	IdempotencyKey string    `json:"idempotencyKey"`
	CreatedAt      time.Time `json:"createdAt"`
}

// PipelineRunWatcherState stores coarse watcher progress and diagnostics.
type PipelineRunWatcherState struct {
	ID                  string     `json:"id"`
	LastSyncedAt        *time.Time `json:"lastSyncedAt,omitempty"`
	LastScanStartedAt   *time.Time `json:"lastScanStartedAt,omitempty"`
	LastScanFinishedAt  *time.Time `json:"lastScanFinishedAt,omitempty"`
	LastSuccessAt       *time.Time `json:"lastSuccessAt,omitempty"`
	LastErrorAt         *time.Time `json:"lastErrorAt,omitempty"`
	ActiveScanLimit     int        `json:"activeScanLimit"`
	LastSyncedRunCount  int        `json:"lastSyncedRunCount"`
	ConsecutiveFailures int        `json:"consecutiveFailures"`
	TotalScans          int64      `json:"totalScans"`
	TotalErrors         int64      `json:"totalErrors"`
	ScanLagSeconds      *int64     `json:"scanLagSeconds,omitempty"`
	LastError           string     `json:"lastError,omitempty"`
	Healthy             bool       `json:"healthy"`
	Stale               bool       `json:"stale"`
	UpdatedAt           time.Time  `json:"updatedAt"`
	LedgerHealth        LedgerHealth `json:"ledgerHealth,omitempty"`
}

// LedgerHealth reports how many pipeline runs have ledger events.
type LedgerHealth struct {
	TotalRuns       int        `json:"totalRuns"`
	RunsWithEvents  int        `json:"runsWithEvents"`
	RunsWithout     int        `json:"runsWithout"`
	LastBackfillAt  *time.Time `json:"lastBackfillAt,omitempty"`
}

// PipelineRunListFilter scopes summary list queries for batch-aware UIs.
type PipelineRunListFilter struct {
	BatchJobID   string
	ExcludeBatch bool
	Status       string
	Page         int
	PageSize     int
}
