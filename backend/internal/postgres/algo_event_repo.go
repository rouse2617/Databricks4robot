package postgres

import (
	"context"
	"fmt"
	"time"

	"data-platform/internal/models"
	"data-platform/internal/repository"
)

// AlgoEventRepo implements repository.AlgoEventRepository using PostgreSQL.
type AlgoEventRepo struct {
	c *Client
}

var _ repository.AlgoEventRepository = (*AlgoEventRepo)(nil)

// NewAlgoEventRepo creates a new AlgoEventRepo.
func NewAlgoEventRepo(c *Client) *AlgoEventRepo { return &AlgoEventRepo{c: c} }

// Insert persists a single algorithm status-change event.
func (r *AlgoEventRepo) Insert(ctx context.Context, event *models.AlgoEvent) error {
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	const q = `
INSERT INTO asset_algo_events (event_id, asset_id, algo_key, prev_status, new_status, run_id, reason, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	err := r.c.db.Exec(ctx, q,
		event.EventID, event.AssetID, event.AlgoKey,
		event.PrevStatus, event.NewStatus,
		event.RunID, event.Reason, event.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("postgres AlgoEventRepo.Insert: %w", err)
	}
	return nil
}

// ListByAsset returns events for the given asset, ordered by created_at DESC.
// When algoKey is non-nil, only events matching that algo_key are returned.
func (r *AlgoEventRepo) ListByAsset(ctx context.Context, assetID string, algoKey *string) ([]*models.AlgoEvent, error) {
	var (
		q    string
		args []interface{}
	)
	if algoKey != nil {
		q = `
SELECT event_id, asset_id, algo_key, prev_status, new_status, run_id, reason, created_at
FROM asset_algo_events
WHERE asset_id = $1 AND algo_key = $2
ORDER BY created_at DESC`
		args = []interface{}{assetID, *algoKey}
	} else {
		q = `
SELECT event_id, asset_id, algo_key, prev_status, new_status, run_id, reason, created_at
FROM asset_algo_events
WHERE asset_id = $1
ORDER BY created_at DESC`
		args = []interface{}{assetID}
	}

	rows, err := r.c.db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("postgres AlgoEventRepo.ListByAsset: %w", err)
	}
	defer rows.Close()

	var out []*models.AlgoEvent
	for rows.Next() {
		var e models.AlgoEvent
		if err := rows.Scan(
			&e.EventID, &e.AssetID, &e.AlgoKey,
			&e.PrevStatus, &e.NewStatus,
			&e.RunID, &e.Reason, &e.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("postgres AlgoEventRepo.ListByAsset scan: %w", err)
		}
		out = append(out, &e)
	}
	return out, nil
}
