package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// LakehouseBronzeCheckpoint holds the latest state recorded by the
// bronze-incremental Cloud Run Job (see migrations/023_lakehouse_bronze_checkpoint.sql).
type LakehouseBronzeCheckpoint struct {
	AppliedSeq int64
	IngestedAt time.Time
	RunID      string
	UpdatedAt  time.Time
}

// LakehouseBronzeCheckpointRepo reads the single-row checkpoint table.
// Writes come from the Cloud Run Job (Python), not from Go, so this repo is
// read-only on the backend side.
type LakehouseBronzeCheckpointRepo struct {
	c *Client
}

func NewLakehouseBronzeCheckpointRepo(c *Client) *LakehouseBronzeCheckpointRepo {
	return &LakehouseBronzeCheckpointRepo{c: c}
}

// Get returns the latest checkpoint, or (nil, nil) when the row does not yet
// exist (fresh deployment — bronze-incremental has never run) or when the
// table itself is missing (migration not applied yet). Callers should treat
// both cases as "Bronze empty / unknown" rather than as errors.
func (r *LakehouseBronzeCheckpointRepo) Get(ctx context.Context) (*LakehouseBronzeCheckpoint, error) {
	const q = `
SELECT applied_seq, ingested_at, COALESCE(run_id, ''), updated_at
FROM lakehouse_bronze_checkpoint
WHERE id = 1`
	db := dbFromCtx(ctx, r.c.db)
	var cp LakehouseBronzeCheckpoint
	err := db.QueryRow(ctx, q).Scan(&cp.AppliedSeq, &cp.IngestedAt, &cp.RunID, &cp.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		if isMissingRelation(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres LakehouseBronzeCheckpointRepo.Get: %w", err)
	}
	return &cp, nil
}
