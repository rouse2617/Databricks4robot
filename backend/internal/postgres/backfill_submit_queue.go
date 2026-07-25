package postgres

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

// SubmitQueue implementation (CYB-3491). The submitter's idempotency story:
// candidates are read lock-free, then each item is re-checked and locked
// (FOR UPDATE SKIP LOCKED) inside a single-item transaction; the Argo submit
// and the uid/status writes ride that transaction. A crash rolls the item
// back to pending; the workflow that may already exist in Argo is healed on
// the next cycle by the deterministic name + AlreadyExists backfill.

var _ repository.SubmitQueue = (*BackfillRepo)(nil)

// WithTx exposes the client's transaction runner to the submitter.
func (r *BackfillRepo) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.c.WithTx(ctx, fn)
}

// submitterCycleLockKey is the base of the per-cluster lock keyspace: each
// cluster's channel locks base ^ fnv64a(cluster), so channels are mutually
// independent across instances (CYB-3678; supersedes the CYB-3677 global key).
const submitterCycleLockKey int64 = 0x63796237_37000001 // "cyb77" | v1

func clusterLockKey(cluster string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(cluster))
	return submitterCycleLockKey ^ int64(h.Sum64())
}

// advisoryLocker is implemented by the pool-backed pgDB (realDB). Tx-bound
// DBs and test fakes don't implement it, which callers treat as "locking
// unavailable — run unguarded" (single-instance environments).
type advisoryLocker interface {
	WithAdvisoryLock(ctx context.Context, key int64, fn func(context.Context) error) (bool, error)
}

// WithSubmitterClusterLock runs one cluster channel's cycle under that
// cluster's cross-instance advisory lock. acquired=false means another
// instance currently runs this cluster's channel and fn was skipped.
// Environments whose pgDB cannot take session locks run fn unguarded
// (acquired=true) — correctness is still covered by the deterministic
// workflow-name + AlreadyExists convergence; the lock only removes wasted
// duplicate attempts.
func (r *BackfillRepo) WithSubmitterClusterLock(ctx context.Context, cluster string, fn func(context.Context) error) (bool, error) {
	if l, ok := r.c.db.(advisoryLocker); ok {
		return l.WithAdvisoryLock(ctx, clusterLockKey(cluster), fn)
	}
	return true, fn(ctx)
}

// IncrementItemSubmitAttempts bumps the durable transient-retry counter
// (CYB-3678) and returns the new value.
func (r *BackfillRepo) IncrementItemSubmitAttempts(ctx context.Context, itemID string) (int, error) {
	db := dbFromCtx(ctx, r.c.db)
	var attempts int
	if err := db.QueryRow(ctx, `
UPDATE backfill_items SET submit_attempts = submit_attempts + 1
WHERE id = $1 RETURNING submit_attempts`, itemID).Scan(&attempts); err != nil {
		return 0, fmt.Errorf("postgres BackfillRepo.IncrementItemSubmitAttempts: %w", err)
	}
	return attempts, nil
}

// ResetFailedItems re-queues a job's DLQ: failed → pending, error cleared,
// attempt counter reset (an explicit human retry earns a fresh cap).
func (r *BackfillRepo) ResetFailedItems(ctx context.Context, jobID string) (int64, error) {
	db := dbFromCtx(ctx, r.c.db)
	n, err := db.ExecResult(ctx, `
UPDATE backfill_items
SET status = 'pending', error_message = NULL, submit_attempts = 0
WHERE job_id = $1 AND status = 'failed'`, jobID)
	if err != nil {
		return 0, fmt.Errorf("postgres BackfillRepo.ResetFailedItems: %w", err)
	}
	return n, nil
}

// FindSubmittableJobs returns running / pilot_running jobs that still have
// pending items, oldest first. Unlike the legacy FindIncompleteJobs it also
// covers pilot_running (the old resume path silently skipped pilots) and no
// longer needs the half-committed-orphan clause: in the submitter model an
// item stays `pending` until its run has a persisted Argo UID, so orphans
// are just pending items.
func (r *BackfillRepo) FindSubmittableJobs(ctx context.Context, limit int) ([]models.BackfillJob, error) {
	if limit <= 0 {
		limit = 50
	}
	perTargetLimit := limit / 4
	if perTargetLimit < 1 {
		perTargetLimit = 1
	}
	// template_version is selected so the submitter honours the version pinned
	// at batch creation instead of silently falling back to the template's
	// current active version (CYB-3677 P0: same-batch version consistency).
	const q = `
	WITH candidates AS (
	  SELECT
	    bj.id, bj.template_id, bj.name, bj.status,
	    bj.completed_count, bj.failed_count, bj.total_count,
	    bj.pilot_phase, bj.pilot_count, COALESCE(bj.template_version, 0) AS template_version,
	    bj.filter_json, bj.created_at, bj.updated_at,
	    COALESCE(bj.created_by, '') AS created_by,
	    row_number() OVER (
	      PARTITION BY COALESCE(
	        NULLIF(bj.filter_json->>'targetId', ''),
	        NULLIF(bj.filter_json->>'target_id', ''),
	        ''
	      )
	      ORDER BY bj.created_at ASC
	    ) AS target_rank
	  FROM backfill_jobs bj
	  WHERE bj.status IN ('running', 'pilot_running')
	    AND EXISTS (
	      SELECT 1 FROM backfill_items bi
	      WHERE bi.job_id = bj.id AND bi.status = 'pending'
	    )
	)
	SELECT bj.id, bj.template_id, bj.name, bj.status,
	  bj.completed_count, bj.failed_count, bj.total_count,
	  bj.pilot_phase, bj.pilot_count, bj.template_version,
	  bj.filter_json, bj.created_at, bj.updated_at, bj.created_by
	FROM candidates bj
	WHERE bj.target_rank <= $2
	ORDER BY bj.created_at ASC
	LIMIT $1`
	db := dbFromCtx(ctx, r.c.db)
	rows, err := db.Query(ctx, q, limit, perTargetLimit)
	if err != nil {
		return nil, fmt.Errorf("postgres BackfillRepo.FindSubmittableJobs: %w", err)
	}
	defer rows.Close()
	var jobs []models.BackfillJob
	for rows.Next() {
		var j models.BackfillJob
		if err := rows.Scan(
			&j.ID, &j.TemplateID, &j.Name, &j.Status,
			&j.CompletedCount, &j.FailedCount, &j.TotalCount,
			&j.PilotPhase, &j.PilotCount, &j.TemplateVersion,
			&j.FilterJSON, &j.CreatedAt, &j.UpdatedAt, &j.CreatedBy,
		); err != nil {
			return nil, fmt.Errorf("postgres BackfillRepo.FindSubmittableJobs scan: %w", err)
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}

// ListSubmittableItemIDs returns pending item ids for a job, oldest first,
// without locking.
func (r *BackfillRepo) ListSubmittableItemIDs(ctx context.Context, jobID string, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 32
	}
	const q = `
	SELECT id FROM backfill_items
	WHERE job_id = $1 AND status = 'pending'
	ORDER BY created_at ASC
	LIMIT $2`
	db := dbFromCtx(ctx, r.c.db)
	rows, err := db.Query(ctx, q, jobID, limit)
	if err != nil {
		return nil, fmt.Errorf("postgres BackfillRepo.ListSubmittableItemIDs: %w", err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("postgres BackfillRepo.ListSubmittableItemIDs scan: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// LockPendingItem re-checks and row-locks one still-pending item inside the
// caller's transaction. SKIP LOCKED makes a concurrently-locked row look like
// "not pending" — the caller skips it; nothing blocks.
func (r *BackfillRepo) LockPendingItem(ctx context.Context, itemID string) (*models.BackfillItem, error) {
	const q = `
	SELECT id, job_id, asset_id, status,
	  pipeline_run_id, workflow_name, error_message, attempts, started_at, finished_at,
	  created_at
	FROM backfill_items
	WHERE id = $1 AND status = 'pending'
	FOR UPDATE SKIP LOCKED`
	db := dbFromCtx(ctx, r.c.db)
	item, err := scanBackfillItem(db.QueryRow(ctx, q, itemID))
	if err != nil {
		if errors.Is(err, errNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres BackfillRepo.LockPendingItem: %w", err)
	}
	return item, nil
}
