package repository

import "context"

// AssetRelationWriter inserts lineage edges (optional capability on asset store).
type AssetRelationWriter interface {
	InsertRevisionOf(ctx context.Context, newAssetID, priorAssetID, runID string) error
	// InsertRelation inserts a generic asset_relations edge (empty metadata).
	InsertRelation(ctx context.Context, parentAssetID, childAssetID, relationType, runID string) error
	// InsertRelationWithMetadata inserts a generic edge carrying caller-context
	// metadata (split_method / run_id / deployment_id …) into
	// asset_relations.metadata (CYB-3281). metadata may be nil.
	InsertRelationWithMetadata(ctx context.Context, parentAssetID, childAssetID, relationType, runID string, metadata map[string]any) error
}
