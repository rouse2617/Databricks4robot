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
