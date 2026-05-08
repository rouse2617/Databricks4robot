package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"data-platform/internal/metrics"
)

// ─── models ──────────────────────────────────────────────────────────────────

// EvalResult mirrors the asset_eval_results row.
type EvalResult struct {
	EvalResultID     string          `json:"eval_result_id"`
	AssetID          string          `json:"asset_id"`
	McapFileID       *string         `json:"mcap_file_id,omitempty"`
	TargetType       string          `json:"target_type"`
	TargetID         string          `json:"target_id"`
	EvalName         string          `json:"eval_name"`
	EvalVersion      string          `json:"eval_version"`
	ParameterVersion *string         `json:"parameter_version,omitempty"`
	RunID            *string         `json:"run_id,omitempty"`
	Status           string          `json:"status"`
	ResultPayload    json.RawMessage `json:"result_payload"`
	OutputURI        *string         `json:"output_uri,omitempty"`
	SummaryURI       *string         `json:"summary_uri,omitempty"`
	SourceType       string          `json:"source_type"`
	SourceName       *string         `json:"source_name,omitempty"`
	SourceVersion    *string         `json:"source_version,omitempty"`
	StartedAt        *time.Time      `json:"started_at,omitempty"`
	FinishedAt       *time.Time      `json:"finished_at,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

// AssetMetric mirrors the asset_metrics row.
type AssetMetric struct {
	AssetID          string    `json:"asset_id"`
	TargetType       string    `json:"target_type"`
	TargetID         string    `json:"target_id"`
	MetricKey        string    `json:"metric_key"`
	MetricType       string    `json:"metric_type"`
	MetricUnit       *string   `json:"metric_unit,omitempty"`
	MetricValue      *float64  `json:"metric_value,omitempty"`
	MetricValueInt   *int64    `json:"metric_value_int,omitempty"`
	MetricValueText  *string   `json:"metric_value_text,omitempty"`
	MetricValueBool  *bool     `json:"metric_value_bool,omitempty"`
	EvalName         string    `json:"eval_name"`
	EvalVersion      string    `json:"eval_version"`
	ParameterVersion *string   `json:"parameter_version,omitempty"`
	RunID            *string   `json:"run_id,omitempty"`
	SourceType       string    `json:"source_type"`
	SourceName       *string   `json:"source_name,omitempty"`
	Confidence       *float64  `json:"confidence,omitempty"`
	RecordedAt       time.Time `json:"recorded_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// EvalResultWriteInput is the input to EvalRepo.Write.
type EvalResultWriteInput struct {
	AssetID          string
	McapFileID       *string
	TargetType       string
	TargetID         string
	EvalName         string
	EvalVersion      string
	ParameterVersion *string
	RunID            *string
	Status           string
	ResultPayload    map[string]any
	OutputURI        *string
	SummaryURI       *string
	SourceType       string
	SourceName       *string
	SourceVersion    *string
	StartedAt        *time.Time
	FinishedAt       *time.Time
	// QueryableKeys: metric_key → metric_type (only these are projected to asset_metrics)
	QueryableKeys map[string]string
}

// ─── EvalRepo ────────────────────────────────────────────────────────────────

type EvalRepo struct {
	c *Client
}

func NewEvalRepo(c *Client) *EvalRepo { return &EvalRepo{c: c} }

// Write inserts an eval result and upserts asset_metrics projections in one transaction.
func (r *EvalRepo) Write(ctx context.Context, in EvalResultWriteInput) (*EvalResult, error) {
	payloadJSON, err := json.Marshal(in.ResultPayload)
	if err != nil {
		return nil, fmt.Errorf("eval_repo: marshal payload: %w", err)
	}
	targetType := in.TargetType
	if targetType == "" {
		targetType = "segment"
	}
	sourceType := in.SourceType
	if sourceType == "" {
		sourceType = "algo"
	}

	var res EvalResult

	txErr := r.c.WithTx(ctx, func(txCtx context.Context) error {
		db := dbFromCtx(txCtx, r.c.db)

		// 1. Insert asset_eval_results
		const insertEval = `
INSERT INTO asset_eval_results
    (asset_id, mcap_file_id, target_type, target_id,
     eval_name, eval_version, parameter_version, run_id, status,
     result_payload, output_uri, summary_uri,
     source_type, source_name, source_version,
     started_at, finished_at)
VALUES
    ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
RETURNING
    eval_result_id, asset_id, mcap_file_id, target_type, target_id,
    eval_name, eval_version, parameter_version, run_id, status,
    result_payload, output_uri, summary_uri,
    source_type, source_name, source_version,
    started_at, finished_at, created_at, updated_at`

		if err := db.QueryRow(txCtx, insertEval,
			in.AssetID, in.McapFileID, targetType, in.TargetID,
			in.EvalName, in.EvalVersion, in.ParameterVersion, in.RunID, in.Status,
			payloadJSON, in.OutputURI, in.SummaryURI,
			sourceType, in.SourceName, in.SourceVersion,
			in.StartedAt, in.FinishedAt,
		).Scan(
			&res.EvalResultID, &res.AssetID, &res.McapFileID, &res.TargetType, &res.TargetID,
			&res.EvalName, &res.EvalVersion, &res.ParameterVersion, &res.RunID, &res.Status,
			&res.ResultPayload, &res.OutputURI, &res.SummaryURI,
			&res.SourceType, &res.SourceName, &res.SourceVersion,
			&res.StartedAt, &res.FinishedAt, &res.CreatedAt, &res.UpdatedAt,
		); err != nil {
			return fmt.Errorf("insert eval_result: %w", err)
		}

		// 2. Upsert queryable metric projections
		const upsertMetric = `
INSERT INTO asset_metrics
    (asset_id, target_type, target_id, metric_key, metric_type,
     eval_name, eval_version, parameter_version, run_id,
     source_type, source_name,
     metric_value, metric_value_int, metric_value_text, metric_value_bool)
VALUES
    ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
ON CONFLICT (asset_id, target_type, target_id, metric_key, eval_name, eval_version)
DO UPDATE SET
    metric_value      = EXCLUDED.metric_value,
    metric_value_int  = EXCLUDED.metric_value_int,
    metric_value_text = EXCLUDED.metric_value_text,
    metric_value_bool = EXCLUDED.metric_value_bool,
    parameter_version = EXCLUDED.parameter_version,
    run_id            = EXCLUDED.run_id,
    updated_at        = now()`

		for key, mType := range in.QueryableKeys {
			rawVal, ok := in.ResultPayload[key]
			if !ok {
				continue
			}
			fval, ival, tval, bval := extractMetricValues(rawVal)
			upsertStart := time.Now()
			if err := db.Exec(txCtx, upsertMetric,
				in.AssetID, targetType, in.TargetID, key, mType,
				in.EvalName, in.EvalVersion, in.ParameterVersion, in.RunID,
				sourceType, in.SourceName,
				fval, ival, tval, bval,
			); err != nil {
				metrics.BackendEvalMetricUpsertsTotal.WithLabelValues("error").Inc()
				metrics.BackendEvalMetricUpsertDurationSeconds.WithLabelValues("error").Observe(time.Since(upsertStart).Seconds())
				return fmt.Errorf("upsert metric %q: %w", key, err)
			}
			metrics.BackendEvalMetricUpsertsTotal.WithLabelValues("ok").Inc()
			metrics.BackendEvalMetricUpsertDurationSeconds.WithLabelValues("ok").Observe(time.Since(upsertStart).Seconds())
		}
		return nil
	})
	if txErr != nil {
		return nil, fmt.Errorf("eval_repo.Write: %w", txErr)
	}
	return &res, nil
}

// ListByAsset returns eval results for an asset, newest first.
func (r *EvalRepo) ListByAsset(ctx context.Context, assetID string, limit int) ([]*EvalResult, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	const q = `
SELECT eval_result_id, asset_id, mcap_file_id, target_type, target_id,
       eval_name, eval_version, parameter_version, run_id, status,
       result_payload, output_uri, summary_uri,
       source_type, source_name, source_version,
       started_at, finished_at, created_at, updated_at
FROM asset_eval_results
WHERE asset_id = $1
ORDER BY created_at DESC
LIMIT $2`

	rows, err := r.c.db.Query(ctx, q, assetID, limit)
	if err != nil {
		return nil, fmt.Errorf("eval_repo.ListByAsset: %w", err)
	}
	defer rows.Close()

	var out []*EvalResult
	for rows.Next() {
		var res EvalResult
		if err := rows.Scan(
			&res.EvalResultID, &res.AssetID, &res.McapFileID, &res.TargetType, &res.TargetID,
			&res.EvalName, &res.EvalVersion, &res.ParameterVersion, &res.RunID, &res.Status,
			&res.ResultPayload, &res.OutputURI, &res.SummaryURI,
			&res.SourceType, &res.SourceName, &res.SourceVersion,
			&res.StartedAt, &res.FinishedAt, &res.CreatedAt, &res.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("eval_repo.ListByAsset scan: %w", err)
		}
		out = append(out, &res)
	}
	return out, nil
}

// ListMetricsByAsset returns metric projections for an asset.
func (r *EvalRepo) ListMetricsByAsset(ctx context.Context, assetID string) ([]*AssetMetric, error) {
	const q = `
SELECT asset_id, target_type, target_id, metric_key, metric_type, metric_unit,
       metric_value, metric_value_int, metric_value_text, metric_value_bool,
       eval_name, eval_version, parameter_version, run_id,
       source_type, source_name, confidence,
       recorded_at, updated_at
FROM asset_metrics
WHERE asset_id = $1
ORDER BY metric_key, eval_version DESC`

	rows, err := r.c.db.Query(ctx, q, assetID)
	if err != nil {
		return nil, fmt.Errorf("eval_repo.ListMetricsByAsset: %w", err)
	}
	defer rows.Close()

	var out []*AssetMetric
	for rows.Next() {
		var m AssetMetric
		if err := rows.Scan(
			&m.AssetID, &m.TargetType, &m.TargetID, &m.MetricKey, &m.MetricType, &m.MetricUnit,
			&m.MetricValue, &m.MetricValueInt, &m.MetricValueText, &m.MetricValueBool,
			&m.EvalName, &m.EvalVersion, &m.ParameterVersion, &m.RunID,
			&m.SourceType, &m.SourceName, &m.Confidence,
			&m.RecordedAt, &m.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("eval_repo.ListMetricsByAsset scan: %w", err)
		}
		out = append(out, &m)
	}
	return out, nil
}

// SearchByMetric returns asset_ids whose metric passes the given numeric filter.
// op must be one of: >=, <=, >, <, =
func (r *EvalRepo) SearchByMetric(ctx context.Context, metricKey, op string, value float64, lifecycleState string, page, pageSize int) ([]string, int, error) {
	validOps := map[string]string{">=": ">=", "<=": "<=", ">": ">", "<": "<", "=": "="}
	sqlOp, ok := validOps[op]
	if !ok {
		return nil, 0, fmt.Errorf("eval_repo: invalid op %q", op)
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	countQ := fmt.Sprintf(`
SELECT COUNT(DISTINCT m.asset_id)
FROM asset_metrics m
JOIN assets a ON a.asset_id::text = m.asset_id
WHERE m.metric_key = $1
  AND m.metric_value %s $2
  AND a.is_deleted = FALSE
  AND ($3 = '' OR a.lifecycle_state = $3)`, sqlOp)

	var total int
	if err := r.c.db.QueryRow(ctx, countQ, metricKey, value, lifecycleState).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("eval_repo.SearchByMetric count: %w", err)
	}

	dataQ := fmt.Sprintf(`
SELECT DISTINCT ON (m.asset_id) m.asset_id
FROM asset_metrics m
JOIN assets a ON a.asset_id::text = m.asset_id
WHERE m.metric_key = $1
  AND m.metric_value %s $2
  AND a.is_deleted = FALSE
  AND ($3 = '' OR a.lifecycle_state = $3)
ORDER BY m.asset_id, m.updated_at DESC
LIMIT $4 OFFSET $5`, sqlOp)

	rows, err := r.c.db.Query(ctx, dataQ, metricKey, value, lifecycleState, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("eval_repo.SearchByMetric: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, 0, fmt.Errorf("eval_repo.SearchByMetric scan: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, total, nil
}

// MetricFilter describes one numeric metric constraint used by SearchByMetrics.
type MetricFilter struct {
	MetricKey string
	Op        string
	Value     float64
}

// SearchByMetrics returns asset_ids that satisfy ALL metric filters.
func (r *EvalRepo) SearchByMetrics(ctx context.Context, filters []MetricFilter, lifecycleState string, page, pageSize int) ([]string, int, error) {
	if len(filters) == 0 {
		return nil, 0, fmt.Errorf("eval_repo: at least one metric filter required")
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	validOps := map[string]string{
		">=": ">=", "<=": "<=", ">": ">", "<": "<", "=": "=",
		"gte": ">=", "lte": "<=", "gt": ">", "lt": "<", "eq": "=",
	}

	var conds []string
	args := make([]any, 0, len(filters)*3+3)
	argPos := 1
	for _, f := range filters {
		key := strings.TrimSpace(f.MetricKey)
		if key == "" {
			return nil, 0, fmt.Errorf("eval_repo: metric_key cannot be empty")
		}
		sqlOp, ok := validOps[strings.ToLower(strings.TrimSpace(f.Op))]
		if !ok {
			return nil, 0, fmt.Errorf("eval_repo: invalid op %q", f.Op)
		}
		conds = append(conds, fmt.Sprintf(
			`EXISTS (
  SELECT 1 FROM asset_metrics m
  WHERE m.asset_id = a.asset_id::text
    AND m.metric_key = $%d
    AND m.metric_value %s $%d
)`, argPos, sqlOp, argPos+1))
		args = append(args, key, f.Value)
		argPos += 2
	}
	lifecyclePos := argPos
	args = append(args, lifecycleState)

	where := "a.is_deleted = FALSE"
	if len(conds) > 0 {
		where += " AND " + strings.Join(conds, " AND ")
	}
	where += fmt.Sprintf(" AND ($%d = '' OR a.lifecycle_state = $%d)", lifecyclePos, lifecyclePos)

	countQ := fmt.Sprintf("SELECT COUNT(*) FROM assets a WHERE %s", where)
	var total int
	if err := r.c.db.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("eval_repo.SearchByMetrics count: %w", err)
	}

	dataQ := fmt.Sprintf(`
SELECT a.asset_id
FROM assets a
WHERE %s
ORDER BY a.updated_at DESC, a.asset_id
LIMIT $%d OFFSET $%d`, where, lifecyclePos+1, lifecyclePos+2)
	dataArgs := append(append([]any{}, args...), pageSize, offset)

	rows, err := r.c.db.Query(ctx, dataQ, dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("eval_repo.SearchByMetrics query: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, 0, fmt.Errorf("eval_repo.SearchByMetrics scan: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, total, nil
}

// extractMetricValues splits a raw interface value into typed metric columns.
func extractMetricValues(v any) (fval *float64, ival *int64, tval *string, bval *bool) {
	switch n := v.(type) {
	case float64:
		fval = &n
	case float32:
		f := float64(n)
		fval = &f
	case int:
		i := int64(n)
		ival = &i
	case int64:
		ival = &n
	case bool:
		bval = &n
	case string:
		tval = &n
	case json.Number:
		f, err := n.Float64()
		if err == nil {
			fval = &f
		}
	}
	return
}
