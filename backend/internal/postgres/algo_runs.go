package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

func (r *AlgoRunRepo) Get(ctx context.Context, runID string) (*models.AlgoRun, error) {
	const q = `
SELECT
  run_id, algo_name, algo_version, algo_kind, triggered_by, status,
  started_at, finished_at, duration_ns,
  input_filter, input_asset_ids, params,
  code_commit, image_digest, pipeline_name, pipeline_version,
  assets_processed, assets_succeeded, assets_failed,
  actions_created, metrics_written, outputs,
  cpu_seconds, gpu_seconds, cost_usd_micros,
  error_class, error_message, tenant_id, project_id,
  created_at, updated_at, row_version
FROM algo_runs WHERE run_id = $1`
	var (
		run           models.AlgoRun
		inputFilter   []byte
		params        []byte
		outputs       []byte
		codeCommit    *string
		imageDigest   *string
		pipelineName  *string
		pipelineVer   *string
		errorClass    *string
		errorMessage  *string
		tenantID      *string
		projectID     *string
	)
	err := r.c.db.QueryRow(ctx, q, runID).Scan(
		&run.RunID, &run.AlgoName, &run.AlgoVersion, &run.AlgoKind, &run.TriggeredBy, &run.Status,
		&run.StartedAt, &run.FinishedAt, &run.DurationNs,
		&inputFilter, &run.InputAssetIDs, &params,
		&codeCommit, &imageDigest, &pipelineName, &pipelineVer,
		&run.AssetsProcessed, &run.AssetsSucceeded, &run.AssetsFailed,
		&run.ActionsCreated, &run.MetricsWritten, &outputs,
		&run.CPUSeconds, &run.GPUSeconds, &run.CostUSDMicros,
		&errorClass, &errorMessage, &tenantID, &projectID,
		&run.CreatedAt, &run.UpdatedAt, &run.RowVersion,
	)
	if err != nil {
		if errors.Is(err, errNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres AlgoRunRepo.Get: %w", err)
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
	return &run, nil
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
