package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

type SavedQueryRepo struct {
	c *Client
}

func NewSavedQueryRepo(c *Client) *SavedQueryRepo { return &SavedQueryRepo{c: c} }

func (r *SavedQueryRepo) List(ctx context.Context) ([]*models.SavedQuery, error) {
	const q = `
SELECT saved_query_id, name, COALESCE(description, ''), resource, schema_version, query_ir_json, COALESCE(owner, ''), created_at, updated_at
FROM saved_queries
ORDER BY updated_at DESC, saved_query_id DESC`
	rows, err := r.c.db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("postgres SavedQueryRepo.List: %w", err)
	}
	defer rows.Close()

	var out []*models.SavedQuery
	for rows.Next() {
		var (
			item      models.SavedQuery
			queryJSON []byte
		)
		if err := rows.Scan(
			&item.SavedQueryID,
			&item.Name,
			&item.Description,
			&item.Resource,
			&item.SchemaVersion,
			&queryJSON,
			&item.Owner,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("postgres SavedQueryRepo.List scan: %w", err)
		}
		if len(queryJSON) > 0 {
			_ = json.Unmarshal(queryJSON, &item.QueryIRJSON)
		}
		out = append(out, &item)
	}
	return out, nil
}

func (r *SavedQueryRepo) Get(ctx context.Context, id string) (*models.SavedQuery, error) {
	const q = `
SELECT saved_query_id, name, COALESCE(description, ''), resource, schema_version, query_ir_json, COALESCE(owner, ''), created_at, updated_at
FROM saved_queries
WHERE saved_query_id::text = $1`
	var (
		item      models.SavedQuery
		queryJSON []byte
	)
	err := r.c.db.QueryRow(ctx, q, id).Scan(
		&item.SavedQueryID,
		&item.Name,
		&item.Description,
		&item.Resource,
		&item.SchemaVersion,
		&queryJSON,
		&item.Owner,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		if err == errNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres SavedQueryRepo.Get: %w", err)
	}
	if len(queryJSON) > 0 {
		_ = json.Unmarshal(queryJSON, &item.QueryIRJSON)
	}
	return &item, nil
}

func (r *SavedQueryRepo) Create(ctx context.Context, item *models.SavedQuery) (*models.SavedQuery, error) {
	payload, err := json.Marshal(item.QueryIRJSON)
	if err != nil {
		return nil, fmt.Errorf("postgres SavedQueryRepo.Create marshal: %w", err)
	}
	const q = `
INSERT INTO saved_queries(name, description, resource, schema_version, query_ir_json, owner)
VALUES ($1, NULLIF($2, ''), $3, $4, $5::jsonb, NULLIF($6, ''))
RETURNING saved_query_id, created_at, updated_at`
	if err := r.c.db.QueryRow(ctx, q, item.Name, item.Description, item.Resource, item.SchemaVersion, payload, item.Owner).Scan(
		&item.SavedQueryID, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("postgres SavedQueryRepo.Create: %w", err)
	}
	return item, nil
}

func (r *SavedQueryRepo) Update(ctx context.Context, item *models.SavedQuery) (*models.SavedQuery, error) {
	payload, err := json.Marshal(item.QueryIRJSON)
	if err != nil {
		return nil, fmt.Errorf("postgres SavedQueryRepo.Update marshal: %w", err)
	}
	const q = `
UPDATE saved_queries
SET name = $2,
    description = NULLIF($3, ''),
    resource = $4,
    schema_version = $5,
    query_ir_json = $6::jsonb,
    owner = NULLIF($7, '')
WHERE saved_query_id::text = $1
RETURNING created_at, updated_at`
	if err := r.c.db.QueryRow(ctx, q, item.SavedQueryID, item.Name, item.Description, item.Resource, item.SchemaVersion, payload, item.Owner).Scan(
		&item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		if err == errNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres SavedQueryRepo.Update: %w", err)
	}
	return item, nil
}

func (r *SavedQueryRepo) Delete(ctx context.Context, id string) error {
	const q = `DELETE FROM saved_queries WHERE saved_query_id::text = $1`
	return r.c.db.Exec(ctx, q, id)
}
