package models

import "time"

// BackfillJob represents a batch backfill job that processes multiple assets
// through a selected pipeline template.
type BackfillJob struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	TemplateID      string                 `json:"templateId"`
	TemplateVersion int                    `json:"templateVersion,omitempty"`
	FilterJSON      map[string]interface{} `json:"filterJson,omitempty"`
	TotalCount      int                    `json:"totalCount"`
	CompletedCount  int                    `json:"completedCount"`
	FailedCount     int                    `json:"failedCount"`
	PilotCount      int                    `json:"pilotCount,omitempty"`
	PilotPhase      string                 `json:"pilotPhase,omitempty"` // none | running | review | done
	Status          string                 `json:"status"`               // running | paused | completed | failed
	CreatedBy       string                 `json:"createdBy,omitempty"`
	CreatedAt       time.Time              `json:"createdAt"`
	UpdatedAt       time.Time              `json:"updatedAt"`
	FinishedAt      *time.Time             `json:"finishedAt,omitempty"`
}

// BackfillItem represents a single asset being processed in a backfill job.
type BackfillItem struct {
	ID            string     `json:"id"`
	JobID         string     `json:"jobId"`
	AssetID       string     `json:"assetId"`
	Status        string     `json:"status"` // pending | running | completed | failed | cancelled
	PipelineRunID *string    `json:"pipelineRunId,omitempty"`
	WorkflowName  *string    `json:"workflowName,omitempty"`
	ErrorMessage  *string    `json:"errorMessage,omitempty"`
	Attempts      int        `json:"attempts,omitempty"`
	StartedAt     *time.Time `json:"startedAt,omitempty"`
	FinishedAt    *time.Time `json:"finishedAt,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`

	// Phase 4: persistent dispatcher (outbox) columns. See
	// openspec/changes/CYB-RUN-DIAGNOSIS-REFACTOR/PHASE4-DESIGN.md.
	//
	// DispatchState is the outbox state machine:
	//   pending      ready to be claimed
	//   claimed      ticker has SKIP-LOCKED-locked this row; lease active
	//   submitting   worker has refreshed lease; submit in flight
	//   failed       retryable; lease holds the backoff window
	//   submitted    ABSORBING — successful, never re-claim
	//   dead         ABSORBING — MaxAttempts reached, never re-claim
	//   legacy_skip  legacy sentinel: pre-migration row retained
	//               for historical traceability; the dispatcher no
	//               longer treats this as a meaningful state.
	DispatchState        string     `json:"dispatchState,omitempty"`          // above enum
	DispatchGeneration   int64      `json:"dispatchGeneration,omitempty"`     // bumped on rerun
	WorkflowNamePlanned  string     `json:"workflowNamePlanned,omitempty"`    // deterministic wfname (Phase 1)
	DispatchLeaseExpires *time.Time `json:"dispatchLeaseExpiresAt,omitempty"` // active lease wall-clock
	DispatchLastError    string     `json:"dispatchLastError,omitempty"`      // for observability
}

type BatchNodeSummary struct {
	BatchJobID      string                 `json:"batchJobId"`
	TemplateID      string                 `json:"templateId"`
	TemplateVersion int                    `json:"templateVersion,omitempty"`
	Subtasks        BatchNodeSubtaskCounts `json:"subtasks"`
	Nodes           []BatchNodeSummaryNode `json:"nodes"`
	DataCoverage    BatchNodeDataCoverage  `json:"dataCoverage"`
	GeneratedAt     time.Time              `json:"generatedAt"`
}

type BatchNodeSubtaskCounts struct {
	Total     int  `json:"total"`
	Completed int  `json:"completed"`
	Failed    int  `json:"failed"`
	Running   int  `json:"running"`
	Pending   int  `json:"pending"`
	Paused    bool `json:"paused"`
}

type BatchNodeSummaryNode struct {
	PipelineNodeID    string                   `json:"pipelineNodeId"`
	DisplayName       string                   `json:"displayName"`
	DagOrder          int                      `json:"dagOrder"`
	Counts            map[string]int           `json:"counts"`
	Attempted         int                      `json:"attempted"`
	FailureRate       float64                  `json:"failureRate"`
	TopFailureReasons []BatchNodeFailureReason `json:"topFailureReasons,omitempty"`
}

type BatchNodeFailureReason struct {
	Message string `json:"message"`
	Count   int    `json:"count"`
}

type BatchNodeDataCoverage struct {
	RunsWithNodeRows int  `json:"runsWithNodeRows"`
	RunsTotal        int  `json:"runsTotal"`
	Complete         bool `json:"complete"`
}

type BatchNodeFailureItem struct {
	BackfillItemID string     `json:"backfillItemId"`
	AssetID        string     `json:"assetId"`
	RunID          string     `json:"runId"`
	WorkflowName   string     `json:"workflowName"`
	PipelineNodeID string     `json:"pipelineNodeId"`
	DisplayName    string     `json:"displayName"`
	Status         string     `json:"status"`
	Message        string     `json:"message,omitempty"`
	StartedAt      *time.Time `json:"startedAt,omitempty"`
	FinishedAt     *time.Time `json:"finishedAt,omitempty"`
}

type BatchNodeFailureListResult struct {
	Items    []BatchNodeFailureItem `json:"items"`
	Total    int                    `json:"total"`
	Page     int                    `json:"page"`
	PageSize int                    `json:"pageSize"`
}

type BackfillItemAttempt struct {
	RunID           string                   `json:"runId"`
	AttemptNo       int                      `json:"attemptNo"`
	Status          string                   `json:"status"`
	TemplateVersion int                      `json:"templateVersion,omitempty"`
	WorkflowName    string                   `json:"workflowName,omitempty"`
	Message         string                   `json:"message,omitempty"`
	NodeProgress    *PipelineRunNodeProgress `json:"nodeProgress,omitempty"`
	IsCurrent       bool                     `json:"isCurrent"`
	StartedAt       *time.Time               `json:"startedAt,omitempty"`
	FinishedAt      *time.Time               `json:"finishedAt,omitempty"`
	CreatedAt       time.Time                `json:"createdAt"`
}

type BackfillItemAttemptsResult struct {
	ItemID       string                `json:"itemId"`
	AssetID      string                `json:"assetId"`
	CurrentRunID string                `json:"currentRunId,omitempty"`
	Attempts     []BackfillItemAttempt `json:"attempts"`
}

type ValidateBackfillAssetsResult struct {
	Registered []string `json:"registered"`
	Unknown    []string `json:"unknown"`
}
