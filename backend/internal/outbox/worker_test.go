package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"data-platform/internal/elasticsearch"
	"data-platform/internal/models"
	"data-platform/internal/repository"
	"data-platform/internal/searchindex"
)

// --- stubs ---

// stubEventRepo is a minimal in-memory AssetEventRepository for unit tests.
type stubEventRepo struct {
	pending      []*models.AssetEvent // events returned by ListPending (consumed in order)
	batchSize    int                  // how many to return per call
	listErr      error                // if set, ListPending returns this error
	published    []int64              // seqs passed to MarkPublished
	failedSeqs   []int64              // seqs passed to MarkFailed
	listCalls    int                  // number of ListPending calls
	pendingCount int64                // returned by CountPending
}

func (s *stubEventRepo) Append(context.Context, repository.AssetEventAppendInput) error {
	return nil
}

func (s *stubEventRepo) ListPending(_ context.Context, limit int) ([]*models.AssetEvent, error) {
	s.listCalls++
	if s.listErr != nil {
		return nil, s.listErr
	}
	n := limit
	if n > len(s.pending) {
		n = len(s.pending)
	}
	batch := s.pending[:n]
	s.pending = s.pending[n:]
	return batch, nil
}

func (s *stubEventRepo) ListByAsset(context.Context, string, repository.AssetEventListOptions) ([]*models.AssetEvent, error) {
	return nil, nil
}

func (s *stubEventRepo) MarkPublished(_ context.Context, seqs []int64) error {
	s.published = append(s.published, seqs...)
	return nil
}

func (s *stubEventRepo) MarkFailed(_ context.Context, seq int64, _ string) error {
	s.failedSeqs = append(s.failedSeqs, seq)
	return nil
}

func (s *stubEventRepo) CountPending(context.Context) (int64, error) {
	return s.pendingCount, nil
}

func (s *stubEventRepo) ComputeSafeHorizon(context.Context) (int64, error) {
	return 0, nil
}

func (s *stubEventRepo) MarkPublishedAndAdvanceCursor(_ context.Context, seqs []int64, _ string) error {
	s.published = append(s.published, seqs...)
	return nil
}

func (s *stubEventRepo) OldestPendingAge(context.Context) (float64, error) {
	return 0, nil
}

// stubBuilder wraps a searchindex.Builder that always returns a fixed doc.
// We need a real Builder because ESWorker.Indexer is *searchindex.Builder (concrete type).
// Instead, we'll use a thin ES mock that accepts anything.

// stubES is a minimal ES client stand-in. Since ESWorker uses *elasticsearch.Client
// (concrete), we need a real HTTP server or to test processBatch indirectly.
// For drain-loop tests we focus on the Run/processBatch interaction via the
// event repo stub — the ES and Builder are exercised through integration tests.

// To test the drain loop without needing real ES/Builder, we create a wrapper
// that replaces processBatch behavior. But since processBatch is not an interface
// method, we test the exported Run behavior by observing ListPending call counts.

// makeEvents creates n pending asset events with sequential event_seq values.
func makeEvents(n int, assetID string) []*models.AssetEvent {
	events := make([]*models.AssetEvent, n)
	for i := range n {
		events[i] = &models.AssetEvent{
			EventID:       "evt-" + assetID + "-" + itoa(i),
			EventSeq:      int64(i + 1),
			EventType:     "asset.updated",
			AggregateType: "asset",
			AssetID:       assetID,
			PublishState:  "pending",
		}
	}
	return events
}

func itoa(i int) string {
	return string(rune('0'+i%10)) + string(rune('0'+i/10))
}

// --- processBatch unit tests ---

func TestProcessBatch_NoPending_ReturnsDone(t *testing.T) {
	repo := &stubEventRepo{}
	w := &ESWorker{
		Events:    repo,
		BatchSize: 10,
	}
	done, err := w.processBatch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !done {
		t.Fatal("expected done=true when no pending events")
	}
}

func TestProcessBatch_ListPendingError_ReturnsFalse(t *testing.T) {
	repo := &stubEventRepo{listErr: errors.New("pg connection lost")}
	w := &ESWorker{
		Events:    repo,
		BatchSize: 10,
	}
	done, err := w.processBatch(context.Background())
	if err == nil {
		t.Fatal("expected error from ListPending")
	}
	if done {
		t.Fatal("expected done=false on error")
	}
}

func TestProcessBatch_PartialBatch_ReturnsDone(t *testing.T) {
	// 3 events with batchSize=10 → partial batch → done=true
	// These events have empty AssetID so they'll be skipped (marked published).
	repo := &stubEventRepo{
		pending: []*models.AssetEvent{
			{EventSeq: 1, AggregateType: "non-asset"},
			{EventSeq: 2, AggregateType: "non-asset"},
			{EventSeq: 3, AggregateType: "non-asset"},
		},
	}
	w := &ESWorker{
		Events:    repo,
		BatchSize: 10,
	}
	done, err := w.processBatch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !done {
		t.Fatal("expected done=true for partial batch (3 < 10)")
	}
	if len(repo.published) != 3 {
		t.Fatalf("expected 3 published seqs, got %d", len(repo.published))
	}
}

func TestProcessBatch_FullBatch_ReturnsNotDone(t *testing.T) {
	// Exactly batchSize events → full batch → done=false (more may exist).
	// Use non-asset events so they get skipped without needing ES/Builder.
	events := make([]*models.AssetEvent, 5)
	for i := range 5 {
		events[i] = &models.AssetEvent{
			EventSeq:      int64(i + 1),
			AggregateType: "non-asset",
		}
	}
	repo := &stubEventRepo{pending: events}
	w := &ESWorker{
		Events:    repo,
		BatchSize: 5,
	}
	done, err := w.processBatch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if done {
		t.Fatal("expected done=false for full batch (5 == 5)")
	}
}

// --- drain loop tests (via Run) ---

// drainCountWorker wraps ESWorker to count drain iterations without needing
// real ES/Builder. It overrides the tick-based Run with a single-tick helper.
func drainOnce(t *testing.T, repo *stubEventRepo, batchSize int) int {
	t.Helper()
	w := &ESWorker{
		Events:    repo,
		BatchSize: batchSize,
	}
	iterations := 0
	for i := range maxDrainIterations {
		_ = i
		done, err := w.processBatch(context.Background())
		iterations++
		if err != nil {
			break
		}
		if done {
			break
		}
	}
	return iterations
}

func TestDrainLoop_SingleBatch(t *testing.T) {
	// 3 events, batchSize=10 → 1 iteration (partial batch → done)
	repo := &stubEventRepo{
		pending: []*models.AssetEvent{
			{EventSeq: 1, AggregateType: "non-asset"},
			{EventSeq: 2, AggregateType: "non-asset"},
			{EventSeq: 3, AggregateType: "non-asset"},
		},
	}
	iters := drainOnce(t, repo, 10)
	if iters != 1 {
		t.Fatalf("expected 1 drain iteration, got %d", iters)
	}
}

func TestDrainLoop_MultipleBatches(t *testing.T) {
	// 10 events, batchSize=3 → 4 iterations (3+3+3+1)
	events := make([]*models.AssetEvent, 10)
	for i := range 10 {
		events[i] = &models.AssetEvent{
			EventSeq:      int64(i + 1),
			AggregateType: "non-asset",
		}
	}
	repo := &stubEventRepo{pending: events}
	iters := drainOnce(t, repo, 3)
	// 3 full batches (3 each) + 1 partial batch (1 event) = 4 iterations
	if iters != 4 {
		t.Fatalf("expected 4 drain iterations, got %d", iters)
	}
	if len(repo.published) != 10 {
		t.Fatalf("expected 10 published seqs, got %d", len(repo.published))
	}
}

func TestDrainLoop_ErrorBreaksDrain(t *testing.T) {
	// First call succeeds (full batch), second call errors → drain stops at 2
	events := make([]*models.AssetEvent, 3)
	for i := range 3 {
		events[i] = &models.AssetEvent{
			EventSeq:      int64(i + 1),
			AggregateType: "non-asset",
		}
	}
	repo := &stubEventRepo{pending: events}
	w := &ESWorker{
		Events:    repo,
		BatchSize: 3,
	}

	iterations := 0
	for range maxDrainIterations {
		// After first successful batch, inject an error
		if iterations == 1 {
			repo.listErr = errors.New("connection reset")
		}
		done, err := w.processBatch(context.Background())
		iterations++
		if err != nil {
			break
		}
		if done {
			break
		}
	}
	if iterations != 2 {
		t.Fatalf("expected 2 iterations (1 success + 1 error), got %d", iterations)
	}
}

func TestDrainLoop_SafetyLimit(t *testing.T) {
	// Simulate infinite pending: ListPending always returns a full batch.
	// The drain loop should stop at maxDrainIterations.
	var calls atomic.Int32
	infiniteRepo := &infinitePendingRepo{calls: &calls, batchSize: 5}
	w := &ESWorker{
		Events:    infiniteRepo,
		BatchSize: 5,
	}

	iterations := 0
	for range maxDrainIterations {
		done, err := w.processBatch(context.Background())
		iterations++
		if err != nil || done {
			break
		}
	}
	if iterations != maxDrainIterations {
		t.Fatalf("expected drain to hit safety limit (%d), got %d iterations", maxDrainIterations, iterations)
	}
}

// infinitePendingRepo always returns a full batch of non-asset events.
type infinitePendingRepo struct {
	calls     *atomic.Int32
	batchSize int
}

func (r *infinitePendingRepo) Append(context.Context, repository.AssetEventAppendInput) error {
	return nil
}

func (r *infinitePendingRepo) ListPending(_ context.Context, limit int) ([]*models.AssetEvent, error) {
	r.calls.Add(1)
	events := make([]*models.AssetEvent, limit)
	for i := range limit {
		events[i] = &models.AssetEvent{
			EventSeq:      int64(r.calls.Load())*1000 + int64(i),
			AggregateType: "non-asset",
		}
	}
	return events, nil
}

func (r *infinitePendingRepo) ListByAsset(context.Context, string, repository.AssetEventListOptions) ([]*models.AssetEvent, error) {
	return nil, nil
}

func (r *infinitePendingRepo) MarkPublished(context.Context, []int64) error { return nil }
func (r *infinitePendingRepo) MarkFailed(context.Context, int64, string) error {
	return nil
}
func (r *infinitePendingRepo) CountPending(context.Context) (int64, error) { return 999, nil }
func (r *infinitePendingRepo) ComputeSafeHorizon(context.Context) (int64, error) {
	return 0, nil
}

func (r *infinitePendingRepo) MarkPublishedAndAdvanceCursor(context.Context, []int64, string) error {
	return nil
}

func (r *infinitePendingRepo) OldestPendingAge(context.Context) (float64, error) {
	return 0, nil
}

// --- verify maxDrainIterations constant ---

func TestMaxDrainIterations_IsReasonable(t *testing.T) {
	if maxDrainIterations < 1 {
		t.Fatal("maxDrainIterations must be at least 1")
	}
	if maxDrainIterations > 1000 {
		t.Fatal("maxDrainIterations seems unreasonably high")
	}
}

// --- verify Builder/ES are not needed for skip-only batches ---

func TestProcessBatch_SkipNonAssetEvents_NoBuilderNeeded(t *testing.T) {
	// Events with non-asset aggregate_type should be skipped (marked published)
	// without calling Builder or ES.
	repo := &stubEventRepo{
		pending: []*models.AssetEvent{
			{EventSeq: 1, AggregateType: "mcap"},
			{EventSeq: 2, AssetID: ""}, // empty asset_id also skipped
		},
	}
	w := &ESWorker{
		Events:    repo,
		Indexer:   nil, // nil — would panic if called
		ES:        nil, // nil — would panic if called
		BatchSize: 10,
	}
	done, err := w.processBatch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !done {
		t.Fatal("expected done=true")
	}
	if len(repo.published) != 2 {
		t.Fatalf("expected 2 published, got %d", len(repo.published))
	}
}

// --- panic recovery tests (§5.1) ---

// spyHealthMarker records MarkUnhealthy calls.
type spyHealthMarker struct {
	reason string
	called bool
}

func (s *spyHealthMarker) MarkUnhealthy(reason string) {
	s.called = true
	s.reason = reason
}

// panicOnFirstListRepo panics on the first ListPending call, simulating an
// unexpected runtime panic inside the drain loop.
type panicOnFirstListRepo struct {
	stubEventRepo
	panicMsg string
}

func (r *panicOnFirstListRepo) ListPending(_ context.Context, limit int) ([]*models.AssetEvent, error) {
	panic(r.panicMsg)
}

func (r *panicOnFirstListRepo) CountPending(context.Context) (int64, error) { return 0, nil }

func TestRun_PanicRecovery_MarksUnhealthy(t *testing.T) {
	health := &spyHealthMarker{}
	repo := &panicOnFirstListRepo{panicMsg: "unexpected nil pointer"}
	w := &ESWorker{
		Events:       repo,
		BatchSize:    10,
		Health:       health,
		FatalOnPanic: false,
	}

	// Run will panic on the first tick's processBatch → recover → return.
	// Use a very short tick so the test completes quickly.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		w.Run(ctx, 1*time.Millisecond)
		close(done)
	}()

	select {
	case <-done:
		// Run returned after panic recovery — good.
	case <-time.After(3 * time.Second):
		t.Fatal("Run did not return after panic recovery within timeout")
	}

	if !health.called {
		t.Fatal("expected Health.MarkUnhealthy to be called on panic")
	}
	if health.reason == "" {
		t.Fatal("expected non-empty reason in MarkUnhealthy")
	}
	if !contains(health.reason, "unexpected nil pointer") {
		t.Fatalf("expected reason to contain panic message, got: %s", health.reason)
	}
}

func TestRun_PanicRecovery_FatalOnPanic_CallsExit(t *testing.T) {
	health := &spyHealthMarker{}
	repo := &panicOnFirstListRepo{panicMsg: "segfault simulation"}

	var exitCode int
	exitCalled := make(chan struct{}, 1)

	w := &ESWorker{
		Events:       repo,
		BatchSize:    10,
		Health:       health,
		FatalOnPanic: true,
		osExit: func(code int) {
			exitCode = code
			exitCalled <- struct{}{}
			// Don't actually exit — just record the call.
			// We need to stop the goroutine, so panic with a sentinel.
			panic("osExit called")
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer func() {
			// Catch the sentinel panic from our fake osExit.
			recover()
			close(done)
		}()
		w.Run(ctx, 1*time.Millisecond)
	}()

	select {
	case <-exitCalled:
		// osExit was called — good.
	case <-time.After(3 * time.Second):
		t.Fatal("osExit was not called within timeout")
	}

	<-done

	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
	if !health.called {
		t.Fatal("expected Health.MarkUnhealthy to be called before os.Exit")
	}
}

func TestRun_PanicRecovery_NilHealth_NoPanic(t *testing.T) {
	// When Health is nil, panic recovery should still work without
	// a nil-pointer dereference on Health.MarkUnhealthy.
	repo := &panicOnFirstListRepo{panicMsg: "boom"}
	w := &ESWorker{
		Events:       repo,
		BatchSize:    10,
		Health:       nil, // no health marker configured
		FatalOnPanic: false,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		w.Run(ctx, 1*time.Millisecond)
		close(done)
	}()

	select {
	case <-done:
		// Returned cleanly — good.
	case <-time.After(3 * time.Second):
		t.Fatal("Run did not return after panic recovery with nil Health")
	}
}

func TestRun_NoPanic_HealthNotCalled(t *testing.T) {
	// Normal operation (no panic) should not trigger MarkUnhealthy.
	health := &spyHealthMarker{}
	repo := &stubEventRepo{} // empty — no pending events
	w := &ESWorker{
		Events:       repo,
		BatchSize:    10,
		Health:       health,
		FatalOnPanic: false,
	}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		w.Run(ctx, 1*time.Millisecond)
		close(done)
	}()

	// Let a few ticks run, then cancel.
	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Run did not return after context cancellation")
	}

	if health.called {
		t.Fatal("MarkUnhealthy should not be called during normal operation")
	}
}

// contains checks if s contains substr (simple helper to avoid strings import).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// --- backoff unit tests (§9) ---

func TestBackoffDuration_Sequence(t *testing.T) {
	// The backoff sequence per §9: 1s → 2s → 5s → 10s → 10s (capped).
	expected := []struct {
		failCount int
		want      time.Duration
	}{
		{0, 0},
		{1, 1 * time.Second},
		{2, 2 * time.Second},
		{3, 5 * time.Second},
		{4, 10 * time.Second},
		{5, 10 * time.Second},  // capped
		{10, 10 * time.Second}, // still capped
		{100, 10 * time.Second},
		{-1, 0}, // negative → no backoff
	}
	for _, tc := range expected {
		got := backoffDuration(tc.failCount)
		if got != tc.want {
			t.Errorf("backoffDuration(%d) = %v, want %v", tc.failCount, got, tc.want)
		}
	}
}

func TestSleepCtx_RespectsCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately
	err := sleepCtx(ctx, 10*time.Second)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

func TestSleepCtx_ZeroDuration(t *testing.T) {
	err := sleepCtx(context.Background(), 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSleepCtx_NegativeDuration(t *testing.T) {
	err := sleepCtx(context.Background(), -1*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSleepCtx_CompletesNormally(t *testing.T) {
	start := time.Now()
	err := sleepCtx(context.Background(), 10*time.Millisecond)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if elapsed := time.Since(start); elapsed < 5*time.Millisecond {
		t.Fatalf("sleep returned too quickly: %v", elapsed)
	}
}

// failNTimesRepo returns errors for the first N ListPending calls, then succeeds.
type failNTimesRepo struct {
	stubEventRepo
	failsRemaining int
	failErr        error
	callCount      int
}

func (r *failNTimesRepo) ListPending(ctx context.Context, limit int) ([]*models.AssetEvent, error) {
	r.callCount++
	if r.failsRemaining > 0 {
		r.failsRemaining--
		return nil, r.failErr
	}
	return r.stubEventRepo.ListPending(ctx, limit)
}

func (r *failNTimesRepo) CountPending(context.Context) (int64, error) { return 0, nil }

func TestRun_BackoffResetsOnSuccess(t *testing.T) {
	// Fail twice, then succeed. After success the backoff counter should reset.
	// We verify by checking that the worker continues to process after failures.
	repo := &failNTimesRepo{
		failsRemaining: 2,
		failErr:        errors.New("pg connection lost"),
		stubEventRepo: stubEventRepo{
			pending: []*models.AssetEvent{
				{EventSeq: 1, AggregateType: "non-asset"},
			},
		},
	}
	w := &ESWorker{
		Events:    repo,
		BatchSize: 10,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		w.Run(ctx, 5*time.Millisecond)
		close(done)
	}()

	// Wait enough time for a few ticks + backoff sleeps (2 failures × ~1-2s each + success tick).
	// The first failure waits 1s, second waits 2s, then success.
	deadline := time.After(8 * time.Second)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			cancel()
			<-done
			t.Fatalf("timed out waiting for successful processing; repo calls=%d, published=%d",
				repo.callCount, len(repo.published))
		case <-ticker.C:
			if len(repo.published) > 0 {
				// Success: the worker recovered from failures and processed events.
				cancel()
				<-done
				return
			}
		}
	}
}

func TestRun_BackoffCancelledDuringWait(t *testing.T) {
	// If context is cancelled during backoff sleep, Run should return promptly.
	repo := &stubEventRepo{listErr: errors.New("pg down")}
	w := &ESWorker{
		Events:    repo,
		BatchSize: 10,
	}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		w.Run(ctx, 1*time.Millisecond)
		close(done)
	}()

	// Let the first tick fire and the worker enter backoff sleep (1s).
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// Run returned promptly after cancel — good.
	case <-time.After(3 * time.Second):
		t.Fatal("Run did not return promptly after context cancellation during backoff")
	}
}

// Ensure the test file compiles with the searchindex import (used by ESWorker).
var _ *searchindex.Builder

// --- partial failure tests (§5.1 step 5) ---

// stubAssetRepoForWorker implements repository.AssetRepository for worker tests.
type stubAssetRepoForWorker struct {
	assets map[string]*models.Asset
}

func (s *stubAssetRepoForWorker) Get(_ context.Context, assetID string) (*models.Asset, error) {
	a, ok := s.assets[assetID]
	if !ok {
		return nil, nil
	}
	return a, nil
}
func (s *stubAssetRepoForWorker) Set(context.Context, *models.Asset) error { return nil }
func (s *stubAssetRepoForWorker) SoftDelete(context.Context, string) error { return nil }
func (s *stubAssetRepoForWorker) ListByMcapFile(context.Context, string) ([]*models.Asset, error) {
	return nil, nil
}
func (s *stubAssetRepoForWorker) WriteSegmentIndex(context.Context, *models.Asset) error {
	return nil
}
func (s *stubAssetRepoForWorker) ListWithFilters(context.Context, string, []interface{}, int, int, string) ([]*models.Asset, int64, error) {
	return nil, 0, nil
}

// stubTagRepo implements repository.AssetTagRepository for worker tests.
type stubTagRepo struct{}

func (s *stubTagRepo) Upsert(context.Context, string, string, string, string, string) error {
	return nil
}
func (s *stubTagRepo) ListByAsset(context.Context, string) ([]*models.AssetTag, error) {
	return nil, nil
}
func (s *stubTagRepo) Delete(context.Context, string, string) error { return nil }

// stubAlgoRepo implements repository.AssetAlgoLatestRepository for worker tests.
type stubAlgoRepo struct{}

func (s *stubAlgoRepo) Upsert(context.Context, *models.AssetAlgoLatest) error { return nil }
func (s *stubAlgoRepo) GetByAlgo(context.Context, string, string) (*models.AssetAlgoLatest, error) {
	return nil, nil
}
func (s *stubAlgoRepo) ListByAsset(context.Context, string) ([]*models.AssetAlgoLatest, error) {
	return nil, nil
}

// stubMcapRepo implements repository.McapFileRepository for worker tests.
type stubMcapRepo struct{}

func (s *stubMcapRepo) Get(context.Context, string) (*models.McapFile, error) { return nil, nil }
func (s *stubMcapRepo) Set(context.Context, *models.McapFile) error           { return nil }
func (s *stubMcapRepo) UpdateIngestState(context.Context, string, models.IngestState) error {
	return nil
}
func (s *stubMcapRepo) List(context.Context, int, int, string, string) ([]*models.McapFile, int64, error) {
	return nil, 0, nil
}

// newFakeESServer creates an httptest.Server that returns a configurable _bulk
// response. failIDs is the set of doc IDs that should be reported as failed.
func newFakeESServer(t *testing.T, failIDs map[string]bool) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		lines := strings.Split(strings.TrimSpace(string(body)), "\n")

		type bulkItem struct {
			Index struct {
				ID     string      `json:"_id"`
				Status int         `json:"status"`
				Error  interface{} `json:"error,omitempty"`
			} `json:"index"`
		}
		var items []bulkItem
		hasErrors := false
		// lines come in pairs: action + doc
		for i := 0; i < len(lines)-1; i += 2 {
			var action struct {
				Index struct {
					ID string `json:"_id"`
				} `json:"index"`
			}
			_ = json.Unmarshal([]byte(lines[i]), &action)
			item := bulkItem{}
			item.Index.ID = action.Index.ID
			if failIDs[action.Index.ID] {
				item.Index.Status = 400
				item.Index.Error = map[string]string{
					"type":   "mapper_parsing_exception",
					"reason": "failed to parse field",
				}
				hasErrors = true
			} else {
				item.Index.Status = 200
			}
			items = append(items, item)
		}

		resp := map[string]interface{}{
			"errors": hasErrors,
			"items":  items,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

func TestProcessBatch_PartialBulkFailure_SplitsPublishedAndFailed(t *testing.T) {
	// Set up: 3 assets, each with events. Asset "b" will fail in ES bulk.
	assetRepo := &stubAssetRepoForWorker{
		assets: map[string]*models.Asset{
			"a": {AssetID: "a", LifecycleState: "active"},
			"b": {AssetID: "b", LifecycleState: "active"},
			"c": {AssetID: "c", LifecycleState: "active"},
		},
	}

	eventRepo := &stubEventRepo{
		pending: []*models.AssetEvent{
			{EventSeq: 1, AggregateType: "asset", AssetID: "a"},
			{EventSeq: 2, AggregateType: "asset", AssetID: "a"},
			{EventSeq: 3, AggregateType: "asset", AssetID: "b"},
			{EventSeq: 4, AggregateType: "asset", AssetID: "c"},
		},
	}

	failIDs := map[string]bool{"b": true}
	esServer := newFakeESServer(t, failIDs)
	defer esServer.Close()

	esClient := elasticsearch.New(esServer.URL, "assets")
	indexer := &searchindex.Builder{
		Assets: assetRepo,
		Tags:   &stubTagRepo{},
		Algos:  &stubAlgoRepo{},
		Mcap:   &stubMcapRepo{},
	}

	w := &ESWorker{
		Events:    eventRepo,
		Indexer:   indexer,
		ES:        esClient,
		BatchSize: 10,
	}

	done, err := w.processBatch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !done {
		t.Fatal("expected done=true (partial batch < batchSize)")
	}

	// Asset "a" (seqs 1,2) and "c" (seq 4) should be published.
	// Asset "b" (seq 3) should be failed.
	if len(eventRepo.published) != 3 {
		t.Fatalf("expected 3 published seqs (a:1,2 + c:4), got %d: %v",
			len(eventRepo.published), eventRepo.published)
	}
	if len(eventRepo.failedSeqs) != 1 {
		t.Fatalf("expected 1 failed seq (b:3), got %d: %v",
			len(eventRepo.failedSeqs), eventRepo.failedSeqs)
	}
	if eventRepo.failedSeqs[0] != 3 {
		t.Fatalf("expected failed seq=3, got %d", eventRepo.failedSeqs[0])
	}
}

func TestProcessBatch_AllBulkSuccess_AllPublished(t *testing.T) {
	// All docs succeed in ES bulk — no partial failure.
	assetRepo := &stubAssetRepoForWorker{
		assets: map[string]*models.Asset{
			"x": {AssetID: "x", LifecycleState: "active"},
			"y": {AssetID: "y", LifecycleState: "active"},
		},
	}

	eventRepo := &stubEventRepo{
		pending: []*models.AssetEvent{
			{EventSeq: 10, AggregateType: "asset", AssetID: "x"},
			{EventSeq: 11, AggregateType: "asset", AssetID: "y"},
		},
	}

	esServer := newFakeESServer(t, nil) // no failures
	defer esServer.Close()

	esClient := elasticsearch.New(esServer.URL, "assets")
	indexer := &searchindex.Builder{
		Assets: assetRepo,
		Tags:   &stubTagRepo{},
		Algos:  &stubAlgoRepo{},
		Mcap:   &stubMcapRepo{},
	}

	w := &ESWorker{
		Events:    eventRepo,
		Indexer:   indexer,
		ES:        esClient,
		BatchSize: 10,
	}

	done, err := w.processBatch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !done {
		t.Fatal("expected done=true")
	}

	if len(eventRepo.published) != 2 {
		t.Fatalf("expected 2 published seqs, got %d", len(eventRepo.published))
	}
	if len(eventRepo.failedSeqs) != 0 {
		t.Fatalf("expected 0 failed seqs, got %d", len(eventRepo.failedSeqs))
	}
}

func TestProcessBatch_AllBulkFailed_AllDeferred(t *testing.T) {
	// All docs fail in ES bulk — all events should be marked failed.
	assetRepo := &stubAssetRepoForWorker{
		assets: map[string]*models.Asset{
			"p": {AssetID: "p", LifecycleState: "active"},
			"q": {AssetID: "q", LifecycleState: "active"},
		},
	}

	eventRepo := &stubEventRepo{
		pending: []*models.AssetEvent{
			{EventSeq: 20, AggregateType: "asset", AssetID: "p"},
			{EventSeq: 21, AggregateType: "asset", AssetID: "q"},
		},
	}

	failIDs := map[string]bool{"p": true, "q": true}
	esServer := newFakeESServer(t, failIDs)
	defer esServer.Close()

	esClient := elasticsearch.New(esServer.URL, "assets")
	indexer := &searchindex.Builder{
		Assets: assetRepo,
		Tags:   &stubTagRepo{},
		Algos:  &stubAlgoRepo{},
		Mcap:   &stubMcapRepo{},
	}

	w := &ESWorker{
		Events:    eventRepo,
		Indexer:   indexer,
		ES:        esClient,
		BatchSize: 10,
	}

	done, err := w.processBatch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !done {
		t.Fatal("expected done=true")
	}

	// No events should be published (all docs failed).
	if len(eventRepo.published) != 0 {
		t.Fatalf("expected 0 published seqs, got %d: %v",
			len(eventRepo.published), eventRepo.published)
	}
	// Both events should be marked failed.
	if len(eventRepo.failedSeqs) != 2 {
		t.Fatalf("expected 2 failed seqs, got %d: %v",
			len(eventRepo.failedSeqs), eventRepo.failedSeqs)
	}
}
