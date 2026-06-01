package backfill

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
	pipelineUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/pipeline"
)

var ErrNotFound = errors.New("backfill job not found")

// Usecase orchestrates backfill job operations.
type Usecase struct {
	repo       repository.BackfillRepository
	pipelineUC *pipelineUC.Usecase
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

	if uc.pipelineUC != nil {
		go uc.runItems(context.Background(), job.ID, templateID, items, "pending")
	}

	return job, nil
}

func (uc *Usecase) runItems(ctx context.Context, jobID, templateID string, items []models.BackfillItem, allowed ...string) {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, status := range allowed {
		allowedSet[status] = struct{}{}
	}
	for _, item := range items {
		if _, ok := allowedSet[item.Status]; !ok {
			continue
		}
		job, err := uc.repo.FindJobByID(ctx, jobID)
		if err != nil || job == nil || job.Status == "paused" {
			return
		}
		_ = uc.executeItem(ctx, item, templateID)
	}
}

// executeItem deploys the pipeline for a single backfill item and tracks status.
func (uc *Usecase) executeItem(ctx context.Context, item models.BackfillItem, templateID string) error {
	job, err := uc.repo.FindJobByID(ctx, item.JobID)
	if err != nil || job == nil || job.Status == "paused" {
		return nil
	}

	_ = uc.repo.UpdateItemStatus(ctx, item.ID, "running", "", "")

	dep, err := uc.pipelineUC.DeployByTemplateID(ctx, templateID, "", []string{item.AssetID})
	if err != nil {
		_ = uc.repo.UpdateItemStatus(ctx, item.ID, "failed", "", err.Error())
		_ = uc.repo.IncrementFailed(ctx, item.JobID)
		return err
	}

	wfName := dep.WorkflowName
	_ = uc.repo.UpdateItemStatus(ctx, item.ID, "completed", wfName, "")
	_ = uc.repo.IncrementCompleted(ctx, item.JobID)
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

// ResumeJob resumes a paused backfill job and re-schedules pending items.
func (uc *Usecase) ResumeJob(ctx context.Context, id string) error {
	job, err := uc.repo.FindJobByID(ctx, id)
	if err != nil {
		return err
	}
	if job == nil {
		return ErrNotFound
	}
	if err := uc.repo.UpdateJobStatus(ctx, id, "running"); err != nil {
		return err
	}
	items, err := uc.repo.FindItemsByJobID(ctx, id)
	if err != nil {
		return err
	}
	if uc.pipelineUC != nil {
		go uc.runItems(context.Background(), id, job.TemplateID, items, "pending")
	}
	return nil
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
		return ErrNotFound
	}

	for _, item := range items {
		if item.Status == "failed" {
			_ = uc.executeItem(ctx, item, job.TemplateID)
		}
	}
	return nil
}
