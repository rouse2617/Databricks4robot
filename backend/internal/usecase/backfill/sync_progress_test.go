package backfill

import (
	"context"
	"io"
	"sync/atomic"
	"testing"
	"time"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/CyberOrigin2077/cyber-databrew/internal/argo"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
	pipelineUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/pipeline"
)

type pausedSyncRepo struct {
	job             *models.BackfillJob
	items           []models.BackfillItem
	findJobByIDHook func()
}

func (r *pausedSyncRepo) SaveJob(context.Context, *models.BackfillJob) error { return nil }
func (r *pausedSyncRepo) FindAllJobs(context.Context) ([]models.BackfillJob, error) {
	return nil, nil
}
func (r *pausedSyncRepo) FindJobByID(_ context.Context, id string) (*models.BackfillJob, error) {
	if r.job != nil && r.job.ID == id {
		if r.findJobByIDHook != nil {
			r.findJobByIDHook()
		}
		copy := *r.job
		return &copy, nil
	}
	return nil, nil
}
func (r *pausedSyncRepo) UpdateJobStatus(_ context.Context, id, status string) error {
	if r.job != nil && r.job.ID == id {
		r.job.Status = status
	}
	return nil
}
func (r *pausedSyncRepo) UpdateJobPilotPhase(_ context.Context, id, status, pilotPhase string) error {
	if r.job != nil && r.job.ID == id {
		r.job.Status = status
		r.job.PilotPhase = pilotPhase
	}
	return nil
}
func (r *pausedSyncRepo) IncrementCompleted(context.Context, string) error { return nil }
func (r *pausedSyncRepo) IncrementFailed(context.Context, string) error    { return nil }
func (r *pausedSyncRepo) SaveItem(context.Context, *models.BackfillItem) error {
	return nil
}
func (r *pausedSyncRepo) SaveItems(context.Context, []models.BackfillItem) error { return nil }
func (r *pausedSyncRepo) FindItemsByJobID(_ context.Context, jobID string) ([]models.BackfillItem, error) {
	out := make([]models.BackfillItem, 0)
	for _, item := range r.items {
		if item.JobID == jobID {
			out = append(out, item)
		}
	}
	return out, nil
}
func (r *pausedSyncRepo) FindItemByID(context.Context, string) (*models.BackfillItem, error) {
	return nil, nil
}
func (r *pausedSyncRepo) FindItemByPipelineRunID(context.Context, string) (*models.BackfillItem, error) {
	return nil, nil
}
func (r *pausedSyncRepo) FindItemByJobAndAssetID(context.Context, string, string) (*models.BackfillItem, error) {
	return nil, nil
}
func (r *pausedSyncRepo) UpdateItemStatus(_ context.Context, id, status, wf, errMsg string) error {
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
func (r *pausedSyncRepo) UpdateItemPipelineRun(_ context.Context, id, pipelineRunID, workflowName, status string) error {
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
func (r *pausedSyncRepo) UpdateJobProgress(_ context.Context, id string, completed, failed int, status string) error {
	if r.job != nil && r.job.ID == id {
		r.job.CompletedCount = completed
		r.job.FailedCount = failed
		r.job.Status = status
	}
	return nil
}
func (r *pausedSyncRepo) CountItemsByStatus(context.Context, string, string) (int, error) {
	return 0, nil
}
func (r *pausedSyncRepo) SummarizeItemStatuses(_ context.Context, jobID string) (repository.BackfillItemStatusSummary, error) {
	var summary repository.BackfillItemStatusSummary
	for _, item := range r.items {
		if item.JobID != jobID {
			continue
		}
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
func (r *pausedSyncRepo) FindItemsByJobIDWithStatuses(_ context.Context, jobID string, statuses []string) ([]models.BackfillItem, error) {
	out := make([]models.BackfillItem, 0)
	for _, item := range r.items {
		if item.JobID != jobID {
			continue
		}
		for _, status := range statuses {
			if item.Status == status {
				out = append(out, item)
				break
			}
		}
	}
	return out, nil
}
func (r *pausedSyncRepo) FindItemsMissingPipelineRun(context.Context, string) ([]models.BackfillItem, error) {
	return nil, nil
}
func (r *pausedSyncRepo) FindItemsByScope(context.Context, repository.BackfillRerunItemFilter) ([]models.BackfillItem, error) {
	return nil, nil
}
func (r *pausedSyncRepo) PrepareItemsForRerun(context.Context, []string) error { return nil }
func (r *pausedSyncRepo) AggregateNodeStatusByBatchJobID(context.Context, string) ([]repository.BatchNodeStatusAggregate, error) {
	return nil, nil
}
func (r *pausedSyncRepo) ListNodeFailures(context.Context, repository.BatchNodeFailureFilter) (*models.BatchNodeFailureListResult, error) {
	return &models.BatchNodeFailureListResult{}, nil
}
func (r *pausedSyncRepo) CountPipelineRunsByBatchJobID(context.Context, string) (int, error) {
	return 0, nil
}
func (r *pausedSyncRepo) CountRunsWithNodeRowsByBatchJobID(context.Context, string) (int, error) {
	return 0, nil
}
func (r *pausedSyncRepo) FindItemsByAssetID(context.Context, string) ([]models.BackfillItem, error) {
	return nil, nil
}

type syncTestRunRepo struct {
	byID           map[string]*models.PipelineRun
	summaries      []models.PipelineRun
	lastListFilter *models.PipelineRunListFilter
}

func (m *syncTestRunRepo) Save(context.Context, *models.PipelineRun) error { return nil }
func (m *syncTestRunRepo) FindAll(context.Context) ([]models.PipelineRun, error) {
	return nil, nil
}
func (m *syncTestRunRepo) FindAllSummaries(context.Context) ([]models.PipelineRun, error) {
	return nil, nil
}
func (m *syncTestRunRepo) ListSummaries(_ context.Context, filter models.PipelineRunListFilter) ([]models.PipelineRun, int, error) {
	m.lastListFilter = &filter
	if m.summaries == nil {
		return nil, 0, nil
	}
	out := make([]models.PipelineRun, len(m.summaries))
	copy(out, m.summaries)
	return out, len(out), nil
}
func (m *syncTestRunRepo) FindByID(_ context.Context, id string) (*models.PipelineRun, error) {
	if m.byID == nil {
		return nil, nil
	}
	return m.byID[id], nil
}
func (m *syncTestRunRepo) FindSummaryByID(_ context.Context, id string) (*models.PipelineRun, error) {
	if m.byID == nil {
		return nil, nil
	}
	return m.byID[id], nil
}
func (m *syncTestRunRepo) FindByWorkflowName(context.Context, string) (*models.PipelineRun, error) {
	return nil, nil
}
func (m *syncTestRunRepo) FindByBatchJobAndAssetID(context.Context, string, string) (*models.PipelineRun, error) {
	return nil, nil
}
func (m *syncTestRunRepo) FindAllByBatchJobAndAssetID(context.Context, string, string) ([]models.PipelineRun, error) {
	return nil, nil
}
func (m *syncTestRunRepo) Delete(context.Context, string) error { return nil }
func (m *syncTestRunRepo) DeleteByTemplateID(context.Context, string) error {
	return nil
}
func (m *syncTestRunRepo) UpdateStatus(_ context.Context, id, status string, finishedAt *time.Time) error {
	if r, ok := m.byID[id]; ok {
		r.Status = status
		r.FinishedAt = finishedAt
	}
	return nil
}
func (m *syncTestRunRepo) UpdateLedgerState(context.Context, string, string) error { return nil }

type syncTestWorkflowClient struct{}

func (syncTestWorkflowClient) CreateWorkflow(context.Context, *wfv1.Workflow, string) error {
	return nil
}
func (syncTestWorkflowClient) GetWorkflow(context.Context, string, string) (*wfv1.Workflow, error) {
	return &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "wf-1", UID: "uid-1"},
		Status:     wfv1.WorkflowStatus{Phase: wfv1.WorkflowSucceeded},
	}, nil
}
func (syncTestWorkflowClient) GetWorkflowStatus(context.Context, string, string) (wfv1.WorkflowPhase, error) {
	return wfv1.WorkflowSucceeded, nil
}
func (syncTestWorkflowClient) DeleteWorkflow(context.Context, string, string) error { return nil }
func (syncTestWorkflowClient) ListWorkflows(context.Context, string, string) ([]wfv1.Workflow, error) {
	return nil, nil
}
func (syncTestWorkflowClient) StopWorkflow(context.Context, string, string) error { return nil }
func (syncTestWorkflowClient) GetWorkflowLogs(context.Context, string, string, string, argo.WorkflowLogOptions) (argo.WorkflowLogResult, error) {
	return argo.WorkflowLogResult{}, nil
}
func (syncTestWorkflowClient) GetWorkflowLogStream(context.Context, string, string, string, argo.WorkflowLogOptions) (io.ReadCloser, error) {
	return io.NopCloser(nil), nil
}
func (syncTestWorkflowClient) RetryWorkflow(context.Context, string, string) error     { return nil }
func (syncTestWorkflowClient) ResubmitWorkflow(context.Context, string, string) error  { return nil }
func (syncTestWorkflowClient) SuspendWorkflow(context.Context, string, string) error   { return nil }
func (syncTestWorkflowClient) ResumeWorkflow(context.Context, string, string) error    { return nil }
func (syncTestWorkflowClient) TerminateWorkflow(context.Context, string, string) error { return nil }

func TestMapRunStatusToItem_PreservesPending(t *testing.T) {
	if got := mapRunStatusToItem("Pending"); got != "pending" {
		t.Fatalf("Pending mapped to %q, want pending", got)
	}
}

func TestSyncJobProgress_PausedStillUpdatesCounts(t *testing.T) {
	ctx := context.Background()
	jobID := "job-1"
	runID := "run-1"
	repo := &pausedSyncRepo{
		job: &models.BackfillJob{
			ID:         jobID,
			Status:     "paused",
			TotalCount: 1,
		},
		items: []models.BackfillItem{
			{
				ID:            "item-1",
				JobID:         jobID,
				Status:        "running",
				PipelineRunID: &runID,
			},
		},
	}
	runRepo := &syncTestRunRepo{
		byID: map[string]*models.PipelineRun{
			runID: {
				ID:              runID,
				WorkflowName:    "wf-1",
				Status:          "Running",
				ArgoWorkflowUID: "uid-1",
			},
		},
	}
	pipeline := pipelineUC.New(nil, nil, nil, syncTestWorkflowClient{}, "default")
	pipeline.SetRunRepositories(nil, runRepo, nil)
	uc := New(repo, pipeline)

	if err := uc.syncJobProgress(ctx, jobID); err != nil {
		t.Fatalf("syncJobProgress: %v", err)
	}
	if repo.items[0].Status != "completed" {
		t.Fatalf("expected item completed, got %q", repo.items[0].Status)
	}
	if repo.job.CompletedCount != 1 {
		t.Fatalf("expected completedCount=1, got %d", repo.job.CompletedCount)
	}
	if repo.job.Status != "paused" {
		t.Fatalf("expected job status paused, got %q", repo.job.Status)
	}
}

func TestGetBatchNodeSummary_SyncsActiveRunsBeforeAggregating(t *testing.T) {
	ctx := context.Background()
	jobID := "job-1"
	runID := "run-1"
	repo := &pausedSyncRepo{
		job: &models.BackfillJob{
			ID:         jobID,
			Status:     "running",
			TotalCount: 1,
		},
		items: []models.BackfillItem{
			{
				ID:            "item-1",
				JobID:         jobID,
				AssetID:       "asset-1",
				Status:        "running",
				PipelineRunID: &runID,
			},
		},
	}
	runRepo := &syncTestRunRepo{
		byID: map[string]*models.PipelineRun{
			runID: {
				ID:              runID,
				WorkflowName:    "wf-1",
				Status:          "Running",
				ArgoWorkflowUID: "uid-1",
			},
		},
	}
	pipeline := pipelineUC.New(nil, nil, nil, syncTestWorkflowClient{}, "default")
	pipeline.SetRunRepositories(nil, runRepo, nil)
	uc := New(repo, pipeline)

	summary, err := uc.GetBatchNodeSummary(ctx, jobID)
	if err != nil {
		t.Fatalf("GetBatchNodeSummary: %v", err)
	}
	if summary.Subtasks.Completed != 1 || summary.Subtasks.Running != 0 {
		t.Fatalf("expected refreshed completed summary, got %+v", summary.Subtasks)
	}
	if repo.job.Status != "completed" {
		t.Fatalf("expected job completed after summary sync, got %q", repo.job.Status)
	}
}

func TestGetJob_ForcesProgressSyncDespiteRecentThrottle(t *testing.T) {
	ctx := context.Background()
	jobID := "job-1"
	runID := "run-1"
	repo := &pausedSyncRepo{
		job: &models.BackfillJob{
			ID:         jobID,
			Status:     "running",
			TotalCount: 1,
		},
		items: []models.BackfillItem{
			{
				ID:            "item-1",
				JobID:         jobID,
				AssetID:       "asset-1",
				Status:        "running",
				PipelineRunID: &runID,
			},
		},
	}
	runRepo := &syncTestRunRepo{
		byID: map[string]*models.PipelineRun{
			runID: {
				ID:              runID,
				WorkflowName:    "wf-1",
				Status:          "Succeeded",
				ArgoWorkflowUID: "uid-1",
			},
		},
	}
	pipeline := pipelineUC.New(nil, nil, nil, syncTestWorkflowClient{}, "default")
	pipeline.SetRunRepositories(nil, runRepo, nil)
	uc := New(repo, pipeline)
	uc.lastSync = map[string]time.Time{jobID: time.Now()}

	job, err := uc.GetJob(ctx, jobID)
	if err != nil {
		t.Fatalf("GetJob: %v", err)
	}
	if job == nil {
		t.Fatal("expected job")
	}
	if job.Status != "completed" {
		t.Fatalf("expected job completed despite throttle, got %q", job.Status)
	}
	if job.CompletedCount != 1 {
		t.Fatalf("expected completedCount=1, got %d", job.CompletedCount)
	}
}

func TestSyncJobProgress_RefreshesBatchScopedActiveRunSummaries(t *testing.T) {
	ctx := context.Background()
	jobID := "job-1"
	runID := "run-1"
	repo := &pausedSyncRepo{
		job: &models.BackfillJob{
			ID:         jobID,
			Status:     "running",
			TotalCount: 1,
		},
		items: []models.BackfillItem{
			{
				ID:            "item-1",
				JobID:         jobID,
				AssetID:       "asset-1",
				Status:        "running",
				PipelineRunID: &runID,
			},
		},
	}
	runRepo := &syncTestRunRepo{
		byID: map[string]*models.PipelineRun{
			runID: {
				ID:              runID,
				WorkflowName:    "wf-1",
				Status:          "Running",
				ArgoWorkflowUID: "uid-1",
			},
		},
		summaries: []models.PipelineRun{
			{
				ID:              runID,
				WorkflowName:    "wf-1",
				Status:          "Running",
				ArgoWorkflowUID: "uid-1",
			},
		},
	}
	pipeline := pipelineUC.New(nil, nil, nil, syncTestWorkflowClient{}, "default")
	pipeline.SetRunRepositories(nil, runRepo, nil)
	uc := New(repo, pipeline)

	if err := uc.syncJobProgressForce(ctx, jobID); err != nil {
		t.Fatalf("syncJobProgressForce: %v", err)
	}
	if runRepo.lastListFilter == nil || !runRepo.lastListFilter.RefreshActive {
		t.Fatalf("expected batch sync to request active refresh, got %+v", runRepo.lastListFilter)
	}
	if repo.items[0].Status != "completed" {
		t.Fatalf("expected item completed, got %q", repo.items[0].Status)
	}
	if repo.job.Status != "completed" {
		t.Fatalf("expected job completed, got %q", repo.job.Status)
	}
	if repo.job.CompletedCount != 1 {
		t.Fatalf("expected completedCount=1, got %d", repo.job.CompletedCount)
	}
}

func TestSyncJobProgress_DoesNotOverwriteConcurrentPause(t *testing.T) {
	ctx := context.Background()
	jobID := "job-1"
	runID := "run-1"
	var findCalls atomic.Int32
	repo := &pausedSyncRepo{
		job: &models.BackfillJob{
			ID:         jobID,
			Status:     "running",
			TotalCount: 1,
		},
		items: []models.BackfillItem{
			{
				ID:            "item-1",
				JobID:         jobID,
				Status:        "running",
				PipelineRunID: &runID,
			},
		},
	}
	repo.findJobByIDHook = func() {
		if findCalls.Add(1) == 2 {
			repo.job.Status = "paused"
		}
	}
	runRepo := &syncTestRunRepo{
		byID: map[string]*models.PipelineRun{
			runID: {
				ID:              runID,
				WorkflowName:    "wf-1",
				Status:          "Running",
				ArgoWorkflowUID: "uid-1",
			},
		},
	}
	pipeline := pipelineUC.New(nil, nil, nil, syncTestWorkflowClient{}, "default")
	pipeline.SetRunRepositories(nil, runRepo, nil)
	uc := New(repo, pipeline)

	if err := uc.syncJobProgress(ctx, jobID); err != nil {
		t.Fatalf("syncJobProgress: %v", err)
	}
	if repo.job.Status != "paused" {
		t.Fatalf("expected job status paused after concurrent pause, got %q", repo.job.Status)
	}
	if repo.job.CompletedCount != 1 {
		t.Fatalf("expected completedCount=1, got %d", repo.job.CompletedCount)
	}
}
