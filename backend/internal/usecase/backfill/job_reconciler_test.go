package backfill

import (
	"context"
	"testing"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// The reconcile backstop (CYB-3078) must finalize + notify a batch whose
// children finished in the background — with no page open and no hook firing.
func TestReconcileActiveJobs_FinalizesAndNotifiesWithoutPageOpen(t *testing.T) {
	ctx := context.Background()
	jobID := "job-reconcile"
	repo := &pausedSyncRepo{
		job: &models.BackfillJob{
			ID:         jobID,
			Name:       "bg-batch",
			Status:     "running",
			TotalCount: 2,
			CreatedAt:  time.Now().Add(-time.Minute),
		},
		items: []models.BackfillItem{
			{ID: "item-1", JobID: jobID, Status: "completed"},
			{ID: "item-2", JobID: jobID, Status: "completed"},
		},
	}
	notifier := &fakeNotifier{}
	uc := New(repo, nil)
	uc.SetNotifier(notifier, "https://dev.example.com")

	// Simulate one backstop tick — no GetJob / GetBatchNodeSummary read involved.
	uc.reconcileActiveJobs(ctx, 100)

	if repo.job.Status != "completed" {
		t.Fatalf("expected job finalized to completed, got %q", repo.job.Status)
	}
	if got := notifier.callCount(); got != 1 {
		t.Fatalf("expected exactly 1 notification from the backstop, got %d", got)
	}
}

// A second reconcile pass must not re-notify (idempotent; once-guarded, and the
// finalized job is no longer returned by FindActiveJobs).
func TestReconcileActiveJobs_DoesNotReNotify(t *testing.T) {
	ctx := context.Background()
	jobID := "job-reconcile-2"
	repo := &pausedSyncRepo{
		job: &models.BackfillJob{ID: jobID, Name: "bg", Status: "running", TotalCount: 1},
		items: []models.BackfillItem{
			{ID: "i1", JobID: jobID, Status: "completed"},
		},
	}
	notifier := &fakeNotifier{}
	uc := New(repo, nil)
	uc.SetNotifier(notifier, "")

	uc.reconcileActiveJobs(ctx, 100)
	uc.reconcileActiveJobs(ctx, 100)

	if got := notifier.callCount(); got != 1 {
		t.Fatalf("expected exactly 1 notification across two passes, got %d", got)
	}
}

// SyncJob (the webhook fast-path wrapper) finalizes + notifies once.
func TestSyncJob_FinalizesAndNotifies(t *testing.T) {
	ctx := context.Background()
	jobID := "job-sync"
	repo := &pausedSyncRepo{
		job: &models.BackfillJob{ID: jobID, Name: "b", Status: "running", TotalCount: 1},
		items: []models.BackfillItem{
			{ID: "i1", JobID: jobID, Status: "completed"},
		},
	}
	notifier := &fakeNotifier{}
	uc := New(repo, nil)
	uc.SetNotifier(notifier, "")

	if err := uc.SyncJob(ctx, jobID); err != nil {
		t.Fatalf("SyncJob: %v", err)
	}
	if repo.job.Status != "completed" {
		t.Fatalf("expected completed, got %q", repo.job.Status)
	}
	if got := notifier.callCount(); got != 1 {
		t.Fatalf("expected 1 notification, got %d", got)
	}
}
