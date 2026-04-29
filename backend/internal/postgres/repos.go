package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"data-platform/internal/models"
	"data-platform/internal/repository"
)

type AssetRepo struct {
	c *Client
}

var _ repository.AssetRepository = (*AssetRepo)(nil)

func NewAssetRepo(c *Client) *AssetRepo { return &AssetRepo{c: c} }

func (r *AssetRepo) Get(ctx context.Context, assetID string) (*models.Asset, error) {
	const q = `
SELECT asset_id, mcap_file_id, start_timestamp_ns, end_timestamp_ns, segment_locator,
  status, lifecycle_state, asset_type, duration_ms,
  owner, reviewer, delivery_count, last_delivered_at, last_delivered_to,
  retention_tier, expire_at, storage_uri, thumb_uri, asset_level,
  created_at, updated_at, version
FROM assets
WHERE asset_id = $1 AND is_deleted = FALSE`
	var (
		a              models.Asset
		status         string
		lifecycleState string
		segLoc         *string
	)
	err := r.c.db.QueryRow(ctx, q, assetID).Scan(
		&a.AssetID, &a.McapFileID, &a.StartTimestampNs, &a.EndTimestampNs, &segLoc,
		&status, &lifecycleState, &a.AssetType, &a.DurationMs,
		&a.Owner, &a.Reviewer, &a.DeliveryCount, &a.LastDeliveredAt, &a.LastDeliveredTo,
		&a.RetentionTier, &a.ExpireAt, &a.StorageURI, &a.ThumbURI, &a.AssetLevel,
		&a.CreatedAt, &a.UpdatedAt, &a.Version,
	)
	if err != nil {
		if errors.Is(err, errNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres AssetRepo.Get: %w", err)
	}
	a.Status = models.AssetStatus(status)
	a.LifecycleState = lifecycleState
	if segLoc != nil {
		a.SegmentLocator = *segLoc
	}
	a.SyncLegacyFields()
	return &a, nil
}

// Set performs an upsert with optimistic concurrency control on the asset
// version. The caller must populate a.Version with the version it observed
// (typically from a prior Get). The version is bumped before write; the DO
// UPDATE clause guards on the previous version, so concurrent writers race
// safely:
//
//   - Insert (no conflict): always succeeds.
//   - Update (conflict): succeeds only when the existing row's version equals
//     the caller's expected version. Otherwise repository.ErrOptimisticLock.
func (r *AssetRepo) Set(ctx context.Context, a *models.Asset) error {
	now := time.Now().UTC()
	if a.CreatedAt.IsZero() {
		a.CreatedAt = now
	}
	a.UpdatedAt = now
	a.Version++
	a.SegmentLocator = models.ComputeSegmentLocator(a.McapFileID, a.StartTimestampNs, a.EndTimestampNs)

	// Sync lifecycle_state ↔ status bidirectionally.
	if a.LifecycleState != "" {
		a.Status = models.AssetStatus(LifecycleStateToStatus(a.LifecycleState))
	} else if a.Status != "" {
		a.LifecycleState = StatusToLifecycleState(string(a.Status))
	}
	// Sync legacy ↔ new typed fields for consistency.
	if a.AssetType != "" && a.SegType == "" {
		a.SegType = a.AssetType
	} else if a.SegType != "" && a.AssetType == "" {
		a.AssetType = a.SegType
	}
	if a.DurationMs != 0 && a.DurationSec == 0 {
		a.DurationSec = float64(a.DurationMs) / 1000.0
	} else if a.DurationSec != 0 && a.DurationMs == 0 {
		a.DurationMs = int64(a.DurationSec * 1000)
	}

	const q = `
INSERT INTO assets(
  asset_id, mcap_file_id, start_timestamp_ns, end_timestamp_ns, segment_locator,
  status, lifecycle_state, asset_type, duration_ms,
  owner, reviewer, delivery_count, last_delivered_at, last_delivered_to,
  retention_tier, expire_at, storage_uri, thumb_uri, asset_level,
  parent_asset_id, root_asset_id, metadata, files,
  tenant_id, project_id,
  is_deleted,
  created_at, updated_at, version
) VALUES (
  $1,$2,$3,$4,$5,
  $6,$7,$8,$9,
  $10,$11,$12,$13,$14,
  $15,$16,$17,$18,$19,
  $20,$21,$22::jsonb,$23::jsonb,
  $24,$25,
  FALSE,
  $26,$27,$28
)
ON CONFLICT (asset_id) DO UPDATE SET
  mcap_file_id=EXCLUDED.mcap_file_id,
  start_timestamp_ns=EXCLUDED.start_timestamp_ns,
  end_timestamp_ns=EXCLUDED.end_timestamp_ns,
  segment_locator=EXCLUDED.segment_locator,
  status=EXCLUDED.status,
  lifecycle_state=EXCLUDED.lifecycle_state,
  asset_type=EXCLUDED.asset_type,
  duration_ms=EXCLUDED.duration_ms,
  owner=EXCLUDED.owner,
  reviewer=EXCLUDED.reviewer,
  delivery_count=EXCLUDED.delivery_count,
  last_delivered_at=EXCLUDED.last_delivered_at,
  last_delivered_to=EXCLUDED.last_delivered_to,
  retention_tier=EXCLUDED.retention_tier,
  expire_at=EXCLUDED.expire_at,
  storage_uri=EXCLUDED.storage_uri,
  thumb_uri=EXCLUDED.thumb_uri,
  asset_level=EXCLUDED.asset_level,
  parent_asset_id=EXCLUDED.parent_asset_id,
  root_asset_id=EXCLUDED.root_asset_id,
  metadata=EXCLUDED.metadata,
  files=EXCLUDED.files,
  tenant_id=EXCLUDED.tenant_id,
  project_id=EXCLUDED.project_id,
  updated_at=EXCLUDED.updated_at,
  version=EXCLUDED.version
WHERE assets.version = EXCLUDED.version - 1`

	// Marshal metadata JSONB (new structured metadata column).
	if a.Metadata == nil {
		a.Metadata = map[string]interface{}{}
	}
	if a.Env != "" {
		a.Metadata["env"] = a.Env
	}
	if a.Task != "" {
		a.Metadata["task"] = a.Task
	}
	for k, v := range a.LifecycleMeta {
		a.Metadata[k] = v
	}
	metadataJSON, _ := json.Marshal(a.Metadata)
	if a.Metadata == nil {
		metadataJSON = []byte(`{}`)
	}
	// Marshal files JSONB (new structured files column).
	if a.FilesJSON == nil && len(a.Files) > 0 {
		a.FilesJSON = make(map[string]interface{}, len(a.Files))
		for k, v := range a.Files {
			a.FilesJSON[k] = v
		}
	}
	filesStructJSON, _ := json.Marshal(a.FilesJSON)
	if a.FilesJSON == nil {
		filesStructJSON = []byte(`{}`)
	}

	// Nullable UUID columns: convert empty string to nil for PG UUID type.
	var parentAssetID, rootAssetID interface{}
	if a.ParentAssetID != "" {
		parentAssetID = a.ParentAssetID
	}
	if a.RootAssetID != "" {
		rootAssetID = a.RootAssetID
	}

	// Nullable text columns.
	var tenantID, projectID interface{}
	if a.TenantID != "" {
		tenantID = a.TenantID
	}
	if a.ProjectID != "" {
		projectID = a.ProjectID
	}

	db := dbFromCtx(ctx, r.c.db)
	rowsAffected, err := db.ExecResult(ctx, q,
		a.AssetID, a.McapFileID, a.StartTimestampNs, a.EndTimestampNs, a.SegmentLocator,
		string(a.Status), a.LifecycleState, a.AssetType, a.DurationMs,
		a.Owner, a.Reviewer, a.DeliveryCount, a.LastDeliveredAt, a.LastDeliveredTo,
		a.RetentionTier, a.ExpireAt, a.StorageURI, a.ThumbURI, a.AssetLevel,
		parentAssetID, rootAssetID, metadataJSON, filesStructJSON,
		tenantID, projectID,
		a.CreatedAt, a.UpdatedAt, a.Version,
	)
	if err != nil {
		return fmt.Errorf("postgres AssetRepo.Set: %w", err)
	}
	if rowsAffected == 0 {
		return repository.ErrOptimisticLock
	}
	return nil
}

func (r *AssetRepo) SoftDelete(ctx context.Context, assetID string) error {
	const q = `UPDATE assets SET is_deleted=TRUE, status='archived', updated_at=now() WHERE asset_id=$1`
	db := dbFromCtx(ctx, r.c.db)
	err := db.Exec(ctx, q, assetID)
	if err != nil {
		return fmt.Errorf("postgres AssetRepo.SoftDelete: %w", err)
	}
	return nil
}

func (r *AssetRepo) ListByMcapFile(ctx context.Context, mcapFileID string) ([]*models.Asset, error) {
	const q = `
SELECT asset_id, mcap_file_id, start_timestamp_ns, end_timestamp_ns, segment_locator,
  status, lifecycle_state, asset_type, duration_ms,
  owner, reviewer, delivery_count, last_delivered_at, last_delivered_to,
  retention_tier, expire_at, storage_uri, thumb_uri, asset_level,
  created_at, updated_at, version
FROM assets
WHERE mcap_file_id = $1 AND is_deleted = FALSE
ORDER BY start_timestamp_ns`
	rows, err := r.c.db.Query(ctx, q, mcapFileID)
	if err != nil {
		return nil, fmt.Errorf("postgres AssetRepo.ListByMcapFile: %w", err)
	}
	defer rows.Close()
	var out []*models.Asset
	for rows.Next() {
		var (
			a              models.Asset
			status         string
			lifecycleState string
			segLoc         *string
		)
		if err := rows.Scan(
			&a.AssetID, &a.McapFileID, &a.StartTimestampNs, &a.EndTimestampNs, &segLoc,
			&status, &lifecycleState, &a.AssetType, &a.DurationMs,
			&a.Owner, &a.Reviewer, &a.DeliveryCount, &a.LastDeliveredAt, &a.LastDeliveredTo,
			&a.RetentionTier, &a.ExpireAt, &a.StorageURI, &a.ThumbURI, &a.AssetLevel,
			&a.CreatedAt, &a.UpdatedAt, &a.Version,
		); err != nil {
			return nil, fmt.Errorf("postgres AssetRepo.ListByMcapFile scan: %w", err)
		}
		a.Status = models.AssetStatus(status)
		a.LifecycleState = lifecycleState
		if segLoc != nil {
			a.SegmentLocator = *segLoc
		}
		a.SyncLegacyFields()
		out = append(out, &a)
	}
	return out, nil
}

// No-op for postgres mode: query path uses table indexes directly.
func (r *AssetRepo) WriteSegmentIndex(ctx context.Context, a *models.Asset) error {
	_ = ctx
	_ = a
	return nil
}

type McapFileRepo struct {
	c *Client
}

var _ repository.McapFileRepository = (*McapFileRepo)(nil)

func NewMcapFileRepo(c *Client) *McapFileRepo { return &McapFileRepo{c: c} }

func (r *McapFileRepo) Get(ctx context.Context, mcapFileID string) (*models.McapFile, error) {
	const q = `
SELECT mcap_file_id, raw_hash_md5,
  mcap_uri, size_bytes, start_timestamp_ns, end_timestamp_ns,
  channel_count, chunk_count, ingest_state, owner, process_state,
  created_at, updated_at, version
FROM mcap_files
WHERE mcap_file_id = $1 AND is_deleted = FALSE`
	var (
		f                 models.McapFile
		ingestState       string
		processStateBytes []byte
	)
	err := r.c.db.QueryRow(ctx, q, mcapFileID).Scan(
		&f.McapFileID, &f.RawHashMD5,
		&f.GCSPath, &f.SizeBytes, &f.StartTimestampNs, &f.EndTimestampNs,
		&f.ChannelCount, &f.ChunkCount, &ingestState, &f.Owner, &processStateBytes,
		&f.CreatedAt, &f.UpdatedAt, &f.Version,
	)
	if err != nil {
		if errors.Is(err, errNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres McapFileRepo.Get: %w", err)
	}
	f.IngestState = models.IngestState(ingestState)
	if f.ProcessState == nil {
		f.ProcessState = map[string]string{}
	}
	_ = json.Unmarshal(processStateBytes, &f.ProcessState)
	return &f, nil
}

func (r *McapFileRepo) Set(ctx context.Context, f *models.McapFile) error {
	now := time.Now().UTC()
	if f.CreatedAt.IsZero() {
		f.CreatedAt = now
	}
	f.UpdatedAt = now
	f.Version++

	// Marshal process_state JSONB (real column).
	processStateJSON, _ := json.Marshal(f.ProcessState)
	if f.ProcessState == nil {
		processStateJSON = []byte(`{}`)
	}

	const q = `
INSERT INTO mcap_files(
  mcap_file_id, raw_hash_md5, is_deleted,
  mcap_uri, size_bytes, file_duration_ms,
  start_timestamp_ns, end_timestamp_ns,
  channel_count, chunk_count, ingest_state,
  owner, process_state,
  created_at, updated_at, version
) VALUES (
  $1,$2,FALSE,
  $3,$4,$5,
  $6,$7,
  $8,$9,$10,
  $11,$12::jsonb,
  $13,$14,$15
)
ON CONFLICT (mcap_file_id) DO UPDATE SET
  raw_hash_md5=EXCLUDED.raw_hash_md5,
  mcap_uri=EXCLUDED.mcap_uri,
  size_bytes=EXCLUDED.size_bytes,
  file_duration_ms=EXCLUDED.file_duration_ms,
  start_timestamp_ns=EXCLUDED.start_timestamp_ns,
  end_timestamp_ns=EXCLUDED.end_timestamp_ns,
  channel_count=EXCLUDED.channel_count,
  chunk_count=EXCLUDED.chunk_count,
  ingest_state=EXCLUDED.ingest_state,
  owner=EXCLUDED.owner,
  process_state=EXCLUDED.process_state,
  updated_at=EXCLUDED.updated_at,
  version=EXCLUDED.version`
	err := r.c.db.Exec(ctx, q,
		f.McapFileID, f.RawHashMD5,
		f.GCSPath, f.SizeBytes, int64(0),
		f.StartTimestampNs, f.EndTimestampNs,
		f.ChannelCount, f.ChunkCount, string(f.IngestState),
		f.Owner, processStateJSON,
		f.CreatedAt, f.UpdatedAt, f.Version,
	)
	if err != nil {
		return fmt.Errorf("postgres McapFileRepo.Set: %w", err)
	}
	return nil
}

func (r *McapFileRepo) List(ctx context.Context, page, pageSize int, ingestState, owner string) ([]*models.McapFile, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	// Build dynamic WHERE clause.
	where := "is_deleted = FALSE"
	args := []any{}
	argIdx := 1

	if ingestState != "" {
		where += fmt.Sprintf(" AND ingest_state = $%d", argIdx)
		args = append(args, ingestState)
		argIdx++
	}
	if owner != "" {
		where += fmt.Sprintf(" AND owner ILIKE $%d", argIdx)
		args = append(args, "%"+owner+"%")
		argIdx++
	}

	var total int64
	countQ := "SELECT COUNT(*) FROM mcap_files WHERE " + where
	err := r.c.db.QueryRow(ctx, countQ, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres McapFileRepo.List count: %w", err)
	}

	selectQ := fmt.Sprintf(`
SELECT mcap_file_id, raw_hash_md5,
  mcap_uri, size_bytes, start_timestamp_ns, end_timestamp_ns,
  channel_count, chunk_count, ingest_state, owner, process_state,
  created_at, updated_at, version
FROM mcap_files
WHERE %s
ORDER BY updated_at DESC
LIMIT $%d OFFSET $%d`, where, argIdx, argIdx+1)
	selectArgs := append(args, pageSize, offset)

	rows, err := r.c.db.Query(ctx, selectQ, selectArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres McapFileRepo.List query: %w", err)
	}
	defer rows.Close()

	var out []*models.McapFile
	for rows.Next() {
		var (
			f                 models.McapFile
			is                string
			processStateBytes []byte
		)
		if err := rows.Scan(
			&f.McapFileID, &f.RawHashMD5,
			&f.GCSPath, &f.SizeBytes, &f.StartTimestampNs, &f.EndTimestampNs,
			&f.ChannelCount, &f.ChunkCount, &is, &f.Owner, &processStateBytes,
			&f.CreatedAt, &f.UpdatedAt, &f.Version,
		); err != nil {
			return nil, 0, fmt.Errorf("postgres McapFileRepo.List scan: %w", err)
		}
		f.IngestState = models.IngestState(is)
		if f.ProcessState == nil {
			f.ProcessState = map[string]string{}
		}
		_ = json.Unmarshal(processStateBytes, &f.ProcessState)
		out = append(out, &f)
	}
	return out, total, nil
}

func (r *McapFileRepo) UpdateIngestState(ctx context.Context, mcapFileID string, state models.IngestState) error {
	const q = `
UPDATE mcap_files
SET ingest_state = $2,
    updated_at = now(),
    version = version + 1
WHERE mcap_file_id = $1`
	err := r.c.db.Exec(ctx, q, mcapFileID, string(state))
	if err != nil {
		return fmt.Errorf("postgres McapFileRepo.UpdateIngestState: %w", err)
	}
	return nil
}

type DeliveryRepo struct {
	c *Client
}

var _ repository.DeliveryRepository = (*DeliveryRepo)(nil)

func NewDeliveryRepo(c *Client) *DeliveryRepo { return &DeliveryRepo{c: c} }

func (r *DeliveryRepo) Set(ctx context.Context, d *models.Delivery) error {
	now := time.Now().UTC()
	if d.CreatedAt.IsZero() {
		d.CreatedAt = now
	}
	d.UpdatedAt = now
	d.Version++

	// Sync legacy AssetCount ↔ ItemCount for backward compat.
	if d.ItemCount == 0 && d.AssetCount > 0 {
		d.ItemCount = int64(d.AssetCount)
	} else if d.AssetCount == 0 && d.ItemCount > 0 {
		d.AssetCount = int(d.ItemCount)
	}
	// Sync Owner ↔ DeliveredBy for backward compat.
	if d.DeliveredBy == "" && d.Owner != "" {
		d.DeliveredBy = d.Owner
	} else if d.Owner == "" && d.DeliveredBy != "" {
		d.Owner = d.DeliveredBy
	}

	// Marshal metadata JSONB (new structured metadata column).
	metadataJSON, _ := json.Marshal(d.Metadata)
	if d.Metadata == nil {
		metadataJSON = []byte(`{}`)
	}

	// Nullable text columns.
	var tenantID, projectID interface{}
	if d.TenantID != "" {
		tenantID = d.TenantID
	}
	if d.ProjectID != "" {
		projectID = d.ProjectID
	}

	const q = `
INSERT INTO deliveries(
  delivery_id, customer_id, status, delivered_at,
  contract_id, delivery_type, requested_by, approved_by, delivered_by,
  manifest_uri, replay_manifest_uri, item_count, total_size_bytes,
  completed_at, metadata, tenant_id, project_id,
  is_deleted,
  created_at, updated_at, version
) VALUES (
  $1,$2,$3,$4,
  $5,$6,$7,$8,$9,
  $10,$11,$12,$13,
  $14,$15::jsonb,$16,$17,
  FALSE,
  $18,$19,$20
)
ON CONFLICT (delivery_id) DO UPDATE SET
  customer_id=EXCLUDED.customer_id,
  status=EXCLUDED.status,
  delivered_at=EXCLUDED.delivered_at,
  contract_id=EXCLUDED.contract_id,
  delivery_type=EXCLUDED.delivery_type,
  requested_by=EXCLUDED.requested_by,
  approved_by=EXCLUDED.approved_by,
  delivered_by=EXCLUDED.delivered_by,
  manifest_uri=EXCLUDED.manifest_uri,
  replay_manifest_uri=EXCLUDED.replay_manifest_uri,
  item_count=EXCLUDED.item_count,
  total_size_bytes=EXCLUDED.total_size_bytes,
  completed_at=EXCLUDED.completed_at,
  metadata=EXCLUDED.metadata,
  tenant_id=EXCLUDED.tenant_id,
  project_id=EXCLUDED.project_id,
  updated_at=EXCLUDED.updated_at,
  version=EXCLUDED.version`
	err := r.c.db.Exec(ctx, q,
		d.DeliveryID, d.CustomerID, string(d.Status), d.DeliveredAt,
		d.ContractID, d.DeliveryType, d.RequestedBy, d.ApprovedBy, d.DeliveredBy,
		d.ManifestURI, d.ReplayManifestURI, d.ItemCount, d.TotalSizeBytes,
		d.CompletedAt, metadataJSON, tenantID, projectID,
		d.CreatedAt, d.UpdatedAt, d.Version,
	)
	if err != nil {
		return fmt.Errorf("postgres DeliveryRepo.Set: %w", err)
	}
	return nil
}

func (r *DeliveryRepo) Get(ctx context.Context, deliveryID string) (*models.Delivery, error) {
	const q = `
SELECT delivery_id, customer_id, status, delivered_at,
  contract_id, delivery_type, requested_by, approved_by, delivered_by,
  manifest_uri, replay_manifest_uri, item_count, total_size_bytes,
  completed_at, metadata, tenant_id, project_id,
  created_at, updated_at, version
FROM deliveries WHERE delivery_id=$1 AND is_deleted=FALSE`
	var (
		d            models.Delivery
		status       string
		metadataJSON []byte
		tenantID     *string
		projectID    *string
	)
	err := r.c.db.QueryRow(ctx, q, deliveryID).Scan(
		&d.DeliveryID, &d.CustomerID, &status, &d.DeliveredAt,
		&d.ContractID, &d.DeliveryType, &d.RequestedBy, &d.ApprovedBy, &d.DeliveredBy,
		&d.ManifestURI, &d.ReplayManifestURI, &d.ItemCount, &d.TotalSizeBytes,
		&d.CompletedAt, &metadataJSON, &tenantID, &projectID,
		&d.CreatedAt, &d.UpdatedAt, &d.Version,
	)
	if err != nil {
		if errors.Is(err, errNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres DeliveryRepo.Get: %w", err)
	}
	d.Status = models.DeliveryStatus(status)
	if tenantID != nil {
		d.TenantID = *tenantID
	}
	if projectID != nil {
		d.ProjectID = *projectID
	}
	// Parse metadata JSONB.
	if len(metadataJSON) > 0 {
		_ = json.Unmarshal(metadataJSON, &d.Metadata)
	}
	// Populate legacy fields from real columns.
	if d.Note == "" {
		if m, ok := d.Metadata["note"].(string); ok {
			d.Note = m
		}
	}
	if d.Owner == "" && d.DeliveredBy != "" {
		d.Owner = d.DeliveredBy
	}
	// Sync legacy AssetCount ↔ ItemCount.
	if d.AssetCount == 0 && d.ItemCount > 0 {
		d.AssetCount = int(d.ItemCount)
	} else if d.ItemCount == 0 && d.AssetCount > 0 {
		d.ItemCount = int64(d.AssetCount)
	}
	// Sync Owner ↔ DeliveredBy.
	if d.DeliveredBy == "" && d.Owner != "" {
		d.DeliveredBy = d.Owner
	}
	return &d, nil
}

func (r *DeliveryRepo) WriteIndexes(ctx context.Context, assetID string, d *models.Delivery) error {
	const q = `
INSERT INTO delivery_items(delivery_id, asset_id, created_at)
VALUES ($1,$2,now())
ON CONFLICT (delivery_id, asset_id) DO NOTHING`
	err := r.c.db.Exec(ctx, q, d.DeliveryID, assetID)
	if err != nil {
		return fmt.Errorf("postgres DeliveryRepo.WriteIndexes: %w", err)
	}
	return nil
}

func (r *DeliveryRepo) List(ctx context.Context, page, pageSize int, status string) ([]*models.Delivery, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	// Count query
	countSQL := "SELECT COUNT(*) FROM deliveries WHERE is_deleted = FALSE"
	var countArgs []interface{}
	if status != "" {
		countSQL += " AND status = $1"
		countArgs = append(countArgs, status)
	}
	var total int64
	if err := r.c.db.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("postgres DeliveryRepo.List count: %w", err)
	}

	// Data query
	dataSQL := `SELECT delivery_id, customer_id, status, delivered_at,
  contract_id, delivery_type, requested_by, approved_by, delivered_by,
  manifest_uri, replay_manifest_uri, item_count, total_size_bytes,
  completed_at, metadata, tenant_id, project_id,
  created_at, updated_at, version
FROM deliveries WHERE is_deleted = FALSE`
	var dataArgs []interface{}
	paramIdx := 1
	if status != "" {
		dataSQL += fmt.Sprintf(" AND status = $%d", paramIdx)
		dataArgs = append(dataArgs, status)
		paramIdx++
	}
	dataSQL += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", paramIdx, paramIdx+1)
	dataArgs = append(dataArgs, pageSize, offset)

	rows, err := r.c.db.Query(ctx, dataSQL, dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres DeliveryRepo.List query: %w", err)
	}
	defer rows.Close()

	var out []*models.Delivery
	for rows.Next() {
		var (
			d            models.Delivery
			st           string
			metadataJSON []byte
			tenantID     *string
			projectID    *string
		)
		if err := rows.Scan(
			&d.DeliveryID, &d.CustomerID, &st, &d.DeliveredAt,
			&d.ContractID, &d.DeliveryType, &d.RequestedBy, &d.ApprovedBy, &d.DeliveredBy,
			&d.ManifestURI, &d.ReplayManifestURI, &d.ItemCount, &d.TotalSizeBytes,
			&d.CompletedAt, &metadataJSON, &tenantID, &projectID,
			&d.CreatedAt, &d.UpdatedAt, &d.Version,
		); err != nil {
			return nil, 0, fmt.Errorf("postgres DeliveryRepo.List scan: %w", err)
		}
		d.Status = models.DeliveryStatus(st)
		if tenantID != nil {
			d.TenantID = *tenantID
		}
		if projectID != nil {
			d.ProjectID = *projectID
		}
		// Parse metadata JSONB.
		if len(metadataJSON) > 0 {
			_ = json.Unmarshal(metadataJSON, &d.Metadata)
		}
		// Populate legacy fields from real columns.
		if d.Note == "" {
			if m, ok := d.Metadata["note"].(string); ok {
				d.Note = m
			}
		}
		if d.Owner == "" && d.DeliveredBy != "" {
			d.Owner = d.DeliveredBy
		}
		// Sync legacy AssetCount ↔ ItemCount.
		if d.AssetCount == 0 && d.ItemCount > 0 {
			d.AssetCount = int(d.ItemCount)
		} else if d.ItemCount == 0 && d.AssetCount > 0 {
			d.ItemCount = int64(d.AssetCount)
		}
		// Sync Owner ↔ DeliveredBy.
		if d.DeliveredBy == "" && d.Owner != "" {
			d.DeliveredBy = d.Owner
		}
		out = append(out, &d)
	}
	return out, total, nil
}

func (r *DeliveryRepo) ListByAsset(ctx context.Context, assetID string) ([]string, error) {
	const q = `
SELECT delivery_id
FROM delivery_items
WHERE asset_id=$1
ORDER BY created_at DESC`
	rows, err := r.c.db.Query(ctx, q, assetID)
	if err != nil {
		return nil, fmt.Errorf("postgres DeliveryRepo.ListByAsset: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("postgres DeliveryRepo.ListByAsset scan: %w", err)
		}
		out = append(out, id)
	}
	return out, nil
}

func (r *DeliveryRepo) ListByCustomer(ctx context.Context, customerID string) ([]string, error) {
	const q = `
SELECT delivery_id
FROM deliveries
WHERE customer_id=$1 AND is_deleted=FALSE
ORDER BY delivered_at DESC NULLS LAST, created_at DESC`
	rows, err := r.c.db.Query(ctx, q, customerID)
	if err != nil {
		return nil, fmt.Errorf("postgres DeliveryRepo.ListByCustomer: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("postgres DeliveryRepo.ListByCustomer scan: %w", err)
		}
		out = append(out, id)
	}
	return out, nil
}

type IdempotencyRepo struct {
	c *Client
}

var _ repository.IdempotencyRepository = (*IdempotencyRepo)(nil)

func NewIdempotencyRepo(c *Client) *IdempotencyRepo { return &IdempotencyRepo{c: c} }

func (r *IdempotencyRepo) Get(ctx context.Context, scope, key string) (*repository.IdempotencyRecord, error) {
	const q = `
SELECT scope, idem_key, request_hash, status_code, response_json, created_at
FROM idempotency_keys
WHERE scope=$1 AND idem_key=$2`
	rec := &repository.IdempotencyRecord{}
	err := r.c.db.QueryRow(ctx, q, scope, key).Scan(
		&rec.Scope, &rec.Key, &rec.RequestHash, &rec.StatusCode, &rec.Response, &rec.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, errNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres IdempotencyRepo.Get: %w", err)
	}
	return rec, nil
}

func (r *IdempotencyRepo) Save(ctx context.Context, rec *repository.IdempotencyRecord) error {
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = time.Now().UTC()
	}
	const q = `
INSERT INTO idempotency_keys(scope, idem_key, request_hash, status_code, response_json, created_at)
VALUES ($1,$2,$3,$4,$5,$6)
ON CONFLICT (scope, idem_key) DO UPDATE SET
  request_hash=EXCLUDED.request_hash,
  status_code=EXCLUDED.status_code,
  response_json=EXCLUDED.response_json,
  created_at=EXCLUDED.created_at`
	err := r.c.db.Exec(ctx, q, rec.Scope, rec.Key, rec.RequestHash, rec.StatusCode, rec.Response, rec.CreatedAt)
	if err != nil {
		return fmt.Errorf("postgres IdempotencyRepo.Save: %w", err)
	}
	return nil
}

// ──────────────────────────────────────────────────────────────────────────────
// AssetTagRepo — upserts tags to the asset_tags projection table.
// ──────────────────────────────────────────────────────────────────────────────

type AssetTagRepo struct {
	c *Client
}

func NewAssetTagRepo(c *Client) *AssetTagRepo { return &AssetTagRepo{c: c} }

// Upsert inserts or updates a tag in the asset_tags table.
func (r *AssetTagRepo) Upsert(ctx context.Context, assetID, tagKey, tagValue, tagType, sourceType string) error {
	const tagQ = `
INSERT INTO asset_tags (asset_id, tag_key, tag_value, tag_type, source_type, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, now(), now())
ON CONFLICT (asset_id, tag_key) DO UPDATE SET
  tag_value   = EXCLUDED.tag_value,
  tag_type    = EXCLUDED.tag_type,
  source_type = EXCLUDED.source_type,
  updated_at  = now()`
	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, tagQ, assetID, tagKey, tagValue, tagType, sourceType); err != nil {
		return fmt.Errorf("postgres AssetTagRepo.Upsert asset_tags: %w", err)
	}
	return nil
}

var _ repository.AssetTagRepository = (*AssetTagRepo)(nil)

// ListByAsset returns all tags for the given asset, ordered by tag_key.
func (r *AssetTagRepo) ListByAsset(ctx context.Context, assetID string) ([]*models.AssetTag, error) {
	const q = `
SELECT asset_id, tag_key, tag_value, tag_type, source_type,
  COALESCE(source_name, ''), COALESCE(source_version, ''),
  COALESCE(run_id, ''), COALESCE(tenant_id, ''), COALESCE(project_id, ''),
  created_at, updated_at
FROM asset_tags
WHERE asset_id = $1
ORDER BY tag_key`
	db := dbFromCtx(ctx, r.c.db)
	rows, err := db.Query(ctx, q, assetID)
	if err != nil {
		return nil, fmt.Errorf("postgres AssetTagRepo.ListByAsset: %w", err)
	}
	defer rows.Close()
	var out []*models.AssetTag
	for rows.Next() {
		var t models.AssetTag
		if err := rows.Scan(
			&t.AssetID, &t.TagKey, &t.TagValue, &t.TagType, &t.SourceType,
			&t.SourceName, &t.SourceVersion,
			&t.RunID, &t.TenantID, &t.ProjectID,
			&t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("postgres AssetTagRepo.ListByAsset scan: %w", err)
		}
		out = append(out, &t)
	}
	return out, nil
}

// Delete removes a tag from the asset_tags table.
func (r *AssetTagRepo) Delete(ctx context.Context, assetID, tagKey string) error {
	const delQ = `DELETE FROM asset_tags WHERE asset_id = $1 AND tag_key = $2`
	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, delQ, assetID, tagKey); err != nil {
		return fmt.Errorf("postgres AssetTagRepo.Delete asset_tags: %w", err)
	}
	return nil
}

// ──────────────────────────────────────────────────────────────────────────────
// AssetAlgoLatestRepo — source of truth for per-algorithm state on an asset.
// PK = (asset_id, algo_name). Concurrent writes from different algo versions
// are made safe by the monotonic guard in Upsert (older algo_version is
// silently discarded), so no lock on `assets` is required for finishes.
// ──────────────────────────────────────────────────────────────────────────────

type AssetAlgoLatestRepo struct {
	c *Client
}

func NewAssetAlgoLatestRepo(c *Client) *AssetAlgoLatestRepo {
	return &AssetAlgoLatestRepo{c: c}
}

var _ repository.AssetAlgoLatestRepository = (*AssetAlgoLatestRepo)(nil)

const algoLatestSelectColumns = `
asset_id, algo_name, algo_version, status,
COALESCE(result_tag, ''), result_score, result_summary,
COALESCE(run_id, ''), COALESCE(method, ''), COALESCE(model_uri, ''),
COALESCE(output_uri, ''), COALESCE(error_code, ''), COALESCE(error_message, ''),
started_at, finished_at,
COALESCE(tenant_id, ''), COALESCE(project_id, ''),
updated_at`

func scanAlgoLatest(rs rowScanner, a *models.AssetAlgoLatest) error {
	var (
		summaryBytes []byte
		score        *float64
	)
	if err := rs.Scan(
		&a.AssetID, &a.AlgoName, &a.AlgoVersion, &a.Status,
		&a.ResultTag, &score, &summaryBytes,
		&a.RunID, &a.Method, &a.ModelURI,
		&a.OutputURI, &a.ErrorCode, &a.ErrorMessage,
		&a.StartedAt, &a.FinishedAt,
		&a.TenantID, &a.ProjectID,
		&a.UpdatedAt,
	); err != nil {
		return err
	}
	a.ResultScore = score
	if len(summaryBytes) > 0 {
		_ = json.Unmarshal(summaryBytes, &a.ResultSummary)
	}
	return nil
}

// GetByAlgo returns the latest row for (assetID, algoName) or (nil, nil) when
// no row exists. Tx-aware via ctx.
func (r *AssetAlgoLatestRepo) GetByAlgo(ctx context.Context, assetID, algoName string) (*models.AssetAlgoLatest, error) {
	const q = `SELECT ` + algoLatestSelectColumns + `
FROM asset_algo_latest
WHERE asset_id = $1 AND algo_name = $2`
	db := dbFromCtx(ctx, r.c.db)
	row := db.QueryRow(ctx, q, assetID, algoName)
	var a models.AssetAlgoLatest
	if err := scanAlgoLatest(row, &a); err != nil {
		if errors.Is(err, errNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres AssetAlgoLatestRepo.GetByAlgo: %w", err)
	}
	return &a, nil
}

// ListByAsset returns all algo rows for the given asset, ordered by algo_name.
func (r *AssetAlgoLatestRepo) ListByAsset(ctx context.Context, assetID string) ([]*models.AssetAlgoLatest, error) {
	const q = `SELECT ` + algoLatestSelectColumns + `
FROM asset_algo_latest
WHERE asset_id = $1
ORDER BY algo_name`
	db := dbFromCtx(ctx, r.c.db)
	rows, err := db.Query(ctx, q, assetID)
	if err != nil {
		return nil, fmt.Errorf("postgres AssetAlgoLatestRepo.ListByAsset: %w", err)
	}
	defer rows.Close()
	var out []*models.AssetAlgoLatest
	for rows.Next() {
		var a models.AssetAlgoLatest
		if err := scanAlgoLatest(rows, &a); err != nil {
			return nil, fmt.Errorf("postgres AssetAlgoLatestRepo.ListByAsset scan: %w", err)
		}
		out = append(out, &a)
	}
	return out, nil
}

// Upsert writes a full algo row with a MONOTONIC GUARD on algo_version: when
// a row with the same (asset_id, algo_name) already exists with a strictly
// newer algo_version, the write is silently dropped (rows-affected = 0) so
// out-of-order concurrent finishes never roll state backwards. Tx-aware.
//
// Distinct algo_version values are compared lexicographically; this is the
// same comparison used elsewhere in the system (e.g. `gripper@v1` < `gripper@v2`).
// If a strict ordering is needed across non-lexicographic versions, callers
// should normalise to a sortable form before writing.
func (r *AssetAlgoLatestRepo) Upsert(ctx context.Context, row *models.AssetAlgoLatest) error {
	if row == nil {
		return errors.New("postgres AssetAlgoLatestRepo.Upsert: nil row")
	}
	summaryBytes := []byte(`{}`)
	if row.ResultSummary != nil {
		b, err := json.Marshal(row.ResultSummary)
		if err != nil {
			return fmt.Errorf("postgres AssetAlgoLatestRepo.Upsert marshal result_summary: %w", err)
		}
		summaryBytes = b
	}
	const q = `
INSERT INTO asset_algo_latest (
  asset_id, algo_name, algo_version, status,
  result_tag, result_score, result_summary,
  run_id, method, model_uri,
  output_uri, error_code, error_message,
  started_at, finished_at,
  tenant_id, project_id, updated_at
) VALUES (
  $1, $2, $3, $4,
  NULLIF($5, ''), $6, $7::jsonb,
  NULLIF($8, ''), NULLIF($9, ''), NULLIF($10, ''),
  NULLIF($11, ''), NULLIF($12, ''), NULLIF($13, ''),
  $14, $15,
  NULLIF($16, ''), NULLIF($17, ''), now()
)
ON CONFLICT (asset_id, algo_name) DO UPDATE SET
  algo_version    = EXCLUDED.algo_version,
  status          = EXCLUDED.status,
  result_tag      = EXCLUDED.result_tag,
  result_score    = EXCLUDED.result_score,
  result_summary  = EXCLUDED.result_summary,
  run_id          = EXCLUDED.run_id,
  method          = COALESCE(EXCLUDED.method, asset_algo_latest.method),
  model_uri       = COALESCE(EXCLUDED.model_uri, asset_algo_latest.model_uri),
  output_uri      = EXCLUDED.output_uri,
  error_code      = EXCLUDED.error_code,
  error_message   = EXCLUDED.error_message,
  started_at      = COALESCE(EXCLUDED.started_at, asset_algo_latest.started_at),
  finished_at     = EXCLUDED.finished_at,
  tenant_id       = COALESCE(EXCLUDED.tenant_id, asset_algo_latest.tenant_id),
  project_id      = COALESCE(EXCLUDED.project_id, asset_algo_latest.project_id),
  updated_at      = now()
WHERE asset_algo_latest.algo_version <= EXCLUDED.algo_version`
	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, q,
		row.AssetID, row.AlgoName, row.AlgoVersion, row.Status,
		row.ResultTag, row.ResultScore, summaryBytes,
		row.RunID, row.Method, row.ModelURI,
		row.OutputURI, row.ErrorCode, row.ErrorMessage,
		row.StartedAt, row.FinishedAt,
		row.TenantID, row.ProjectID,
	); err != nil {
		return fmt.Errorf("postgres AssetAlgoLatestRepo.Upsert: %w", err)
	}
	return nil
}

// ──────────────────────────────────────────────────────────────────────────────
// AssetEventRepo — appends events to the asset_events outbox table.
// ──────────────────────────────────────────────────────────────────────────────

// AssetEventRepo provides append-only access to the asset_events outbox table.
type AssetEventRepo struct {
	c *Client
}

// NewAssetEventRepo creates a new AssetEventRepo.
func NewAssetEventRepo(c *Client) *AssetEventRepo { return &AssetEventRepo{c: c} }

var _ repository.AssetEventRepository = (*AssetEventRepo)(nil)

// ListPending returns up to `limit` events with publish_state='pending',
// ordered by event_seq ascending (oldest first) for monotonic consumption.
func (r *AssetEventRepo) ListPending(ctx context.Context, limit int) ([]*models.AssetEvent, error) {
	if limit <= 0 {
		limit = 100
	}
	const q = `
SELECT event_id, event_seq, event_type, aggregate_type, payload_schema_version,
  COALESCE(asset_id::text, ''), COALESCE(mcap_file_id::text, ''),
  COALESCE(tenant_id, ''), COALESCE(project_id, ''),
  event_source, publish_state, event_payload,
  retry_count, COALESCE(last_error, ''),
  occurred_at, created_at, published_at
FROM asset_events
WHERE publish_state = 'pending'
ORDER BY event_seq ASC
LIMIT $1`
	rows, err := r.c.db.Query(ctx, q, limit)
	if err != nil {
		return nil, fmt.Errorf("postgres AssetEventRepo.ListPending: %w", err)
	}
	defer rows.Close()
	var out []*models.AssetEvent
	for rows.Next() {
		var e models.AssetEvent
		if err := rows.Scan(
			&e.EventID, &e.EventSeq, &e.EventType, &e.AggregateType, &e.PayloadSchemaVersion,
			&e.AssetID, &e.McapFileID,
			&e.TenantID, &e.ProjectID,
			&e.EventSource, &e.PublishState, &e.EventPayload,
			&e.RetryCount, &e.LastError,
			&e.OccurredAt, &e.CreatedAt, &e.PublishedAt,
		); err != nil {
			return nil, fmt.Errorf("postgres AssetEventRepo.ListPending scan: %w", err)
		}
		out = append(out, &e)
	}
	return out, nil
}

// Append inserts a new event row into asset_events. Tx-aware via ctx so the
// caller's transaction (e.g. inside Client.WithTx) atomically wraps the
// business write and the event row — this is the outbox correctness
// guarantee that lets consumers replay safely.
func (r *AssetEventRepo) Append(ctx context.Context, in repository.AssetEventAppendInput) error {
	if in.EventType == "" {
		return errors.New("postgres AssetEventRepo.Append: event_type is required")
	}
	payload := in.EventPayload
	if payload == nil {
		payload = []byte(`{}`)
	}
	schemaVer := in.PayloadSchemaVersion
	if schemaVer == "" {
		schemaVer = "v1"
	}
	source := in.EventSource
	if source == "" {
		source = "backend"
	}
	aggregateType := in.AggregateType
	if aggregateType == "" {
		aggregateType = "asset"
	}

	// Convert empty strings to NULL for nullable columns.
	nullable := func(s string) interface{} {
		if s == "" {
			return nil
		}
		return s
	}

	const q = `
INSERT INTO asset_events (
  event_id, event_type, aggregate_type, payload_schema_version,
  asset_id, mcap_file_id,
  tenant_id, project_id,
  event_source,
  actor_type, actor_id, request_id, idempotency_key, run_id,
  event_payload
) VALUES (
  gen_random_uuid(), $1, $2, $3,
  $4, $5,
  $6, $7,
  $8,
  $9, $10, $11, $12, $13,
  $14::jsonb
)`

	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, q,
		in.EventType, aggregateType, schemaVer,
		nullable(in.AssetID), nullable(in.McapFileID),
		nullable(in.TenantID), nullable(in.ProjectID),
		source,
		nullable(in.ActorType), nullable(in.ActorID), nullable(in.RequestID),
		nullable(in.IdempotencyKey), nullable(in.RunID),
		payload,
	); err != nil {
		return fmt.Errorf("postgres AssetEventRepo.Append: %w", err)
	}
	return nil
}

// ListByAsset returns events for a given asset, optionally filtered by event
// types, ordered by event_seq DESC (most recent first). Used by the
// "list algo events" API (replaces the deprecated algo_events table).
// limit ≤ 0 defaults to 100.
func (r *AssetEventRepo) ListByAsset(ctx context.Context, assetID string, eventTypes []string, limit int) ([]*models.AssetEvent, error) {
	if limit <= 0 {
		limit = 100
	}
	args := []interface{}{assetID, limit}
	whereTypes := ""
	if len(eventTypes) > 0 {
		// Build a positional ANY($N) parameter; pgx accepts []string via $N=ANY.
		args = []interface{}{assetID, eventTypes, limit}
		whereTypes = " AND event_type = ANY($2)"
	}
	limitParam := "$2"
	if len(eventTypes) > 0 {
		limitParam = "$3"
	}
	q := `
SELECT event_id, event_seq, event_type, aggregate_type, payload_schema_version,
  COALESCE(asset_id::text, ''), COALESCE(mcap_file_id::text, ''),
  COALESCE(tenant_id, ''), COALESCE(project_id, ''),
  event_source, publish_state, event_payload,
  retry_count, COALESCE(last_error, ''),
  occurred_at, created_at, published_at
FROM asset_events
WHERE asset_id = $1` + whereTypes + `
ORDER BY event_seq DESC
LIMIT ` + limitParam
	rows, err := r.c.db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("postgres AssetEventRepo.ListByAsset: %w", err)
	}
	defer rows.Close()
	var out []*models.AssetEvent
	for rows.Next() {
		var e models.AssetEvent
		if err := rows.Scan(
			&e.EventID, &e.EventSeq, &e.EventType, &e.AggregateType, &e.PayloadSchemaVersion,
			&e.AssetID, &e.McapFileID,
			&e.TenantID, &e.ProjectID,
			&e.EventSource, &e.PublishState, &e.EventPayload,
			&e.RetryCount, &e.LastError,
			&e.OccurredAt, &e.CreatedAt, &e.PublishedAt,
		); err != nil {
			return nil, fmt.Errorf("postgres AssetEventRepo.ListByAsset scan: %w", err)
		}
		out = append(out, &e)
	}
	return out, nil
}

// ListWithFilters queries assets with a parameterized WHERE clause, pagination, and ordering.
func (r *AssetRepo) ListWithFilters(ctx context.Context, whereSQL string, args []interface{},
	page, pageSize int, orderBy string) ([]*models.Asset, int64, error) {

	if orderBy == "" {
		orderBy = "created_at DESC"
	}
	offset := (page - 1) * pageSize

	// Build the base WHERE clause — always exclude soft-deleted rows.
	baseWhere := "is_deleted = FALSE"
	if whereSQL != "" {
		baseWhere += " AND " + whereSQL
	}

	// --- COUNT query ---
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM assets WHERE %s", baseWhere)
	var total int64
	err := r.c.db.QueryRow(ctx, countSQL, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres AssetRepo.ListWithFilters count: %w", err)
	}

	// --- DATA query ---
	nextParam := len(args) + 1
	dataSQL := fmt.Sprintf(
		`SELECT asset_id, mcap_file_id, start_timestamp_ns, end_timestamp_ns, segment_locator,
  status, lifecycle_state, asset_type, duration_ms,
  owner, reviewer, delivery_count, last_delivered_at, last_delivered_to,
  retention_tier, expire_at, storage_uri, thumb_uri, asset_level,
  created_at, updated_at, version
FROM assets WHERE %s ORDER BY %s LIMIT $%d OFFSET $%d`,
		baseWhere, orderBy, nextParam, nextParam+1,
	)
	dataArgs := append(append([]interface{}{}, args...), pageSize, offset)

	rows, err := r.c.db.Query(ctx, dataSQL, dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres AssetRepo.ListWithFilters query: %w", err)
	}
	defer rows.Close()

	var out []*models.Asset
	for rows.Next() {
		var (
			a              models.Asset
			status         string
			lifecycleState string
			segLoc         *string
		)
		if err := rows.Scan(
			&a.AssetID, &a.McapFileID, &a.StartTimestampNs, &a.EndTimestampNs, &segLoc,
			&status, &lifecycleState, &a.AssetType, &a.DurationMs,
			&a.Owner, &a.Reviewer, &a.DeliveryCount, &a.LastDeliveredAt, &a.LastDeliveredTo,
			&a.RetentionTier, &a.ExpireAt, &a.StorageURI, &a.ThumbURI, &a.AssetLevel,
			&a.CreatedAt, &a.UpdatedAt, &a.Version,
		); err != nil {
			return nil, 0, fmt.Errorf("postgres AssetRepo.ListWithFilters scan: %w", err)
		}
		a.Status = models.AssetStatus(status)
		a.LifecycleState = lifecycleState
		if segLoc != nil {
			a.SegmentLocator = *segLoc
		}
		a.SyncLegacyFields()
		out = append(out, &a)
	}
	return out, total, nil
}

// MergeCfAlgo writes algo status updates to the asset_algo_latest table with
// optimistic locking on the asset version.
// Returns the new version. If expectedVersion doesn't match, returns repository.ErrOptimisticLock.
func (r *AssetRepo) MergeCfAlgo(ctx context.Context, assetID string, expectedVersion int64,
	algoKV map[string]interface{}, filesKV map[string]interface{}) (int64, error) {

	// Bump the asset version with optimistic lock.
	const q = `
UPDATE assets
SET version = version + 1,
    updated_at = now()
WHERE asset_id = $1 AND version = $2 AND is_deleted = FALSE`

	rowsAffected, err := r.c.db.ExecResult(ctx, q, assetID, expectedVersion)
	if err != nil {
		return 0, fmt.Errorf("postgres AssetRepo.MergeCfAlgo: %w", err)
	}
	if rowsAffected == 0 {
		return 0, repository.ErrOptimisticLock
	}

	// Write to asset_algo_latest for any :status keys.
	for key, val := range algoKV {
		algoName, algoVersion, field := parseAlgoKVKey(key)
		if field != "status" || algoName == "" {
			continue
		}
		status, _ := val.(string)
		if status == "" {
			continue
		}
		const algoLatestQ = `
INSERT INTO asset_algo_latest (asset_id, algo_name, algo_version, status, updated_at)
VALUES ($1, $2, $3, $4, now())
ON CONFLICT (asset_id, algo_name) DO UPDATE SET
  algo_version = EXCLUDED.algo_version,
  status       = EXCLUDED.status,
  updated_at   = now()`
		if err := r.c.db.Exec(ctx, algoLatestQ, assetID, algoName, algoVersion, status); err != nil {
			return expectedVersion + 1, fmt.Errorf("postgres AssetRepo.MergeCfAlgo asset_algo_latest: %w", err)
		}
	}

	return expectedVersion + 1, nil
}

// parseAlgoKVKey parses a cf_algo key in the format "<algo_name>@<version>:<field>"
// and returns the components. Returns empty strings if the format is invalid.
func parseAlgoKVKey(key string) (algoName, algoVersion, field string) {
	// Find the last colon to split field.
	colonIdx := -1
	for i := len(key) - 1; i >= 0; i-- {
		if key[i] == ':' {
			colonIdx = i
			break
		}
	}
	if colonIdx <= 0 || colonIdx >= len(key)-1 {
		return "", "", ""
	}
	field = key[colonIdx+1:]
	prefix := key[:colonIdx] // "algo_name@version"

	// Find the @ to split name and version.
	atIdx := -1
	for i := len(prefix) - 1; i >= 0; i-- {
		if prefix[i] == '@' {
			atIdx = i
			break
		}
	}
	if atIdx <= 0 || atIdx >= len(prefix)-1 {
		return "", "", ""
	}
	algoName = prefix[:atIdx]
	algoVersion = prefix[atIdx+1:]
	return
}
