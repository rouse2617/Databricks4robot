package models

import "time"

// PipelineTemplate represents a reusable pipeline template stored in the
// pipeline_templates table. The Pipeline field holds the full pipeline
// definition as arbitrary JSON.
type PipelineTemplate struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Version      int                    `json:"version"`
	VersionCount int                    `json:"versionCount,omitempty"`
	Pipeline     map[string]interface{} `json:"pipeline"`
	NodeCount    int                    `json:"nodeCount"`
	CreatedAt    time.Time              `json:"createdAt"`
	UpdatedAt    time.Time              `json:"updatedAt"`
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
