package backfill

import (
	"context"
	"io"
	"sync"
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

	// notifyMu/notified give ClaimJobNotification real exactly-once claim
	// semantics (unlike the other stub mocks in this package, which always
	// return false) so tests can exercise the CYB-3071 notification path.
	notifyMu sync.Mutex
	notified bool
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
func (r *pausedSyncRepo) ClaimJobNotification(_ context.Context, _ string) (bool, error) {
	r.notifyMu.Lock()
	defer r.notifyMu.Unlock()
	if r.notified {
		return false, nil
	}
	r.notified = true
	return true, nil
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

func (r *pausedSyncRepo) ClaimNextItem(ctx context.Context, jobID string) (*models.BackfillItem, error) {
	return nil, nil
}

func (r *pausedSyncRepo) ResetStaleItems(ctx context.Context, leaseTimeoutSec int, maxAttempts int) (int, error) {
	return 0, nil
}

func (r *pausedSyncRepo) FindIncompleteJobs(ctx context.Context) ([]models.BackfillJob, error) {
	return nil, nil
}

func (r *pausedSyncRepo) FindActiveJobs(ctx context.Context, _ int) ([]models.BackfillJob, error) {
	if r.job != nil && r.job.Status != "completed" && r.job.Status != "failed" && r.job.Status != "paused" {
		return []models.BackfillJob{*r.job}, nil
	}
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
func (r *pausedSyncRepo) CountItemsByStatus(_ context.Context, jobID, status string) (int, error) {
	n := 0
	for i := range r.items {
		if r.items[i].JobID == jobID && r.items[i].Status == status {
			n++
		}
	}
	return n, nil
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

func TestMapRunStatusToItem_SubmittedSemantics(t *testing.T) {
	// CYB-3491: any in-flight run WITH a persisted Argo UID projects the item
	// to "submitted"; a uid-less run is an unsubmitted placeholder and yields
	// NO transition (empty) — never "failed", never in-flight (invariant ②).
	if got := mapRunStatusToItem("Pending", "uid-1"); got != "submitted" {
		t.Fatalf("Pending+uid mapped to %q, want submitted", got)
	}
	if got := mapRunStatusToItem("Running", "uid-1"); got != "submitted" {
		t.Fatalf("Running+uid mapped to %q, want submitted", got)
	}
	if got := mapRunStatusToItem("Pending", ""); got != "" {
		t.Fatalf("placeholder Pending mapped to %q, want no transition", got)
	}
	if got := mapRunStatusToItem("Succeeded", ""); got != "completed" {
		t.Fatalf("Succeeded mapped to %q, want completed (uid irrelevant at terminal)", got)
	}
	if got := mapRunStatusToItem("Failed", "uid-1"); got != "failed" {
		t.Fatalf("Failed mapped to %q, want failed", got)
	}
}

func TestRunAlreadySubmitted(t *testing.T) {
	now := time.Now().UTC()
	cases := []struct {
		name string
		run  *models.PipelineRun
		want bool
	}{
		{"nil run → deploy", nil, false},
		{"placeholder pending (no uid, no startedAt) → deploy", &models.PipelineRun{Status: "Pending"}, false},
		{"submitted queued (argo uid, still Pending) → skip", &models.PipelineRun{Status: "Pending", ArgoWorkflowUID: "uid-1"}, true},
		{"submitted running (startedAt set) → skip", &models.PipelineRun{Status: "Running", StartedAt: &now}, true},
		{"terminal succeeded → deploy (retry allowed)", &models.PipelineRun{Status: "Succeeded", ArgoWorkflowUID: "uid-1"}, false},
		{"terminal failed → deploy (retry allowed)", &models.PipelineRun{Status: "Failed", StartedAt: &now}, false},
	}
	for _, tc := range cases {
		if got := runAlreadySubmitted(tc.run); got != tc.want {
			t.Errorf("%s: runAlreadySubmitted = %v, want %v", tc.name, got, tc.want)
		}
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

// CYB-3490: node-summary is a pure ledger read — it must NOT reconcile or
// sync on the request path, even when the run ledger is ahead of the item.
func TestGetBatchNodeSummary_PureRead_NoSyncSideEffects(t *testing.T) {
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

	summary, err := uc.GetBatchNodeSummary(ctx, jobID)
	if err != nil {
		t.Fatalf("GetBatchNodeSummary: %v", err)
	}
	// The ledger says running — a pure read reports exactly that, without
	// flipping the item/job as a side effect.
	if summary.Subtasks.Running != 1 || summary.Subtasks.Completed != 0 {
		t.Fatalf("expected as-is ledger summary (running=1), got %+v", summary.Subtasks)
	}
	if repo.job.Status != "running" {
		t.Fatalf("expected job untouched by read, got %q", repo.job.Status)
	}
	if runRepo.lastListFilter != nil {
		t.Fatalf("expected no batch run listing on read path, got %+v", runRepo.lastListFilter)
	}
}

// CYB-3490: GetJob is a pure ledger read; convergence is owned by the
// background job reconciler. Same fixture, two phases: the read must not
// mutate anything, then one reconciler pass converges it.
func TestGetJob_PureRead_ReconcilerOwnsConvergence(t *testing.T) {
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
		summaries: []models.PipelineRun{
			{
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

	// Phase 1: pure read — run ledger is already Succeeded, but the read
	// reports the item ledger as-is and performs no sync.
	job, err := uc.GetJob(ctx, jobID)
	if err != nil {
		t.Fatalf("GetJob: %v", err)
	}
	if job == nil {
		t.Fatal("expected job")
	}
	if job.Status != "running" || job.CompletedCount != 0 {
		t.Fatalf("expected untouched ledger view, got status=%q completed=%d", job.Status, job.CompletedCount)
	}

	// Phase 2: one background reconciler pass converges the same fixture.
	uc.reconcileActiveJobs(ctx, 10)
	if repo.job.Status != "completed" {
		t.Fatalf("expected job completed after reconciler pass, got %q", repo.job.Status)
	}
	if repo.job.CompletedCount != 1 {
		t.Fatalf("expected completedCount=1 after reconciler pass, got %d", repo.job.CompletedCount)
	}
}

// CYB-3490: the background sync maps items from persisted (already
// watcher-projected) run summaries in one batch-scoped query — it does not
// request an Argo refresh from the list layer.
func TestSyncJobProgress_MapsFromPersistedBatchSummaries(t *testing.T) {
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
		summaries: []models.PipelineRun{
			{
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

	if err := uc.syncJobProgressForce(ctx, jobID); err != nil {
		t.Fatalf("syncJobProgressForce: %v", err)
	}
	// CYB-3490: the sync maps from persisted run summaries (batch-scoped
	// single query) — it does not ask the list layer to refresh from Argo.
	if runRepo.lastListFilter == nil || runRepo.lastListFilter.BatchJobID != jobID {
		t.Fatalf("expected batch-scoped summary listing, got %+v", runRepo.lastListFilter)
	}
	if runRepo.lastListFilter.RefreshActive {
		t.Fatalf("expected no refresh-active request from sync, got %+v", runRepo.lastListFilter)
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
