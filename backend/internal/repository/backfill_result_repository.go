package repository

import (
	"context"
	"encoding/json"
	"errors"
)

var ErrReportManifestNotFound = errors.New("report manifest not found")

// ReportManifest describes a registered backfill report identity.
type ReportManifest struct {
	ReportID   string
	Version    string
	SchemaJSON json.RawMessage
}

// AlgoRunResultWriteInput upserts a staged algorithm result row.
type AlgoRunResultWriteInput struct {
	AssetID       string
	AlgoKey       string
	Version       string
	ReportID      string
	ResultPayload map[string]any
}

// BackfillResultRepository persists backfill result staging rows and manifest lookups.
type BackfillResultRepository interface {
	GetReportManifest(ctx context.Context, reportID string) (*ReportManifest, error)
	UpsertAlgoRunResult(ctx context.Context, in AlgoRunResultWriteInput) error
	HasAlgoRunResult(ctx context.Context, assetID, algoKey, version string) (bool, error)
}
