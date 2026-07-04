package repository

import (
	"context"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// BackfillRepository defines persistence operations for backfill jobs and items.
type BackfillRepository interface {
	// Job operations.
	SaveJob(ctx context.Context, j *models.BackfillJob) error
	FindAllJobs(ctx context.Context) ([]models.BackfillJob, error)
	FindJobByID(ctx context.Context, id string) (*models.BackfillJob, error)
	UpdateJobStatus(ctx context.Context, id, status string) error
	UpdateJobPilotPhase(ctx context.Context, id, status, pilotPhase string) error
	IncrementCompleted(ctx context.Context, id string) error
	IncrementFailed(ctx context.Context, id string) error

	// Item operations.
	SaveItem(ctx context.Context, item *models.BackfillItem) error
	SaveItems(ctx context.Context, items []models.BackfillItem) error
	FindItemsByJobID(ctx context.Context, jobID string) ([]models.BackfillItem, error)
	FindItemByID(ctx context.Context, id string) (*models.BackfillItem, error)
	FindItemByPipelineRunID(ctx context.Context, pipelineRunID string) (*models.BackfillItem, error)
	UpdateItemStatus(ctx context.Context, id, status, workflowName, errorMsg string) error
	UpdateItemPipelineRun(ctx context.Context, id, pipelineRunID, workflowName, status string) error
	UpdateJobProgress(ctx context.Context, id string, completed, failed int, status string) error
	CountItemsByStatus(ctx context.Context, jobID, status string) (int, error)
	SummarizeItemStatuses(ctx context.Context, jobID string) (BackfillItemStatusSummary, error)
	FindItemsByJobIDWithStatuses(ctx context.Context, jobID string, statuses []string) ([]models.BackfillItem, error)
	FindItemsMissingPipelineRun(ctx context.Context, jobID string) ([]models.BackfillItem, error)
	FindItemsByScope(ctx context.Context, filter BackfillRerunItemFilter) ([]models.BackfillItem, error)
	PrepareItemsForRerun(ctx context.Context, itemIDs []string) error
	AggregateNodeStatusByBatchJobID(ctx context.Context, jobID string) ([]BatchNodeStatusAggregate, error)
	ListNodeFailures(ctx context.Context, filter BatchNodeFailureFilter) (*models.BatchNodeFailureListResult, error)
	CountPipelineRunsByBatchJobID(ctx context.Context, jobID string) (int, error)
	CountRunsWithNodeRowsByBatchJobID(ctx context.Context, jobID string) (int, error)
	FindItemsByAssetID(ctx context.Context, assetID string) ([]models.BackfillItem, error)
	FindItemByJobAndAssetID(ctx context.Context, jobID, assetID string) (*models.BackfillItem, error)

	// ── Phase 4 dispatcher surface (outbox) ──────────────────────────────────
	// See openspec/changes/CYB-RUN-DIAGNOSIS-REFACTOR/PHASE4-DESIGN.md.

	// ClaimNextDispatch atomically picks the oldest ready backfill_item
	// using FOR UPDATE SKIP LOCKED and sets dispatch_state='claimed' with
	// a lease held until now()+leaseSec. Returns nil if no item is ready
	// (or all ready items have exhausted MaxAttempts). The reserved row
	// stays under SKIP LOCKED for the duration of the surrounding
	// transaction. Returns errors.ErrNoRows (mapped to a return of (nil,
	// nil)) when the candidate set is empty.
	ClaimNextDispatch(ctx context.Context, leaseSec, maxAttempts int) (*models.BackfillItem, error)

	// MarkDispatchSubmitting is called by a worker right before invoking
	// the pipeline Deploy, to keep the lease alive while a slow submit is
	// in flight and to set dispatch_state='submitting'. Idempotent.
	MarkDispatchSubmitting(ctx context.Context, itemID string, leaseSec int) error

	// MarkDispatched is the success-terminal state transition. Stamps
	// the determined argo workflow name + UID into backfill_items
	// (alongside the legacy columns used by other endpoints) and clears
	// the lease. After this call, the row is in the absorbing
	// 'submitted' state.
	MarkDispatched(ctx context.Context, itemID, argoWorkflowName, argoUID string) error

	// MarkDispatchFailedRetryable transitions to 'failed' and sets the
	// lease to NOW()+leaseSec — the lease IS the backoff window
	// (Decision B). Increments attempts and records dispatch_last_error.
	// The row stays out of ClaimNextDispatch until the lease expires.
	MarkDispatchFailedRetryable(ctx context.Context, itemID string, attempts int, lastErr string, leaseSec int) error

	// MarkDispatchDead is the absorbing failure terminal state, reached
	// only when attempts exceed MaxAttempts. Same as
	// MarkDispatchFailedRetryable but with state='dead'.
	MarkDispatchDead(ctx context.Context, itemID string, attempts int, lastErr string) error

	// UpdateItemDispatchFields writes back the realtime runtime fields
	// (template_id, template_version, target_id, dispatch_lease_expires_at)
	// from the dispatcher into backfill_items. Bumps attempts to mirror
	// the dispatcher's claim for downstream observers.
	UpdateItemDispatchFields(ctx context.Context, itemID string, templateID string, templateVersion int, targetID string, leaseSec int) error

	// ResetStaleDispatchedItems is the new reaper. It transitions
	// 'claimed' / 'submitting' rows whose lease has expired back to
	// 'pending' (provided they still have headroom under MaxAttempts).
	// ABsorbing 'submitted' / 'dead' states are explicitly excluded.
	// Returns the number of rows reclaimed.
	ResetStaleDispatchedItems(ctx context.Context, leaseSec, maxAttempts int) (int, error)

	// EnqueueForDispatcher transitions the given item IDs to
	// dispatch_state='pending' and stashes a workflow_name_planned for
	// each (computed by the runner parameter so we don't bake deterministic
	// naming into the SQL). Used by the mode='outbox' code path (Commit
	// B) at CreateBackfill / ResumeJob / Rerun / ContinueFull.
	// The commit_generation stays as-is; rerun bumps it separately.
	EnqueueForDispatcher(ctx context.Context, itemIDs []string, resolveWfName func(itemID string) string) error
}

// BackfillItemStatusSummary aggregates item counts by coarse status bucket.
type BackfillItemStatusSummary struct {
	Completed int
	Failed    int
	Pending   int
	Running   int
}

type BackfillRerunItemFilter struct {
	JobID          string
	Statuses       []string
	ItemIDs        []string
	AssetIDs       []string
	PipelineNodeID string
	NodeStatuses   []string
}

type BatchNodeStatusAggregate struct {
	PipelineNodeID string
	DisplayName    string
	Status         string
	Count          int
}

type BatchNodeFailureFilter struct {
	JobID          string
	PipelineNodeID string
	Statuses       []string
	Page           int
	PageSize       int
	Query          string
}
