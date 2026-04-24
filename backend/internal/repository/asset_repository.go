package repository

import (
	"context"

	"data-platform/internal/models"
)

// AssetRepository defines persistence operations required by asset usecases.
// This interface is intentionally storage-agnostic to decouple usecases from Bigtable.
type AssetRepository interface {
	Get(ctx context.Context, assetID string) (*models.Asset, error)
	Set(ctx context.Context, a *models.Asset) error
	SoftDelete(ctx context.Context, assetID string) error
	ListByMcapFile(ctx context.Context, mcapFileID string) ([]*models.Asset, error)
	WriteSegmentIndex(ctx context.Context, a *models.Asset) error
}

