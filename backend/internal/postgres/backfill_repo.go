package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

// BackfillRepo persists backfill jobs and items.
type BackfillRepo struct {
	c *Client
}

// NewBackfillRepo creates a BackfillRepo.
func NewBackfillRepo(c *Client) *BackfillRepo {
	return &BackfillRepo{c: c}
}

var _ repository.BackfillRepository = (*BackfillRepo)(nil)

// ── Jobs ────────────────────────────────────────────────────────────────────

const backfillJobSelectCols = `id, name, template_id, filter_json,
  total_count, completed_count, failed_count, status, created_at, updated_at`

func scanBackfillJob(rs rowScanner) (*models.BackfillJob, error) {
	var (
		j          models.BackfillJob
		filterJSON []byte
	)
	if err := rs.Scan(
		&j.ID, &j.Name, &j.TemplateID, &filterJSON,
		&j.TotalCount, &j.CompletedCount, &j.FailedCount, &j.Status,
		&j.CreatedAt, &j.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if len(filterJSON) > 0 {
		_ = json.Unmarshal(filterJSON, &j.FilterJSON)
	}
	return &j, nil
}

// SaveJob inserts a backfill job.
func (r *BackfillRepo) SaveJob(ctx context.Context, j *models.BackfillJob) error {
	if j == nil {
		return errors.New("postgres BackfillRepo.SaveJob: nil job")
	}
	now := time.Now().UTC()
	if j.ID == "" {
		j.ID = uuid.New().String()
	}
	if j.CreatedAt.IsZero() {
		j.CreatedAt = now
	}
	j.UpdatedAt = now

	filterJSON, _ := json.Marshal(j.FilterJSON)
	if len(filterJSON) == 0 {
		filterJSON = []byte(`null`)
	}

	const q = `
	INSERT INTO backfill_jobs (id, name, template_id, filter_json,
	  total_count, completed_count, failed_count, status, created_at, updated_at)
	VALUES ($1, $2, $3, $4::jsonb, $5, $6, $7, $8, $9, $10)`
	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, q,
		j.ID, j.Name, j.TemplateID, filterJSON,
		j.TotalCount, j.CompletedCount, j.FailedCount, j.Status,
		j.CreatedAt, j.UpdatedAt,
	); err != nil {
		return fmt.Errorf("postgres BackfillRepo.SaveJob: %w", err)
	}
	return nil
}

// FindAllJobs returns all backfill jobs ordered by created_at DESC.
func (r *BackfillRepo) FindAllJobs(ctx context.Context) ([]models.BackfillJob, error) {
	q := `SELECT ` + backfillJobSelectCols + `
	FROM backfill_jobs
	ORDER BY created_at DESC`
	db := dbFromCtx(ctx, r.c.db)
	rows, err := db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("postgres BackfillRepo.FindAllJobs: %w", err)
	}
	defer rows.Close()
	var out []models.BackfillJob
	for rows.Next() {
		j, err := scanBackfillJob(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres BackfillRepo.FindAllJobs scan: %w", err)
		}
		out = append(out, *j)
	}
	return out, nil
}

// FindJobByID returns a backfill job by id, or (nil, nil) when not found.
func (r *BackfillRepo) FindJobByID(ctx context.Context, id string) (*models.BackfillJob, error) {
	q := `SELECT ` + backfillJobSelectCols + `
	FROM backfill_jobs
	WHERE id = $1`
	db := dbFromCtx(ctx, r.c.db)
	j, err := scanBackfillJob(db.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, errNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres BackfillRepo.FindJobByID: %w", err)
	}
	return j, nil
}

// UpdateJobStatus sets the status for a backfill job.
func (r *BackfillRepo) UpdateJobStatus(ctx context.Context, id, status string) error {
	const q = `UPDATE backfill_jobs SET status = $2, updated_at = NOW() WHERE id = $1`
	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, q, id, status); err != nil {
		return fmt.Errorf("postgres BackfillRepo.UpdateJobStatus: %w", err)
	}
	return nil
}

// IncrementCompleted increments the completed_count for a backfill job.
func (r *BackfillRepo) IncrementCompleted(ctx context.Context, id string) error {
	const q = `UPDATE backfill_jobs SET completed_count = completed_count + 1, updated_at = NOW() WHERE id = $1`
	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, q, id); err != nil {
		return fmt.Errorf("postgres BackfillRepo.IncrementCompleted: %w", err)
	}
	return nil
}

// IncrementFailed increments the failed_count for a backfill job.
func (r *BackfillRepo) IncrementFailed(ctx context.Context, id string) error {
	const q = `UPDATE backfill_jobs SET failed_count = failed_count + 1, updated_at = NOW() WHERE id = $1`
	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, q, id); err != nil {
		return fmt.Errorf("postgres BackfillRepo.IncrementFailed: %w", err)
	}
	return nil
}

// ── Items ───────────────────────────────────────────────────────────────────

const backfillItemSelectCols = `id, job_id, asset_id, status,
  pipeline_run_id, workflow_name, error_message, started_at, finished_at, created_at`

func scanBackfillItem(rs rowScanner) (*models.BackfillItem, error) {
	var item models.BackfillItem
	if err := rs.Scan(
		&item.ID, &item.JobID, &item.AssetID, &item.Status,
		&item.PipelineRunID, &item.WorkflowName, &item.ErrorMessage, &item.StartedAt, &item.FinishedAt,
		&item.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &item, nil
}

// SaveItem inserts a single backfill item.
func (r *BackfillRepo) SaveItem(ctx context.Context, item *models.BackfillItem) error {
	if item == nil {
		return errors.New("postgres BackfillRepo.SaveItem: nil item")
	}
	now := time.Now().UTC()
	if item.ID == "" {
		item.ID = uuid.New().String()
	}
	item.CreatedAt = now

	const q = `
	INSERT INTO backfill_items (id, job_id, asset_id, status,
	  pipeline_run_id, workflow_name, error_message, started_at, finished_at, created_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, q,
		item.ID, item.JobID, item.AssetID, item.Status,
		item.PipelineRunID, item.WorkflowName, item.ErrorMessage, item.StartedAt, item.FinishedAt, item.CreatedAt,
	); err != nil {
		return fmt.Errorf("postgres BackfillRepo.SaveItem: %w", err)
	}
	return nil
}

// SaveItems bulk-inserts backfill items in a batch.
func (r *BackfillRepo) SaveItems(ctx context.Context, items []models.BackfillItem) error {
	if len(items) == 0 {
		return nil
	}
	now := time.Now().UTC()
	db := dbFromCtx(ctx, r.c.db)

	for i := range items {
		if items[i].ID == "" {
			items[i].ID = uuid.New().String()
		}
		items[i].CreatedAt = now
	}

	const q = `
	INSERT INTO backfill_items (id, job_id, asset_id, status,
	  pipeline_run_id, workflow_name, error_message, started_at, finished_at, created_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	for _, item := range items {
		if err := db.Exec(ctx, q,
			item.ID, item.JobID, item.AssetID, item.Status,
			item.PipelineRunID, item.WorkflowName, item.ErrorMessage, item.StartedAt, item.FinishedAt, item.CreatedAt,
		); err != nil {
			return fmt.Errorf("postgres BackfillRepo.SaveItems: %w", err)
		}
	}
	return nil
}

// FindItemsByJobID returns all items for a backfill job ordered by created_at ASC.
func (r *BackfillRepo) FindItemsByJobID(ctx context.Context, jobID string) ([]models.BackfillItem, error) {
	q := `SELECT ` + backfillItemSelectCols + `
	FROM backfill_items
	WHERE job_id = $1
	ORDER BY created_at ASC`
	db := dbFromCtx(ctx, r.c.db)
	rows, err := db.Query(ctx, q, jobID)
	if err != nil {
		return nil, fmt.Errorf("postgres BackfillRepo.FindItemsByJobID: %w", err)
	}
	defer rows.Close()
	var out []models.BackfillItem
	for rows.Next() {
		item, err := scanBackfillItem(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres BackfillRepo.FindItemsByJobID scan: %w", err)
		}
		out = append(out, *item)
	}
	return out, nil
}

// FindItemByID returns a single backfill item by id, or (nil, nil).
func (r *BackfillRepo) FindItemByID(ctx context.Context, id string) (*models.BackfillItem, error) {
	q := `SELECT ` + backfillItemSelectCols + `
	FROM backfill_items
	WHERE id = $1`
	db := dbFromCtx(ctx, r.c.db)
	item, err := scanBackfillItem(db.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, errNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres BackfillRepo.FindItemByID: %w", err)
	}
	return item, nil
}

// UpdateItemStatus sets status, workflow_name, error_message for a backfill item.
func (r *BackfillRepo) UpdateItemStatus(ctx context.Context, id, status, workflowName, errorMsg string) error {
	const q = `UPDATE backfill_items SET
	  status = $2, workflow_name = $3, error_message = $4,
	  started_at = CASE WHEN $2 IN ('running','completed','failed') AND started_at IS NULL THEN NOW() ELSE started_at END,
	  finished_at = CASE WHEN $2 IN ('completed','failed','cancelled') THEN NOW() ELSE NULL END
	WHERE id = $1`
	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, q, id, status, nullIfEmpty(workflowName), nullIfEmpty(errorMsg)); err != nil {
		return fmt.Errorf("postgres BackfillRepo.UpdateItemStatus: %w", err)
	}
	return nil
}

// CountItemsByStatus counts items with a given status for a job.
func (r *BackfillRepo) CountItemsByStatus(ctx context.Context, jobID, status string) (int, error) {
	const q = `SELECT COUNT(*) FROM backfill_items WHERE job_id = $1 AND status = $2`
	db := dbFromCtx(ctx, r.c.db)
	var n int
	if err := db.QueryRow(ctx, q, jobID, status).Scan(&n); err != nil {
		return 0, fmt.Errorf("postgres BackfillRepo.CountItemsByStatus: %w", err)
	}
	return n, nil
}

// UpdateItemPipelineRun links an item to its pipeline run and workflow name.
func (r *BackfillRepo) UpdateItemPipelineRun(ctx context.Context, id, pipelineRunID, workflowName, status string) error {
	const q = `UPDATE backfill_items SET
	  pipeline_run_id = $2, workflow_name = $3, status = $4,
	  started_at = CASE WHEN started_at IS NULL THEN NOW() ELSE started_at END
	WHERE id = $1`
	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, q, id, nullIfEmpty(pipelineRunID), nullIfEmpty(workflowName), status); err != nil {
		return fmt.Errorf("postgres BackfillRepo.UpdateItemPipelineRun: %w", err)
	}
	return nil
}

// UpdateJobProgress updates aggregate counters and job status.
func (r *BackfillRepo) UpdateJobProgress(ctx context.Context, id string, completed, failed int, status string) error {
	const q = `UPDATE backfill_jobs SET
	  completed_count = $2, failed_count = $3, status = $4, updated_at = NOW()
	WHERE id = $1`
	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, q, id, completed, failed, status); err != nil {
		return fmt.Errorf("postgres BackfillRepo.UpdateJobProgress: %w", err)
	}
	return nil
}
