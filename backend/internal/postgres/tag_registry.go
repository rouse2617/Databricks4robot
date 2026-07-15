package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

// TagRegistryRepo persists managed tag definitions (CYB-3246 Phase 2).
type TagRegistryRepo struct {
	c *Client
}

func NewTagRegistryRepo(c *Client) *TagRegistryRepo { return &TagRegistryRepo{c: c} }

const tagRegistryCols = `"key", description, type, "values", max_length, propagation, created_by, created_at, updated_at`

func scanTagRegistryEntry(s rowScanner) (*models.TagRegistryEntry, error) {
	var e models.TagRegistryEntry
	if err := s.Scan(
		&e.Key,
		&e.Description,
		&e.Type,
		&e.Values,
		&e.MaxLength,
		&e.Propagation,
		&e.CreatedBy,
		&e.CreatedAt,
		&e.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if e.Values == nil {
		e.Values = []string{}
	}
	return &e, nil
}

func (r *TagRegistryRepo) Count(ctx context.Context) (int, error) {
	var n int
	if err := r.c.db.QueryRow(ctx, `SELECT count(*) FROM tag_registry`).Scan(&n); err != nil {
		return 0, fmt.Errorf("postgres TagRegistryRepo.Count: %w", err)
	}
	return n, nil
}

func (r *TagRegistryRepo) List(ctx context.Context) ([]*models.TagRegistryEntry, error) {
	rows, err := r.c.db.Query(ctx, `SELECT `+tagRegistryCols+` FROM tag_registry ORDER BY "key"`)
	if err != nil {
		return nil, fmt.Errorf("postgres TagRegistryRepo.List: %w", err)
	}
	defer rows.Close()

	var out []*models.TagRegistryEntry
	for rows.Next() {
		e, err := scanTagRegistryEntry(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres TagRegistryRepo.List scan: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *TagRegistryRepo) Get(ctx context.Context, key string) (*models.TagRegistryEntry, error) {
	row := r.c.db.QueryRow(ctx, `SELECT `+tagRegistryCols+` FROM tag_registry WHERE "key" = $1`, key)
	e, err := scanTagRegistryEntry(row)
	if err != nil {
		if errors.Is(err, errNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres TagRegistryRepo.Get: %w", err)
	}
	return e, nil
}

func (r *TagRegistryRepo) Create(ctx context.Context, e *models.TagRegistryEntry) (*models.TagRegistryEntry, error) {
	if e.Values == nil {
		e.Values = []string{} // NOT NULL column; nil would violate the constraint
	}
	const q = `
INSERT INTO tag_registry("key", description, type, "values", max_length, propagation, created_by)
VALUES ($1, $2, $3, $4::text[], $5, $6, $7)
RETURNING created_at, updated_at`
	if err := r.c.db.QueryRow(ctx, q,
		e.Key, e.Description, e.Type, e.Values, e.MaxLength, e.Propagation, e.CreatedBy,
	).Scan(&e.CreatedAt, &e.UpdatedAt); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, repository.ErrDuplicateTagKey
		}
		return nil, fmt.Errorf("postgres TagRegistryRepo.Create: %w", err)
	}
	return e, nil
}

func (r *TagRegistryRepo) Update(ctx context.Context, e *models.TagRegistryEntry) (*models.TagRegistryEntry, error) {
	if e.Values == nil {
		e.Values = []string{} // NOT NULL column; nil would violate the constraint
	}
	const q = `
UPDATE tag_registry
SET description = $2,
    type = $3,
    "values" = $4::text[],
    max_length = $5,
    propagation = $6,
    updated_at = now()
WHERE "key" = $1
RETURNING created_by, created_at, updated_at`
	if err := r.c.db.QueryRow(ctx, q,
		e.Key, e.Description, e.Type, e.Values, e.MaxLength, e.Propagation,
	).Scan(&e.CreatedBy, &e.CreatedAt, &e.UpdatedAt); err != nil {
		if errors.Is(err, errNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres TagRegistryRepo.Update: %w", err)
	}
	if e.Values == nil {
		e.Values = []string{}
	}
	return e, nil
}

func (r *TagRegistryRepo) Delete(ctx context.Context, key string) error {
	return r.c.db.Exec(ctx, `DELETE FROM tag_registry WHERE "key" = $1`, key)
}
