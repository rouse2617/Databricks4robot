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
	"errors"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"strings"
	"sync"
	"time"

	pipelineUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/pipeline"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)


// rng is the package-local pseudo-random source. The seed is fixed
// at startup so jitter is reproducible across runs (test stability);
// backoff intervals are seconds-scale and the source has plenty
// of internal state, so production de-correlation is unaffected.
var rng = rand.New(rand.NewSource(0xdabbad00))

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

// Dispatcher is the long-running instance of the dispatch loop.
// One should be created and started per backend replica.
type Dispatcher struct {
	repo       repository.BackfillRepository
	pipelineUC *pipelineUC.Usecase
	cfg        DispatcherConfig

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
func NewDispatcher(repo repository.BackfillRepository, pipelineUC *pipelineUC.Usecase, cfg DispatcherConfig) *Dispatcher {
	cfg = cfg.normalized()
	return &Dispatcher{
		repo:          repo,
		pipelineUC:    pipelineUC,
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
			d.logger.Warn("dispatch channel full, dropping claim (lease will expire)",
				"item_id", item.ID, "buffer_size", d.cfg.JobBufferSize)
			// Note: we don't release the lease ourselves here. The
			// reaper on the next tick will reset it back to 'pending'.
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

// processItem is the per-item workflow: heartbeat → Deploy → outcome
// dispatch. The Phase 1 idempotency hinge (Deploy.Phase 1.5 adopts on
// AlreadyExists, rejects on Terminating) is invoked implicitly via
// DeployByTemplateID. The Phase 1 EOFError / 409 detection in
// submitRuntimeWorkflow means we don't need to special-case anything
// here — pipe the Outcome out via the state machine.
func (d *Dispatcher) processItem(logger *slog.Logger, item *models.BackfillItem) {
	// Per-item context with the lease's remaining lifetime as the deadline.
	// -5 seconds for safety margin between worker timeout and lease expiry.
	deadline := time.Duration(d.cfg.LeaseSec-5) * time.Second
	if deadline <= 0 {
		deadline = time.Duration(d.cfg.LeaseSec) * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), deadline)
	defer cancel()

	// 1. Refresh lease + state=submitting (heartbeat before slow submit).
	if err := d.repo.MarkDispatchSubmitting(ctx, item.ID, d.cfg.LeaseSec); err != nil {
		logger.Warn("mark submitting", "item_id", item.ID, "err", err)
		return // leave lease, reaper will reset
	}

	// 2. Resolve deterministic workflow name. Phase 1's batch
	// placeholder is already deterministic per-asset; if it's
	// missing (rare: a row mutated externally), derive a fresh one.
	wfName := strings.TrimSpace(item.WorkflowNamePlanned)
	if wfName == "" && item.PipelineRunID != nil {
		// Fallback: only happens if migration ran without our setter.
		wfName = deriveWfNameFallback(item)
	}

	// 3. Resolve runtime params from job filter_json (best-effort).
	targetID := ""
	templateVersion := 0
	if item.JobID != "" {
		if job, _ := d.repo.FindJobByID(ctx, item.JobID); job != nil {
			templateVersion = job.TemplateVersion
			if f, ok := job.FilterJSON["targetId"].(string); ok {
				targetID = strings.TrimSpace(f)
			}
		}
	}

	// 4. Update runtime fields on the row so we have a self-consistent
	// record before submit (helps debug if submit crashes the worker).
	_ = d.repo.UpdateItemDispatchFields(ctx, item.ID, "", templateVersion, targetID, d.cfg.LeaseSec)

	// 5. Submit via the Phase 1 idempotency hinge.
	preallocRunID := ""
	if item.PipelineRunID != nil {
		preallocRunID = strings.TrimSpace(*item.PipelineRunID)
	}
	dep, err := d.pipelineUC.DeployByTemplateID(ctx, "", "", []string{item.AssetID}, pipelineUC.DeployOptions{
		BatchJobID:               item.JobID,
		TemplateVersion:          templateVersion,
		TargetID:                 targetID,
		PreallocatedRunID:        preallocRunID,
		PreallocatedWorkflowName: wfName,
		AllowUnknownAssets:       true,
	})

	// 6. Outcome dispatch.
	switch {
	case err == nil:
		argoUID := ""
		argoName := wfName
		if dep != nil {
			argoUID = dep.ID
			if n := strings.TrimSpace(dep.WorkflowName); n != "" {
				argoName = n
			}
		}
		if mErr := d.repo.MarkDispatched(ctx, item.ID, argoName, argoUID); mErr != nil {
			logger.Warn("mark dispatched", "item_id", item.ID, "err", mErr)
		}
		// Sync the legacy columns too (backfill_items.pipeline_run_id /
		// workflow_name) so ReportBatchSubtaskFailure and SyncJobProgress
		// keep working while we still have legacy paths in Commit A.
		if argoUID != "" && argoName != "" {
			_ = d.repo.UpdateItemPipelineRun(ctx, item.ID, argoUID, argoName, "running")
		}
		logger.Info("dispatched",
			"item_id", item.ID, "asset_id", item.AssetID, "argo_uid", argoUID, "wf", argoName)

	case errors.Is(err, pipelineUC.ErrWorkflowBeingDeleted):
		// Phase 1 Terminating-during-adopt guard. Park back to pending
		// and let the next claim retry against a clean slot. Short
		// retry lease so we revisit sooner than the full backoff
		// schedule.
		logger.Info("terminating object, reset to pending", "item_id", item.ID)
		_ = d.repo.MarkDispatchFailedRetryable(ctx, item.ID, item.Attempts, "terminating object", 1)

	case isPermanentDeployError(err):
		// Template config wrong / asset not found / etc. No retry.
		logger.Warn("permanent error, marking dead", "item_id", item.ID, "err", err)
		_ = d.repo.MarkDispatchDead(ctx, item.ID, item.Attempts+1, err.Error())

	default:
		// Retryable. Exponential backoff via lease (Decision B).
		attempts := item.Attempts // MarkDispatchFailedRetryable will accept the next attempts number
		backoff := d.backoffForAttempt(attempts)
		logger.Warn("retryable failure",
			"item_id", item.ID, "attempts", attempts,
			"backoff_sec", int(backoff.Seconds()), "err", err)
		_ = d.repo.MarkDispatchFailedRetryable(ctx, item.ID, attempts, err.Error(), int(backoff.Seconds()))
	}
}

// backoffForAttempt returns backoff for the (about-to-be-recorded) next
// attempt index. Jitter is ~20% to avoid retry storms under partial
// recovery of downstream components.
func (d *Dispatcher) backoffForAttempt(attempts int) time.Duration {
	if attempts < 1 {
		attempts = 1
	}
	// 2^(attempts-1)
	exp := math.Pow(2, float64(attempts-1))
	dur := time.Duration(float64(d.cfg.BackoffBase) * exp)
	if dur > d.cfg.BackoffMax {
		dur = d.cfg.BackoffMax
	}
	// Jitter ±10% (deterministic — see rng var above).
	jitter := time.Duration(rng.Int63n(int64(dur) / 5))
	if rng.Intn(2) == 0 {
		dur += jitter
	} else {
		dur -= jitter
	}
	if dur < d.cfg.BackoffBase {
		dur = d.cfg.BackoffBase
	}
	return dur
}

// isPermanentDeployError reports whether err comes from a category that
// should NOT be retried (template missing, asset validation failure,
// target resolution failure). These are typically deploy-side config
// bugs that no amount of retry fixes.
//
// Pipelines lacking these sentinel errors should treat any failure as
// retryable. We match conservatively — when in doubt, retry.
func isPermanentDeployError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, pipelineUC.ErrTemplateNotFound) ||
		errors.Is(err, pipelineUC.ErrInvalidArgument) ||
		errors.Is(err, pipelineUC.ErrWorkflowUnavailable)
}

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
