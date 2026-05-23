package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

type AlgoRunRepo struct {
	c *Client
}

var _ repository.AlgoRunRepository = (*AlgoRunRepo)(nil)

func NewAlgoRunRepo(c *Client) *AlgoRunRepo {
	return &AlgoRunRepo{c: c}
}

func (r *AlgoRunRepo) Insert(ctx context.Context, run *models.AlgoRun) error {
	now := time.Now().UTC()
	if run.CreatedAt.IsZero() {
		run.CreatedAt = now
	}
	run.UpdatedAt = now
	if run.Status == "" {
		run.Status = models.AlgoRunStatusPending
	}
	inputFilter, params, outputs := algoRunJSONFields(run)

	const q = `
INSERT INTO algo_runs (
  run_id, algo_name, algo_version, algo_kind, triggered_by, status,
  input_filter, input_asset_ids, params,
  code_commit, image_digest, pipeline_name, pipeline_version,
  outputs, tenant_id, project_id,
  created_at, updated_at, row_version
) VALUES (
  $1,$2,$3,$4,$5,$6,
  $7::jsonb,$8,$9::jsonb,
  $10,$11,$12,$13,
  $14::jsonb,$15,$16,
  $17,$18,$19
)`
	err := r.c.db.Exec(ctx, q,
		run.RunID, run.AlgoName, run.AlgoVersion, run.AlgoKind, run.TriggeredBy, run.Status,
		inputFilter, run.InputAssetIDs, params,
		nullIfEmpty(run.CodeCommit), nullIfEmpty(run.ImageDigest),
		nullIfEmpty(run.PipelineName), nullIfEmpty(run.PipelineVersion),
		outputs, nullIfEmpty(run.TenantID), nullIfEmpty(run.ProjectID),
		run.CreatedAt, run.UpdatedAt, run.RowVersion,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return repository.ErrDuplicateRunID
		}
		return fmt.Errorf("postgres AlgoRunRepo.Insert: %w", err)
	}
	return nil
}

const algoRunSelectCols = `
  run_id, algo_name, algo_version, algo_kind, triggered_by, status,
  started_at, finished_at, duration_ns,
  input_filter, input_asset_ids, params,
  code_commit, image_digest, pipeline_name, pipeline_version,
  assets_processed, assets_succeeded, assets_failed,
  actions_created, metrics_written, outputs,
  cpu_seconds, gpu_seconds, cost_usd_micros,
  error_class, error_message, tenant_id, project_id,
  external_runtime, external_url,
  created_at, updated_at, row_version`

func (r *AlgoRunRepo) scanAlgoRun(ctx context.Context, row rowScanner) (*models.AlgoRun, error) {
	var (
		run             models.AlgoRun
		inputFilter     []byte
		params          []byte
		outputs         []byte
		codeCommit      *string
		imageDigest     *string
		pipelineName    *string
		pipelineVer     *string
		errorClass      *string
		errorMessage    *string
		tenantID        *string
		projectID       *string
		externalRuntime *string
		externalUrl     *string
	)
	err := row.Scan(
		&run.RunID, &run.AlgoName, &run.AlgoVersion, &run.AlgoKind, &run.TriggeredBy, &run.Status,
		&run.StartedAt, &run.FinishedAt, &run.DurationNs,
		&inputFilter, &run.InputAssetIDs, &params,
		&codeCommit, &imageDigest, &pipelineName, &pipelineVer,
		&run.AssetsProcessed, &run.AssetsSucceeded, &run.AssetsFailed,
		&run.ActionsCreated, &run.MetricsWritten, &outputs,
		&run.CPUSeconds, &run.GPUSeconds, &run.CostUSDMicros,
		&errorClass, &errorMessage, &tenantID, &projectID,
		&externalRuntime, &externalUrl,
		&run.CreatedAt, &run.UpdatedAt, &run.RowVersion,
	)
	if err != nil {
		if errors.Is(err, errNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres AlgoRunRepo.scan: %w", err)
	}
	decodeAlgoRunJSON(&run, inputFilter, params, outputs)
	run.CodeCommit = derefStr(codeCommit)
	run.ImageDigest = derefStr(imageDigest)
	run.PipelineName = derefStr(pipelineName)
	run.PipelineVersion = derefStr(pipelineVer)
	run.ErrorClass = derefStr(errorClass)
	run.ErrorMessage = derefStr(errorMessage)
	run.TenantID = derefStr(tenantID)
	run.ProjectID = derefStr(projectID)
	run.ExternalRuntime = externalRuntime
	run.ExternalUrl = externalUrl
	return &run, nil
}

func (r *AlgoRunRepo) Get(ctx context.Context, runID string) (*models.AlgoRun, error) {
	return r.scanAlgoRun(ctx, r.c.db.QueryRow(ctx, "SELECT "+algoRunSelectCols+" FROM algo_runs WHERE run_id = $1", runID))
}

func (r *AlgoRunRepo) Exists(ctx context.Context, runID string) (bool, error) {
	const q = `SELECT EXISTS (SELECT 1 FROM algo_runs WHERE run_id = $1)`
	var ok bool
	if err := r.c.db.QueryRow(ctx, q, runID).Scan(&ok); err != nil {
		return false, fmt.Errorf("postgres AlgoRunRepo.Exists: %w", err)
	}
	return ok, nil
}

func (r *AlgoRunRepo) Start(ctx context.Context, runID string, startedAt time.Time) error {
	const q = `
UPDATE algo_runs SET
  status = $2,
  started_at = $3,
  updated_at = $4,
  row_version = row_version + 1
WHERE run_id = $1 AND status = $5`
	rows, err := r.c.db.ExecResult(ctx, q,
		runID, models.AlgoRunStatusRunning, startedAt, time.Now().UTC(),
		models.AlgoRunStatusPending,
	)
	if err != nil {
		return fmt.Errorf("postgres AlgoRunRepo.Start: %w", err)
	}
	if rows == 0 {
		cur, gerr := r.Get(ctx, runID)
		if gerr != nil {
			return gerr
		}
		if cur == nil {
			return repository.ErrAlgoRunNotFound
		}
		if cur.Status == models.AlgoRunStatusRunning {
			return nil
		}
		return repository.ErrAlgoRunBadState
	}
	return nil
}

func (r *AlgoRunRepo) Finish(ctx context.Context, runID string, patch repository.AlgoRunFinishPatch) error {
	outputs, _ := json.Marshal(patch.Outputs)
	if patch.Outputs == nil {
		outputs = []byte("{}")
	}
	const q = `
UPDATE algo_runs SET
  status = $2,
  finished_at = $3,
  assets_processed = $4,
  assets_succeeded = $5,
  assets_failed = $6,
  actions_created = $7,
  metrics_written = $8,
  outputs = $9::jsonb,
  cpu_seconds = $10,
  gpu_seconds = $11,
  cost_usd_micros = $12,
  error_class = $13,
  error_message = $14,
  updated_at = $15,
  row_version = row_version + 1
WHERE run_id = $1 AND status = $16`
	rows, err := r.c.db.ExecResult(ctx, q,
		runID, patch.Status, patch.FinishedAt,
		patch.AssetsProcessed, patch.AssetsSucceeded, patch.AssetsFailed,
		patch.ActionsCreated, patch.MetricsWritten, outputs,
		patch.CPUSeconds, patch.GPUSeconds, patch.CostUSDMicros,
		nullIfEmpty(patch.ErrorClass), nullIfEmpty(patch.ErrorMessage),
		time.Now().UTC(),
		models.AlgoRunStatusRunning,
	)
	if err != nil {
		return fmt.Errorf("postgres AlgoRunRepo.Finish: %w", err)
	}
	if rows == 0 {
		cur, gerr := r.Get(ctx, runID)
		if gerr != nil {
			return gerr
		}
		if cur == nil {
			return repository.ErrAlgoRunNotFound
		}
		if cur.Status == patch.Status {
			return nil
		}
		return repository.ErrAlgoRunBadState
	}
	return nil
}

// List returns algo_runs matching the given filter, ordered by created_at DESC.
func (r *AlgoRunRepo) List(ctx context.Context, filter repository.AlgoRunListFilter) ([]*models.AlgoRun, int64, error) {
	where := []string{"1=1"}
	args := []any{}
	idx := 1

	if filter.AlgoName != "" {
		where = append(where, fmt.Sprintf("algo_name = $%d", idx))
		args = append(args, filter.AlgoName)
		idx++
	}
	if filter.Status != "" {
		where = append(where, fmt.Sprintf("status = $%d", idx))
		args = append(args, filter.Status)
		idx++
	}
	if filter.StartedAfter != nil {
		where = append(where, fmt.Sprintf("started_at >= $%d", idx))
		args = append(args, *filter.StartedAfter)
		idx++
	}
	if filter.StartedBefore != nil {
		where = append(where, fmt.Sprintf("started_at <= $%d", idx))
		args = append(args, *filter.StartedBefore)
		idx++
	}
	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 50
	}
	offset := (page - 1) * pageSize

	countQ := fmt.Sprintf("SELECT COUNT(*) FROM algo_runs WHERE %s", strings.Join(where, " AND "))
	var total int64
	if err := r.c.db.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("postgres AlgoRunRepo.List count: %w", err)
	}

	args = append(args, pageSize, offset)

	q := fmt.Sprintf("SELECT %s FROM algo_runs WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d",
		algoRunSelectCols, strings.Join(where, " AND "), idx, idx+1)

	rows, err := r.c.db.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres AlgoRunRepo.List: %w", err)
	}
	defer rows.Close()

	var out []*models.AlgoRun
	for rows.Next() {
		run, err := r.scanAlgoRun(ctx, rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, run)
	}
	return out, total, nil
}

// Cancel transitions a pending/running run to cancelled with a reason.
func (r *AlgoRunRepo) Cancel(ctx context.Context, runID, reason string, finishedAt time.Time) error {
	const q = `
UPDATE algo_runs SET
  status = $2,
  finished_at = $3,
  error_class = 'cancelled',
  error_message = $4,
  updated_at = $5,
  row_version = row_version + 1
WHERE run_id = $1 AND status IN ($6, $7)`
	rows, err := r.c.db.ExecResult(ctx, q,
		runID, models.AlgoRunStatusCancelled, finishedAt, reason,
		time.Now().UTC(), models.AlgoRunStatusPending, models.AlgoRunStatusRunning,
	)
	if err != nil {
		return fmt.Errorf("postgres AlgoRunRepo.Cancel: %w", err)
	}
	if rows == 0 {
		cur, gerr := r.Get(ctx, runID)
		if gerr != nil {
			return gerr
		}
		if cur == nil {
			return repository.ErrAlgoRunNotFound
		}
		if cur.Status == models.AlgoRunStatusCancelled {
			return nil // already cancelled, idempotent
		}
		return repository.ErrAlgoRunBadState
	}
	return nil
}

// GetAffectedAssets returns all assets in asset_algo_latest that were processed
// by the given run_id.
func (r *AlgoRunRepo) GetAffectedAssets(ctx context.Context, runID string) ([]*repository.AffectedAsset, error) {
	const q = `
SELECT asset_id, algo_name, algo_version, status,
  COALESCE(result_tag, ''), result_score, COALESCE(run_id, ''),
  updated_at
FROM asset_algo_latest
WHERE run_id = $1
ORDER BY asset_id`
	rows, err := r.c.db.Query(ctx, q, runID)
	if err != nil {
		return nil, fmt.Errorf("postgres AlgoRunRepo.GetAffectedAssets: %w", err)
	}
	defer rows.Close()

	var out []*repository.AffectedAsset
	for rows.Next() {
		var a repository.AffectedAsset
		if err := rows.Scan(
			&a.AssetID, &a.AlgoName, &a.AlgoVersion, &a.Status,
			&a.ResultTag, &a.ResultScore, &a.RunID,
			&a.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("postgres AlgoRunRepo.GetAffectedAssets scan: %w", err)
		}
		out = append(out, &a)
	}
	return out, nil
}

func algoRunJSONFields(run *models.AlgoRun) (inputFilter, params, outputs []byte) {
	if run.InputFilter == nil {
		inputFilter = []byte("{}")
	} else {
		inputFilter, _ = json.Marshal(run.InputFilter)
	}
	if run.Params == nil {
		params = []byte("{}")
	} else {
		params, _ = json.Marshal(run.Params)
	}
	if run.Outputs == nil {
		outputs = []byte("{}")
	} else {
		outputs, _ = json.Marshal(run.Outputs)
	}
	return inputFilter, params, outputs
}

func decodeAlgoRunJSON(run *models.AlgoRun, inputFilter, params, outputs []byte) {
	if len(inputFilter) > 0 {
		_ = json.Unmarshal(inputFilter, &run.InputFilter)
	}
	if len(params) > 0 {
		_ = json.Unmarshal(params, &run.Params)
	}
	if len(outputs) > 0 {
		_ = json.Unmarshal(outputs, &run.Outputs)
	}
}

func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
