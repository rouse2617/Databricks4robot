package backfill

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	pipelineUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/pipeline"
)

// ── fakes ────────────────────────────────────────────────────────────────────

// fakeSubmitQueue is an in-memory repository.SubmitQueue backed by the same
// pausedSyncRepo the other backfill tests use, so item mutations made by the
// submitter are visible to assertions.
type fakeSubmitQueue struct {
	repo *pausedSyncRepo
	jobs []models.BackfillJob

	mu             sync.Mutex
	lockedNow      map[string]bool // simulate a row concurrently locked elsewhere
	lastListLimit  int
	withTxErrs     []error
	rollbackedByTx int
}

func (q *fakeSubmitQueue) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	err := fn(ctx)
	if err != nil {
		q.mu.Lock()
		q.rollbackedByTx++
		q.withTxErrs = append(q.withTxErrs, err)
		q.mu.Unlock()
	}
	return err
}

func (q *fakeSubmitQueue) FindSubmittableJobs(_ context.Context, _ int) ([]models.BackfillJob, error) {
	return q.jobs, nil
}

func (q *fakeSubmitQueue) ListSubmittableItemIDs(_ context.Context, jobID string, limit int) ([]string, error) {
	q.mu.Lock()
	q.lastListLimit = limit
	q.mu.Unlock()
	ids := []string{}
	for _, it := range q.repo.items {
		if it.JobID == jobID && it.Status == "pending" {
			ids = append(ids, it.ID)
			if len(ids) >= limit {
				break
			}
		}
	}
	return ids, nil
}

func (q *fakeSubmitQueue) LockPendingItem(_ context.Context, itemID string) (*models.BackfillItem, error) {
	q.mu.Lock()
	locked := q.lockedNow[itemID]
	q.mu.Unlock()
	if locked {
		return nil, nil // SKIP LOCKED: someone else holds the row
	}
	for i := range q.repo.items {
		if q.repo.items[i].ID == itemID && q.repo.items[i].Status == "pending" {
			cp := q.repo.items[i]
			return &cp, nil
		}
	}
	return nil, nil
}

// fakeDeployer scripts the pipeline-side behaviour per asset.
type fakeDeployer struct {
	mu sync.Mutex

	deployErrByAsset map[string]error // nil entry → success
	runsByID         map[string]*models.PipelineRun

	upserts   []string
	deploys   []string
	commits   []string
	failures  []string
	refreshes []string
}

func (d *fakeDeployer) GetRun(_ context.Context, id string) (*models.PipelineRun, error) {
	if r, ok := d.runsByID[id]; ok {
		return r, nil
	}
	return nil, errors.New("not found")
}

func (d *fakeDeployer) UpsertBatchSubtaskRun(_ context.Context, in pipelineUC.BatchSubtaskRunInput) (string, string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.upserts = append(d.upserts, in.AssetID)
	return "run-" + in.AssetID, "wf-batch-" + in.AssetID, nil
}

func (d *fakeDeployer) DeployByTemplateID(_ context.Context, _ string, _ string, assetIDs []string, _ ...pipelineUC.DeployOptions) (*models.PipelineDeployment, error) {
	asset := assetIDs[0]
	d.mu.Lock()
	d.deploys = append(d.deploys, asset)
	err := d.deployErrByAsset[asset]
	d.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return &models.PipelineDeployment{ID: "run-" + asset, WorkflowName: "wf-batch-" + asset}, nil
}

func (d *fakeDeployer) CommitBatchSubtaskDeploy(_ context.Context, runID string, _ *models.PipelineDeployment) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.commits = append(d.commits, runID)
	return nil
}

func (d *fakeDeployer) RecordBatchSubtaskFailure(_ context.Context, in pipelineUC.BatchSubtaskRunInput) (string, string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.failures = append(d.failures, in.AssetID)
	return "run-" + in.AssetID, "wf-batch-" + in.AssetID, nil
}

func (d *fakeDeployer) RefreshRunFromWorkflowByName(_ context.Context, workflowName, _ string) (*models.PipelineRun, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.refreshes = append(d.refreshes, workflowName)
	return &models.PipelineRun{WorkflowName: workflowName, ArgoWorkflowUID: "uid-backfilled"}, nil
}

func newSubmitterFixture(job *models.BackfillJob, items []models.BackfillItem) (*Usecase, *pausedSyncRepo, *fakeSubmitQueue, *fakeDeployer) {
	repo := &pausedSyncRepo{job: job, items: items}
	q := &fakeSubmitQueue{repo: repo, jobs: []models.BackfillJob{*job}, lockedNow: map[string]bool{}}
	d := &fakeDeployer{deployErrByAsset: map[string]error{}, runsByID: map[string]*models.PipelineRun{}}
	uc := New(repo, nil)
	uc.deployer = d
	uc.SetSubmitQueue(q)
	return uc, repo, q, d
}

func itemStatus(repo *pausedSyncRepo, id string) string {
	for i := range repo.items {
		if repo.items[i].ID == id {
			return repo.items[i].Status
		}
	}
	return "<missing>"
}

// ── tests ────────────────────────────────────────────────────────────────────

// Happy path: a pending item is deployed and moves pending → submitted with
// the run committed. Forward-only: nothing ever goes back to pending.
func TestSubmitter_SubmitsPendingItemToSubmitted(t *testing.T) {
	ctx := context.Background()
	job := &models.BackfillJob{ID: "job-1", Status: "running", TemplateID: "tpl-1", TemplateVersion: 1, TotalCount: 1}
	uc, repo, _, d := newSubmitterFixture(job, []models.BackfillItem{
		{ID: "item-1", JobID: "job-1", AssetID: "asset-1", Status: "pending"},
	})

	uc.runSubmitterCycle(ctx)

	if got := itemStatus(repo, "item-1"); got != "submitted" {
		t.Fatalf("item status = %q, want submitted", got)
	}
	if len(d.deploys) != 1 || d.deploys[0] != "asset-1" {
		t.Fatalf("deploys = %v, want [asset-1]", d.deploys)
	}
	if len(d.commits) != 1 {
		t.Fatalf("commits = %v, want exactly one", d.commits)
	}
	if len(d.failures) != 0 {
		t.Fatalf("failures = %v, want none", d.failures)
	}
}

// AlreadyExists is success: our own prior submission survived a crash — the
// UID is backfilled (webhook-equivalent refresh) and the item is submitted.
// It is NOT a failure and attempts is untouched (auto re-submission never
// bumps the manual-retry counter).
func TestSubmitter_AlreadyExistsBackfillsAndMarksSubmitted(t *testing.T) {
	ctx := context.Background()
	job := &models.BackfillJob{ID: "job-1", Status: "running", TemplateID: "tpl-1", TemplateVersion: 1, TotalCount: 1}
	uc, repo, _, d := newSubmitterFixture(job, []models.BackfillItem{
		{ID: "item-1", JobID: "job-1", AssetID: "asset-1", Status: "pending", Attempts: 0},
	})
	d.deployErrByAsset["asset-1"] = fmt.Errorf("submit: %w", errAlreadyExistsForTest)

	uc.runSubmitterCycle(ctx)

	if got := itemStatus(repo, "item-1"); got != "submitted" {
		t.Fatalf("item status = %q, want submitted", got)
	}
	if len(d.refreshes) != 1 {
		t.Fatalf("refreshes = %v, want exactly one uid backfill", d.refreshes)
	}
	if len(d.failures) != 0 {
		t.Fatalf("failures = %v, want none (already-exists is success)", d.failures)
	}
	if repo.items[0].Attempts != 0 {
		t.Fatalf("attempts = %d, want 0 (auto re-submission never bumps attempts)", repo.items[0].Attempts)
	}
}

// errAlreadyExistsForTest matches isWorkflowAlreadyExists via its message,
// deliberately exercising the string fallback (error chains that crossed a
// non-%w boundary).
var errAlreadyExistsForTest = errors.New("workflows.argoproj.io \"wf-batch-asset-1\" already exists")

// A deterministic submission failure surfaces: the item is failed with the
// message recorded — never silently retried into a loop.
func TestSubmitter_DeterministicFailureSurfaces(t *testing.T) {
	ctx := context.Background()
	job := &models.BackfillJob{ID: "job-1", Status: "running", TemplateID: "tpl-1", TemplateVersion: 1, TotalCount: 1}
	uc, repo, _, d := newSubmitterFixture(job, []models.BackfillItem{
		{ID: "item-1", JobID: "job-1", AssetID: "asset-1", Status: "pending"},
	})
	d.deployErrByAsset["asset-1"] = errors.New("transpile: node X invalid")

	uc.runSubmitterCycle(ctx)

	if got := itemStatus(repo, "item-1"); got != "failed" {
		t.Fatalf("item status = %q, want failed", got)
	}
	if len(d.failures) != 1 {
		t.Fatalf("failures = %v, want one recorded subtask failure", d.failures)
	}
}

// SKIP LOCKED: a row concurrently locked by another worker is skipped —
// no deploy, no state change, no error.
func TestSubmitter_SkipsConcurrentlyLockedItem(t *testing.T) {
	ctx := context.Background()
	job := &models.BackfillJob{ID: "job-1", Status: "running", TemplateID: "tpl-1", TemplateVersion: 1, TotalCount: 1}
	uc, repo, q, d := newSubmitterFixture(job, []models.BackfillItem{
		{ID: "item-1", JobID: "job-1", AssetID: "asset-1", Status: "pending"},
	})
	q.lockedNow["item-1"] = true

	uc.runSubmitterCycle(ctx)

	if got := itemStatus(repo, "item-1"); got != "pending" {
		t.Fatalf("item status = %q, want pending (untouched)", got)
	}
	if len(d.deploys) != 0 {
		t.Fatalf("deploys = %v, want none for a locked row", d.deploys)
	}
}

// A paused job's items are left pending (no deploy); resume kicks the
// submitter and they go out on the next cycle.
func TestSubmitter_PausedJobLeavesItemsPending(t *testing.T) {
	ctx := context.Background()
	job := &models.BackfillJob{ID: "job-1", Status: "paused", TemplateID: "tpl-1", TemplateVersion: 1, TotalCount: 1}
	uc, repo, q, d := newSubmitterFixture(job, []models.BackfillItem{
		{ID: "item-1", JobID: "job-1", AssetID: "asset-1", Status: "pending"},
	})
	// FindSubmittableJobs would not return a paused job in production; this
	// exercises the second, per-item guard (pause raced the cycle).
	q.jobs[0].Status = "running"

	uc.runSubmitterCycle(ctx)

	if got := itemStatus(repo, "item-1"); got != "pending" {
		t.Fatalf("item status = %q, want pending (job paused)", got)
	}
	if len(d.deploys) != 0 {
		t.Fatalf("deploys = %v, want none while paused", d.deploys)
	}
}

// Pilot quota: a pilot_running job only submits within the remaining quota —
// the candidate list is capped at quota, and a fully-attempted pilot submits
// nothing (the sync flips it to pilot_review).
func TestSubmitter_PilotQuotaGatesSubmissions(t *testing.T) {
	ctx := context.Background()
	job := &models.BackfillJob{ID: "job-1", Status: "pilot_running", PilotCount: 1, TemplateID: "tpl-1", TemplateVersion: 1, TotalCount: 3}
	uc, repo, q, d := newSubmitterFixture(job, []models.BackfillItem{
		{ID: "item-1", JobID: "job-1", AssetID: "asset-1", Status: "pending"},
		{ID: "item-2", JobID: "job-1", AssetID: "asset-2", Status: "pending"},
		{ID: "item-3", JobID: "job-1", AssetID: "asset-3", Status: "pending"},
	})

	uc.runSubmitterCycle(ctx)

	if q.lastListLimit != 1 {
		t.Fatalf("candidate limit = %d, want pilot quota 1", q.lastListLimit)
	}
	if len(d.deploys) != 1 {
		t.Fatalf("deploys = %v, want exactly the pilot quota (1)", d.deploys)
	}
	submitted := 0
	for _, it := range repo.items {
		if it.Status == "submitted" {
			submitted++
		}
	}
	if submitted != 1 {
		t.Fatalf("submitted = %d, want 1", submitted)
	}

	// Quota exhausted (attempted = total - pending = 1 ≥ pilotCount): the
	// next cycle submits nothing more.
	d.deploys = nil
	uc.runSubmitterCycle(ctx)
	if len(d.deploys) != 0 {
		t.Fatalf("deploys after quota exhausted = %v, want none", d.deploys)
	}
}

// Idempotency guard: an item whose run is already live in Argo (uid persisted,
// e.g. crash between run commit and item update) is marked submitted without
// a second deploy — the duplicate-workflow bug class.
func TestSubmitter_AlreadyLiveRunSkipsDeploy(t *testing.T) {
	ctx := context.Background()
	runID := "run-live"
	job := &models.BackfillJob{ID: "job-1", Status: "running", TemplateID: "tpl-1", TemplateVersion: 1, TotalCount: 1}
	uc, repo, _, d := newSubmitterFixture(job, []models.BackfillItem{
		{ID: "item-1", JobID: "job-1", AssetID: "asset-1", Status: "pending", PipelineRunID: &runID},
	})
	d.runsByID[runID] = &models.PipelineRun{ID: runID, WorkflowName: "wf-live", ArgoWorkflowUID: "uid-live", Status: "Running"}

	uc.runSubmitterCycle(ctx)

	if got := itemStatus(repo, "item-1"); got != "submitted" {
		t.Fatalf("item status = %q, want submitted", got)
	}
	if len(d.deploys) != 0 {
		t.Fatalf("deploys = %v, want none (run already live)", d.deploys)
	}
}

// Infra error mid-submit rolls the transaction back: the item stays pending
// for the next cycle, and nothing is marked failed.
func TestSubmitter_InfraErrorRollsBackToPending(t *testing.T) {
	ctx := context.Background()
	job := &models.BackfillJob{ID: "job-1", Status: "running", TemplateID: "tpl-1", TemplateVersion: 1, TotalCount: 1}
	uc, repo, q, d := newSubmitterFixture(job, []models.BackfillItem{
		{ID: "item-1", JobID: "job-1", AssetID: "asset-1", Status: "pending"},
	})
	// UpsertBatchSubtaskRun failing is an infra-class error → rollback path.
	failing := &upsertFailingDeployer{fakeDeployer: d}
	uc.deployer = failing

	uc.runSubmitterCycle(ctx)

	if got := itemStatus(repo, "item-1"); got != "pending" {
		t.Fatalf("item status = %q, want pending (rolled back, retried next cycle)", got)
	}
	if q.rollbackedByTx == 0 {
		t.Fatal("expected the item transaction to roll back")
	}
	if len(d.failures) != 0 {
		t.Fatalf("failures = %v, want none for an infra error", d.failures)
	}
}

type upsertFailingDeployer struct{ *fakeDeployer }

func (d *upsertFailingDeployer) UpsertBatchSubtaskRun(context.Context, pipelineUC.BatchSubtaskRunInput) (string, string, error) {
	return "", "", errors.New("db unavailable")
}

// Lifecycle smoke: Start → Kick → Stop terminates cleanly; Kick before Start
// is a safe no-op.
func TestSubmitter_StartKickStopLifecycle(t *testing.T) {
	job := &models.BackfillJob{ID: "job-1", Status: "running", TemplateID: "tpl-1", TemplateVersion: 1, TotalCount: 1}
	uc, repo, _, _ := newSubmitterFixture(job, []models.BackfillItem{
		{ID: "item-1", JobID: "job-1", AssetID: "asset-1", Status: "pending"},
	})

	uc.KickSubmitter() // before Start: no panic
	uc.StartSubmitter()
	uc.KickSubmitter()
	uc.StopSubmitter()

	// The eager boot cycle must have dispatched the pending item — this is
	// the redeploy-survival property (no ResumeIncompleteBatches needed).
	if got := itemStatus(repo, "item-1"); got != "submitted" {
		t.Fatalf("item status after boot cycle = %q, want submitted", got)
	}
}

// isWorkflowAlreadyExists must catch both the typed error and the string form.
func TestIsWorkflowAlreadyExists(t *testing.T) {
	if isWorkflowAlreadyExists(nil) {
		t.Fatal("nil is not already-exists")
	}
	if !isWorkflowAlreadyExists(errors.New(`workflows.argoproj.io "x" already exists`)) {
		t.Fatal("string form should match")
	}
	if isWorkflowAlreadyExists(errors.New("connection refused")) {
		t.Fatal("unrelated error must not match")
	}
	if !strings.Contains(errAlreadyExistsForTest.Error(), "already exists") {
		t.Fatal("fixture must carry the marker")
	}
}
