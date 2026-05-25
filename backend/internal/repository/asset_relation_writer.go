package repository

import "context"

// AssetRelationWriter inserts lineage edges (optional capability on asset store).
type AssetRelationWriter interface {
	InsertRevisionOf(ctx context.Context, newAssetID, priorAssetID, runID string) error
	// InsertRelation inserts a generic asset_relations edge.
	InsertRelation(ctx context.Context, parentAssetID, childAssetID, relationType, runID string) error
}
