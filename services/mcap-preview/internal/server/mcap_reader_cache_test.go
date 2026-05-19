package server

import (
	"bytes"
	"context"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/services/mcap-preview/internal/gcsrs"
)

// fakeOpener returns a bytes.Reader; the cache's bypass path (non-gcsrs) is
// exercised, which still walks every cache code path except the actual
// template-Clone path (covered separately via integration smoke on dev).
func fakeOpener(payload []byte, callCount *int64) Config_OpenMCAP {
	return func(_ context.Context, _ string) (io.ReadSeeker, *gcsrs.Stats, func(), error) {
		atomic.AddInt64(callCount, 1)
		return bytes.NewReader(payload), nil, nil, nil
	}
}

func TestMCAPReaderCache_BypassPath_CallsOpenerEveryTime(t *testing.T) {
	var calls int64
	c := newMCAPReaderCache(1*time.Minute, fakeOpener([]byte("hello"), &calls))

	for i := 0; i < 3; i++ {
		rs, _, _, err := c.Acquire(context.Background(), "gs://b/o.mcap")
		if err != nil {
			t.Fatalf("Acquire %d: %v", i, err)
		}
		b, _ := io.ReadAll(rs)
		if string(b) != "hello" {
			t.Errorf("got %q want hello", b)
		}
	}
	// Bypass path = the opener returns *bytes.Reader (not *gcsrs.GCSReadSeeker)
	// so the cache must transparently re-open every time.
	if got := atomic.LoadInt64(&calls); got != 3 {
		t.Fatalf("opener calls=%d want 3 (bypass each time)", got)
	}
}

func TestMCAPReaderCache_DifferentKeys_Isolated(t *testing.T) {
	var calls int64
	c := newMCAPReaderCache(1*time.Minute, fakeOpener([]byte("x"), &calls))

	for _, uri := range []string{"gs://a/1.mcap", "gs://b/2.mcap", "gs://a/1.mcap"} {
		if _, _, _, err := c.Acquire(context.Background(), uri); err != nil {
			t.Fatalf("acquire %s: %v", uri, err)
		}
	}
	// 3 calls in bypass mode (no caching across calls) — but the
	// singleflight should not collapse across distinct keys.
	if got := atomic.LoadInt64(&calls); got != 3 {
		t.Fatalf("opener calls=%d want 3", got)
	}
}

func TestMCAPReaderCache_OpenerError_Propagates(t *testing.T) {
	want := io.ErrUnexpectedEOF
	c := newMCAPReaderCache(1*time.Minute, func(_ context.Context, _ string) (io.ReadSeeker, *gcsrs.Stats, func(), error) {
		return nil, nil, nil, want
	})
	_, _, _, err := c.Acquire(context.Background(), "gs://b/o.mcap")
	if err != want {
		t.Fatalf("err=%v want %v", err, want)
	}
}

func TestMCAPReaderCache_NilOpener(t *testing.T) {
	c := newMCAPReaderCache(1*time.Minute, nil)
	if _, _, _, err := c.Acquire(context.Background(), "gs://b/o.mcap"); err == nil {
		t.Fatal("want err from nil opener")
	}
}

// Hammer Acquire concurrently to make sure the cache survives the race
// detector. Bypass path is benign; this primarily proves we don't deadlock
// under singleflight on simultaneous misses on the same key.
func TestMCAPReaderCache_Concurrent(t *testing.T) {
	var calls int64
	c := newMCAPReaderCache(1*time.Minute, fakeOpener([]byte("ok"), &calls))
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				_, _, _, _ = c.Acquire(context.Background(), "gs://b/o.mcap")
			}
		}()
	}
	wg.Wait()
}

func TestMCAPReaderCache_Invalidate(t *testing.T) {
	c := newMCAPReaderCache(1*time.Minute, fakeOpener([]byte("x"), new(int64)))
	// The map.delete path is safe even on a never-inserted key; just exercise.
	c.Invalidate("gs://b/never-inserted.mcap")
}

// TestMCAPReaderCache_BoundedSize directly exercises the put() eviction
// policy. The bypass-only test path can't populate `items`, so we manually
// stuff sentinel entries with controlled expiresAt to verify that:
//  1. Cache size never exceeds maxEntries.
//  2. When over capacity, already-expired entries are dropped before any
//     live entries.
//  3. With nothing expired, the entry with the earliest expiry wins eviction.
func TestMCAPReaderCache_BoundedSize(t *testing.T) {
	c := newMCAPReaderCacheWithMax(1*time.Hour, 3, fakeOpener(nil, new(int64)))
	now := time.Now()
	c.put("expired-1", &mcapCacheEntry{expiresAt: now.Add(-1 * time.Minute)})
	c.put("live-1", &mcapCacheEntry{expiresAt: now.Add(10 * time.Minute)})
	c.put("live-2", &mcapCacheEntry{expiresAt: now.Add(20 * time.Minute)})
	if got := len(c.items); got != 3 {
		t.Fatalf("after 3 puts len=%d want 3", got)
	}
	// Adding a 4th should evict the expired one first.
	c.put("live-3", &mcapCacheEntry{expiresAt: now.Add(30 * time.Minute)})
	if got := len(c.items); got != 3 {
		t.Fatalf("after eviction len=%d want 3", got)
	}
	if _, stillThere := c.items["expired-1"]; stillThere {
		t.Errorf("expired entry should be evicted first")
	}
	// One more — nothing's expired now, so the soonest-expiring live entry
	// (live-1) should be evicted.
	c.put("live-4", &mcapCacheEntry{expiresAt: now.Add(40 * time.Minute)})
	if got := len(c.items); got != 3 {
		t.Fatalf("after second eviction len=%d want 3", got)
	}
	if _, stillThere := c.items["live-1"]; stillThere {
		t.Errorf("soonest-expiring live entry should be evicted")
	}
}
