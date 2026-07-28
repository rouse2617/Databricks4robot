package subtask

import (
	"time"
)

const (
	RunStatusSucceeded = "succeeded"
	RunStatusFailed    = "failed"
	RunStatusEmpty     = "empty"
)

// PipelineBinding is one (template, target) pair a subscription task fans a
// message out to. One task pulls from a single subscription and dispatches a
// batch per binding for every message it receives.
type PipelineBinding struct {
	TemplateID      string `json:"templateId"`
	TemplateVersion *int   `json:"templateVersion,omitempty"`
	TargetID        string `json:"targetId"`
}

type Task struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`

	ProjectID          string `json:"projectId"`
	SubscriptionID     string `json:"subscriptionId"`
	PullIntervalSec    int    `json:"pullIntervalSeconds"`
	MaxMessagesPerPull int    `json:"maxMessagesPerPull"`

	// One batch is dispatched per binding for every pulled message set.
	PipelineBindings []PipelineBinding `json:"pipelineBindings"`

	LastRunAt     *time.Time `json:"lastRunAt,omitempty"`
	LastRunStatus string     `json:"lastRunStatus,omitempty"`
	LastBatchIDs  []string   `json:"lastBatchIds,omitempty"`
	LastError     string     `json:"lastError,omitempty"`
	LastSuccessAt *time.Time `json:"lastSuccessAt,omitempty"`
	CreatedBy     string     `json:"createdBy,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}
