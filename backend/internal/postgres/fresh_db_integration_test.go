//go:build integration

package postgres

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/config"
	"github.com/CyberOrigin2077/cyber-databrew/internal/id"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// TestFreshDB_AssetCRUD expects an empty database with all migrations already applied
// (see .github/workflows/test-integration.yml backend-fresh-db job).
func TestFreshDB_AssetCRUD(t *testing.T) {
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

	// Note: this test previously gated on `assets.status` being dropped by
	// pre-baseline migration 025. The 2026-07-08 Atlas baseline reset
	// (backend/migrations/20260708104125_baseline_from_dev.sql) snapshotted
	// the live dev schema, which still had the column, so the gate could
	// never pass again. models.Asset.Status is already derived from
	// lifecycle_state (see asset.go: "API-only; derived from lifecycle_state
	// (not stored in PostgreSQL)"), so the DB column's presence is
	// irrelevant to correctness — the CRUD + derived-status assertions
	// below still cover the meaningful behavior. (CYB-3381)

	assetID, err := id.GenerateAssetID()
	if err != nil {
		t.Fatalf("GenerateAssetID: %v", err)
	}
	mcapFileID, err := id.GenerateMcapFileID()
	if err != nil {
		t.Fatalf("GenerateMcapFileID: %v", err)
	}

	// Migration 038 introduced dual FKs creating a circular dependency:
	//   fk_mcap_asset:  mcap_files → assets (DEFERRABLE INITIALLY DEFERRED)
	//   fk_assets_mcap: assets      → mcap_files (NOT deferred)
	//
	// McapFileRepo.Set does not use dbFromCtx (runs outside any caller
	// transaction), so we insert via raw SQL within a tx instead. The
	// deferred FK is checked only at commit time, after both rows exist.
	repo := NewAssetRepo(client)
	if err := client.WithTx(ctx, func(txCtx context.Context) error {
		db := dbFromCtx(txCtx, client.db)
		if err := db.Exec(txCtx, `
INSERT INTO mcap_files(mcap_file_id, mcap_uri, ingest_state, owner, created_at, updated_at)
VALUES($1, $2, $3, $4, NOW(), NOW())`, mcapFileID, "gs://smoke/test.mcap", string(models.IngestStatePending), "smoke"); err != nil {
			return fmt.Errorf("insert mcap_file: %w", err)
		}
		if err := repo.InsertNew(txCtx, &models.Asset{
			AssetID:          mcapFileID,
			McapFileID:       mcapFileID,
			StartTimestampNs: 1,
			EndTimestampNs:   2,
			LifecycleState:   string(LifecycleReady),
			AssetType:        "raw_mcap",
			Owner:            "smoke",
		}); err != nil {
			return fmt.Errorf("insert placeholder asset: %w", err)
		}
		return nil
	}); err != nil {
		t.Fatalf("seed mcap_file + placeholder asset: %v", err)
	}

	a := &models.Asset{
		AssetID:          assetID,
		McapFileID:       mcapFileID,
		StartTimestampNs: 1,
		EndTimestampNs:   2,
		LifecycleState:   string(LifecycleReady),
		AssetType:        "segment",
		Owner:            "smoke",
	}
	if err := repo.InsertNew(ctx, a); err != nil {
		t.Fatalf("InsertNew: %v", err)
	}

	got, err := repo.Get(ctx, assetID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("Get returned nil asset")
	}
	if got.LifecycleState != string(LifecycleReady) {
		t.Fatalf("lifecycle_state: got %q want %q", got.LifecycleState, LifecycleReady)
	}
	if got.Status != models.AssetStatusApproved {
		t.Fatalf("derived status: got %q want approved", got.Status)
	}

	got.LifecycleState = string(LifecycleRejected)
	if err := repo.Set(ctx, got); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got2, err := repo.Get(ctx, assetID)
	if err != nil || got2 == nil {
		t.Fatalf("Get after update: %v", err)
	}
	if got2.LifecycleState != string(LifecycleRejected) {
		t.Fatalf("lifecycle after update: got %q", got2.LifecycleState)
	}

	if err := repo.SoftDelete(ctx, assetID); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}
}
