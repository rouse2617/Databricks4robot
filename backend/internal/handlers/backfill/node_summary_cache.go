package backfill

import (
	"sync"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// batchNodeSummaryCache caches the /backfill/{id}/node-summary response per
// jobID for a short window so a front-end poll does not translate into a hot
// DB query.
type batchNodeSummaryCache struct {
	mu      sync.Mutex
	entries map[string]batchNodeSummaryEntry
}

type batchNodeSummaryEntry struct {
	summary   *models.BatchNodeSummary
	expiresAt time.Time
}

// batchNodeSummaryCacheTTL bounds staleness: short enough that a user opening
// the page repeatedly still sees fresh status, long enough to absorb the
// front-end's poll without hitting the DB. The companion
// refreshBatchReadModel throttle (see usecase.shouldSyncProgress) runs every
// syncProgressMinInterval (10s) and overwrites the cache entry on its next
// miss, keeping the cached summary within roughly one sync window of fresh.
const batchNodeSummaryCacheTTL = 5 * time.Second

var globalBatchNodeSummaryCache = &batchNodeSummaryCache{
	entries: make(map[string]batchNodeSummaryEntry),
}

func (c *batchNodeSummaryCache) get(jobID string) (*models.BatchNodeSummary, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[jobID]
	if !ok || time.Now().After(e.expiresAt) {
		return nil, false
	}
	return e.summary, true
}

func (c *batchNodeSummaryCache) set(jobID string, summary *models.BatchNodeSummary) {
	if summary == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[jobID] = batchNodeSummaryEntry{
		summary:   summary,
		expiresAt: time.Now().Add(batchNodeSummaryCacheTTL),
	}
}
