package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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

func nullIntIfZero(v int) any {
	if v == 0 {
		return nil
	}
	return v
}

// ── Jobs ────────────────────────────────────────────────────────────────────

const backfillJobSelectCols = `id, name, template_id, filter_json,
  total_count, completed_count, failed_count, status,
  COALESCE(template_version, 0), COALESCE(pilot_count, 0), COALESCE(pilot_phase, ''),
  created_at, updated_at,
  COALESCE(created_by, ''), finished_at`

func scanBackfillJob(rs rowScanner) (*models.BackfillJob, error) {
	var (
		j          models.BackfillJob
		filterJSON []byte
	)
	if err := rs.Scan(
		&j.ID, &j.Name, &j.TemplateID, &filterJSON,
		&j.TotalCount, &j.CompletedCount, &j.FailedCount, &j.Status,
		&j.TemplateVersion, &j.PilotCount, &j.PilotPhase,
		&j.CreatedAt, &j.UpdatedAt,
		&j.CreatedBy, &j.FinishedAt,
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
	  total_count, completed_count, failed_count, status,
	  template_version, pilot_count, pilot_phase, created_at, updated_at, created_by)
	VALUES ($1, $2, $3, $4::jsonb, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`
	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, q,
		j.ID, j.Name, j.TemplateID, filterJSON,
		j.TotalCount, j.CompletedCount, j.FailedCount, j.Status,
		nullIntIfZero(j.TemplateVersion), j.PilotCount, nullIfEmpty(j.PilotPhase),
		j.CreatedAt, j.UpdatedAt, nullIfEmpty(j.CreatedBy),
	); err != nil {
		return fmt.Errorf("postgres BackfillRepo.SaveJob: %w", err)
	}
	return nil
}

// FindAllJobs returns all backfill jobs ordered by created_at DESC.
func (r *BackfillRepo) FindAllJobs(ctx context.Context) ([]models.BackfillJob, error) {
	// Enrich the list with each job's subtask run span (earliest start, latest
	// finish) so the UI can show a real run duration that excludes submit/queue/
	// pause waiting. MIN/MAX ignore NULL item timestamps, so jobs whose subtasks
	// never started yield NULL spans.
	q := `SELECT ` + backfillJobSelectCols + `,
	  rs.run_started_at, rs.run_finished_at
	FROM backfill_jobs
	LEFT JOIN (
	  SELECT job_id, MIN(started_at) AS run_started_at, MAX(finished_at) AS run_finished_at
	  FROM backfill_items GROUP BY job_id
	) rs ON rs.job_id = backfill_jobs.id
	ORDER BY created_at DESC`
	db := dbFromCtx(ctx, r.c.db)
	rows, err := db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("postgres BackfillRepo.FindAllJobs: %w", err)
	}
	defer rows.Close()
	var out []models.BackfillJob
	for rows.Next() {
		j, err := scanBackfillJobWithRunSpan(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres BackfillRepo.FindAllJobs scan: %w", err)
		}
		out = append(out, *j)
	}
	return out, nil
}

// scanBackfillJobWithRunSpan scans the base job columns plus the aggregated
// subtask run span (run_started_at, run_finished_at). Used by list queries that
// LEFT JOIN the per-job MIN(started_at)/MAX(finished_at) of backfill_items.
func scanBackfillJobWithRunSpan(rs rowScanner) (*models.BackfillJob, error) {
	var (
		j          models.BackfillJob
		filterJSON []byte
	)
	if err := rs.Scan(
		&j.ID, &j.Name, &j.TemplateID, &filterJSON,
		&j.TotalCount, &j.CompletedCount, &j.FailedCount, &j.Status,
		&j.TemplateVersion, &j.PilotCount, &j.PilotPhase,
		&j.CreatedAt, &j.UpdatedAt,
		&j.CreatedBy, &j.FinishedAt,
		&j.RunStartedAt, &j.RunFinishedAt,
	); err != nil {
		return nil, err
	}
	if len(filterJSON) > 0 {
		_ = json.Unmarshal(filterJSON, &j.FilterJSON)
	}
	return &j, nil
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
// finished_at is stamped when the status is a terminal state (completed/failed).
func (r *BackfillRepo) UpdateJobStatus(ctx context.Context, id, status string) error {
	const q = `UPDATE backfill_jobs
	  SET status = $2, updated_at = NOW(),
	      finished_at = CASE WHEN $2 IN ('completed','failed') THEN NOW() ELSE finished_at END
	  WHERE id = $1`
	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, q, id, status); err != nil {
		return fmt.Errorf("postgres BackfillRepo.UpdateJobStatus: %w", err)
	}
	return nil
}

func (r *BackfillRepo) UpdateJobPilotPhase(ctx context.Context, id, status, pilotPhase string) error {
	const q = `UPDATE backfill_jobs SET status = $2, pilot_phase = $3, updated_at = NOW(),
	  finished_at = CASE WHEN $2 IN ('completed','failed') THEN COALESCE(finished_at, NOW()) ELSE finished_at END
	WHERE id = $1`
	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, q, id, status, pilotPhase); err != nil {
		return fmt.Errorf("postgres BackfillRepo.UpdateJobPilotPhase: %w", err)
	}
	return nil
}

// ClaimJobNotification atomically claims the completion-notification slot for
// a job: it returns true only for the one caller that flips
// notification_sent_at from NULL to now(), across any number of concurrent
// backend instances racing on the same job. Callers that lose the race (or
// call again for an already-notified job) get false with no error.
func (r *BackfillRepo) ClaimJobNotification(ctx context.Context, id string) (bool, error) {
	const q = `
	UPDATE backfill_jobs
	SET notification_sent_at = NOW()
	WHERE id = $1 AND notification_sent_at IS NULL
	RETURNING id`
	db := dbFromCtx(ctx, r.c.db)
	var claimedID string
	if err := db.QueryRow(ctx, q, id).Scan(&claimedID); err != nil {
		if errors.Is(err, errNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("postgres BackfillRepo.ClaimJobNotification: %w", err)
	}
	return true, nil
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
  pipeline_run_id, workflow_name, error_message, attempts, started_at, finished_at, created_at`

func scanBackfillItem(rs rowScanner) (*models.BackfillItem, error) {
	var item models.BackfillItem
	if err := rs.Scan(
		&item.ID, &item.JobID, &item.AssetID, &item.Status,
		&item.PipelineRunID, &item.WorkflowName, &item.ErrorMessage, &item.Attempts, &item.StartedAt, &item.FinishedAt,
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

// SaveItems bulk-inserts backfill items using multi-row INSERT statements.
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

	const colsPerRow = 10
	const rowsPerStmt = 50
	for start := 0; start < len(items); start += rowsPerStmt {
		end := start + rowsPerStmt
		if end > len(items) {
			end = len(items)
		}
		chunk := items[start:end]

		var sb strings.Builder
		sb.WriteString(`
INSERT INTO backfill_items (id, job_id, asset_id, status,
  pipeline_run_id, workflow_name, error_message, started_at, finished_at, created_at)
VALUES `)

		args := make([]any, 0, len(chunk)*colsPerRow)
		for i, item := range chunk {
			if i > 0 {
				sb.WriteString(",")
			}
			base := i*colsPerRow + 1
			fmt.Fprintf(
				&sb,
				"($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d)",
				base, base+1, base+2, base+3, base+4,
				base+5, base+6, base+7, base+8, base+9,
			)
			args = append(
				args,
				item.ID, item.JobID, item.AssetID, item.Status,
				item.PipelineRunID, item.WorkflowName, item.ErrorMessage,
				item.StartedAt, item.FinishedAt, item.CreatedAt,
			)
		}

		if err := db.Exec(ctx, sb.String(), args...); err != nil {
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

// FindItemByPipelineRunID returns a backfill item linked to a pipeline run id, or (nil, nil).
func (r *BackfillRepo) FindItemByPipelineRunID(ctx context.Context, pipelineRunID string) (*models.BackfillItem, error) {
	pipelineRunID = strings.TrimSpace(pipelineRunID)
	if pipelineRunID == "" {
		return nil, nil
	}
	q := `SELECT ` + backfillItemSelectCols + `
	FROM backfill_items
	WHERE pipeline_run_id = $1
	ORDER BY created_at DESC
	LIMIT 1`
	db := dbFromCtx(ctx, r.c.db)
	item, err := scanBackfillItem(db.QueryRow(ctx, q, pipelineRunID))
	if err != nil {
		if errors.Is(err, errNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres BackfillRepo.FindItemByPipelineRunID: %w", err)
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

// SummarizeItemStatuses returns aggregate counts per status bucket in one query.
func (r *BackfillRepo) SummarizeItemStatuses(ctx context.Context, jobID string) (repository.BackfillItemStatusSummary, error) {
	const q = `
WITH current_items AS (
  SELECT DISTINCT ON (job_id, asset_id) status
  FROM backfill_items
  WHERE job_id = $1
  ORDER BY job_id, asset_id,
    CASE status
      WHEN 'completed' THEN 0
      WHEN 'failed' THEN 1
      WHEN 'cancelled' THEN 1
      WHEN 'running' THEN 2
      WHEN 'awaiting_result' THEN 2
      WHEN 'pending' THEN 3
      ELSE 4
    END,
    CASE WHEN pipeline_run_id IS NULL OR pipeline_run_id = '' THEN 1 ELSE 0 END,
    finished_at DESC NULLS LAST,
    started_at DESC NULLS LAST,
    created_at DESC
)
SELECT
  COUNT(*) FILTER (WHERE status = 'completed'),
  COUNT(*) FILTER (WHERE status IN ('failed', 'cancelled')),
  COUNT(*) FILTER (WHERE status = 'pending'),
  COUNT(*) FILTER (WHERE status IN ('running', 'awaiting_result'))
FROM current_items`
	db := dbFromCtx(ctx, r.c.db)
	var summary repository.BackfillItemStatusSummary
	if err := db.QueryRow(ctx, q, jobID).Scan(
		&summary.Completed,
		&summary.Failed,
		&summary.Pending,
		&summary.Running,
	); err != nil {
		return summary, fmt.Errorf("postgres BackfillRepo.SummarizeItemStatuses: %w", err)
	}
	return summary, nil
}

// FindItemsByJobIDWithStatuses returns items matching any of the given statuses.
func (r *BackfillRepo) FindItemsByJobIDWithStatuses(ctx context.Context, jobID string, statuses []string) ([]models.BackfillItem, error) {
	if len(statuses) == 0 {
		return nil, nil
	}
	placeholders := make([]string, len(statuses))
	args := make([]any, 0, len(statuses)+1)
	args = append(args, jobID)
	for i, status := range statuses {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args = append(args, status)
	}
	q := `SELECT ` + backfillItemSelectCols + `
FROM backfill_items
WHERE job_id = $1 AND status IN (` + strings.Join(placeholders, ",") + `)
ORDER BY created_at ASC`
	db := dbFromCtx(ctx, r.c.db)
	rows, err := db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("postgres BackfillRepo.FindItemsByJobIDWithStatuses: %w", err)
	}
	defer rows.Close()
	var out []models.BackfillItem
	for rows.Next() {
		item, err := scanBackfillItem(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres BackfillRepo.FindItemsByJobIDWithStatuses scan: %w", err)
		}
		out = append(out, *item)
	}
	return out, nil
}

// FindItemsMissingPipelineRun returns the current backfill item per asset that still
// lacks a pipeline run ledger row.
func (r *BackfillRepo) FindItemsMissingPipelineRun(ctx context.Context, jobID string) ([]models.BackfillItem, error) {
	q := `SELECT
  bi.id, bi.job_id, bi.asset_id, bi.status,
  bi.pipeline_run_id, bi.workflow_name, bi.error_message, bi.started_at, bi.finished_at, bi.created_at
FROM (
  SELECT DISTINCT ON (job_id, asset_id) *
  FROM backfill_items
  WHERE job_id = $1
  ORDER BY job_id, asset_id,
    CASE status
      WHEN 'completed' THEN 0
      WHEN 'failed' THEN 1
      WHEN 'cancelled' THEN 1
      WHEN 'running' THEN 2
      WHEN 'pending' THEN 3
      ELSE 4
    END,
    CASE WHEN pipeline_run_id IS NULL OR pipeline_run_id = '' THEN 1 ELSE 0 END,
    finished_at DESC NULLS LAST,
    started_at DESC NULLS LAST,
    created_at DESC
) bi
WHERE bi.pipeline_run_id IS NULL OR bi.pipeline_run_id = ''
   OR NOT EXISTS (
     SELECT 1 FROM pipeline_runs pr WHERE pr.id = bi.pipeline_run_id
   )
ORDER BY bi.created_at ASC`
	db := dbFromCtx(ctx, r.c.db)
	rows, err := db.Query(ctx, q, jobID)
	if err != nil {
		return nil, fmt.Errorf("postgres BackfillRepo.FindItemsMissingPipelineRun: %w", err)
	}
	defer rows.Close()
	var out []models.BackfillItem
	for rows.Next() {
		item, err := scanBackfillItem(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres BackfillRepo.FindItemsMissingPipelineRun scan: %w", err)
		}
		out = append(out, *item)
	}
	return out, nil
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
	  completed_count = $2, failed_count = $3, status = $4, updated_at = NOW(),
	  finished_at = CASE WHEN $4 IN ('completed','failed') THEN COALESCE(finished_at, NOW()) ELSE finished_at END
	WHERE id = $1`
	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, q, id, completed, failed, status); err != nil {
		return fmt.Errorf("postgres BackfillRepo.UpdateJobProgress: %w", err)
	}
	return nil
}

func (r *BackfillRepo) FindItemsByScope(ctx context.Context, filter repository.BackfillRerunItemFilter) ([]models.BackfillItem, error) {
	db := dbFromCtx(ctx, r.c.db)
	args := []any{filter.JobID}
	where := []string{"bi.job_id = $1"}
	joinNode := strings.TrimSpace(filter.PipelineNodeID) != ""
	if len(filter.Statuses) > 0 {
		args = append(args, filter.Statuses)
		where = append(where, fmt.Sprintf("bi.status = ANY($%d)", len(args)))
	}
	if len(filter.ItemIDs) > 0 {
		args = append(args, filter.ItemIDs)
		where = append(where, fmt.Sprintf("bi.id = ANY($%d)", len(args)))
	}
	if len(filter.AssetIDs) > 0 {
		args = append(args, filter.AssetIDs)
		where = append(where, fmt.Sprintf("bi.asset_id = ANY($%d)", len(args)))
	}
	if joinNode {
		args = append(args, filter.PipelineNodeID)
		where = append(where, fmt.Sprintf("n.pipeline_node_id = $%d", len(args)))
		statusExpr := projectedAssetNodeStatusSQLWithRun("n", "pr")
		statuses := filter.NodeStatuses
		if len(statuses) == 0 {
			statuses = []string{"Failed", "Error"}
		}
		args = append(args, statuses)
		where = append(where, fmt.Sprintf("%s = ANY($%d)", statusExpr, len(args)))
	}
	from := "FROM backfill_items bi"
	if joinNode {
		from += " INNER JOIN pipeline_runs pr ON pr.id = bi.pipeline_run_id INNER JOIN pipeline_run_asset_nodes n ON n.run_id = pr.id AND n.asset_id = bi.asset_id"
	}
	q := `SELECT
  bi.id, bi.job_id, bi.asset_id, bi.status,
  bi.pipeline_run_id, bi.workflow_name, bi.error_message, bi.attempts, bi.started_at, bi.finished_at, bi.created_at
` + from + `
WHERE ` + strings.Join(where, " AND ") + `
ORDER BY bi.created_at ASC`
	rows, err := db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("postgres BackfillRepo.FindItemsByScope: %w", err)
	}
	defer rows.Close()
	var out []models.BackfillItem
	for rows.Next() {
		item, err := scanBackfillItem(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres BackfillRepo.FindItemsByScope scan: %w", err)
		}
		out = append(out, *item)
	}
	return out, nil
}

func (r *BackfillRepo) PrepareItemsForRerun(ctx context.Context, itemIDs []string) error {
	if len(itemIDs) == 0 {
		return nil
	}
	const q = `UPDATE backfill_items SET
	  status = 'pending',
	  error_message = NULL,
	  started_at = NULL,
	  finished_at = NULL
	WHERE id = ANY($1)`
	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, q, itemIDs); err != nil {
		return fmt.Errorf("postgres BackfillRepo.PrepareItemsForRerun: %w", err)
	}
	return nil
}

func projectedAssetNodeStatusSQL(alias string) string {
	prefix := ""
	if strings.TrimSpace(alias) != "" {
		prefix = strings.TrimSpace(alias) + "."
	}
	status := prefix + "status"
	message := "LOWER(COALESCE(" + prefix + "message, ''))"
	signals := []string{
		"invalidimagename",
		"invalid image name",
		"invalid reference format",
		"failed to apply default image tag",
		"couldn't parse image name",
		"errimagepull",
		"imagepullbackoff",
		"failed to pull image",
		"pull access denied",
		"manifest unknown",
		"unauthorized: authentication required",
	}
	parts := make([]string, 0, len(signals))
	for _, signal := range signals {
		parts = append(parts, message+" LIKE '%"+strings.ReplaceAll(signal, "'", "''")+"%'")
	}
	return "CASE WHEN LOWER(COALESCE(" + status + ", '')) = 'pending' AND (" + strings.Join(parts, " OR ") + ") THEN 'Error' ELSE COALESCE(NULLIF(" + status + ", ''), 'Pending') END"
}

func projectedSchedulerDiagnosticNodeSQL(alias string) string {
	prefix := ""
	if strings.TrimSpace(alias) != "" {
		prefix = strings.TrimSpace(alias) + "."
	}
	message := "LOWER(COALESCE(" + prefix + "message, ''))"
	signals := []string{
		"unschedulable",
		"insufficient cpu",
		"insufficient memory",
		"insufficient ephemeral-storage",
		"insufficient nvidia.com/gpu",
		"untolerated taint",
		"didn't match pod's node affinity",
		"didn't match node selector",
		"preemption is not helpful",
	}
	parts := make([]string, 0, len(signals))
	for _, signal := range signals {
		parts = append(parts, message+" LIKE '%"+strings.ReplaceAll(signal, "'", "''")+"%'")
	}
	return strings.Join(parts, " OR ")
}

func projectedAssetNodeStatusSQLWithRun(nodeAlias, runAlias string) string {
	base := projectedAssetNodeStatusSQL(nodeAlias)
	nodePrefix := ""
	if strings.TrimSpace(nodeAlias) != "" {
		nodePrefix = strings.TrimSpace(nodeAlias) + "."
	}
	runPrefix := ""
	if strings.TrimSpace(runAlias) != "" {
		runPrefix = strings.TrimSpace(runAlias) + "."
	}
	nodeStatus := "LOWER(COALESCE(" + nodePrefix + "status, ''))"
	runTerminal := "LOWER(COALESCE(" + runPrefix + "status, '')) IN ('failed', 'error', 'expired')"
	schedulerDiagnostic := projectedSchedulerDiagnosticNodeSQL(nodeAlias)
	return "CASE WHEN " + runTerminal + " AND (" + nodeStatus + " = 'running' OR (" + nodeStatus + " = 'pending' AND (" + schedulerDiagnostic + "))) THEN 'Error' ELSE " + base + " END"
}

func (r *BackfillRepo) AggregateNodeStatusByBatchJobID(ctx context.Context, jobID string) ([]repository.BatchNodeStatusAggregate, error) {
	statusExpr := projectedAssetNodeStatusSQLWithRun("n", "pr")
	q := `
SELECT
  n.pipeline_node_id,
  COALESCE(NULLIF(n.display_name, ''), n.pipeline_node_id) AS display_name,
  ` + statusExpr + ` AS status,
  COUNT(*) AS cnt
FROM pipeline_run_asset_nodes n
INNER JOIN pipeline_runs pr ON pr.id = n.run_id
INNER JOIN (
  SELECT DISTINCT ON (job_id, asset_id) *
  FROM backfill_items
  WHERE job_id = $1
  ORDER BY job_id, asset_id,
    CASE status
      WHEN 'completed' THEN 0
      WHEN 'failed' THEN 1
      WHEN 'cancelled' THEN 1
      WHEN 'running' THEN 2
      WHEN 'pending' THEN 3
      ELSE 4
    END,
    CASE WHEN pipeline_run_id IS NULL OR pipeline_run_id = '' THEN 1 ELSE 0 END,
    finished_at DESC NULLS LAST,
    started_at DESC NULLS LAST,
    created_at DESC
) bi ON bi.pipeline_run_id = pr.id AND bi.asset_id = n.asset_id
WHERE bi.job_id = $1
GROUP BY 1, 2, 3
ORDER BY 2, 1, 3`
	db := dbFromCtx(ctx, r.c.db)
	rows, err := db.Query(ctx, q, jobID)
	if err != nil {
		return nil, fmt.Errorf("postgres BackfillRepo.AggregateNodeStatusByBatchJobID: %w", err)
	}
	defer rows.Close()
	out := []repository.BatchNodeStatusAggregate{}
	for rows.Next() {
		var row repository.BatchNodeStatusAggregate
		if err := rows.Scan(&row.PipelineNodeID, &row.DisplayName, &row.Status, &row.Count); err != nil {
			return nil, fmt.Errorf("postgres BackfillRepo.AggregateNodeStatusByBatchJobID scan: %w", err)
		}
		out = append(out, row)
	}
	return out, nil
}

func (r *BackfillRepo) ListNodeFailures(ctx context.Context, filter repository.BatchNodeFailureFilter) (*models.BatchNodeFailureListResult, error) {
	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	statuses := filter.Statuses
	if len(statuses) == 0 {
		statuses = []string{"Failed", "Error"}
	}
	statusExpr := projectedAssetNodeStatusSQLWithRun("n", "pr")
	args := []any{filter.JobID, filter.PipelineNodeID, statuses}
	where := []string{"bi.job_id = $1", "n.pipeline_node_id = $2", statusExpr + " = ANY($3)"}
	if q := strings.TrimSpace(filter.Query); q != "" {
		args = append(args, "%"+q+"%")
		where = append(where, fmt.Sprintf("bi.asset_id ILIKE $%d", len(args)))
	}
	whereSQL := strings.Join(where, " AND ")
	db := dbFromCtx(ctx, r.c.db)
	countQ := `
WITH current_items AS (
  SELECT DISTINCT ON (job_id, asset_id) *
  FROM backfill_items
  WHERE job_id = $1
  ORDER BY job_id, asset_id,
    CASE status
      WHEN 'completed' THEN 0
      WHEN 'failed' THEN 1
      WHEN 'cancelled' THEN 1
      WHEN 'running' THEN 2
      WHEN 'pending' THEN 3
      ELSE 4
    END,
    CASE WHEN pipeline_run_id IS NULL OR pipeline_run_id = '' THEN 1 ELSE 0 END,
    finished_at DESC NULLS LAST,
    started_at DESC NULLS LAST,
    created_at DESC
)
SELECT COUNT(*)
FROM current_items bi
INNER JOIN pipeline_runs pr ON pr.id = bi.pipeline_run_id
INNER JOIN pipeline_run_asset_nodes n ON n.run_id = pr.id AND n.asset_id = bi.asset_id
WHERE ` + whereSQL
	var total int
	if err := db.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("postgres BackfillRepo.ListNodeFailures count: %w", err)
	}
	args = append(args, pageSize, (page-1)*pageSize)
	q := `
WITH current_items AS (
  SELECT DISTINCT ON (job_id, asset_id) *
  FROM backfill_items
  WHERE job_id = $1
  ORDER BY job_id, asset_id,
    CASE status
      WHEN 'completed' THEN 0
      WHEN 'failed' THEN 1
      WHEN 'cancelled' THEN 1
      WHEN 'running' THEN 2
      WHEN 'pending' THEN 3
      ELSE 4
    END,
    CASE WHEN pipeline_run_id IS NULL OR pipeline_run_id = '' THEN 1 ELSE 0 END,
    finished_at DESC NULLS LAST,
    started_at DESC NULLS LAST,
    created_at DESC
)
SELECT
  bi.id, bi.asset_id, pr.id, COALESCE(pr.workflow_name, bi.workflow_name, ''),
  n.pipeline_node_id, COALESCE(NULLIF(n.display_name, ''), n.pipeline_node_id),
  ` + statusExpr + `, COALESCE(NULLIF(n.message, ''), NULLIF(pr.message, ''), ''), n.started_at, n.finished_at
FROM current_items bi
INNER JOIN pipeline_runs pr ON pr.id = bi.pipeline_run_id
INNER JOIN pipeline_run_asset_nodes n ON n.run_id = pr.id AND n.asset_id = bi.asset_id
WHERE ` + whereSQL + `
ORDER BY n.updated_at DESC, bi.created_at ASC
LIMIT $` + fmt.Sprint(len(args)-1) + ` OFFSET $` + fmt.Sprint(len(args))
	rows, err := db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("postgres BackfillRepo.ListNodeFailures: %w", err)
	}
	defer rows.Close()
	items := []models.BatchNodeFailureItem{}
	for rows.Next() {
		var item models.BatchNodeFailureItem
		if err := rows.Scan(
			&item.BackfillItemID, &item.AssetID, &item.RunID, &item.WorkflowName,
			&item.PipelineNodeID, &item.DisplayName, &item.Status, &item.Message,
			&item.StartedAt, &item.FinishedAt,
		); err != nil {
			return nil, fmt.Errorf("postgres BackfillRepo.ListNodeFailures scan: %w", err)
		}
		items = append(items, item)
	}
	return &models.BatchNodeFailureListResult{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func (r *BackfillRepo) CountPipelineRunsByBatchJobID(ctx context.Context, jobID string) (int, error) {
	const q = `
SELECT COUNT(*)
FROM (
  SELECT DISTINCT ON (job_id, asset_id) *
  FROM backfill_items
  WHERE job_id = $1
  ORDER BY job_id, asset_id,
    CASE status
      WHEN 'completed' THEN 0
      WHEN 'failed' THEN 1
      WHEN 'cancelled' THEN 1
      WHEN 'running' THEN 2
      WHEN 'pending' THEN 3
      ELSE 4
    END,
    CASE WHEN pipeline_run_id IS NULL OR pipeline_run_id = '' THEN 1 ELSE 0 END,
    finished_at DESC NULLS LAST,
    started_at DESC NULLS LAST,
    created_at DESC
) bi
INNER JOIN pipeline_runs pr ON pr.id = bi.pipeline_run_id
WHERE bi.job_id = $1`
	db := dbFromCtx(ctx, r.c.db)
	var total int
	if err := db.QueryRow(ctx, q, jobID).Scan(&total); err != nil {
		return 0, fmt.Errorf("postgres BackfillRepo.CountPipelineRunsByBatchJobID: %w", err)
	}
	return total, nil
}

func (r *BackfillRepo) CountRunsWithNodeRowsByBatchJobID(ctx context.Context, jobID string) (int, error) {
	const q = `
SELECT COUNT(DISTINCT bi.id)
FROM (
  SELECT DISTINCT ON (job_id, asset_id) *
  FROM backfill_items
  WHERE job_id = $1
  ORDER BY job_id, asset_id,
    CASE status
      WHEN 'completed' THEN 0
      WHEN 'failed' THEN 1
      WHEN 'cancelled' THEN 1
      WHEN 'running' THEN 2
      WHEN 'pending' THEN 3
      ELSE 4
    END,
    CASE WHEN pipeline_run_id IS NULL OR pipeline_run_id = '' THEN 1 ELSE 0 END,
    finished_at DESC NULLS LAST,
    started_at DESC NULLS LAST,
    created_at DESC
) bi
INNER JOIN pipeline_runs pr ON pr.id = bi.pipeline_run_id
INNER JOIN pipeline_run_asset_nodes n ON n.run_id = pr.id AND n.asset_id = bi.asset_id
WHERE bi.job_id = $1`
	db := dbFromCtx(ctx, r.c.db)
	var total int
	if err := db.QueryRow(ctx, q, jobID).Scan(&total); err != nil {
		return 0, fmt.Errorf("postgres BackfillRepo.CountRunsWithNodeRowsByBatchJobID: %w", err)
	}
	return total, nil
}

func (r *BackfillRepo) FindItemsByAssetID(ctx context.Context, assetID string) ([]models.BackfillItem, error) {
	q := `SELECT ` + backfillItemSelectCols + `
	FROM backfill_items
	WHERE asset_id = $1
	ORDER BY created_at DESC`
	db := dbFromCtx(ctx, r.c.db)
	rows, err := db.Query(ctx, q, assetID)
	if err != nil {
		return nil, fmt.Errorf("postgres BackfillRepo.FindItemsByAssetID: %w", err)
	}
	defer rows.Close()
	var out []models.BackfillItem
	for rows.Next() {
		item, err := scanBackfillItem(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres BackfillRepo.FindItemsByAssetID scan: %w", err)
		}
		out = append(out, *item)
	}
	return out, nil
}

func (r *BackfillRepo) FindItemByJobAndAssetID(ctx context.Context, jobID, assetID string) (*models.BackfillItem, error) {
	const q = `
SELECT ` + backfillItemSelectCols + `
FROM (
  SELECT DISTINCT ON (job_id, asset_id) *
  FROM backfill_items
  WHERE job_id = $1 AND asset_id = $2
  ORDER BY job_id, asset_id,
    CASE status
      WHEN 'completed' THEN 0
      WHEN 'failed' THEN 1
      WHEN 'cancelled' THEN 1
      WHEN 'running' THEN 2
      WHEN 'pending' THEN 3
      ELSE 4
    END,
    CASE WHEN pipeline_run_id IS NULL OR pipeline_run_id = '' THEN 1 ELSE 0 END,
    finished_at DESC NULLS LAST,
    started_at DESC NULLS LAST,
    created_at DESC
) bi`
	db := dbFromCtx(ctx, r.c.db)
	item, err := scanBackfillItem(db.QueryRow(ctx, q, jobID, assetID))
	if err != nil {
		if errors.Is(err, errNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres BackfillRepo.FindItemByJobAndAssetID: %w", err)
	}
	return item, nil
}

// ClaimNextItem atomically claims one pending item using FOR UPDATE SKIP LOCKED.
func (r *BackfillRepo) ClaimNextItem(ctx context.Context, jobID string) (*models.BackfillItem, error) {
	const q = `
	UPDATE backfill_items
	SET status = 'running',
	    started_at = NOW(),
	    attempts = attempts + 1
	WHERE id = (
		SELECT id FROM backfill_items
		WHERE job_id = $1 AND status = 'pending'
		ORDER BY created_at ASC
		LIMIT 1
		FOR UPDATE SKIP LOCKED
	)
	RETURNING id, job_id, asset_id, status,
	  pipeline_run_id, workflow_name, error_message, attempts, started_at, finished_at,
	  created_at`
	db := dbFromCtx(ctx, r.c.db)
	item, err := scanBackfillItem(db.QueryRow(ctx, q, jobID))
	if err != nil {
		if errors.Is(err, errNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres BackfillRepo.ClaimNextItem: %w", err)
	}
	return item, nil
}

// ResetStaleItems reclaims items stuck in running status beyond lease timeout.
func (r *BackfillRepo) ResetStaleItems(ctx context.Context, leaseTimeoutSec int, maxAttempts int) (int, error) {
	const q = `
	WITH reclaimed AS (
		UPDATE backfill_items bi
		SET status = CASE WHEN bi.attempts >= $2 THEN 'failed' ELSE 'pending' END,
		    started_at = NULL,
		    error_message = CASE WHEN bi.attempts >= $2
			  THEN 'max attempts exceeded after lease timeout'
			  ELSE 'reclaimed: lease expired'
		    END
		FROM backfill_jobs bj
		WHERE bi.job_id = bj.id
		  AND bi.status = 'running'
		  AND bi.started_at < NOW() - ($1 || ' seconds')::interval
		  AND bj.status = 'running'
		RETURNING bi.id
	)
	SELECT COUNT(*) FROM reclaimed`
	db := dbFromCtx(ctx, r.c.db)
	var count int
	if err := db.QueryRow(ctx, q, leaseTimeoutSec, maxAttempts).Scan(&count); err != nil {
		return 0, fmt.Errorf("postgres BackfillRepo.ResetStaleItems: %w", err)
	}
	return count, nil
}

// FindIncompleteJobs returns running backfill jobs with at least one pending item.
func (r *BackfillRepo) FindIncompleteJobs(ctx context.Context) ([]models.BackfillJob, error) {
	const q = `
	SELECT DISTINCT bj.id, bj.template_id, bj.name, bj.status,
	  bj.completed_count, bj.failed_count, bj.total_count,
	  bj.pilot_phase, bj.pilot_count,
	  bj.filter_json, bj.created_at, bj.updated_at
	FROM backfill_jobs bj
	JOIN backfill_items bi ON bi.job_id = bj.id
	WHERE bj.status = 'running'
	  AND bi.status = 'pending'
	ORDER BY bj.created_at ASC`
	db := dbFromCtx(ctx, r.c.db)
	rows, err := db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("postgres BackfillRepo.FindIncompleteJobs: %w", err)
	}
	defer rows.Close()
	var jobs []models.BackfillJob
	for rows.Next() {
		var j models.BackfillJob
		if err := rows.Scan(
			&j.ID, &j.TemplateID, &j.Name, &j.Status,
			&j.CompletedCount, &j.FailedCount, &j.TotalCount,
			&j.PilotPhase, &j.PilotCount,
			&j.FilterJSON, &j.CreatedAt, &j.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("postgres BackfillRepo.FindIncompleteJobs scan: %w", err)
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}

// FindActiveJobs returns non-terminal, non-paused batch jobs (running, pending,
// pilot_running, …) up to limit, oldest first. Unlike FindIncompleteJobs it does
// not require pending items — the reconcile backstop (CYB-3078) must also catch
// jobs whose items are all dispatched and merely waiting on Argo to finish.
func (r *BackfillRepo) FindActiveJobs(ctx context.Context, limit int) ([]models.BackfillJob, error) {
	if limit <= 0 {
		limit = 200
	}
	const q = `
	SELECT bj.id, bj.template_id, bj.name, bj.status,
	  bj.completed_count, bj.failed_count, bj.total_count,
	  bj.pilot_phase, bj.pilot_count,
	  bj.filter_json, bj.created_at, bj.updated_at
	FROM backfill_jobs bj
	WHERE bj.status NOT IN ('completed', 'failed', 'paused')
	ORDER BY bj.created_at ASC
	LIMIT $1`
	db := dbFromCtx(ctx, r.c.db)
	rows, err := db.Query(ctx, q, limit)
	if err != nil {
		return nil, fmt.Errorf("postgres BackfillRepo.FindActiveJobs: %w", err)
	}
	defer rows.Close()
	var jobs []models.BackfillJob
	for rows.Next() {
		var j models.BackfillJob
		if err := rows.Scan(
			&j.ID, &j.TemplateID, &j.Name, &j.Status,
			&j.CompletedCount, &j.FailedCount, &j.TotalCount,
			&j.PilotPhase, &j.PilotCount,
			&j.FilterJSON, &j.CreatedAt, &j.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("postgres BackfillRepo.FindActiveJobs scan: %w", err)
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}
