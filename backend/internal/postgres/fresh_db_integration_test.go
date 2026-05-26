//go:build integration

package postgres

import (
	"context"
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

	var hasStatus bool
	err = client.db.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1 FROM information_schema.columns
  WHERE table_schema = 'public' AND table_name = 'assets' AND column_name = 'status'
)`).Scan(&hasStatus)
	if err != nil {
		t.Fatalf("check assets.status column: %v", err)
	}
	if hasStatus {
		t.Fatal("assets.status column still exists; migration 025 must run before this test")
	}

	assetID, err := id.GenerateAssetID()
	if err != nil {
		t.Fatalf("GenerateAssetID: %v", err)
	}
	mcapFileID, err := id.GenerateMcapFileID()
	if err != nil {
		t.Fatalf("GenerateMcapFileID: %v", err)
	}

	// After migration 038 (fk_mcap_asset), mcap_files is a 1:1 extension of
	// raw_mcap assets: mcap_file_id must reference an existing asset_id.
	// Create a placeholder raw_mcap asset first, then the mcap_file row.
	repo := NewAssetRepo(client)
	placeholder := &models.Asset{
		AssetID:          mcapFileID,
		McapFileID:       mcapFileID,
		StartTimestampNs: 1,
		EndTimestampNs:   2,
		LifecycleState:   string(LifecycleReady),
		AssetType:        "raw_mcap",
		Owner:            "smoke",
	}
	if err := repo.InsertNew(ctx, placeholder); err != nil {
		t.Fatalf("InsertNew placeholder raw_mcap: %v", err)
	}

	mcapRepo := NewMcapFileRepo(client)
	if err := mcapRepo.Set(ctx, &models.McapFile{
		McapFileID:  mcapFileID,
		GCSPath:     "gs://smoke/test.mcap",
		IngestState: models.IngestStatePending,
		Owner:       "smoke",
	}); err != nil {
		t.Fatalf("mcap Set: %v", err)
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
