package repository

import (
	"context"
	"errors"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

var ErrLogicalAssetNotFound = errors.New("logical asset not found")

// LogicalAssetRepository persists logical_assets coordinator rows.
type LogicalAssetRepository interface {
	Get(ctx context.Context, logicalAssetID string) (*models.LogicalAsset, error)
	Insert(ctx context.Context, la *models.LogicalAsset) error
	BumpRevision(ctx context.Context, logicalAssetID string, newRevision int64) error
	MaxRevision(ctx context.Context, logicalAssetID string) (int64, error)
	CurrentAssetID(ctx context.Context, logicalAssetID string) (string, error)
	ClearCurrentForLogical(ctx context.Context, logicalAssetID string) error
}
