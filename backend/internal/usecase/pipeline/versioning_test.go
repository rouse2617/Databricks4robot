package pipeline

import (
	"context"
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

func TestSeedPipelineOutputVersion(t *testing.T) {
	ctx := context.Background()
	laRepo := &mockLogicalAssetRepo{}
	a := &models.Asset{AssetID: "out-1", AssetType: "dataset"}
	if err := seedPipelineOutputVersion(ctx, laRepo, a); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if a.LogicalAssetID != "out-1" || a.Revision != 1 || !a.IsCurrent {
		t.Fatalf("unexpected asset version fields: %#v", a)
	}
	if len(laRepo.inserted) != 1 || laRepo.inserted[0].LogicalAssetID != "out-1" {
		t.Fatalf("logical asset not inserted: %#v", laRepo.inserted)
	}
}
