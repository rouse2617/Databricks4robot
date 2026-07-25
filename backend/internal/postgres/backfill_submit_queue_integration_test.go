//go:build integration

package postgres

import (
	"context"
	"os"
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/config"
)

// TestFreshDB_SubmitQueue verifies the CYB-3491 submitter persistence surface
// against a real PostgreSQL:
//
//  1. FindSubmittableJobs picks running AND pilot_running jobs with pending
//     items (the legacy resume path silently skipped pilots), and skips jobs
//     whose items are all in-flight/terminal.
//  2. ListSubmittableItemIDs returns pending ids oldest-first.
//  3. LockPendingItem is the cross-worker mutual exclusion: while one
//     transaction holds the row, a second transaction sees nil (SKIP LOCKED)
//     instead of blocking; after rollback the row is lockable again; a
//     non-pending item is never lockable.
//
// Requires an empty DB with migrations applied (see test-integration.yml
// backend-fresh-db job) + INTEGRATION_DB=1.
func TestFreshDB_SubmitQueue(t *testing.T) {
	if os.Getenv("INTEGRATION_DB") != "1" {
		t.Skip("set INTEGRATION_DB=1 with Postgres env (DB_HOST, DB_USER, DB_PASSWORD, DB_NAME)")
	}

	ctx := context.Background()
	cfg := config.Load()
	client, err := New(ctx, cfg)
	if err != nil {
		t.Fatalf("postgres connect: %v", err)
	}
	t.Cleanup(client.Close)

	db := dbFromCtx(ctx, client.db)
	const jobRun = "cyb3491-job-running"
	const jobPilot = "cyb3491-job-pilot"
	const jobDone = "cyb3491-job-done"

	cleanup := func() {
		_ = db.Exec(ctx, `DELETE FROM backfill_items WHERE job_id LIKE 'cyb3491-job-%'`)
		_ = db.Exec(ctx, `DELETE FROM backfill_jobs WHERE id LIKE 'cyb3491-job-%'`)
		_ = db.Exec(ctx, `DELETE FROM pipeline_templates WHERE id = 'tpl-cyb3491'`)
	}
	cleanup()
	t.Cleanup(cleanup)

	if err := db.Exec(ctx, `
INSERT INTO pipeline_templates(id, name, pipeline) VALUES('tpl-cyb3491', 'nw-delivery-dev', '{}')`); err != nil {
		t.Fatalf("seed template: %v", err)
	}
	for _, j := range []struct{ id, status string }{
		{jobRun, "running"},
		{jobPilot, "pilot_running"},
		{jobDone, "running"},
	} {
		if err := db.Exec(ctx, `
INSERT INTO backfill_jobs(id, name, template_id, status, total_count)
VALUES($1, $1, 'tpl-cyb3491', $2, 2)`, j.id, j.status); err != nil {
			t.Fatalf("seed job %s: %v", j.id, err)
		}
	}
	// jobRun: two pending items with distinct ages (ordering check).
	// jobPilot: one pending item (pilot jobs must be submittable).
	// jobDone: only submitted/completed items (must NOT be submittable).
	for _, it := range []struct{ id, job, status, age string }{
		{"cyb3491-item-old", jobRun, "pending", "-2 hours"},
		{"cyb3491-item-new", jobRun, "pending", "-1 hours"},
		{"cyb3491-item-pilot", jobPilot, "pending", "-1 hours"},
		{"cyb3491-item-inflight", jobDone, "submitted", "-1 hours"},
		{"cyb3491-item-done", jobDone, "completed", "-1 hours"},
	} {
		if err := db.Exec(ctx, `
INSERT INTO backfill_items(id, job_id, asset_id, status, created_at)
VALUES($1, $2, $1, $3, NOW() + $4::interval)`, it.id, it.job, it.status, it.age); err != nil {
			t.Fatalf("seed item %s: %v", it.id, err)
		}
	}

	// CYB-3677: the pinned template version must round-trip through
	// FindSubmittableJobs (NULL coalesces to 0 for the others).
	if err := db.Exec(ctx, `UPDATE backfill_jobs SET template_version = 7 WHERE id = $1`, jobRun); err != nil {
		t.Fatalf("pin template_version: %v", err)
	}
	if err := db.Exec(ctx, `UPDATE backfill_jobs SET created_by = 'alice@example.com' WHERE id = $1`, jobRun); err != nil {
		t.Fatalf("set created_by: %v", err)
	}

	repo := NewBackfillRepo(client)

	// 1. Submittable jobs: running + pilot_running with pending items only.
	jobs, err := repo.FindSubmittableJobs(ctx, 50)
	if err != nil {
		t.Fatalf("FindSubmittableJobs: %v", err)
	}
	for _, j := range jobs {
		if j.ID == jobRun && j.TemplateVersion != 7 {
			t.Errorf("job %s template_version = %d, want 7 (CYB-3677 pin)", jobRun, j.TemplateVersion)
		}
		if j.ID == jobRun && j.CreatedBy != "alice@example.com" {
			t.Errorf("job %s created_by = %q, want alice@example.com", jobRun, j.CreatedBy)
		}
		if j.ID == jobPilot && j.TemplateVersion != 0 {
			t.Errorf("job %s template_version = %d, want 0 (NULL coalesced)", jobPilot, j.TemplateVersion)
		}
	}
	got := map[string]bool{}
	for _, j := range jobs {
		if len(j.ID) >= 7 && j.ID[:7] == "cyb3491" {
			got[j.ID] = true
		}
	}
	if !got[jobRun] || !got[jobPilot] {
		t.Fatalf("submittable jobs missing running/pilot: %v", got)
	}
	if got[jobDone] {
		t.Fatalf("job with no pending items must not be submittable: %v", got)
	}

	// 2. Candidates are pending-only, oldest first.
	ids, err := repo.ListSubmittableItemIDs(ctx, jobRun, 10)
	if err != nil {
		t.Fatalf("ListSubmittableItemIDs: %v", err)
	}
	if len(ids) != 2 || ids[0] != "cyb3491-item-old" || ids[1] != "cyb3491-item-new" {
		t.Fatalf("candidates = %v, want [cyb3491-item-old cyb3491-item-new]", ids)
	}

	// 3a. SKIP LOCKED exclusivity: tx1 locks; tx2 sees nil without blocking.
	tx1Locked := make(chan struct{})
	tx1Release := make(chan struct{})
	tx1Done := make(chan error, 1)
	go func() {
		tx1Done <- client.WithTx(ctx, func(txCtx context.Context) error {
			item, err := repo.LockPendingItem(txCtx, "cyb3491-item-old")
			if err != nil {
				return err
			}
			if item == nil {
				t.Error("tx1: expected to lock the pending item")
			}
			close(tx1Locked)
			<-tx1Release
			return context.Canceled // force rollback: item stays pending
		})
	}()
	<-tx1Locked
	err = client.WithTx(ctx, func(txCtx context.Context) error {
		item, err := repo.LockPendingItem(txCtx, "cyb3491-item-old")
		if err != nil {
			return err
		}
		if item != nil {
			t.Errorf("tx2: expected nil for a row locked by tx1 (SKIP LOCKED), got %q", item.ID)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("tx2: %v", err)
	}
	close(tx1Release)
	if err := <-tx1Done; err == nil {
		t.Fatalf("tx1: expected forced rollback error")
	}

	// 3b. After tx1's rollback the item is pending again and lockable.
	err = client.WithTx(ctx, func(txCtx context.Context) error {
		item, err := repo.LockPendingItem(txCtx, "cyb3491-item-old")
		if err != nil {
			return err
		}
		if item == nil {
			t.Error("post-rollback: expected the item to be lockable again")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("post-rollback lock: %v", err)
	}

	// 3c. Non-pending items are never lockable.
	err = client.WithTx(ctx, func(txCtx context.Context) error {
		item, err := repo.LockPendingItem(txCtx, "cyb3491-item-inflight")
		if err != nil {
			return err
		}
		if item != nil {
			t.Errorf("submitted item must not be lockable, got %q", item.ID)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("non-pending lock: %v", err)
	}
}

// TestFreshDB_SubmitterCycleLock verifies the CYB-3677 cross-instance mutual
// exclusion against a real PostgreSQL: two clients = two pools = two sessions.
// While session 1 holds the cycle advisory lock, session 2's try is refused
// without blocking; after release, session 2 acquires. Session scoping also
// means a crashed holder's lock dies with its connection (natural lease).
func TestFreshDB_SubmitterCycleLock(t *testing.T) {
	if os.Getenv("INTEGRATION_DB") != "1" {
		t.Skip("set INTEGRATION_DB=1 with Postgres env (DB_HOST, DB_USER, DB_PASSWORD, DB_NAME)")
	}

	ctx := context.Background()
	cfg := config.Load()
	c1, err := New(ctx, cfg)
	if err != nil {
		t.Fatalf("postgres connect #1: %v", err)
	}
	t.Cleanup(c1.Close)
	c2, err := New(ctx, cfg)
	if err != nil {
		t.Fatalf("postgres connect #2: %v", err)
	}
	t.Cleanup(c2.Close)

	r1, r2 := NewBackfillRepo(c1), NewBackfillRepo(c2)

	ran1 := 0
	acq1, err := r1.WithSubmitterClusterLock(ctx, "clu-int", func(ctx context.Context) error {
		ran1++
		// While held, a second session must be refused, non-blocking.
		acq2, err := r2.WithSubmitterClusterLock(ctx, "clu-int", func(context.Context) error {
			t.Error("second session must not run while the lock is held")
			return nil
		})
		if err != nil {
			return err
		}
		if acq2 {
			t.Error("second session acquired the lock while held, want refusal")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("session 1 lock: %v", err)
	}
	if !acq1 || ran1 != 1 {
		t.Fatalf("session 1: acquired=%v ran=%d, want true/1", acq1, ran1)
	}

	// After release the lock is free again.
	acq2b, err := r2.WithSubmitterClusterLock(ctx, "clu-int", func(context.Context) error { return nil })
	if err != nil {
		t.Fatalf("session 2 post-release lock: %v", err)
	}
	if !acq2b {
		t.Fatal("session 2 must acquire after session 1 released")
	}
}
