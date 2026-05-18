// Package gcsrs provides a page-cached io.ReadSeeker over a GCS object.
//
// The MCAP reader does many small Read+Seek calls when parsing the
// summary/index trailer; serving each one as its own GCS Range request would
// be both expensive and slow. Instead we batch reads through fixed-size
// pages held in an LRU cache, mirroring the pattern proven out in
// backend/cmd/mcap-range-demo. Stats are exposed for inclusion in the
// manifest response so callers can sanity-check IO cost.
//
// Copied (intentionally not imported) from backend/cmd/mcap-range-demo so the
// services/mcap-preview module stays an independent build target.
package gcsrs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"

	"cloud.google.com/go/storage"
	"github.com/golang/groupcache/lru"
)

// Default knobs; overridable via Options. Page size of 1 MiB and 64 cached
// pages tracks the demo defaults and keeps total in-memory footprint bounded
// at ~64 MiB per concurrent reader.
const (
	DefaultPageSize    int64 = 1 << 20
	DefaultCacheBytes  int64 = 64 * (1 << 20)
	defaultCacheItems        = 64
)

// Options tunes the page-cached reader.
type Options struct {
	// PageSize is the GCS Range read size in bytes. Must be > 0.
	PageSize int64
	// CacheBytes bounds total cache footprint. The number of cached pages
	// is roughly CacheBytes / PageSize.
	CacheBytes int64
}

// Stats is a cheap snapshot of IO cost for inclusion in API responses.
type Stats struct {
	RangeCalls int64 `json:"gcs_range_requests"`
	BytesRead  int64 `json:"gcs_bytes_read"`
}

// sharedPages holds the page cache and IO counters shared between a base
// reader and any clones derived from it. The LRU is not concurrency-safe, so
// we serialize cache access with `mu`; counters are atomics so Stats() can be
// read without acquiring the mutex.
type sharedPages struct {
	mu          sync.Mutex
	cache       *lru.Cache
	pageSize    int64
	rangeCalls  int64
	cacheHits   int64
	cacheMisses int64
	bytesFetch  int64
}

// GCSReadSeeker implements io.ReadSeeker over GCS object bytes via Range reads,
// with an LRU page cache to coalesce small reads. Clone() returns a reader
// view that shares the underlying page cache; this lets the mcap-preview
// service hold a "template" reader hot across requests while each request
// gets its own seek cursor.
type GCSReadSeeker struct {
	ctx    context.Context
	obj    *storage.ObjectHandle
	size   int64
	pos    int64
	shared *sharedPages
}

// New constructs a page-cached ReadSeeker for obj. opts is optional.
func New(ctx context.Context, obj *storage.ObjectHandle, size int64, opts *Options) *GCSReadSeeker {
	pageSize := DefaultPageSize
	cacheBytes := DefaultCacheBytes
	if opts != nil {
		if opts.PageSize > 0 {
			pageSize = opts.PageSize
		}
		if opts.CacheBytes > 0 {
			cacheBytes = opts.CacheBytes
		}
	}
	maxItems := int(cacheBytes / pageSize)
	if maxItems < 1 {
		maxItems = defaultCacheItems
	}
	return &GCSReadSeeker{
		ctx:  ctx,
		obj:  obj,
		size: size,
		shared: &sharedPages{
			cache:    lru.New(maxItems),
			pageSize: pageSize,
		},
	}
}

// Clone returns a new reader that shares this reader's page cache and IO
// counters but has its own seek cursor and per-call context. Use this from a
// reader cache where the template reader is long-lived but each request needs
// an independent cursor and (importantly) a context tied to its own deadline.
//
// Concurrent reads through clones are safe; the underlying LRU access is
// serialized internally.
func (g *GCSReadSeeker) Clone(ctx context.Context) *GCSReadSeeker {
	return &GCSReadSeeker{
		ctx:    ctx,
		obj:    g.obj,
		size:   g.size,
		pos:    0,
		shared: g.shared,
	}
}

// Size returns the object size in bytes (cached at New time).
func (g *GCSReadSeeker) Size() int64 { return g.size }

func (g *GCSReadSeeker) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if g.pos >= g.size {
		return 0, io.EOF
	}
	remain := g.size - g.pos
	toRead := int64(len(p))
	if toRead > remain {
		toRead = remain
	}

	var copied int64
	for copied < toRead {
		pageIdx := g.pos / g.shared.pageSize
		pageOff := g.pos % g.shared.pageSize
		page, err := g.getPage(pageIdx)
		if err != nil {
			return int(copied), err
		}
		available := int64(len(page)) - pageOff
		need := toRead - copied
		if need > available {
			need = available
		}
		copy(p[copied:copied+need], page[pageOff:pageOff+need])
		copied += need
		g.pos += need
	}
	return int(copied), nil
}

func (g *GCSReadSeeker) Seek(offset int64, whence int) (int64, error) {
	var newPos int64
	switch whence {
	case io.SeekStart:
		newPos = offset
	case io.SeekCurrent:
		newPos = g.pos + offset
	case io.SeekEnd:
		newPos = g.size + offset
	default:
		return 0, errors.New("invalid whence")
	}
	if newPos < 0 {
		return 0, errors.New("negative seek position")
	}
	if newPos > g.size {
		newPos = g.size
	}
	g.pos = newPos
	return g.pos, nil
}

// Stats returns a snapshot of the IO counters. Reflects work done across all
// clones that share this reader's page cache.
func (g *GCSReadSeeker) Stats() Stats {
	return Stats{
		RangeCalls: atomic.LoadInt64(&g.shared.rangeCalls),
		BytesRead:  atomic.LoadInt64(&g.shared.bytesFetch),
	}
}

// StatsSummary returns a human-readable line; useful for one-off debug logs.
func (g *GCSReadSeeker) StatsSummary() string {
	return fmt.Sprintf(
		"page_size=%d range_calls=%d cache_hits=%d cache_misses=%d bytes_fetched=%d",
		g.shared.pageSize,
		atomic.LoadInt64(&g.shared.rangeCalls),
		atomic.LoadInt64(&g.shared.cacheHits),
		atomic.LoadInt64(&g.shared.cacheMisses),
		atomic.LoadInt64(&g.shared.bytesFetch),
	)
}

func (g *GCSReadSeeker) getPage(pageIdx int64) ([]byte, error) {
	g.shared.mu.Lock()
	if got, ok := g.shared.cache.Get(pageIdx); ok {
		page := got.([]byte)
		g.shared.mu.Unlock()
		atomic.AddInt64(&g.shared.cacheHits, 1)
		return page, nil
	}
	g.shared.mu.Unlock()
	atomic.AddInt64(&g.shared.cacheMisses, 1)

	start := pageIdx * g.shared.pageSize
	if start >= g.size {
		return nil, io.EOF
	}
	length := g.shared.pageSize
	if max := g.size - start; length > max {
		length = max
	}
	r, err := g.obj.NewRangeReader(g.ctx, start, length)
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()

	buf := make([]byte, length)
	n, err := io.ReadFull(r, buf)
	atomic.AddInt64(&g.shared.rangeCalls, 1)
	atomic.AddInt64(&g.shared.bytesFetch, int64(n))
	if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
		buf = buf[:n]
	} else if err != nil {
		return nil, err
	}
	g.shared.mu.Lock()
	g.shared.cache.Add(pageIdx, buf)
	g.shared.mu.Unlock()
	return buf, nil
}
