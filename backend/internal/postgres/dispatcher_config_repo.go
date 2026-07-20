package postgres

import (
	"context"
	"fmt"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// DispatcherConfigRepo persists per-cluster dispatcher tuning (CYB-3679).
type DispatcherConfigRepo struct {
	c *Client
}

func NewDispatcherConfigRepo(c *Client) *DispatcherConfigRepo {
	return &DispatcherConfigRepo{c: c}
}

const dispatcherConfigCols = `cluster_id, max_concurrency, submit_batch, rate_per_sec, paused, updated_by, updated_at`

// List returns all explicit cluster rows.
func (r *DispatcherConfigRepo) List(ctx context.Context) ([]models.DispatcherConfig, error) {
	rows, err := r.c.db.Query(ctx, `SELECT `+dispatcherConfigCols+` FROM dispatcher_configs ORDER BY cluster_id`)
	if err != nil {
		return nil, fmt.Errorf("dispatcher config: list: %w", err)
	}
	defer rows.Close()
	var out []models.DispatcherConfig
	for rows.Next() {
		var c models.DispatcherConfig
		if err := rows.Scan(&c.ClusterID, &c.MaxConcurrency, &c.SubmitBatch, &c.RatePerSec, &c.Paused, &c.UpdatedBy, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("dispatcher config: scan: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// Upsert writes one cluster's config (full-row semantics).
func (r *DispatcherConfigRepo) Upsert(ctx context.Context, cfg *models.DispatcherConfig) error {
	if cfg == nil || cfg.ClusterID == "" {
		return fmt.Errorf("dispatcher config: cluster_id is required")
	}
	_, err := r.c.db.ExecResult(ctx, `
		INSERT INTO dispatcher_configs (cluster_id, max_concurrency, submit_batch, rate_per_sec, paused, updated_by, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, now())
		ON CONFLICT (cluster_id) DO UPDATE SET
			max_concurrency = EXCLUDED.max_concurrency,
			submit_batch    = EXCLUDED.submit_batch,
			rate_per_sec    = EXCLUDED.rate_per_sec,
			paused          = EXCLUDED.paused,
			updated_by      = EXCLUDED.updated_by,
			updated_at      = now()`,
		cfg.ClusterID, cfg.MaxConcurrency, cfg.SubmitBatch, cfg.RatePerSec, cfg.Paused, cfg.UpdatedBy)
	if err != nil {
		return fmt.Errorf("dispatcher config: upsert %s: %w", cfg.ClusterID, err)
	}
	return nil
}

// Delete removes a cluster's row (back to compiled defaults).
func (r *DispatcherConfigRepo) Delete(ctx context.Context, clusterID string) error {
	if clusterID == "" {
		return fmt.Errorf("dispatcher config: cluster_id is required")
	}
	if _, err := r.c.db.ExecResult(ctx, `DELETE FROM dispatcher_configs WHERE cluster_id = $1`, clusterID); err != nil {
		return fmt.Errorf("dispatcher config: delete %s: %w", clusterID, err)
	}
	return nil
}
