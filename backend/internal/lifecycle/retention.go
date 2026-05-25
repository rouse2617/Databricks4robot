package lifecycle

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/filter"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

// RetentionConfig controls the retention job behavior.
type RetentionConfig struct {
	// Interval is how often the job scans for expired assets.
	Interval time.Duration
	// BatchSize is the maximum number of assets archived per scan cycle.
	BatchSize int
	// DryRun logs what would be archived without making changes.
	DryRun bool
}

// DefaultRetentionConfig returns conservative defaults.
func DefaultRetentionConfig() RetentionConfig {
	return RetentionConfig{
		Interval:  10 * time.Minute,
		BatchSize: 200,
		DryRun:    false,
	}
}

// RetentionJob periodically checks assets.expire_at and transitions expired
// assets to lifecycle_state='archived'. It emits lifecycle events so the
// outbox pipeline can propagate the change to ES and other consumers.
//
// P1 scope: the job archives expired assets. Actual deletion (GCS cleanup,
// row removal) is a manual P2 follow-up.
type RetentionJob struct {
	Assets   repository.AssetRepository
	Events   repository.AssetEventRepository
	TxRunner repository.TxRunner
	Config   RetentionConfig
}

// Run blocks until ctx is cancelled, scanning for expired assets on each tick.
func (j *RetentionJob) Run(ctx context.Context) error {
	cfg := j.Config
	if cfg.Interval <= 0 {
		cfg.Interval = DefaultRetentionConfig().Interval
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = DefaultRetentionConfig().BatchSize
	}

	// One immediate scan on startup.
	j.scanOnce(ctx, cfg)

	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			j.scanOnce(ctx, cfg)
		}
	}
}

// scanOnce finds expired assets and archives them.
func (j *RetentionJob) scanOnce(ctx context.Context, cfg RetentionConfig) {
	// Find assets where expire_at < now() and lifecycle_state is not already
	// terminal (archived, superseded, failed, rejected).
	const whereSQL = `expire_at IS NOT NULL
		AND expire_at <= now()
		AND lifecycle_state NOT IN ('archived', 'superseded', 'failed', 'rejected')`

	assets, _, err := j.Assets.ListWithFilters(ctx, whereSQL, nil, 1, cfg.BatchSize,
		filter.OrderByClause{SQL: "expire_at ASC"})
	if err != nil {
		slog.Warn("retention job: list expired assets failed", "err", err)
		return
	}

	if len(assets) == 0 {
		return
	}

	slog.Info("retention job: found expired assets", "count", len(assets), "dry_run", cfg.DryRun)

	for _, a := range assets {
		if a == nil || a.AssetID == "" {
			continue
		}
		if cfg.DryRun {
			slog.Info("retention job: would archive (dry_run)",
				"asset_id", a.AssetID,
				"expire_at", a.ExpireAt,
				"lifecycle_state", a.LifecycleState)
			continue
		}
		if err := j.archiveOne(ctx, a); err != nil {
			slog.Warn("retention job: archive failed",
				"asset_id", a.AssetID, "err", err)
			continue
		}
		slog.Info("retention job: archived expired asset",
			"asset_id", a.AssetID,
			"expire_at", a.ExpireAt,
			"prev_state", a.LifecycleState)
	}
}

// archiveOne transitions a single asset to lifecycle_state='archived' and
// emits a lifecycle event in the same transaction.
func (j *RetentionJob) archiveOne(ctx context.Context, a *models.Asset) error {
	prevState := a.LifecycleState
	a.LifecycleState = "archived"
	a.Status = models.AssetStatusArchived
	a.UpdatedAt = time.Now().UTC()

	if j.TxRunner != nil {
		return j.TxRunner.WithTx(ctx, func(txCtx context.Context) error {
			if err := j.Assets.Set(txCtx, a); err != nil {
				return err
			}
			return j.emitRetentionEvent(txCtx, a.AssetID, prevState)
		})
	}

	if err := j.Assets.Set(ctx, a); err != nil {
		return err
	}
	return j.emitRetentionEvent(ctx, a.AssetID, prevState)
}

// emitRetentionEvent appends a lifecycle event for the retention archive.
func (j *RetentionJob) emitRetentionEvent(ctx context.Context, assetID, prevState string) error {
	if j.Events == nil {
		return nil
	}
	payload, _ := json.Marshal(map[string]any{
		"reason":     "retention_expired",
		"prev_state": prevState,
		"new_state":  "archived",
	})
	return j.Events.Append(ctx, repository.AssetEventAppendInput{
		EventType:    "lifecycle_archived",
		AssetID:      assetID,
		EventPayload: payload,
	})
}
