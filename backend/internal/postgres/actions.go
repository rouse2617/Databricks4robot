package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/CyberOrigin2077/cyber-databrew/internal/id"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

// ActionRepo persists rows to the `actions` table — seg-internal time-bounded
// annotations. See docs/review/data-platform-design.md §5.2.15.
type ActionRepo struct {
	c *Client
}

// NewActionRepo creates an ActionRepo bound to c.
func NewActionRepo(c *Client) *ActionRepo { return &ActionRepo{c: c} }

var _ repository.ActionRepository = (*ActionRepo)(nil)

const actionSelectCols = `action_id::text, asset_id::text,
  start_ns, end_ns, action_index,
  COALESCE(primary_label, ''), COALESCE(labels, '{}'::text[]),
  COALESCE(description, ''), COALESCE(attrs, '{}'::jsonb),
  COALESCE(source_type, 'human'), COALESCE(source_name, ''), COALESCE(source_version, ''),
  COALESCE(run_id, ''), confidence, COALESCE(external_id, ''),
  COALESCE(task_id, ''), COALESCE(tenant_id, ''), COALESCE(project_id, ''),
  is_deleted, version, created_at, updated_at`

const actionIDSchemaMismatchHint = "actions.action_id column is incompatible with 8-char short IDs; run migration 018_actions_id_to_short_id.sql"

// scanAction reads one row produced by a SELECT using actionSelectCols.
func scanAction(scanner rowScanner) (*models.Action, error) {
	var (
		a          models.Action
		actionIdx  *int
		confidence *float64
		attrsBytes []byte
	)
	if err := scanner.Scan(
		&a.ActionID, &a.AssetID,
		&a.StartNs, &a.EndNs, &actionIdx,
		&a.PrimaryLabel, &a.Labels,
		&a.Description, &attrsBytes,
		&a.SourceType, &a.SourceName, &a.SourceVersion,
		&a.RunID, &confidence, &a.ExternalID,
		&a.TaskID,
		&a.TenantID, &a.ProjectID,
		&a.IsDeleted, &a.Version, &a.CreatedAt, &a.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if actionIdx != nil {
		a.ActionIndex = actionIdx
	}
	if confidence != nil {
		a.Confidence = confidence
	}
	if len(attrsBytes) > 0 {
		_ = json.Unmarshal(attrsBytes, &a.Attrs)
	}
	if a.Attrs == nil {
		a.Attrs = map[string]interface{}{}
	}
	if a.Labels == nil {
		a.Labels = []string{}
	}
	return &a, nil
}

// Insert inserts a single action. Returns repository.ErrOptimisticLock when
// the partial UNIQUE on (asset_id, source_name, external_id) is violated so
// callers can decide whether to upsert or surface a conflict.
func (r *ActionRepo) Insert(ctx context.Context, a *models.Action) error {
	if a == nil {
		return errors.New("postgres ActionRepo.Insert: action is nil")
	}
	if a.AssetID == "" {
		return errors.New("postgres ActionRepo.Insert: asset_id is required")
	}
	if a.EndNs < a.StartNs {
		return errors.New("postgres ActionRepo.Insert: end_ns must be >= start_ns")
	}
	if a.SourceType == "" {
		a.SourceType = models.ActionSourceHuman
	}
	if a.ActionID == "" {
		idv, err := id.GenerateActionID()
		if err != nil {
			return fmt.Errorf("postgres ActionRepo.Insert: generate action id: %w", err)
		}
		a.ActionID = idv
	}
	if !id.ValidateActionID(a.ActionID) {
		return errors.New("postgres ActionRepo.Insert: invalid action_id (must be 8 alphanumeric characters)")
	}
	labels := a.Labels
	if labels == nil {
		labels = []string{}
	}
	attrs := a.Attrs
	attrsJSON, _ := json.Marshal(attrs)
	if len(attrsJSON) == 0 {
		attrsJSON = []byte(`{}`)
	}

	nullable := func(s string) interface{} {
		if s == "" {
			return nil
		}
		return s
	}
	const q = `
INSERT INTO actions (
  action_id, asset_id, start_ns, end_ns, action_index,
  primary_label, labels, description, attrs,
  source_type, source_name, source_version, run_id, confidence, external_id,
  task_id, tenant_id, project_id
) VALUES (
  $1, $2, $3, $4, $5,
  $6, $7::text[], $8, $9::jsonb,
  $10, $11, $12, $13, $14, $15,
  $16, $17, $18
)
RETURNING action_id::text, version, created_at, updated_at`

	db := dbFromCtx(ctx, r.c.db)
	row := db.QueryRow(ctx, q,
		a.ActionID, a.AssetID, a.StartNs, a.EndNs, a.ActionIndex,
		nullable(a.PrimaryLabel), labels, nullable(a.Description), attrsJSON,
		a.SourceType, nullable(a.SourceName), nullable(a.SourceVersion),
		nullable(a.RunID), a.Confidence, nullable(a.ExternalID),
		nullable(a.TaskID), nullable(a.TenantID), nullable(a.ProjectID),
	)
	if err := row.Scan(&a.ActionID, &a.Version, &a.CreatedAt, &a.UpdatedAt); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" {
				return repository.ErrOptimisticLock
			}
			if isActionIDSchemaMismatch(pgErr) {
				return fmt.Errorf("%w: %s", repository.ErrSchemaMismatch, actionIDSchemaMismatchHint)
			}
		}
		return fmt.Errorf("postgres ActionRepo.Insert: %w", err)
	}
	return nil
}

func isActionIDSchemaMismatch(pgErr *pgconn.PgError) bool {
	if pgErr == nil {
		return false
	}
	// 22P02: invalid_text_representation. For legacy DBs where actions.action_id
	// is still UUID-typed, inserting an 8-char short id triggers this parse error.
	return pgErr.Code == "22P02" && strings.Contains(strings.ToLower(pgErr.Message), "type uuid")
}

// Get returns the action by id, or (nil, nil) when not found / soft-deleted.
func (r *ActionRepo) Get(ctx context.Context, actionID string) (*models.Action, error) {
	q := `SELECT ` + actionSelectCols + `
FROM actions WHERE action_id = $1 AND is_deleted = FALSE`
	db := dbFromCtx(ctx, r.c.db)
	a, err := scanAction(db.QueryRow(ctx, q, actionID))
	if err != nil {
		if errors.Is(err, errNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres ActionRepo.Get: %w", err)
	}
	return a, nil
}

// Update applies a partial patch and bumps version via CAS.
func (r *ActionRepo) Update(ctx context.Context, actionID string, expectedVersion int64, patch repository.ActionPatch) (*models.Action, error) {
	if actionID == "" {
		return nil, errors.New("postgres ActionRepo.Update: action_id is required")
	}
	sets := []string{"updated_at = now()", "version = version + 1"}
	args := []interface{}{}
	idx := 1
	addSet := func(col, cast string, val interface{}) {
		sets = append(sets, fmt.Sprintf("%s = $%d%s", col, idx, cast))
		args = append(args, val)
		idx++
	}
	if patch.StartNs != nil {
		addSet("start_ns", "", *patch.StartNs)
	}
	if patch.EndNs != nil {
		addSet("end_ns", "", *patch.EndNs)
	}
	if patch.ActionIndex != nil {
		addSet("action_index", "", *patch.ActionIndex)
	}
	if patch.PrimaryLabel != nil {
		addSet("primary_label", "", *patch.PrimaryLabel)
	}
	if patch.Labels != nil {
		addSet("labels", "::text[]", *patch.Labels)
	}
	if patch.Description != nil {
		addSet("description", "", *patch.Description)
	}
	if patch.Attrs != nil {
		body, err := json.Marshal(patch.Attrs)
		if err != nil {
			return nil, fmt.Errorf("postgres ActionRepo.Update: marshal attrs: %w", err)
		}
		addSet("attrs", "::jsonb", body)
	}
	if patch.SourceType != nil {
		addSet("source_type", "", *patch.SourceType)
	}
	if patch.SourceName != nil {
		addSet("source_name", "", *patch.SourceName)
	}
	if patch.SourceVersion != nil {
		addSet("source_version", "", *patch.SourceVersion)
	}
	if patch.RunID != nil {
		addSet("run_id", "", *patch.RunID)
	}
	if patch.Confidence != nil {
		addSet("confidence", "", *patch.Confidence)
	}
	// Always include the WHERE bind parameters at the end.
	whereVer := idx
	args = append(args, actionID)
	idx++
	args = append(args, expectedVersion)
	q := fmt.Sprintf(`
UPDATE actions
SET %s
WHERE action_id = $%d AND version = $%d AND is_deleted = FALSE
RETURNING `+actionSelectCols, strings.Join(sets, ", "), whereVer, whereVer+1)

	db := dbFromCtx(ctx, r.c.db)
	row := db.QueryRow(ctx, q, args...)
	a, err := scanAction(row)
	if err != nil {
		if errors.Is(err, errNoRows) {
			// Distinguish "missing/soft-deleted" from "version mismatch".
			cur, getErr := r.Get(ctx, actionID)
			if getErr != nil {
				return nil, fmt.Errorf("postgres ActionRepo.Update: %w", getErr)
			}
			if cur == nil {
				return nil, nil
			}
			return nil, repository.ErrOptimisticLock
		}
		return nil, fmt.Errorf("postgres ActionRepo.Update: %w", err)
	}
	return a, nil
}

// SoftDelete marks the action is_deleted=TRUE and bumps version via CAS.
func (r *ActionRepo) SoftDelete(ctx context.Context, actionID string, expectedVersion int64) (*models.Action, error) {
	if actionID == "" {
		return nil, errors.New("postgres ActionRepo.SoftDelete: action_id is required")
	}
	const q = `
UPDATE actions
SET is_deleted = TRUE, updated_at = now(), version = version + 1
WHERE action_id = $1 AND version = $2 AND is_deleted = FALSE
RETURNING ` + actionSelectCols
	db := dbFromCtx(ctx, r.c.db)
	a, err := scanAction(db.QueryRow(ctx, q, actionID, expectedVersion))
	if err != nil {
		if errors.Is(err, errNoRows) {
			cur, getErr := r.Get(ctx, actionID)
			if getErr != nil {
				return nil, fmt.Errorf("postgres ActionRepo.SoftDelete: %w", getErr)
			}
			if cur == nil {
				return nil, nil
			}
			return nil, repository.ErrOptimisticLock
		}
		return nil, fmt.Errorf("postgres ActionRepo.SoftDelete: %w", err)
	}
	return a, nil
}

// ListByAsset returns actions for the given seg matching opts.
func (r *ActionRepo) ListByAsset(ctx context.Context, assetID string, opts repository.ActionListOptions) ([]*models.Action, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 200
	}
	if limit > 1000 {
		limit = 1000
	}

	args := []interface{}{assetID}
	conds := []string{"asset_id = $1", "is_deleted = FALSE"}
	idx := 2
	if opts.PointAtNs != nil {
		conds = append(conds, fmt.Sprintf("start_ns <= $%d AND end_ns > $%d", idx, idx))
		args = append(args, *opts.PointAtNs)
		idx++
	}
	if opts.FromNs != nil {
		conds = append(conds, fmt.Sprintf("end_ns > $%d", idx))
		args = append(args, *opts.FromNs)
		idx++
	}
	if opts.ToNs != nil {
		conds = append(conds, fmt.Sprintf("start_ns < $%d", idx))
		args = append(args, *opts.ToNs)
		idx++
	}
	if opts.Label != "" {
		conds = append(conds, fmt.Sprintf("(primary_label = $%d OR labels @> ARRAY[$%d]::text[])", idx, idx))
		args = append(args, opts.Label)
		idx++
	}
	args = append(args, limit)

	q := `SELECT ` + actionSelectCols + `
FROM actions
WHERE ` + strings.Join(conds, " AND ") + `
ORDER BY start_ns ASC, action_id ASC
LIMIT $` + fmt.Sprint(idx)

	db := dbFromCtx(ctx, r.c.db)
	rows, err := db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("postgres ActionRepo.ListByAsset: %w", err)
	}
	defer rows.Close()
	var out []*models.Action
	for rows.Next() {
		a, err := scanAction(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres ActionRepo.ListByAsset scan: %w", err)
		}
		out = append(out, a)
	}
	return out, nil
}
