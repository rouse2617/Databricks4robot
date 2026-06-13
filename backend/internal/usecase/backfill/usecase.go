package backfill

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
	pipelineUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/pipeline"
)

const maxConcurrentBatchItems = 5

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

	sem := make(chan struct{}, maxConcurrentBatchItems)
	var wg sync.WaitGroup
	for _, item := range items {
		if len(allowedSet) > 0 {
			if _, ok := allowedSet[item.Status]; !ok {
				continue
			}
		}
		job, err := uc.repo.FindJobByID(ctx, jobID)
		if err != nil || job == nil || job.Status == "paused" {
			return
		}

		wg.Add(1)
		sem <- struct{}{}
		go func(item models.BackfillItem) {
			defer wg.Done()
			defer func() { <-sem }()
			_ = uc.executeItem(ctx, item, templateID, jobID)
		}(item)
	}
	wg.Wait()
	_ = uc.syncJobProgress(ctx, jobID)
}

// executeItem deploys the pipeline for a single backfill item and tracks status.
func (uc *Usecase) executeItem(ctx context.Context, item models.BackfillItem, templateID, jobID string) error {
	job, err := uc.repo.FindJobByID(ctx, item.JobID)
	if err != nil || job == nil || job.Status == "paused" {
		return nil
	}

	dep, err := uc.pipelineUC.DeployByTemplateID(
		ctx,
		templateID,
		"",
		[]string{item.AssetID},
		pipelineUC.DeployOptions{BatchJobID: jobID},
	)
	if err != nil {
		_ = uc.repo.UpdateItemStatus(ctx, item.ID, "failed", "", err.Error())
		_ = uc.syncJobProgress(ctx, jobID)
		return err
	}

	_ = uc.repo.UpdateItemPipelineRun(ctx, item.ID, dep.ID, dep.WorkflowName, "running")
	_ = uc.syncJobProgress(ctx, jobID)
	return nil
}

// ListJobs returns all backfill jobs with refreshed progress.
func (uc *Usecase) ListJobs(ctx context.Context) ([]models.BackfillJob, error) {
	jobs, err := uc.repo.FindAllJobs(ctx)
	if err != nil {
		return nil, err
	}
	for i := range jobs {
		_ = uc.syncJobProgress(ctx, jobs[i].ID)
		if refreshed, err := uc.repo.FindJobByID(ctx, jobs[i].ID); err == nil && refreshed != nil {
			jobs[i] = *refreshed
		}
	}
	return jobs, nil
}

// GetJob returns a backfill job without loading all items.
func (uc *Usecase) GetJob(ctx context.Context, id string) (*models.BackfillJob, error) {
	job, err := uc.repo.FindJobByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if job == nil {
		return nil, nil
	}
	_ = uc.syncJobProgress(ctx, id)
	return uc.repo.FindJobByID(ctx, id)
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
			_ = uc.repo.UpdateItemStatus(ctx, item.ID, "pending", "", "")
		}
	}
	if err := uc.repo.UpdateJobStatus(ctx, jobID, "running"); err != nil {
		return err
	}
	if uc.pipelineUC != nil {
		go uc.runItems(context.Background(), jobID, job.TemplateID, items, "pending")
	}
	return nil
}

func (uc *Usecase) syncJobProgress(ctx context.Context, jobID string) error {
	job, err := uc.repo.FindJobByID(ctx, jobID)
	if err != nil || job == nil {
		return err
	}
	if job.Status == "paused" {
		return nil
	}

	items, err := uc.repo.FindItemsByJobID(ctx, jobID)
	if err != nil {
		return err
	}

	var completed, failed, pending, running int
	for _, item := range items {
		status := item.Status
		if item.PipelineRunID != nil && *item.PipelineRunID != "" && uc.pipelineUC != nil {
			run, err := uc.pipelineUC.GetRun(ctx, *item.PipelineRunID)
			if err == nil && run != nil {
				mapped := mapRunStatusToItem(run.Status)
				if mapped != status {
					status = mapped
					wf := run.WorkflowName
					_ = uc.repo.UpdateItemStatus(ctx, item.ID, mapped, wf, "")
				}
			}
		}
		switch status {
		case "completed":
			completed++
		case "failed", "cancelled":
			failed++
		case "running":
			running++
		default:
			pending++
		}
	}

	jobStatus := job.Status
	switch {
	case pending > 0 || running > 0:
		jobStatus = "running"
	case failed > 0 && completed+failed == job.TotalCount:
		jobStatus = "failed"
	case completed == job.TotalCount:
		jobStatus = "completed"
	}
	return uc.repo.UpdateJobProgress(ctx, jobID, completed, failed, jobStatus)
}

func mapRunStatusToItem(runStatus string) string {
	switch strings.ToLower(runStatus) {
	case "succeeded", "success":
		return "completed"
	case "failed", "error":
		return "failed"
	case "running", "pending":
		return "running"
	default:
		return "running"
	}
}
