package backfill

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	pipelineUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/pipeline"
)

// ── CYB-3678: governor ───────────────────────────────────────────────────────

func TestClusterGovernorFloorMath(t *testing.T) {
	if g := newClusterGovernor(20); g.floor != 5 || g.effective != 20 {
		t.Fatalf("floor=%d effective=%d, want 5/20", g.floor, g.effective)
	}
	if g := newClusterGovernor(2); g.floor != 1 {
		t.Fatalf("floor=%d, want 1 (never below 1)", g.floor)
	}
}

func TestClusterGovernorAIMD(t *testing.T) {
	g := newClusterGovernor(20)

	// Distress via a transient outcome → halve.
	g.record(10*time.Millisecond, outcomeTransient)
	g.endCycle()
	if g.slots() != 10 {
		t.Fatalf("effective = %d, want 10 (halved)", g.slots())
	}
	// Repeated distress bottoms at the floor (configured/4 = 5), never 0.
	for i := 0; i < 5; i++ {
		g.record(10*time.Millisecond, outcomeTransient)
		g.endCycle()
	}
	if g.slots() != 5 {
		t.Fatalf("effective = %d, want floor 5", g.slots())
	}
	// Health → ×1.5 growth, capped at configured.
	for i := 0; i < 10; i++ {
		g.record(10*time.Millisecond, outcomeOK)
		g.endCycle()
	}
	if g.slots() != 20 {
		t.Fatalf("effective = %d, want back to configured 20", g.slots())
	}
	// Majority-slow window is distress even without errors.
	g.record(2*time.Second, outcomeOK)
	g.record(2*time.Second, outcomeOK)
	g.record(10*time.Millisecond, outcomeOK)
	g.endCycle()
	if g.slots() != 10 {
		t.Fatalf("effective = %d, want 10 (slow-majority distress)", g.slots())
	}
	// Empty window is a no-op.
	before := g.slots()
	g.endCycle()
	if g.slots() != before {
		t.Fatalf("empty window changed effective %d → %d", before, g.slots())
	}
}

func TestGovernorForIsPerClusterAndSticky(t *testing.T) {
	uc := &Usecase{}
	a1 := uc.governorFor("a")
	b := uc.governorFor("b")
	if a1 == b {
		t.Fatal("clusters must not share a governor")
	}
	if uc.governorFor("a") != a1 {
		t.Fatal("governor must be sticky per cluster")
	}
}

// ── CYB-3678: error classification ───────────────────────────────────────────

func TestClassifySubmitError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want submitOutcome
	}{
		{"nil", nil, outcomeOK},
		{"invalid argument", fmt.Errorf("x: %w", pipelineUC.ErrInvalidArgument), outcomePermanent},
		{"template missing", fmt.Errorf("x: %w", pipelineUC.ErrTemplateNotFound), outcomePermanent},
		{"asset missing", fmt.Errorf("x: %w", pipelineUC.ErrAssetNotFound), outcomePermanent},
		{"transpile marker", errors.New("transpile: node X invalid"), outcomePermanent},
		{"resource ceiling zh", errors.New("执行目标不支持该资源规格"), outcomePermanent},
		{"timeout is transient", context.DeadlineExceeded, outcomeTransient},
		{"conn refused is transient", errors.New("dial tcp: connection refused"), outcomeTransient},
		{"429 is transient", errors.New("too many requests (429)"), outcomeTransient},
		{"unknown defaults transient", errors.New("weird"), outcomeTransient},
	}
	for _, tc := range cases {
		if got := classifySubmitError(tc.err); got != tc.want {
			t.Fatalf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}

// ── CYB-3678: attempt cap → DLQ ──────────────────────────────────────────────

// A transient submit failure leaves the item pending (retried next cycle)
// until the durable attempt cap, then dead-letters it.
func TestSubmitter_TransientRetriesThenDLQ(t *testing.T) {
	ctx := context.Background()
	job := &models.BackfillJob{ID: "job-1", Status: "running", TemplateID: "tpl-1", TemplateVersion: 1, TotalCount: 1}
	uc, repo, _, d := newSubmitterFixture(job, []models.BackfillItem{
		{ID: "item-1", JobID: "job-1", AssetID: "asset-1", Status: "pending"},
	})
	d.deployErrByAsset["asset-1"] = errors.New("dial tcp: connection refused")

	for i := 1; i < maxSubmitAttempts; i++ {
		uc.runSubmitterCycle(ctx)
		if got := itemStatus(repo, "item-1"); got != "pending" {
			t.Fatalf("cycle %d: status = %q, want pending (transient retry)", i, got)
		}
	}
	uc.runSubmitterCycle(ctx) // attempt == cap → DLQ
	if got := itemStatus(repo, "item-1"); got != "failed" {
		t.Fatalf("status after cap = %q, want failed (DLQ)", got)
	}
	// CYB-4026 D5: the failure reason must be persisted even though the item
	// already has a pipeline_run_id (the DLQ path goes through
	// MarkItemFailedWithRun, not the reason-less UpdateItemPipelineRun).
	if msg := itemError(repo, "item-1"); msg == "" || !strings.Contains(msg, "max submit attempts") {
		t.Fatalf("DLQ error_message = %q, want it to contain the original cause", msg)
	}
}

// Permanent errors dead-letter immediately — no attempt burn.
func TestSubmitter_PermanentFailsImmediately(t *testing.T) {
	job := &models.BackfillJob{ID: "job-1", Status: "running", TemplateID: "tpl-1", TemplateVersion: 1, TotalCount: 1}
	uc, repo, _, d := newSubmitterFixture(job, []models.BackfillItem{
		{ID: "item-1", JobID: "job-1", AssetID: "asset-1", Status: "pending"},
	})
	d.deployErrByAsset["asset-1"] = errors.New("transpile: node X invalid")

	uc.runSubmitterCycle(context.Background())

	if got := itemStatus(repo, "item-1"); got != "failed" {
		t.Fatalf("status = %q, want failed (permanent)", got)
	}
	repo.attemptsMu.Lock()
	defer repo.attemptsMu.Unlock()
	if repo.submitAttempts["item-1"] != 0 {
		t.Fatalf("attempts = %d, want 0 (permanent path must not burn the cap)", repo.submitAttempts["item-1"])
	}
}

// ── CYB-3678: per-cluster sharding ───────────────────────────────────────────

// clusterLockRecorder records which clusters were locked.
type clusterLockRecorder struct {
	*fakeSubmitQueue
	clusters []string
	deny     map[string]bool
}

func (q *clusterLockRecorder) WithSubmitterClusterLock(ctx context.Context, cluster string, fn func(context.Context) error) (bool, error) {
	q.mu.Lock()
	q.clusters = append(q.clusters, cluster)
	denied := q.deny[cluster]
	q.mu.Unlock()
	if denied {
		return false, nil
	}
	return true, fn(ctx)
}

func newShardedFixture(t *testing.T) (*Usecase, *pausedSyncRepo, *clusterLockRecorder, *fakeDeployer) {
	t.Helper()
	jobA := models.BackfillJob{ID: "job-a", Status: "running", TemplateID: "tpl-1", TemplateVersion: 1, TotalCount: 1,
		FilterJSON: map[string]interface{}{"target_id": "target-a"}}
	jobB := models.BackfillJob{ID: "job-b", Status: "running", TemplateID: "tpl-1", TemplateVersion: 1, TotalCount: 1,
		FilterJSON: map[string]interface{}{"target_id": "target-b"}}
	repo := &pausedSyncRepo{job: &jobA, extraJobs: []models.BackfillJob{jobB}, items: []models.BackfillItem{
		{ID: "item-a", JobID: "job-a", AssetID: "asset-a", Status: "pending"},
		{ID: "item-b", JobID: "job-b", AssetID: "asset-b", Status: "pending"},
	}}
	q := &fakeSubmitQueue{repo: repo, jobs: []models.BackfillJob{jobA, jobB}, lockedNow: map[string]bool{}}
	lq := &clusterLockRecorder{fakeSubmitQueue: q, deny: map[string]bool{}}
	d := &fakeDeployer{
		deployErrByAsset: map[string]error{},
		runsByID:         map[string]*models.PipelineRun{},
		clusterByTarget:  map[string]string{"target-a": "cluster-a", "target-b": "cluster-b"},
	}
	uc := New(repo, nil)
	uc.deployer = d
	uc.SetSubmitQueue(lq)
	return uc, repo, lq, d
}

// Jobs shard to their clusters: both channels run, each under ITS OWN lock.
func TestRunSubmitterCycle_ShardsByCluster(t *testing.T) {
	uc, repo, lq, d := newShardedFixture(t)

	uc.runSubmitterCycle(context.Background())

	got := map[string]bool{}
	for _, c := range lq.clusters {
		got[c] = true
	}
	if !got["cluster-a"] || !got["cluster-b"] {
		t.Fatalf("locked clusters = %v, want both cluster-a and cluster-b", lq.clusters)
	}
	if len(d.deploys) != 2 {
		t.Fatalf("deploys = %v, want both assets", d.deploys)
	}
	if itemStatus(repo, "item-a") != "submitted" || itemStatus(repo, "item-b") != "submitted" {
		t.Fatal("both items must be submitted")
	}
}

// A cluster whose lock is held elsewhere is skipped WITHOUT affecting the
// other cluster — the isolation the sharding exists for.
func TestRunSubmitterCycle_ClusterIsolation(t *testing.T) {
	uc, repo, lq, d := newShardedFixture(t)
	lq.deny["cluster-b"] = true

	uc.runSubmitterCycle(context.Background())

	if len(d.deploys) != 1 || d.deploys[0] != "asset-a" {
		t.Fatalf("deploys = %v, want only asset-a (cluster-b held elsewhere)", d.deploys)
	}
	if itemStatus(repo, "item-a") != "submitted" {
		t.Fatal("cluster-a must proceed while cluster-b is held")
	}
	if itemStatus(repo, "item-b") != "pending" {
		t.Fatal("cluster-b item must stay pending for its holder")
	}
}

func TestRunSubmitterCycle_TargetFairnessPreventsBackpressureStarvation(t *testing.T) {
	old := perJobSubmitBatch
	perJobSubmitBatch = 1
	defer func() { perJobSubmitBatch = old }()

	jobs := make([]models.BackfillJob, 0, 26)
	items := make([]models.BackfillItem, 0, 26)
	for i := 0; i < 25; i++ {
		jobID := fmt.Sprintf("job-a-%02d", i)
		assetID := fmt.Sprintf("asset-a-%02d", i)
		jobs = append(jobs, models.BackfillJob{
			ID: jobID, Status: "running", TemplateID: "tpl-1", TemplateVersion: 1, TotalCount: 1,
			FilterJSON: map[string]interface{}{"target_id": "target-a"},
		})
		items = append(items, models.BackfillItem{ID: "item-" + assetID, JobID: jobID, AssetID: assetID, Status: "pending"})
	}
	jobs = append(jobs, models.BackfillJob{
		ID: "job-b-00", Status: "running", TemplateID: "tpl-1", TemplateVersion: 1, TotalCount: 1,
		FilterJSON: map[string]interface{}{"target_id": "target-b"},
	})
	items = append(items, models.BackfillItem{ID: "item-asset-b-00", JobID: "job-b-00", AssetID: "asset-b-00", Status: "pending"})

	repo := &pausedSyncRepo{job: &jobs[0], extraJobs: jobs[1:], items: items}
	q := &fakeSubmitQueue{repo: repo, jobs: jobs, lockedNow: map[string]bool{}}
	d := &fakeDeployer{
		deployErrByAsset:  map[string]error{},
		runsByID:          map[string]*models.PipelineRun{},
		clusterByTarget:   map[string]string{"target-a": "cluster-a", "target-b": "cluster-b"},
		nsByTarget:        map[string]string{"target-a": "ns-a", "target-b": "ns-b"},
		maxActiveByTarget: map[string]int{"target-a": 100, "target-b": 100},
		activeWFByKey:     map[string]int{"cluster-a/ns-a": 150, "cluster-b/ns-b": 0},
	}
	uc := New(repo, nil)
	uc.deployer = d
	uc.SetSubmitQueue(q)

	uc.runSubmitterCycle(context.Background())

	if itemStatus(repo, "item-asset-b-00") != "submitted" {
		t.Fatalf("healthy target-b item status = %q, want submitted", itemStatus(repo, "item-asset-b-00"))
	}
	for i := 0; i < 25; i++ {
		if got := itemStatus(repo, fmt.Sprintf("item-asset-a-%02d", i)); got != "pending" {
			t.Fatalf("backpressured target-a item %02d status = %q, want pending", i, got)
		}
	}
}

// CYB-3681 review: two clusters that share a namespace NAME (e.g. both use
// "argo") must not read each other's active-workflow count. Cluster-a's "argo"
// is saturated (150 ≥ 100) while cluster-b's "argo" is idle (0). With the old
// namespace-only key both collided on "argo" and the last watcher write won —
// either wedging the idle cluster or over-dispatching into the saturated one.
// The composite (cluster, namespace) key keeps the two observations distinct,
// so cluster-a defers while cluster-b proceeds.
func TestRunSubmitterCycle_BackpressureKeyIsolatesSharedNamespaceAcrossClusters(t *testing.T) {
	old := perJobSubmitBatch
	perJobSubmitBatch = 1
	defer func() { perJobSubmitBatch = old }()

	jobs := []models.BackfillJob{
		{
			ID: "job-a", Status: "running", TemplateID: "tpl-1", TemplateVersion: 1, TotalCount: 1,
			FilterJSON: map[string]interface{}{"target_id": "target-a"},
		},
		{
			ID: "job-b", Status: "running", TemplateID: "tpl-1", TemplateVersion: 1, TotalCount: 1,
			FilterJSON: map[string]interface{}{"target_id": "target-b"},
		},
	}
	items := []models.BackfillItem{
		{ID: "item-a", JobID: "job-a", AssetID: "asset-a", Status: "pending"},
		{ID: "item-b", JobID: "job-b", AssetID: "asset-b", Status: "pending"},
	}

	repo := &pausedSyncRepo{job: &jobs[0], extraJobs: jobs[1:], items: items}
	q := &fakeSubmitQueue{repo: repo, jobs: jobs, lockedNow: map[string]bool{}}
	d := &fakeDeployer{
		deployErrByAsset:  map[string]error{},
		runsByID:          map[string]*models.PipelineRun{},
		clusterByTarget:   map[string]string{"target-a": "cluster-a", "target-b": "cluster-b"},
		nsByTarget:        map[string]string{"target-a": "argo", "target-b": "argo"},
		maxActiveByTarget: map[string]int{"target-a": 100, "target-b": 100},
		activeWFByKey:     map[string]int{"cluster-a/argo": 150, "cluster-b/argo": 0},
	}
	uc := New(repo, nil)
	uc.deployer = d
	uc.SetSubmitQueue(q)

	uc.runSubmitterCycle(context.Background())

	if got := itemStatus(repo, "item-a"); got != "pending" {
		t.Fatalf("cluster-a item status = %q, want pending (its own argo is saturated)", got)
	}
	if got := itemStatus(repo, "item-b"); got != "submitted" {
		t.Fatalf("cluster-b item status = %q, want submitted (its own argo is idle, not cluster-a's count)", got)
	}
}

// ── CYB-3678: self-kick ──────────────────────────────────────────────────────

// A channel that fills its whole batch (backlog remains) self-kicks instead
// of idling until the next tick.
func TestRunSubmitterCycle_SelfKickOnFullBatch(t *testing.T) {
	old := perJobSubmitBatch
	perJobSubmitBatch = 2
	defer func() { perJobSubmitBatch = old }()

	job := &models.BackfillJob{ID: "job-1", Status: "running", TemplateID: "tpl-1", TemplateVersion: 1, TotalCount: 3}
	uc, _, _, _ := newSubmitterFixture(job, []models.BackfillItem{
		{ID: "i1", JobID: "job-1", AssetID: "a1", Status: "pending"},
		{ID: "i2", JobID: "job-1", AssetID: "a2", Status: "pending"},
		{ID: "i3", JobID: "job-1", AssetID: "a3", Status: "pending"},
	})
	uc.submitKick = make(chan struct{}, 1)

	uc.runSubmitterCycle(context.Background()) // submits 2 of 3 → full batch

	select {
	case <-uc.submitKick:
	default:
		t.Fatal("full batch with backlog must self-kick")
	}
}

// A partial batch (backlog drained) must NOT self-kick.
func TestRunSubmitterCycle_NoSelfKickWhenDrained(t *testing.T) {
	job := &models.BackfillJob{ID: "job-1", Status: "running", TemplateID: "tpl-1", TemplateVersion: 1, TotalCount: 1}
	uc, _, _, _ := newSubmitterFixture(job, []models.BackfillItem{
		{ID: "i1", JobID: "job-1", AssetID: "a1", Status: "pending"},
	})
	uc.submitKick = make(chan struct{}, 1)

	uc.runSubmitterCycle(context.Background())

	select {
	case <-uc.submitKick:
		t.Fatal("drained backlog must not self-kick")
	default:
	}
}

// ── CYB-3678: channel breaker ────────────────────────────────────────────────

// Consecutive transient failures ≥ threshold trip the channel: the rest of
// the cycle is skipped (items stay pending) and no self-kick fires.
func TestRunClusterChannel_BreakerTripsOnConsecutiveTransient(t *testing.T) {
	old := perJobSubmitBatch
	perJobSubmitBatch = consecutiveTransientBreaker + 2
	defer func() { perJobSubmitBatch = old }()

	items := make([]models.BackfillItem, 0, consecutiveTransientBreaker+2)
	errs := map[string]error{}
	for i := 0; i < consecutiveTransientBreaker+2; i++ {
		id := fmt.Sprintf("i%02d", i)
		asset := fmt.Sprintf("a%02d", i)
		items = append(items, models.BackfillItem{ID: id, JobID: "job-1", AssetID: asset, Status: "pending"})
		errs[asset] = errors.New("dial tcp: connection refused") // all transient
	}
	job := &models.BackfillJob{ID: "job-1", Status: "running", TemplateID: "tpl-1", TemplateVersion: 1, TotalCount: len(items)}
	uc, repo, _, d := newSubmitterFixture(job, items)
	d.deployErrByAsset = errs
	uc.submitKick = make(chan struct{}, 1)

	uc.runSubmitterCycle(context.Background())

	// Every attempted item stays pending (transient), and the breaker
	// suppressed the self-kick a full-batch would otherwise fire.
	for _, it := range items {
		if got := itemStatus(repo, it.ID); got != "pending" {
			t.Fatalf("%s status = %q, want pending", it.ID, got)
		}
	}
	select {
	case <-uc.submitKick:
		t.Fatal("tripped breaker must suppress the self-kick")
	default:
	}
}

// A cancelled context stops token acquisition mid-batch: remaining items
// stay pending for the next cycle (durable, never lost).
func TestSubmitJobBatch_CancelledContextStops(t *testing.T) {
	job := &models.BackfillJob{ID: "job-1", Status: "running", TemplateID: "tpl-1", TemplateVersion: 1, TotalCount: 2}
	uc, repo, _, _ := newSubmitterFixture(job, []models.BackfillItem{
		{ID: "i1", JobID: "job-1", AssetID: "a1", Status: "pending"},
		{ID: "i2", JobID: "job-1", AssetID: "a2", Status: "pending"},
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	attempts, _, _ := uc.submitJobBatch(ctx, job, uc.governorFor("default"))
	if attempts != 0 {
		t.Fatalf("attempts = %d, want 0 under cancelled ctx", attempts)
	}
	if itemStatus(repo, "i1") != "pending" || itemStatus(repo, "i2") != "pending" {
		t.Fatal("items must stay pending when the cycle is cancelled")
	}
}
