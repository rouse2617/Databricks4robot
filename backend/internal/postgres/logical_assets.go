package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

type LogicalAssetRepo struct {
	c *Client
}

var _ repository.LogicalAssetRepository = (*LogicalAssetRepo)(nil)

func NewLogicalAssetRepo(c *Client) *LogicalAssetRepo {
	return &LogicalAssetRepo{c: c}
}

func (r *LogicalAssetRepo) Get(ctx context.Context, logicalAssetID string) (*models.LogicalAsset, error) {
	const q = `
SELECT logical_asset_id, asset_type, COALESCE(display_name, ''), COALESCE(description, ''),
  COALESCE(owner, ''), status, current_revision, total_revisions,
  metadata, created_at, updated_at
FROM logical_assets
WHERE logical_asset_id = $1`
	var (
		la            models.LogicalAsset
		metadataBytes []byte
	)
	err := r.c.db.QueryRow(ctx, q, logicalAssetID).Scan(
		&la.LogicalAssetID, &la.AssetType, &la.DisplayName, &la.Description,
		&la.Owner, &la.Status, &la.CurrentRevision, &la.TotalRevisions,
		&metadataBytes, &la.CreatedAt, &la.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, errNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres LogicalAssetRepo.Get: %w", err)
	}
	if len(metadataBytes) > 0 {
		_ = json.Unmarshal(metadataBytes, &la.Metadata)
	}
	return &la, nil
}

func (r *LogicalAssetRepo) Insert(ctx context.Context, la *models.LogicalAsset) error {
	now := time.Now().UTC()
	if la.CreatedAt.IsZero() {
		la.CreatedAt = now
	}
	la.UpdatedAt = now
	if la.Status == "" {
		la.Status = "active"
	}
	if la.Metadata == nil {
		la.Metadata = map[string]interface{}{}
	}
	meta, _ := json.Marshal(la.Metadata)
	const q = `
INSERT INTO logical_assets(
  logical_asset_id, asset_type, display_name, description, owner, status,
  current_revision, total_revisions, metadata, created_at, updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb,$10,$11)`
	return r.c.db.Exec(ctx, q,
		la.LogicalAssetID, la.AssetType, nullIfEmpty(la.DisplayName), nullIfEmpty(la.Description),
		nullIfEmpty(la.Owner), la.Status, la.CurrentRevision, la.TotalRevisions, meta,
		la.CreatedAt, la.UpdatedAt,
	)
}

func (r *LogicalAssetRepo) BumpRevision(ctx context.Context, logicalAssetID string, newRevision int64) error {
	const q = `
UPDATE logical_assets SET
  current_revision = $2,
  total_revisions = GREATEST(total_revisions + 1, $2),
  updated_at = now()
WHERE logical_asset_id = $1`
	n, err := r.c.db.ExecResult(ctx, q, logicalAssetID, newRevision)
	if err != nil {
		return fmt.Errorf("postgres LogicalAssetRepo.BumpRevision: %w", err)
	}
	if n == 0 {
		return repository.ErrLogicalAssetNotFound
	}
	return nil
}

func (r *LogicalAssetRepo) MaxRevision(ctx context.Context, logicalAssetID string) (int64, error) {
	const q = `SELECT COALESCE(MAX(revision), 0) FROM assets WHERE logical_asset_id = $1 AND is_deleted = FALSE`
	var max int64
	if err := r.c.db.QueryRow(ctx, q, logicalAssetID).Scan(&max); err != nil {
		return 0, fmt.Errorf("postgres LogicalAssetRepo.MaxRevision: %w", err)
	}
	return max, nil
}

func (r *LogicalAssetRepo) CurrentAssetID(ctx context.Context, logicalAssetID string) (string, error) {
	const q = `
SELECT asset_id FROM assets
WHERE logical_asset_id = $1 AND is_current = TRUE AND is_deleted = FALSE
LIMIT 1`
	var id string
	err := r.c.db.QueryRow(ctx, q, logicalAssetID).Scan(&id)
	if err != nil {
		if errors.Is(err, errNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("postgres LogicalAssetRepo.CurrentAssetID: %w", err)
	}
	return id, nil
}

func (r *LogicalAssetRepo) ClearCurrentForLogical(ctx context.Context, logicalAssetID string) error {
	const q = `
UPDATE assets SET is_current = FALSE, updated_at = now()
WHERE logical_asset_id = $1 AND is_current = TRUE AND is_deleted = FALSE`
	return r.c.db.Exec(ctx, q, logicalAssetID)
}

func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
