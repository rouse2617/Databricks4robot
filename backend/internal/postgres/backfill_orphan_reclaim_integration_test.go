//go:build integration

package postgres

import (
	"context"
	"os"
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/config"
)

// TestFreshDB_ClaimNextItem_ReclaimsHalfCommittedOrphans verifies CYB-3480:
// ClaimNextItem re-claims a batch item stuck 'running' on an unsubmitted
// placeholder run (a "half-committed orphan"), while leaving genuinely-running
// items (Argo UID assigned) and freshly-created in-flight deploys alone. It
// also confirms normal 'pending' items are still claimed (and claimed first).
//
// Requires an empty DB with migrations applied (see test-integration.yml
// backend-fresh-db job) + INTEGRATION_DB=1.
func TestFreshDB_ClaimNextItem_ReclaimsHalfCommittedOrphans(t *testing.T) {
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
	const job = "cyb3480-job"

	cleanup := func() {
		_ = db.Exec(ctx, `DELETE FROM backfill_items WHERE job_id = $1`, job)
		_ = db.Exec(ctx, `DELETE FROM backfill_jobs WHERE id = $1`, job)
		_ = db.Exec(ctx, `DELETE FROM pipeline_runs WHERE id LIKE 'cyb3480-run-%'`)
		_ = db.Exec(ctx, `DELETE FROM pipeline_templates WHERE id = 'tpl-cyb3480'`)
		_ = db.Exec(ctx, `DELETE FROM execution_targets WHERE id = 'cyb3480-target'`)
	}
	cleanup()
	t.Cleanup(cleanup)

	// FK parents: backfill_jobs.template_id -> pipeline_templates,
	// pipeline_runs.execution_target_id -> execution_targets.
	if err := db.Exec(ctx, `
INSERT INTO pipeline_templates(id, name, pipeline) VALUES('tpl-cyb3480', 'nw-delivery-dev', '{}')`); err != nil {
		t.Fatalf("seed template: %v", err)
	}
	if err := db.Exec(ctx, `
INSERT INTO execution_targets(id, name, cluster, namespace)
VALUES('cyb3480-target', 'video-proc-prod', 'default', 'video-proc-prod')`); err != nil {
		t.Fatalf("seed execution target: %v", err)
	}

	if err := db.Exec(ctx, `
INSERT INTO backfill_jobs(id, name, template_id, status, total_count)
VALUES($1, 'cyb3480', 'tpl-cyb3480', 'running', 4)`, job); err != nil {
		t.Fatalf("seed job: %v", err)
	}

	// Four pipeline runs covering every branch of the claim predicate.
	//   pending: real run (UID set) backing a normal pending item
	//   orphan : placeholder (no UID, `-batch-` name) created long ago -> stuck
	//   real   : genuinely running (UID set) -> must NOT be reclaimed
	//   fresh  : placeholder created just now -> in-flight deploy, must NOT be reclaimed
	runs := []struct {
		id, wf, uid, age string
	}{
		{"cyb3480-run-pending", "cyb3480-pending-uuid", "uid-pending", "-1 hours"},
		{"cyb3480-run-orphan", "nw-delivery-dev-batch-orphan", "", "-1 hours"},
		{"cyb3480-run-real", "cyb3480-real-uuid", "uid-real", "-1 hours"},
		{"cyb3480-run-fresh", "nw-delivery-dev-batch-fresh", "", "-10 seconds"},
	}
	for _, r := range runs {
		if err := db.Exec(ctx, `
INSERT INTO pipeline_runs(id, pipeline_name, workflow_name, execution_target_id, argo_namespace, argo_workflow_uid, created_at)
VALUES($1, 'nw-delivery-dev', $2, 'cyb3480-target', 'video-proc-prod', $3, NOW() + $4::interval)`,
			r.id, r.wf, r.uid, r.age); err != nil {
			t.Fatalf("seed run %s: %v", r.id, err)
		}
	}

	// Items. started_at is set recent for the orphan on purpose, to prove the
	// predicate keys staleness on the run's created_at (stable) rather than the
	// item's started_at (which the reaper<->sync oscillation keeps refreshing).
	items := []struct {
		id, asset, status, runID, wf string
	}{
		{"cyb3480-item-pending", "asset-p", "pending", "cyb3480-run-pending", "cyb3480-pending-uuid"},
		{"cyb3480-item-orphan", "asset-o", "running", "cyb3480-run-orphan", "nw-delivery-dev-batch-orphan"},
		{"cyb3480-item-real", "asset-r", "running", "cyb3480-run-real", "cyb3480-real-uuid"},
		{"cyb3480-item-fresh", "asset-f", "running", "cyb3480-run-fresh", "nw-delivery-dev-batch-fresh"},
	}
	for _, it := range items {
		if err := db.Exec(ctx, `
INSERT INTO backfill_items(id, job_id, asset_id, status, pipeline_run_id, workflow_name, started_at, created_at)
VALUES($1, $2, $3, $4, $5, $6, NOW(), NOW() - INTERVAL '1 hour')`,
			it.id, job, it.asset, it.status, it.runID, it.wf); err != nil {
			t.Fatalf("seed item %s: %v", it.id, err)
		}
	}

	repo := NewBackfillRepo(client)

	// Claim 1: pending sorts first (status='pending' DESC).
	c1, err := repo.ClaimNextItem(ctx, job)
	if err != nil || c1 == nil {
		t.Fatalf("claim 1: err=%v item=%v", err, c1)
	}
	if c1.ID != "cyb3480-item-pending" {
		t.Fatalf("claim 1: got %q, want cyb3480-item-pending", c1.ID)
	}

	// Claim 2: the pending item is now running on a real run (UID set) so it is
	// no longer claimable; the orphan is the only remaining claimable item.
	c2, err := repo.ClaimNextItem(ctx, job)
	if err != nil || c2 == nil {
		t.Fatalf("claim 2: err=%v item=%v", err, c2)
	}
	if c2.ID != "cyb3480-item-orphan" {
		t.Fatalf("claim 2: got %q, want cyb3480-item-orphan (orphan not reclaimed)", c2.ID)
	}

	// Simulate the deploy completing: the orphan's run gets an Argo UID.
	if err := db.Exec(ctx, `UPDATE pipeline_runs SET argo_workflow_uid = 'uid-deployed' WHERE id = 'cyb3480-run-orphan'`); err != nil {
		t.Fatalf("simulate deploy: %v", err)
	}

	// Claim 3: nothing claimable — real is excluded by its UID, fresh by its
	// young run created_at, and the orphan by its now-assigned UID.
	c3, err := repo.ClaimNextItem(ctx, job)
	if err != nil {
		t.Fatalf("claim 3: err=%v", err)
	}
	if c3 != nil {
		t.Fatalf("claim 3: got %q, want nil (genuine-running / fresh-deploy must not be reclaimed)", c3.ID)
	}
}
