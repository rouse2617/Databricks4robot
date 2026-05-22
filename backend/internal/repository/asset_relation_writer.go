package repository

import "context"

// AssetRelationWriter inserts lineage edges (optional capability on asset store).
type AssetRelationWriter interface {
	InsertRevisionOf(ctx context.Context, newAssetID, priorAssetID, runID string) error
}
