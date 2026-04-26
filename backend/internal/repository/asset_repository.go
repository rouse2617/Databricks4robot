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

	// ListWithFilters queries assets using a parameterized WHERE clause.
	// whereSQL may be empty (no additional filters beyond is_deleted=FALSE).
	// Returns matching assets and total count for pagination.
	ListWithFilters(ctx context.Context, whereSQL string, args []interface{},
		page, pageSize int, orderBy string) ([]*models.Asset, int64, error)

	// MergeCfAlgo atomically merges cf_algo and cf_files JSONB fields using
	// optimistic locking. Returns the new version number.
	// If expectedVersion does not match, returns ErrOptimisticLock.
	MergeCfAlgo(ctx context.Context, assetID string, expectedVersion int64,
		algoKV map[string]interface{}, filesKV map[string]interface{}) (int64, error)
}

