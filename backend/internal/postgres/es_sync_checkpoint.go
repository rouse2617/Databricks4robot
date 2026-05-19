package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

// ESSyncCheckpointRepo persists per-shard high-water marks of event_seq that
// the ES subscriber has applied (see migrations/022_es_sync_checkpoint.sql).
//
// Writes are idempotent (UPSERT + GREATEST), so it is always safe for a
// subscriber to call Upsert with stale data after a retry: the watermark only
// moves forward.
type ESSyncCheckpointRepo struct {
	c *Client
}

// NewESSyncCheckpointRepo constructs the repo bound to a postgres client.
func NewESSyncCheckpointRepo(c *Client) *ESSyncCheckpointRepo {
	return &ESSyncCheckpointRepo{c: c}
}

func isMissingRelation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "42P01"
}

// Upsert advances applied_seq for shardID to GREATEST(current, appliedSeq).
// A non-positive appliedSeq is a no-op (defensive — checkpoint should never
// regress).
func (r *ESSyncCheckpointRepo) Upsert(ctx context.Context, shardID int, appliedSeq int64) error {
	if appliedSeq <= 0 {
		return nil
	}
	const q = `
INSERT INTO es_sync_checkpoint (shard_id, applied_seq, updated_at)
VALUES ($1, $2, now())
ON CONFLICT (shard_id) DO UPDATE
   SET applied_seq = GREATEST(es_sync_checkpoint.applied_seq, EXCLUDED.applied_seq),
       updated_at  = now()`
	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, q, shardID, appliedSeq); err != nil {
		if isMissingRelation(err) {
			return nil
		}
		return fmt.Errorf("postgres ESSyncCheckpointRepo.Upsert: %w", err)
	}
	return nil
}

// MinAppliedSeq returns MIN(applied_seq) across all shards. This is the
// conservative cross-shard watermark: every event_seq <= the returned value
// is guaranteed to have been ack'd by ES on its shard.
//
// Returns 0 if no shards have reported yet (fresh deployment / empty table).
// expectedShards lets callers force "MIN over a full set of shards" — when
// fewer rows exist than expectedShards (e.g. some shards never received any
// event yet), MinAppliedSeq returns 0 so consumer_lag stays conservative and
// doesn't silently overstate ES progress. Set expectedShards <= 0 to skip the
// gate (use observed rows only).
//
// Idle-shard advance: when idleAfterSec > 0 and publishedMax > 0, any row
// whose updated_at is older than idleAfterSec is treated as having
// applied_seq = publishedMax for MIN purposes. This prevents a low-traffic
// shard from dragging consumer_lag up forever when nothing new is published
// to its key space. Pass idleAfterSec=0 to disable (legacy behavior).
func (r *ESSyncCheckpointRepo) MinAppliedSeq(ctx context.Context, expectedShards int, idleAfterSec int, publishedMax int64) (int64, error) {
	const countQ = `SELECT COUNT(*)::int FROM es_sync_checkpoint`
	db := dbFromCtx(ctx, r.c.db)
	if expectedShards > 0 {
		var n int
		if err := db.QueryRow(ctx, countQ).Scan(&n); err != nil {
			if isMissingRelation(err) {
				return 0, nil
			}
			return 0, fmt.Errorf("postgres ESSyncCheckpointRepo.MinAppliedSeq count: %w", err)
		}
		if n < expectedShards {
			return 0, nil
		}
	}
	var seq int64
	if idleAfterSec > 0 && publishedMax > 0 {
		const advQ = `
SELECT COALESCE(
  MIN(CASE WHEN EXTRACT(EPOCH FROM (now() - updated_at)) > $1 THEN $2 ELSE applied_seq END),
  0
) FROM es_sync_checkpoint`
		if err := db.QueryRow(ctx, advQ, idleAfterSec, publishedMax).Scan(&seq); err != nil {
			if isMissingRelation(err) {
				return 0, nil
			}
			return 0, fmt.Errorf("postgres ESSyncCheckpointRepo.MinAppliedSeq idle-advance: %w", err)
		}
		return seq, nil
	}
	const minQ = `SELECT COALESCE(MIN(applied_seq), 0) FROM es_sync_checkpoint`
	if err := db.QueryRow(ctx, minQ).Scan(&seq); err != nil {
		if isMissingRelation(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("postgres ESSyncCheckpointRepo.MinAppliedSeq: %w", err)
	}
	return seq, nil
}
