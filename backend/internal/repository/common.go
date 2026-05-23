package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// ErrOptimisticLock is returned when an optimistic lock conflict is detected
// (i.e., the expected version does not match the current version in the database).
var ErrOptimisticLock = errors.New("optimistic lock conflict: version mismatch")

// ErrSchemaMismatch is returned when runtime SQL expects a schema shape that the
// connected database does not currently satisfy (usually missed migrations).
var ErrSchemaMismatch = errors.New("database schema mismatch")

// TxRunner runs the provided function inside a single transaction. Repos
// dispatched within fn should be tx-aware (read tx from ctx) so that all
// writes commit or roll back atomically. Required for event-stream correctness:
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
	AddItems(ctx context.Context, deliveryID string, assetIDs []string) error
	RefreshAssetDeliveryIndex(ctx context.Context, assetID string) error
	ListByCustomer(ctx context.Context, customerID string) ([]string, error)
	ListByAsset(ctx context.Context, assetID string) ([]string, error)
	ListItems(ctx context.Context, deliveryID string) ([]*models.DeliveryItem, error)

	// List returns a paginated list of deliveries, optionally filtered by status and customer_id.
	// Empty status or customerID means no filter on that dimension.
	List(ctx context.Context, page, pageSize int, status, customerID string) ([]*models.Delivery, int64, error)

	// Update persists changes to an existing delivery using optimistic locking.
	// Returns ErrOptimisticLock when expectedRowVersion does not match the current version.
	Update(ctx context.Context, d *models.Delivery, expectedRowVersion int64) error
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

// AssetTagUpsertInput captures the full identity + payload of a single
// `asset_tags` write. As of CYB-1015 the row identity is
// (asset_id, tag_key, tag_value, source_type, source_version) — multiple
// sources may coexist on the same (asset_id, tag_key).
type AssetTagUpsertInput struct {
	AssetID       string
	TagKey        string
	TagValue      string
	TagType       string
	SourceType    string
	SourceName    string
	SourceVersion string
	RunID         string
	TenantID      string
	ProjectID     string
}

// AssetTagRepository defines persistence operations for the asset_tags
// projection table.
type AssetTagRepository interface {
	// Upsert inserts or refreshes a single tag assertion. Idempotent on
	// (asset_id, tag_key, tag_value, source_type, source_version).
	Upsert(ctx context.Context, in AssetTagUpsertInput) error
	ListByAsset(ctx context.Context, assetID string) ([]*models.AssetTag, error)
	// Delete removes tag rows for (assetID, tagKey). When sourceType is the
	// empty string all sources for the key are removed; otherwise only the
	// matching source is deleted.
	Delete(ctx context.Context, assetID, tagKey, sourceType string) error
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
// appending to asset_events. event_id, event_seq, occurred_at,
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

// AssetEventListOptions defines filters for querying an asset event stream.
// The HTTP API uses DESC order (newest first) and paginates by event_seq.
type AssetEventListOptions struct {
	EventTypes        []string
	EventTypePatterns []string
	AlgoKey           string
	BeforeEventSeq    *int64
	AfterEventSeq     *int64
	StartTime         *time.Time
	EndTime           *time.Time
	Limit             int
}

// AssetEventRepository persists rows to the asset_events table.
//
// Strong invariants:
//
//   - Every business state mutation must Append exactly one event in the
//     SAME transaction as the state write (transactional outbox).
//   - Append is tx-aware via context.
//   - A relay publishes pending rows to Pub/Sub; the search projector consumes
//     those messages and rebuilds Elasticsearch from current-state tables.
type AssetEventRepository interface {
	Append(ctx context.Context, in AssetEventAppendInput) error
	ListPending(ctx context.Context, limit int) ([]*models.AssetEvent, error)
	// ListPendingSafe returns pending rows with occurred_at older than now()-safetyLag,
	// ordered by event_seq (outbox relay publishing horizon).
	ListPendingSafe(ctx context.Context, safetyLag time.Duration, limit int) ([]*models.AssetEvent, error)
	ListByAsset(ctx context.Context, assetID string, opts AssetEventListOptions) ([]*models.AssetEvent, error)
	// ListVersionPromotedByLogical returns version_promoted events for all revisions in a family.
	ListVersionPromotedByLogical(ctx context.Context, logicalAssetID string) ([]*models.AssetEvent, error)
	// ListGlobal returns recent events across all assets, ordered by event_seq DESC.
	// Used by the global events page (/events) for operational visibility.
	ListGlobal(ctx context.Context, opts AssetEventListOptions) ([]*models.AssetEvent, error)
	// MarkPublished sets publish_state='published' and published_at=now() for the given monotonic event_seq values.
	MarkPublished(ctx context.Context, eventSeqs []int64) error
	// MarkFailed increments retry_count and stores last_error; rows stay pending for retry (or manual fix when retry_count is high).
	MarkFailed(ctx context.Context, eventSeq int64, errMsg string) error
	// CountPending returns the number of rows still awaiting sink delivery.
	CountPending(ctx context.Context) (int64, error)
	// CountPendingClaimable counts pending rows that are old enough for the relay
	// to claim (publish_state='pending' AND occurred_at < now()-safetyLag), matching
	// the pending visibility branch in ClaimPendingSafe.
	CountPendingClaimable(ctx context.Context, safetyLag time.Duration) (int64, error)
	// CountProcessing counts rows currently claimed by a relay (publish_state='processing').
	CountProcessing(ctx context.Context) (int64, error)
	// OldestPendingAge returns the age of the oldest pending event as
	// seconds since its occurred_at timestamp. Returns 0 when no pending
	// events exist.
	OldestPendingAge(ctx context.Context) (float64, error)
	// PublishStateCounts returns COUNT(*) grouped by publish_state for asset_events.
	PublishStateCounts(ctx context.Context) (map[string]int64, error)

	// ListBetweenSeq returns rows with lowerExclusive < event_seq <= upperInclusive,
	// ordered by event_seq ASC, for Iceberg Bronze staging.
	ListBetweenSeq(ctx context.Context, lowerExclusive, upperInclusive int64, limit int) ([]*models.AssetEvent, error)
}

// OutboxDLQRepository persists permanently failed events to the dead letter queue.
type OutboxDLQRepository interface {
	// MoveToDLQ moves events that have exceeded the retry threshold from
	// asset_events to outbox_dlq. Returns the number of events moved.
	MoveToDLQ(ctx context.Context, retryThreshold int) (int64, error)
	// Count returns rows currently stored in outbox_dlq (archived failures).
	Count(ctx context.Context) (int64, error)
}

// AssetUsageStatRepository persists per-asset engagement counters (CYB-1094).
type AssetUsageStatRepository interface {
	// RecordView increments view_count and sets last_viewed_at (CYB-1095).
	// Idempotent on asset_id — upserts when no row exists.
	RecordView(ctx context.Context, assetID string) error
	// ToggleFavorite increments or decrements favorite_count and returns the
	// new count (CYB-1096). Returns (newCount, nil).
	ToggleFavorite(ctx context.Context, assetID string) (int, error)
	// GetByAsset returns the usage stats for a given asset, or nil when no row exists.
	GetByAsset(ctx context.Context, assetID string) (*models.AssetUsageStat, error)
}
