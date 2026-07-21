// Package schedtask hosts the scheduled-task (定时任务) feature: rules that
// periodically pull asset-ids from a configured source and create a batch on a
// pipeline template. Replaces the external grace-sync Cloud Run Job (CYB-3744).
package schedtask

import (
	"encoding/json"
	"time"
)

// TriggerMode determines how a rule chooses which asset-ids each run picks up.
// Recurring modes (incremental/rolling) fire on the interval; range/ids are
// one-shot (usually via 立即运行). See design.md for full semantics.
type TriggerMode string

const (
	TriggerIncremental TriggerMode = "incremental"
	TriggerRolling     TriggerMode = "rolling"
	TriggerRange       TriggerMode = "range"
	TriggerIDs         TriggerMode = "ids"
)

// RunStatus values recorded in scheduled_tasks.last_run_status. Empty until
// the rule has run at least once.
const (
	RunStatusSucceeded = "succeeded"
	RunStatusFailed    = "failed"
	RunStatusEmpty     = "empty"   // fetched 0 ids — not a failure, but observable
	RunStatusSkipped   = "skipped" // e.g. paused or claim lost mid-cycle
)

// Rule is the persisted definition of a scheduled task. The Rule struct is
// the transport shape (used by repo and API); the migration owns the DB DDL
// (migrations/20260721150926_scheduled_tasks.sql).
type Rule struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	Enabled         bool            `json:"enabled"`
	TemplateID      string          `json:"templateId"`
	TemplateVersion *int            `json:"templateVersion,omitempty"`
	TargetID        string          `json:"targetId"`
	Scheduling      json.RawMessage `json:"scheduling,omitempty"` // resource_defaults.scheduling shape

	SourceType   string          `json:"sourceType"`
	SourceConfig json.RawMessage `json:"sourceConfig"`
	TriggerMode  TriggerMode     `json:"triggerMode"`
	// TriggerConfig is mode-specific (see model_trigger.go decode helpers).
	TriggerConfig json.RawMessage `json:"triggerConfig"`

	Cursor            string     `json:"cursor,omitempty"`
	LastRunAt         *time.Time `json:"lastRunAt,omitempty"`
	LastRunStatus     string     `json:"lastRunStatus,omitempty"`
	LastBatchID       string     `json:"lastBatchId,omitempty"`
	LastError         string     `json:"lastError,omitempty"`
	LastSuccessAt     *time.Time `json:"lastSuccessAt,omitempty"`
	RunNowRequestedAt *time.Time `json:"runNowRequestedAt,omitempty"`
	CreatedBy         string     `json:"createdBy,omitempty"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
}

// TriggerConfig is the decoded per-mode config. Only the fields for the
// selected TriggerMode are meaningful; others are ignored. Kept flat so the
// UI form is 1:1.
type TriggerConfig struct {
	IntervalSeconds        int        `json:"intervalSeconds,omitempty"`        // incremental + rolling
	LookbackSeconds        int        `json:"lookbackSeconds,omitempty"`        // rolling; also incremental first-run fallback
	InitialLookbackSeconds int        `json:"initialLookbackSeconds,omitempty"` // incremental only (first run when cursor is empty)
	From                   *time.Time `json:"from,omitempty"`                   // range
	To                     *time.Time `json:"to,omitempty"`                     // range
	IDs                    []string   `json:"ids,omitempty"`                    // ids
	NotifyOnEmpty          bool       `json:"notifyOnEmpty,omitempty"`          // opt-in per rule
	StuckThresholdSeconds  int        `json:"stuckThresholdSeconds,omitempty"`  // 0 = default (see scheduler)
}
