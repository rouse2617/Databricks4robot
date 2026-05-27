package repository

import "context"

type AssetLineageProjection struct {
	UpstreamIDs   []string
	DownstreamIDs []string
	RelationTypes []string
}

type AssetLineageRepository interface {
	GetLineageProjection(ctx context.Context, assetID string) (*AssetLineageProjection, error)
}
