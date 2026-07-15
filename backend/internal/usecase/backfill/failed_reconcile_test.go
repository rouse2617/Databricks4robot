package backfill

import (
	"context"
	"fmt"
	"testing"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	pipelineUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/pipeline"
)

// phaseWorkflowClient reports a fixed Argo phase for every workflow so tests can
// drive reconciliation of a persisted-`failed` run toward runtime truth. It
// embeds syncTestWorkflowClient (sync_progress_test.go) and overrides only the
// phase-returning methods.
type phaseWorkflowClient struct {
	syncTestWorkflowClient
	phase wfv1.WorkflowPhase
}

func (c phaseWorkflowClient) GetWorkflow(context.Context, string, string) (*wfv1.Workflow, error) {
	return &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "wf-1", UID: "uid-1"},
		Status:     wfv1.WorkflowStatus{Phase: c.phase},
	}, nil
}

func (c phaseWorkflowClient) GetWorkflowStatus(context.Context, string, string) (wfv1.WorkflowPhase, error) {
	return c.phase, nil
}

func newFailedReconcileUC(repo *pausedSyncRepo, runRepo *syncTestRunRepo, phase wfv1.WorkflowPhase) *Usecase {
	pipeline := pipelineUC.New(nil, nil, nil, phaseWorkflowClient{phase: phase}, "default")
	pipeline.SetRunRepositories(nil, runRepo, nil)
	return New(repo, pipeline)
}

// A batch item stuck at `failed` whose Argo workflow actually Succeeded must be
// healed back to `completed` on sync — otherwise a transient/misclassified
// failure inflates failedCount forever (CYB-3474). TemplateID is left empty so
// the parent-run upsert short-circuits before touching the nil template repo.
func TestSyncJobProgress_HealsMisclassifiedFailedItem(t *testing.T) {
	ctx := context.Background()
	jobID, runID := "job-1", "run-1"
	repo := &pausedSyncRepo{
		job:   &models.BackfillJob{ID: jobID, Status: "running", TotalCount: 1},
		items: []models.BackfillItem{{ID: "item-1", JobID: jobID, Status: "failed", PipelineRunID: &runID}},
	}
	runRepo := &syncTestRunRepo{byID: map[string]*models.PipelineRun{
		runID: {ID: runID, WorkflowName: "wf-1", Status: "Failed"},
	}}
	uc := newFailedReconcileUC(repo, runRepo, wfv1.WorkflowSucceeded)

	if err := uc.syncJobProgressForce(ctx, jobID); err != nil {
		t.Fatalf("syncJobProgressForce: %v", err)
	}
	if repo.items[0].Status != "completed" {
		t.Fatalf("misclassified failed item should heal to completed, got %q", repo.items[0].Status)
	}
}

// A genuinely-failed item (Argo workflow really Failed) must NOT be flipped away
// from `failed` by the reconcile pass.
func TestSyncJobProgress_PreservesGenuineFailedItem(t *testing.T) {
	ctx := context.Background()
	jobID, runID := "job-1", "run-1"
	repo := &pausedSyncRepo{
		job:   &models.BackfillJob{ID: jobID, Status: "running", TotalCount: 1},
		items: []models.BackfillItem{{ID: "item-1", JobID: jobID, Status: "failed", PipelineRunID: &runID}},
	}
	runRepo := &syncTestRunRepo{byID: map[string]*models.PipelineRun{
		runID: {ID: runID, WorkflowName: "wf-1", Status: "Failed"},
	}}
	uc := newFailedReconcileUC(repo, runRepo, wfv1.WorkflowFailed)

	if err := uc.syncJobProgressForce(ctx, jobID); err != nil {
		t.Fatalf("syncJobProgressForce: %v", err)
	}
	if repo.items[0].Status != "failed" {
		t.Fatalf("genuine failed item must stay failed, got %q", repo.items[0].Status)
	}
}

// A `failed` item whose workflow is actually still Running must move back to
// `running` so it re-enters the normal working set.
func TestSyncJobProgress_MovesFailedItemBackToRunningWhenWorkflowRunning(t *testing.T) {
	ctx := context.Background()
	jobID, runID := "job-1", "run-1"
	repo := &pausedSyncRepo{
		job:   &models.BackfillJob{ID: jobID, Status: "running", TotalCount: 1},
		items: []models.BackfillItem{{ID: "item-1", JobID: jobID, Status: "failed", PipelineRunID: &runID}},
	}
	runRepo := &syncTestRunRepo{byID: map[string]*models.PipelineRun{
		runID: {ID: runID, WorkflowName: "wf-1", Status: "Failed"},
	}}
	uc := newFailedReconcileUC(repo, runRepo, wfv1.WorkflowRunning)

	if err := uc.syncJobProgressForce(ctx, jobID); err != nil {
		t.Fatalf("syncJobProgressForce: %v", err)
	}
	if repo.items[0].Status != "running" {
		t.Fatalf("failed item with running workflow should become running, got %q", repo.items[0].Status)
	}
}

// The failed-item reconcile pass is bounded to maxFailedReconcilePerSync per
// sync; a batch with more stuck-failed items converges over successive syncs.
func TestSyncJobProgress_FailedReconcileIsBoundedAndConverges(t *testing.T) {
	ctx := context.Background()
	jobID := "job-1"
	total := maxFailedReconcilePerSync + 10
	items := make([]models.BackfillItem, 0, total)
	byID := make(map[string]*models.PipelineRun, total)
	for i := 0; i < total; i++ {
		runID := fmt.Sprintf("run-%d", i)
		rid := runID
		items = append(items, models.BackfillItem{
			ID:            fmt.Sprintf("item-%d", i),
			JobID:         jobID,
			Status:        "failed",
			PipelineRunID: &rid,
		})
		byID[runID] = &models.PipelineRun{ID: runID, WorkflowName: "wf-1", Status: "Failed"}
	}
	repo := &pausedSyncRepo{
		job:   &models.BackfillJob{ID: jobID, Status: "running", TotalCount: total},
		items: items,
	}
	runRepo := &syncTestRunRepo{byID: byID}
	uc := newFailedReconcileUC(repo, runRepo, wfv1.WorkflowSucceeded)

	countStatus := func(status string) int {
		n := 0
		for _, it := range repo.items {
			if it.Status == status {
				n++
			}
		}
		return n
	}

	if err := uc.syncJobProgressForce(ctx, jobID); err != nil {
		t.Fatalf("syncJobProgressForce (pass 1): %v", err)
	}
	if got := countStatus("completed"); got != maxFailedReconcilePerSync {
		t.Fatalf("pass 1 should heal exactly %d items, healed %d", maxFailedReconcilePerSync, got)
	}
	if got := countStatus("failed"); got != 10 {
		t.Fatalf("pass 1 should leave 10 failed for next sync, left %d", got)
	}

	if err := uc.syncJobProgressForce(ctx, jobID); err != nil {
		t.Fatalf("syncJobProgressForce (pass 2): %v", err)
	}
	if got := countStatus("completed"); got != total {
		t.Fatalf("pass 2 should heal remaining items (want %d completed), got %d", total, got)
	}
}
