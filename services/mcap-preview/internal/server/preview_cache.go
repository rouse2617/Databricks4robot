package server

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

type previewCacheEntry struct {
	path      string
	expiresAt time.Time
}

type previewCache struct {
	dir   string
	ttl   time.Duration
	mu    sync.Mutex
	items map[string]previewCacheEntry
	sf    singleflight.Group
}

func newPreviewCache(dir string, ttl time.Duration) *previewCache {
	_ = os.MkdirAll(dir, 0o755)
	return &previewCache{
		dir:   dir,
		ttl:   ttl,
		items: map[string]previewCacheEntry{},
	}
}

func (c *previewCache) key(parts ...string) string {
	h := sha1.New()
	for _, p := range parts {
		_, _ = h.Write([]byte(p))
		_, _ = h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func (c *previewCache) get(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.items[key]
	if !ok || time.Now().After(e.expiresAt) {
		return "", false
	}
	if _, err := os.Stat(e.path); err != nil {
		delete(c.items, key)
		return "", false
	}
	return e.path, true
}

func (c *previewCache) put(key, path string) {
	c.mu.Lock()
	c.items[key] = previewCacheEntry{
		path:      path,
		expiresAt: time.Now().Add(c.ttl),
	}
	c.mu.Unlock()
}

// finalPath is the resting place of a successful cache entry.
func (c *previewCache) finalPath(key string) string {
	return filepath.Join(c.dir, key+".mp4")
}

// newTempFile returns an open writable temp file in the cache directory
// and its path. The caller is expected to either commit (via commit) or
// remove the path when finished.
func (c *previewCache) newTempFile(key string) (*os.File, string, error) {
	pattern := key + ".tmp.*.mp4"
	f, err := os.CreateTemp(c.dir, pattern)
	if err != nil {
		return nil, "", err
	}
	return f, f.Name(), nil
}

// commit atomically renames tmp into the canonical slot and registers it
// in the in-memory index.
func (c *previewCache) commit(key, tmp string) (string, error) {
	final := c.finalPath(key)
	if err := os.Rename(tmp, final); err != nil {
		return "", err
	}
	c.put(key, final)
	return final, nil
}

func (c *previewCache) getOrBuild(ctx context.Context, key string, build func(dst string) error) (string, error) {
	if p, ok := c.get(key); ok {
		return p, nil
	}
	v, err, _ := c.sf.Do(key, func() (any, error) {
		if p, ok := c.get(key); ok {
			return p, nil
		}
		dst := filepath.Join(c.dir, key+".mp4")
		tmp := dst + ".tmp"
		_ = os.Remove(tmp)
		if err := build(tmp); err != nil {
			_ = os.Remove(tmp)
			return "", err
		}
		if err := os.Rename(tmp, dst); err != nil {
			_ = os.Remove(tmp)
			return "", err
		}
		c.put(key, dst)
		return dst, nil
	})
	if err != nil {
		return "", err
	}
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}
	return v.(string), nil
}

