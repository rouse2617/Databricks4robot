package pipeline

import (
	"context"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

func seedPipelineOutputVersion(ctx context.Context, laRepo repository.LogicalAssetRepository, a *models.Asset) error {
	if laRepo == nil {
		return nil
	}
	a.LogicalAssetID = a.AssetID
	a.Revision = 1
	a.IsCurrent = true
	la := &models.LogicalAsset{
		LogicalAssetID:  a.LogicalAssetID,
		AssetType:       a.AssetType,
		Owner:           a.Owner,
		CurrentRevision: 1,
		TotalRevisions:  1,
	}
	return laRepo.Insert(ctx, la)
}
