package server

import (
	"context"
	"io"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/CyberOrigin2077/cyber-databrew/services/mcap-preview/internal/gcsrs"
)

// mcapReaderCache holds per-MCAP page-cached readers across requests so we
// stop re-doing the GCS Attrs + summary Range reads (~1.5–2 s for a 5+ GB
// MCAP) on every preview call.
//
// Cached entry is a "template" *gcsrs.GCSReadSeeker; each Acquire returns a
// fresh Clone() with the caller's context and an independent seek cursor.
// Page bytes and IO counters are shared. Entries expire after TTL; if the
// cache is full on insert, the entry with the soonest expiry is evicted.
//
// Memory bound: each cached template owns a `gcsrs.DefaultCacheBytes` (64
// MiB) page LRU. With maxEntries = defaultMCAPReaderCacheMax (16) the cache
// is bounded at ~1 GiB worst case — well inside an 8 GiB instance even
// alongside ffmpeg's working set. Bump the cap if you have headroom; do NOT
// leave it unbounded, since a sweep across N distinct MCAPs (dagster crawl,
// search paging) would otherwise pin N × 64 MiB until process exit.
//
// When the configured opener returns something other than a *gcsrs.GCSReadSeeker
// (e.g. local-file fixtures used by tests), Acquire transparently bypasses the
// cache and calls the opener each time — preserves identical semantics for
// those callers.
type mcapReaderCache struct {
	mu         sync.Mutex
	items      map[string]*mcapCacheEntry
	sf         singleflight.Group
	ttl        time.Duration
	maxEntries int
	opener     Config_OpenMCAP
}

type mcapCacheEntry struct {
	template  *gcsrs.GCSReadSeeker
	expiresAt time.Time
}

// defaultMCAPReaderCacheMax caps the number of cached MCAP readers. See the
// memory-bound note on mcapReaderCache.
const defaultMCAPReaderCacheMax = 16

func newMCAPReaderCache(ttl time.Duration, opener Config_OpenMCAP) *mcapReaderCache {
	return newMCAPReaderCacheWithMax(ttl, defaultMCAPReaderCacheMax, opener)
}

func newMCAPReaderCacheWithMax(ttl time.Duration, max int, opener Config_OpenMCAP) *mcapReaderCache {
	if max < 1 {
		max = 1
	}
	return &mcapReaderCache{
		items:      map[string]*mcapCacheEntry{},
		ttl:        ttl,
		maxEntries: max,
		opener:     opener,
	}
}

// Acquire returns a ReadSeeker over the named MCAP. The signature mirrors
// Config_OpenMCAP so call sites can swap one for the other. The release
// closure may be nil (cached path) or invoke the opener's closer (bypass
// path).
//
// Cached path: subsequent calls within TTL return Clone()s that share the
// page cache; only one GCS Attrs + summary read in total.
//
// Bypass path: when the opener returns something other than *gcsrs.GCSReadSeeker
// (e.g. *bytes.Reader in tests), we cannot safely share it. Acquire returns
// the opener's result verbatim every time; no caching, no Clone.
func (c *mcapReaderCache) Acquire(ctx context.Context, mcapURI string) (io.ReadSeeker, *gcsrs.Stats, func(), error) {
	if c == nil || c.opener == nil {
		return nil, nil, nil, errNilOpener
	}
	if entry := c.get(mcapURI); entry != nil {
		rs := entry.template.Clone(ctx)
		stats := rs.Stats()
		return rs, &stats, nil, nil
	}
	v, err, _ := c.sf.Do(mcapURI, func() (any, error) {
		if entry := c.get(mcapURI); entry != nil {
			return entry, nil
		}
		rs, stats, closer, err := c.opener(ctx, mcapURI)
		if err != nil {
			return nil, err
		}
		gcs, ok := rs.(*gcsrs.GCSReadSeeker)
		if !ok {
			// Non-cacheable reader (e.g. local file in tests). Pass it
			// through directly. Concurrent bypass acquires for the same
			// URI collapse via singleflight and would share this reader
			// (BUG potential for parallel local-file tests); production
			// always returns *gcsrs.GCSReadSeeker, so we accept the
			// tradeoff rather than disable singleflight here.
			return &bypassOpen{rs: rs, stats: stats, closer: closer}, nil
		}
		// Template owns the underlying reader; opener's closer (if any)
		// is intentionally dropped — the page cache lives in `gcs` and is
		// reclaimed when the cache entry is evicted.
		_ = closer
		entry := &mcapCacheEntry{
			template:  gcs,
			expiresAt: time.Now().Add(c.ttl),
		}
		c.put(mcapURI, entry)
		return entry, nil
	})
	if err != nil {
		return nil, nil, nil, err
	}
	switch r := v.(type) {
	case *mcapCacheEntry:
		rs := r.template.Clone(ctx)
		stats := rs.Stats()
		return rs, &stats, nil, nil
	case *bypassOpen:
		return r.rs, r.stats, r.closer, nil
	default:
		return nil, nil, nil, errUnexpectedCacheValue
	}
}

// Invalidate drops the cached entry for mcapURI. Useful when the underlying
// MCAP is known to have changed; otherwise rely on TTL.
func (c *mcapReaderCache) Invalidate(mcapURI string) {
	c.mu.Lock()
	delete(c.items, mcapURI)
	c.mu.Unlock()
}

func (c *mcapReaderCache) get(mcapURI string) *mcapCacheEntry {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.items[mcapURI]
	if !ok {
		return nil
	}
	if time.Now().After(e.expiresAt) {
		delete(c.items, mcapURI)
		return nil
	}
	// Bump expiry on access; long-lived hot MCAPs stay warm.
	e.expiresAt = time.Now().Add(c.ttl)
	return e
}

func (c *mcapReaderCache) put(mcapURI string, entry *mcapCacheEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// Drop already-expired entries first; only fall back to "evict soonest"
	// when nothing's expired but we're still over capacity.
	if len(c.items) >= c.maxEntries {
		now := time.Now()
		for k, e := range c.items {
			if now.After(e.expiresAt) {
				delete(c.items, k)
			}
		}
	}
	if len(c.items) >= c.maxEntries {
		var (
			oldestKey string
			oldestAt  time.Time
		)
		first := true
		for k, e := range c.items {
			if first || e.expiresAt.Before(oldestAt) {
				oldestKey = k
				oldestAt = e.expiresAt
				first = false
			}
		}
		if oldestKey != "" {
			delete(c.items, oldestKey)
		}
	}
	c.items[mcapURI] = entry
}

// bypassOpen carries an opener result through the singleflight when the
// underlying reader can't participate in the page cache. The fields mirror
// Config_OpenMCAP's return values.
type bypassOpen struct {
	rs     io.ReadSeeker
	stats  *gcsrs.Stats
	closer func()
}

var (
	// errNilOpener is returned when Acquire is called on a zero/unwired cache.
	// Surfaces as a configuration bug rather than a runtime upstream error.
	errNilOpener           = &cacheError{msg: "mcap reader cache: opener not configured"}
	errUnexpectedCacheValue = &cacheError{msg: "mcap reader cache: unexpected internal value"}
)

type cacheError struct{ msg string }

func (e *cacheError) Error() string { return e.msg }
