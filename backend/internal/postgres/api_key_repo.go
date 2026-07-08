package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

// APIKeyRepo persists API keys.
type APIKeyRepo struct {
	c *Client
}

// NewAPIKeyRepo creates an APIKeyRepo.
func NewAPIKeyRepo(c *Client) *APIKeyRepo { return &APIKeyRepo{c: c} }

var _ repository.APIKeyRepository = (*APIKeyRepo)(nil)

const apiKeySelectCols = `id, key_prefix, secret_hash, name, owner, scopes, status, expires_at, last_used_at, created_by, created_at` // pragma: allowlist secret

func scanAPIKey(rs rowScanner) (*models.APIKey, error) {
	var k models.APIKey
	if err := rs.Scan(
		&k.ID, &k.KeyPrefix, &k.SecretHash, &k.Name, &k.Owner, &k.Scopes,
		&k.Status, &k.ExpiresAt, &k.LastUsedAt, &k.CreatedBy, &k.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &k, nil
}

func (r *APIKeyRepo) Create(ctx context.Context, k *models.APIKey) error {
	if k.ID == "" {
		k.ID = uuid.New().String()
	}
	if k.Status == "" {
		k.Status = "active"
	}
	if k.Scopes == nil {
		k.Scopes = []string{}
	}
	q := `INSERT INTO api_keys
	  (id, key_prefix, secret_hash, name, owner, scopes, status, expires_at, created_by)
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`
	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, q,
		k.ID, k.KeyPrefix, k.SecretHash, k.Name, k.Owner, k.Scopes, k.Status, k.ExpiresAt, k.CreatedBy,
	); err != nil {
		return fmt.Errorf("postgres APIKeyRepo.Create: %w", err)
	}
	return nil
}

func (r *APIKeyRepo) FindByPrefix(ctx context.Context, prefix string) (*models.APIKey, error) {
	q := `SELECT ` + apiKeySelectCols + ` FROM api_keys WHERE key_prefix = $1`
	db := dbFromCtx(ctx, r.c.db)
	k, err := scanAPIKey(db.QueryRow(ctx, q, prefix))
	if err != nil {
		if errors.Is(err, errNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres APIKeyRepo.FindByPrefix: %w", err)
	}
	return k, nil
}

func (r *APIKeyRepo) List(ctx context.Context) ([]models.APIKey, error) {
	q := `SELECT ` + apiKeySelectCols + ` FROM api_keys ORDER BY created_at DESC`
	db := dbFromCtx(ctx, r.c.db)
	rows, err := db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("postgres APIKeyRepo.List: %w", err)
	}
	defer rows.Close()
	var out []models.APIKey
	for rows.Next() {
		k, err := scanAPIKey(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres APIKeyRepo.List scan: %w", err)
		}
		out = append(out, *k)
	}
	return out, rows.Err()
}

func (r *APIKeyRepo) Revoke(ctx context.Context, id string) error {
	q := `UPDATE api_keys SET status = 'revoked' WHERE id = $1`
	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, q, id); err != nil {
		return fmt.Errorf("postgres APIKeyRepo.Revoke: %w", err)
	}
	return nil
}

func (r *APIKeyRepo) TouchLastUsed(ctx context.Context, id string) error {
	q := `UPDATE api_keys SET last_used_at = now() WHERE id = $1`
	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, q, id); err != nil {
		return fmt.Errorf("postgres APIKeyRepo.TouchLastUsed: %w", err)
	}
	return nil
}
