//go:build integration

package postgres

import (
	"context"
	"os"
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/config"
)

// TestFreshDB_AdvanceItemAndCountAtomic verifies the CTE that replaced the
// O(N²) SyncJob-per-webhook cascade. The webhook hot path now costs O(1) per
// event; correctness rides on this single method being both:
//
//  1. Correct — a non-terminal → terminal transition advances the item AND
//     increments the right job counter (completed → completed_count;
//     failed / cancelled → failed_count).
//  2. Idempotent — Argo's exit-hook is at-least-once. A repeat delivery must
//     find the item already terminal, update zero rows, and NOT double-count.
//     Plain IncrementCompleted/IncrementFailed have no such guard; that is
//     why they are not called from the webhook path.
//
// Requires an empty DB with migrations applied + INTEGRATION_DB=1.
func TestFreshDB_AdvanceItemAndCountAtomic(t *testing.T) {
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
	const (
		jobID    = "cyb-advance-job"
		tmplID   = "tpl-cyb-advance"
		itemDone = "cyb-advance-item-done"
		itemFail = "cyb-advance-item-fail"
		itemNoop = "cyb-advance-item-noop"
	)
	cleanup := func() {
		_ = db.Exec(ctx, `DELETE FROM backfill_items WHERE job_id = $1`, jobID)
		_ = db.Exec(ctx, `DELETE FROM backfill_jobs WHERE id = $1`, jobID)
		_ = db.Exec(ctx, `DELETE FROM pipeline_templates WHERE id = $1`, tmplID)
	}
	cleanup()
	t.Cleanup(cleanup)

	if err := db.Exec(ctx, `
INSERT INTO pipeline_templates(id, name, pipeline)
VALUES($1, 'advance-tpl', '{}')`, tmplID); err != nil {
		t.Fatalf("seed template: %v", err)
	}
	if err := db.Exec(ctx, `
INSERT INTO backfill_jobs(id, name, template_id, status, total_count, completed_count, failed_count)
VALUES($1, $1, $2, 'running', 3, 0, 0)`, jobID, tmplID); err != nil {
		t.Fatalf("seed job: %v", err)
	}
	for _, id := range []string{itemDone, itemFail, itemNoop} {
		if err := db.Exec(ctx, `
INSERT INTO backfill_items(id, job_id, asset_id, status)
VALUES($1, $2, $1, 'submitted')`, id, jobID); err != nil {
			t.Fatalf("seed item %s: %v", id, err)
		}
	}
	repo := NewBackfillRepo(client)

	// 1. completed transition bumps completed_count.
	if err := repo.AdvanceItemAndCountAtomic(ctx, itemDone, "completed", "wf-done", ""); err != nil {
		t.Fatalf("advance completed: %v", err)
	}
	// 2. failed transition bumps failed_count, records error message.
	if err := repo.AdvanceItemAndCountAtomic(ctx, itemFail, "failed", "wf-fail", "boom"); err != nil {
		t.Fatalf("advance failed: %v", err)
	}
	assertCounts(t, ctx, db, jobID, 1, 1)
	assertItemStatus(t, ctx, db, itemDone, "completed", "wf-done", "")
	assertItemStatus(t, ctx, db, itemFail, "failed", "wf-fail", "boom")

	// 3. Idempotency: replaying the completed webhook 4 more times must NOT
	//    double-count. This is the core guarantee — Argo's exit-hook can and
	//    does redeliver, and the old IncrementCompleted would +1 every call.
	for i := 0; i < 4; i++ {
		if err := repo.AdvanceItemAndCountAtomic(ctx, itemDone, "completed", "wf-done", ""); err != nil {
			t.Fatalf("advance completed replay %d: %v", i, err)
		}
	}
	// Also replay the failed one to prove the guard covers both branches.
	for i := 0; i < 4; i++ {
		if err := repo.AdvanceItemAndCountAtomic(ctx, itemFail, "failed", "wf-fail", "boom"); err != nil {
			t.Fatalf("advance failed replay %d: %v", i, err)
		}
	}
	assertCounts(t, ctx, db, jobID, 1, 1)

	// 4. cancelled counts as failed (matches SummarizeItemStatuses' bucketing).
	if err := repo.AdvanceItemAndCountAtomic(ctx, itemNoop, "cancelled", "wf-cancel", "paused"); err != nil {
		t.Fatalf("advance cancelled: %v", err)
	}
	assertCounts(t, ctx, db, jobID, 1, 2)

	// 5. Unsupported status is rejected without touching the DB — non-terminal
	//    transitions belong to UpdateItemStatus / UpdateItemPipelineRun.
	if err := repo.AdvanceItemAndCountAtomic(ctx, itemDone, "running", "", ""); err == nil {
		t.Fatal("expected error for non-terminal status, got nil")
	}
	assertCounts(t, ctx, db, jobID, 1, 2)

	// 6. Unknown item id is a silent no-op — the RETURNING set is empty, so
	//    the counter update runs over an empty IN () and does nothing.
	if err := repo.AdvanceItemAndCountAtomic(ctx, "ghost-item", "completed", "wf-ghost", ""); err != nil {
		t.Fatalf("advance ghost: %v", err)
	}
	assertCounts(t, ctx, db, jobID, 1, 2)
}

func assertCounts(t *testing.T, ctx context.Context, db pgDB, jobID string, wantCompleted, wantFailed int) {
	t.Helper()
	var completed, failed int
	if err := db.QueryRow(ctx,
		`SELECT completed_count, failed_count FROM backfill_jobs WHERE id = $1`, jobID,
	).Scan(&completed, &failed); err != nil {
		t.Fatalf("read counts: %v", err)
	}
	if completed != wantCompleted || failed != wantFailed {
		t.Fatalf("counts: got completed=%d failed=%d, want completed=%d failed=%d",
			completed, failed, wantCompleted, wantFailed)
	}
}

func assertItemStatus(t *testing.T, ctx context.Context, db pgDB, itemID, wantStatus, wantWF, wantErr string) {
	t.Helper()
	var status, wf, errMsg string
	if err := db.QueryRow(ctx,
		`SELECT status, COALESCE(workflow_name, ''), COALESCE(error_message, '') FROM backfill_items WHERE id = $1`, itemID,
	).Scan(&status, &wf, &errMsg); err != nil {
		t.Fatalf("read item %s: %v", itemID, err)
	}
	if status != wantStatus || wf != wantWF || errMsg != wantErr {
		t.Fatalf("item %s: got status=%q wf=%q err=%q, want status=%q wf=%q err=%q",
			itemID, status, wf, errMsg, wantStatus, wantWF, wantErr)
	}
}
