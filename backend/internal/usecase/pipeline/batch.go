package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// CreateBatchJob creates an async batch job that will create pipeline runs
// for each asset ID in the background. Returns immediately with a batch ID.
func (uc *Usecase) CreateBatchJob(ctx context.Context, templateID, name string, assetIDs []string, targetID string, templateVersion int) (*models.BackfillJob, error) {
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
	go uc.processBatchJob(context.Background(), batchID, templateID, targetID, resolvedVersion, items)

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

// processBatchJob runs assets concurrently (max 5 at a time) and creates
// pipeline runs via CreateRunByTemplateID. Errors are per-item — one failure
// does not cancel the batch.
func (uc *Usecase) processBatchJob(ctx context.Context, jobID, templateID, targetID string, templateVersion int, items []models.BackfillItem) {
	slog.Info("batch job started", "batchID", jobID, "totalItems", len(items))

	if err := uc.backfillRepo.UpdateJobStatus(ctx, jobID, "processing"); err != nil {
		slog.Warn("batch job: failed to update status to processing", "batchID", jobID, "err", err)
		return
	}

	sem := make(chan struct{}, 5)
	var wg sync.WaitGroup

	for i := range items {
		sem <- struct{}{}
		wg.Add(1)
		go func(item models.BackfillItem) {
			defer wg.Done()
			defer func() { <-sem }()

			_ = uc.backfillRepo.UpdateItemStatus(ctx, item.ID, "processing", "", "")

			opts := []DeployOptions{{
				TargetID:          targetID,
				TemplateVersion:   templateVersion,
				BatchJobID:        jobID,
				AllowUnknownAssets: true,
			}}

			run, err := uc.CreateRunByTemplateID(ctx, templateID, "", []string{item.AssetID}, opts...)
			if err != nil {
				errMsg := err.Error()
				slog.Warn("batch job: asset failed", "assetID", item.AssetID, "err", errMsg)
				_ = uc.backfillRepo.UpdateItemStatus(ctx, item.ID, "failed", "", errMsg)
				_ = uc.backfillRepo.IncrementFailed(ctx, jobID)
				return
			}

			_ = uc.backfillRepo.UpdateItemPipelineRun(ctx, item.ID, run.ID, run.WorkflowName, string(run.Status))
			_ = uc.backfillRepo.IncrementCompleted(ctx, jobID)
		}(items[i])
	}

	wg.Wait()

	summary, err := uc.backfillRepo.SummarizeItemStatuses(ctx, jobID)
	if err != nil {
		slog.Warn("batch job: summarize failed", "batchID", jobID, "err", err)
		_ = uc.backfillRepo.UpdateJobStatus(ctx, jobID, "failed")
		return
	}

	status := "completed"
	if summary.Failed > 0 && summary.Completed == 0 {
		status = "failed"
	}
	total := summary.Completed + summary.Failed + summary.Pending + summary.Running
	_ = uc.backfillRepo.UpdateJobProgress(ctx, jobID, summary.Completed, summary.Failed, status)

	slog.Info("batch job completed",
		"batchID", jobID,
		"total", total,
		"completed", summary.Completed,
		"failed", summary.Failed,
	)
}
