package pipeline

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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

	batchID := "batch_" + uuid.New().String()
	now := time.Now().UTC()

	if name == "" {
		name = t.Name + "-" + time.Now().Format("2006-01-02")
	}

	job := &models.BackfillJob{
		ID:          batchID,
		TemplateID:  templateID,
		Name:        name,
		Status:      "pending",
		PilotPhase:  "none",
		TotalCount:  len(assetIDs),
		CreatedAt:   now,
		UpdatedAt:   now,
		FilterJSON:  map[string]interface{}{},
	}

	if targetID != "" && targetID != "default" {
		job.FilterJSON["target_id"] = targetID
	}
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

	// Process in background — never use request context for goroutines.
	// A dedicated cancellable context lets StopBatchRuns halt submission.
	jobCtx, cancel := context.WithCancel(context.Background())
	uc.registerBatchCancel(batchID, cancel)
	go uc.processBatchJob(jobCtx, batchID, templateID, targetID, resolvedVersion, items, owner, submitWorkers)

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
func (uc *Usecase) processBatchJob(ctx context.Context, jobID, templateID, targetID string, templateVersion int, items []models.BackfillItem, owner string, submitWorkers int) {
	if submitWorkers <= 0 {
		submitWorkers = defaultSubmitWorkers
	}
	defer uc.unregisterBatchCancel(jobID)
	slog.Info("batch job started", "batchID", jobID, "totalItems", len(items), "workers", submitWorkers)

	if err := uc.backfillRepo.UpdateJobStatus(ctx, jobID, "processing"); err != nil {
		slog.Warn("batch job: failed to update status to processing", "batchID", jobID, "err", err)
		return
	}

	type result struct {
		item  models.BackfillItem
		runID string
		err   error
	}

	work := make(chan models.BackfillItem, len(items))
	results := make(chan result, len(items))

	// Feed all items into the work channel
	for _, item := range items {
		work <- item
	}
	close(work)

	// Start worker pool
	var wg sync.WaitGroup
	for w := 0; w < submitWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case item, ok := <-work:
					if !ok {
						return
					}
					// select picks a ready case at random, so re-check
					// cancellation before submitting to make stop prompt.
					if ctx.Err() != nil {
						return
					}
					if err := uc.backfillRepo.UpdateItemStatus(ctx, item.ID, "processing", "", ""); err != nil {
						slog.Warn("batch job: mark item processing failed", "batchID", jobID, "itemID", item.ID, "err", err)
					}
					opts := []DeployOptions{{
						TargetID:           targetID,
						TemplateVersion:    templateVersion,
						BatchJobID:         jobID,
						AllowUnknownAssets: true,
						Owner:              owner,
					}}
					itemCtx, itemCancel := context.WithTimeout(ctx, deployTimeout)
					run, err := uc.CreateRunByTemplateID(itemCtx, templateID, "", []string{item.AssetID}, opts...)
					itemCancel()
					if err != nil {
						results <- result{item: item, err: err}
					} else {
						results <- result{item: item, runID: run.ID}
					}
				}
			}
		}()
	}

	// Close results when all workers are done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	for r := range results {
		if r.err != nil {
			// A cancelled batch surfaces context errors on in-flight submits;
			// these are not real asset failures, so leave the item untouched.
			if errors.Is(r.err, context.Canceled) {
				continue
			}
			// DeadlineExceeded from the per-item timeout IS a real failure — mark
			// the item as failed so the batch can proceed to a terminal state.
			errMsg := r.err.Error()
			slog.Warn("batch job: asset failed", "assetID", r.item.AssetID, "err", errMsg)
			if err := uc.backfillRepo.UpdateItemStatus(ctx, r.item.ID, "failed", "", errMsg); err != nil {
				slog.Warn("batch job: mark item failed failed", "batchID", jobID, "itemID", r.item.ID, "err", err)
			}
			if err := uc.backfillRepo.IncrementFailed(ctx, jobID); err != nil {
				slog.Warn("batch job: increment failed counter failed", "batchID", jobID, "err", err)
			}
		} else {
			if err := uc.backfillRepo.UpdateItemPipelineRun(ctx, r.item.ID, r.runID, "", "completed"); err != nil {
				slog.Warn("batch job: bind item run failed", "batchID", jobID, "itemID", r.item.ID, "err", err)
			}
			if err := uc.backfillRepo.IncrementCompleted(ctx, jobID); err != nil {
				slog.Warn("batch job: increment completed counter failed", "batchID", jobID, "err", err)
			}
		}
	}

	// Terminal bookkeeping must not run on the (possibly cancelled) job ctx:
	// when StopBatchRuns cancels the batch, reusing it would make the summary
	// query and the final status write fail with context.Canceled, leaving the
	// job stuck in "processing". jobCtx carries no request deadline, so a fresh
	// background ctx is the correct scope for recording the final state.
	finalCtx, cancelFinal := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelFinal()

	summary, err := uc.backfillRepo.SummarizeItemStatuses(finalCtx, jobID)
	if err != nil {
		slog.Warn("batch job: summarize failed", "batchID", jobID, "err", err)
		_ = uc.backfillRepo.UpdateJobStatus(finalCtx, jobID, "failed")
		return
	}

	status := "completed"
	switch {
	case ctx.Err() != nil:
		// StopBatchRuns cancelled this batch mid-flight; preserve the
		// cancelled status instead of overwriting it with completed/failed.
		status = "cancelled"
	case summary.Failed > 0 && summary.Completed == 0:
		status = "failed"
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
