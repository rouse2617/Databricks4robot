package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

type SearchReindexJobRepo struct {
	c *Client
}

var _ repository.SearchReindexJobRepository = (*SearchReindexJobRepo)(nil)

func NewSearchReindexJobRepo(c *Client) *SearchReindexJobRepo {
	return &SearchReindexJobRepo{c: c}
}

const searchReindexJobColumns = `
id, status, dry_run, page_size, next_page, stop_requested,
total_assets, assets_scanned, documents_indexed, documents_deleted, failed,
error, error_samples, elasticsearch_doc_count, index_cleared,
created_at, updated_at, started_at, finished_at
`

func scanSearchReindexJob(scanner interface{ Scan(dest ...any) error }) (*repository.SearchReindexJob, error) {
	var (
		job                 repository.SearchReindexJob
		status              string
		errorSamplesBytes   []byte
		startedAt           pgtype.Timestamptz
		finishedAt          pgtype.Timestamptz
		elasticsearchDocCnt pgtype.Int8
	)
	if err := scanner.Scan(
		&job.ID,
		&status,
		&job.DryRun,
		&job.PageSize,
		&job.NextPage,
		&job.StopRequested,
		&job.TotalAssets,
		&job.AssetsScanned,
		&job.DocumentsIndexed,
		&job.DocumentsDeleted,
		&job.Failed,
		&job.Error,
		&errorSamplesBytes,
		&elasticsearchDocCnt,
		&job.IndexCleared,
		&job.CreatedAt,
		&job.UpdatedAt,
		&startedAt,
		&finishedAt,
	); err != nil {
		return nil, err
	}
	job.Status = repository.SearchReindexJobStatus(status)
	if len(errorSamplesBytes) > 0 {
		_ = json.Unmarshal(errorSamplesBytes, &job.ErrorSamples)
	}
	if startedAt.Valid {
		t := startedAt.Time.UTC()
		job.StartedAt = &t
	}
	if finishedAt.Valid {
		t := finishedAt.Time.UTC()
		job.FinishedAt = &t
	}
	if elasticsearchDocCnt.Valid {
		v := elasticsearchDocCnt.Int64
		job.ElasticsearchDocCount = &v
	}
	return &job, nil
}

func (r *SearchReindexJobRepo) Create(ctx context.Context, dryRun bool, pageSize int) (*repository.SearchReindexJob, error) {
	db := dbFromCtx(ctx, r.c.db)
	id := "rj_" + uuid.NewString()
	row := db.QueryRow(ctx, fmt.Sprintf(`
INSERT INTO search_reindex_jobs (
	id, status, dry_run, page_size, next_page, stop_requested
) VALUES (
	$1, $2, $3, $4, 1, FALSE
)
RETURNING %s`, searchReindexJobColumns),
		id,
		string(repository.SearchReindexJobStatusQueued),
		dryRun,
		pageSize,
	)
	job, err := scanSearchReindexJob(row)
	if err != nil {
		return nil, fmt.Errorf("postgres SearchReindexJobRepo.Create: %w", err)
	}
	return job, nil
}

func (r *SearchReindexJobRepo) Get(ctx context.Context, jobID string) (*repository.SearchReindexJob, error) {
	db := dbFromCtx(ctx, r.c.db)
	row := db.QueryRow(ctx, fmt.Sprintf(`
SELECT %s
FROM search_reindex_jobs
WHERE id = $1`, searchReindexJobColumns), jobID)
	job, err := scanSearchReindexJob(row)
	if err != nil {
		if err == errNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres SearchReindexJobRepo.Get: %w", err)
	}
	return job, nil
}

func (r *SearchReindexJobRepo) ListRecent(ctx context.Context, limit int) ([]*repository.SearchReindexJob, error) {
	db := dbFromCtx(ctx, r.c.db)
	rows, err := db.Query(ctx, fmt.Sprintf(`
SELECT %s
FROM search_reindex_jobs
ORDER BY created_at DESC
LIMIT $1`, searchReindexJobColumns), limit)
	if err != nil {
		return nil, fmt.Errorf("postgres SearchReindexJobRepo.ListRecent query: %w", err)
	}
	defer rows.Close()
	out := make([]*repository.SearchReindexJob, 0, limit)
	for rows.Next() {
		job, err := scanSearchReindexJob(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres SearchReindexJobRepo.ListRecent scan: %w", err)
		}
		out = append(out, job)
	}
	return out, nil
}

func (r *SearchReindexJobRepo) ClaimForRun(ctx context.Context, jobID string) (*repository.SearchReindexJob, error) {
	db := dbFromCtx(ctx, r.c.db)
	row := db.QueryRow(ctx, fmt.Sprintf(`
UPDATE search_reindex_jobs
SET
	status = $2,
	stop_requested = FALSE,
	started_at = COALESCE(started_at, now()),
	finished_at = NULL,
	updated_at = now()
WHERE id = $1
  AND status IN ($3, $4)
RETURNING %s`, searchReindexJobColumns),
		jobID,
		string(repository.SearchReindexJobStatusRunning),
		string(repository.SearchReindexJobStatusQueued),
		string(repository.SearchReindexJobStatusPaused),
	)
	job, err := scanSearchReindexJob(row)
	if err != nil {
		if err == errNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres SearchReindexJobRepo.ClaimForRun: %w", err)
	}
	return job, nil
}

func (r *SearchReindexJobRepo) RequestStop(ctx context.Context, jobID string) (*repository.SearchReindexJob, error) {
	db := dbFromCtx(ctx, r.c.db)
	row := db.QueryRow(ctx, fmt.Sprintf(`
UPDATE search_reindex_jobs
SET
	stop_requested = TRUE,
	updated_at = now()
WHERE id = $1
RETURNING %s`, searchReindexJobColumns), jobID)
	job, err := scanSearchReindexJob(row)
	if err != nil {
		if err == errNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres SearchReindexJobRepo.RequestStop: %w", err)
	}
	return job, nil
}

func (r *SearchReindexJobRepo) Resume(ctx context.Context, jobID string) (*repository.SearchReindexJob, error) {
	db := dbFromCtx(ctx, r.c.db)
	row := db.QueryRow(ctx, fmt.Sprintf(`
UPDATE search_reindex_jobs
SET
	status = $2,
	stop_requested = FALSE,
	error = '',
	updated_at = now(),
	finished_at = NULL
WHERE id = $1
  AND status IN ($3, $4)
RETURNING %s`, searchReindexJobColumns),
		jobID,
		string(repository.SearchReindexJobStatusQueued),
		string(repository.SearchReindexJobStatusPaused),
		string(repository.SearchReindexJobStatusFailed),
	)
	job, err := scanSearchReindexJob(row)
	if err != nil {
		if err == errNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres SearchReindexJobRepo.Resume: %w", err)
	}
	return job, nil
}

func (r *SearchReindexJobRepo) IsStopRequested(ctx context.Context, jobID string) (bool, error) {
	db := dbFromCtx(ctx, r.c.db)
	var stopRequested bool
	err := db.QueryRow(ctx, `
SELECT stop_requested
FROM search_reindex_jobs
WHERE id = $1`, jobID).Scan(&stopRequested)
	if err != nil {
		if err == errNoRows {
			return false, nil
		}
		return false, fmt.Errorf("postgres SearchReindexJobRepo.IsStopRequested: %w", err)
	}
	return stopRequested, nil
}

func (r *SearchReindexJobRepo) UpdateProgress(ctx context.Context, jobID string, progress repository.SearchReindexJobProgress) error {
	db := dbFromCtx(ctx, r.c.db)
	errorSamplesJSON, _ := json.Marshal(progress.ErrorSamples)
	if errorSamplesJSON == nil {
		errorSamplesJSON = []byte(`[]`)
	}
	if err := db.Exec(ctx, `
UPDATE search_reindex_jobs
SET
	total_assets = $2,
	next_page = $3,
	assets_scanned = $4,
	documents_indexed = $5,
	documents_deleted = $6,
	failed = $7,
	error_samples = $8::jsonb,
	index_cleared = $9,
	updated_at = now()
WHERE id = $1`,
		jobID,
		progress.TotalAssets,
		progress.NextPage,
		progress.AssetsScanned,
		progress.DocumentsIndexed,
		progress.DocumentsDeleted,
		progress.Failed,
		string(errorSamplesJSON),
		progress.IndexCleared,
	); err != nil {
		return fmt.Errorf("postgres SearchReindexJobRepo.UpdateProgress: %w", err)
	}
	return nil
}

func (r *SearchReindexJobRepo) MarkPaused(ctx context.Context, jobID string) error {
	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, `
UPDATE search_reindex_jobs
SET
	status = $2,
	finished_at = now(),
	updated_at = now()
WHERE id = $1`,
		jobID,
		string(repository.SearchReindexJobStatusPaused),
	); err != nil {
		return fmt.Errorf("postgres SearchReindexJobRepo.MarkPaused: %w", err)
	}
	return nil
}

func (r *SearchReindexJobRepo) MarkFailed(ctx context.Context, jobID string, errorMsg string, errorSamples []string) error {
	db := dbFromCtx(ctx, r.c.db)
	errorSamplesJSON, _ := json.Marshal(errorSamples)
	if errorSamplesJSON == nil {
		errorSamplesJSON = []byte(`[]`)
	}
	if err := db.Exec(ctx, `
UPDATE search_reindex_jobs
SET
	status = $2,
	error = $3,
	error_samples = $4::jsonb,
	finished_at = now(),
	updated_at = now()
WHERE id = $1`,
		jobID,
		string(repository.SearchReindexJobStatusFailed),
		errorMsg,
		string(errorSamplesJSON),
	); err != nil {
		return fmt.Errorf("postgres SearchReindexJobRepo.MarkFailed: %w", err)
	}
	return nil
}

func (r *SearchReindexJobRepo) MarkSucceeded(ctx context.Context, jobID string, esDocCount *int64) error {
	db := dbFromCtx(ctx, r.c.db)
	var esCount any
	if esDocCount != nil {
		esCount = *esDocCount
	}
	if err := db.Exec(ctx, `
UPDATE search_reindex_jobs
SET
	status = $2,
	error = '',
	stop_requested = FALSE,
	elasticsearch_doc_count = $3,
	finished_at = now(),
	updated_at = now()
WHERE id = $1`,
		jobID,
		string(repository.SearchReindexJobStatusSucceeded),
		esCount,
	); err != nil {
		return fmt.Errorf("postgres SearchReindexJobRepo.MarkSucceeded: %w", err)
	}
	return nil
}

// MarkAbandoned moves a paused job to abandoned state.
func (r *SearchReindexJobRepo) MarkAbandoned(ctx context.Context, jobID string) error {
	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, `
		UPDATE search_reindex_jobs
		SET status = $2, finished_at = now(), updated_at = now()
		WHERE id = $1 AND status = $3`,
		jobID,
		string(repository.SearchReindexJobStatusAbandoned),
		string(repository.SearchReindexJobStatusPaused),
	); err != nil {
		return fmt.Errorf("postgres SearchReindexJobRepo.MarkAbandoned: %w", err)
	}
	return nil
}

func (r *SearchReindexJobRepo) MarkFailedByTimeout(ctx context.Context, olderThan time.Duration) error {
	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, `
UPDATE search_reindex_jobs
SET
	status = $1,
	error = 'runner interrupted before completion',
	finished_at = now(),
	updated_at = now()
WHERE status = $2
  AND updated_at < now() - ($3::text)::interval`,
		string(repository.SearchReindexJobStatusFailed),
		string(repository.SearchReindexJobStatusRunning),
		olderThan.String(),
	); err != nil {
		return fmt.Errorf("postgres SearchReindexJobRepo.MarkFailedByTimeout: %w", err)
	}
	return nil
}
