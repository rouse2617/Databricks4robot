// CYB-3384: SyncHealthCache exposes the PG↔ES gap to the planner so it can
// route facet aggregations to PG when the two engines are out of sync, and to
// ES otherwise. The cache refreshes in the background at a coarse cadence
// (default 30s) so per-request planning stays lock-free and cheap.
package queryplan

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

// GapFetcher returns the current PG↔ES gap in # of assets. Implementations
// typically wrap GET /api/v1/search/sync-progress or an equivalent internal
// call. Errors are surfaced so the cache can log and keep the previous value.
type GapFetcher func(ctx context.Context) (int64, error)

// SyncHealthCache holds a monotonically-updated snapshot of the PG↔ES sync
// gap and refresh timestamp. Reads are lock-free via atomic; writes hold the
// mutex briefly. A cache with a nil fetcher always reports gap=0 (used in
// tests and any deployment that doesn't wire /sync-progress).
type SyncHealthCache struct {
	fetcher GapFetcher
	ttl     time.Duration

	gap       atomic.Int64
	updatedAt atomic.Int64 // unix nano
	mu        sync.Mutex   // serializes concurrent Refresh calls
}

// NewSyncHealthCache constructs a cache. ttl controls the background refresh
// cadence; when 0 or negative it defaults to 30s. fetcher may be nil for
// disabled-cache mode (Gap always returns 0).
func NewSyncHealthCache(fetcher GapFetcher, ttl time.Duration) *SyncHealthCache {
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	return &SyncHealthCache{fetcher: fetcher, ttl: ttl}
}

// Gap returns the last observed PG↔ES asset gap. Zero when the cache has
// never refreshed or has no fetcher wired.
func (c *SyncHealthCache) Gap() int64 {
	if c == nil {
		return 0
	}
	return c.gap.Load()
}

// UpdatedAt returns the wall-clock time of the last successful refresh, or
// the zero time when no refresh has succeeded.
func (c *SyncHealthCache) UpdatedAt() time.Time {
	if c == nil {
		return time.Time{}
	}
	nanos := c.updatedAt.Load()
	if nanos == 0 {
		return time.Time{}
	}
	return time.Unix(0, nanos).UTC()
}

// Refresh pulls a fresh snapshot from the fetcher and stores it. Safe to call
// concurrently; a mutex serializes overlapping refreshes so only one HTTP
// call is in flight at a time. Returns the error surfaced by the fetcher; on
// error the previous value is preserved.
func (c *SyncHealthCache) Refresh(ctx context.Context) error {
	if c == nil || c.fetcher == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	gap, err := c.fetcher(ctx)
	if err != nil {
		return err
	}
	c.gap.Store(gap)
	c.updatedAt.Store(time.Now().UnixNano())
	return nil
}

// Run performs an initial synchronous refresh, then continues to refresh on
// every ttl tick until ctx is cancelled. It's designed to be launched in a
// goroutine at server startup; Run returns nil once ctx is done. The initial
// refresh error (if any) is logged but not returned so a broken /sync-progress
// endpoint never crashes startup.
func (c *SyncHealthCache) Run(ctx context.Context) error {
	if c == nil || c.fetcher == nil {
		return nil
	}
	if err := c.Refresh(ctx); err != nil {
		slog.Warn("sync health cache: initial refresh failed", "err", err)
	}
	ticker := time.NewTicker(c.ttl)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := c.Refresh(ctx); err != nil {
				slog.Warn("sync health cache: refresh failed", "err", err)
			}
		}
	}
}
