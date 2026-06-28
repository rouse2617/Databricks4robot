package pipeline

import (
	"context"
	"sync"
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

// cancelBatchRepo is a minimal BackfillRepository that records the final job
// status written by processBatchJob. All other interface methods are inherited
// from the embedded nil interface and must not be called by the cancelled path.
type cancelBatchRepo struct {
	repository.BackfillRepository
	mu            sync.Mutex
	finalStatus   string
	progressCalls int
}

func (m *cancelBatchRepo) UpdateJobStatus(context.Context, string, string) error { return nil }

func (m *cancelBatchRepo) UpdateItemStatus(context.Context, string, string, string, string) error {
	return nil
}

func (m *cancelBatchRepo) SummarizeItemStatuses(context.Context, string) (repository.BackfillItemStatusSummary, error) {
	return repository.BackfillItemStatusSummary{}, nil
}

func (m *cancelBatchRepo) UpdateJobProgress(_ context.Context, _ string, _, _ int, status string) error {
	m.mu.Lock()
	m.finalStatus = status
	m.progressCalls++
	m.mu.Unlock()
	return nil
}

// TestProcessBatchJobCancelledContextPreservesCancelledStatus verifies that a
// batch whose context is cancelled before processing neither submits runs (the
// nil template/run deps would panic on CreateRunByTemplateID) nor clobbers the
// cancelled status back to completed.
func TestProcessBatchJobCancelledContextPreservesCancelledStatus(t *testing.T) {
	repo := &cancelBatchRepo{}
	uc := New(&mockTemplateRepo{}, nil, nil, nil, "ns")
	uc.SetBackfillRepo(repo)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	items := []models.BackfillItem{
		{ID: "i1", AssetID: "a1"},
		{ID: "i2", AssetID: "a2"},
	}
	uc.processBatchJob(ctx, "job-1", "tmpl-1", "", 1, items, "owner", 4)

	if repo.progressCalls != 1 {
		t.Fatalf("UpdateJobProgress calls = %d, want 1", repo.progressCalls)
	}
	if repo.finalStatus != "cancelled" {
		t.Fatalf("final status = %q, want cancelled", repo.finalStatus)
	}
}

// TestCancelBatchTriggersRegisteredCancel verifies the cancel registry wiring
// used by StopBatchRuns to halt an in-flight submission goroutine.
func TestCancelBatchTriggersRegisteredCancel(t *testing.T) {
	uc := &Usecase{}
	ctx, cancel := context.WithCancel(context.Background())
	uc.registerBatchCancel("job-x", cancel)

	if !uc.cancelBatch("job-x") {
		t.Fatal("cancelBatch returned false for a registered job")
	}
	select {
	case <-ctx.Done():
	default:
		t.Fatal("expected context to be cancelled")
	}

	uc.unregisterBatchCancel("job-x")
	if uc.cancelBatch("job-x") {
		t.Fatal("cancelBatch returned true after unregister")
	}
}
