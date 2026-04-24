package bigtable

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"cloud.google.com/go/bigtable"

	"data-platform/internal/models"
	"data-platform/internal/repository"
)

// AssetRepo provides read/write access to the `assets` Bigtable table.
type AssetRepo struct {
	table    btTable
	idxTable btTable
}

// Ensure Bigtable implementation satisfies repository abstraction.
var _ repository.AssetRepository = (*AssetRepo)(nil)

func NewAssetRepo(c *Client) *AssetRepo {
	return &AssetRepo{
		table:    c.Table(TableAssets),
		idxTable: c.Table(TableIdxSegmentsByFile),
	}
}

// Get fetches a single asset by ID.
func (r *AssetRepo) Get(ctx context.Context, assetID string) (*models.Asset, error) {
	row, err := r.table.ReadRow(ctx, AssetKey(assetID),
		bigtable.RowFilter(bigtable.LatestNFilter(1)),
	)
	if err != nil {
		return nil, fmt.Errorf("AssetRepo.Get: %w", err)
	}
	if row == nil {
		return nil, nil
	}
	return rowToAsset(row), nil
}

// Set writes all columns of an asset (full overwrite / upsert).
func (r *AssetRepo) Set(ctx context.Context, a *models.Asset) error {
	now := time.Now()
	if a.CreatedAt.IsZero() {
		a.CreatedAt = now
	}
	a.UpdatedAt = now
	a.Version++

	mut := bigtable.NewMutation()

	ts := bigtable.Now()
	cf := CFMeta

	mut.Set(cf, "mcap_file_id", ts, B(a.McapFileID))
	mut.Set(cf, "start_timestamp_ns", ts, PackInt64(a.StartTimestampNs))
	mut.Set(cf, "end_timestamp_ns", ts, PackInt64(a.EndTimestampNs))
	mut.Set(cf, "duration_sec", ts, B(strconv.FormatFloat(a.DurationSec, 'f', 6, 64)))
	mut.Set(cf, "reviewer", ts, B(a.Reviewer))
	mut.Set(cf, "status", ts, B(string(a.Status)))
	mut.Set(cf, "owner", ts, B(a.Owner))
	mut.Set(cf, "type", ts, B(a.SegType))
	mut.Set(cf, "env", ts, B(a.Env))
	mut.Set(cf, "task", ts, B(a.Task))
	mut.Set(cf, "delivery_count", ts, PackInt64(int64(a.DeliveryCount)))
	mut.Set(cf, "last_delivered_to", ts, B(a.LastDeliveredTo))
	if a.LastDeliveredAt != nil {
		mut.Set(cf, "last_delivered_at", ts, RFC3339(*a.LastDeliveredAt))
	}
	mut.Set(cf, "created_at", ts, RFC3339(a.CreatedAt))
	mut.Set(cf, "updated_at", ts, RFC3339(a.UpdatedAt))
	mut.Set(cf, "version", ts, PackInt64(a.Version))

	for k, v := range a.AlgoResults {
		mut.Set(CFAlgo, k, ts, B(v))
	}
	for k, v := range a.Tags {
		mut.Set(CFTag, k, ts, B(v))
	}

	if err := r.table.Apply(ctx, AssetKey(a.AssetID), mut); err != nil {
		return fmt.Errorf("AssetRepo.Set: %w", err)
	}
	return nil
}

// WriteSegmentIndex writes one row to idx_segments_by_file.
// Row key pattern: <mcap_file_id>#<start_timestamp_ns>#<asset_id>.
func (r *AssetRepo) WriteSegmentIndex(ctx context.Context, a *models.Asset) error {
	mut := bigtable.NewMutation()
	mut.Set(CFRef, "asset_id", bigtable.Now(), B(a.AssetID))
	if err := r.idxTable.Apply(ctx, IdxSegmentKey(a.McapFileID, a.StartTimestampNs, a.AssetID), mut); err != nil {
		return fmt.Errorf("AssetRepo.WriteSegmentIndex: %w", err)
	}
	return nil
}

// SoftDelete marks an asset as archived (status=archived) without removing it.
func (r *AssetRepo) SoftDelete(ctx context.Context, assetID string) error {
	mut := bigtable.NewMutation()
	mut.Set(CFMeta, "status", bigtable.Now(), B(string(models.AssetStatusArchived)))
	mut.Set(CFMeta, "updated_at", bigtable.Now(), RFC3339(time.Now()))
	if err := r.table.Apply(ctx, AssetKey(assetID), mut); err != nil {
		return fmt.Errorf("AssetRepo.SoftDelete: %w", err)
	}
	return nil
}

// ListByMcapFile returns all assets for a given MCAP file via the secondary index.
func (r *AssetRepo) ListByMcapFile(ctx context.Context, mcapFileID string) ([]*models.Asset, error) {
	prefix := mcapFileID + "#"
	var assetIDs []string

	if err := r.idxTable.ReadRows(ctx, bigtable.PrefixRange(prefix), func(row bigtable.Row) bool {
		// row key: <mcap_file_id>#<start_timestamp_ns>#<asset_id>
		if assetID := ParseIdxSegmentAssetID(row.Key()); assetID != "" {
			assetIDs = append(assetIDs, assetID)
		}
		return true
	}, bigtable.RowFilter(bigtable.LatestNFilter(1))); err != nil {
		return nil, fmt.Errorf("AssetRepo.ListByMcapFile: read index: %w", err)
	}

	return r.GetBatch(ctx, assetIDs)
}

// GetBatch fetches multiple assets in a single RPC.
func (r *AssetRepo) GetBatch(ctx context.Context, assetIDs []string) ([]*models.Asset, error) {
	if len(assetIDs) == 0 {
		return nil, nil
	}

	rs := bigtable.RowList{}
	for _, id := range assetIDs {
		rs = append(rs, AssetKey(id))
	}

	var assets []*models.Asset
	if err := r.table.ReadRows(ctx, rs, func(row bigtable.Row) bool {
		assets = append(assets, rowToAsset(row))
		return true
	}, bigtable.RowFilter(bigtable.LatestNFilter(1))); err != nil {
		return nil, fmt.Errorf("AssetRepo.GetBatch: %w", err)
	}
	return assets, nil
}

// McapFileRepo provides read/write access to the `mcap_files` Bigtable table.
type McapFileRepo struct {
	table btTable
}

var _ repository.McapFileRepository = (*McapFileRepo)(nil)

func NewMcapFileRepo(c *Client) *McapFileRepo {
	return &McapFileRepo{table: c.Table(TableMcapFiles)}
}

// Get fetches a single McapFile by ID.
func (r *McapFileRepo) Get(ctx context.Context, mcapFileID string) (*models.McapFile, error) {
	row, err := r.table.ReadRow(ctx, McapFileKey(mcapFileID),
		bigtable.RowFilter(bigtable.LatestNFilter(1)),
	)
	if err != nil {
		return nil, fmt.Errorf("McapFileRepo.Get: %w", err)
	}
	if row == nil {
		return nil, nil
	}
	return rowToMcapFile(row), nil
}

// Set writes all columns of a McapFile.
func (r *McapFileRepo) Set(ctx context.Context, f *models.McapFile) error {
	now := time.Now()
	if f.CreatedAt.IsZero() {
		f.CreatedAt = now
	}
	f.UpdatedAt = now
	f.Version++

	ts := bigtable.Now()
	mut := bigtable.NewMutation()

	// Keep both qualifiers during transition:
	// - mcap_uri (canonical in sql.md)
	// - gcs_path (legacy API field)
	mut.Set(CFMeta, "mcap_uri", ts, B(f.GCSPath))
	mut.Set(CFMeta, "gcs_path", ts, B(f.GCSPath))
	mut.Set(CFMeta, "size_bytes", ts, PackInt64(f.SizeBytes))
	mut.Set(CFMeta, "raw_hash_md5", ts, B(f.RawHashMD5))
	mut.Set(CFMeta, "ingest_state", ts, B(string(f.IngestState)))
	mut.Set(CFMeta, "start_timestamp_ns", ts, PackInt64(f.StartTimestampNs))
	mut.Set(CFMeta, "end_timestamp_ns", ts, PackInt64(f.EndTimestampNs))
	mut.Set(CFMeta, "channel_count", ts, PackInt64(int64(f.ChannelCount)))
	mut.Set(CFMeta, "chunk_count", ts, PackInt64(int64(f.ChunkCount)))
	mut.Set(CFMeta, "owner", ts, B(f.Owner))
	mut.Set(CFMeta, "created_at", ts, RFC3339(f.CreatedAt))
	mut.Set(CFMeta, "updated_at", ts, RFC3339(f.UpdatedAt))
	mut.Set(CFMeta, "version", ts, PackInt64(f.Version))

	for k, v := range f.ProcessState {
		mut.Set(CFProcess, k, ts, B(v))
	}

	if err := r.table.Apply(ctx, McapFileKey(f.McapFileID), mut); err != nil {
		return fmt.Errorf("McapFileRepo.Set: %w", err)
	}
	return nil
}

// UpdateIngestState flips the ingest_state field.
func (r *McapFileRepo) UpdateIngestState(ctx context.Context, mcapFileID string, state models.IngestState) error {
	mut := bigtable.NewMutation()
	mut.Set(CFMeta, "ingest_state", bigtable.Now(), B(string(state)))
	mut.Set(CFMeta, "updated_at", bigtable.Now(), RFC3339(time.Now()))
	if err := r.table.Apply(ctx, McapFileKey(mcapFileID), mut); err != nil {
		return fmt.Errorf("McapFileRepo.UpdateIngestState: %w", err)
	}
	return nil
}

// DeliveryRepo provides read/write access to deliveries + secondary indexes.
type DeliveryRepo struct {
	table       btTable
	idxAsset    btTable
	idxCustomer btTable
}

var _ repository.DeliveryRepository = (*DeliveryRepo)(nil)

func NewDeliveryRepo(c *Client) *DeliveryRepo {
	return &DeliveryRepo{
		table:       c.Table(TableDeliveries),
		idxAsset:    c.Table(TableIdxAssetDeliveries),
		idxCustomer: c.Table(TableIdxCustomerDeliveries),
	}
}

// Get fetches a single delivery by ID.
func (r *DeliveryRepo) Get(ctx context.Context, deliveryID string) (*models.Delivery, error) {
	row, err := r.table.ReadRow(ctx, DeliveryKey(deliveryID),
		bigtable.RowFilter(bigtable.LatestNFilter(1)),
	)
	if err != nil {
		return nil, fmt.Errorf("DeliveryRepo.Get: %w", err)
	}
	if row == nil {
		return nil, nil
	}
	return rowToDelivery(row), nil
}

// Set writes all columns of a delivery.
func (r *DeliveryRepo) Set(ctx context.Context, d *models.Delivery) error {
	now := time.Now()
	if d.CreatedAt.IsZero() {
		d.CreatedAt = now
	}
	d.UpdatedAt = now
	d.Version++

	ts := bigtable.Now()
	mut := bigtable.NewMutation()

	mut.Set(CFMeta, "customer_id", ts, B(d.CustomerID))
	mut.Set(CFMeta, "status", ts, B(string(d.Status)))
	mut.Set(CFMeta, "manifest_uri", ts, B(d.ManifestURI))
	mut.Set(CFMeta, "contract_id", ts, B(d.ContractID))
	mut.Set(CFMeta, "note", ts, B(d.Note))
	mut.Set(CFMeta, "asset_count", ts, PackInt64(int64(d.AssetCount)))
	mut.Set(CFMeta, "owner", ts, B(d.Owner))
	mut.Set(CFMeta, "created_at", ts, RFC3339(d.CreatedAt))
	mut.Set(CFMeta, "updated_at", ts, RFC3339(d.UpdatedAt))
	mut.Set(CFMeta, "version", ts, PackInt64(d.Version))
	if d.DeliveredAt != nil {
		mut.Set(CFMeta, "delivered_at", ts, RFC3339(*d.DeliveredAt))
	}

	if err := r.table.Apply(ctx, DeliveryKey(d.DeliveryID), mut); err != nil {
		return fmt.Errorf("DeliveryRepo.Set: %w", err)
	}
	return nil
}

// WriteIndexes writes both secondary index rows for one (asset, delivery) pair.
func (r *DeliveryRepo) WriteIndexes(ctx context.Context, assetID string, d *models.Delivery) error {
	at := d.CreatedAt
	if d.DeliveredAt != nil {
		at = *d.DeliveredAt
	}

	refVal := B(d.DeliveryID) // index row just stores delivery_id as the value

	// idx_asset_deliveries: asset → deliveries
	mutA := bigtable.NewMutation()
	mutA.Set(CFRef, "delivery_id", bigtable.Now(), refVal)
	if err := r.idxAsset.Apply(ctx, IdxAssetDeliveryKey(assetID, at, d.DeliveryID), mutA); err != nil {
		return fmt.Errorf("DeliveryRepo.WriteIndexes: asset idx: %w", err)
	}

	// idx_customer_deliveries: customer → deliveries
	mutC := bigtable.NewMutation()
	mutC.Set(CFRef, "delivery_id", bigtable.Now(), refVal)
	if err := r.idxCustomer.Apply(ctx, IdxCustomerDeliveryKey(d.CustomerID, at, d.DeliveryID), mutC); err != nil {
		return fmt.Errorf("DeliveryRepo.WriteIndexes: customer idx: %w", err)
	}

	return nil
}

// ListByAsset returns all delivery IDs for a given asset (via secondary index).
func (r *DeliveryRepo) ListByAsset(ctx context.Context, assetID string) ([]string, error) {
	prefix := assetID + "#"
	var ids []string
	if err := r.idxAsset.ReadRows(ctx, bigtable.PrefixRange(prefix), func(row bigtable.Row) bool {
		for _, col := range row[CFRef] {
			ids = append(ids, S(col.Value))
		}
		return true
	}, bigtable.RowFilter(bigtable.LatestNFilter(1))); err != nil {
		return nil, fmt.Errorf("DeliveryRepo.ListByAsset: %w", err)
	}
	return ids, nil
}

// ListByCustomer returns all delivery IDs for a given customer (via secondary index).
func (r *DeliveryRepo) ListByCustomer(ctx context.Context, customerID string) ([]string, error) {
	prefix := customerID + "#"
	var ids []string
	if err := r.idxCustomer.ReadRows(ctx, bigtable.PrefixRange(prefix), func(row bigtable.Row) bool {
		for _, col := range row[CFRef] {
			ids = append(ids, S(col.Value))
		}
		return true
	}, bigtable.RowFilter(bigtable.LatestNFilter(1))); err != nil {
		return nil, fmt.Errorf("DeliveryRepo.ListByCustomer: %w", err)
	}
	return ids, nil
}

type IdempotencyRepo struct {
	table btTable
}

var _ repository.IdempotencyRepository = (*IdempotencyRepo)(nil)

func NewIdempotencyRepo(c *Client) *IdempotencyRepo {
	return &IdempotencyRepo{table: c.Table(TableIdempotencyKeys)}
}

func idemRowKey(scope, key string) string {
	return RowKeyPrefix + scope + "#" + key
}

// Get returns stored record if present; nil,nil when not found.
func (r *IdempotencyRepo) Get(ctx context.Context, scope, key string) (*repository.IdempotencyRecord, error) {
	row, err := r.table.ReadRow(ctx, idemRowKey(scope, key), bigtable.RowFilter(bigtable.LatestNFilter(1)))
	if err != nil {
		return nil, fmt.Errorf("IdempotencyRepo.Get: %w", err)
	}
	if row == nil {
		return nil, nil
	}
	rec := &repository.IdempotencyRecord{Scope: scope, Key: key}
	for _, col := range row[CFIdem] {
		qual := col.Column[len(CFIdem)+1:]
		switch qual {
		case "request_hash":
			rec.RequestHash = S(col.Value)
		case "status_code":
			v, _ := strconv.Atoi(S(col.Value))
			rec.StatusCode = v
		case "response_json":
			rec.Response = append([]byte(nil), col.Value...)
		case "created_at":
			rec.CreatedAt = ParseRFC3339(col.Value)
		}
	}
	return rec, nil
}

func (r *IdempotencyRepo) Save(ctx context.Context, rec *repository.IdempotencyRecord) error {
	mut := bigtable.NewMutation()
	ts := bigtable.Now()
	mut.Set(CFIdem, "request_hash", ts, B(rec.RequestHash))
	mut.Set(CFIdem, "status_code", ts, B(strconv.Itoa(rec.StatusCode)))
	mut.Set(CFIdem, "response_json", ts, rec.Response)
	mut.Set(CFIdem, "created_at", ts, RFC3339(time.Now()))
	if err := r.table.Apply(ctx, idemRowKey(rec.Scope, rec.Key), mut); err != nil {
		return fmt.Errorf("IdempotencyRepo.Save: %w", err)
	}
	return nil
}

// Row → model conversion helpers.
func rowToAsset(row bigtable.Row) *models.Asset {
	a := &models.Asset{
		AssetID:     rowKeyID(row.Key()),
		AlgoResults: map[string]string{},
		Tags:        map[string]string{},
	}

	for _, col := range row[CFMeta] {
		qual := col.Column[len(CFMeta)+1:] // strip "cf:meta:"
		switch qual {
		case "mcap_file_id":
			a.McapFileID = S(col.Value)
		case "start_timestamp_ns":
			a.StartTimestampNs = UnpackInt64(col.Value)
		case "end_timestamp_ns":
			a.EndTimestampNs = UnpackInt64(col.Value)
		case "duration_sec":
			a.DurationSec, _ = strconv.ParseFloat(S(col.Value), 64)
		case "reviewer":
			a.Reviewer = S(col.Value)
		case "status":
			a.Status = models.AssetStatus(S(col.Value))
		case "owner":
			a.Owner = S(col.Value)
		case "type":
			a.SegType = S(col.Value)
		case "env":
			a.Env = S(col.Value)
		case "task":
			a.Task = S(col.Value)
		case "delivery_count":
			a.DeliveryCount = int(UnpackInt64(col.Value))
		case "last_delivered_to":
			a.LastDeliveredTo = S(col.Value)
		case "last_delivered_at":
			t := ParseRFC3339(col.Value)
			a.LastDeliveredAt = &t
		case "created_at":
			a.CreatedAt = ParseRFC3339(col.Value)
		case "updated_at":
			a.UpdatedAt = ParseRFC3339(col.Value)
		case "version":
			a.Version = UnpackInt64(col.Value)
		}
	}

	for _, col := range row[CFAlgo] {
		qual := col.Column[len(CFAlgo)+1:]
		a.AlgoResults[qual] = S(col.Value)
	}

	for _, col := range row[CFTag] {
		qual := col.Column[len(CFTag)+1:]
		a.Tags[qual] = S(col.Value)
	}

	return a
}

func rowToMcapFile(row bigtable.Row) *models.McapFile {
	f := &models.McapFile{
		McapFileID:   rowKeyID(row.Key()),
		ProcessState: map[string]string{},
	}

	for _, col := range row[CFMeta] {
		qual := col.Column[len(CFMeta)+1:]
		switch qual {
		case "mcap_uri":
			f.GCSPath = S(col.Value)
		case "gcs_path":
			// Backward compatibility: only apply if canonical field is absent.
			if f.GCSPath == "" {
				f.GCSPath = S(col.Value)
			}
		case "size_bytes":
			f.SizeBytes = UnpackInt64(col.Value)
		case "raw_hash_md5":
			f.RawHashMD5 = S(col.Value)
		case "ingest_state":
			f.IngestState = models.IngestState(S(col.Value))
		case "start_timestamp_ns":
			f.StartTimestampNs = UnpackInt64(col.Value)
		case "end_timestamp_ns":
			f.EndTimestampNs = UnpackInt64(col.Value)
		case "channel_count":
			f.ChannelCount = int(UnpackInt64(col.Value))
		case "chunk_count":
			f.ChunkCount = int(UnpackInt64(col.Value))
		case "owner":
			f.Owner = S(col.Value)
		case "created_at":
			f.CreatedAt = ParseRFC3339(col.Value)
		case "updated_at":
			f.UpdatedAt = ParseRFC3339(col.Value)
		case "version":
			f.Version = UnpackInt64(col.Value)
		}
	}

	for _, col := range row[CFProcess] {
		qual := col.Column[len(CFProcess)+1:]
		f.ProcessState[qual] = strconv.Quote(S(col.Value))
	}

	return f
}

func rowToDelivery(row bigtable.Row) *models.Delivery {
	d := &models.Delivery{DeliveryID: rowKeyID(row.Key())}
	for _, col := range row[CFMeta] {
		qual := col.Column[len(CFMeta)+1:]
		switch qual {
		case "customer_id":
			d.CustomerID = S(col.Value)
		case "status":
			d.Status = models.DeliveryStatus(S(col.Value))
		case "manifest_uri":
			d.ManifestURI = S(col.Value)
		case "contract_id":
			d.ContractID = S(col.Value)
		case "note":
			d.Note = S(col.Value)
		case "asset_count":
			d.AssetCount = int(UnpackInt64(col.Value))
		case "owner":
			d.Owner = S(col.Value)
		case "delivered_at":
			t := ParseRFC3339(col.Value)
			d.DeliveredAt = &t
		case "created_at":
			d.CreatedAt = ParseRFC3339(col.Value)
		case "updated_at":
			d.UpdatedAt = ParseRFC3339(col.Value)
		case "version":
			d.Version = UnpackInt64(col.Value)
		}
	}
	return d
}

// rowKeyID strips the "v1#" prefix from a row key.
func rowKeyID(key string) string {
	if len(key) > 3 {
		return key[3:]
	}
	return key
}
