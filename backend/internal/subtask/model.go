package subtask

import (
	"time"
)

const (
	RunStatusSucceeded = "succeeded"
	RunStatusFailed    = "failed"
	RunStatusEmpty     = "empty"
)

type Task struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Enabled         bool   `json:"enabled"`
	TemplateID      string `json:"templateId"`
	TemplateVersion *int   `json:"templateVersion,omitempty"`
	TargetID        string `json:"targetId"`

	ProjectID          string `json:"projectId"`
	SubscriptionID     string `json:"subscriptionId"`
	PullIntervalSec    int    `json:"pullIntervalSeconds"`
	MaxMessagesPerPull int    `json:"maxMessagesPerPull"`

	LastRunAt     *time.Time `json:"lastRunAt,omitempty"`
	LastRunStatus string     `json:"lastRunStatus,omitempty"`
	LastBatchID   string     `json:"lastBatchId,omitempty"`
	LastError     string     `json:"lastError,omitempty"`
	LastSuccessAt *time.Time `json:"lastSuccessAt,omitempty"`
	CreatedBy     string     `json:"createdBy,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}
