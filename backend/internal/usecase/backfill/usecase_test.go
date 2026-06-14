package backfill

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

type mockBackfillRepo struct {
	jobs  map[string]*models.BackfillJob
	items []models.BackfillItem
}

func (m *mockBackfillRepo) SaveJob(_ context.Context, _ *models.BackfillJob) error { return nil }
func (m *mockBackfillRepo) FindAllJobs(_ context.Context) ([]models.BackfillJob, error) {
	return nil, nil
}
func (m *mockBackfillRepo) FindJobByID(_ context.Context, id string) (*models.BackfillJob, error) {
	if id == "missing" {
		return nil, nil
	}
	if m.jobs != nil {
		if j, ok := m.jobs[id]; ok {
			return j, nil
		}
		return nil, nil
	}
	return &models.BackfillJob{ID: id, TemplateID: "tpl-1"}, nil
}
func (m *mockBackfillRepo) UpdateJobStatus(_ context.Context, id, status string) error {
	if m.jobs != nil {
		if j, ok := m.jobs[id]; ok {
			j.Status = status
		}
	}
	return nil
}
func (m *mockBackfillRepo) IncrementCompleted(_ context.Context, _ string) error { return nil }
func (m *mockBackfillRepo) IncrementFailed(_ context.Context, _ string) error    { return nil }
func (m *mockBackfillRepo) SaveItem(_ context.Context, _ *models.BackfillItem) error {
	return nil
}
func (m *mockBackfillRepo) SaveItems(_ context.Context, _ []models.BackfillItem) error { return nil }
func (m *mockBackfillRepo) FindItemsByJobID(_ context.Context, _ string) ([]models.BackfillItem, error) {
	return m.items, nil
}
func (m *mockBackfillRepo) FindItemByID(_ context.Context, _ string) (*models.BackfillItem, error) {
	return nil, nil
}
func (m *mockBackfillRepo) UpdateItemStatus(_ context.Context, id, status, wf, errMsg string) error {
	for i := range m.items {
		if m.items[i].ID == id {
			m.items[i].Status = status
			if wf != "" {
				m.items[i].WorkflowName = &wf
			}
			if errMsg != "" {
				m.items[i].ErrorMessage = &errMsg
			}
		}
	}
	return nil
}
func (m *mockBackfillRepo) UpdateItemPipelineRun(_ context.Context, id, pipelineRunID, workflowName, status string) error {
	for i := range m.items {
		if m.items[i].ID == id {
			m.items[i].Status = status
			if pipelineRunID != "" {
				m.items[i].PipelineRunID = &pipelineRunID
			}
			if workflowName != "" {
				m.items[i].WorkflowName = &workflowName
			}
		}
	}
	return nil
}
func (m *mockBackfillRepo) UpdateJobProgress(_ context.Context, id string, completed, failed int, status string) error {
	if m.jobs != nil {
		if j, ok := m.jobs[id]; ok {
			j.CompletedCount = completed
			j.FailedCount = failed
			j.Status = status
		}
	}
	return nil
}
func (m *mockBackfillRepo) CountItemsByStatus(_ context.Context, _, _ string) (int, error) {
	return 0, nil
}
func (m *mockBackfillRepo) SummarizeItemStatuses(_ context.Context, _ string) (repository.BackfillItemStatusSummary, error) {
	return repository.BackfillItemStatusSummary{}, nil
}
func (m *mockBackfillRepo) FindItemsByJobIDWithStatuses(_ context.Context, _ string, _ []string) ([]models.BackfillItem, error) {
	return nil, nil
}
func (m *mockBackfillRepo) FindItemsMissingPipelineRun(_ context.Context, _ string) ([]models.BackfillItem, error) {
	return nil, nil
}

func TestRetryFailed_JobNotFound(t *testing.T) {
	uc := New(&mockBackfillRepo{}, nil)
	err := uc.RetryFailed(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestResumeJob_SchedulesPendingItems(t *testing.T) {
	repo := &mockBackfillRepo{
		jobs: map[string]*models.BackfillJob{
			"job-1": {ID: "job-1", TemplateID: "tpl-1", Status: "paused"},
		},
		items: []models.BackfillItem{
			{ID: "item-1", JobID: "job-1", AssetID: "asset-1", Status: "pending"},
			{ID: "item-2", JobID: "job-1", AssetID: "asset-2", Status: "completed"},
		},
	}
	uc := New(repo, nil)
	if err := uc.ResumeJob(context.Background(), "job-1"); err != nil {
		t.Fatalf("ResumeJob: %v", err)
	}
	if repo.jobs["job-1"].Status != "running" {
		t.Fatalf("expected running, got %q", repo.jobs["job-1"].Status)
	}
}

func TestCreateBackfill_AcceptsUnknownAssetIDs(t *testing.T) {
	uc := New(&mockBackfillRepo{}, nil)
	job, err := uc.CreateBackfill(context.Background(), "manual-batch", "tpl-1", []string{
		" custom-a ",
		"custom-b",
		"custom-a",
	})
	if err != nil {
		t.Fatalf("CreateBackfill: %v", err)
	}
	if job.TotalCount != 2 {
		t.Fatalf("expected 2 items after dedupe, got %d", job.TotalCount)
	}
}

func TestCreateBackfill_RejectsTooManyAssets(t *testing.T) {
	uc := New(&mockBackfillRepo{}, nil)
	assetIDs := make([]string, MaxBackfillAssetCount+1)
	for i := range assetIDs {
		assetIDs[i] = fmt.Sprintf("asset-%d", i)
	}
	_, err := uc.CreateBackfill(context.Background(), "too-big", "tpl-1", assetIDs)
	if !errors.Is(err, ErrTooManyAssets) {
		t.Fatalf("expected ErrTooManyAssets, got %v", err)
	}
}

func TestDeriveJobStatus(t *testing.T) {
	status := deriveJobStatus(repository.BackfillItemStatusSummary{
		Completed: 8,
		Failed:    2,
	}, 10)
	if status != "failed" {
		t.Fatalf("expected failed, got %q", status)
	}
	status = deriveJobStatus(repository.BackfillItemStatusSummary{
		Completed: 10,
	}, 10)
	if status != "completed" {
		t.Fatalf("expected completed, got %q", status)
	}
}

func TestCreateBackfill_1000Assets_ReturnsPendingImmediately(t *testing.T) {
	repo := &trackingBackfillRepo{}
	uc := New(repo, nil)

	assetIDs := make([]string, 1000)
	for i := range assetIDs {
		assetIDs[i] = fmt.Sprintf("asset-%04d", i)
	}

	job, err := uc.CreateBackfill(context.Background(), "large-batch", "tpl-1", assetIDs)
	if err != nil {
		t.Fatalf("CreateBackfill: %v", err)
	}
	if job.Status != "pending" {
		t.Fatalf("expected immediate pending status, got %q", job.Status)
	}
	if job.TotalCount != 1000 {
		t.Fatalf("expected totalCount 1000, got %d", job.TotalCount)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		repo.mu.Lock()
		chunkSizes := append([]int(nil), repo.saveItemChunkSizes...)
		status := ""
		if repo.job != nil {
			status = repo.job.Status
		}
		repo.mu.Unlock()
		if len(chunkSizes) >= 2 && status == "running" {
			if chunkSizes[0] != 500 || chunkSizes[1] != 500 {
				t.Fatalf("expected chunk sizes [500,500], got %v", chunkSizes)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	repo.mu.Lock()
	defer repo.mu.Unlock()
	t.Fatalf(
		"materialize did not finish in time: chunks=%v jobStatus=%q",
		repo.saveItemChunkSizes,
		func() string {
			if repo.job == nil {
				return ""
			}
			return repo.job.Status
		}(),
	)
}

type trackingBackfillRepo struct {
	mu                 sync.Mutex
	job                *models.BackfillJob
	saveItemChunkSizes []int
	items              []models.BackfillItem
}

func (r *trackingBackfillRepo) SaveJob(_ context.Context, job *models.BackfillJob) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	copyJob := *job
	r.job = &copyJob
	return nil
}
func (r *trackingBackfillRepo) FindAllJobs(_ context.Context) ([]models.BackfillJob, error) {
	return nil, nil
}
func (r *trackingBackfillRepo) FindJobByID(_ context.Context, id string) (*models.BackfillJob, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.job != nil && r.job.ID == id {
		copyJob := *r.job
		return &copyJob, nil
	}
	return nil, nil
}
func (r *trackingBackfillRepo) UpdateJobStatus(_ context.Context, _ string, status string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.job != nil {
		r.job.Status = status
	}
	return nil
}
func (r *trackingBackfillRepo) IncrementCompleted(_ context.Context, _ string) error { return nil }
func (r *trackingBackfillRepo) IncrementFailed(_ context.Context, _ string) error    { return nil }
func (r *trackingBackfillRepo) SaveItem(_ context.Context, _ *models.BackfillItem) error {
	return nil
}
func (r *trackingBackfillRepo) SaveItems(_ context.Context, items []models.BackfillItem) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.saveItemChunkSizes = append(r.saveItemChunkSizes, len(items))
	r.items = append(r.items, items...)
	return nil
}
func (r *trackingBackfillRepo) FindItemsByJobID(_ context.Context, _ string) ([]models.BackfillItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]models.BackfillItem(nil), r.items...), nil
}
func (r *trackingBackfillRepo) FindItemByID(_ context.Context, _ string) (*models.BackfillItem, error) {
	return nil, nil
}
func (r *trackingBackfillRepo) UpdateItemStatus(_ context.Context, id, status, wf, errMsg string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.items {
		if r.items[i].ID == id {
			r.items[i].Status = status
			if wf != "" {
				r.items[i].WorkflowName = &wf
			}
			if errMsg != "" {
				r.items[i].ErrorMessage = &errMsg
			}
		}
	}
	return nil
}
func (r *trackingBackfillRepo) UpdateItemPipelineRun(_ context.Context, id, pipelineRunID, workflowName, status string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.items {
		if r.items[i].ID == id {
			r.items[i].Status = status
			if pipelineRunID != "" {
				r.items[i].PipelineRunID = &pipelineRunID
			}
			if workflowName != "" {
				r.items[i].WorkflowName = &workflowName
			}
		}
	}
	return nil
}
func (r *trackingBackfillRepo) UpdateJobProgress(_ context.Context, _ string, completed, failed int, status string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.job != nil {
		r.job.CompletedCount = completed
		r.job.FailedCount = failed
		r.job.Status = status
	}
	return nil
}
func (r *trackingBackfillRepo) CountItemsByStatus(_ context.Context, _, _ string) (int, error) {
	return 0, nil
}
func (r *trackingBackfillRepo) SummarizeItemStatuses(_ context.Context, _ string) (repository.BackfillItemStatusSummary, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var summary repository.BackfillItemStatusSummary
	for _, item := range r.items {
		switch item.Status {
		case "completed":
			summary.Completed++
		case "failed", "cancelled":
			summary.Failed++
		case "running":
			summary.Running++
		default:
			summary.Pending++
		}
	}
	return summary, nil
}
func (r *trackingBackfillRepo) FindItemsByJobIDWithStatuses(_ context.Context, _ string, statuses []string) ([]models.BackfillItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	allowed := make(map[string]struct{}, len(statuses))
	for _, status := range statuses {
		allowed[status] = struct{}{}
	}
	var out []models.BackfillItem
	for _, item := range r.items {
		if _, ok := allowed[item.Status]; ok {
			out = append(out, item)
		}
	}
	return out, nil
}
func (r *trackingBackfillRepo) FindItemsMissingPipelineRun(_ context.Context, _ string) ([]models.BackfillItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []models.BackfillItem
	for _, item := range r.items {
		if item.PipelineRunID == nil || strings.TrimSpace(*item.PipelineRunID) == "" {
			out = append(out, item)
		}
	}
	return out, nil
}
