package main

import (
	"context"
	"errors"
	"fmt"
	"io"

	"cloud.google.com/go/storage"
	"github.com/golang/groupcache/lru"
)

// gcsReadSeeker implements io.ReadSeeker over GCS object bytes using Range reads.
// It batches many small reads via fixed-size page cache to reduce Range request count.
type gcsReadSeeker struct {
	ctx      context.Context
	obj      *storage.ObjectHandle
	size     int64
	pos      int64
	pageSize int64
	cache    *lru.Cache

	rangeCalls  int64
	cacheHits   int64
	cacheMisses int64
	bytesFetch  int64
}

func newGCSReadSeeker(ctx context.Context, obj *storage.ObjectHandle, size int64) *gcsReadSeeker {
	c := lru.New(64) // up to 64 pages in memory
	return &gcsReadSeeker{
		ctx:      ctx,
		obj:      obj,
		size:     size,
		pos:      0,
		pageSize: 1 << 20, // 1 MiB
		cache:    c,
	}
}

func (g *gcsReadSeeker) Read(p []byte) (int, error) {
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
		pageIdx := g.pos / g.pageSize
		pageOff := g.pos % g.pageSize
		page, err := g.getPage(pageIdx)
		if err != nil {
			return int(copied), err
		}
		available := int64(len(page)) - pageOff
		need := toRead - copied
		if need > available {
			need = available
		}
		start := int(pageOff)
		end := int(pageOff + need)
		dstStart := int(copied)
		dstEnd := int(copied + need)
		copy(p[dstStart:dstEnd], page[start:end])
		copied += need
		g.pos += need
	}
	return int(copied), nil
}

func (g *gcsReadSeeker) Seek(offset int64, whence int) (int64, error) {
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

func (g *gcsReadSeeker) getPage(pageIdx int64) ([]byte, error) {
	if got, ok := g.cache.Get(pageIdx); ok {
		g.cacheHits++
		return got.([]byte), nil
	}
	g.cacheMisses++

	start := pageIdx * g.pageSize
	if start >= g.size {
		return nil, io.EOF
	}
	length := g.pageSize
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
	g.rangeCalls++
	g.bytesFetch += int64(n)
	if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
		buf = buf[:n]
	} else if err != nil {
		return nil, err
	}
	g.cache.Add(pageIdx, buf)
	return buf, nil
}

func (g *gcsReadSeeker) StatsSummary() string {
	return fmt.Sprintf(
		"GCS page-cache stats: page_size=%d cache_entries<=64 range_calls=%d cache_hits=%d cache_misses=%d bytes_fetched=%d (%.2f MiB)",
		g.pageSize,
		g.rangeCalls,
		g.cacheHits,
		g.cacheMisses,
		g.bytesFetch,
		float64(g.bytesFetch)/(1024*1024),
	)
}
