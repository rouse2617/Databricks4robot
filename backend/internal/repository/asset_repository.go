package repository

import (
	"context"
	"errors"

	"github.com/CyberOrigin2077/cyber-databrew/internal/filter"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// ErrDuplicateAssetID is returned when inserting an asset whose asset_id already exists.
var ErrDuplicateAssetID = errors.New("duplicate asset id")

// AssetRepository defines persistence operations required by asset usecases.
// Storage-agnostic by design — concrete implementations live in
// internal/postgres.
//
// NOTE: Per-algorithm state (status, run_id, output_uri, finished_at, ...)
// is owned by AssetAlgoLatestRepository, NOT this repo. The previous
// MergeCfAlgo method on this interface implemented optimistic locking on
// `assets.version` for algo writes; that path was retired in the Issue-2
// fix (see data-platform-design.md §5.3.1) because contention on
// assets.version was the dominant write hotspot. Any algo-state mutation
// must now go through AssetAlgoLatestRepository.Upsert + AssetEventRepository.Append
// inside Client.WithTx.
type AssetRepository interface {
	Get(ctx context.Context, assetID string) (*models.Asset, error)
	// GetAll returns an asset regardless of soft-delete status.
	// Used by GET /assets/:id so soft-deleted assets remain accessible
	// per the API contract ("soft-deleted assets can still be retrieved").
	GetAll(ctx context.Context, assetID string) (*models.Asset, error)
	// InsertNew inserts a new asset row. It must not update an existing row.
	// Returns ErrDuplicateAssetID when asset_id is already taken.
	InsertNew(ctx context.Context, a *models.Asset) error
	Set(ctx context.Context, a *models.Asset) error
	SoftDelete(ctx context.Context, assetID string) error
	ListByMcapFile(ctx context.Context, mcapFileID string) ([]*models.Asset, error)
	WriteSegmentIndex(ctx context.Context, a *models.Asset) error

	// ListWithFilters queries assets using a parameterized WHERE clause.
	// whereSQL may be empty (no additional filters beyond is_deleted=FALSE).
	// Returns matching assets and total count for pagination.
	ListWithFilters(ctx context.Context, whereSQL string, args []interface{},
		page, pageSize int, orderBy filter.OrderByClause) ([]*models.Asset, int64, error)
}
