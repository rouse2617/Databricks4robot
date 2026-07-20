package backfill

import (
	"context"
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/CyberOrigin2077/cyber-databrew/internal/metrics"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// ── CYB-3677: template version pinning ───────────────────────────────────────

// The template_version column is authoritative: when set, the submitter must
// never consult the template's current active version — even when it moved
// mid-batch (the fixture has no pipelineUC, so any resolveTemplateVersion
// call would yield 0 and the assertion below would catch it).
func TestSubmitter_PinnedColumnVersionWins(t *testing.T) {
	job := &models.BackfillJob{ID: "job-1", Status: "running", TemplateID: "tpl-1", TemplateVersion: 3, TotalCount: 1}
	uc, _, _, d := newSubmitterFixture(job, []models.BackfillItem{
		{ID: "item-1", JobID: "job-1", AssetID: "asset-1", Status: "pending"},
	})

	uc.runSubmitterCycle(context.Background())

	if len(d.upsertVersions) != 1 || d.upsertVersions[0] != 3 {
		t.Fatalf("upsert versions = %v, want [3]", d.upsertVersions)
	}
	if len(d.deployVersions) != 1 || d.deployVersions[0] != 3 {
		t.Fatalf("deploy versions = %v, want [3]", d.deployVersions)
	}
}

// Legacy rows (pre CYB-3677) pinned the version in filter_json only; the
// submitter must honour that pin before ever considering the active version.
func TestSubmitter_FilterJSONVersionPinFallback(t *testing.T) {
	job := &models.BackfillJob{
		ID: "job-1", Status: "running", TemplateID: "tpl-1", TotalCount: 1,
		FilterJSON: map[string]interface{}{"template_version": float64(5)}, // JSON round-trip type
	}
	uc, _, _, d := newSubmitterFixture(job, []models.BackfillItem{
		{ID: "item-1", JobID: "job-1", AssetID: "asset-1", Status: "pending"},
	})

	uc.runSubmitterCycle(context.Background())

	if len(d.deployVersions) != 1 || d.deployVersions[0] != 5 {
		t.Fatalf("deploy versions = %v, want [5]", d.deployVersions)
	}
}

// A row with no pin anywhere falls back to the active version AND counts the
// fallback — new batches always write the column, so this firing on new data
// means the pinning write path regressed.
func TestSubmitter_UnpinnedFallsBackToActiveAndCountsIt(t *testing.T) {
	before := testutil.ToFloat64(metrics.DispatcherTemplateFallbackTotal)
	job := &models.BackfillJob{ID: "job-1", Status: "running", TemplateID: "tpl-1", TotalCount: 1}
	uc, _, _, d := newSubmitterFixture(job, []models.BackfillItem{
		{ID: "item-1", JobID: "job-1", AssetID: "asset-1", Status: "pending"},
	})

	uc.runSubmitterCycle(context.Background())

	// No pipelineUC in the fixture → resolveTemplateVersion yields 0; the
	// point is the fallback PATH was taken and counted, not the value.
	if len(d.deployVersions) != 1 || d.deployVersions[0] != 0 {
		t.Fatalf("deploy versions = %v, want [0] (resolver-less fixture)", d.deployVersions)
	}
	if got := testutil.ToFloat64(metrics.DispatcherTemplateFallbackTotal) - before; got != 1 {
		t.Fatalf("fallback metric delta = %v, want 1", got)
	}
}

func TestTemplateVersionFromBackfillFilter(t *testing.T) {
	cases := []struct {
		name   string
		values map[string]interface{}
		want   int
	}{
		{"json float64", map[string]interface{}{"template_version": float64(4)}, 4},
		{"native int", map[string]interface{}{"template_version": 7}, 7},
		{"numeric string", map[string]interface{}{"template_version": " 9 "}, 9},
		{"garbage string", map[string]interface{}{"template_version": "x"}, 0},
		{"wrong type", map[string]interface{}{"template_version": true}, 0},
		{"absent", map[string]interface{}{}, 0},
		{"nil map", nil, 0},
	}
	for _, tc := range cases {
		if got := templateVersionFromBackfillFilter(tc.values); got != tc.want {
			t.Fatalf("%s: got %d, want %d", tc.name, got, tc.want)
		}
	}
}

// ── CYB-3677: cross-instance cycle lock ──────────────────────────────────────

// lockingSubmitQueue wraps fakeSubmitQueue with a scriptable cycle lock.
type lockingSubmitQueue struct {
	*fakeSubmitQueue
	acquired  bool
	lockErr   error
	lockCalls int
}

func (q *lockingSubmitQueue) WithSubmitterClusterLock(ctx context.Context, _ string, fn func(context.Context) error) (bool, error) {
	q.lockCalls++
	if q.lockErr != nil {
		return false, q.lockErr
	}
	if !q.acquired {
		return false, nil
	}
	return true, fn(ctx)
}

func newLockedFixture(acquired bool, lockErr error) (*Usecase, *lockingSubmitQueue, *fakeDeployer) {
	job := &models.BackfillJob{ID: "job-1", Status: "running", TemplateID: "tpl-1", TemplateVersion: 1, TotalCount: 1}
	uc, _, q, d := newSubmitterFixture(job, []models.BackfillItem{
		{ID: "item-1", JobID: "job-1", AssetID: "asset-1", Status: "pending"},
	})
	lq := &lockingSubmitQueue{fakeSubmitQueue: q, acquired: acquired, lockErr: lockErr}
	uc.SetSubmitQueue(lq)
	return uc, lq, d
}

// Lock held elsewhere → the cycle is skipped entirely: no claims, no deploys.
func TestRunSubmitterCycle_SkipsWhenLockHeldElsewhere(t *testing.T) {
	uc, lq, d := newLockedFixture(false, nil)

	uc.runSubmitterCycle(context.Background())

	if lq.lockCalls != 1 {
		t.Fatalf("lock calls = %d, want 1", lq.lockCalls)
	}
	if len(d.deploys) != 0 {
		t.Fatalf("deploys = %v, want none (cycle must be skipped)", d.deploys)
	}
}

// Lock acquired → the cycle runs inside fn exactly once.
func TestRunSubmitterCycle_RunsUnderLock(t *testing.T) {
	uc, lq, d := newLockedFixture(true, nil)

	uc.runSubmitterCycle(context.Background())

	if lq.lockCalls != 1 {
		t.Fatalf("lock calls = %d, want 1", lq.lockCalls)
	}
	if len(d.deploys) != 1 {
		t.Fatalf("deploys = %v, want exactly one", d.deploys)
	}
}

// Lock ERROR (transient DB trouble) degrades to running unguarded:
// availability over economy — idempotency already guarantees correctness.
func TestRunSubmitterCycle_LockErrorRunsUnguarded(t *testing.T) {
	uc, _, d := newLockedFixture(false, errors.New("conn refused"))

	uc.runSubmitterCycle(context.Background())

	if len(d.deploys) != 1 {
		t.Fatalf("deploys = %v, want exactly one (unguarded fallback)", d.deploys)
	}
}

// Missing wiring (no queue / no deployer) is a silent no-op, never a panic.
func TestRunSubmitterCycle_NoQueueOrDeployerIsNoop(t *testing.T) {
	(&Usecase{}).runSubmitterCycle(context.Background())                          // both nil
	(&Usecase{submitQueue: &fakeSubmitQueue{}}).runSubmitterCycle(context.Background()) // deployer nil
}

// A failing FindSubmittableJobs aborts the cycle without touching items.
func TestRunSubmitterCycleLocked_FindErrorAborts(t *testing.T) {
	job := &models.BackfillJob{ID: "job-1", Status: "running", TemplateID: "tpl-1", TemplateVersion: 1, TotalCount: 1}
	uc, _, q, d := newSubmitterFixture(job, []models.BackfillItem{
		{ID: "item-1", JobID: "job-1", AssetID: "asset-1", Status: "pending"},
	})
	q.findErr = errors.New("db down")

	uc.runSubmitterCycle(context.Background())

	if len(d.deploys) != 0 {
		t.Fatalf("deploys = %v, want none on find error", d.deploys)
	}
}

// Queues without a lock surface (plain fakes, single-instance envs) run
// unguarded — the optional interface must never be a hard dependency.
func TestRunSubmitterCycle_NoLockerRunsDirectly(t *testing.T) {
	job := &models.BackfillJob{ID: "job-1", Status: "running", TemplateID: "tpl-1", TemplateVersion: 1, TotalCount: 1}
	uc, repo, _, d := newSubmitterFixture(job, []models.BackfillItem{
		{ID: "item-1", JobID: "job-1", AssetID: "asset-1", Status: "pending"},
	})

	uc.runSubmitterCycle(context.Background())

	if len(d.deploys) != 1 {
		t.Fatalf("deploys = %v, want exactly one", d.deploys)
	}
	if got := itemStatus(repo, "item-1"); got != "submitted" {
		t.Fatalf("item status = %q, want submitted", got)
	}
}
