package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/schedtask"
)

// ScheduledTaskRepo persists scheduled_tasks rows (CYB-3744). Follows the
// same style as DispatcherConfigRepo (pgx via c.db.Query/ExecResult).
type ScheduledTaskRepo struct {
	c *Client
}

func NewScheduledTaskRepo(c *Client) *ScheduledTaskRepo {
	return &ScheduledTaskRepo{c: c}
}

const scheduledTaskCols = `id, name, enabled, template_id, template_version, target_id, scheduling,
	source_type, source_config, trigger_mode, trigger_config, cursor,
	last_run_at, last_run_status, last_batch_id, last_error, last_success_at,
	run_now_requested_at, created_by, created_at, updated_at`

// List returns all rules ordered by name.
func (r *ScheduledTaskRepo) List(ctx context.Context) ([]schedtask.Rule, error) {
	rows, err := r.c.db.Query(ctx, `SELECT `+scheduledTaskCols+` FROM scheduled_tasks ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("scheduled_tasks: list: %w", err)
	}
	defer rows.Close()
	var out []schedtask.Rule
	for rows.Next() {
		rule, err := scanScheduledTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rule)
	}
	return out, rows.Err()
}

// Get returns a single rule by id, or (nil, nil) if absent.
func (r *ScheduledTaskRepo) Get(ctx context.Context, id string) (*schedtask.Rule, error) {
	row := r.c.db.QueryRow(ctx, `SELECT `+scheduledTaskCols+` FROM scheduled_tasks WHERE id = $1`, id)
	rule, err := scanScheduledTask(row)
	if err != nil {
		return nil, fmt.Errorf("scheduled_tasks: get %s: %w", id, err)
	}
	if rule.ID == "" {
		return nil, nil
	}
	return &rule, nil
}

// Create inserts a fresh rule (id/name required upstream).
func (r *ScheduledTaskRepo) Create(ctx context.Context, rule *schedtask.Rule) error {
	if rule.ID == "" || rule.Name == "" {
		return fmt.Errorf("scheduled_tasks: id and name are required")
	}
	_, err := r.c.db.ExecResult(ctx, `
		INSERT INTO scheduled_tasks (
			id, name, enabled, template_id, template_version, target_id, scheduling,
			source_type, source_config, trigger_mode, trigger_config,
			created_by, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12, now(), now())`,
		rule.ID, rule.Name, rule.Enabled, rule.TemplateID, nullableInt(rule.TemplateVersion), rule.TargetID, nullableJSON(rule.Scheduling),
		rule.SourceType, nullableJSON(rule.SourceConfig), string(rule.TriggerMode), nullableJSON(rule.TriggerConfig),
		rule.CreatedBy,
	)
	if err != nil {
		return fmt.Errorf("scheduled_tasks: create %s: %w", rule.ID, err)
	}
	return nil
}

// Update replaces the mutable fields of a rule (definition only — observation
// fields are updated by scheduler paths, not this handler).
func (r *ScheduledTaskRepo) Update(ctx context.Context, rule *schedtask.Rule) error {
	if rule.ID == "" {
		return fmt.Errorf("scheduled_tasks: id is required")
	}
	_, err := r.c.db.ExecResult(ctx, `
		UPDATE scheduled_tasks SET
			name = $2, enabled = $3, template_id = $4, template_version = $5,
			target_id = $6, scheduling = $7,
			source_type = $8, source_config = $9, trigger_mode = $10, trigger_config = $11,
			updated_at = now()
		WHERE id = $1`,
		rule.ID, rule.Name, rule.Enabled, rule.TemplateID, nullableInt(rule.TemplateVersion), rule.TargetID, nullableJSON(rule.Scheduling),
		rule.SourceType, nullableJSON(rule.SourceConfig), string(rule.TriggerMode), nullableJSON(rule.TriggerConfig),
	)
	if err != nil {
		return fmt.Errorf("scheduled_tasks: update %s: %w", rule.ID, err)
	}
	return nil
}

// Delete removes a rule.
func (r *ScheduledTaskRepo) Delete(ctx context.Context, id string) error {
	_, err := r.c.db.ExecResult(ctx, `DELETE FROM scheduled_tasks WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("scheduled_tasks: delete %s: %w", id, err)
	}
	return nil
}

// SetEnabled toggles the enabled flag.
func (r *ScheduledTaskRepo) SetEnabled(ctx context.Context, id string, enabled bool) error {
	_, err := r.c.db.ExecResult(ctx, `UPDATE scheduled_tasks SET enabled = $2, updated_at = now() WHERE id = $1`, id, enabled)
	if err != nil {
		return fmt.Errorf("scheduled_tasks: set_enabled %s: %w", id, err)
	}
	return nil
}

// RequestRunNow sets run_now_requested_at = now(). The scheduler picks it up
// on its next tick regardless of interval.
func (r *ScheduledTaskRepo) RequestRunNow(ctx context.Context, id string) error {
	_, err := r.c.db.ExecResult(ctx, `UPDATE scheduled_tasks SET run_now_requested_at = now(), updated_at = now() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("scheduled_tasks: run_now %s: %w", id, err)
	}
	return nil
}

// ClaimDueRules atomically claims all rules that are due to run right now. A
// row is due when it's enabled AND either:
//   - run_now_requested_at is set, OR
//   - trigger_mode allows scheduling (incremental/rolling) AND the interval
//     since last_run_at has elapsed (last_run_at IS NULL counts as "always
//     overdue" for its first run).
//
// The UPDATE ... RETURNING pattern gives us multi-replica single-runner
// semantics for free: at most one replica sees each row's "just-claimed"
// state, because the WHERE clause references last_run_at that we're about to
// overwrite in the same statement.
//
// intervalSecondsPath is a JSONB path into trigger_config that carries the
// per-rule interval (e.g. {intervalSeconds:3600}); rules without an interval
// default to defaultIntervalSeconds.
func (r *ScheduledTaskRepo) ClaimDueRules(ctx context.Context, now time.Time, defaultIntervalSeconds int) ([]schedtask.Rule, error) {
	rows, err := r.c.db.Query(ctx, `
		WITH due AS (
			SELECT id
			FROM scheduled_tasks
			WHERE enabled = true
			  AND (
			    run_now_requested_at IS NOT NULL
			    OR (
			      trigger_mode IN ('incremental','rolling')
			      AND (
			        last_run_at IS NULL
			        OR EXTRACT(EPOCH FROM ($1::timestamptz - last_run_at))
			           >= COALESCE((trigger_config->>'intervalSeconds')::int, $2)
			      )
			    )
			  )
			ORDER BY last_run_at NULLS FIRST
			-- Bounded fan-out per tick; if you truly have >64 due rules
			-- something upstream is wrong and we don't want to fan out further.
			LIMIT 64
			FOR UPDATE SKIP LOCKED
		)
		UPDATE scheduled_tasks t
		SET last_run_at = $1,
		    updated_at = now(),
		    run_now_requested_at = NULL
		FROM due
		WHERE t.id = due.id
		RETURNING `+prefixed(scheduledTaskCols, "t.")+``,
		now, defaultIntervalSeconds,
	)
	if err != nil {
		return nil, fmt.Errorf("scheduled_tasks: claim: %w", err)
	}
	defer rows.Close()
	var out []schedtask.Rule
	for rows.Next() {
		rule, err := scanScheduledTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rule)
	}
	return out, rows.Err()
}

// RecordSuccess persists the outcome of a successful (or "empty") run.
func (r *ScheduledTaskRepo) RecordSuccess(ctx context.Context, id string, status string, cursor string, batchID string, at time.Time) error {
	_, err := r.c.db.ExecResult(ctx, `
		UPDATE scheduled_tasks SET
			last_run_status = $2,
			cursor = COALESCE(NULLIF($3, ''), cursor),
			last_batch_id = COALESCE(NULLIF($4, ''), last_batch_id),
			last_error = '',
			last_success_at = $5,
			updated_at = now()
		WHERE id = $1`,
		id, status, cursor, batchID, at,
	)
	if err != nil {
		return fmt.Errorf("scheduled_tasks: record_success %s: %w", id, err)
	}
	return nil
}

// RecordFailure persists the outcome of a failed run. Does NOT advance the
// cursor — the next tick will retry the same window.
func (r *ScheduledTaskRepo) RecordFailure(ctx context.Context, id string, errMsg string) error {
	_, err := r.c.db.ExecResult(ctx, `
		UPDATE scheduled_tasks SET
			last_run_status = 'failed',
			last_error = $2,
			updated_at = now()
		WHERE id = $1`,
		id, errMsg,
	)
	if err != nil {
		return fmt.Errorf("scheduled_tasks: record_failure %s: %w", id, err)
	}
	return nil
}

// ─── scan helpers ─────────────────────────────────────────────

// rowScanner is already declared in client.go; reused here.

func scanScheduledTask(row rowScanner) (schedtask.Rule, error) {
	var (
		r                 schedtask.Rule
		templateVersion   sql.NullInt64
		scheduling        []byte
		sourceConfig      []byte
		triggerConfig     []byte
		cursor            sql.NullString
		lastRunAt         sql.NullTime
		lastRunStatus     sql.NullString
		lastBatchID       sql.NullString
		lastError         sql.NullString
		lastSuccessAt     sql.NullTime
		runNowRequestedAt sql.NullTime
		createdBy         sql.NullString
		triggerMode       string
	)
	err := row.Scan(
		&r.ID, &r.Name, &r.Enabled, &r.TemplateID, &templateVersion, &r.TargetID, &scheduling,
		&r.SourceType, &sourceConfig, &triggerMode, &triggerConfig, &cursor,
		&lastRunAt, &lastRunStatus, &lastBatchID, &lastError, &lastSuccessAt,
		&runNowRequestedAt, &createdBy, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return schedtask.Rule{}, nil
		}
		return schedtask.Rule{}, err
	}
	if templateVersion.Valid {
		v := int(templateVersion.Int64)
		r.TemplateVersion = &v
	}
	r.Scheduling = json.RawMessage(scheduling)
	r.SourceConfig = json.RawMessage(sourceConfig)
	r.TriggerMode = schedtask.TriggerMode(triggerMode)
	r.TriggerConfig = json.RawMessage(triggerConfig)
	if cursor.Valid {
		r.Cursor = cursor.String
	}
	if lastRunAt.Valid {
		t := lastRunAt.Time
		r.LastRunAt = &t
	}
	if lastRunStatus.Valid {
		r.LastRunStatus = lastRunStatus.String
	}
	if lastBatchID.Valid {
		r.LastBatchID = lastBatchID.String
	}
	if lastError.Valid {
		r.LastError = lastError.String
	}
	if lastSuccessAt.Valid {
		t := lastSuccessAt.Time
		r.LastSuccessAt = &t
	}
	if runNowRequestedAt.Valid {
		t := runNowRequestedAt.Time
		r.RunNowRequestedAt = &t
	}
	if createdBy.Valid {
		r.CreatedBy = createdBy.String
	}
	return r, nil
}

func nullableInt(v *int) any {
	if v == nil {
		return nil
	}
	return *v
}

// nullableJSON persists an empty raw message as SQL NULL to avoid inserting
// "" (which would violate jsonb typing).
func nullableJSON(v json.RawMessage) any {
	if len(v) == 0 {
		return []byte("{}")
	}
	return []byte(v)
}

// prefixed rewrites a comma-separated column list to reference an alias.
// e.g. prefixed("a, b", "t.") = "t.a, t.b".
func prefixed(cols, prefix string) string {
	var out []byte
	inSpace := true
	for i := 0; i < len(cols); i++ {
		ch := cols[i]
		if inSpace && ((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_') {
			out = append(out, prefix...)
			inSpace = false
		}
		if ch == ',' || ch == '\n' || ch == '\t' || ch == ' ' {
			inSpace = true
		} else if ch != '\r' {
			inSpace = false
		}
		out = append(out, ch)
	}
	return string(out)
}
