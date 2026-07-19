package postgres

import (
	"context"
	"encoding/json"
	"fmt"
)

// RuntimeConfigBlobRepo persists content-addressed runtime-config bytes as
// the rebuildable source of truth (CYB-3680). The in-cluster ConfigMap is a
// disposable projection of these rows.
type RuntimeConfigBlobRepo struct {
	c *Client
}

func NewRuntimeConfigBlobRepo(c *Client) *RuntimeConfigBlobRepo {
	return &RuntimeConfigBlobRepo{c: c}
}

// Upsert stores the files keyed by content hash. Re-upserting an existing
// hash only bumps last_used_at (content is immutable by construction — the
// hash IS the content identity).
func (r *RuntimeConfigBlobRepo) Upsert(ctx context.Context, hash string, files map[string]string) error {
	if hash == "" {
		return fmt.Errorf("runtime config blob: hash is required")
	}
	payload, err := json.Marshal(files)
	if err != nil {
		return fmt.Errorf("runtime config blob: marshal files: %w", err)
	}
	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, `
INSERT INTO runtime_config_blobs (hash, files)
VALUES ($1, $2)
ON CONFLICT (hash) DO UPDATE SET last_used_at = now()`, hash, payload); err != nil {
		return fmt.Errorf("postgres RuntimeConfigBlobRepo.Upsert: %w", err)
	}
	return nil
}
