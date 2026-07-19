package pipeline

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// CreateBatchJob creates an async batch job that will create pipeline runs
// for each asset ID in the background. Returns immediately with a batch ID.
//
// submitWorkers caps how many goroutines submit workflows to Argo in parallel
// from this process. It does NOT limit the actual cluster-side concurrency of
// the submitted runs — that is governed by Argo controller parallelism. Callers
// must not treat it as a true concurrency limit on running workflows.
// dedupPreservingOrder returns ids with duplicates (and blank entries)
// removed, keeping first-occurrence order.
func dedupPreservingOrder(ids []string) []string {
	seen := make(map[string]bool, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

func (uc *Usecase) CreateBatchJob(ctx context.Context, templateID, name string, assetIDs []string, targetID string, templateVersion int, submitWorkers int, owner string) (*models.BackfillJob, error) {
	if uc.backfillRepo == nil {
		return nil, fmt.Errorf("%w: backfill repository is not configured", ErrInvalidArgument)
	}

	t, err := uc.templateRepo.FindByID(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("find template: %w", err)
	}
	if t == nil {
		return nil, ErrTemplateNotFound
	}

	resolvedVersion := t.Version
	if templateVersion > 0 && templateVersion != t.Version {
		resolvedVersion = templateVersion
	} else if templateVersion == 0 && t.ActiveVersion > 0 && t.ActiveVersion != t.Version {
		resolvedVersion = t.ActiveVersion
	}

	if len(assetIDs) == 0 {
		return nil, fmt.Errorf("%w: asset_ids is required", ErrInvalidArgument)
	}
	// Dedup asset ids, order-preserving (G1 load-test finding): duplicate ids
	// create duplicate items, but progress summaries dedup per asset — the
	// job's total_count then exceeds what the summary can ever reach and the
	// job stays "running" forever. One asset = one item.
	assetIDs = dedupPreservingOrder(assetIDs)

	batchID := "batch_" + uuid.New().String()
	now := time.Now().UTC()

	if name == "" {
		name = t.Name + "-" + time.Now().Format("2006-01-02")
	}

	// Submitter mode (default, CYB-3677): the job is born 'running' — the
	// status the durable backfill submitter selects — and this call only
	// persists; submission is owned by the submitter (crash-resumable,
	// idempotent). Legacy mode keeps the old 'pending' + in-memory goroutine.
	status := "running"
	if uc.batchDispatchLegacy {
		status = "pending"
	}
	job := &models.BackfillJob{
		ID:         batchID,
		TemplateID: templateID,
		Name:       name,
		Status:     status,
		PilotPhase: "none",
		TotalCount: len(assetIDs),
		// TemplateVersion pins the batch to the version resolved at creation
		// time (P0: the submitter must never mix versions within one batch
		// when the template's active version moves mid-dispatch).
		TemplateVersion: resolvedVersion,
		CreatedAt:       now,
		UpdatedAt:       now,
		FilterJSON:      map[string]interface{}{},
		CreatedBy:       owner,
	}

	if targetID != "" && targetID != "default" {
		job.FilterJSON["target_id"] = targetID
	}
	// Kept alongside the column for one release so a legacy-flag rollback
	// still sees the pin.
	if resolvedVersion > 0 {
		job.FilterJSON["template_version"] = resolvedVersion
	}

	if err := uc.backfillRepo.SaveJob(ctx, job); err != nil {
		return nil, fmt.Errorf("save batch job: %w", err)
	}

	items := make([]models.BackfillItem, len(assetIDs))
	for i, aid := range assetIDs {
		items[i] = models.BackfillItem{
			ID:        uuid.New().String(),
			JobID:     batchID,
			AssetID:   aid,
			Status:    "pending",
			CreatedAt: now,
		}
	}

	if err := uc.backfillRepo.SaveItems(ctx, items); err != nil {
		_ = uc.backfillRepo.UpdateJobStatus(ctx, batchID, "failed")
		return nil, fmt.Errorf("save batch items: %w", err)
	}

	if uc.batchDispatchLegacy {
		// Legacy (pre CYB-3677, rollback only): one-shot in-memory dispatch
		// goroutine. Not crash-resumable — a restart strands pending items.
		jobCtx, cancel := context.WithCancel(context.Background())
		uc.registerBatchCancel(batchID, cancel)
		go uc.processBatchJob(jobCtx, batchID, templateID, targetID, resolvedVersion, items, owner, submitWorkers, job.Name)
		return job, nil
	}

	// Submitter mode: persistence IS the dispatch. Kick the submitter so the
	// first cycle starts now instead of on the next 15s tick; durability
	// never depends on the kick (boot-eager + ticker re-list this job).
	if uc.batchSubmitKick != nil {
		uc.batchSubmitKick()
	}
	return job, nil
}

// GetBatchJobStatus returns the current status of a batch job.
func (uc *Usecase) GetBatchJobStatus(ctx context.Context, batchID string) (*models.BackfillJob, error) {
	if uc.backfillRepo == nil {
		return nil, fmt.Errorf("%w: backfill repository is not configured", ErrInvalidArgument)
	}
	job, err := uc.backfillRepo.FindJobByID(ctx, batchID)
	if err != nil {
		return nil, err
	}
	return job, nil
}

// defaultSubmitWorkers controls how many goroutines submit workflows to Argo
// in parallel. 20 is safe for Argo Server (~200 QPS × 3ms each = 666/s).
const defaultSubmitWorkers = 20

// deployTimeout caps how long a single CreateRunByTemplateID call may take
// before the worker gives up and marks the item as failed. Without this,
// a hanging Argo API call holds the worker goroutine forever, preventing
// the batch job from ever reaching a terminal state.
const deployTimeout = 60 * time.Second

// processBatchJob submits all items to Argo using a parallel worker pool and
// lets the controller manage concurrency via parallelism config. Errors are
// per-item — one failure does not cancel the batch.
//
// CYB-3491: this legacy (pipeline-level) batch path now feeds its worker pool
// from the materialized item list instead of ClaimNextItem (deleted with the
// backfill execution queue). Behaviour parity: this path never had restart
// resume (its jobs run under status "processing", which the old resume never
// selected); the claim call only de-duplicated within this one pool, which a
// channel feed does just as well. Items are re-checked against the ledger
// right before submission so an item cancelled/failed elsewhere is skipped.
func (uc *Usecase) processBatchJob(ctx context.Context, jobID, templateID, targetID string, templateVersion int, items []models.BackfillItem, owner string, submitWorkers int, batchName string) {
	if submitWorkers <= 0 {
		submitWorkers = defaultSubmitWorkers
	}
	defer uc.unregisterBatchCancel(jobID)
	slog.Info("batch job started", "batchID", jobID, "totalItems", len(items), "workers", submitWorkers)

	if err := uc.backfillRepo.UpdateJobStatus(ctx, jobID, "processing"); err != nil {
		slog.Warn("batch job: failed to update status to processing", "batchID", jobID, "err", err)
		return
	}

	feed := make(chan models.BackfillItem)
	go func() {
		defer close(feed)
		for i := range items {
			select {
			case <-ctx.Done():
				return
			case feed <- items[i]:
			}
		}
	}()

	var wg sync.WaitGroup
	for w := 0; w < submitWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range feed {
				if ctx.Err() != nil {
					return
				}
				// Re-check the ledger: only still-pending items are submitted.
				if fresh, err := uc.backfillRepo.FindItemByID(ctx, item.ID); err != nil || fresh == nil || fresh.Status != "pending" {
					continue
				}
				opts := []DeployOptions{{
					TargetID:           targetID,
					TemplateVersion:    templateVersion,
					BatchJobID:         jobID,
					AllowUnknownAssets: true,
					Owner:              owner,
				}}
				itemCtx, itemCancel := context.WithTimeout(ctx, deployTimeout)
				run, err := uc.CreateRunByTemplateID(itemCtx, templateID, batchName, []string{item.AssetID}, opts...)
				itemCancel()
				if err != nil {
					if errors.Is(err, context.Canceled) {
						continue
					}
					errMsg := err.Error()
					slog.Warn("batch job: asset failed", "assetID", item.AssetID, "err", errMsg)
					_ = uc.backfillRepo.UpdateItemStatus(ctx, item.ID, "failed", "", errMsg)
					_ = uc.backfillRepo.IncrementFailed(ctx, jobID)
				} else {
					// Submission only queues the workflow in Argo; it has NOT
					// finished. CYB-3491: the in-flight item state is "submitted".
					_ = uc.backfillRepo.UpdateItemPipelineRun(ctx, item.ID, run.ID, run.WorkflowName, "submitted")
				}
			}
		}()
	}
	wg.Wait()

	finalCtx, cancelFinal := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelFinal()

	summary, err := uc.backfillRepo.SummarizeItemStatuses(finalCtx, jobID)
	if err != nil {
		slog.Warn("batch job: summarize failed", "batchID", jobID, "err", err)
		_ = uc.backfillRepo.UpdateJobStatus(finalCtx, jobID, "failed")
		return
	}

	// Submission is done, but the workflows are still executing in Argo. Derive
	// status from real item state instead of assuming "completed": items just
	// submitted are "running" (Pending in Argo). The backfill status sync only
	// promotes the job to "completed" once Argo workflows actually finish.
	status := "running"
	switch {
	case ctx.Err() != nil:
		status = "cancelled"
	case summary.Pending == 0 && summary.Running == 0:
		// Every item already reached a terminal state.
		if summary.Completed == 0 && summary.Failed > 0 {
			status = "failed"
		} else {
			status = "completed"
		}
	}
	total := summary.Completed + summary.Failed + summary.Pending + summary.Running
	if err := uc.backfillRepo.UpdateJobProgress(finalCtx, jobID, summary.Completed, summary.Failed, status); err != nil {
		slog.Warn("batch job: update final progress failed", "batchID", jobID, "err", err)
	}

	slog.Info("batch job completed",
		"batchID", jobID,
		"total", total,
		"completed", summary.Completed,
		"failed", summary.Failed,
	)
}
