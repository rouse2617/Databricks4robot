package backfill

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

// fakeNotifier records every SendText call so tests can assert on how many
// times, and with what text, a notification would have gone out.
type fakeNotifier struct {
	mu    sync.Mutex
	calls []string
}

func (f *fakeNotifier) SendText(_ context.Context, text string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, text)
	return nil
}

func (f *fakeNotifier) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

func (f *fakeNotifier) lastCall() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.calls) == 0 {
		return ""
	}
	return f.calls[len(f.calls)-1]
}

func TestSyncJobProgress_NotifiesOnceWhenJobCompletes(t *testing.T) {
	ctx := context.Background()
	jobID := "job-1"
	repo := &pausedSyncRepo{
		job: &models.BackfillJob{
			ID:         jobID,
			Name:       "my-batch",
			Status:     "running",
			TotalCount: 2,
		},
		items: []models.BackfillItem{
			{ID: "item-1", JobID: jobID, Status: "completed"},
			{ID: "item-2", JobID: jobID, Status: "completed"},
		},
	}
	notifier := &fakeNotifier{}
	uc := New(repo, nil)
	uc.SetNotifier(notifier, "https://dev.example.com")

	if err := uc.syncJobProgressForce(ctx, jobID); err != nil {
		t.Fatalf("syncJobProgressForce: %v", err)
	}

	if repo.job.Status != "completed" {
		t.Fatalf("expected job status completed, got %q", repo.job.Status)
	}
	if got := notifier.callCount(); got != 1 {
		t.Fatalf("expected exactly 1 notification, got %d", got)
	}
	text := notifier.lastCall()
	for _, want := range []string{"my-batch", "completed", "总数：2", "成功：2", "失败：0", "https://dev.example.com/pipeline/batch/job-1"} {
		if !strings.Contains(text, want) {
			t.Errorf("notification text missing %q; got: %s", want, text)
		}
	}
}

func TestSyncJobProgress_NotifiesOnceWhenJobFails(t *testing.T) {
	ctx := context.Background()
	jobID := "job-2"
	repo := &pausedSyncRepo{
		job: &models.BackfillJob{
			ID:         jobID,
			Name:       "flaky-batch",
			Status:     "running",
			TotalCount: 3,
		},
		items: []models.BackfillItem{
			{ID: "item-1", JobID: jobID, Status: "completed"},
			{ID: "item-2", JobID: jobID, Status: "failed"},
			{ID: "item-3", JobID: jobID, Status: "failed"},
		},
	}
	notifier := &fakeNotifier{}
	uc := New(repo, nil)
	uc.SetNotifier(notifier, "")

	if err := uc.syncJobProgressForce(ctx, jobID); err != nil {
		t.Fatalf("syncJobProgressForce: %v", err)
	}

	if repo.job.Status != "failed" {
		t.Fatalf("expected job status failed, got %q", repo.job.Status)
	}
	if got := notifier.callCount(); got != 1 {
		t.Fatalf("expected exactly 1 notification, got %d", got)
	}
	text := notifier.lastCall()
	for _, want := range []string{"flaky-batch", "总数：3", "成功：1", "失败：2"} {
		if !strings.Contains(text, want) {
			t.Errorf("notification text missing %q; got: %s", want, text)
		}
	}
	if strings.Contains(text, "链接") {
		t.Errorf("expected no link when frontendBaseURL is empty; got: %s", text)
	}
}

func TestSyncJobProgress_NoNotifierConfigured_DoesNotPanicOrSend(t *testing.T) {
	ctx := context.Background()
	jobID := "job-3"
	repo := &pausedSyncRepo{
		job: &models.BackfillJob{ID: jobID, Status: "running", TotalCount: 1},
		items: []models.BackfillItem{
			{ID: "item-1", JobID: jobID, Status: "completed"},
		},
	}
	uc := New(repo, nil) // SetNotifier never called; notifier stays nil

	if err := uc.syncJobProgressForce(ctx, jobID); err != nil {
		t.Fatalf("syncJobProgressForce: %v", err)
	}
	if repo.job.Status != "completed" {
		t.Fatalf("expected job status completed, got %q", repo.job.Status)
	}
	// No assertion beyond "did not panic" — there is nothing to observe
	// without a notifier, which is the point of the nil-safety guard.
}

func TestNotifyJobTerminalIfNeeded_AlreadyTerminalJobDoesNotReNotify(t *testing.T) {
	ctx := context.Background()
	repo := &pausedSyncRepo{}
	notifier := &fakeNotifier{}
	uc := New(repo, nil)
	uc.SetNotifier(notifier, "")

	previouslyCompleted := &models.BackfillJob{ID: "job-4", Status: "completed", TotalCount: 1}
	uc.notifyJobTerminalIfNeeded(ctx, previouslyCompleted, "completed", repository.BackfillItemStatusSummary{Completed: 1})

	if got := notifier.callCount(); got != 0 {
		t.Fatalf("expected no notification for a job that was already terminal, got %d", got)
	}
}

func TestNotifyJobTerminalIfNeeded_ConcurrentCallersNotifyExactlyOnce(t *testing.T) {
	ctx := context.Background()
	repo := &pausedSyncRepo{}
	notifier := &fakeNotifier{}
	uc := New(repo, nil)
	uc.SetNotifier(notifier, "")

	// Simulates multiple backend instances (Cloud Run MAX_INSTANCES=5)
	// observing the same running -> completed transition at nearly the same
	// time. Only one should win the ClaimJobNotification race and send.
	previousJob := &models.BackfillJob{ID: "job-5", Name: "racy-batch", Status: "running", TotalCount: 5}
	const concurrentObservers = 8
	var wg sync.WaitGroup
	wg.Add(concurrentObservers)
	for i := 0; i < concurrentObservers; i++ {
		go func() {
			defer wg.Done()
			uc.notifyJobTerminalIfNeeded(ctx, previousJob, "completed", repository.BackfillItemStatusSummary{Completed: 5})
		}()
	}
	wg.Wait()

	if got := notifier.callCount(); got != 1 {
		t.Fatalf("expected exactly 1 notification across %d concurrent observers, got %d", concurrentObservers, got)
	}
}
