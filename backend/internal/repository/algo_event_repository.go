package repository

import (
	"context"

	"data-platform/internal/models"
)

// AlgoEventRepository defines persistence operations for algorithm lifecycle events.
type AlgoEventRepository interface {
	// Insert persists a single algorithm status-change event.
	Insert(ctx context.Context, event *models.AlgoEvent) error

	// ListByAsset returns all events for the given asset, ordered by created_at DESC.
	// When algoKey is non-nil, only events matching that algo_key are returned.
	ListByAsset(ctx context.Context, assetID string, algoKey *string) ([]*models.AlgoEvent, error)
}
