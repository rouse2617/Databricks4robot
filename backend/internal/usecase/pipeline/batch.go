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
		ID:         batchID,
		TemplateID: templateID,
		Name:       name,
		Status:     "pending",
		PilotPhase: "none",
		TotalCount: len(assetIDs),
		CreatedAt:  now,
		UpdatedAt:  now,
		FilterJSON: map[string]interface{}{},
		CreatedBy:  owner,
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
	go uc.processBatchJob(jobCtx, batchID, templateID, targetID, resolvedVersion, items, owner, submitWorkers, job.Name)

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
// Items are claimed from the database using ClaimNextItem so processing
// survives service restarts.
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

	var wg sync.WaitGroup
	for w := 0; w < submitWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}
				if ctx.Err() != nil {
					return
				}
				item, err := uc.backfillRepo.ClaimNextItem(ctx, jobID)
				if err != nil {
					slog.Warn("batch job: claim next item failed", "batchID", jobID, "err", err)
					time.Sleep(time.Second)
					continue
				}
				if item == nil {
					return
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
					_ = uc.backfillRepo.UpdateItemPipelineRun(ctx, item.ID, run.ID, "", "completed")
					_ = uc.backfillRepo.IncrementCompleted(ctx, jobID)
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

	status := "completed"
	switch {
	case ctx.Err() != nil:
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
