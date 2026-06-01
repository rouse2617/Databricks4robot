package models

import "time"

// PipelineTemplate represents a reusable pipeline template stored in the
// pipeline_templates table. The Pipeline field holds the full pipeline
// definition as arbitrary JSON.
type PipelineTemplate struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Version   int                    `json:"version"`
	Pipeline  map[string]interface{} `json:"pipeline"`
	NodeCount int                    `json:"nodeCount"`
	CreatedAt time.Time              `json:"createdAt"`
	UpdatedAt time.Time              `json:"updatedAt"`
}

// PipelineDeployment represents a single deployment of a pipeline template
// to a workflow run. Each deployment records the target workflow, current
// status, and optionally the rendered manifest and pipeline JSON snapshot.
type PipelineDeployment struct {
	ID              string                 `json:"id"`
	TemplateID      *string                `json:"templateId,omitempty"`
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
// CYB-1532 starts with a compatibility default target backed by current Argo
// configuration; persistence can be added once migrations are approved.
type ExecutionTarget struct {
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	Cluster              string `json:"cluster"`
	Namespace            string `json:"namespace"`
	ArgoServerConfigured bool   `json:"argoServerConfigured"`
	Status               string `json:"status"`
	IsDefault            bool   `json:"isDefault"`
	Description          string `json:"description,omitempty"`
}
