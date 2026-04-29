package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"data-platform/internal/models"
)

// ErrOptimisticLock is returned when an optimistic lock conflict is detected
// (i.e., the expected version does not match the current version in the database).
var ErrOptimisticLock = errors.New("optimistic lock conflict: version mismatch")

// TxRunner runs the provided function inside a single transaction. Repos
// dispatched within fn should be tx-aware (read tx from ctx) so that all
// writes commit or roll back atomically. Required for outbox correctness:
// a business-state write and its asset_events append must land in the same
// transaction so consumers never see the projection without the event or
// vice versa.
type TxRunner interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// McapFileRepository defines persistence operations for mcap file metadata.
type McapFileRepository interface {
	Get(ctx context.Context, mcapFileID string) (*models.McapFile, error)
	Set(ctx context.Context, f *models.McapFile) error
	UpdateIngestState(ctx context.Context, mcapFileID string, state models.IngestState) error
	List(ctx context.Context, page, pageSize int, ingestState, owner string) ([]*models.McapFile, int64, error)
}

// DeliveryRepository defines persistence operations for delivery records/indexes.
type DeliveryRepository interface {
	Set(ctx context.Context, d *models.Delivery) error
	Get(ctx context.Context, deliveryID string) (*models.Delivery, error)
	WriteIndexes(ctx context.Context, assetID string, d *models.Delivery) error
	ListByCustomer(ctx context.Context, customerID string) ([]string, error)
	ListByAsset(ctx context.Context, assetID string) ([]string, error)

	// List returns a paginated list of deliveries, optionally filtered by status.
	// status may be empty to return all deliveries.
	List(ctx context.Context, page, pageSize int, status string) ([]*models.Delivery, int64, error)
}

// IdempotencyRecord stores one idempotent request result.
type IdempotencyRecord struct {
	Scope       string
	Key         string
	RequestHash string
	StatusCode  int
	Response    json.RawMessage
	CreatedAt   time.Time
}

// IdempotencyRepository persists idempotency keys/results.
type IdempotencyRepository interface {
	Get(ctx context.Context, scope, key string) (*IdempotencyRecord, error)
	Save(ctx context.Context, rec *IdempotencyRecord) error
}

// AssetTagRepository defines persistence operations for the asset_tags
// projection table.
type AssetTagRepository interface {
	Upsert(ctx context.Context, assetID, tagKey, tagValue, tagType, sourceType string) error
	ListByAsset(ctx context.Context, assetID string) ([]*models.AssetTag, error)
	Delete(ctx context.Context, assetID, tagKey string) error
}

// AssetAlgoLatestRepository is the source-of-truth store for per-algorithm
// state on an asset. The semantics are:
//
//   - Upsert is monotonic on algo_version: a write whose algo_version is
//     older than the persisted one is discarded silently (returns nil and
//     leaves the row untouched). This makes concurrent finishes from
//     different runs safe without holding any lock on `assets`.
//   - GetByAlgo returns nil, nil when the row does not exist (i.e. the
//     algorithm has never been run on this asset).
//   - Methods are tx-aware via context: when invoked inside Client.WithTx
//     they execute on the transaction; otherwise on the pool.
type AssetAlgoLatestRepository interface {
	Upsert(ctx context.Context, row *models.AssetAlgoLatest) error
	GetByAlgo(ctx context.Context, assetID, algoName string) (*models.AssetAlgoLatest, error)
	ListByAsset(ctx context.Context, assetID string) ([]*models.AssetAlgoLatest, error)
}

// AssetEventAppendInput captures the fields a producer can populate when
// appending to the asset_events outbox. event_id, event_seq, occurred_at,
// created_at and publish_state are assigned by the database.
type AssetEventAppendInput struct {
	EventType            string
	AggregateType        string // defaults to "asset" when empty; routing key for downstream sinks
	PayloadSchemaVersion string // defaults to "v1" when empty
	AssetID              string // empty → SQL NULL
	McapFileID           string // empty → SQL NULL
	TenantID             string
	ProjectID            string
	EventSource          string // defaults to "backend" when empty
	ActorType            string
	ActorID              string
	RequestID            string
	IdempotencyKey       string
	RunID                string
	EventPayload         []byte // raw JSON; nil → '{}'
}

// AssetEventRepository persists rows to the asset_events outbox table.
//
// Strong invariants:
//
//   - Every business state mutation must Append exactly one event in the
//     SAME transaction as the state write (outbox pattern). Consumers rely
//     on this for replayability and at-least-once delivery.
//   - Append is tx-aware via context.
//   - ListPending is consumed by the Outbox Worker; rows return in
//     ascending event_seq order.
type AssetEventRepository interface {
	Append(ctx context.Context, in AssetEventAppendInput) error
	ListPending(ctx context.Context, limit int) ([]*models.AssetEvent, error)
	ListByAsset(ctx context.Context, assetID string, eventTypes []string, limit int) ([]*models.AssetEvent, error)
}
