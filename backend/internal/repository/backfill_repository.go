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
	// IncrementItemSubmitAttempts bumps the durable transient-retry counter
	// (CYB-3678 DLQ) and returns the new value.
	IncrementItemSubmitAttempts(ctx context.Context, itemID string) (int, error)
	// ResetFailedItems re-queues a job's DLQ (failed items → pending, error
	// cleared, attempt counter reset). Returns how many items were revived.
	ResetFailedItems(ctx context.Context, jobID string) (int64, error)

	// ClaimJobNotification atomically claims the completion-notification slot
	// for a job (notification_sent_at IS NULL -> now()). Returns true only for
	// the caller that performs the claim; concurrent/repeat callers get false
	// with no error, so at most one caller ever proceeds to send.
	ClaimJobNotification(ctx context.Context, id string) (bool, error)

	// Item operations.
	SaveItem(ctx context.Context, item *models.BackfillItem) error
	SaveItems(ctx context.Context, items []models.BackfillItem) error
	FindItemsByJobID(ctx context.Context, jobID string) ([]models.BackfillItem, error)
	FindItemByID(ctx context.Context, id string) (*models.BackfillItem, error)
	FindItemByPipelineRunID(ctx context.Context, pipelineRunID string) (*models.BackfillItem, error)
	UpdateItemStatus(ctx context.Context, id, status, workflowName, errorMsg string) error
	UpdateItemPipelineRun(ctx context.Context, id, pipelineRunID, workflowName, status string) error
	// AdvanceItemAndCountAtomic transitions a backfill item to a terminal
	// status (completed / failed / cancelled) AND increments the corresponding
	// job counter in one CTE. Idempotent by construction: the item is only
	// advanced when currently non-terminal, so an at-least-once webhook
	// redelivery finds the row already terminal, updates zero rows, and does
	// NOT touch the counter — unlike the plain IncrementCompleted /
	// IncrementFailed pair, which would double-count on retry.
	//
	// Replaces the O(N²) full-batch aggregate that ran on every child terminal
	// (CYB-3078 fast path); reconciler still full-scans every 60s as the
	// authoritative baseline, so drift and lost events converge.
	AdvanceItemAndCountAtomic(ctx context.Context, itemID, newStatus, workflowName, errorMsg string) error
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




	// FindActiveJobs returns non-terminal, non-paused batch jobs up to limit,
	// oldest first, regardless of whether items are still pending. Used by the
	// reconcile backstop to finalize + notify jobs whose children have finished.
	// CountStaleBackfillItems returns how many backfill_items are still
	// pending/submitted while their linked pipeline_run is already terminal
	// (watcher→reconciler gap). Used for monitoring; the reconciler itself
	// resolves the gap on the next cycle.
	CountStaleBackfillItems(ctx context.Context) (int, error)
	FindActiveJobs(ctx context.Context, limit int) ([]models.BackfillJob, error)
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

// SubmitQueue is the P2 submitter's persistence surface (CYB-3491): find jobs
// with submittable work, list candidate items lock-free, and lock exactly one
// pending item inside the caller's transaction. It is deliberately a separate
// interface from BackfillRepository so existing test doubles do not need to
// grow these methods; production wiring implements both on the same repo.
type SubmitQueue interface {
	// WithTx runs fn inside one transaction; repo calls made with fn's ctx
	// ride that transaction (dbFromCtx). The submitter locks and commits one
	// item per transaction.
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
	// FindSubmittableJobs returns running / pilot_running jobs that still
	// have pending items, oldest first.
	FindSubmittableJobs(ctx context.Context, limit int) ([]models.BackfillJob, error)
	// ListSubmittableItemIDs returns candidate pending item ids for one job,
	// oldest first, WITHOUT taking locks — candidates are re-checked and
	// locked one by one via LockPendingItem.
	ListSubmittableItemIDs(ctx context.Context, jobID string, limit int) ([]string, error)
	// LockPendingItem locks the item row FOR UPDATE SKIP LOCKED iff it is
	// still pending. Returns nil when the item is gone, no longer pending, or
	// currently locked by another submitter (same instance or another one) —
	// the caller simply skips it. This row lock is what guarantees no two
	// workers attempt the same item concurrently.
	LockPendingItem(ctx context.Context, itemID string) (*models.BackfillItem, error)
}
