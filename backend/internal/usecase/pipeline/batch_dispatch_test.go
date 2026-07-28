package pipeline

import (
	"context"
	"sync"
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

// dispatchBatchRepo captures what CreateBatchJob persists.
type dispatchBatchRepo struct {
	repository.BackfillRepository
	mu            sync.Mutex
	savedJob      *models.BackfillJob
	savedItems    []models.BackfillItem
	statusUpdates []string
	progressCalls int
}

func (m *dispatchBatchRepo) IncrementItemSubmitAttempts(context.Context, string) (int, error) {
	return 0, nil
}

func (m *dispatchBatchRepo) ResetFailedItems(context.Context, string) (int64, error) { return 0, nil }

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

// ── CYB-3677: submitter mode ─────────────────────────────────────────────────

// Submitter mode persists the job born 'running' with the version pinned in
// the COLUMN (and mirrored in filter_json for one release) and kicks the
// submitter — persistence IS the dispatch.
func TestCreateBatchJobSubmitterModePersistsRunningAndKicks(t *testing.T) {
	uc, repo := newDispatchFixture(t)
	kicked := 0
	uc.SetBatchSubmitKicker(func() { kicked++ })

	_, err := uc.CreateBatchJob(context.Background(), "tpl-1", "b", []string{"a1", "a2"}, "", 0, "owner")
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
}

// A nil kicker must be tolerated: durability never depends on the kick (the
// submitter's 15s ticker re-lists the job regardless).
func TestCreateBatchJobSubmitterModeNilKickerIsFine(t *testing.T) {
	uc, repo := newDispatchFixture(t)

	if _, err := uc.CreateBatchJob(context.Background(), "tpl-1", "b", []string{"a1"}, "", 0, "o"); err != nil {
		t.Fatal(err)
	}
	if repo.job().Status != "running" {
		t.Fatalf("job status = %q, want running", repo.job().Status)
	}
}

// An explicit version request must be pinned verbatim, not the active one.
func TestCreateBatchJobPinsExplicitVersion(t *testing.T) {
	uc, repo := newDispatchFixture(t)

	if _, err := uc.CreateBatchJob(context.Background(), "tpl-1", "b", []string{"a1"}, "", 2, "o"); err != nil {
		t.Fatal(err)
	}
	if repo.job().TemplateVersion != 2 {
		t.Fatalf("job.TemplateVersion = %d, want 2 (explicit pin)", repo.job().TemplateVersion)
	}
}

// ── CYB-3677: error paths ────────────────────────────────────────────────────

type errTemplateRepo struct {
	repository.PipelineTemplateRepository
}

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
		if _, err := uc.CreateBatchJob(ctx, "tpl-1", "b", []string{"a1"}, "", 0, "o"); err == nil {
			t.Fatal("want error for missing backfill repo")
		}
	})
	t.Run("template repo error", func(t *testing.T) {
		uc := New(&errTemplateRepo{}, nil, nil, nil, "ns")
		uc.SetBackfillRepo(&dispatchBatchRepo{})
		if _, err := uc.CreateBatchJob(ctx, "tpl-1", "b", []string{"a1"}, "", 0, "o"); err == nil {
			t.Fatal("want error when template lookup fails")
		}
	})
	t.Run("active version pins when newer than saved", func(t *testing.T) {
		repo := &dispatchBatchRepo{}
		uc := New(&mockTemplateRepo{byID: map[string]*models.PipelineTemplate{
			"tpl-1": {ID: "tpl-1", Name: "demo", Version: 3, ActiveVersion: 2},
		}}, nil, nil, nil, "ns")
		uc.SetBackfillRepo(repo)
		if _, err := uc.CreateBatchJob(ctx, "tpl-1", "b", []string{"a1"}, "", 0, "o"); err != nil {
			t.Fatal(err)
		}
		if repo.job().TemplateVersion != 2 {
			t.Fatalf("job.TemplateVersion = %d, want 2 (active pin)", repo.job().TemplateVersion)
		}
	})
	t.Run("template not found", func(t *testing.T) {
		uc := New(tpl, nil, nil, nil, "ns")
		uc.SetBackfillRepo(&dispatchBatchRepo{})
		if _, err := uc.CreateBatchJob(ctx, "tpl-missing", "b", []string{"a1"}, "", 0, "o"); err == nil {
			t.Fatal("want error for missing template")
		}
	})
	t.Run("empty asset ids", func(t *testing.T) {
		uc := New(tpl, nil, nil, nil, "ns")
		uc.SetBackfillRepo(&dispatchBatchRepo{})
		if _, err := uc.CreateBatchJob(ctx, "tpl-1", "b", nil, "", 0, "o"); err == nil {
			t.Fatal("want error for empty asset_ids")
		}
	})
	t.Run("save job fails", func(t *testing.T) {
		uc := New(tpl, nil, nil, nil, "ns")
		uc.SetBackfillRepo(&erroringBatchRepo{saveJobErr: context.DeadlineExceeded})
		if _, err := uc.CreateBatchJob(ctx, "tpl-1", "b", []string{"a1"}, "", 0, "o"); err == nil {
			t.Fatal("want error when SaveJob fails")
		}
	})
	t.Run("save items fails marks job failed", func(t *testing.T) {
		repo := &erroringBatchRepo{saveItemsErr: context.DeadlineExceeded}
		uc := New(tpl, nil, nil, nil, "ns")
		uc.SetBackfillRepo(repo)
		if _, err := uc.CreateBatchJob(ctx, "tpl-1", "b", []string{"a1"}, "", 0, "o"); err == nil {
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
		if _, err := uc.CreateBatchJob(ctx, "tpl-1", "b", []string{"a1"}, "cluster-b", 0, "o"); err != nil {
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
		if _, err := uc.CreateBatchJob(ctx, "tpl-1", "", []string{"a1"}, "", 0, "o"); err != nil {
			t.Fatal(err)
		}
		if repo.job().Name == "" {
			t.Fatal("want defaulted batch name")
		}
	})
}

// ── CYB-3678: DLQ list / retry ───────────────────────────────────────────────

type dlqBatchRepo struct {
	dispatchBatchRepo
	items   []models.BackfillItem
	revived int64
}

func (m *dlqBatchRepo) FindItemsByJobID(_ context.Context, jobID string) ([]models.BackfillItem, error) {
	out := []models.BackfillItem{}
	for _, it := range m.items {
		if it.JobID == jobID {
			out = append(out, it)
		}
	}
	return out, nil
}

func (m *dlqBatchRepo) ResetFailedItems(_ context.Context, jobID string) (int64, error) {
	var n int64
	for i := range m.items {
		if m.items[i].JobID == jobID && m.items[i].Status == "failed" {
			m.items[i].Status = "pending"
			n++
		}
	}
	m.revived = n
	return n, nil
}

func TestListBatchDLQ(t *testing.T) {
	msg := "boom"
	repo := &dlqBatchRepo{items: []models.BackfillItem{
		{ID: "i1", JobID: "j1", AssetID: "a1", Status: "failed", ErrorMessage: &msg},
		{ID: "i2", JobID: "j1", AssetID: "a2", Status: "completed"},
		{ID: "i3", JobID: "j2", AssetID: "a3", Status: "failed"},
	}}
	uc := &Usecase{}
	uc.SetBackfillRepo(repo)

	got, err := uc.ListBatchDLQ(context.Background(), "j1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "i1" {
		t.Fatalf("dlq = %+v, want only i1", got)
	}

	if _, err := (&Usecase{}).ListBatchDLQ(context.Background(), "j1"); err == nil {
		t.Fatal("nil repo must error")
	}
}

func TestRetryBatchDLQ(t *testing.T) {
	repo := &dlqBatchRepo{items: []models.BackfillItem{
		{ID: "i1", JobID: "j1", AssetID: "a1", Status: "failed"},
		{ID: "i2", JobID: "j1", AssetID: "a2", Status: "failed"},
	}}
	uc := &Usecase{}
	uc.SetBackfillRepo(repo)
	kicked := 0
	uc.SetBatchSubmitKicker(func() { kicked++ })

	n, err := uc.RetryBatchDLQ(context.Background(), "j1")
	if err != nil || n != 2 {
		t.Fatalf("revived = %d err = %v, want 2/nil", n, err)
	}
	repo.mu.Lock()
	statusOK := len(repo.statusUpdates) == 1 && repo.statusUpdates[0] == "running"
	repo.mu.Unlock()
	if !statusOK {
		t.Fatalf("status updates = %v, want [running]", repo.statusUpdates)
	}
	if kicked != 1 {
		t.Fatalf("kicks = %d, want 1", kicked)
	}

	// Nothing to revive → no status flip, no kick.
	n, err = uc.RetryBatchDLQ(context.Background(), "j1")
	if err != nil || n != 0 {
		t.Fatalf("second revive = %d err=%v, want 0/nil", n, err)
	}
	if kicked != 1 {
		t.Fatalf("kicks = %d, want still 1", kicked)
	}

	if _, err := (&Usecase{}).RetryBatchDLQ(context.Background(), "j1"); err == nil {
		t.Fatal("nil repo must error")
	}
}

// ── CYB-3678: cluster resolution + DLQ error branches ────────────────────────

func TestResolveTargetClusterID(t *testing.T) {
	uc := &Usecase{}
	// No target repo, empty target → the built-in default target (no
	// ClusterID) → "default".
	if got := uc.ResolveTargetClusterID(context.Background(), ""); got != "default" {
		t.Fatalf("empty target → %q, want default", got)
	}
	// Unresolvable target id (no repo) → "default", never an error.
	if got := uc.ResolveTargetClusterID(context.Background(), "ghost"); got != "default" {
		t.Fatalf("unresolvable target → %q, want default", got)
	}
	// A resolvable target with a ClusterID → that cluster.
	target := &models.ExecutionTarget{ID: "tgt-b", ClusterID: " cluster-b ", Enabled: true}
	uc2 := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, newMockAssetRepo(), nil, "default")
	uc2.SetRunRepositories(&mockTargetRepo{byID: map[string]*models.ExecutionTarget{"tgt-b": target}}, nil, nil)
	if got := uc2.ResolveTargetClusterID(context.Background(), "tgt-b"); got != "cluster-b" {
		t.Fatalf("target with cluster → %q, want cluster-b", got)
	}
}

type erroringDLQRepo struct {
	dlqBatchRepo
	findErr   error
	statusErr error
	resetErr  error
}

func (m *erroringDLQRepo) ResetFailedItems(ctx context.Context, jobID string) (int64, error) {
	if m.resetErr != nil {
		return 0, m.resetErr
	}
	return m.dlqBatchRepo.ResetFailedItems(ctx, jobID)
}

func (m *erroringDLQRepo) FindItemsByJobID(ctx context.Context, jobID string) ([]models.BackfillItem, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	return m.dlqBatchRepo.FindItemsByJobID(ctx, jobID)
}

func (m *erroringDLQRepo) UpdateJobStatus(ctx context.Context, id, status string) error {
	if m.statusErr != nil {
		return m.statusErr
	}
	return m.dlqBatchRepo.UpdateJobStatus(ctx, id, status)
}

func TestBatchDLQErrorBranches(t *testing.T) {
	ctx := context.Background()
	t.Run("list surfaces repo error", func(t *testing.T) {
		uc := &Usecase{}
		uc.SetBackfillRepo(&erroringDLQRepo{findErr: context.DeadlineExceeded})
		if _, err := uc.ListBatchDLQ(ctx, "j1"); err == nil {
			t.Fatal("want find error")
		}
	})
	t.Run("retry surfaces status error", func(t *testing.T) {
		repo := &erroringDLQRepo{statusErr: context.DeadlineExceeded}
		repo.items = []models.BackfillItem{{ID: "i1", JobID: "j1", Status: "failed"}}
		uc := &Usecase{}
		uc.SetBackfillRepo(repo)
		if _, err := uc.RetryBatchDLQ(ctx, "j1"); err == nil {
			t.Fatal("want status error")
		}
	})
	t.Run("retry surfaces reset error", func(t *testing.T) {
		repo := &erroringDLQRepo{resetErr: context.DeadlineExceeded}
		uc := &Usecase{}
		uc.SetBackfillRepo(repo)
		if _, err := uc.RetryBatchDLQ(ctx, "j1"); err == nil {
			t.Fatal("want reset error")
		}
	})
}
