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
SELECT asset_id, mcap_file_id, start_timestamp_ns, status, cf_meta, cf_algo, cf_tag, created_at, updated_at, version
FROM assets
WHERE asset_id = $1 AND is_deleted = FALSE`
	var (
		a                                    models.Asset
		status                               string
		cfMetaBytes, cfAlgoBytes, cfTagBytes []byte
	)
	err := r.c.db.QueryRow(ctx, q, assetID).Scan(
		&a.AssetID, &a.McapFileID, &a.StartTimestampNs, &status,
		&cfMetaBytes, &cfAlgoBytes, &cfTagBytes,
		&a.CreatedAt, &a.UpdatedAt, &a.Version,
	)
	if err != nil {
		if errors.Is(err, errNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres AssetRepo.Get: %w", err)
	}
	a.Status = models.AssetStatus(status)
	applyAssetJSON(&a, cfMetaBytes, cfAlgoBytes, cfTagBytes)
	return &a, nil
}

func (r *AssetRepo) Set(ctx context.Context, a *models.Asset) error {
	now := time.Now().UTC()
	if a.CreatedAt.IsZero() {
		a.CreatedAt = now
	}
	a.UpdatedAt = now
	a.Version++
	meta, algo, tag := assetJSON(a)
	const q = `
INSERT INTO assets(asset_id, mcap_file_id, start_timestamp_ns, status, is_deleted, cf_meta, cf_algo, cf_tag, created_at, updated_at, version)
VALUES ($1,$2,$3,$4,FALSE,$5::jsonb,$6::jsonb,$7::jsonb,$8,$9,$10)
ON CONFLICT (asset_id) DO UPDATE SET
  mcap_file_id=EXCLUDED.mcap_file_id,
  start_timestamp_ns=EXCLUDED.start_timestamp_ns,
  status=EXCLUDED.status,
  cf_meta=EXCLUDED.cf_meta,
  cf_algo=EXCLUDED.cf_algo,
  cf_tag=EXCLUDED.cf_tag,
  updated_at=EXCLUDED.updated_at,
  version=EXCLUDED.version`
	err := r.c.db.Exec(ctx, q,
		a.AssetID, a.McapFileID, a.StartTimestampNs, string(a.Status),
		meta, algo, tag, a.CreatedAt, a.UpdatedAt, a.Version,
	)
	if err != nil {
		return fmt.Errorf("postgres AssetRepo.Set: %w", err)
	}
	return nil
}

func (r *AssetRepo) SoftDelete(ctx context.Context, assetID string) error {
	const q = `UPDATE assets SET is_deleted=TRUE, status='archived', updated_at=now() WHERE asset_id=$1`
	err := r.c.db.Exec(ctx, q, assetID)
	if err != nil {
		return fmt.Errorf("postgres AssetRepo.SoftDelete: %w", err)
	}
	return nil
}

func (r *AssetRepo) ListByMcapFile(ctx context.Context, mcapFileID string) ([]*models.Asset, error) {
	const q = `
SELECT asset_id, mcap_file_id, start_timestamp_ns, status, cf_meta, cf_algo, cf_tag, created_at, updated_at, version
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
			a                                    models.Asset
			status                               string
			cfMetaBytes, cfAlgoBytes, cfTagBytes []byte
		)
		if err := rows.Scan(
			&a.AssetID, &a.McapFileID, &a.StartTimestampNs, &status,
			&cfMetaBytes, &cfAlgoBytes, &cfTagBytes,
			&a.CreatedAt, &a.UpdatedAt, &a.Version,
		); err != nil {
			return nil, fmt.Errorf("postgres AssetRepo.ListByMcapFile scan: %w", err)
		}
		a.Status = models.AssetStatus(status)
		applyAssetJSON(&a, cfMetaBytes, cfAlgoBytes, cfTagBytes)
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
SELECT mcap_file_id, raw_hash_md5, cf_meta, cf_process, created_at, updated_at, version
FROM mcap_files
WHERE mcap_file_id = $1 AND is_deleted = FALSE`
	var (
		f                       models.McapFile
		metaBytes, processBytes []byte
	)
	err := r.c.db.QueryRow(ctx, q, mcapFileID).Scan(
		&f.McapFileID, &f.RawHashMD5, &metaBytes, &processBytes, &f.CreatedAt, &f.UpdatedAt, &f.Version,
	)
	if err != nil {
		if errors.Is(err, errNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres McapFileRepo.Get: %w", err)
	}
	applyMcapJSON(&f, metaBytes, processBytes)
	return &f, nil
}

func (r *McapFileRepo) Set(ctx context.Context, f *models.McapFile) error {
	now := time.Now().UTC()
	if f.CreatedAt.IsZero() {
		f.CreatedAt = now
	}
	f.UpdatedAt = now
	f.Version++
	meta, process := mcapJSON(f)
	const q = `
INSERT INTO mcap_files(mcap_file_id, raw_hash_md5, is_deleted, cf_meta, cf_process, created_at, updated_at, version)
VALUES ($1,$2,FALSE,$3::jsonb,$4::jsonb,$5,$6,$7)
ON CONFLICT (mcap_file_id) DO UPDATE SET
  raw_hash_md5=EXCLUDED.raw_hash_md5,
  cf_meta=EXCLUDED.cf_meta,
  cf_process=EXCLUDED.cf_process,
  updated_at=EXCLUDED.updated_at,
  version=EXCLUDED.version`
	err := r.c.db.Exec(ctx, q, f.McapFileID, f.RawHashMD5, meta, process, f.CreatedAt, f.UpdatedAt, f.Version)
	if err != nil {
		return fmt.Errorf("postgres McapFileRepo.Set: %w", err)
	}
	return nil
}

func (r *McapFileRepo) UpdateIngestState(ctx context.Context, mcapFileID string, state models.IngestState) error {
	const q = `
UPDATE mcap_files
SET cf_meta = jsonb_set(cf_meta, '{ingest_state}', to_jsonb($2::text), true),
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
	meta, _ := json.Marshal(map[string]any{
		"manifest_uri": d.ManifestURI,
		"contract_id":  d.ContractID,
		"note":         d.Note,
		"asset_count":  d.AssetCount,
		"owner":        d.Owner,
	})
	const q = `
INSERT INTO deliveries(delivery_id, customer_id, status, delivered_at, is_deleted, cf_meta, created_at, updated_at, version)
VALUES ($1,$2,$3,$4,FALSE,$5::jsonb,$6,$7,$8)
ON CONFLICT (delivery_id) DO UPDATE SET
  customer_id=EXCLUDED.customer_id,
  status=EXCLUDED.status,
  delivered_at=EXCLUDED.delivered_at,
  cf_meta=EXCLUDED.cf_meta,
  updated_at=EXCLUDED.updated_at,
  version=EXCLUDED.version`
	err := r.c.db.Exec(ctx, q, d.DeliveryID, d.CustomerID, string(d.Status), d.DeliveredAt, meta, d.CreatedAt, d.UpdatedAt, d.Version)
	if err != nil {
		return fmt.Errorf("postgres DeliveryRepo.Set: %w", err)
	}
	return nil
}

func (r *DeliveryRepo) Get(ctx context.Context, deliveryID string) (*models.Delivery, error) {
	const q = `
SELECT delivery_id, customer_id, status, delivered_at, cf_meta, created_at, updated_at, version
FROM deliveries WHERE delivery_id=$1 AND is_deleted=FALSE`
	var (
		d      models.Delivery
		status string
		meta   []byte
	)
	err := r.c.db.QueryRow(ctx, q, deliveryID).Scan(
		&d.DeliveryID, &d.CustomerID, &status, &d.DeliveredAt, &meta, &d.CreatedAt, &d.UpdatedAt, &d.Version,
	)
	if err != nil {
		if errors.Is(err, errNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres DeliveryRepo.Get: %w", err)
	}
	d.Status = models.DeliveryStatus(status)
	var m map[string]any
	_ = json.Unmarshal(meta, &m)
	if v, ok := m["manifest_uri"].(string); ok {
		d.ManifestURI = v
	}
	if v, ok := m["contract_id"].(string); ok {
		d.ContractID = v
	}
	if v, ok := m["note"].(string); ok {
		d.Note = v
	}
	if v, ok := m["owner"].(string); ok {
		d.Owner = v
	}
	if v, ok := m["asset_count"].(float64); ok {
		d.AssetCount = int(v)
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

func assetJSON(a *models.Asset) (meta, algo, tag []byte) {
	metaMap := map[string]any{
		"end_timestamp_ns":  a.EndTimestampNs,
		"duration_sec":      a.DurationSec,
		"reviewer":          a.Reviewer,
		"owner":             a.Owner,
		"type":              a.SegType,
		"env":               a.Env,
		"task":              a.Task,
		"delivery_count":    a.DeliveryCount,
		"last_delivered_to": a.LastDeliveredTo,
		"created_at":        a.CreatedAt,
		"updated_at":        a.UpdatedAt,
	}
	if a.LastDeliveredAt != nil {
		metaMap["last_delivered_at"] = a.LastDeliveredAt
	}
	meta, _ = json.Marshal(metaMap)
	algo, _ = json.Marshal(a.AlgoResults)
	tag, _ = json.Marshal(a.Tags)
	return
}

func applyAssetJSON(a *models.Asset, meta, algo, tag []byte) {
	var m map[string]any
	_ = json.Unmarshal(meta, &m)
	if v, ok := m["end_timestamp_ns"].(float64); ok {
		a.EndTimestampNs = int64(v)
	}
	if v, ok := m["duration_sec"].(float64); ok {
		a.DurationSec = v
	}
	if v, ok := m["reviewer"].(string); ok {
		a.Reviewer = v
	}
	if v, ok := m["owner"].(string); ok {
		a.Owner = v
	}
	if v, ok := m["type"].(string); ok {
		a.SegType = v
	}
	if v, ok := m["env"].(string); ok {
		a.Env = v
	}
	if v, ok := m["task"].(string); ok {
		a.Task = v
	}
	if v, ok := m["delivery_count"].(float64); ok {
		a.DeliveryCount = int(v)
	}
	if v, ok := m["last_delivered_to"].(string); ok {
		a.LastDeliveredTo = v
	}
	if v, ok := m["last_delivered_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
			a.LastDeliveredAt = &t
		}
	}
	if a.AlgoResults == nil {
		a.AlgoResults = map[string]string{}
	}
	if a.Tags == nil {
		a.Tags = map[string]string{}
	}
	_ = json.Unmarshal(algo, &a.AlgoResults)
	_ = json.Unmarshal(tag, &a.Tags)
}

func mcapJSON(f *models.McapFile) (meta, process []byte) {
	metaMap := map[string]any{
		"gcs_path":           f.GCSPath,
		"size_bytes":         f.SizeBytes,
		"ingest_state":       f.IngestState,
		"start_timestamp_ns": f.StartTimestampNs,
		"end_timestamp_ns":   f.EndTimestampNs,
		"channel_count":      f.ChannelCount,
		"chunk_count":        f.ChunkCount,
		"owner":              f.Owner,
	}
	meta, _ = json.Marshal(metaMap)
	process, _ = json.Marshal(f.ProcessState)
	return
}

func applyMcapJSON(f *models.McapFile, meta, process []byte) {
	var m map[string]any
	_ = json.Unmarshal(meta, &m)
	if v, ok := m["gcs_path"].(string); ok {
		f.GCSPath = v
	}
	if v, ok := m["size_bytes"].(float64); ok {
		f.SizeBytes = int64(v)
	}
	if v, ok := m["ingest_state"].(string); ok {
		f.IngestState = models.IngestState(v)
	}
	if v, ok := m["start_timestamp_ns"].(float64); ok {
		f.StartTimestampNs = int64(v)
	}
	if v, ok := m["end_timestamp_ns"].(float64); ok {
		f.EndTimestampNs = int64(v)
	}
	if v, ok := m["channel_count"].(float64); ok {
		f.ChannelCount = int(v)
	}
	if v, ok := m["chunk_count"].(float64); ok {
		f.ChunkCount = int(v)
	}
	if v, ok := m["owner"].(string); ok {
		f.Owner = v
	}
	if f.ProcessState == nil {
		f.ProcessState = map[string]string{}
	}
	_ = json.Unmarshal(process, &f.ProcessState)
}
