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
func (m *mockBackfillRepo) UpdateJobPilotPhase(_ context.Context, id, status, pilotPhase string) error {
	if m.jobs != nil {
		if j, ok := m.jobs[id]; ok {
			j.Status = status
			j.PilotPhase = pilotPhase
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
func (m *mockBackfillRepo) FindItemByPipelineRunID(_ context.Context, _ string) (*models.BackfillItem, error) {
	return nil, nil
}
func (m *mockBackfillRepo) FindItemByJobAndAssetID(_ context.Context, jobID, assetID string) (*models.BackfillItem, error) {
	for i := range m.items {
		if m.items[i].JobID == jobID && m.items[i].AssetID == assetID {
			item := m.items[i]
			return &item, nil
		}
	}
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
func (m *mockBackfillRepo) FindItemsByScope(_ context.Context, filter repository.BackfillRerunItemFilter) ([]models.BackfillItem, error) {
	out := []models.BackfillItem{}
	for _, item := range m.items {
		if item.JobID != filter.JobID {
			continue
		}
		if len(filter.Statuses) > 0 && !containsString(filter.Statuses, item.Status) {
			continue
		}
		if len(filter.ItemIDs) > 0 && !containsString(filter.ItemIDs, item.ID) {
			continue
		}
		if len(filter.AssetIDs) > 0 && !containsString(filter.AssetIDs, item.AssetID) {
			continue
		}
		out = append(out, item)
	}
	return out, nil
}
func (m *mockBackfillRepo) PrepareItemsForRerun(_ context.Context, itemIDs []string) error {
	for _, id := range itemIDs {
		_ = m.UpdateItemStatus(context.Background(), id, "pending", "", "")
	}
	return nil
}
func (m *mockBackfillRepo) AggregateNodeStatusByBatchJobID(_ context.Context, _ string) ([]repository.BatchNodeStatusAggregate, error) {
	return nil, nil
}
func (m *mockBackfillRepo) ListNodeFailures(_ context.Context, _ repository.BatchNodeFailureFilter) (*models.BatchNodeFailureListResult, error) {
	return &models.BatchNodeFailureListResult{}, nil
}
func (m *mockBackfillRepo) CountPipelineRunsByBatchJobID(_ context.Context, _ string) (int, error) {
	return 0, nil
}
func (m *mockBackfillRepo) CountRunsWithNodeRowsByBatchJobID(_ context.Context, _ string) (int, error) {
	return 0, nil
}
func (m *mockBackfillRepo) FindItemsByAssetID(_ context.Context, _ string) ([]models.BackfillItem, error) {
	return nil, nil
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
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

func TestCreateBackfill_DoesNotDuplicateItemsDuringMaterialization(t *testing.T) {
	repo := &trackingBackfillRepo{}
	uc := New(repo, nil)

	job, err := uc.CreateBackfill(context.Background(), "batch", "tpl-1", []string{
		"asset-1",
		"asset-2",
		"asset-3",
	})
	if err != nil {
		t.Fatalf("CreateBackfill: %v", err)
	}

	waitForTrackingRepo(t, repo, func(job *models.BackfillJob, items []models.BackfillItem) bool {
		return job != nil && job.Status == "running" && len(items) == 3
	})

	repo.mu.Lock()
	defer repo.mu.Unlock()
	if len(repo.items) != job.TotalCount {
		t.Fatalf("expected %d persisted items, got %d", job.TotalCount, len(repo.items))
	}
	if len(repo.saveItemChunkSizes) != 1 || repo.saveItemChunkSizes[0] != 3 {
		t.Fatalf("expected one initial SaveItems chunk [3], got %v", repo.saveItemChunkSizes)
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

func TestGetBatchNodeSummary_UsesLogicalBatchTotalForCoverage(t *testing.T) {
	repo := &trackingBackfillRepo{
		job: &models.BackfillJob{
			ID:              "job-1",
			TemplateID:      "tpl-1",
			TemplateVersion: 3,
			TotalCount:      100,
			Status:          "running",
		},
		items: make([]models.BackfillItem, 100),
		aggregates: []repository.BatchNodeStatusAggregate{
			{PipelineNodeID: "step-1", DisplayName: "step-1", Status: "Succeeded", Count: 100},
		},
		runsWithNodeRows: 100,
	}
	for i := range repo.items {
		repo.items[i] = models.BackfillItem{
			ID:      fmt.Sprintf("item-%03d", i),
			JobID:   "job-1",
			AssetID: fmt.Sprintf("asset-%03d", i),
			Status:  "completed",
		}
	}
	uc := New(repo, nil)

	summary, err := uc.GetBatchNodeSummary(context.Background(), "job-1")
	if err != nil {
		t.Fatalf("GetBatchNodeSummary: %v", err)
	}
	if summary.Subtasks.Total != 100 || summary.Subtasks.Completed != 100 || summary.Subtasks.Pending != 0 {
		t.Fatalf("unexpected subtask counts: %+v", summary.Subtasks)
	}
	if summary.DataCoverage.RunsTotal != 100 || summary.DataCoverage.RunsWithNodeRows != 100 || !summary.DataCoverage.Complete {
		t.Fatalf("unexpected data coverage: %+v", summary.DataCoverage)
	}
	if len(summary.Nodes) != 1 {
		t.Fatalf("expected one node, got %d", len(summary.Nodes))
	}
	node := summary.Nodes[0]
	if node.Counts["Succeeded"] != 100 || node.Counts["Pending"] != 0 || node.Attempted != 100 {
		t.Fatalf("unexpected node summary: %+v", node)
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

	waitForTrackingRepo(t, repo, func(job *models.BackfillJob, items []models.BackfillItem) bool {
		return job != nil && job.Status == "running" && len(items) == 1000
	})

	repo.mu.Lock()
	defer repo.mu.Unlock()
	if len(repo.saveItemChunkSizes) != 2 || repo.saveItemChunkSizes[0] != 500 || repo.saveItemChunkSizes[1] != 500 {
		t.Fatalf("expected chunk sizes [500,500], got %v", repo.saveItemChunkSizes)
	}
	if len(repo.items) != 1000 {
		t.Fatalf("expected 1000 persisted items, got %d", len(repo.items))
	}
}

func waitForTrackingRepo(
	t *testing.T,
	repo *trackingBackfillRepo,
	ready func(*models.BackfillJob, []models.BackfillItem) bool,
) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		repo.mu.Lock()
		var job *models.BackfillJob
		if repo.job != nil {
			copyJob := *repo.job
			job = &copyJob
		}
		items := append([]models.BackfillItem(nil), repo.items...)
		repo.mu.Unlock()
		if ready(job, items) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	repo.mu.Lock()
	defer repo.mu.Unlock()
	status := ""
	if repo.job != nil {
		status = repo.job.Status
	}
	t.Fatalf("condition not met in time: chunks=%v itemCount=%d jobStatus=%q", repo.saveItemChunkSizes, len(repo.items), status)
}

type trackingBackfillRepo struct {
	mu                 sync.Mutex
	job                *models.BackfillJob
	saveItemChunkSizes []int
	items              []models.BackfillItem
	aggregates         []repository.BatchNodeStatusAggregate
	runsTotal          int
	runsWithNodeRows   int
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
func (r *trackingBackfillRepo) UpdateJobPilotPhase(_ context.Context, _ string, status, pilotPhase string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.job != nil {
		r.job.Status = status
		r.job.PilotPhase = pilotPhase
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
func (r *trackingBackfillRepo) FindItemByPipelineRunID(_ context.Context, _ string) (*models.BackfillItem, error) {
	return nil, nil
}
func (r *trackingBackfillRepo) FindItemByJobAndAssetID(_ context.Context, jobID, assetID string) (*models.BackfillItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.items {
		if r.items[i].JobID == jobID && r.items[i].AssetID == assetID {
			item := r.items[i]
			return &item, nil
		}
	}
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
func (r *trackingBackfillRepo) FindItemsByScope(_ context.Context, filter repository.BackfillRerunItemFilter) ([]models.BackfillItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []models.BackfillItem{}
	for _, item := range r.items {
		if filter.JobID != "" && item.JobID != filter.JobID {
			continue
		}
		if len(filter.Statuses) > 0 && !containsString(filter.Statuses, item.Status) {
			continue
		}
		out = append(out, item)
	}
	return out, nil
}
func (r *trackingBackfillRepo) PrepareItemsForRerun(_ context.Context, itemIDs []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.items {
		if containsString(itemIDs, r.items[i].ID) {
			r.items[i].Status = "pending"
			r.items[i].ErrorMessage = nil
			r.items[i].StartedAt = nil
			r.items[i].FinishedAt = nil
		}
	}
	return nil
}
func (r *trackingBackfillRepo) AggregateNodeStatusByBatchJobID(_ context.Context, _ string) ([]repository.BatchNodeStatusAggregate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]repository.BatchNodeStatusAggregate(nil), r.aggregates...), nil
}
func (r *trackingBackfillRepo) ListNodeFailures(_ context.Context, _ repository.BatchNodeFailureFilter) (*models.BatchNodeFailureListResult, error) {
	return &models.BatchNodeFailureListResult{}, nil
}
func (r *trackingBackfillRepo) CountPipelineRunsByBatchJobID(_ context.Context, _ string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.runsTotal, nil
}
func (r *trackingBackfillRepo) CountRunsWithNodeRowsByBatchJobID(_ context.Context, _ string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.runsWithNodeRows, nil
}
func (r *trackingBackfillRepo) FindItemsByAssetID(_ context.Context, _ string) ([]models.BackfillItem, error) {
	return nil, nil
}
