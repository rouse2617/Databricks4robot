package bigtable

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"strings"
	"time"

	"cloud.google.com/go/bigtable"
)

const (
	// Canonical table names align with the external data4cyber schema doc.
	// Use repository-relative references in docs (no machine-local absolute paths).
	TableAssets                = "assets"
	TableMcapFiles             = "mcap_files"
	TableDeliveries            = "deliveries"
	TableIdxSegmentsByFile     = "idx_segments_by_file"
	TableIdxAssetDeliveries    = "idx_asset_deliveries"
	TableIdxCustomerDeliveries = "idx_customer_deliveries"
	TableIdempotencyKeys       = "idempotency_keys"

	CFMeta    = "cf:meta"
	CFAlgo    = "cf:algo"
	CFTag     = "cf:tag"
	CFProcess = "cf:process"
	CFRef     = "ref"
	CFIdem    = "meta"

	RowKeyPrefix = "v1#"
)

type btTable interface {
	ReadRow(ctx context.Context, row string, opts ...bigtable.ReadOption) (bigtable.Row, error)
	Apply(ctx context.Context, row string, m *bigtable.Mutation, opts ...bigtable.ApplyOption) error
	ReadRows(ctx context.Context, arg bigtable.RowSet, f func(bigtable.Row) bool, opts ...bigtable.ReadOption) error
}

type btDataClient interface {
	Open(name string) btTable
	Close() error
}

type realDataClient struct {
	inner *bigtable.Client
}

func (c *realDataClient) Open(name string) btTable { return &realTable{inner: c.inner.Open(name)} }
func (c *realDataClient) Close() error             { return c.inner.Close() }

type realTable struct {
	inner *bigtable.Table
}

func (t *realTable) ReadRow(ctx context.Context, row string, opts ...bigtable.ReadOption) (bigtable.Row, error) {
	return t.inner.ReadRow(ctx, row, opts...)
}
func (t *realTable) Apply(ctx context.Context, row string, m *bigtable.Mutation, opts ...bigtable.ApplyOption) error {
	return t.inner.Apply(ctx, row, m, opts...)
}
func (t *realTable) ReadRows(ctx context.Context, arg bigtable.RowSet, f func(bigtable.Row) bool, opts ...bigtable.ReadOption) error {
	return t.inner.ReadRows(ctx, arg, f, opts...)
}

var newDataClient = func(ctx context.Context, project, instance string) (btDataClient, error) {
	c, err := bigtable.NewClient(ctx, project, instance)
	if err != nil {
		return nil, err
	}
	return &realDataClient{inner: c}, nil
}

// Client wraps the Bigtable data client.
type Client struct {
	inner    btDataClient
	project  string
	instance string
}

func New(ctx context.Context, project, instance string) (*Client, error) {
	c, err := newDataClient(ctx, project, instance)
	if err != nil {
		return nil, fmt.Errorf("bigtable.New: %w", err)
	}
	return &Client{inner: c, project: project, instance: instance}, nil
}

func (c *Client) Table(name string) btTable {
	return c.inner.Open(name)
}

func (c *Client) Close() error {
	return c.inner.Close()
}

// ──────────────────────────────────────────────────────────────────────────────
// Row key helpers
// ──────────────────────────────────────────────────────────────────────────────

// AssetKey returns the Bigtable row key for an asset.
func AssetKey(assetID string) string { return RowKeyPrefix + assetID }

// McapFileKey returns the row key for a mcap_file row.
func McapFileKey(mcapFileID string) string { return RowKeyPrefix + mcapFileID }

// DeliveryKey returns the row key for a delivery row.
func DeliveryKey(deliveryID string) string { return RowKeyPrefix + deliveryID }

// IdxSegmentKey returns the secondary-index row key for idx_segments_by_file.
// Pattern (sql.md): <mcap_file_id>#<start_timestamp_ns>#<asset_id>
// start_timestamp_ns uses zero-padded decimal to keep lexicographic ordering.
func IdxSegmentKey(mcapFileID string, startNs int64, assetID string) string {
	return fmt.Sprintf("%s#%020d#%s", mcapFileID, startNs, assetID)
}

// IdxAssetDeliveryKey returns the secondary-index row key for idx_asset_deliveries.
// Pattern (sql.md): <asset_id>#<delivered_at_rev>#<delivery_id>
// delivered_at_rev = MaxInt64 - delivered_at_ms.
func IdxAssetDeliveryKey(assetID string, deliveredAt time.Time, deliveryID string) string {
	rev := math.MaxInt64 - deliveredAt.UnixMilli()
	return fmt.Sprintf("%s#%020d#%s", assetID, rev, deliveryID)
}

// IdxCustomerDeliveryKey returns the secondary-index row key for idx_customer_deliveries.
// Pattern (sql.md): <customer_id>#<delivered_at_rev>#<delivery_id>
// delivered_at_rev = MaxInt64 - delivered_at_ms.
func IdxCustomerDeliveryKey(customerID string, deliveredAt time.Time, deliveryID string) string {
	rev := math.MaxInt64 - deliveredAt.UnixMilli()
	return fmt.Sprintf("%s#%020d#%s", customerID, rev, deliveryID)
}

// ParseIdxSegmentAssetID extracts asset_id from:
// <mcap_file_id>#<start_timestamp_ns>#<asset_id>
func ParseIdxSegmentAssetID(key string) string {
	idx := strings.LastIndex(key, "#")
	if idx < 0 || idx+1 >= len(key) {
		return ""
	}
	return key[idx+1:]
}

// ──────────────────────────────────────────────────────────────────────────────
// Encoding helpers
// ──────────────────────────────────────────────────────────────────────────────

// PackInt64 encodes an int64 as big-endian bytes (Bigtable canonical form).
func PackInt64(v int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

// UnpackInt64 decodes big-endian bytes back to int64.
func UnpackInt64(b []byte) int64 {
	return int64(binary.BigEndian.Uint64(b))
}

// B converts a string to bytes for Bigtable cell values.
func B(s string) []byte { return []byte(s) }

// S converts Bigtable cell bytes to string.
func S(b []byte) string { return string(b) }

// RFC3339 formats a time for storage.
func RFC3339(t time.Time) []byte { return B(t.UTC().Format(time.RFC3339Nano)) }

// ParseRFC3339 parses a stored time string, returning zero time on error.
func ParseRFC3339(b []byte) time.Time {
	t, _ := time.Parse(time.RFC3339Nano, S(b))
	return t
}
