package pipeline

import (
	"context"
	"fmt"
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
