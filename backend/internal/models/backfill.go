package models

import "time"

// BackfillJob represents a batch backfill job that processes multiple assets
// through a selected pipeline template.
type BackfillJob struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	TemplateID     string                 `json:"templateId"`
	FilterJSON     map[string]interface{} `json:"filterJson,omitempty"`
	TotalCount     int                    `json:"totalCount"`
	CompletedCount int                    `json:"completedCount"`
	FailedCount    int                    `json:"failedCount"`
	Status         string                 `json:"status"`      // running | paused | completed | failed
	CreatedAt      time.Time              `json:"createdAt"`
	UpdatedAt      time.Time              `json:"updatedAt"`
}

// BackfillItem represents a single asset being processed in a backfill job.
type BackfillItem struct {
	ID           string     `json:"id"`
	JobID        string     `json:"jobId"`
	AssetID      string     `json:"assetId"`
	Status       string     `json:"status"` // pending | running | completed | failed | cancelled
	WorkflowName *string    `json:"workflowName,omitempty"`
	ErrorMessage *string    `json:"errorMessage,omitempty"`
	StartedAt    *time.Time `json:"startedAt,omitempty"`
	FinishedAt   *time.Time `json:"finishedAt,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
}
