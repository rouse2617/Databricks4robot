package repository

import (
	"context"

	"data-platform/internal/models"
)

// ActionListOptions filters the seg-scoped action list.
//
// Semantics:
//   - PointAtNs (when non-nil) returns actions whose [start_ns, end_ns) covers
//     the timestamp.
//   - FromNs / ToNs (when non-nil) limit results to actions whose interval
//     overlaps [from, to). Either bound may be nil.
//   - Label (when non-empty) matches when primary_label == Label OR Label is
//     present in labels[].
//   - Limit defaults to 200 in implementations when 0.
type ActionListOptions struct {
	PointAtNs *int64
	FromNs    *int64
	ToNs      *int64
	Label     string
	Limit     int
}

// ActionRepository owns persistence for the `actions` table — seg-internal
// time-bounded annotations (mcap → seg → action).
//
// Methods are tx-aware via context; when invoked inside Client.WithTx they
// execute on the transaction so the caller can append a matching asset_events
// row in the same atomic write.
type ActionRepository interface {
	// Insert persists a new action row (action_id is server-assigned when empty).
	// On UNIQUE conflict against (asset_id, source_name, external_id) returns
	// ErrOptimisticLock so callers can choose to upsert or surface the conflict.
	Insert(ctx context.Context, a *models.Action) error

	// Get returns the action by id, or (nil, nil) when not found / soft-deleted.
	Get(ctx context.Context, actionID string) (*models.Action, error)

	// ListByAsset returns actions for a seg matching opts, ordered by
	// (start_ns ASC, action_id ASC) for stable pagination.
	ListByAsset(ctx context.Context, assetID string, opts ActionListOptions) ([]*models.Action, error)
}
