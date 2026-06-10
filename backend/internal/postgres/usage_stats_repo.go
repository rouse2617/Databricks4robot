package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

// UsageStatsRepo implements repository.AssetUsageStatRepository against
// the asset_usage_stats table (CYB-1094/1095/1096).
type UsageStatsRepo struct {
	c *Client
}

var _ repository.AssetUsageStatRepository = (*UsageStatsRepo)(nil)

func NewUsageStatsRepo(c *Client) *UsageStatsRepo { return &UsageStatsRepo{c: c} }

// RecordView increments view_count and sets last_viewed_at for the given asset.
// Creates the row if it does not yet exist (CYB-1095).
func (r *UsageStatsRepo) RecordView(ctx context.Context, assetID string) error {
	const q = `
INSERT INTO asset_usage_stats (asset_id, view_count, last_viewed_at, created_at, updated_at)
VALUES ($1, 1, now(), now(), now())
ON CONFLICT (asset_id) DO UPDATE SET
    view_count    = asset_usage_stats.view_count + 1,
    last_viewed_at = now(),
    updated_at     = now()`
	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, q, assetID); err != nil {
		return fmt.Errorf("postgres UsageStatsRepo.RecordView: %w", err)
	}
	return nil
}

// ToggleFavorite flips the favorite state: if no row exists or favorite_count=0
// it increments to 1; if favorite_count >= 1 it decrements to 0.
// Returns the new favorite_count (CYB-1096).
func (r *UsageStatsRepo) ToggleFavorite(ctx context.Context, assetID string) (int, error) {
	// First try to get the current state.
	const getQ = `SELECT COALESCE(favorite_count, 0) FROM asset_usage_stats WHERE asset_id = $1`
	db := dbFromCtx(ctx, r.c.db)
	var current int
	err := db.QueryRow(ctx, getQ, assetID).Scan(&current)
	if err != nil && !errors.Is(err, errNoRows) {
		return 0, fmt.Errorf("postgres UsageStatsRepo.ToggleFavorite get: %w", err)
	}

	var newCount int
	if current >= 1 {
		newCount = 0
	} else {
		newCount = 1
	}

	const upsertQ = `
INSERT INTO asset_usage_stats (asset_id, favorite_count, created_at, updated_at)
VALUES ($1, $2, now(), now())
ON CONFLICT (asset_id) DO UPDATE SET
    favorite_count = $2,
    updated_at     = now()`
	if err := db.Exec(ctx, upsertQ, assetID, newCount); err != nil {
		return 0, fmt.Errorf("postgres UsageStatsRepo.ToggleFavorite upsert: %w", err)
	}
	return newCount, nil
}

// GetByAsset returns the usage stats row for the given asset, or nil if none exists.
func (r *UsageStatsRepo) GetByAsset(ctx context.Context, assetID string) (*models.AssetUsageStat, error) {
	const q = `
SELECT id, asset_id, COALESCE(logical_asset_id, ''),
  COALESCE(view_count, 0), last_viewed_at,
  COALESCE(favorite_count, 0), created_at, updated_at
FROM asset_usage_stats
WHERE asset_id = $1`
	db := dbFromCtx(ctx, r.c.db)
	var s models.AssetUsageStat
	err := db.QueryRow(ctx, q, assetID).Scan(
		&s.ID, &s.AssetID, &s.LogicalAssetID,
		&s.ViewCount, &s.LastViewedAt,
		&s.FavoriteCount, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, errNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres UsageStatsRepo.GetByAsset: %w", err)
	}
	return &s, nil
}
