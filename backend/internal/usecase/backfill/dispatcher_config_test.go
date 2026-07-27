package backfill

import (
	"context"
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"golang.org/x/time/rate"

	"github.com/CyberOrigin2077/cyber-databrew/internal/metrics"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// ── CYB-3679: online dispatcher tuning ──────────────────────────────────────

type fakeDispatcherStore struct {
	rows      []models.DispatcherConfig
	listErr   error
	upserts   []models.DispatcherConfig
	upsertErr error
	deletes   []string
	deleteErr error
}

func (f *fakeDispatcherStore) List(context.Context) ([]models.DispatcherConfig, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.rows, nil
}
func (f *fakeDispatcherStore) Upsert(_ context.Context, cfg *models.DispatcherConfig) error {
	if f.upsertErr != nil {
		return f.upsertErr
	}
	f.upserts = append(f.upserts, *cfg)
	return nil
}
func (f *fakeDispatcherStore) Delete(_ context.Context, cluster string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	f.deletes = append(f.deletes, cluster)
	return nil
}

func validCfg(cluster string) *models.DispatcherConfig {
	return &models.DispatcherConfig{ClusterID: cluster, MaxConcurrency: 32, SubmitBatch: 25, RatePerSec: 5}
}

func TestValidateDispatcherConfig(t *testing.T) {
	cases := []struct {
		name string
		mut  func(*models.DispatcherConfig)
		ok   bool
	}{
		{"valid", func(*models.DispatcherConfig) {}, true},
		{"nil cluster", func(c *models.DispatcherConfig) { c.ClusterID = "" }, false},
		{"concurrency low", func(c *models.DispatcherConfig) { c.MaxConcurrency = 0 }, false},
		{"concurrency high", func(c *models.DispatcherConfig) { c.MaxConcurrency = 257 }, false},
		{"batch low", func(c *models.DispatcherConfig) { c.SubmitBatch = 0 }, false},
		{"batch high", func(c *models.DispatcherConfig) { c.SubmitBatch = 201 }, false},
		{"rate low", func(c *models.DispatcherConfig) { c.RatePerSec = 0.05 }, false},
		{"rate high", func(c *models.DispatcherConfig) { c.RatePerSec = 101 }, false},
		{"boundaries", func(c *models.DispatcherConfig) { c.MaxConcurrency = 256; c.SubmitBatch = 200; c.RatePerSec = 100 }, true},
	}
	for _, tc := range cases {
		cfg := validCfg("clu-a")
		tc.mut(cfg)
		err := validateDispatcherConfig(cfg)
		if tc.ok && err != nil {
			t.Fatalf("%s: unexpected err %v", tc.name, err)
		}
		if !tc.ok && !errors.Is(err, ErrInvalidDispatcherConfig) {
			t.Fatalf("%s: err = %v, want ErrInvalidDispatcherConfig", tc.name, err)
		}
	}
	if err := validateDispatcherConfig(nil); !errors.Is(err, ErrInvalidDispatcherConfig) {
		t.Fatalf("nil cfg: err = %v", err)
	}
}

// A refresh failure must keep the previous snapshot: a DB blip cannot
// silently un-pause a cluster.
func TestRefreshDispatcherConfigs_ErrorKeepsSnapshot(t *testing.T) {
	store := &fakeDispatcherStore{rows: []models.DispatcherConfig{{ClusterID: "clu-a", Paused: true}}}
	uc := &Usecase{}
	uc.SetDispatcherConfigRepo(store)

	uc.refreshDispatcherConfigs(context.Background())
	if cfg, ok := uc.dispatcherConfigFor("clu-a"); !ok || !cfg.Paused {
		t.Fatalf("cfg = %+v ok=%v, want paused row", cfg, ok)
	}

	store.listErr = errors.New("db down")
	uc.refreshDispatcherConfigs(context.Background())
	if cfg, ok := uc.dispatcherConfigFor("clu-a"); !ok || !cfg.Paused {
		t.Fatalf("cfg = %+v ok=%v — snapshot must survive a failed refresh", cfg, ok)
	}
}

// No repo wired → refresh and lookups are silent no-ops.
func TestDispatcherConfig_NoRepoNoop(t *testing.T) {
	uc := &Usecase{}
	uc.refreshDispatcherConfigs(context.Background())
	if _, ok := uc.dispatcherConfigFor("clu-a"); ok {
		t.Fatal("no repo must yield no config")
	}
	if err := uc.SaveDispatcherConfig(context.Background(), validCfg("clu-a")); err == nil {
		t.Fatal("save without repo must error")
	}
	if err := uc.DeleteDispatcherConfig(context.Background(), "clu-a"); err == nil {
		t.Fatal("delete without repo must error")
	}
}

func TestSaveDispatcherConfig(t *testing.T) {
	store := &fakeDispatcherStore{}
	uc := &Usecase{}
	uc.SetDispatcherConfigRepo(store)

	// Validation rejects before touching the store.
	bad := validCfg("clu-a")
	bad.MaxConcurrency = 0
	if err := uc.SaveDispatcherConfig(context.Background(), bad); !errors.Is(err, ErrInvalidDispatcherConfig) {
		t.Fatalf("err = %v", err)
	}
	if len(store.upserts) != 0 {
		t.Fatal("invalid config must not reach the store")
	}

	// Upsert error propagates.
	store.upsertErr = errors.New("db down")
	if err := uc.SaveDispatcherConfig(context.Background(), validCfg("clu-a")); err == nil {
		t.Fatal("want upsert error")
	}

	// Success persists AND refreshes the snapshot.
	store.upsertErr = nil
	store.rows = []models.DispatcherConfig{*validCfg("clu-a")}
	if err := uc.SaveDispatcherConfig(context.Background(), validCfg("clu-a")); err != nil {
		t.Fatalf("save: %v", err)
	}
	if len(store.upserts) != 1 {
		t.Fatalf("upserts = %d, want 1", len(store.upserts))
	}
	if _, ok := uc.dispatcherConfigFor("clu-a"); !ok {
		t.Fatal("snapshot must contain the saved row")
	}
}

func TestDeleteDispatcherConfig(t *testing.T) {
	store := &fakeDispatcherStore{rows: []models.DispatcherConfig{*validCfg("clu-a")}}
	uc := &Usecase{}
	uc.SetDispatcherConfigRepo(store)
	uc.refreshDispatcherConfigs(context.Background())

	if err := uc.DeleteDispatcherConfig(context.Background(), ""); !errors.Is(err, ErrInvalidDispatcherConfig) {
		t.Fatalf("empty cluster err = %v", err)
	}
	store.deleteErr = errors.New("db down")
	if err := uc.DeleteDispatcherConfig(context.Background(), "clu-a"); err == nil {
		t.Fatal("want delete error")
	}
	store.deleteErr = nil
	store.rows = nil
	if err := uc.DeleteDispatcherConfig(context.Background(), "clu-a"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, ok := uc.dispatcherConfigFor("clu-a"); ok {
		t.Fatal("snapshot must drop the deleted row")
	}
}

// applyConfig retunes concurrency/floor/rate/batch and clamps effective.
func TestGovernorApplyConfig(t *testing.T) {
	g := newClusterGovernor(20)
	g.applyConfig(8, 5, 30)
	if g.configured != 8 || g.floor != 2 {
		t.Fatalf("configured=%d floor=%d, want 8/2", g.configured, g.floor)
	}
	if g.effective != 8 {
		t.Fatalf("effective = %d, want clamped to 8", g.effective)
	}
	if g.limiter.Limit() != 5 || g.limiter.Burst() != 10 {
		t.Fatalf("limiter = %v/%d, want 5/10", g.limiter.Limit(), g.limiter.Burst())
	}
	if g.submitBatchLimit() != 30 {
		t.Fatalf("batchLimit = %d, want 30", g.submitBatchLimit())
	}

	// Raising the ceiling lifts the floor clamp; effective stays (AIMD grows it).
	g.effective = 1
	g.applyConfig(40, 5, 0)
	if g.floor != 10 || g.effective != 10 {
		t.Fatalf("floor=%d effective=%d, want 10/10 (clamped up to floor)", g.floor, g.effective)
	}
	// Tiny ceilings keep a floor of at least 1.
	g.applyConfig(3, 5, 0)
	if g.floor != 1 {
		t.Fatalf("floor = %d, want 1", g.floor)
	}
	g.applyConfig(40, 5, 0)
	g.effective = 1
	// Same values are a no-op (idempotent).
	g.applyConfig(40, 5, 0)
	if g.configured != 40 {
		t.Fatalf("configured = %d", g.configured)
	}
	// Zero values leave concurrency/rate untouched.
	g.applyConfig(0, 0, 0)
	if g.configured != 40 || g.limiter.Limit() != 5 {
		t.Fatal("zero config values must not reset tuning")
	}
}

// CYB-4026 D4: burst must never be 0 — a legal slow rate (0.1–0.9/s) with
// burst=0 makes limiter.Wait fail immediately and stalls the whole cluster.
func TestBurstForRate_NeverZero(t *testing.T) {
	cases := []struct {
		rate float64
		want int
	}{
		{0.1, 1}, {0.5, 1}, {0.9, 2}, {1, 2}, {5, 10}, {10, 20},
	}
	for _, tc := range cases {
		if got := burstForRate(tc.rate); got != tc.want {
			t.Errorf("burstForRate(%v) = %d, want %d", tc.rate, got, tc.want)
		}
		if burstForRate(tc.rate) < 1 {
			t.Errorf("burstForRate(%v) < 1 — cluster would stall", tc.rate)
		}
	}
}

// CYB-4026 D4: applyConfig with a sub-1 rate keeps burst >= 1 (online-tuning
// path), and the default constructor's burst is >= 1 too.
func TestGovernorBurstFloor(t *testing.T) {
	g := newClusterGovernor(8)
	if g.limiter.Burst() < 1 {
		t.Fatalf("constructor burst = %d, want >= 1", g.limiter.Burst())
	}
	g.applyConfig(8, 0.5, 0)
	if g.limiter.Burst() < 1 {
		t.Fatalf("applyConfig(rate=0.5) burst = %d, want >= 1", g.limiter.Burst())
	}
	if g.limiter.Limit() != rate.Limit(0.5) {
		t.Fatalf("limit = %v, want 0.5", g.limiter.Limit())
	}
}

// A paused cluster skips its whole channel: no deploys, gauge=1.
func TestRunClusterChannel_PausedSkipsDispatch(t *testing.T) {
	job := &models.BackfillJob{ID: "job-1", Status: "running", TemplateID: "tpl-1", TemplateVersion: 1, TotalCount: 1}
	uc, _, _, d := newSubmitterFixture(job, []models.BackfillItem{
		{ID: "item-1", JobID: "job-1", AssetID: "asset-1", Status: "pending"},
	})
	store := &fakeDispatcherStore{rows: []models.DispatcherConfig{{ClusterID: "default", Paused: true, MaxConcurrency: 8, SubmitBatch: 8, RatePerSec: 5}}}
	uc.SetDispatcherConfigRepo(store)

	uc.runSubmitterCycle(context.Background())

	if len(d.deploys) != 0 {
		t.Fatalf("deploys = %v, want none (paused)", d.deploys)
	}
	if got := testutil.ToFloat64(metrics.DispatcherChannelPaused.WithLabelValues("default")); got != 1 {
		t.Fatalf("paused gauge = %v, want 1", got)
	}

	// Un-pause: dispatch resumes and the gauge resets.
	store.rows[0].Paused = false
	uc.runSubmitterCycle(context.Background())
	if len(d.deploys) != 1 {
		t.Fatalf("deploys = %v, want 1 after unpause", d.deploys)
	}
	if got := testutil.ToFloat64(metrics.DispatcherChannelPaused.WithLabelValues("default")); got != 0 {
		t.Fatalf("paused gauge = %v, want 0", got)
	}
}

// The per-cluster submit_batch override bounds one job's batch.
func TestSubmitJobBatch_ConfigOverridesBatchLimit(t *testing.T) {
	items := make([]models.BackfillItem, 3)
	for i := range items {
		items[i] = models.BackfillItem{ID: string(rune('a' + i)), JobID: "job-1", AssetID: string(rune('a' + i)), Status: "pending"}
	}
	job := &models.BackfillJob{ID: "job-1", Status: "running", TemplateID: "tpl-1", TemplateVersion: 1, TotalCount: 3}
	uc, _, _, d := newSubmitterFixture(job, items)
	store := &fakeDispatcherStore{rows: []models.DispatcherConfig{{ClusterID: "default", MaxConcurrency: 8, SubmitBatch: 2, RatePerSec: 50}}}
	uc.SetDispatcherConfigRepo(store)

	uc.runSubmitterCycle(context.Background())

	// First pass dispatches only submit_batch=2 (self-kick handles the rest
	// in production; the single test cycle stops at the override).
	if len(d.deploys) != 2 {
		t.Fatalf("deploys = %d, want 2 (submit_batch override)", len(d.deploys))
	}
}

// CYB-4026 D3: with a per-cluster submit_batch override BELOW the compiled
// perJobSubmitBatch default, a full batch must still self-kick. The old code
// compared attempts against the compiled constant (128), so an override of 2
// never triggered the kick and a large backlog degraded to one small batch per
// tick. Keep the compiled default large here (unlike the pre-existing kick
// test which set it to 2, masking the bug).
func TestSubmitJobBatch_OverrideBelowDefaultStillKicks(t *testing.T) {
	old := perJobSubmitBatch
	perJobSubmitBatch = 128
	defer func() { perJobSubmitBatch = old }()

	items := make([]models.BackfillItem, 3)
	for i := range items {
		items[i] = models.BackfillItem{ID: string(rune('a' + i)), JobID: "job-1", AssetID: string(rune('a' + i)), Status: "pending"}
	}
	job := &models.BackfillJob{ID: "job-1", Status: "running", TemplateID: "tpl-1", TemplateVersion: 1, TotalCount: 3}
	uc, _, _, d := newSubmitterFixture(job, items)
	store := &fakeDispatcherStore{rows: []models.DispatcherConfig{{ClusterID: "default", MaxConcurrency: 8, SubmitBatch: 2, RatePerSec: 50}}}
	uc.SetDispatcherConfigRepo(store)
	uc.submitKick = make(chan struct{}, 1)

	uc.runSubmitterCycle(context.Background())

	if len(d.deploys) != 2 {
		t.Fatalf("deploys = %d, want 2 (submit_batch override)", len(d.deploys))
	}
	select {
	case <-uc.submitKick:
	default:
		t.Fatal("override-sized full batch with backlog must self-kick (D3)")
	}
}

func TestDispatcherStatus(t *testing.T) {
	store := &fakeDispatcherStore{rows: []models.DispatcherConfig{
		{ClusterID: "clu-paused", MaxConcurrency: 8, SubmitBatch: 8, RatePerSec: 5, Paused: true},
		{ClusterID: "clu-tuned", MaxConcurrency: 16, SubmitBatch: 8, RatePerSec: 5},
	}}
	uc := &Usecase{}
	uc.SetDispatcherConfigRepo(store)
	// Live governor for clu-tuned, halved below configured → aimd_backoff.
	g := newClusterGovernor(16)
	g.effective = 8
	uc.governors = map[string]*clusterGovernor{"clu-tuned": g, "clu-live-only": newClusterGovernor(20)}

	statuses, err := uc.DispatcherStatus(context.Background())
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	byID := map[string]models.DispatcherClusterStatus{}
	for _, s := range statuses {
		byID[s.ClusterID] = s
	}
	if len(byID) != 3 {
		t.Fatalf("clusters = %v, want 3 (rows ∪ live governors)", byID)
	}
	if s := byID["clu-paused"]; !s.HasRow || s.SuppressionReason != "paused" {
		t.Fatalf("clu-paused = %+v", s)
	}
	if s := byID["clu-tuned"]; s.EffectiveConcurrency != 8 || s.SuppressionReason != "aimd_backoff" {
		t.Fatalf("clu-tuned = %+v", s)
	}
	if s := byID["clu-live-only"]; s.HasRow || s.SuppressionReason != "" {
		t.Fatalf("clu-live-only = %+v (defaults, healthy)", s)
	}
}

// With nothing configured and no live governors, status still reports the
// default cluster (the UI always has a row to edit).
func TestDispatcherStatus_EmptyShowsDefault(t *testing.T) {
	uc := &Usecase{}
	uc.SetDispatcherConfigRepo(&fakeDispatcherStore{})
	statuses, err := uc.DispatcherStatus(context.Background())
	if err != nil || len(statuses) != 1 || statuses[0].ClusterID != "default" {
		t.Fatalf("statuses = %+v err=%v", statuses, err)
	}
	if statuses[0].HasRow {
		t.Fatal("default synthetic row must report HasRow=false")
	}
}
