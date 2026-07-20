package pipeline

import (
	"context"
	"errors"
	"testing"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// ── CYB-3681: bulk-pull writeback ────────────────────────────────────────────

func TestWatcherBulkModeEnabled(t *testing.T) {
	t.Setenv(watcherModeEnv, "")
	if !watcherBulkModeEnabled() {
		t.Fatal("default must be bulk")
	}
	t.Setenv(watcherModeEnv, "LEGACY")
	if watcherBulkModeEnabled() {
		t.Fatal("legacy (case-insensitive) must disable bulk")
	}
	t.Setenv(watcherModeEnv, "bulk")
	if !watcherBulkModeEnabled() {
		t.Fatal("explicit bulk must enable bulk")
	}
}

func TestMarkWorkflowApplied_RVGate(t *testing.T) {
	uc := &Usecase{}
	if !uc.markWorkflowApplied("run-1", "100") {
		t.Fatal("first observation must apply")
	}
	if uc.markWorkflowApplied("run-1", "100") {
		t.Fatal("same RV must be gated")
	}
	if !uc.markWorkflowApplied("run-1", "101") {
		t.Fatal("new RV must apply")
	}
	// Empty RV can never gate (no basis for "unchanged").
	if !uc.markWorkflowApplied("run-2", "") || !uc.markWorkflowApplied("run-2", "") {
		t.Fatal("empty RV must always apply")
	}
	uc.forgetWorkflowApplied("run-1")
	if !uc.markWorkflowApplied("run-1", "101") {
		t.Fatal("forgotten run must re-apply")
	}
}

// bulkFixture wires a Usecase with runRepo + wfClient and one active run per
// given workflow name, all on the default cluster / "test-ns" namespace.
func bulkFixture(t *testing.T, wfNames ...string) (*Usecase, *mockRunRepo, *mockWorkflowClient, []models.PipelineRun) {
	t.Helper()
	runRepo := &mockRunRepo{}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, newMockAssetRepo(), nil, "test-ns")
	uc.SetRunRepositories(nil, runRepo, &mockRunNodeRepo{})
	client := &mockWorkflowClient{}
	uc.wfClient = client
	runs := make([]models.PipelineRun, 0, len(wfNames))
	for i, name := range wfNames {
		run := models.PipelineRun{
			ID:           "run-" + name,
			Status:       "Running",
			WorkflowName: name,
			PipelineName: "p",
		}
		_ = runRepo.Save(context.Background(), &run)
		runs = append(runs, run)
		_ = i
	}
	return uc, runRepo, client, runs
}

func activeWorkflow(name, rv string, phase wfv1.WorkflowPhase) wfv1.Workflow {
	return wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "test-ns", UID: types.UID("uid-" + name), ResourceVersion: rv},
		Status:     wfv1.WorkflowStatus{Phase: phase},
	}
}

func TestMergeRunsByID(t *testing.T) {
	a := []models.PipelineRun{{ID: "1"}, {ID: "2"}, {ID: "1"}}
	b := []models.PipelineRun{{ID: "2"}, {ID: "3"}}
	got := mergeRunsByID(a, b)
	var ids []string
	for _, r := range got {
		ids = append(ids, r.ID)
	}
	// a's dup ("1") dropped within a; b's overlap ("2") dropped; "3" appended.
	want := []string{"1", "2", "3"}
	if len(ids) != len(want) {
		t.Fatalf("ids = %v, want %v", ids, want)
	}
	for i := range want {
		if ids[i] != want[i] {
			t.Fatalf("ids = %v, want %v", ids, want)
		}
	}
	if got := mergeRunsByID(nil, nil); len(got) != 0 {
		t.Fatalf("empty merge = %v, want []", got)
	}
}

// Load-test regression (CYB-3681): a pile of just-COMPLETED recent runs must
// not crowd an older still-active run out of the watcher's candidate set. The
// candidate loader is status-based (FindActiveRunSummaries), so completed runs
// never consume the budget and the active run is always refreshed.
func TestSyncActiveRunEvents_ActiveNotStarvedByCompletedBurst(t *testing.T) {
	t.Setenv(watcherModeEnv, "")
	runRepo := &mockRunRepo{}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, newMockAssetRepo(), nil, "test-ns")
	uc.SetRunRepositories(nil, runRepo, &mockRunNodeRepo{})
	uc.SetRunEventRepo(&mockRunEventRepo{})
	client := &mockWorkflowClient{}
	uc.wfClient = client

	// 600 already-completed runs (the "recent burst") + 1 older active run.
	for i := 0; i < 600; i++ {
		_ = runRepo.Save(context.Background(), &models.PipelineRun{
			ID: "done-" + string(rune(i)), Status: "Succeeded", WorkflowName: "done-wf", PipelineName: "p",
		})
	}
	_ = runRepo.Save(context.Background(), &models.PipelineRun{
		ID: "run-active", Status: "Running", WorkflowName: "wf-active", PipelineName: "p",
	})
	client.listWorkflowsFn = func(context.Context, string, string) ([]wfv1.Workflow, error) {
		return []wfv1.Workflow{activeWorkflow("wf-active", "7", wfv1.WorkflowSucceeded)}, nil
	}

	if _, err := uc.SyncActiveRunEvents(context.Background(), 50); err != nil {
		t.Fatalf("SyncActiveRunEvents: %v", err)
	}
	if got := runRepo.byID["run-active"].Status; got != "Succeeded" {
		t.Fatalf("active run status = %q, want Succeeded (must not be starved by the completed burst)", got)
	}
}

// Listed workflows are applied from the snapshot with ZERO targeted GETs; an
// unchanged snapshot next tick applies nothing (RV gate).
func TestBulkSync_AppliesListedAndGatesUnchanged(t *testing.T) {
	uc, runRepo, client, runs := bulkFixture(t, "wf-a")
	gets := 0
	client.getWorkflowFn = func(context.Context, string, string) (*wfv1.Workflow, error) {
		gets++
		return nil, errors.New("must not GET")
	}
	client.listWorkflowsFn = func(_ context.Context, ns, sel string) ([]wfv1.Workflow, error) {
		if ns != "test-ns" || sel != activeWorkflowSelector {
			t.Fatalf("list ns=%q sel=%q", ns, sel)
		}
		return []wfv1.Workflow{activeWorkflow("wf-a", "7", wfv1.WorkflowRunning)}, nil
	}

	if n := uc.bulkSyncActiveRuns(context.Background(), runs, []int{0}, 50, false); n != 1 {
		t.Fatalf("synced = %d, want 1", n)
	}
	if gets != 0 {
		t.Fatalf("targeted GETs = %d, want 0", gets)
	}
	if got := runRepo.byID["run-wf-a"].Status; got != "Running" {
		t.Fatalf("status = %q", got)
	}
	// Second tick, same RV → gated.
	if n := uc.bulkSyncActiveRuns(context.Background(), runs, []int{0}, 50, false); n != 0 {
		t.Fatalf("synced = %d, want 0 (RV gate)", n)
	}
	// Recalibration bypasses the gate.
	if n := uc.bulkSyncActiveRuns(context.Background(), runs, []int{0}, 50, true); n != 1 {
		t.Fatalf("synced = %d, want 1 (recalibrate)", n)
	}
}

// A workflow that reaches terminal phase in the snapshot drops out of the RV
// gate map (no unbounded memory on churn).
func TestBulkSync_TerminalApplyForgetsRV(t *testing.T) {
	uc, runRepo, client, runs := bulkFixture(t, "wf-a")
	client.listWorkflowsFn = func(context.Context, string, string) ([]wfv1.Workflow, error) {
		return []wfv1.Workflow{activeWorkflow("wf-a", "9", wfv1.WorkflowSucceeded)}, nil
	}
	if n := uc.bulkSyncActiveRuns(context.Background(), runs, []int{0}, 50, false); n != 1 {
		t.Fatalf("synced = %d, want 1", n)
	}
	if got := runRepo.byID["run-wf-a"].Status; got != "Succeeded" {
		t.Fatalf("status = %q, want Succeeded", got)
	}
	uc.watcherRVMu.Lock()
	_, still := uc.watcherAppliedRV["run-wf-a"]
	uc.watcherRVMu.Unlock()
	if still {
		t.Fatal("terminal run must be evicted from the RV gate")
	}
}

// Runs absent from the snapshot fall to a bounded targeted GET: a terminal
// workflow still readable via GET is applied.
func TestBulkSync_AbsentRunProbedViaGet(t *testing.T) {
	uc, runRepo, client, runs := bulkFixture(t, "wf-gone")
	client.listWorkflowsFn = func(context.Context, string, string) ([]wfv1.Workflow, error) {
		return nil, nil // empty active snapshot
	}
	client.getWorkflowFn = func(_ context.Context, name, ns string) (*wfv1.Workflow, error) {
		wf := activeWorkflow(name, "3", wfv1.WorkflowFailed)
		wf.Namespace = ns
		return &wf, nil
	}
	if n := uc.bulkSyncActiveRuns(context.Background(), runs, []int{0}, 50, false); n != 1 {
		t.Fatalf("synced = %d, want 1", n)
	}
	if got := runRepo.byID["run-wf-gone"].Status; got != "Failed" {
		t.Fatalf("status = %q, want Failed", got)
	}
}

// The residual GET budget bounds snapshot-absent probes per tick.
func TestBulkSync_ResidualLimitBoundsProbes(t *testing.T) {
	uc, _, client, runs := bulkFixture(t, "wf-1", "wf-2", "wf-3")
	client.listWorkflowsFn = func(context.Context, string, string) ([]wfv1.Workflow, error) { return nil, nil }
	gets := 0
	client.getWorkflowFn = func(_ context.Context, name, ns string) (*wfv1.Workflow, error) {
		gets++
		wf := activeWorkflow(name, "1", wfv1.WorkflowRunning)
		return &wf, nil
	}
	if n := uc.bulkSyncActiveRuns(context.Background(), runs, []int{0, 1, 2}, 2, false); n != 2 {
		t.Fatalf("synced = %d, want 2 (residual budget)", n)
	}
	if gets != 2 {
		t.Fatalf("gets = %d, want 2", gets)
	}
}

// A failed LIST must not orphan-probe the whole namespace: the cluster
// degrades to the bounded per-run path for the tick.
func TestBulkSync_ListErrorFallsBackBounded(t *testing.T) {
	uc, runRepo, client, runs := bulkFixture(t, "wf-1", "wf-2")
	client.listWorkflowsFn = func(context.Context, string, string) ([]wfv1.Workflow, error) {
		return nil, errors.New("apiserver 500")
	}
	gets := 0
	client.getWorkflowFn = func(_ context.Context, name, ns string) (*wfv1.Workflow, error) {
		gets++
		wf := activeWorkflow(name, "1", wfv1.WorkflowRunning)
		return &wf, nil
	}
	if n := uc.bulkSyncActiveRuns(context.Background(), runs, []int{0, 1}, 1, false); n != 1 {
		t.Fatalf("synced = %d, want 1 (bounded fallback)", n)
	}
	if gets != 1 {
		t.Fatalf("gets = %d, want 1", gets)
	}
	if got := runRepo.byID["run-wf-1"].Status; got != "Running" {
		t.Fatalf("status = %q", got)
	}
}

// A resolve-client error degrades to the bounded per-run fallback (which
// itself no-ops without a usable client) — never a panic, never a mass orphan.
func TestBulkSync_ResolveErrorFallsBack(t *testing.T) {
	uc, _, _, runs := bulkFixture(t, "wf-1")
	uc.SetArgoFactory(&stubArgoFactory{forced: true, err: errors.New("kubeconfig broken")})
	if n := uc.bulkSyncActiveRuns(context.Background(), runs, []int{0}, 50, false); n != 1 {
		t.Fatalf("synced = %d, want 1 (fallback attempted)", n)
	}
}

// A run with an empty workflow name can never match the snapshot; it takes
// the residual probe path (refreshPipelineRunStatus no-ops on empty names).
func TestBulkSync_EmptyWorkflowNameGoesResidual(t *testing.T) {
	uc, runRepo, client, _ := bulkFixture(t)
	run := models.PipelineRun{ID: "r-noname", Status: "Running", WorkflowName: ""}
	_ = runRepo.Save(context.Background(), &run)
	client.listWorkflowsFn = func(context.Context, string, string) ([]wfv1.Workflow, error) {
		return []wfv1.Workflow{activeWorkflow("", "1", wfv1.WorkflowRunning)}, nil
	}
	if n := uc.bulkSyncActiveRuns(context.Background(), []models.PipelineRun{run}, []int{0}, 50, false); n != 1 {
		t.Fatalf("synced = %d, want 1 (residual probe)", n)
	}
	if got := runRepo.byID["r-noname"].Status; got != "Running" {
		t.Fatalf("status = %q, must be untouched", got)
	}
}

// Empty active set is a no-op — at both levels (grouping never produces an
// empty group, but the cluster body guards it anyway).
func TestBulkSync_EmptyActiveSet(t *testing.T) {
	uc, _, _, runs := bulkFixture(t, "wf-1")
	if n := uc.bulkSyncActiveRuns(context.Background(), runs, nil, 50, false); n != 0 {
		t.Fatalf("synced = %d, want 0", n)
	}
	if n := uc.bulkSyncClusterRuns(context.Background(), runs, nil, 50, false); n != 0 {
		t.Fatalf("cluster synced = %d, want 0", n)
	}
}

// No client at all (unwired test/deploy shapes) → bounded fallback path where
// refreshPipelineRunStatus itself no-ops; never a panic.
func TestBulkSync_NilClientNoop(t *testing.T) {
	runRepo := &mockRunRepo{}
	uc := New(&mockTemplateRepo{}, &mockDeploymentRepo{}, newMockAssetRepo(), nil, "test-ns")
	uc.SetRunRepositories(nil, runRepo, &mockRunNodeRepo{})
	run := models.PipelineRun{ID: "r1", Status: "Running", WorkflowName: "wf-x"}
	_ = runRepo.Save(context.Background(), &run)
	if n := uc.bulkSyncActiveRuns(context.Background(), []models.PipelineRun{run}, []int{0}, 50, false); n != 1 {
		t.Fatalf("synced = %d, want 1 (fallback attempts, refresh no-ops)", n)
	}
}

// End-to-end through SyncActiveRunEvents in bulk mode: the active run is
// refreshed from the LIST snapshot, watcher state is persisted, and no
// rotating-window GETs happen.
func TestSyncActiveRunEvents_BulkMode(t *testing.T) {
	t.Setenv(watcherModeEnv, "")
	uc, runRepo, client, _ := bulkFixture(t, "wf-a")
	uc.SetRunEventRepo(&mockRunEventRepo{})
	lists := 0
	client.listWorkflowsFn = func(context.Context, string, string) ([]wfv1.Workflow, error) {
		lists++
		return []wfv1.Workflow{activeWorkflow("wf-a", "42", wfv1.WorkflowSucceeded)}, nil
	}
	// Targeted GETs still happen post-terminal (ledger backfill section) —
	// the bulk contract is only that the ACTIVE refresh came from the LIST.
	client.getWorkflowFn = func(_ context.Context, name, ns string) (*wfv1.Workflow, error) {
		wf := activeWorkflow(name, "42", wfv1.WorkflowSucceeded)
		return &wf, nil
	}
	synced, err := uc.SyncActiveRunEvents(context.Background(), 50)
	if err != nil {
		t.Fatalf("SyncActiveRunEvents: %v", err)
	}
	if synced < 1 {
		t.Fatalf("synced = %d, want ≥1", synced)
	}
	if lists != 1 {
		t.Fatalf("lists = %d, want 1 (active refresh must come from the LIST)", lists)
	}
	if got := runRepo.byID["run-wf-a"].Status; got != "Succeeded" {
		t.Fatalf("status = %q, want Succeeded", got)
	}
}

// Legacy mode still takes the rotating-window per-run GET path.
func TestSyncActiveRunEvents_LegacyMode(t *testing.T) {
	t.Setenv(watcherModeEnv, "legacy")
	uc, runRepo, client, _ := bulkFixture(t, "wf-a")
	uc.SetRunEventRepo(&mockRunEventRepo{})
	lists := 0
	client.listWorkflowsFn = func(context.Context, string, string) ([]wfv1.Workflow, error) {
		lists++
		return nil, nil
	}
	client.getWorkflowFn = func(_ context.Context, name, ns string) (*wfv1.Workflow, error) {
		wf := activeWorkflow(name, "1", wfv1.WorkflowSucceeded)
		return &wf, nil
	}
	if _, err := uc.SyncActiveRunEvents(context.Background(), 50); err != nil {
		t.Fatalf("SyncActiveRunEvents: %v", err)
	}
	if lists != 0 {
		t.Fatalf("lists = %d, want 0 in legacy mode", lists)
	}
	if got := runRepo.byID["run-wf-a"].Status; got != "Succeeded" {
		t.Fatalf("status = %q, want Succeeded", got)
	}
}
