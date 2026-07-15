package repository

import (
	"context"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// APIKeyRepository persists API keys used by SDK / API callers.
type APIKeyRepository interface {
	Create(ctx context.Context, k *models.APIKey) error
	// FindByPrefix returns the active-or-revoked key row for a prefix, or
	// (nil, nil) if none. Callers verify the secret + status/expiry.
	FindByPrefix(ctx context.Context, prefix string) (*models.APIKey, error)
	List(ctx context.Context) ([]models.APIKey, error)
	Revoke(ctx context.Context, id string) error
	TouchLastUsed(ctx context.Context, id string) error
}
