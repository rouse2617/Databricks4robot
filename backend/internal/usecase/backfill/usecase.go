package backfill

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
	pipelineUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/pipeline"
)

// Usecase orchestrates backfill job operations.
type Usecase struct {
	repo        repository.BackfillRepository
	pipelineUC  *pipelineUC.Usecase
}

// New creates a Usecase.
func New(repo repository.BackfillRepository, pipelineUC *pipelineUC.Usecase) *Usecase {
	return &Usecase{repo: repo, pipelineUC: pipelineUC}
}

// CreateBackfill creates a new backfill job and its items.
func (uc *Usecase) CreateBackfill(ctx context.Context, name, templateID string, assetIDs []string) (*models.BackfillJob, error) {
	job := &models.BackfillJob{
		ID:         uuid.New().String(),
		Name:       name,
		TemplateID: templateID,
		TotalCount: len(assetIDs),
		Status:     "running",
		CreatedAt:  time.Now().UTC(),
	}
	if err := uc.repo.SaveJob(ctx, job); err != nil {
		return nil, fmt.Errorf("save backfill job: %w", err)
	}

	// Create items for each asset.
	items := make([]models.BackfillItem, len(assetIDs))
	for i, aid := range assetIDs {
		items[i] = models.BackfillItem{
			ID:      uuid.New().String(),
			JobID:   job.ID,
			AssetID: aid,
			Status:  "pending",
		}
	}
	if err := uc.repo.SaveItems(ctx, items); err != nil {
		return nil, fmt.Errorf("save backfill items: %w", err)
	}

	// Deploy pipeline for each item in the background.
	// TODO: make this async with a worker queue for large backfills.
	go func() {
		for _, item := range items {
			_ = uc.executeItem(context.Background(), item, templateID)
		}
	}()

	return job, nil
}

// executeItem deploys the pipeline for a single backfill item and tracks status.
func (uc *Usecase) executeItem(ctx context.Context, item models.BackfillItem, templateID string) error {
	_ = uc.repo.UpdateItemStatus(ctx, item.ID, "running", "", "")

	dep, err := uc.pipelineUC.DeployByTemplateID(ctx, templateID, "", []string{item.AssetID})
	if err != nil {
		_ = uc.repo.UpdateItemStatus(ctx, item.ID, "failed", "", err.Error())
		_ = uc.repo.IncrementFailed(ctx, item.JobID)
		return err
	}

	wfName := dep.WorkflowName
	_ = uc.repo.UpdateItemStatus(ctx, item.ID, "running", wfName, "")
	return nil
}

// ListJobs returns all backfill jobs.
func (uc *Usecase) ListJobs(ctx context.Context) ([]models.BackfillJob, error) {
	return uc.repo.FindAllJobs(ctx)
}

// GetJob returns a backfill job with its items.
func (uc *Usecase) GetJob(ctx context.Context, id string) (*models.BackfillJob, []models.BackfillItem, error) {
	job, err := uc.repo.FindJobByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if job == nil {
		return nil, nil, nil
	}

	items, err := uc.repo.FindItemsByJobID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	return job, items, nil
}

// PauseJob pauses a running backfill job.
func (uc *Usecase) PauseJob(ctx context.Context, id string) error {
	return uc.repo.UpdateJobStatus(ctx, id, "paused")
}

// ResumeJob resumes a paused backfill job.
func (uc *Usecase) ResumeJob(ctx context.Context, id string) error {
	return uc.repo.UpdateJobStatus(ctx, id, "running")
}

// RetryFailed retries all failed items for a backfill job.
func (uc *Usecase) RetryFailed(ctx context.Context, jobID string) error {
	items, err := uc.repo.FindItemsByJobID(ctx, jobID)
	if err != nil {
		return err
	}

	job, err := uc.repo.FindJobByID(ctx, jobID)
	if err != nil {
		return err
	}
	if job == nil {
		return fmt.Errorf("backfill job not found: %s", jobID)
	}

	for _, item := range items {
		if item.Status == "failed" {
			_ = uc.executeItem(ctx, item, job.TemplateID)
		}
	}
	return nil
}
