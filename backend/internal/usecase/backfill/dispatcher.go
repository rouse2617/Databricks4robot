// Package backfill — Phase 4 persistent dispatcher (outbox).
//
// See openspec/changes/CYB-RUN-DIAGNOSIS-REFACTOR/PHASE4-DESIGN.md.
//
// The dispatcher drives the backfill_items outbox state machine:
//
//	pending → claimed → submitting → submitted (absorbing)
//	               │       │
//	               ▼       ▼
//	              failed (retryable, lease holds backoff)
//	               │
//	               ▼ (attempts >= MaxAttempts)
//	              dead (absorbing)
//
// Architecture (Decision A, 2026-07-05):
//
//	┌──► worker[0]  ──► submitOne ──┐
//	│                                │
//	├──► worker[1]  ──► submitOne ──┼─► Argo
//	ticker ─items─► ├──► worker[2]  ──► submitOne ──┤
//	│                                │
//	└──► worker[N-1]──► submitOne ──┘
//
// A single ticker claims one item per tick via the FOR UPDATE SKIP LOCKED
// CTE in ClaimNextDispatch, pushes it to a bounded in-memory channel,
// and lets one of N workers dispatch it. Multi-replica safety comes
// purely from SKIP LOCKED + lease — no leader election.
//
// Backoff (Decision B): the lease field IS the backoff window.
// MarkDispatchFailedRetryable parks the row at NOW()+backoff, and
// ClaimNextDispatch's WHERE clause `(lease IS NULL OR lease < NOW())`
// excludes it until the backoff expires. No separate `next_retry_at`
// column needed.
//
// This file is Commit A: dispatcher exists, has correct behaviour, but
// no entry point in the codebase has been wired to write
// dispatch_state='pending' yet. Entries still flow through the legacy
// runItems path. Commit B switches the entries; Commit C deletes the
// legacy path. Only the dispatcher remains.
package backfill

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"sync"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

// rng is the package-local pseudo-random source. The seed is fixed
// at startup so jitter is reproducible across runs (test stability);
// backoff intervals are seconds-scale and the source has plenty
// of internal state, so production de-correlation is unaffected.
//
// A *rand.Rand is NOT safe for concurrent use, and backoffForAttempt
// runs from N worker goroutines — rngMu serialises access.
var (
	rngMu sync.Mutex
	rng   = rand.New(rand.NewSource(0xdabbad00))
)

// TaskHandler encapsulates the business logic of processing a claimed backfill item.
// The Dispatcher is decoupled from concrete business implementations via this interface.
type TaskHandler interface {
	// Handle processes a single backfill item. It is responsible for:
	// - Reading job context (template, parameters)
	// - Invoking the appropriate business handler (e.g., workflow deployment)
	// - Transitioning the item to the correct state (submitted, failed, or dead)
	// - Returning error only if the item processing failed catastrophically (not retryable)
	Handle(ctx context.Context, item *models.BackfillItem) error
}

// DispatcherConfig is the tunables for the dispatcher loop.
// Defaults are enforced by `DispatcherConfig.normalized()`. All times are
// seconds except where time.Duration is taken directly.
type DispatcherConfig struct {
	// Tick is the dispatch-loop cadence (how often we attempt to claim + run reaper).
	// Default 5s.
	Tick time.Duration

	// LeaseSec is the duration a dispatcher claim is held before the reaper
	// considers it stale. Default 60s. Must be > Deploy submit p99 + a margin.
	LeaseSec int

	// MaxAttempts is the per-row failure threshold before transitioning to
	// the absorbing 'dead' state. Default 3.
	MaxAttempts int

	// WorkerCount is the number of concurrent submit goroutines reading
	// from the bounded jobs channel. Default 5.
	WorkerCount int

	// BatchSize is the maximum claims per tick (drives how many items are
	// in flight concurrently per replica per tick). Default 4. Should be
	// <= WorkerCount.
	BatchSize int

	// JobBufferSize is the bounded channel capacity. Claims block (drop)
	// when the channel is full. Default 64.
	JobBufferSize int

	// BackoffBase / BackoffMax drive the exponential-backoff schedule on
	// retryable failures (Decision B). Default 1s / 30s.
	BackoffBase time.Duration
	BackoffMax  time.Duration

	// InstanceID is the per-instance identifier embedded in every
	// slog line this dispatcher emits. Auto-generated if empty.
	InstanceID string
}

// normalized fills defaults for any zero-valued fields.
func (c DispatcherConfig) normalized() DispatcherConfig {
	if c.Tick <= 0 {
		c.Tick = 5 * time.Second
	}
	if c.LeaseSec <= 0 {
		c.LeaseSec = 60
	}
	if c.MaxAttempts <= 0 {
		c.MaxAttempts = 3
	}
	if c.WorkerCount <= 0 {
		c.WorkerCount = 5
	}
	if c.BatchSize <= 0 {
		c.BatchSize = 4
	}
	if c.JobBufferSize <= 0 {
		c.JobBufferSize = 64
	}
	if c.BackoffBase <= 0 {
		c.BackoffBase = 1 * time.Second
	}
	if c.BackoffMax <= 0 {
		c.BackoffMax = 30 * time.Second
	}
	if c.InstanceID == "" {
		c.InstanceID = fmt.Sprintf("dispatcher-%d", rand.Int63())
	}
	if c.BatchSize > c.WorkerCount {
		c.BatchSize = c.WorkerCount
	}
	return c
}

// Dispatcher is the long-running instance of the outbox dispatch loop.
// It is a purely generic, orchestration-agnostic Outbox engine that:
// - Claims items atomically via FOR UPDATE SKIP LOCKED
// - Manages lease lifetimes and exponential backoff windows
// - Distributes claimed items to a worker pool via buffered channel
// - Reaps stale leases via the configured reaper interval
//
// Business logic (what to do with each item) is completely decoupled
// via the TaskHandler interface, eliminating circular dependencies and
// enabling horizontal extension to new runtimes (K8s, Databricks, etc).
type Dispatcher struct {
	repo    repository.BackfillRepository
	handler TaskHandler // Decoupled via interface, not concrete implementation
	cfg     DispatcherConfig

	jobs   chan *models.BackfillItem
	stopCh chan struct{}
	wg     sync.WaitGroup

	// tickerStopped closes after the ticker goroutine has exited, signalling
	// the workers that no new items will be claimed (they should drain the
	// channel and exit).
	tickerStopped chan struct{}

	logger *slog.Logger
}

// NewDispatcher builds the dispatcher struct. The loop is not started
// until Start is called.
func NewDispatcher(repo repository.BackfillRepository, handler TaskHandler, cfg DispatcherConfig) *Dispatcher {
	cfg = cfg.normalized()
	return &Dispatcher{
		repo:          repo,
		handler:       handler,
		cfg:           cfg,
		jobs:          make(chan *models.BackfillItem, cfg.JobBufferSize),
		stopCh:        make(chan struct{}),
		tickerStopped: make(chan struct{}),
		logger:        slog.Default().With(slog.String("component", "backfill-dispatcher"), slog.String("instance", cfg.InstanceID)),
	}
}

// Start launches the ticker + worker goroutines. Returns immediately;
// the goroutines run until Stop is called or ctx is done.
//
// Caller is responsible for calling Stop on shutdown so the loop drains
// gracefully (no leaked lease > leaseSec, no in-flight submit lost).
func (d *Dispatcher) Start(ctx context.Context) {
	d.logger.Info("dispatcher starting",
		"tick", d.cfg.Tick, "lease_sec", d.cfg.LeaseSec,
		"workers", d.cfg.WorkerCount, "batch_size", d.cfg.BatchSize,
		"job_buffer", d.cfg.JobBufferSize, "max_attempts", d.cfg.MaxAttempts)

	// Ticker goroutine: one is enough (single source of fairness).
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		defer close(d.tickerStopped)
		defer close(d.jobs) // workers see closed channel after drain
		d.tickerLoop(ctx)
	}()

	// Worker goroutines.
	for i := 0; i < d.cfg.WorkerCount; i++ {
		d.wg.Add(1)
		go func(workerIdx int) {
			defer d.wg.Done()
			d.workerLoop(workerIdx)
		}(i)
	}
}

// Stop signals the dispatcher to begin draining. Goroutines do not
// block forever — Stop returns once both the ticker and all workers exit
// (within tick + lease).
func (d *Dispatcher) Stop() {
	close(d.stopCh)
	d.wg.Wait()
	d.logger.Info("dispatcher stopped")
}

// tickerLoop is the single producer end of the channel. Per tick:
//  1. Run the reaper (ResetStaleDispatchedItems)
//  2. Claim up to BatchSize items, push each to jobs (drop on full)
//
// Exits when ctx is done OR stopCh is closed. Exits close tickerStopped
// (which the workers don't actually consume, but it's an external
// signal) and jobs (which is what causes the workers to exit).
func (d *Dispatcher) tickerLoop(ctx context.Context) {
	// Run one immediate cycle on start so existing pending items are
	// claimed without waiting for the first tick.
	d.tick(ctx)
	t := time.NewTicker(d.cfg.Tick)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			d.logger.Info("ticker exit: ctx done")
			return
		case <-d.stopCh:
			d.logger.Info("ticker exit: stop")
			return
		case <-t.C:
			d.tick(ctx)
		}
	}
}

// tick runs one dispatcher cycle: reaper, then claim-and-fill.
//
// Called by tickerLoop on Tick cadence, plus once at ticker start.
func (d *Dispatcher) tick(ctx context.Context) {
	// 1. Reaper first — releases stuck lease rows so today's items flow.
	if _, err := d.repo.ResetStaleDispatchedItems(ctx, d.cfg.LeaseSec, d.cfg.MaxAttempts); err != nil {
		d.logger.Warn("dispatcher reaper failed", "err", err)
	}

	// 2. Claim up to BatchSize items. Stop at first empty candidate set
	//    so a quiet batch doesn't waste connection budget.
	for i := 0; i < d.cfg.BatchSize; i++ {
		select {
		case <-ctx.Done():
			return
		case <-d.stopCh:
			return
		default:
		}

		item, err := d.repo.ClaimNextDispatch(ctx, d.cfg.LeaseSec, d.cfg.MaxAttempts)
		if err != nil {
			d.logger.Warn("claim dispatch", "err", err)
			// Pause briefly to avoid hammering the DB on persistent errors.
			time.Sleep(time.Second)
			continue
		}
		if item == nil {
			return // queue empty for now
		}

		// Push to channel. If the channel is full, the lease will
		// naturally expire (leaseSec). Drop is the documented backpressure
		// policy — see Decision A in PHASE4-DESIGN.md.
		select {
		case d.jobs <- item:
			d.logger.Debug("dispatch queued", "item_id", item.ID, "asset_id", item.AssetID, "wf", item.WorkflowNamePlanned)
		default:
			// Channel full → this replica is saturated (backpressure). STOP
			// claiming for this tick: continuing would strand up to BatchSize
			// freshly-claimed rows in 'claimed' (lease held) that no other
			// healthy replica can pick up until the lease expires. Leave the
			// rest of the queue for replicas with free workers; the reaper
			// reclaims the one row we just claimed once its lease lapses.
			d.logger.Warn("dispatch channel full, backing off ticker (lease will expire)",
				"item_id", item.ID, "buffer_size", d.cfg.JobBufferSize)
			return
		}
	}
}

// workerLoop is one of N consumers. Reads from d.jobs and processes one
// item at a time. Exits when d.jobs is closed (ticker exited).
func (d *Dispatcher) workerLoop(idx int) {
	logger := d.logger.With(slog.Int("worker", idx))
	for item := range d.jobs { // closed by ticker on exit
		select {
		case <-d.stopCh:
			// Outer shutdown: process the item we already pulled
			// (its lease is alive — must complete or the lease
			// will expire and the reaper will reset it).
			d.processItem(logger, item)
			continue
		default:
			d.processItem(logger, item)
		}
	}
	logger.Debug("worker exit: channel closed")
}

// processItem is the pure orchestration gateway: it creates a lease-scoped context
// and delegates all business logic to the injected TaskHandler. This keeps the
// Dispatcher agnostic to concrete business implementations and eliminates
// circular dependencies.
func (d *Dispatcher) processItem(logger *slog.Logger, item *models.BackfillItem) {
	logger.Info(">>> PROCESSING ITEM", "item_id", item.ID, "job_id", item.JobID)
	// Per-item context with the lease's remaining lifetime as the deadline.
	// -5 seconds for safety margin between worker timeout and lease expiry.
	deadline := time.Duration(d.cfg.LeaseSec-5) * time.Second
	if deadline <= 0 {
		deadline = time.Duration(d.cfg.LeaseSec) * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), deadline)
	defer cancel()

	// Delegate all business logic to the handler. The handler is responsible for:
	// - Reading job context and resolving runtime parameters
	// - Invoking business services (e.g., workflow deployment)
	// - Transitioning item states (submitted, failed, or dead)
	if err := d.handler.Handle(ctx, item); err != nil {
		// If handler.Handle returns non-nil, it signals a transient failure;
		// the handler itself manages state transitions, so we just log here.
		logger.Warn("handler returned error", "item_id", item.ID, "err", err)
	}
	logger.Info(">>> FINISHED ITEM", "item_id", item.ID)
}
// NOTE: Backoff, error classification, and failure handling logic has been
// moved to BackfillTaskHandler. The Dispatcher is now a pure orchestration
// kernel with no business logic dependencies.

// deriveWfNameFallback is a defence-in-depth path: if a row reaches the
// dispatcher without workflow_name_planned populated (e.g. legacy
// migration drift), synthesise a deterministic name from stable inputs.
// Should rarely fire; log via the caller's logger.
func deriveWfNameFallback(item *models.BackfillItem) string {
	// Trim, lowercase, replace non-allowed with '-'. K8s DNS-1123-ish.
	sanitise := func(s string) string {
		out := make([]rune, 0, len(s))
		for _, r := range s {
			switch {
			case r >= 'a' && r <= 'z':
				out = append(out, r)
			case r >= '0' && r <= '9':
				out = append(out, r)
			default:
				out = append(out, '-')
			}
		}
		return string(out)
	}
	job := sanitise(item.JobID)
	id := item.ID
	if len(id) > 8 {
		id = id[len(id)-8:]
	}
	return fmt.Sprintf("dispatcher-%s-%s-g%d", job, id, item.DispatchGeneration)
}
