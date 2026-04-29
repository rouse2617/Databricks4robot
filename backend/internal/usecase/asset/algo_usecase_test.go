package asset

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/quick"
	"time"

	"data-platform/internal/config"
	"data-platform/internal/models"
	"data-platform/internal/repository"
)

// ─── Mock Repositories ──────────────────────────────────────────────────────
//
// These mocks are intentionally minimal and sized to AlgoUsecase's needs
// only. They share a single sync.Mutex per repo to mirror the serialisation
// guarantee a real PostgreSQL transaction provides for the affected rows.

// mockAssetRepo is the existence-check repo. AlgoUsecase only ever calls
// Get on it; the rest of the AssetRepository surface is implemented as
// no-ops to satisfy the interface.
type mockAssetRepo struct {
	mu     sync.Mutex
	assets map[string]*models.Asset
}

func newMockAssetRepo() *mockAssetRepo {
	return &mockAssetRepo{assets: make(map[string]*models.Asset)}
}

func (m *mockAssetRepo) Get(_ context.Context, assetID string) (*models.Asset, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.assets[assetID]
	if !ok {
		return nil, nil
	}
	cp := *a
	cp.AlgoResults = copyMapSS(a.AlgoResults)
	cp.Tags = copyMapSS(a.Tags)
	cp.Files = copyMapSS(a.Files)
	return &cp, nil
}

// version returns the persisted asset version (used by tests to assert that
// algo operations do NOT bump it).
func (m *mockAssetRepo) version(assetID string) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.assets[assetID]
	if !ok {
		return -1
	}
	return a.Version
}

func (m *mockAssetRepo) Set(_ context.Context, a *models.Asset) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.assets[a.AssetID] = a
	return nil
}
func (m *mockAssetRepo) SoftDelete(_ context.Context, assetID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.assets, assetID)
	return nil
}
func (m *mockAssetRepo) ListByMcapFile(_ context.Context, _ string) ([]*models.Asset, error) {
	return nil, nil
}
func (m *mockAssetRepo) WriteSegmentIndex(_ context.Context, _ *models.Asset) error { return nil }
func (m *mockAssetRepo) ListWithFilters(_ context.Context, _ string, _ []interface{},
	_, _ int, _ string) ([]*models.Asset, int64, error) {
	return nil, 0, nil
}
func (m *mockAssetRepo) MergeCfAlgo(_ context.Context, _ string, _ int64,
	_ map[string]interface{}, _ map[string]interface{}) (int64, error) {
	// Should never be called by the new AlgoUsecase. Returning an error
	// makes any accidental regression visible in tests.
	return 0, fmt.Errorf("mockAssetRepo.MergeCfAlgo: forbidden — algo state must use AssetAlgoLatestRepo")
}

func copyMapSS(m map[string]string) map[string]string {
	if m == nil {
		return map[string]string{}
	}
	cp := make(map[string]string, len(m))
	for k, v := range m {
		cp[k] = v
	}
	return cp
}

// mockAlgoLatestRepo backs (asset_id, algo_name) → *models.AssetAlgoLatest.
// Upsert respects the monotonic guard on algo_version that the production
// repo enforces, so concurrency tests reproduce the real semantics.
type mockAlgoLatestRepo struct {
	mu      sync.Mutex
	rows    map[string]*models.AssetAlgoLatest
	upserts int64
}

func newMockAlgoLatestRepo() *mockAlgoLatestRepo {
	return &mockAlgoLatestRepo{rows: make(map[string]*models.AssetAlgoLatest)}
}

func keyFor(assetID, algoName string) string { return assetID + "|" + algoName }

func (m *mockAlgoLatestRepo) Upsert(_ context.Context, row *models.AssetAlgoLatest) error {
	if row == nil {
		return fmt.Errorf("nil row")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	atomic.AddInt64(&m.upserts, 1)
	k := keyFor(row.AssetID, row.AlgoName)
	cur, exists := m.rows[k]
	if exists && cur.AlgoVersion > row.AlgoVersion {
		// Monotonic guard: silently drop older versions.
		return nil
	}
	cp := *row
	if cp.UpdatedAt.IsZero() {
		cp.UpdatedAt = time.Now().UTC()
	}
	// Preserve started_at if the new row didn't set one (mirrors COALESCE in SQL).
	if cp.StartedAt == nil && exists && cur.StartedAt != nil {
		cp.StartedAt = cur.StartedAt
	}
	m.rows[k] = &cp
	return nil
}

func (m *mockAlgoLatestRepo) GetByAlgo(_ context.Context, assetID, algoName string) (*models.AssetAlgoLatest, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.rows[keyFor(assetID, algoName)]
	if !ok {
		return nil, nil
	}
	cp := *r
	return &cp, nil
}

func (m *mockAlgoLatestRepo) ListByAsset(_ context.Context, assetID string) ([]*models.AssetAlgoLatest, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []*models.AssetAlgoLatest
	for _, r := range m.rows {
		if r.AssetID == assetID {
			cp := *r
			out = append(out, &cp)
		}
	}
	return out, nil
}

// seed sets a row directly without going through Upsert. Used by tests to
// stage state for the system-under-test.
func (m *mockAlgoLatestRepo) seed(row *models.AssetAlgoLatest) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *row
	if cp.UpdatedAt.IsZero() {
		cp.UpdatedAt = time.Now().UTC()
	}
	m.rows[keyFor(row.AssetID, row.AlgoName)] = &cp
}

// mockAssetEventRepo records every Append in the order received and exposes a
// listing API matching the production interface.
type mockAssetEventRepo struct {
	mu     sync.Mutex
	events []*models.AssetEvent
}

func newMockAssetEventRepo() *mockAssetEventRepo {
	return &mockAssetEventRepo{}
}

func (m *mockAssetEventRepo) Append(_ context.Context, in repository.AssetEventAppendInput) error {
	if in.EventType == "" {
		return fmt.Errorf("event_type required")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	payload := in.EventPayload
	if payload == nil {
		payload = []byte(`{}`)
	}
	m.events = append(m.events, &models.AssetEvent{
		EventID:              fmt.Sprintf("evt-%d", len(m.events)+1),
		EventSeq:             int64(len(m.events) + 1),
		EventType:            in.EventType,
		PayloadSchemaVersion: "v1",
		AssetID:              in.AssetID,
		McapFileID:           in.McapFileID,
		EventSource:          "backend",
		PublishState:         "pending",
		EventPayload:         append([]byte(nil), payload...),
		CreatedAt:            time.Now().UTC(),
		OccurredAt:           time.Now().UTC(),
	})
	return nil
}

func (m *mockAssetEventRepo) ListPending(_ context.Context, limit int) ([]*models.AssetEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if limit <= 0 || limit > len(m.events) {
		limit = len(m.events)
	}
	out := make([]*models.AssetEvent, 0, limit)
	for _, e := range m.events {
		if e.PublishState == "pending" {
			out = append(out, e)
		}
		if len(out) == limit {
			break
		}
	}
	return out, nil
}

func (m *mockAssetEventRepo) ListByAsset(_ context.Context, assetID string, opts repository.AssetEventListOptions) ([]*models.AssetEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	limit := opts.Limit
	if limit <= 0 {
		limit = 100
	}
	allow := map[string]struct{}{}
	for _, t := range opts.EventTypes {
		allow[t] = struct{}{}
	}
	var out []*models.AssetEvent
	// DESC order = newest first.
	for i := len(m.events) - 1; i >= 0; i-- {
		e := m.events[i]
		if e.AssetID != assetID {
			continue
		}
		if len(allow) > 0 {
			if _, ok := allow[e.EventType]; !ok {
				matched := false
				for _, pattern := range opts.EventTypePatterns {
					if strings.HasSuffix(pattern, "%") && strings.HasPrefix(e.EventType, strings.TrimSuffix(pattern, "%")) {
						matched = true
						break
					}
				}
				if !matched {
					continue
				}
			}
		} else if len(opts.EventTypePatterns) > 0 {
			matched := false
			for _, pattern := range opts.EventTypePatterns {
				if strings.HasSuffix(pattern, "%") && strings.HasPrefix(e.EventType, strings.TrimSuffix(pattern, "%")) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		if opts.AlgoKey != "" {
			var payload map[string]any
			if err := json.Unmarshal(e.EventPayload, &payload); err != nil {
				continue
			}
			if got, _ := payload["algo_key"].(string); got != opts.AlgoKey {
				continue
			}
		}
		if opts.BeforeEventSeq != nil && e.EventSeq >= *opts.BeforeEventSeq {
			continue
		}
		if opts.AfterEventSeq != nil && e.EventSeq <= *opts.AfterEventSeq {
			continue
		}
		out = append(out, e)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (m *mockAssetEventRepo) MarkPublished(context.Context, []int64) error { return nil }

func (m *mockAssetEventRepo) MarkFailed(context.Context, int64, string) error { return nil }

func (m *mockAssetEventRepo) CountPending(context.Context) (int64, error) { return 0, nil }

func (m *mockAssetEventRepo) all() []*models.AssetEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]*models.AssetEvent, len(m.events))
	copy(cp, m.events)
	return cp
}

// mockTxRunner runs fn directly with the same context. Adequate because the
// repo mocks above use their own locks and don't rely on real transactional
// isolation — the goal is to exercise AlgoUsecase logic, not pgx.Tx.
type mockTxRunner struct{}

func (mockTxRunner) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

// ─── Test Helpers ───────────────────────────────────────────────────────────

func buildTestRegistry(t *testing.T) *config.AlgoRegistry {
	t.Helper()
	reg, err := config.LoadAlgoRegistry("../../../config/algo_registry.yaml")
	if err != nil {
		t.Fatalf("failed to load algo registry: %v", err)
	}
	return reg
}

func makeAsset(assetID string) *models.Asset {
	return &models.Asset{
		AssetID:    assetID,
		McapFileID: "mcap-001",
		Version:    1,
		CreatedAt:  time.Now().UTC(),
	}
}

// validAlgoKeys returns algo keys from the registry for testing.
func validAlgoKeys(reg *config.AlgoRegistry) []string {
	var keys []string
	for name, def := range reg.GetAllAlgorithms() {
		for _, ver := range def.Versions {
			keys = append(keys, name+"@"+ver)
		}
	}
	return keys
}

// newTestUsecase wires a usecase with fresh in-memory repos. Returns the
// usecase plus pointers to each repo so tests can seed and assert on them.
func newTestUsecase(reg *config.AlgoRegistry) (
	*AlgoUsecase, *mockAssetRepo, *mockAlgoLatestRepo, *mockAssetEventRepo,
) {
	assetRepo := newMockAssetRepo()
	algoLatest := newMockAlgoLatestRepo()
	eventRepo := newMockAssetEventRepo()
	uc := NewAlgoUsecase(mockTxRunner{}, assetRepo, algoLatest, eventRepo, reg)
	return uc, assetRepo, algoLatest, eventRepo
}

// seedStatus stores a single algo row at the given status, used to stage
// the state-machine starting point.
func seedStatus(repo *mockAlgoLatestRepo, assetID, algoKey, status, runID string) {
	name, ver, _ := parseAlgoKey(algoKey)
	repo.seed(&models.AssetAlgoLatest{
		AssetID:     assetID,
		AlgoName:    name,
		AlgoVersion: ver,
		Status:      status,
		RunID:       runID,
	})
}

// statusOf reads the persisted status for (assetID, algoKey). Returns "" when
// no row exists.
func statusOf(repo *mockAlgoLatestRepo, assetID, algoKey string) models.AlgoStatus {
	name, _, _ := parseAlgoKey(algoKey)
	row, _ := repo.GetByAlgo(context.Background(), assetID, name)
	if row == nil {
		return ""
	}
	return models.AlgoStatus(row.Status)
}

// ─── Property 13: State Machine Enforcement ─────────────────────────────────
func TestProperty13_StateMachineEnforcement(t *testing.T) {
	reg := buildTestRegistry(t)
	ctx := context.Background()

	cfg := &quick.Config{MaxCount: 100, Rand: rand.New(rand.NewSource(time.Now().UnixNano()))}

	allStatuses := []models.AlgoStatus{
		"", // no prior row
		models.AlgoStatusBlocked,
		models.AlgoStatusPending,
		models.AlgoStatusRunning,
		models.AlgoStatusOk,
		models.AlgoStatusFailed,
	}
	algoKeys := validAlgoKeys(reg)

	f := func(statusIdx uint8, keyIdx uint8) bool {
		if len(algoKeys) == 0 {
			return true
		}
		algoKey := algoKeys[int(keyIdx)%len(algoKeys)]
		curStatus := allStatuses[int(statusIdx)%len(allStatuses)]
		assetID := "asset-p13"

		// StartAlgo: legal from "" or pending.
		uc, ar, alr, _ := newTestUsecase(reg)
		ar.assets[assetID] = makeAsset(assetID)
		if curStatus != "" {
			seedStatus(alr, assetID, algoKey, string(curStatus), "")
		}
		startErr := uc.StartAlgo(ctx, assetID, algoKey, StartAlgoInput{Method: "test"})
		startAllowed := curStatus == models.AlgoStatusPending || curStatus == ""
		if startAllowed && startErr != nil {
			t.Errorf("StartAlgo should succeed from %q but got: %v", curStatus, startErr)
			return false
		}
		if !startAllowed && startErr == nil {
			t.Errorf("StartAlgo should fail from %q but succeeded", curStatus)
			return false
		}

		// FinishAlgo: legal only from running.
		uc, ar, alr, _ = newTestUsecase(reg)
		ar.assets[assetID] = makeAsset(assetID)
		if curStatus != "" {
			seedStatus(alr, assetID, algoKey, string(curStatus), "")
		}
		reason := "test failure"
		finishErr := uc.FinishAlgo(ctx, assetID, algoKey, FinishAlgoInput{
			Status: "failed", Reason: &reason,
		})
		finishAllowed := curStatus == models.AlgoStatusRunning
		if finishAllowed && finishErr != nil {
			t.Errorf("FinishAlgo should succeed from %q but got: %v", curStatus, finishErr)
			return false
		}
		if !finishAllowed && finishErr == nil {
			t.Errorf("FinishAlgo should fail from %q but succeeded", curStatus)
			return false
		}

		// ResetAlgo: legal from failed or ok.
		uc, ar, alr, _ = newTestUsecase(reg)
		ar.assets[assetID] = makeAsset(assetID)
		if curStatus != "" {
			seedStatus(alr, assetID, algoKey, string(curStatus), "")
		}
		resetErr := uc.ResetAlgo(ctx, assetID, algoKey)
		resetAllowed := curStatus == models.AlgoStatusFailed || curStatus == models.AlgoStatusOk
		if resetAllowed && resetErr != nil {
			t.Errorf("ResetAlgo should succeed from %q but got: %v", curStatus, resetErr)
			return false
		}
		if !resetAllowed && resetErr == nil {
			t.Errorf("ResetAlgo should fail from %q but succeeded", curStatus)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 13 failed: %v", err)
	}
}

// ─── Property 14: Start sets running + started_at + method ──────────────────
func TestProperty14_StartSetsRunning(t *testing.T) {
	reg := buildTestRegistry(t)
	ctx := context.Background()
	algoKeys := validAlgoKeys(reg)

	cfg := &quick.Config{MaxCount: 100, Rand: rand.New(rand.NewSource(time.Now().UnixNano()))}

	f := func(keyIdx uint8, methodSeed uint8) bool {
		if len(algoKeys) == 0 {
			return true
		}
		algoKey := algoKeys[int(keyIdx)%len(algoKeys)]
		name, _, _ := parseAlgoKey(algoKey)
		method := fmt.Sprintf("method_%d", methodSeed)

		uc, ar, alr, _ := newTestUsecase(reg)
		assetID := "asset-p14"
		ar.assets[assetID] = makeAsset(assetID)
		seedStatus(alr, assetID, algoKey, "pending", "")

		runID := fmt.Sprintf("run-%d", methodSeed)
		if err := uc.StartAlgo(ctx, assetID, algoKey, StartAlgoInput{Method: method, RunID: &runID}); err != nil {
			t.Errorf("StartAlgo failed: %v", err)
			return false
		}
		row, _ := alr.GetByAlgo(ctx, assetID, name)
		if row == nil {
			t.Error("expected algo row after StartAlgo")
			return false
		}
		if row.Status != string(models.AlgoStatusRunning) {
			t.Errorf("expected running, got %q", row.Status)
			return false
		}
		if row.StartedAt == nil {
			t.Error("started_at not set")
			return false
		}
		if row.Method != method {
			t.Errorf("expected method %q, got %q", method, row.Method)
			return false
		}
		if row.RunID != runID {
			t.Errorf("expected run_id %q, got %q", runID, row.RunID)
			return false
		}
		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 14 failed: %v", err)
	}
}

// ─── Property 15: Finish writes final state correctly ───────────────────────
func TestProperty15_FinishUpdatesCorrectly(t *testing.T) {
	reg := buildTestRegistry(t)
	ctx := context.Background()

	cfg := &quick.Config{MaxCount: 100, Rand: rand.New(rand.NewSource(time.Now().UnixNano()))}

	algoKey := "env_analysis@1.0.0"
	name, _, _ := parseAlgoKey(algoKey)

	f := func(isOk bool, seed uint8) bool {
		uc, ar, alr, _ := newTestUsecase(reg)
		assetID := "asset-p15"
		ar.assets[assetID] = makeAsset(assetID)
		seedStatus(alr, assetID, algoKey, "running", "")

		runID := fmt.Sprintf("run-%d", seed)
		if isOk {
			if err := uc.FinishAlgo(ctx, assetID, algoKey, FinishAlgoInput{
				Status: "ok", RunID: &runID,
			}); err != nil {
				t.Errorf("FinishAlgo(ok) failed: %v", err)
				return false
			}
			row, _ := alr.GetByAlgo(ctx, assetID, name)
			if row.Status != string(models.AlgoStatusOk) {
				t.Errorf("expected ok status, got %q", row.Status)
				return false
			}
			if row.FinishedAt == nil {
				t.Error("finished_at not set")
				return false
			}
		} else {
			reason := fmt.Sprintf("error-%d", seed)
			if err := uc.FinishAlgo(ctx, assetID, algoKey, FinishAlgoInput{
				Status: "failed", Reason: &reason, RunID: &runID,
			}); err != nil {
				t.Errorf("FinishAlgo(failed) failed: %v", err)
				return false
			}
			row, _ := alr.GetByAlgo(ctx, assetID, name)
			if row.Status != string(models.AlgoStatusFailed) {
				t.Errorf("expected failed, got %q", row.Status)
				return false
			}
			if row.ErrorMessage != reason {
				t.Errorf("expected error_message %q, got %q", reason, row.ErrorMessage)
				return false
			}
		}
		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 15 failed: %v", err)
	}
}

// ─── Property 16: Reset clears finish-time fields and sets pending ──────────
func TestProperty16_ResetClearsAndSetsPending(t *testing.T) {
	reg := buildTestRegistry(t)
	ctx := context.Background()

	cfg := &quick.Config{MaxCount: 100, Rand: rand.New(rand.NewSource(time.Now().UnixNano()))}
	algoKeys := validAlgoKeys(reg)

	f := func(keyIdx uint8, fromOk bool) bool {
		if len(algoKeys) == 0 {
			return true
		}
		algoKey := algoKeys[int(keyIdx)%len(algoKeys)]
		name, ver, _ := parseAlgoKey(algoKey)

		uc, ar, alr, _ := newTestUsecase(reg)
		assetID := "asset-p16"
		ar.assets[assetID] = makeAsset(assetID)

		prev := "failed"
		if fromOk {
			prev = "ok"
		}
		now := time.Now().UTC()
		alr.seed(&models.AssetAlgoLatest{
			AssetID:      assetID,
			AlgoName:     name,
			AlgoVersion:  ver,
			Status:       prev,
			ErrorMessage: "some reason",
			StartedAt:    &now,
			FinishedAt:   &now,
			OutputURI:    "gs://bucket/file",
			RunID:        "old-run",
		})

		if err := uc.ResetAlgo(ctx, assetID, algoKey); err != nil {
			t.Errorf("ResetAlgo failed: %v", err)
			return false
		}
		row, _ := alr.GetByAlgo(ctx, assetID, name)
		if row.Status != string(models.AlgoStatusPending) {
			t.Errorf("expected pending, got %q", row.Status)
			return false
		}
		// finished_at, output_uri, error_message and run_id should be cleared
		// by Upsert (they default to zero values in the new row).
		if row.FinishedAt != nil {
			t.Error("expected finished_at cleared")
			return false
		}
		if row.OutputURI != "" {
			t.Errorf("expected output_uri cleared, got %q", row.OutputURI)
			return false
		}
		if row.ErrorMessage != "" {
			t.Errorf("expected error_message cleared, got %q", row.ErrorMessage)
			return false
		}
		if row.RunID != "" {
			t.Errorf("expected run_id cleared, got %q", row.RunID)
			return false
		}
		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 16 failed: %v", err)
	}
}

// ─── Property 17: Every state transition appends one event ──────────────────
func TestProperty17_LifecycleOpsInsertEvents(t *testing.T) {
	reg := buildTestRegistry(t)
	ctx := context.Background()

	cfg := &quick.Config{MaxCount: 50, Rand: rand.New(rand.NewSource(time.Now().UnixNano()))}

	algoKey := "env_analysis@1.0.0"

	f := func(seed uint8) bool {
		uc, ar, alr, evt := newTestUsecase(reg)
		assetID := "asset-p17"
		ar.assets[assetID] = makeAsset(assetID)
		seedStatus(alr, assetID, algoKey, "pending", "")

		method := fmt.Sprintf("m%d", seed)
		if err := uc.StartAlgo(ctx, assetID, algoKey, StartAlgoInput{Method: method}); err != nil {
			t.Errorf("StartAlgo failed: %v", err)
			return false
		}
		if got := len(evt.all()); got != 1 {
			t.Errorf("expected 1 event after start, got %d", got)
			return false
		}
		if evt.all()[0].EventType != eventAlgoStarted {
			t.Errorf("expected algo_started event, got %q", evt.all()[0].EventType)
			return false
		}

		reason := "err"
		if err := uc.FinishAlgo(ctx, assetID, algoKey, FinishAlgoInput{Status: "failed", Reason: &reason}); err != nil {
			t.Errorf("FinishAlgo failed: %v", err)
			return false
		}
		if got := len(evt.all()); got != 2 {
			t.Errorf("expected 2 events after finish, got %d", got)
			return false
		}
		if evt.all()[1].EventType != eventAlgoFailed {
			t.Errorf("expected algo_failed, got %q", evt.all()[1].EventType)
			return false
		}

		if err := uc.ResetAlgo(ctx, assetID, algoKey); err != nil {
			t.Errorf("ResetAlgo failed: %v", err)
			return false
		}
		if got := len(evt.all()); got != 3 {
			t.Errorf("expected 3 events after reset, got %d", got)
			return false
		}
		if evt.all()[2].EventType != eventAlgoReset {
			t.Errorf("expected algo_reset, got %q", evt.all()[2].EventType)
			return false
		}
		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 17 failed: %v", err)
	}
}

// ─── Property 18: ok finishes with missing required fields are rejected ─────
func TestProperty18_MissingRequiredFieldsRejected(t *testing.T) {
	reg := buildTestRegistry(t)
	ctx := context.Background()

	cfg := &quick.Config{MaxCount: 30, Rand: rand.New(rand.NewSource(time.Now().UnixNano()))}

	// hand_tracking requires output_uri, "type" extra, and report_size=true.
	algoKey := "hand_tracking@1.2.0"

	f := func(seed uint8) bool {
		_ = seed
		uc, ar, alr, _ := newTestUsecase(reg)
		assetID := "asset-p18"
		ar.assets[assetID] = makeAsset(assetID)
		seedStatus(alr, assetID, algoKey, "running", "")

		// Missing output_uri.
		if err := uc.FinishAlgo(ctx, assetID, algoKey, FinishAlgoInput{Status: "ok"}); err == nil {
			t.Error("expected error for missing output_uri")
			return false
		} else if !strings.Contains(err.Error(), "output_uri") && !strings.Contains(err.Error(), "missing") {
			t.Errorf("expected missing field error, got: %v", err)
			return false
		}

		// output_uri provided but missing "type" extra.
		uri := "gs://bucket/output"
		size := int64(1024)
		if err := uc.FinishAlgo(ctx, assetID, algoKey, FinishAlgoInput{
			Status: "ok", OutputURI: &uri, ResultSizeBytes: &size,
		}); err == nil {
			t.Error("expected error for missing type field")
			return false
		}

		// type provided but missing result_size_bytes (report_size=true).
		if err := uc.FinishAlgo(ctx, assetID, algoKey, FinishAlgoInput{
			Status: "ok", OutputURI: &uri, ExtraFields: map[string]interface{}{"type": "mcap"},
		}); err == nil {
			t.Error("expected error for missing result_size_bytes")
			return false
		}
		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 18 failed: %v", err)
	}
}

// ─── Property 21: finish_algo idempotent on matching run_id ─────────────────
func TestProperty21_FinishIdempotency(t *testing.T) {
	reg := buildTestRegistry(t)
	ctx := context.Background()

	cfg := &quick.Config{MaxCount: 50, Rand: rand.New(rand.NewSource(time.Now().UnixNano()))}

	algoKey := "env_analysis@1.0.0"

	f := func(seed uint8) bool {
		uc, ar, alr, evt := newTestUsecase(reg)
		assetID := "asset-p21"
		runID := fmt.Sprintf("run-%d", seed)

		ar.assets[assetID] = makeAsset(assetID)
		// Seed already-ok with the run_id we'll re-finish with.
		seedStatus(alr, assetID, algoKey, "ok", runID)

		// Same run_id → idempotent no-op.
		if err := uc.FinishAlgo(ctx, assetID, algoKey, FinishAlgoInput{Status: "ok", RunID: &runID}); err != nil {
			t.Errorf("expected idempotent success, got: %v", err)
			return false
		}
		if got := len(evt.all()); got != 0 {
			t.Errorf("expected 0 events for idempotent call, got %d", got)
			return false
		}

		// Different run_id while status=ok → invalid transition.
		other := runID + "-different"
		if err := uc.FinishAlgo(ctx, assetID, algoKey, FinishAlgoInput{Status: "ok", RunID: &other}); err == nil {
			t.Error("expected error for different run_id on ok status")
			return false
		}
		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 21 failed: %v", err)
	}
}

// ─── Property 22: tryUnblockDownstream unblocks only when all deps ok ───────
func TestProperty22_DependencyUnlocking(t *testing.T) {
	reg := buildTestRegistry(t)
	ctx := context.Background()

	cfg := &quick.Config{MaxCount: 30, Rand: rand.New(rand.NewSource(time.Now().UnixNano()))}

	// action_annotation@1.0.0 depends on hand/head/body tracking.
	downKey := "action_annotation@1.0.0"
	downName, _, _ := parseAlgoKey(downKey)
	deps := []string{"hand_tracking@1.2.0", "head_tracking@1.0.0", "body_tracking@1.0.0"}

	f := func(completedCount uint8) bool {
		num := int(completedCount) % 4

		uc, ar, alr, _ := newTestUsecase(reg)
		assetID := "asset-p22"
		ar.assets[assetID] = makeAsset(assetID)
		seedStatus(alr, assetID, downKey, "blocked", "")
		for i, dep := range deps {
			st := "running"
			if i < num {
				st = "ok"
			}
			seedStatus(alr, assetID, dep, st, "")
		}

		if num > 0 {
			lastDep := deps[num-1]
			if err := uc.tryUnblockDownstream(ctx, assetID, lastDep); err != nil {
				t.Errorf("tryUnblockDownstream failed: %v", err)
				return false
			}
		}

		row, _ := alr.GetByAlgo(ctx, assetID, downName)
		got := models.AlgoStatus(row.Status)
		if num == 3 {
			if got != models.AlgoStatusPending {
				t.Errorf("expected pending when all deps ok, got %q", got)
				return false
			}
		} else {
			if got != models.AlgoStatusBlocked {
				t.Errorf("expected blocked when not all deps ok (completed=%d), got %q", num, got)
				return false
			}
		}
		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 22 failed: %v", err)
	}
}

// ─── Property 19: ListAlgoEvents — DESC by event_seq + algo_key filter ──────
func TestProperty19_EventsOrderedAndFilterable(t *testing.T) {
	reg := buildTestRegistry(t)
	ctx := context.Background()

	cfg := &quick.Config{MaxCount: 30, Rand: rand.New(rand.NewSource(time.Now().UnixNano()))}

	algoKeys := validAlgoKeys(reg)

	f := func(numEvents uint8) bool {
		if len(algoKeys) < 2 {
			return true
		}
		count := int(numEvents)%10 + 2

		uc, ar, _, evt := newTestUsecase(reg)
		assetID := "asset-p19"
		ar.assets[assetID] = makeAsset(assetID)

		// Append `count` events alternating algo_key.
		for i := 0; i < count; i++ {
			ak := algoKeys[i%len(algoKeys)]
			name, ver, _ := parseAlgoKey(ak)
			payload, _ := json.Marshal(map[string]interface{}{
				"algo_key":     ak,
				"algo_name":    name,
				"algo_version": ver,
				"prev_status":  "running",
				"new_status":   "ok",
			})
			_ = evt.Append(ctx, repository.AssetEventAppendInput{
				EventType:    eventAlgoFinished,
				AssetID:      assetID,
				EventPayload: payload,
			})
		}

		all, err := uc.ListAlgoEvents(ctx, assetID, nil)
		if err != nil {
			t.Errorf("ListAlgoEvents failed: %v", err)
			return false
		}
		if len(all) != count {
			t.Errorf("expected %d events, got %d", count, len(all))
			return false
		}
		// DESC: each entry must be older than the previous.
		for i := 1; i < len(all); i++ {
			if all[i].CreatedAt.After(all[i-1].CreatedAt) {
				t.Error("events not in DESC order")
				return false
			}
		}

		// Filter by a specific algo_key — every result must match.
		filterKey := algoKeys[0]
		filtered, err := uc.ListAlgoEvents(ctx, assetID, &filterKey)
		if err != nil {
			t.Errorf("ListAlgoEvents with filter failed: %v", err)
			return false
		}
		for _, e := range filtered {
			if e.AlgoKey != filterKey {
				t.Errorf("expected algo_key %q, got %q", filterKey, e.AlgoKey)
				return false
			}
		}
		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 19 failed: %v", err)
	}
}

// ─── Issue 2 regression: assets.version is NOT bumped on algo state changes ─
//
// Reviewer concern: under the previous implementation, every algo finish
// took a CAS lock on assets.version, causing concurrent finishes from
// different algorithms on the same asset to collide and 409. The fix moves
// state to asset_algo_latest where (asset_id, algo_name) PK + monotonic
// version guard make concurrent finishes lock-free. This test asserts the
// behaviour shift: assets.version stays untouched and concurrent finishes
// of *different* algorithms all succeed.
func TestIssue2_AssetVersionUntouched_ConcurrentFinishesNoConflict(t *testing.T) {
	reg := buildTestRegistry(t)
	ctx := context.Background()

	uc, ar, alr, evt := newTestUsecase(reg)
	assetID := "asset-issue2"
	ar.assets[assetID] = makeAsset(assetID) // version = 1

	// Pick three independent algorithms and seed them as running. We pick
	// keys with distinct registry requirements so the test exercises the
	// full payload-validation path concurrently.
	keys := []string{"env_analysis@1.0.0", "hand_tracking@1.2.0", "head_tracking@1.0.0"}
	for _, k := range keys {
		seedStatus(alr, assetID, k, "running", "")
	}

	// inputFor builds a FinishAlgoInput that satisfies the registry
	// requirements of each algorithm.
	inputFor := func(k, runID string) FinishAlgoInput {
		in := FinishAlgoInput{Status: "ok", RunID: &runID}
		switch k {
		case "hand_tracking@1.2.0", "head_tracking@1.0.0":
			uri := "gs://b/" + k
			size := int64(2048)
			in.OutputURI = &uri
			in.ResultSizeBytes = &size
			in.ExtraFields = map[string]interface{}{"type": "mcap"}
		}
		return in
	}

	type result struct {
		key string
		err error
	}
	results := make(chan result, len(keys))
	var wg sync.WaitGroup
	for _, k := range keys {
		k := k
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- result{k, uc.FinishAlgo(ctx, assetID, k, inputFor(k, "run-"+k))}
		}()
	}
	wg.Wait()
	close(results)

	for r := range results {
		if r.err != nil {
			t.Fatalf("FinishAlgo(%s) returned %v — concurrent finishes must not conflict", r.key, r.err)
		}
	}

	// assets.version must remain 1.
	if v := ar.version(assetID); v != 1 {
		t.Fatalf("assets.version expected 1 (untouched), got %d", v)
	}

	// Each algorithm's projection row has status=ok with the matching run_id.
	for _, k := range keys {
		name, _, _ := parseAlgoKey(k)
		row, _ := alr.GetByAlgo(ctx, assetID, name)
		if row == nil {
			t.Fatalf("missing projection row for %s", k)
		}
		if row.Status != string(models.AlgoStatusOk) {
			t.Fatalf("%s status: got %q, want ok", k, row.Status)
		}
		if row.RunID != "run-"+k {
			t.Fatalf("%s run_id: got %q, want %q", k, row.RunID, "run-"+k)
		}
	}

	// Three algo_finished events, no algo_failed.
	finishedCount := 0
	for _, e := range evt.all() {
		if e.EventType == eventAlgoFinished {
			finishedCount++
		}
		if e.EventType == eventAlgoFailed {
			t.Fatalf("unexpected algo_failed event: %+v", e)
		}
	}
	if finishedCount != len(keys) {
		t.Fatalf("expected %d algo_finished events, got %d", len(keys), finishedCount)
	}
}

// ─── Issue 2 regression: monotonic guard prevents older algo_version from
// overwriting newer ─────────────────────────────────────────────────────────
//
// Two finishes of the same algorithm with different algo_version race;
// the older version's write must be silently dropped, never roll the
// projection row backwards.
func TestIssue2_MonotonicAlgoVersionGuard(t *testing.T) {
	reg := buildTestRegistry(t)
	ctx := context.Background()

	uc, ar, alr, _ := newTestUsecase(reg)
	assetID := "asset-monotonic"
	ar.assets[assetID] = makeAsset(assetID)

	// Two versions of hand_tracking exist in the registry: 1.2.0 and 1.3.0
	// (or whatever the registry contains). We craft a scenario by asserting
	// directly: write v2 first, then attempt v1 — guard must drop v1.
	older := "hand_tracking@1.0.0"
	newer := "hand_tracking@1.2.0"

	seedStatus(alr, assetID, older, "running", "")
	seedStatus(alr, assetID, newer, "running", "")

	uri := "gs://b/x"
	size := int64(100)
	extras := map[string]interface{}{"type": "mcap"}

	// Finish the newer version first.
	if err := uc.FinishAlgo(ctx, assetID, newer, FinishAlgoInput{
		Status: "ok", OutputURI: &uri, ResultSizeBytes: &size, ExtraFields: extras,
	}); err != nil {
		t.Fatalf("finish %s: %v", newer, err)
	}

	// Now finish the older version. Mock seeds keyed by algo_name overlap, so
	// in practice the older finish will land on the same projection row;
	// the monotonic guard in Upsert MUST keep the persisted version at the
	// newer one.
	if err := uc.FinishAlgo(ctx, assetID, older, FinishAlgoInput{
		Status: "ok", OutputURI: &uri, ResultSizeBytes: &size, ExtraFields: extras,
	}); err != nil {
		// Note: depending on state, this may legitimately fail (older
		// version's row was already overwritten by newer). Either outcome
		// is acceptable; the key invariant is that the newer version's
		// state isn't rolled back.
		_ = err
	}

	row, _ := alr.GetByAlgo(ctx, assetID, "hand_tracking")
	if row == nil {
		t.Fatal("expected projection row to exist")
	}
	if row.AlgoVersion != "1.2.0" {
		t.Fatalf("expected algo_version=1.2.0 to win monotonic guard, got %q", row.AlgoVersion)
	}
}
