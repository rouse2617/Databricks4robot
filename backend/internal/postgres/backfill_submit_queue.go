package postgres

import (
	"context"
	"errors"
	"fmt"

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

// submitterCycleLockKey scopes the submitter's cross-instance mutual
// exclusion. One fixed key = one global cycle at a time (CYB-3677); CYB-3678
// shards this to per-cluster keys.
const submitterCycleLockKey int64 = 0x63796237_37000001 // "cyb77" | v1

// advisoryLocker is implemented by the pool-backed pgDB (realDB). Tx-bound
// DBs and test fakes don't implement it, which callers treat as "locking
// unavailable — run unguarded" (single-instance environments).
type advisoryLocker interface {
	WithAdvisoryLock(ctx context.Context, key int64, fn func(context.Context) error) (bool, error)
}

// WithSubmitterCycleLock runs one submitter cycle under the cross-instance
// advisory lock. acquired=false means another instance is currently running a
// cycle and fn was skipped. Environments whose pgDB cannot take session locks
// run fn unguarded (acquired=true) — correctness is still covered by the
// deterministic-workflow-name + AlreadyExists convergence; the lock only
// removes wasted duplicate attempts.
func (r *BackfillRepo) WithSubmitterCycleLock(ctx context.Context, fn func(context.Context) error) (bool, error) {
	if l, ok := r.c.db.(advisoryLocker); ok {
		return l.WithAdvisoryLock(ctx, submitterCycleLockKey, fn)
	}
	return true, fn(ctx)
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
	// template_version is selected so the submitter honours the version pinned
	// at batch creation instead of silently falling back to the template's
	// current active version (CYB-3677 P0: same-batch version consistency).
	const q = `
	SELECT bj.id, bj.template_id, bj.name, bj.status,
	  bj.completed_count, bj.failed_count, bj.total_count,
	  bj.pilot_phase, bj.pilot_count, COALESCE(bj.template_version, 0),
	  bj.filter_json, bj.created_at, bj.updated_at
	FROM backfill_jobs bj
	WHERE bj.status IN ('running', 'pilot_running')
	  AND EXISTS (
	    SELECT 1 FROM backfill_items bi
	    WHERE bi.job_id = bj.id AND bi.status = 'pending'
	  )
	ORDER BY bj.created_at ASC
	LIMIT $1`
	db := dbFromCtx(ctx, r.c.db)
	rows, err := db.Query(ctx, q, limit)
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
			&j.FilterJSON, &j.CreatedAt, &j.UpdatedAt,
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
