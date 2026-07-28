package repository

import (
	"context"
	"errors"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/filter"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// DurationRow is a lightweight three-column projection used by the batch
// duration-lookup endpoint (CYB-4294). Duplicated in the postgres package
// so the concrete repo can return the same value type; kept here as the
// repository-facing contract.
type DurationRow struct {
	AssetID      string
	GraceVideoID string
	DurationMs   int64
}

// AssetCostRow is one aggregated cost/GPU/CPU row per (asset_id[, algo_key])
// returned by AssetRepository.LookupCosts. AlgoKey is empty when the caller
// aggregates over asset only (byAlgo=false); populated with the Argo
// pipeline_run_nodes.template_name otherwise. Used by the batch cost-lookup
// endpoint (CYB-4306).
type AssetCostRow struct {
	AssetID      string
	AlgoKey      string
	TotalCostUSD float64
	GPUSec       float64
	CPUSec       float64
	RunCount     int64
}

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
	// FindExistingIDs returns the subset of assetIDs that exist and are not soft-deleted.
	FindExistingIDs(ctx context.Context, assetIDs []string) (map[string]struct{}, error)
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
	// ListByLogicalAssetID returns non-deleted revisions for a logical asset family.
	ListByLogicalAssetID(ctx context.Context, logicalAssetID string) ([]*models.Asset, error)
	WriteSegmentIndex(ctx context.Context, a *models.Asset) error

	// ListWithFilters queries assets using a parameterized WHERE clause.
	// whereSQL may be empty (no additional filters beyond is_deleted=FALSE).
	// Returns matching assets and total count for pagination.
	ListWithFilters(ctx context.Context, whereSQL string, args []interface{},
		page, pageSize int, orderBy filter.OrderByClause) ([]*models.Asset, int64, error)

	// ListDescendants returns all non-deleted descendant assets reachable via
	// asset_relations parent→child edges (recursive CTE). Used by the tag
	// propagator (CYB-1068) to apply tags to the full descendant tree.
	ListDescendants(ctx context.Context, assetID string) ([]*models.Asset, error)

	// LookupDurations returns duration_ms for non-deleted assets whose
	// asset_id OR grace_video_id matches any element of ids. Optional
	// [minMs, maxMs] bounds filter the resulting rows in-SQL (0 disables the
	// bound). Empty ids returns (nil, nil) without a round-trip. Used by the
	// batch duration-lookup endpoint (CYB-4294).
	LookupDurations(ctx context.Context, ids []string, minMs, maxMs int64) ([]DurationRow, error)

	// LookupCosts aggregates leaf-pod costs / GPU-sec / CPU-sec / run_count
	// per asset_id (and optionally per algo_key = template_name) over the
	// [startAt, endAt] window on pipeline_runs.finished_at. assetIDs must
	// already be resolved (this is the second query in the two-step batch
	// cost pipeline; the caller resolves grace_video_id → asset_id first).
	// Uses the same leaf-pod predicate as the existing single-run cost path
	// (pipeline_repo.go:767) to avoid double-counting rollup nodes.
	// CYB-4306.
	LookupCosts(ctx context.Context, assetIDs []string, startAt, endAt time.Time, byAlgo bool) ([]AssetCostRow, error)
}
