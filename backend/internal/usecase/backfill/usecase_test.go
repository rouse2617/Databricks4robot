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
	pipelineUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/pipeline"
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
func (m *mockBackfillRepo) ClaimJobNotification(_ context.Context, _ string) (bool, error) {
	return false, nil
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

func (m *mockBackfillRepo) ClaimNextItem(ctx context.Context, jobID string) (*models.BackfillItem, error) {
	return nil, nil
}

func (m *mockBackfillRepo) ResetStaleItems(ctx context.Context, leaseTimeoutSec int, maxAttempts int) (int, error) {
	return 0, nil
}

func (m *mockBackfillRepo) FindIncompleteJobs(ctx context.Context) ([]models.BackfillJob, error) {
	return nil, nil
}

func (m *mockBackfillRepo) FindActiveJobs(ctx context.Context, _ int) ([]models.BackfillJob, error) {
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
			// Mirror the postgres repo's nullIfEmpty: empty clears the column.
			if pipelineRunID != "" {
				m.items[i].PipelineRunID = &pipelineRunID
			} else {
				m.items[i].PipelineRunID = nil
			}
			if workflowName != "" {
				m.items[i].WorkflowName = &workflowName
			} else {
				m.items[i].WorkflowName = nil
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
func (m *mockBackfillRepo) FindItemsByJobIDWithStatuses(_ context.Context, jobID string, statuses []string) ([]models.BackfillItem, error) {
	out := []models.BackfillItem{}
	for _, item := range m.items {
		if item.JobID != jobID {
			continue
		}
		if len(statuses) > 0 && !containsString(statuses, item.Status) {
			continue
		}
		out = append(out, item)
	}
	return out, nil
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

func TestCreateBackfill_PersistsConfigSelectionInFilterJSON(t *testing.T) {
	repo := &trackingBackfillRepo{}
	uc := New(repo, nil)
	job, err := uc.CreateBackfill(context.Background(), "batch", "tpl-1", []string{
		"asset-1",
	}, CreateBackfillOptions{
		TargetID: "video-proc-dev",
		ConfigSelection: &pipelineUC.RuntimeConfigSelection{
			Mode:           "inline",
			FileName:       "runtime-config.yaml",
			Content:        "threshold: 0.8\n",
			MountPath:      "/workspace/configs",
			TargetFilename: "effective.yaml",
		},
	})
	if err != nil {
		t.Fatalf("CreateBackfill: %v", err)
	}
	if job.FilterJSON == nil {
		t.Fatal("expected filter json to be stored")
	}
	raw, ok := job.FilterJSON["configSelection"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected configSelection object, got %#v", job.FilterJSON["configSelection"])
	}
	if raw["mode"] != "inline" || raw["targetFilename"] != "effective.yaml" {
		t.Fatalf("unexpected configSelection payload %#v", raw)
	}
	if got := job.FilterJSON["targetId"]; got != "video-proc-dev" {
		t.Fatalf("expected targetId to be persisted, got %#v", got)
	}
}

func TestStringFromBackfillFilterAcceptsTargetAliases(t *testing.T) {
	values := map[string]interface{}{
		"target_id": " video-proc-dev ",
	}
	if got := stringFromBackfillFilter(values, "targetId", "target_id"); got != "video-proc-dev" {
		t.Fatalf("expected target alias to resolve, got %q", got)
	}
}

func TestCreateBackfill_PersistsTargetIDInFilterJSON(t *testing.T) {
	repo := &trackingBackfillRepo{}
	uc := New(repo, nil)
	job, err := uc.CreateBackfill(context.Background(), "batch", "tpl-1", []string{
		"asset-1",
	}, CreateBackfillOptions{
		TargetID: "video-proc-dev",
	})
	if err != nil {
		t.Fatalf("CreateBackfill: %v", err)
	}
	if job.FilterJSON == nil {
		t.Fatal("expected filter json to be stored")
	}
	if job.FilterJSON["targetId"] != "video-proc-dev" {
		t.Fatalf("targetId = %#v, want video-proc-dev", job.FilterJSON["targetId"])
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

// CYB-3491: in-flight runs usually have NO projected asset-node rows yet
// (node projection is only near-real-time). Such unprojected in-flight runs
// must be counted as the frontier node's Running so the node overview agrees
// with the subtask "运行中" count — not dumped into Pending (the old behaviour
// that produced "节点运行中=0 / 节点排队=N" while N subtasks showed running).
func TestGetBatchNodeSummary_AttributesInFlightRunsToFrontierRunning(t *testing.T) {
	repo := &trackingBackfillRepo{
		job: &models.BackfillJob{
			ID:         "job-1",
			TemplateID: "tpl-1",
			TotalCount: 100,
			Status:     "running",
		},
		items: make([]models.BackfillItem, 100),
		aggregates: []repository.BatchNodeStatusAggregate{
			{PipelineNodeID: "step-1", DisplayName: "step-1", Status: "Succeeded", Count: 60},
		},
		runsWithNodeRows: 60,
	}
	for i := range repo.items {
		status := "completed" // first 60 have projected Succeeded node rows
		if i >= 60 {
			status = "submitted" // 40 in-flight, no node rows yet
		}
		repo.items[i] = models.BackfillItem{
			ID:      fmt.Sprintf("item-%03d", i),
			JobID:   "job-1",
			AssetID: fmt.Sprintf("asset-%03d", i),
			Status:  status,
		}
	}
	uc := New(repo, nil)

	summary, err := uc.GetBatchNodeSummary(context.Background(), "job-1")
	if err != nil {
		t.Fatalf("GetBatchNodeSummary: %v", err)
	}
	if summary.Subtasks.Running != 40 || summary.Subtasks.Completed != 60 {
		t.Fatalf("unexpected subtask counts: %+v", summary.Subtasks)
	}
	if len(summary.Nodes) != 1 {
		t.Fatalf("expected one node, got %d", len(summary.Nodes))
	}
	node := summary.Nodes[0]
	if node.Counts["Succeeded"] != 60 || node.Counts["Running"] != 40 || node.Counts["Pending"] != 0 {
		t.Fatalf("want Succeeded=60 Running=40 Pending=0 (in-flight → frontier running), got %+v", node.Counts)
	}
}

// CYB-3491: the unprojected fill must split by subtask truth — never-started
// items ('pending') are queued; in-flight items are running.
func TestGetBatchNodeSummary_SplitsUnprojectedFillBySubtaskStatus(t *testing.T) {
	repo := &trackingBackfillRepo{
		job: &models.BackfillJob{
			ID:         "job-1",
			TemplateID: "tpl-1",
			TotalCount: 100,
			Status:     "running",
		},
		items: make([]models.BackfillItem, 100),
		aggregates: []repository.BatchNodeStatusAggregate{
			{PipelineNodeID: "step-1", DisplayName: "step-1", Status: "Succeeded", Count: 20},
		},
		runsWithNodeRows: 20,
	}
	for i := range repo.items {
		status := "pending" // 50 never started
		switch {
		case i < 20:
			status = "completed"
		case i < 50:
			status = "submitted" // 30 in-flight
		}
		repo.items[i] = models.BackfillItem{
			ID:      fmt.Sprintf("item-%03d", i),
			JobID:   "job-1",
			AssetID: fmt.Sprintf("asset-%03d", i),
			Status:  status,
		}
	}
	uc := New(repo, nil)

	summary, err := uc.GetBatchNodeSummary(context.Background(), "job-1")
	if err != nil {
		t.Fatalf("GetBatchNodeSummary: %v", err)
	}
	node := summary.Nodes[0]
	if node.Counts["Succeeded"] != 20 || node.Counts["Running"] != 30 || node.Counts["Pending"] != 50 {
		t.Fatalf("want Succeeded=20 Running=30 Pending=50, got %+v", node.Counts)
	}
}

func TestBatchNodeOrderFromPipeline_NormalizesStepIDs(t *testing.T) {
	order := batchNodeOrderFromPipeline(map[string]interface{}{
		"nodes": []interface{}{
			map[string]interface{}{"id": "head_tracking"},
			map[string]interface{}{"id": "hand_detection"},
			map[string]interface{}{"id": "hand_tracking"},
			map[string]interface{}{"id": "find_tony_stats"},
			map[string]interface{}{"id": "ss_delivery_lerobot"},
		},
	})
	if got := order[normalizeBatchPipelineNodeID("step-head-tracking")]; got != 1 {
		t.Fatalf("step-head-tracking order = %d, want 1", got)
	}
	if got := order[normalizeBatchPipelineNodeID("step-find-tony-stats")]; got != 4 {
		t.Fatalf("step-find-tony-stats order = %d, want 4", got)
	}
}

func TestBatchNodeOrderFromPipeline_PrefersDagEdges(t *testing.T) {
	order := batchNodeOrderFromPipeline(map[string]interface{}{
		"nodes": []interface{}{
			map[string]interface{}{"id": "find_tony_stats"},
			map[string]interface{}{"id": "hand_detection"},
			map[string]interface{}{"id": "head_tracking"},
			map[string]interface{}{"id": "ss_delivery_lerobot"},
			map[string]interface{}{"id": "hand_tracking"},
		},
		"edges": []interface{}{
			map[string]interface{}{"source": "head_tracking", "target": "hand_detection"},
			map[string]interface{}{"source": "hand_detection", "target": "hand_tracking"},
			map[string]interface{}{"source": "hand_tracking", "target": "find_tony_stats"},
			map[string]interface{}{"source": "find_tony_stats", "target": "ss_delivery_lerobot"},
		},
	})
	if got := order[normalizeBatchPipelineNodeID("step-head-tracking")]; got != 1 {
		t.Fatalf("step-head-tracking order = %d, want 1", got)
	}
	if got := order[normalizeBatchPipelineNodeID("step-ss-delivery-lerobot")]; got != 5 {
		t.Fatalf("step-ss-delivery-lerobot order = %d, want 5", got)
	}
}

func TestBatchNodeOrderFromPipeline_DualFormat(t *testing.T) {
	// Readable pod names (CYB-3076): nodes carry uuid ids + component names.
	// Both the new readable template name and the legacy step-node-<uuid> name
	// must resolve to the same DAG order.
	order := batchNodeOrderFromPipeline(map[string]interface{}{
		"nodes": []interface{}{
			map[string]interface{}{
				"id":        "node-aaaaaaaa-0000-0000-0000-000000000001",
				"component": map[string]interface{}{"name": "head-track"},
			},
			map[string]interface{}{
				"id":        "node-bbbbbbbb-0000-0000-0000-000000000002",
				"component": map[string]interface{}{"name": "transcode"},
			},
		},
		"edges": []interface{}{
			map[string]interface{}{
				"source": "node-aaaaaaaa-0000-0000-0000-000000000001",
				"target": "node-bbbbbbbb-0000-0000-0000-000000000002",
			},
		},
	})

	// New readable format.
	if got := order[normalizeBatchPipelineNodeID("step-head-track")]; got != 1 {
		t.Fatalf("new-format step-head-track order = %d, want 1", got)
	}
	if got := order[normalizeBatchPipelineNodeID("step-transcode")]; got != 2 {
		t.Fatalf("new-format step-transcode order = %d, want 2", got)
	}
	// Legacy format for historical runs (stored as step-node-<uuid>).
	if got := order[normalizeBatchPipelineNodeID("step-node-aaaaaaaa-0000-0000-0000-000000000001")]; got != 1 {
		t.Fatalf("legacy head-track order = %d, want 1", got)
	}
	if got := order[normalizeBatchPipelineNodeID("step-node-bbbbbbbb-0000-0000-0000-000000000002")]; got != 2 {
		t.Fatalf("legacy transcode order = %d, want 2", got)
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
func (r *trackingBackfillRepo) ClaimJobNotification(_ context.Context, _ string) (bool, error) {
	return false, nil
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
		// Mirror the production SQL (SummarizeItemStatuses): 'submitted' and
		// 'awaiting_result' are in-flight and count as Running, not Pending.
		switch item.Status {
		case "completed":
			summary.Completed++
		case "failed", "cancelled":
			summary.Failed++
		case "running", "awaiting_result", "submitted":
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

func (r *trackingBackfillRepo) ClaimNextItem(ctx context.Context, jobID string) (*models.BackfillItem, error) {
	return nil, nil
}

func (r *trackingBackfillRepo) ResetStaleItems(ctx context.Context, leaseTimeoutSec int, maxAttempts int) (int, error) {
	return 0, nil
}

func (r *trackingBackfillRepo) FindIncompleteJobs(ctx context.Context) ([]models.BackfillJob, error) {
	return nil, nil
}

func (r *trackingBackfillRepo) FindActiveJobs(ctx context.Context, _ int) ([]models.BackfillJob, error) {
	return nil, nil
}

func TestPauseJob_SetsPausedStatus(t *testing.T) {
	repo := &mockBackfillRepo{
		jobs: map[string]*models.BackfillJob{
			"job-1": {ID: "job-1", Status: "running"},
		},
	}
	uc := New(repo, nil)
	result, err := uc.PauseJob(context.Background(), "job-1", PauseJobOptions{})
	if err != nil {
		t.Fatalf("PauseJob: %v", err)
	}
	if result.Status != "paused" {
		t.Fatalf("unexpected status: %q", result.Status)
	}
	if repo.jobs["job-1"].Status != "paused" {
		t.Fatalf("job status not updated: %q", repo.jobs["job-1"].Status)
	}
}

func TestPauseJob_StopRunning_ResetsStoppedItemsToPending(t *testing.T) {
	// When pausing with StopRunning, stopping the workflow removes it from Argo.
	// The item must be reset to "pending" so a later resume re-submits it; if it
	// stayed "running" it would be invisible to ResumeJob's ClaimNextItem and the
	// executeItem dedup guard would skip redeploy — stranding it forever.
	runID := "run-1"
	repo := &mockBackfillRepo{
		jobs: map[string]*models.BackfillJob{
			"job-1": {ID: "job-1", Status: "running", TotalCount: 1},
		},
		items: []models.BackfillItem{
			{ID: "item-1", JobID: "job-1", AssetID: "a1", Status: "running", PipelineRunID: &runID},
		},
	}
	runRepo := &syncTestRunRepo{byID: map[string]*models.PipelineRun{
		runID: {ID: runID, WorkflowName: "wf-1", Status: "Running", ArgoNamespace: "default"},
	}}
	pipeline := pipelineUC.New(nil, nil, nil, syncTestWorkflowClient{}, "default")
	pipeline.SetRunRepositories(nil, runRepo, nil)
	uc := New(repo, pipeline)
	// Run the async stop-cleanup inline so its effects are deterministic here.
	uc.spawn = func(f func()) { f() }

	result, err := uc.PauseJob(context.Background(), "job-1", PauseJobOptions{StopRunning: true})
	if err != nil {
		t.Fatalf("PauseJob: %v", err)
	}
	if result.StoppedCount != 1 {
		t.Fatalf("StoppedCount = %d, want 1", result.StoppedCount)
	}
	if repo.items[0].Status != "pending" {
		t.Fatalf("stopped item status = %q, want pending (so resume re-runs it)", repo.items[0].Status)
	}
}

// CYB-3572: pausing with StopRunning must NOT stop the runs inline — that
// serialized N per-run Stop calls in the request and hit the 60s gateway
// timeout (504). The job is marked paused and the count is reported
// immediately; the actual stops run in the background.
func TestPauseJob_StopRunning_DefersStopsToBackground(t *testing.T) {
	runID := "run-1"
	repo := &mockBackfillRepo{
		jobs: map[string]*models.BackfillJob{
			"job-1": {ID: "job-1", Status: "running", TotalCount: 1},
		},
		items: []models.BackfillItem{
			{ID: "item-1", JobID: "job-1", AssetID: "a1", Status: "running", PipelineRunID: &runID},
		},
	}
	runRepo := &syncTestRunRepo{byID: map[string]*models.PipelineRun{
		runID: {ID: runID, WorkflowName: "wf-1", Status: "Running", ArgoNamespace: "default"},
	}}
	pipeline := pipelineUC.New(nil, nil, nil, syncTestWorkflowClient{}, "default")
	pipeline.SetRunRepositories(nil, runRepo, nil)
	uc := New(repo, pipeline)
	// Capture the background job instead of running it — proves PauseJob returns
	// without doing the stop work inline.
	var deferred func()
	uc.spawn = func(f func()) { deferred = f }

	result, err := uc.PauseJob(context.Background(), "job-1", PauseJobOptions{StopRunning: true})
	if err != nil {
		t.Fatalf("PauseJob: %v", err)
	}
	if result.Status != "paused" {
		t.Fatalf("status = %q, want paused", result.Status)
	}
	if result.StoppedCount != 1 {
		t.Fatalf("StoppedCount = %d, want 1 (reported immediately)", result.StoppedCount)
	}
	// The per-run stop work was scheduled to the background rather than run
	// inline — that is the fix for the 60s-gateway-timeout 504. Running the
	// captured job then performs the reset without error.
	if deferred == nil {
		t.Fatal("expected the stop-cleanup to be scheduled to the background, not run inline")
	}
	deferred()
}

// CYB-3491 P2 — the claim/reaper/worker-pool queue (and its P0 pool-recovery
// stopgap) is deleted; dispatch is owned by the submitter (see submitter.go
// and submitter_test.go).
