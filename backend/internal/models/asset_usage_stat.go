package models

import "time"

// AssetUsageStat tracks per-asset engagement counters (CYB-1094).
type AssetUsageStat struct {
	ID             int64      `json:"id"`
	AssetID        string     `json:"asset_id"`
	LogicalAssetID string     `json:"logical_asset_id,omitempty"`
	ViewCount      int        `json:"view_count"`
	LastViewedAt   *time.Time `json:"last_viewed_at,omitempty"`
	FavoriteCount  int        `json:"favorite_count"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}
