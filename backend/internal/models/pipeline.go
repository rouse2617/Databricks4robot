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
	ID           string                 `json:"id"`
	TemplateID   string                 `json:"templateId"`
	PipelineName string                 `json:"pipelineName"`
	WorkflowName string                 `json:"workflowName"`
	Status       string                 `json:"status"`
	NodeCount    int                    `json:"nodeCount"`
	Manifest     *string                `json:"manifest,omitempty"`
	PipelineJSON map[string]interface{} `json:"pipelineJSON,omitempty"`
	CreatedAt    time.Time              `json:"createdAt"`
	UpdatedAt    time.Time              `json:"updatedAt"`
	FinishedAt   *time.Time             `json:"finishedAt,omitempty"`
}
