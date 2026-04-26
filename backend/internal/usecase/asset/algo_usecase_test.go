package asset

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"testing"
	"testing/quick"
	"time"

	"data-platform/internal/config"
	"data-platform/internal/models"
	"data-platform/internal/repository"
)

// ─── Mock Repositories ──────────────────────────────────────────────────────

// mockAssetRepo is an in-memory AssetRepository for testing.
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
	// Return a copy to avoid mutation.
	cp := *a
	cp.AlgoResults = copyMapSS(a.AlgoResults)
	cp.Tags = copyMapSS(a.Tags)
	cp.Files = copyMapSS(a.Files)
	return &cp, nil
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

func (m *mockAssetRepo) WriteSegmentIndex(_ context.Context, _ *models.Asset) error {
	return nil
}

func (m *mockAssetRepo) ListWithFilters(_ context.Context, _ string, _ []interface{},
	_, _ int, _ string) ([]*models.Asset, int64, error) {
	return nil, 0, nil
}

func (m *mockAssetRepo) MergeCfAlgo(_ context.Context, assetID string, expectedVersion int64,
	algoKV map[string]interface{}, filesKV map[string]interface{}) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.assets[assetID]
	if !ok {
		return 0, fmt.Errorf("asset not found")
	}
	if a.Version != expectedVersion {
		return 0, repository.ErrOptimisticLock
	}
	// Merge algoKV into AlgoResults.
	for k, v := range algoKV {
		if v == nil {
			delete(a.AlgoResults, k)
		} else {
			a.AlgoResults[k] = fmt.Sprintf("%v", v)
		}
	}
	// Merge filesKV into Files.
	for k, v := range filesKV {
		if v == nil {
			delete(a.Files, k)
		} else {
			a.Files[k] = fmt.Sprintf("%v", v)
		}
	}
	a.Version++
	return a.Version, nil
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

// mockEventRepo is an in-memory AlgoEventRepository for testing.
type mockEventRepo struct {
	mu     sync.Mutex
	events []*models.AlgoEvent
}

func newMockEventRepo() *mockEventRepo {
	return &mockEventRepo{}
}

func (m *mockEventRepo) Insert(_ context.Context, event *models.AlgoEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
	return nil
}

func (m *mockEventRepo) ListByAsset(_ context.Context, assetID string, algoKey *string) ([]*models.AlgoEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []*models.AlgoEvent
	for i := len(m.events) - 1; i >= 0; i-- {
		e := m.events[i]
		if e.AssetID != assetID {
			continue
		}
		if algoKey != nil && e.AlgoKey != *algoKey {
			continue
		}
		result = append(result, e)
	}
	return result, nil
}

func (m *mockEventRepo) allEvents() []*models.AlgoEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]*models.AlgoEvent, len(m.events))
	copy(cp, m.events)
	return cp
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

func makeAsset(assetID string, algoResults map[string]string) *models.Asset {
	if algoResults == nil {
		algoResults = map[string]string{}
	}
	return &models.Asset{
		AssetID:     assetID,
		McapFileID:  "mcap-001",
		AlgoResults: algoResults,
		Tags:        map[string]string{},
		Files:       map[string]string{},
		Version:     1,
		CreatedAt:   time.Now().UTC(),
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

// ─── Property 13: State Machine Enforcement ─────────────────────────────────
// **Validates: Requirements 5.7, 6.8, 7.4**
// Only legal state transitions succeed; all others return errors.
func TestProperty13_StateMachineEnforcement(t *testing.T) {
	reg := buildTestRegistry(t)
	ctx := context.Background()

	cfg := &quick.Config{MaxCount: 100, Rand: rand.New(rand.NewSource(time.Now().UnixNano()))}

	allStatuses := []models.AlgoStatus{
		"", // no prior status
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
		currentStatus := allStatuses[int(statusIdx)%len(allStatuses)]

		assetRepo := newMockAssetRepo()
		eventRepo := newMockEventRepo()
		uc := NewAlgoUsecase(assetRepo, eventRepo, reg)

		assetID := "asset-p13"
		ar := map[string]string{}
		if currentStatus != "" {
			ar[algoKey+":status"] = string(currentStatus)
		}
		assetRepo.assets[assetID] = makeAsset(assetID, ar)

		// Test StartAlgo: should only succeed from pending or empty.
		startErr := uc.StartAlgo(ctx, assetID, algoKey, StartAlgoInput{Method: "test"})
		startAllowed := currentStatus == models.AlgoStatusPending || currentStatus == ""
		if startAllowed && startErr != nil {
			t.Errorf("StartAlgo should succeed from %q but got: %v", currentStatus, startErr)
			return false
		}
		if !startAllowed && startErr == nil {
			t.Errorf("StartAlgo should fail from %q but succeeded", currentStatus)
			return false
		}

		// Reset asset for FinishAlgo test.
		ar2 := map[string]string{}
		if currentStatus != "" {
			ar2[algoKey+":status"] = string(currentStatus)
		}
		assetRepo.assets[assetID] = makeAsset(assetID, ar2)

		// Test FinishAlgo: should only succeed from running.
		reason := "test failure"
		finishErr := uc.FinishAlgo(ctx, assetID, algoKey, FinishAlgoInput{
			Status: "failed",
			Reason: &reason,
		})
		finishAllowed := currentStatus == models.AlgoStatusRunning
		if finishAllowed && finishErr != nil {
			t.Errorf("FinishAlgo should succeed from %q but got: %v", currentStatus, finishErr)
			return false
		}
		if !finishAllowed && finishErr == nil {
			t.Errorf("FinishAlgo should fail from %q but succeeded", currentStatus)
			return false
		}

		// Reset asset for ResetAlgo test.
		ar3 := map[string]string{}
		if currentStatus != "" {
			ar3[algoKey+":status"] = string(currentStatus)
		}
		assetRepo.assets[assetID] = makeAsset(assetID, ar3)

		// Test ResetAlgo: should only succeed from failed or ok.
		resetErr := uc.ResetAlgo(ctx, assetID, algoKey)
		resetAllowed := currentStatus == models.AlgoStatusFailed || currentStatus == models.AlgoStatusOk
		if resetAllowed && resetErr != nil {
			t.Errorf("ResetAlgo should succeed from %q but got: %v", currentStatus, resetErr)
			return false
		}
		if !resetAllowed && resetErr == nil {
			t.Errorf("ResetAlgo should fail from %q but succeeded", currentStatus)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 13 failed: %v", err)
	}
}

// ─── Property 14: Start Operation Sets Running Status ───────────────────────
// **Validates: Requirements 5.1**
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
		method := fmt.Sprintf("method_%d", methodSeed)

		assetRepo := newMockAssetRepo()
		eventRepo := newMockEventRepo()
		uc := NewAlgoUsecase(assetRepo, eventRepo, reg)

		assetID := "asset-p14"
		// Start from pending status.
		assetRepo.assets[assetID] = makeAsset(assetID, map[string]string{
			algoKey + ":status": "pending",
		})

		runID := fmt.Sprintf("run-%d", methodSeed)
		err := uc.StartAlgo(ctx, assetID, algoKey, StartAlgoInput{
			Method: method,
			RunID:  &runID,
		})
		if err != nil {
			t.Errorf("StartAlgo failed: %v", err)
			return false
		}

		// Verify status is now running.
		asset, _ := assetRepo.Get(ctx, assetID)
		status := getAlgoStatus(asset, algoKey)
		if status != models.AlgoStatusRunning {
			t.Errorf("expected running, got %q", status)
			return false
		}

		// Verify started_at is set and is valid RFC3339.
		startedAt := getAlgoField(asset, algoKey, models.AlgoFieldStartedAt)
		if startedAt == "" {
			t.Error("started_at not set")
			return false
		}
		if _, err := time.Parse(time.RFC3339, startedAt); err != nil {
			t.Errorf("started_at is not valid RFC3339: %v", err)
			return false
		}

		// Verify method is stored.
		storedMethod := getAlgoField(asset, algoKey, models.AlgoFieldMethod)
		if storedMethod != method {
			t.Errorf("expected method %q, got %q", method, storedMethod)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 14 failed: %v", err)
	}
}

// ─── Property 15: Finish Operation Correctly Updates cf_algo ────────────────
// **Validates: Requirements 6.1, 6.2**
func TestProperty15_FinishUpdatesCorrectly(t *testing.T) {
	reg := buildTestRegistry(t)
	ctx := context.Background()

	cfg := &quick.Config{MaxCount: 100, Rand: rand.New(rand.NewSource(time.Now().UnixNano()))}

	// Use env_analysis which has no required fields and report_size=false.
	algoKey := "env_analysis@1.0.0"

	f := func(isOk bool, seed uint8) bool {
		assetRepo := newMockAssetRepo()
		eventRepo := newMockEventRepo()
		uc := NewAlgoUsecase(assetRepo, eventRepo, reg)

		assetID := "asset-p15"
		assetRepo.assets[assetID] = makeAsset(assetID, map[string]string{
			algoKey + ":status": "running",
		})

		runID := fmt.Sprintf("run-%d", seed)
		if isOk {
			err := uc.FinishAlgo(ctx, assetID, algoKey, FinishAlgoInput{
				Status: "ok",
				RunID:  &runID,
			})
			if err != nil {
				t.Errorf("FinishAlgo(ok) failed: %v", err)
				return false
			}
			asset, _ := assetRepo.Get(ctx, assetID)
			if getAlgoStatus(asset, algoKey) != models.AlgoStatusOk {
				t.Error("expected ok status")
				return false
			}
			finishedAt := getAlgoField(asset, algoKey, models.AlgoFieldFinishedAt)
			if finishedAt == "" {
				t.Error("finished_at not set")
				return false
			}
		} else {
			reason := fmt.Sprintf("error-%d", seed)
			err := uc.FinishAlgo(ctx, assetID, algoKey, FinishAlgoInput{
				Status: "failed",
				Reason: &reason,
				RunID:  &runID,
			})
			if err != nil {
				t.Errorf("FinishAlgo(failed) failed: %v", err)
				return false
			}
			asset, _ := assetRepo.Get(ctx, assetID)
			if getAlgoStatus(asset, algoKey) != models.AlgoStatusFailed {
				t.Error("expected failed status")
				return false
			}
			storedReason := getAlgoField(asset, algoKey, models.AlgoFieldReason)
			if storedReason != reason {
				t.Errorf("expected reason %q, got %q", reason, storedReason)
				return false
			}
		}
		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 15 failed: %v", err)
	}
}

// ─── Property 16: Reset Clears Fields and Sets Pending ──────────────────────
// **Validates: Requirements 7.1**
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

		assetRepo := newMockAssetRepo()
		eventRepo := newMockEventRepo()
		uc := NewAlgoUsecase(assetRepo, eventRepo, reg)

		assetID := "asset-p16"
		prevStatus := "failed"
		if fromOk {
			prevStatus = "ok"
		}
		assetRepo.assets[assetID] = makeAsset(assetID, map[string]string{
			algoKey + ":status":      prevStatus,
			algoKey + ":reason":      "some reason",
			algoKey + ":started_at":  "2025-01-01T00:00:00Z",
			algoKey + ":finished_at": "2025-01-01T01:00:00Z",
			algoKey + ":output_uri":  "gs://bucket/file",
		})
		assetRepo.assets[assetID].Files[algoKey] = "gs://bucket/file"

		err := uc.ResetAlgo(ctx, assetID, algoKey)
		if err != nil {
			t.Errorf("ResetAlgo failed: %v", err)
			return false
		}

		asset, _ := assetRepo.Get(ctx, assetID)
		if getAlgoStatus(asset, algoKey) != models.AlgoStatusPending {
			t.Errorf("expected pending, got %q", getAlgoStatus(asset, algoKey))
			return false
		}

		// Cleared fields should be gone (nil values delete from map in mock).
		for _, field := range []string{models.AlgoFieldReason, models.AlgoFieldStartedAt, models.AlgoFieldFinishedAt, models.AlgoFieldOutputURI} {
			if v := getAlgoField(asset, algoKey, field); v != "" {
				t.Errorf("expected field %s to be cleared, got %q", field, v)
				return false
			}
		}

		// cf_files[algo_key] should be cleared.
		if _, ok := asset.Files[algoKey]; ok {
			t.Error("expected cf_files[algo_key] to be cleared")
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 16 failed: %v", err)
	}
}

// ─── Property 17: All Lifecycle Operations Insert Event Records ─────────────
// **Validates: Requirements 5.4, 6.4, 7.3**
func TestProperty17_LifecycleOpsInsertEvents(t *testing.T) {
	reg := buildTestRegistry(t)
	ctx := context.Background()

	cfg := &quick.Config{MaxCount: 100, Rand: rand.New(rand.NewSource(time.Now().UnixNano()))}

	// Use env_analysis (no required fields, no report_size).
	algoKey := "env_analysis@1.0.0"

	f := func(seed uint8) bool {
		assetRepo := newMockAssetRepo()
		eventRepo := newMockEventRepo()
		uc := NewAlgoUsecase(assetRepo, eventRepo, reg)

		assetID := "asset-p17"
		assetRepo.assets[assetID] = makeAsset(assetID, map[string]string{
			algoKey + ":status": "pending",
		})

		// Start → should insert event.
		method := fmt.Sprintf("m%d", seed)
		err := uc.StartAlgo(ctx, assetID, algoKey, StartAlgoInput{Method: method})
		if err != nil {
			t.Errorf("StartAlgo failed: %v", err)
			return false
		}
		events := eventRepo.allEvents()
		if len(events) != 1 {
			t.Errorf("expected 1 event after start, got %d", len(events))
			return false
		}
		if events[0].NewStatus != string(models.AlgoStatusRunning) {
			t.Errorf("expected running event, got %q", events[0].NewStatus)
			return false
		}

		// Finish(failed) → should insert event.
		reason := "err"
		err = uc.FinishAlgo(ctx, assetID, algoKey, FinishAlgoInput{
			Status: "failed",
			Reason: &reason,
		})
		if err != nil {
			t.Errorf("FinishAlgo failed: %v", err)
			return false
		}
		events = eventRepo.allEvents()
		if len(events) != 2 {
			t.Errorf("expected 2 events after finish, got %d", len(events))
			return false
		}
		if events[1].NewStatus != string(models.AlgoStatusFailed) {
			t.Errorf("expected failed event, got %q", events[1].NewStatus)
			return false
		}

		// Reset → should insert event.
		err = uc.ResetAlgo(ctx, assetID, algoKey)
		if err != nil {
			t.Errorf("ResetAlgo failed: %v", err)
			return false
		}
		events = eventRepo.allEvents()
		if len(events) != 3 {
			t.Errorf("expected 3 events after reset, got %d", len(events))
			return false
		}
		if events[2].NewStatus != string(models.AlgoStatusPending) {
			t.Errorf("expected pending event, got %q", events[2].NewStatus)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 17 failed: %v", err)
	}
}

// ─── Property 18: Success Finish Missing Required Fields Rejected ───────────
// **Validates: Requirements 6.5, 6.6**
func TestProperty18_MissingRequiredFieldsRejected(t *testing.T) {
	reg := buildTestRegistry(t)
	ctx := context.Background()

	cfg := &quick.Config{MaxCount: 100, Rand: rand.New(rand.NewSource(time.Now().UnixNano()))}

	// Use hand_tracking which requires output_uri, type, and report_size=true.
	algoKey := "hand_tracking@1.2.0"

	f := func(seed uint8) bool {
		assetRepo := newMockAssetRepo()
		eventRepo := newMockEventRepo()
		uc := NewAlgoUsecase(assetRepo, eventRepo, reg)

		assetID := "asset-p18"
		assetRepo.assets[assetID] = makeAsset(assetID, map[string]string{
			algoKey + ":status": "running",
		})

		// Finish with ok but missing output_uri → should fail.
		err := uc.FinishAlgo(ctx, assetID, algoKey, FinishAlgoInput{
			Status: "ok",
		})
		if err == nil {
			t.Error("expected error for missing output_uri")
			return false
		}
		if !strings.Contains(err.Error(), "output_uri") && !strings.Contains(err.Error(), "missing") {
			t.Errorf("expected missing field error, got: %v", err)
			return false
		}

		// Finish with ok, output_uri provided but missing "type" extra field → should fail.
		uri := "gs://bucket/output"
		sizeBytes := int64(1024)
		err = uc.FinishAlgo(ctx, assetID, algoKey, FinishAlgoInput{
			Status:          "ok",
			OutputURI:       &uri,
			ResultSizeBytes: &sizeBytes,
			// Missing ExtraFields["type"]
		})
		if err == nil {
			t.Error("expected error for missing type field")
			return false
		}

		// Finish with ok, output_uri and type provided but missing result_size_bytes → should fail.
		err = uc.FinishAlgo(ctx, assetID, algoKey, FinishAlgoInput{
			Status:      "ok",
			OutputURI:   &uri,
			ExtraFields: map[string]interface{}{"type": "mcap"},
			// Missing ResultSizeBytes (report_size=true)
		})
		if err == nil {
			t.Error("expected error for missing result_size_bytes")
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 18 failed: %v", err)
	}
}

// ─── Property 21: finish_algo Idempotency (run_id dedup) ───────────────────
// **Validates: Requirements 16.1, 16.2**
func TestProperty21_FinishIdempotency(t *testing.T) {
	reg := buildTestRegistry(t)
	ctx := context.Background()

	cfg := &quick.Config{MaxCount: 100, Rand: rand.New(rand.NewSource(time.Now().UnixNano()))}

	algoKey := "env_analysis@1.0.0"

	f := func(seed uint8) bool {
		assetRepo := newMockAssetRepo()
		eventRepo := newMockEventRepo()
		uc := NewAlgoUsecase(assetRepo, eventRepo, reg)

		assetID := "asset-p21"
		runID := fmt.Sprintf("run-%d", seed)

		// Set up asset with algo already in ok status with matching run_id.
		assetRepo.assets[assetID] = makeAsset(assetID, map[string]string{
			algoKey + ":status": "ok",
			algoKey + ":run_id": runID,
		})

		// Finish with same run_id → should be idempotent (return nil).
		err := uc.FinishAlgo(ctx, assetID, algoKey, FinishAlgoInput{
			Status: "ok",
			RunID:  &runID,
		})
		if err != nil {
			t.Errorf("expected idempotent success, got: %v", err)
			return false
		}

		// No new events should be inserted for idempotent call.
		events := eventRepo.allEvents()
		if len(events) != 0 {
			t.Errorf("expected 0 events for idempotent call, got %d", len(events))
			return false
		}

		// Finish with different run_id → should fail with state transition error.
		differentRunID := runID + "-different"
		err = uc.FinishAlgo(ctx, assetID, algoKey, FinishAlgoInput{
			Status: "ok",
			RunID:  &differentRunID,
		})
		if err == nil {
			t.Error("expected error for different run_id on ok status")
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 21 failed: %v", err)
	}
}

// ─── Property 22: Dependency Unlocking Correctness ──────────────────────────
// **Validates: Requirements 15.3, 15.4**
func TestProperty22_DependencyUnlocking(t *testing.T) {
	reg := buildTestRegistry(t)
	ctx := context.Background()

	cfg := &quick.Config{MaxCount: 100, Rand: rand.New(rand.NewSource(time.Now().UnixNano()))}

	// action_annotation@1.0.0 depends on hand_tracking@1.2.0, head_tracking@1.0.0, body_tracking@1.0.0.
	downstreamKey := "action_annotation@1.0.0"
	deps := []string{"hand_tracking@1.2.0", "head_tracking@1.0.0", "body_tracking@1.0.0"}

	f := func(completedCount uint8) bool {
		// Complete 0 to 3 dependencies.
		numCompleted := int(completedCount) % 4

		assetRepo := newMockAssetRepo()
		eventRepo := newMockEventRepo()
		uc := NewAlgoUsecase(assetRepo, eventRepo, reg)

		assetID := "asset-p22"
		ar := map[string]string{
			downstreamKey + ":status": "blocked",
		}
		// Set completed deps to ok, rest to running.
		for i, dep := range deps {
			if i < numCompleted {
				ar[dep+":status"] = "ok"
			} else {
				ar[dep+":status"] = "running"
			}
		}
		assetRepo.assets[assetID] = makeAsset(assetID, ar)

		// Call tryUnblockDownstream with the last completed dep.
		if numCompleted > 0 {
			completedAlgo := deps[numCompleted-1]
			asset, _ := assetRepo.Get(ctx, assetID)
			err := uc.tryUnblockDownstream(ctx, asset, completedAlgo)
			if err != nil {
				t.Errorf("tryUnblockDownstream failed: %v", err)
				return false
			}
		}

		asset, _ := assetRepo.Get(ctx, assetID)
		downstreamStatus := getAlgoStatus(asset, downstreamKey)

		if numCompleted == 3 {
			// All deps ok → should be pending.
			if downstreamStatus != models.AlgoStatusPending {
				t.Errorf("expected pending when all deps ok, got %q", downstreamStatus)
				return false
			}
		} else {
			// Not all deps ok → should remain blocked.
			if downstreamStatus != models.AlgoStatusBlocked {
				t.Errorf("expected blocked when not all deps ok (completed=%d), got %q", numCompleted, downstreamStatus)
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 22 failed: %v", err)
	}
}

// ─── Property 23: cf_files Synced with cf_algo ──────────────────────────────
// **Validates: Requirements 12.3, 12.4**
func TestProperty23_CfFilesSyncedWithCfAlgo(t *testing.T) {
	reg := buildTestRegistry(t)
	ctx := context.Background()

	cfg := &quick.Config{MaxCount: 100, Rand: rand.New(rand.NewSource(time.Now().UnixNano()))}

	// Use env_analysis (no required fields, uri_required=false, report_size=false).
	algoKey := "env_analysis@1.0.0"

	f := func(seed uint8) bool {
		assetRepo := newMockAssetRepo()
		eventRepo := newMockEventRepo()
		uc := NewAlgoUsecase(assetRepo, eventRepo, reg)

		assetID := "asset-p23"
		assetRepo.assets[assetID] = makeAsset(assetID, map[string]string{
			algoKey + ":status": "running",
		})

		uri := fmt.Sprintf("gs://bucket/output-%d", seed)

		// Finish with ok and output_uri → cf_files[algo_key] should be set.
		err := uc.FinishAlgo(ctx, assetID, algoKey, FinishAlgoInput{
			Status:    "ok",
			OutputURI: &uri,
		})
		if err != nil {
			t.Errorf("FinishAlgo(ok) failed: %v", err)
			return false
		}

		asset, _ := assetRepo.Get(ctx, assetID)
		if asset.Files[algoKey] != uri {
			t.Errorf("expected cf_files[%s]=%q, got %q", algoKey, uri, asset.Files[algoKey])
			return false
		}

		// Now reset → cf_files[algo_key] should be cleared.
		err = uc.ResetAlgo(ctx, assetID, algoKey)
		if err != nil {
			t.Errorf("ResetAlgo failed: %v", err)
			return false
		}

		asset, _ = assetRepo.Get(ctx, assetID)
		if _, ok := asset.Files[algoKey]; ok {
			t.Errorf("expected cf_files[%s] to be cleared after reset", algoKey)
			return false
		}

		// Set to running again, then finish with failed → cf_files[algo_key] should be nil.
		assetRepo.mu.Lock()
		assetRepo.assets[assetID].AlgoResults[algoKey+":status"] = "running"
		assetRepo.mu.Unlock()

		reason := "test error"
		err = uc.FinishAlgo(ctx, assetID, algoKey, FinishAlgoInput{
			Status: "failed",
			Reason: &reason,
		})
		if err != nil {
			t.Errorf("FinishAlgo(failed) failed: %v", err)
			return false
		}

		asset, _ = assetRepo.Get(ctx, assetID)
		if _, ok := asset.Files[algoKey]; ok {
			t.Errorf("expected cf_files[%s] to be nil after failed finish", algoKey)
			return false
		}

		return true
	}

	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("Property 23 failed: %v", err)
	}
}

// ─── Property 19: Events Ordered by created_at DESC and Filterable by algo_key ─
// **Validates: Requirements 8.1, 8.2**
func TestProperty19_EventsOrderedAndFilterable(t *testing.T) {
	reg := buildTestRegistry(t)
	ctx := context.Background()

	cfg := &quick.Config{MaxCount: 100, Rand: rand.New(rand.NewSource(time.Now().UnixNano()))}

	algoKeys := validAlgoKeys(reg)

	f := func(numEvents uint8) bool {
		if len(algoKeys) < 2 {
			return true
		}
		count := int(numEvents)%10 + 2 // 2 to 11 events

		assetRepo := newMockAssetRepo()
		eventRepo := newMockEventRepo()
		uc := NewAlgoUsecase(assetRepo, eventRepo, reg)

		assetID := "asset-p19"
		assetRepo.assets[assetID] = makeAsset(assetID, nil)

		// Insert events with increasing timestamps, alternating algo keys.
		baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		for i := 0; i < count; i++ {
			algoKey := algoKeys[i%len(algoKeys)]
			event := &models.AlgoEvent{
				EventID:   fmt.Sprintf("evt-%d", i),
				AssetID:   assetID,
				AlgoKey:   algoKey,
				NewStatus: "running",
				CreatedAt: baseTime.Add(time.Duration(i) * time.Second),
			}
			_ = eventRepo.Insert(ctx, event)
		}

		// List all events → should be in reverse order (DESC).
		events, err := uc.ListAlgoEvents(ctx, assetID, nil)
		if err != nil {
			t.Errorf("ListAlgoEvents failed: %v", err)
			return false
		}
		if len(events) != count {
			t.Errorf("expected %d events, got %d", count, len(events))
			return false
		}
		for i := 1; i < len(events); i++ {
			if events[i].CreatedAt.After(events[i-1].CreatedAt) {
				t.Error("events not in DESC order")
				return false
			}
		}

		// Filter by specific algo_key → all returned events should match.
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
