package backfill

import (
	"context"
	"errors"
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
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
