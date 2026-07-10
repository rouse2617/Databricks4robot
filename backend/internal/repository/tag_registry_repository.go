package repository

import (
	"context"
	"errors"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// ErrDuplicateTagKey is returned when creating a tag definition whose key
// already exists in the registry (CYB-3246 Phase 2).
var ErrDuplicateTagKey = errors.New("duplicate tag key")

// TagRegistryRepository persists managed tag definitions. The validation hot
// path does NOT go through this repo — it reads an in-memory map on
// config.TagRegistry; this repo is used at startup (seed + load) and by the
// admin CRUD API, which refreshes the in-memory map after each write.
type TagRegistryRepository interface {
	// Count returns the number of stored definitions (used to decide seeding).
	Count(ctx context.Context) (int, error)
	// List returns all definitions ordered by key.
	List(ctx context.Context) ([]*models.TagRegistryEntry, error)
	// Get returns one definition, or (nil, nil) when the key does not exist.
	Get(ctx context.Context, key string) (*models.TagRegistryEntry, error)
	// Create inserts a new definition, returning ErrDuplicateTagKey on conflict.
	Create(ctx context.Context, e *models.TagRegistryEntry) (*models.TagRegistryEntry, error)
	// Update mutates an existing definition, returning (nil, nil) when absent.
	Update(ctx context.Context, e *models.TagRegistryEntry) (*models.TagRegistryEntry, error)
	// Delete removes a definition; deleting an absent key is a no-op.
	Delete(ctx context.Context, key string) error
}
