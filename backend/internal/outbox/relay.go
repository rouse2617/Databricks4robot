package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"sync/atomic"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/metrics"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

// RelayConfig controls outbox relay polling and safety horizon.
type RelayConfig struct {
	BatchSize           int
	Interval            time.Duration
	SafetyLag           time.Duration
	ProcessingLease     time.Duration
	MaxRetries          int
	ProgressLogInterval time.Duration // 0 = disable periodic progress logs
	DLQEveryNBatches    int           // run MoveToDLQ every N flush cycles (0 = disable periodic DLQ)
	// ParallelOrderingKeys limits concurrent goroutines processing different ordering keys in flushOnce.
	// Events sharing a key are always serialized. 1 means legacy fully sequential flush across keys.
	ParallelOrderingKeys int
}

// DefaultRelayConfig returns conservative defaults.
func DefaultRelayConfig() RelayConfig {
	return RelayConfig{
		BatchSize:            200,
		Interval:             500 * time.Millisecond,
		SafetyLag:            2 * time.Second,
		ProcessingLease:      30 * time.Second,
		MaxRetries:           20,
		ProgressLogInterval:  time.Minute,
		DLQEveryNBatches:     20,
		ParallelOrderingKeys: 8,
	}
}

// Relay reads pending asset_events rows and publishes them to the configured bus.
type Relay struct {
	Events                     repository.AssetEventRepository
	DLQ                        repository.OutboxDLQRepository
	Publisher                  EventPublisher
	Config                     RelayConfig
	publishedSinceLastProgress atomic.Int64
}

// Run blocks until ctx is cancelled, polling and flushing pending events.
func (r *Relay) Run(ctx context.Context) error {
	cfg := r.Config
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = DefaultRelayConfig().BatchSize
	}
	if cfg.Interval <= 0 {
		cfg.Interval = DefaultRelayConfig().Interval
	}
	if cfg.SafetyLag < 0 {
		cfg.SafetyLag = 0
	}
	if cfg.ProcessingLease <= 0 {
		cfg.ProcessingLease = DefaultRelayConfig().ProcessingLease
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = DefaultRelayConfig().MaxRetries
	}
	if cfg.ProgressLogInterval < 0 {
		cfg.ProgressLogInterval = 0
	}

	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()
	var progressTicker *time.Ticker
	var progressC <-chan time.Time
	if cfg.ProgressLogInterval > 0 {
		progressTicker = time.NewTicker(cfg.ProgressLogInterval)
		progressC = progressTicker.C
		defer progressTicker.Stop()
	}

	// One immediate flush so startup does not wait a full tick.
	if err := r.flushOnce(ctx, cfg); err != nil && !errors.Is(err, context.Canceled) {
		slog.Warn("outbox relay initial flush", "err", err)
	}

	batches := 0
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := r.flushOnce(ctx, cfg); err != nil && !errors.Is(err, context.Canceled) {
				slog.Warn("outbox relay flush", "err", err)
			}
			batches++
			if r.DLQ != nil && cfg.DLQEveryNBatches > 0 && batches%cfg.DLQEveryNBatches == 0 {
				n, err := r.DLQ.MoveToDLQ(ctx, cfg.MaxRetries)
				if err != nil {
					slog.Warn("outbox relay dlq", "err", err)
				} else if n > 0 {
					slog.Info("outbox relay moved rows to dlq", "count", n)
				}
			}
		case <-progressC:
			r.logProgress(ctx, cfg)
		}
	}
}

func (r *Relay) flushOnce(ctx context.Context, cfg RelayConfig) error {
	started := time.Now()
	defer func() {
		metrics.OutboxRelayFlushDurationSeconds.Observe(time.Since(started).Seconds())
	}()
	events, err := r.listProcessable(ctx, cfg)
	if err != nil {
		return err
	}
	if len(events) == 0 {
		return nil
	}

	parallel := cfg.ParallelOrderingKeys
	if parallel <= 0 {
		parallel = DefaultRelayConfig().ParallelOrderingKeys
	}
	metrics.OutboxRelayParallelKeys.Set(float64(parallel))

	groups := make(map[string][]*models.AssetEvent)
	order := make([]string, 0)
	for _, ev := range events {
		k := orderingKeyFor(ev)
		if _, ok := groups[k]; !ok {
			order = append(order, k)
			groups[k] = nil
		}
		groups[k] = append(groups[k], ev)
	}

	if parallel <= 1 || len(order) <= 1 {
		for _, k := range order {
			for _, ev := range groups[k] {
				if err := r.publishOne(ctx, cfg, ev); err != nil && !errors.Is(err, context.Canceled) {
					return err
				}
			}
		}
		return nil
	}

	if parallel > len(order) {
		parallel = len(order)
	}
	metrics.OutboxRelayParallelKeys.Set(float64(parallel))
	sem := semaphore.NewWeighted(int64(parallel))
	g, gctx := errgroup.WithContext(ctx)
	for _, k := range order {
		k := k
		evs := groups[k]
		g.Go(func() error {
			if err := sem.Acquire(gctx, 1); err != nil {
				return err
			}
			defer sem.Release(1)
			for _, ev := range evs {
				if err := r.publishOne(gctx, cfg, ev); err != nil && !errors.Is(err, context.Canceled) {
					return err
				}
			}
			return nil
		})
	}
	return g.Wait()
}

type pendingClaimer interface {
	ClaimPendingSafe(ctx context.Context, safetyLag, processingLease time.Duration, limit int) ([]*models.AssetEvent, error)
}

func (r *Relay) listProcessable(ctx context.Context, cfg RelayConfig) ([]*models.AssetEvent, error) {
	if claimer, ok := r.Events.(pendingClaimer); ok {
		return claimer.ClaimPendingSafe(ctx, cfg.SafetyLag, cfg.ProcessingLease, cfg.BatchSize)
	}
	return r.Events.ListPendingSafe(ctx, cfg.SafetyLag, cfg.BatchSize)
}

func (r *Relay) publishOne(ctx context.Context, cfg RelayConfig, ev *models.AssetEvent) error {
	if ev == nil {
		return nil
	}
	if r.Publisher == nil {
		return r.Events.MarkFailed(ctx, ev.EventSeq, "outbox publisher is not configured")
	}
	if int(ev.RetryCount) > cfg.MaxRetries {
		return nil
	}
	data, err := json.Marshal(ev)
	if err != nil {
		return r.Events.MarkFailed(ctx, ev.EventSeq, err.Error())
	}
	orderingKey := orderingKeyFor(ev)
	pr, err := r.Publisher.Publish(ctx, orderingKey, data)
	if err != nil {
		r.Publisher.ResumePublishAfterError(orderingKey)
		return r.Events.MarkFailed(ctx, ev.EventSeq, err.Error())
	}
	if _, err := pr.Get(ctx); err != nil {
		r.Publisher.ResumePublishAfterError(orderingKey)
		return r.Events.MarkFailed(ctx, ev.EventSeq, err.Error())
	}
	if err := r.Events.MarkPublished(ctx, []int64{ev.EventSeq}); err != nil {
		return err
	}
	r.publishedSinceLastProgress.Add(1)
	metrics.OutboxRelayPublishedTotal.WithLabelValues(orderingKeyKind(ev)).Inc()
	if !ev.CreatedAt.IsZero() {
		metrics.OutboxEventLagSeconds.Observe(time.Since(ev.CreatedAt).Seconds())
	}
	return nil
}

func orderingKeyKind(ev *models.AssetEvent) string {
	if ev == nil {
		return "na"
	}
	if ev.AssetID != "" {
		return "asset"
	}
	if ev.McapFileID != "" {
		return "mcap"
	}
	return "na"
}

func orderingKeyFor(ev *models.AssetEvent) string {
	if ev.AssetID != "" {
		return ev.AssetID
	}
	if ev.McapFileID != "" {
		return "mcap:" + ev.McapFileID
	}
	return "_na"
}

func (r *Relay) logProgress(ctx context.Context, cfg RelayConfig) {
	if r == nil || r.Events == nil {
		return
	}
	published := r.publishedSinceLastProgress.Swap(0)

	pending, err := r.Events.CountPending(ctx)
	if err != nil {
		slog.Warn("outbox relay progress count pending", "err", err)
		return
	}
	oldestSec, err := r.Events.OldestPendingAge(ctx)
	if err != nil {
		slog.Warn("outbox relay progress oldest pending age", "err", err)
		oldestSec = 0
	}

	ratePerMin := 0.0
	if cfg.ProgressLogInterval > 0 {
		ratePerMin = float64(published) * time.Minute.Seconds() / cfg.ProgressLogInterval.Seconds()
	}

	etaMin := 0.0
	if pending > 0 && ratePerMin > 0 {
		etaMin = float64(pending) / ratePerMin
	}

	parallelKeys := cfg.ParallelOrderingKeys
	if parallelKeys <= 0 {
		parallelKeys = DefaultRelayConfig().ParallelOrderingKeys
	}

	slog.Info(
		"outbox relay progress",
		"window_sec", int(cfg.ProgressLogInterval.Seconds()),
		"published", published,
		"rate_per_min", round1(ratePerMin),
		"pending", pending,
		"oldest_pending_sec", round1(oldestSec),
		"eta_min", round1(etaMin),
		"parallel_ordering_keys", parallelKeys,
	)
}

func round1(v float64) float64 {
	return math.Round(v*10) / 10
}
