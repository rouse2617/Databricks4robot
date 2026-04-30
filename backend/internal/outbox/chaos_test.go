package outbox

import (
	"context"
	"errors"
	"math/rand"
	"sync/atomic"
	"testing"
	"time"

	"data-platform/internal/models"
	"data-platform/internal/repository"
)

// ─── Chaos Test: Outbox Worker under adverse conditions ─────────────────────
// P2-T-2: Tests worker behavior under PG restarts, ES flapping, and
// concurrent failures.

// flappingEventRepo simulates intermittent PG failures (connection resets).
// Each call to ListPending has a configurable probability of failing.
type flappingEventRepo struct {
	stubEventRepo
	failRate float64 // 0.0 = never fail, 1.0 = always fail
	rng      *rand.Rand
}

func (r *flappingEventRepo) ListPending(ctx context.Context, limit int) ([]*models.AssetEvent, error) {
	if r.rng.Float64() < r.failRate {
		return nil, errors.New("chaos: pg connection reset by peer")
	}
	return r.stubEventRepo.ListPending(ctx, limit)
}

func (r *flappingEventRepo) CountPending(context.Context) (int64, error) {
	if r.rng.Float64() < r.failRate {
		return 0, errors.New("chaos: pg count failed")
	}
	return r.stubEventRepo.CountPending(context.Background())
}

func (r *flappingEventRepo) OldestPendingAge(context.Context) (float64, error) {
	return 0, nil
}

// TestChaos_FlappingPG verifies the worker survives intermittent PG failures
// and eventually processes all events when PG stabilizes.
func TestChaos_FlappingPG(t *testing.T) {
	// Create 20 non-asset events (will be skipped/published without ES).
	events := make([]*models.AssetEvent, 20)
	for i := range 20 {
		events[i] = &models.AssetEvent{
			EventSeq:      int64(i + 1),
			AggregateType: "non-asset",
		}
	}

	repo := &flappingEventRepo{
		stubEventRepo: stubEventRepo{pending: events},
		failRate:      0.3, // 30% failure rate
		rng:           rand.New(rand.NewSource(42)),
	}

	w := &ESWorker{
		Events:    repo,
		BatchSize: 5,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		w.Run(ctx, 5*time.Millisecond)
		close(done)
	}()

	// Wait for all events to be processed (with retries due to flapping).
	deadline := time.After(15 * time.Second)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			cancel()
			<-done
			t.Fatalf("timed out: only %d of 20 events published", len(repo.published))
		case <-ticker.C:
			if len(repo.published) >= 20 {
				cancel()
				<-done
				return
			}
		}
	}
}

// markPublishedFlappingRepo simulates MarkPublishedAndAdvanceCursor failures.
type markPublishedFlappingRepo struct {
	stubEventRepo
	markFailRate float64
	rng          *rand.Rand
	markCalls    atomic.Int32
}

func (r *markPublishedFlappingRepo) MarkPublishedAndAdvanceCursor(_ context.Context, seqs []int64, _ string) error {
	r.markCalls.Add(1)
	if r.rng.Float64() < r.markFailRate {
		return errors.New("chaos: cursor advance failed")
	}
	r.published = append(r.published, seqs...)
	return nil
}

func (r *markPublishedFlappingRepo) CountPending(context.Context) (int64, error) { return 0, nil }
func (r *markPublishedFlappingRepo) OldestPendingAge(context.Context) (float64, error) {
	return 0, nil
}

// TestChaos_CursorAdvanceFailure verifies the worker handles cursor advance
// failures gracefully (events stay pending for retry).
func TestChaos_CursorAdvanceFailure(t *testing.T) {
	events := make([]*models.AssetEvent, 10)
	for i := range 10 {
		events[i] = &models.AssetEvent{
			EventSeq:      int64(i + 1),
			AggregateType: "non-asset",
		}
	}

	repo := &markPublishedFlappingRepo{
		stubEventRepo: stubEventRepo{pending: events},
		markFailRate:  0.5, // 50% cursor advance failure
		rng:           rand.New(rand.NewSource(99)),
	}

	w := &ESWorker{
		Events:    repo,
		BatchSize: 5,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		w.Run(ctx, 5*time.Millisecond)
		close(done)
	}()

	// Let the worker run for a bit, then check it didn't crash.
	time.Sleep(2 * time.Second)
	cancel()
	<-done

	// The worker should have attempted multiple mark calls.
	if repo.markCalls.Load() == 0 {
		t.Fatal("expected at least one MarkPublishedAndAdvanceCursor call")
	}
	t.Logf("chaos: %d mark calls, %d events published", repo.markCalls.Load(), len(repo.published))
}

// rapidFireRepo generates events faster than the worker can consume them.
type rapidFireRepo struct {
	generated atomic.Int64
	published atomic.Int64
	batchSize int
}

func (r *rapidFireRepo) Append(context.Context, repository.AssetEventAppendInput) error {
	return nil
}

func (r *rapidFireRepo) ListPending(_ context.Context, limit int) ([]*models.AssetEvent, error) {
	n := limit
	if n > r.batchSize {
		n = r.batchSize
	}
	events := make([]*models.AssetEvent, n)
	for i := range n {
		seq := r.generated.Add(1)
		events[i] = &models.AssetEvent{
			EventSeq:      seq,
			AggregateType: "non-asset",
		}
	}
	return events, nil
}

func (r *rapidFireRepo) ListByAsset(context.Context, string, repository.AssetEventListOptions) ([]*models.AssetEvent, error) {
	return nil, nil
}

func (r *rapidFireRepo) MarkPublished(context.Context, []int64) error { return nil }
func (r *rapidFireRepo) MarkFailed(context.Context, int64, string) error {
	return nil
}
func (r *rapidFireRepo) CountPending(context.Context) (int64, error) { return 999, nil }
func (r *rapidFireRepo) ComputeSafeHorizon(context.Context) (int64, error) {
	return 0, nil
}

func (r *rapidFireRepo) MarkPublishedAndAdvanceCursor(_ context.Context, seqs []int64, _ string) error {
	r.published.Add(int64(len(seqs)))
	return nil
}

func (r *rapidFireRepo) OldestPendingAge(context.Context) (float64, error) {
	return 0, nil
}

// TestChaos_RapidFireEvents verifies the worker handles high event throughput
// without crashing and respects the maxDrainIterations safety limit.
func TestChaos_RapidFireEvents(t *testing.T) {
	repo := &rapidFireRepo{batchSize: 50}

	w := &ESWorker{
		Events:    repo,
		BatchSize: 50,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		w.Run(ctx, 10*time.Millisecond)
		close(done)
	}()

	// Let the worker run for 2 seconds under high load.
	time.Sleep(2 * time.Second)
	cancel()
	<-done

	published := repo.published.Load()
	generated := repo.generated.Load()

	t.Logf("chaos rapid-fire: generated=%d, published=%d", generated, published)

	// The worker should have processed a significant number of events.
	if published == 0 {
		t.Fatal("expected some events to be published under rapid-fire conditions")
	}

	// The drain loop safety limit should prevent infinite processing per tick.
	// With 50 events/batch and maxDrainIterations=100, max per tick = 5000.
	// Over 2 seconds with 10ms ticks, that's ~200 ticks × 5000 = 1M max.
	// This is a sanity check, not a precise bound.
	if published > 2_000_000 {
		t.Fatalf("published %d events — seems unreasonably high, safety limit may not be working", published)
	}
}

// TestChaos_WorkerSurvivesContextCancel verifies clean shutdown under load.
func TestChaos_WorkerSurvivesContextCancel(t *testing.T) {
	repo := &rapidFireRepo{batchSize: 10}
	health := &spyHealthMarker{}

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

	// Cancel immediately while the worker is processing.
	time.Sleep(10 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// Clean shutdown — good.
	case <-time.After(5 * time.Second):
		t.Fatal("worker did not shut down within 5 seconds after context cancel")
	}

	if health.called {
		t.Fatal("MarkUnhealthy should not be called during clean shutdown")
	}
}
