package pipeline

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

// dispatchBatchRepo captures what CreateBatchJob persists. Legacy-mode tests
// additionally tolerate the background goroutine's bookkeeping calls.
type dispatchBatchRepo struct {
	repository.BackfillRepository
	mu            sync.Mutex
	savedJob      *models.BackfillJob
	savedItems    []models.BackfillItem
	statusUpdates []string
	progressCalls int
}

func (m *dispatchBatchRepo) SaveJob(_ context.Context, j *models.BackfillJob) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *j
	m.savedJob = &cp
	return nil
}

func (m *dispatchBatchRepo) SaveItems(_ context.Context, items []models.BackfillItem) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.savedItems = append([]models.BackfillItem(nil), items...)
	return nil
}

func (m *dispatchBatchRepo) UpdateJobStatus(_ context.Context, _, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.statusUpdates = append(m.statusUpdates, status)
	return nil
}

func (m *dispatchBatchRepo) SummarizeItemStatuses(ctx context.Context, _ string) (repository.BackfillItemStatusSummary, error) {
	if err := ctx.Err(); err != nil {
		return repository.BackfillItemStatusSummary{}, err
	}
	return repository.BackfillItemStatusSummary{}, nil
}

func (m *dispatchBatchRepo) UpdateJobProgress(_ context.Context, _ string, _, _ int, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.progressCalls++
	return nil
}

func (m *dispatchBatchRepo) job() *models.BackfillJob {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.savedJob
}

func newDispatchFixture(t *testing.T) (*Usecase, *dispatchBatchRepo) {
	t.Helper()
	repo := &dispatchBatchRepo{}
	tpl := &mockTemplateRepo{byID: map[string]*models.PipelineTemplate{
		"tpl-1": {ID: "tpl-1", Name: "demo", Version: 3, ActiveVersion: 3},
	}}
	uc := New(tpl, nil, nil, nil, "ns")
	uc.SetBackfillRepo(repo)
	return uc, repo
}

// ── CYB-3677: submitter mode (default) ───────────────────────────────────────

// Default mode persists the job born 'running' with the version pinned in the
// COLUMN (and mirrored in filter_json for one release), kicks the submitter,
// and registers NO in-memory goroutine — persistence IS the dispatch.
func TestCreateBatchJobSubmitterModePersistsRunningAndKicks(t *testing.T) {
	uc, repo := newDispatchFixture(t)
	kicked := 0
	uc.SetBatchSubmitKicker(func() { kicked++ })

	job, err := uc.CreateBatchJob(context.Background(), "tpl-1", "b", []string{"a1", "a2"}, "", 0, 0, "owner")
	if err != nil {
		t.Fatal(err)
	}

	saved := repo.job()
	if saved.Status != "running" {
		t.Fatalf("job status = %q, want running", saved.Status)
	}
	if saved.TemplateVersion != 3 {
		t.Fatalf("job.TemplateVersion = %d, want 3 (pinned column)", saved.TemplateVersion)
	}
	if v, ok := saved.FilterJSON["template_version"]; !ok || v != 3 {
		t.Fatalf("filter_json pin = %v, want 3 (rollback mirror)", v)
	}
	if kicked != 1 {
		t.Fatalf("kicker calls = %d, want 1", kicked)
	}
	if len(repo.savedItems) != 2 {
		t.Fatalf("items = %d, want 2", len(repo.savedItems))
	}
	if uc.cancelBatch(job.ID) {
		t.Fatal("submitter mode must not register an in-memory dispatch goroutine")
	}
}

// A nil kicker must be tolerated: durability never depends on the kick (the
// submitter's 15s ticker re-lists the job regardless).
func TestCreateBatchJobSubmitterModeNilKickerIsFine(t *testing.T) {
	uc, repo := newDispatchFixture(t)

	if _, err := uc.CreateBatchJob(context.Background(), "tpl-1", "b", []string{"a1"}, "", 0, 0, "o"); err != nil {
		t.Fatal(err)
	}
	if repo.job().Status != "running" {
		t.Fatalf("job status = %q, want running", repo.job().Status)
	}
}

// An explicit version request must be pinned verbatim, not the active one.
func TestCreateBatchJobPinsExplicitVersion(t *testing.T) {
	uc, repo := newDispatchFixture(t)

	if _, err := uc.CreateBatchJob(context.Background(), "tpl-1", "b", []string{"a1"}, "", 2, 0, "o"); err != nil {
		t.Fatal(err)
	}
	if repo.job().TemplateVersion != 2 {
		t.Fatalf("job.TemplateVersion = %d, want 2 (explicit pin)", repo.job().TemplateVersion)
	}
}

// ── CYB-3677: legacy mode (rollback flag) ────────────────────────────────────

// Legacy mode preserves the pre-3677 shape byte-for-byte: job born 'pending',
// in-memory goroutine registered (cancelBatch returns true), pin still written
// to the column so a later flag flip inherits it.
func TestCreateBatchJobLegacyModeKeepsGoroutine(t *testing.T) {
	uc, repo := newDispatchFixture(t)
	uc.SetBatchDispatchMode("legacy")
	kicked := 0
	uc.SetBatchSubmitKicker(func() { kicked++ })

	job, err := uc.CreateBatchJob(context.Background(), "tpl-1", "b", []string{"a1"}, "", 0, 0, "o")
	if err != nil {
		t.Fatal(err)
	}

	if repo.job().Status != "pending" {
		t.Fatalf("job status = %q, want pending (legacy)", repo.job().Status)
	}
	if repo.job().TemplateVersion != 3 {
		t.Fatalf("job.TemplateVersion = %d, want 3 (column written in both modes)", repo.job().TemplateVersion)
	}
	if kicked != 0 {
		t.Fatalf("kicker calls = %d, want 0 in legacy mode", kicked)
	}
	// The goroutine is registered; cancel it and wait for its bookkeeping to
	// finish so the test doesn't leak it.
	if !uc.cancelBatch(job.ID) {
		t.Fatal("legacy mode must register the in-memory dispatch goroutine")
	}
	deadline := time.After(5 * time.Second)
	for {
		repo.mu.Lock()
		done := repo.progressCalls >= 1
		repo.mu.Unlock()
		if done {
			break
		}
		select {
		case <-deadline:
			t.Fatal("legacy goroutine never finished after cancel")
		case <-time.After(10 * time.Millisecond):
		}
	}
}

// ── CYB-3677: error paths ────────────────────────────────────────────────────


type errTemplateRepo struct{ repository.PipelineTemplateRepository }

func (r *errTemplateRepo) FindByID(context.Context, string) (*models.PipelineTemplate, error) {
	return nil, context.DeadlineExceeded
}

type erroringBatchRepo struct {
	dispatchBatchRepo
	saveJobErr   error
	saveItemsErr error
}

func (m *erroringBatchRepo) SaveJob(ctx context.Context, j *models.BackfillJob) error {
	if m.saveJobErr != nil {
		return m.saveJobErr
	}
	return m.dispatchBatchRepo.SaveJob(ctx, j)
}

func (m *erroringBatchRepo) SaveItems(ctx context.Context, items []models.BackfillItem) error {
	if m.saveItemsErr != nil {
		return m.saveItemsErr
	}
	return m.dispatchBatchRepo.SaveItems(ctx, items)
}

func TestCreateBatchJobErrorPaths(t *testing.T) {
	ctx := context.Background()
	tpl := &mockTemplateRepo{byID: map[string]*models.PipelineTemplate{
		"tpl-1": {ID: "tpl-1", Name: "demo", Version: 3, ActiveVersion: 3},
	}}

	t.Run("nil backfill repo", func(t *testing.T) {
		uc := New(tpl, nil, nil, nil, "ns")
		if _, err := uc.CreateBatchJob(ctx, "tpl-1", "b", []string{"a1"}, "", 0, 0, "o"); err == nil {
			t.Fatal("want error for missing backfill repo")
		}
	})
	t.Run("template repo error", func(t *testing.T) {
		uc := New(&errTemplateRepo{}, nil, nil, nil, "ns")
		uc.SetBackfillRepo(&dispatchBatchRepo{})
		if _, err := uc.CreateBatchJob(ctx, "tpl-1", "b", []string{"a1"}, "", 0, 0, "o"); err == nil {
			t.Fatal("want error when template lookup fails")
		}
	})
	t.Run("active version pins when newer than saved", func(t *testing.T) {
		repo := &dispatchBatchRepo{}
		uc := New(&mockTemplateRepo{byID: map[string]*models.PipelineTemplate{
			"tpl-1": {ID: "tpl-1", Name: "demo", Version: 3, ActiveVersion: 2},
		}}, nil, nil, nil, "ns")
		uc.SetBackfillRepo(repo)
		if _, err := uc.CreateBatchJob(ctx, "tpl-1", "b", []string{"a1"}, "", 0, 0, "o"); err != nil {
			t.Fatal(err)
		}
		if repo.job().TemplateVersion != 2 {
			t.Fatalf("job.TemplateVersion = %d, want 2 (active pin)", repo.job().TemplateVersion)
		}
	})
	t.Run("template not found", func(t *testing.T) {
		uc := New(tpl, nil, nil, nil, "ns")
		uc.SetBackfillRepo(&dispatchBatchRepo{})
		if _, err := uc.CreateBatchJob(ctx, "tpl-missing", "b", []string{"a1"}, "", 0, 0, "o"); err == nil {
			t.Fatal("want error for missing template")
		}
	})
	t.Run("empty asset ids", func(t *testing.T) {
		uc := New(tpl, nil, nil, nil, "ns")
		uc.SetBackfillRepo(&dispatchBatchRepo{})
		if _, err := uc.CreateBatchJob(ctx, "tpl-1", "b", nil, "", 0, 0, "o"); err == nil {
			t.Fatal("want error for empty asset_ids")
		}
	})
	t.Run("save job fails", func(t *testing.T) {
		uc := New(tpl, nil, nil, nil, "ns")
		uc.SetBackfillRepo(&erroringBatchRepo{saveJobErr: context.DeadlineExceeded})
		if _, err := uc.CreateBatchJob(ctx, "tpl-1", "b", []string{"a1"}, "", 0, 0, "o"); err == nil {
			t.Fatal("want error when SaveJob fails")
		}
	})
	t.Run("save items fails marks job failed", func(t *testing.T) {
		repo := &erroringBatchRepo{saveItemsErr: context.DeadlineExceeded}
		uc := New(tpl, nil, nil, nil, "ns")
		uc.SetBackfillRepo(repo)
		if _, err := uc.CreateBatchJob(ctx, "tpl-1", "b", []string{"a1"}, "", 0, 0, "o"); err == nil {
			t.Fatal("want error when SaveItems fails")
		}
		repo.mu.Lock()
		defer repo.mu.Unlock()
		if len(repo.statusUpdates) != 1 || repo.statusUpdates[0] != "failed" {
			t.Fatalf("status updates = %v, want [failed]", repo.statusUpdates)
		}
	})
	t.Run("target id recorded in filter", func(t *testing.T) {
		repo := &dispatchBatchRepo{}
		uc := New(tpl, nil, nil, nil, "ns")
		uc.SetBackfillRepo(repo)
		if _, err := uc.CreateBatchJob(ctx, "tpl-1", "b", []string{"a1"}, "cluster-b", 0, 0, "o"); err != nil {
			t.Fatal(err)
		}
		if got := repo.job().FilterJSON["target_id"]; got != "cluster-b" {
			t.Fatalf("filter target_id = %v, want cluster-b", got)
		}
	})
	t.Run("default batch name from template", func(t *testing.T) {
		repo := &dispatchBatchRepo{}
		uc := New(tpl, nil, nil, nil, "ns")
		uc.SetBackfillRepo(repo)
		if _, err := uc.CreateBatchJob(ctx, "tpl-1", "", []string{"a1"}, "", 0, 0, "o"); err != nil {
			t.Fatal(err)
		}
		if repo.job().Name == "" {
			t.Fatal("want defaulted batch name")
		}
	})
}

func TestSetBatchDispatchMode(t *testing.T) {
	uc := &Usecase{}
	cases := []struct {
		mode string
		want bool
	}{
		{"legacy", true},
		{" LEGACY ", true},
		{"submitter", false},
		{"", false},
		{"anything-else", false},
	}
	for _, tc := range cases {
		uc.SetBatchDispatchMode(tc.mode)
		if uc.batchDispatchLegacy != tc.want {
			t.Fatalf("mode %q: legacy = %v, want %v", tc.mode, uc.batchDispatchLegacy, tc.want)
		}
	}
}
