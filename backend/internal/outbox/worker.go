// Package outbox implements the in-process ES sink worker for asset_events.
package outbox

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"
	"time"

	"data-platform/internal/elasticsearch"
	"data-platform/internal/metrics"
	"data-platform/internal/repository"
	"data-platform/internal/searchindex"
)

// HealthMarker allows the worker to signal health status changes.
// The concrete implementation lives in internal/outbox/health.go (task 2.9).
type HealthMarker interface {
	MarkUnhealthy(reason string)
}

// ESWorker polls pending asset_events and projects affected assets into Elasticsearch.
type ESWorker struct {
	Events       repository.AssetEventRepository
	Indexer      *searchindex.Builder
	ES           *elasticsearch.Client
	DLQ          repository.OutboxDLQRepository // optional; moves permanently failed events to DLQ
	BatchSize    int
	SinkName     string       // outbox_sink_cursors identifier; defaults to "es_assets"
	FatalOnPanic bool         // when true, os.Exit(1) after panic recovery (§5.1)
	RetryLimit   int          // retry_count threshold for warning logs (§10); 0 disables
	Health       HealthMarker // optional; called on panic to degrade health
	// osExit is an indirection for testing; defaults to os.Exit.
	osExit func(code int)
}

// maxDrainIterations caps the number of processBatch calls within a single
// tick to prevent infinite loops (e.g. if events are produced faster than
// consumed). When the limit is reached the worker yields to the next tick.
const maxDrainIterations = 100

// fetchBackoffSteps defines the exponential backoff durations for consecutive
// PG fetch failures (§9). After each failure the worker waits the next step
// before retrying; the last value (10s) is the cap.
var fetchBackoffSteps = []time.Duration{
	1 * time.Second,
	2 * time.Second,
	5 * time.Second,
	10 * time.Second,
}

// backoffDuration returns the backoff duration for the given consecutive
// failure count (0-indexed). Returns 0 for failCount <= 0.
func backoffDuration(failCount int) time.Duration {
	if failCount <= 0 {
		return 0
	}
	idx := failCount - 1
	if idx >= len(fetchBackoffSteps) {
		idx = len(fetchBackoffSteps) - 1
	}
	return fetchBackoffSteps[idx]
}

// sleepCtx sleeps for d or until ctx is cancelled, whichever comes first.
// Returns ctx.Err() if the context was cancelled.
func sleepCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// sinkName returns the configured sink identifier, defaulting to "es_assets".
func (w *ESWorker) sinkName() string {
	if w.SinkName != "" {
		return w.SinkName
	}
	return "es_assets"
}

// processBatch processes one batch of pending events. It returns done=true
// when there are no more pending events (batch was empty or smaller than
// BatchSize), signalling the drain loop to stop.
func (w *ESWorker) processBatch(ctx context.Context) (done bool, err error) {
	if w.BatchSize <= 0 {
		w.BatchSize = 100
	}
	pending, err := w.Events.ListPending(ctx, w.BatchSize)
	if err != nil {
		return false, err
	}
	if len(pending) == 0 {
		return true, nil
	}

	seqsByAsset := make(map[string][]int64)
	var skipSeqs []int64
	for _, e := range pending {
		// Log a warning for events that have exceeded the retry limit (§10).
		if w.RetryLimit > 0 && e.RetryCount >= w.RetryLimit {
			slog.Warn("outbox event exceeded retry limit",
				"event_seq", e.EventSeq,
				"asset_id", e.AssetID,
				"retry_count", e.RetryCount,
				"retry_limit", w.RetryLimit,
			)
		}
		if e.AggregateType != "" && e.AggregateType != "asset" {
			skipSeqs = append(skipSeqs, e.EventSeq)
			continue
		}
		if e.AssetID == "" {
			skipSeqs = append(skipSeqs, e.EventSeq)
			continue
		}
		seqsByAsset[e.AssetID] = append(seqsByAsset[e.AssetID], e.EventSeq)
	}

	if len(skipSeqs) > 0 {
		if err := w.Events.MarkPublishedAndAdvanceCursor(ctx, skipSeqs, w.sinkName()); err != nil {
			return false, fmt.Errorf("mark non-asset skipped events: %w", err)
		}
		metrics.OutboxEventsPublished.Add(float64(len(skipSeqs)))
	}

	var delSeqs []int64
	var docs []elasticsearch.BulkIndexDoc
	indexedAssetIDs := make([]string, 0, len(seqsByAsset))

	for assetID, seqs := range seqsByAsset {
		doc, ok, err := w.Indexer.Build(ctx, assetID)
		if err != nil {
			for _, seq := range seqs {
				_ = w.Events.MarkFailed(ctx, seq, err.Error())
			}
			metrics.OutboxEventsFailedMark.Add(float64(len(seqs)))
			continue
		}
		if !ok {
			if err := w.ES.DeleteDocument(ctx, assetID); err != nil {
				for _, seq := range seqs {
					_ = w.Events.MarkFailed(ctx, seq, err.Error())
				}
				metrics.OutboxEventsFailedMark.Add(float64(len(seqs)))
				continue
			}
			delSeqs = append(delSeqs, seqs...)
			continue
		}
		docs = append(docs, elasticsearch.BulkIndexDoc{ID: assetID, Doc: doc})
		indexedAssetIDs = append(indexedAssetIDs, assetID)
	}

	var markSeqs []int64
	markSeqs = append(markSeqs, delSeqs...)

	if len(docs) > 0 {
		bulkResult, err := w.ES.BulkIndex(ctx, docs)
		if err != nil {
			// Transport-level / HTTP-level failure: mark all indexed events as failed.
			metrics.OutboxESBulkFailures.Inc()
			var fail int64
			for _, id := range indexedAssetIDs {
				fail += int64(len(seqsByAsset[id]))
				for _, seq := range seqsByAsset[id] {
					_ = w.Events.MarkFailed(ctx, seq, err.Error())
				}
			}
			metrics.OutboxEventsFailedMark.Add(float64(fail))
			return false, err
		}

		// Build set of failed doc IDs for partial-failure splitting.
		failedDocIDs := make(map[string]string) // doc ID → error reason
		for _, item := range bulkResult.Failed {
			reason := item.Error
			if reason == "" {
				reason = fmt.Sprintf("ES bulk item status %d", item.Status)
			}
			failedDocIDs[item.ID] = reason
		}

		if len(failedDocIDs) > 0 {
			metrics.OutboxBulkPartialFailure.Inc()
			slog.Warn("es bulk partial failure",
				"failed_count", len(failedDocIDs),
				"total", len(docs),
			)
		}

		metrics.OutboxBulkDocs.Add(float64(len(bulkResult.Succeeded)))

		for _, id := range indexedAssetIDs {
			if reason, failed := failedDocIDs[id]; failed {
				// This doc failed in ES — mark its events as failed.
				for _, seq := range seqsByAsset[id] {
					_ = w.Events.MarkFailed(ctx, seq, reason)
				}
				metrics.OutboxEventsFailedMark.Add(float64(len(seqsByAsset[id])))
			} else {
				// This doc succeeded — collect seqs for cursor advance.
				markSeqs = append(markSeqs, seqsByAsset[id]...)
			}
		}
	}

	if len(markSeqs) > 0 {
		if err := w.Events.MarkPublishedAndAdvanceCursor(ctx, markSeqs, w.sinkName()); err != nil {
			return false, err
		}
		metrics.OutboxEventsPublished.Add(float64(len(markSeqs)))
	}
	// batch full → possibly more pending; batch smaller than limit → pending drained
	return len(pending) < w.BatchSize, nil
}

// Run starts a polling loop until ctx is cancelled.
// Each tick triggers a drain loop that repeatedly calls processBatch until
// pending events are exhausted or maxDrainIterations is reached (§5.1).
//
// A top-level defer/recover prevents a goroutine panic from silently killing
// the worker while /healthz still reports healthy. On panic the worker logs
// the stack, marks itself unhealthy via Health.MarkUnhealthy, and optionally
// calls os.Exit(1) (controlled by FatalOnPanic) so that K8s liveness can
// restart the container.
func (w *ESWorker) Run(ctx context.Context, tick time.Duration) {
	// Panic recovery (§5.1): surface the panic instead of letting the
	// goroutine die silently.
	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()
			slog.Error("outbox worker panic",
				"recover", fmt.Sprint(r),
				"stack", string(stack),
			)
			if w.Health != nil {
				w.Health.MarkUnhealthy("panic: " + fmt.Sprint(r))
			}
			if w.FatalOnPanic {
				exitFn := w.osExit
				if exitFn == nil {
					exitFn = os.Exit
				}
				exitFn(1)
			}
		}
	}()

	if tick <= 0 {
		tick = 30 * time.Second
	}
	t := time.NewTicker(tick)
	defer t.Stop()

	var consecutiveFetchFails int // tracks consecutive PG fetch failures for backoff (§9)

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			n, err := w.Events.CountPending(ctx)
			if err == nil {
				metrics.OutboxPendingGauge.Set(float64(n))
			}

			// Sample the age of the oldest pending event (§11.1).
			if age, err := w.Events.OldestPendingAge(ctx); err == nil {
				metrics.OutboxOldestPendingAge.Set(age)
			}

			// Sweep permanently failed events to DLQ before processing.
			if w.DLQ != nil && w.RetryLimit > 0 {
				if moved, err := w.DLQ.MoveToDLQ(ctx, w.RetryLimit); err != nil {
					slog.Warn("dlq sweep failed", "err", err)
				} else if moved > 0 {
					slog.Info("moved events to DLQ", "count", moved)
				}
			}

			// Drain loop: repeatedly process batches within the same tick
			// until pending is empty or the safety limit is hit.
			for i := range maxDrainIterations {
				batchStart := time.Now()
				done, err := w.processBatch(ctx)
				metrics.OutboxBatchDurationMs.Observe(float64(time.Since(batchStart).Milliseconds()))
				if err != nil {
					consecutiveFetchFails++
					wait := backoffDuration(consecutiveFetchFails)
					slog.Error("outbox worker batch failed",
						"err", err,
						"drain_iteration", i,
						"consecutive_failures", consecutiveFetchFails,
						"backoff", wait,
					)
					if err := sleepCtx(ctx, wait); err != nil {
						return // context cancelled during backoff
					}
					break // exit drain, wait for next tick
				}
				consecutiveFetchFails = 0 // reset on success
				metrics.OutboxWorkerBatches.Inc()
				if done {
					break // pending exhausted
				}
			}
		}
	}
}
