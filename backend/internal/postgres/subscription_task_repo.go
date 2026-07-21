package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/subtask"
)

type SubscriptionTaskRepo struct {
	c *Client
}

func NewSubscriptionTaskRepo(c *Client) *SubscriptionTaskRepo {
	return &SubscriptionTaskRepo{c: c}
}

const subTaskCols = `id, name, enabled, template_id, template_version, target_id, scheduling,
	project_id, subscription_id, pull_interval_seconds, max_messages_per_pull,
	last_run_at, last_run_status, last_batch_id, last_error, last_success_at,
	created_by, created_at, updated_at`

func (r *SubscriptionTaskRepo) List(ctx context.Context) ([]subtask.Task, error) {
	rows, err := r.c.db.Query(ctx, `SELECT `+subTaskCols+` FROM subscription_tasks ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("subscription_tasks: list: %w", err)
	}
	defer rows.Close()
	var out []subtask.Task
	for rows.Next() {
		t, err := scanSubTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *SubscriptionTaskRepo) Get(ctx context.Context, id string) (*subtask.Task, error) {
	row := r.c.db.QueryRow(ctx, `SELECT `+subTaskCols+` FROM subscription_tasks WHERE id = $1`, id)
	t, err := scanSubTask(row)
	if err != nil {
		return nil, fmt.Errorf("subscription_tasks: get %s: %w", id, err)
	}
	if t.ID == "" {
		return nil, nil
	}
	return &t, nil
}

func (r *SubscriptionTaskRepo) Create(ctx context.Context, t *subtask.Task) error {
	if t.ID == "" || t.Name == "" {
		return fmt.Errorf("subscription_tasks: id and name are required")
	}
	_, err := r.c.db.ExecResult(ctx, `
		INSERT INTO subscription_tasks (
			id, name, enabled, template_id, template_version, target_id, scheduling,
			project_id, subscription_id, pull_interval_seconds, max_messages_per_pull,
			created_by, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12, now(), now())`,
		t.ID, t.Name, t.Enabled, t.TemplateID, nullableInt(t.TemplateVersion), t.TargetID, []byte("{}"),
		t.ProjectID, t.SubscriptionID, t.PullIntervalSec, t.MaxMessagesPerPull,
		t.CreatedBy,
	)
	if err != nil {
		return fmt.Errorf("subscription_tasks: create %s: %w", t.ID, err)
	}
	return nil
}

func (r *SubscriptionTaskRepo) Update(ctx context.Context, t *subtask.Task) error {
	if t.ID == "" {
		return fmt.Errorf("subscription_tasks: id is required")
	}
	_, err := r.c.db.ExecResult(ctx, `
		UPDATE subscription_tasks SET
			name = $2, enabled = $3, template_id = $4, template_version = $5,
			target_id = $6, scheduling = $7,
			project_id = $8, subscription_id = $9,
			pull_interval_seconds = $10, max_messages_per_pull = $11,
			updated_at = now()
		WHERE id = $1`,
		t.ID, t.Name, t.Enabled, t.TemplateID, nullableInt(t.TemplateVersion), t.TargetID, []byte("{}"),
		t.ProjectID, t.SubscriptionID, t.PullIntervalSec, t.MaxMessagesPerPull,
	)
	if err != nil {
		return fmt.Errorf("subscription_tasks: update %s: %w", t.ID, err)
	}
	return nil
}

func (r *SubscriptionTaskRepo) Delete(ctx context.Context, id string) error {
	_, err := r.c.db.ExecResult(ctx, `DELETE FROM subscription_tasks WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("subscription_tasks: delete %s: %w", id, err)
	}
	return nil
}

func (r *SubscriptionTaskRepo) SetEnabled(ctx context.Context, id string, enabled bool) error {
	_, err := r.c.db.ExecResult(ctx, `UPDATE subscription_tasks SET enabled = $2, updated_at = now() WHERE id = $1`, id, enabled)
	if err != nil {
		return fmt.Errorf("subscription_tasks: set_enabled %s: %w", id, err)
	}
	return nil
}

func (r *SubscriptionTaskRepo) ClaimDueTasks(ctx context.Context, now time.Time) ([]subtask.Task, error) {
	rows, err := r.c.db.Query(ctx, `
		WITH due AS (
			SELECT id
			FROM subscription_tasks
			WHERE enabled = true
			  AND (
			    last_run_at IS NULL
			    OR EXTRACT(EPOCH FROM ($1::timestamptz - last_run_at)) >= pull_interval_seconds
			  )
			ORDER BY last_run_at NULLS FIRST
			LIMIT 64
			FOR UPDATE SKIP LOCKED
		)
		UPDATE subscription_tasks t
		SET last_run_at = $1,
		    updated_at = now()
		FROM due
		WHERE t.id = due.id
		RETURNING `+prefixed(subTaskCols, "t.")+``,
		now,
	)
	if err != nil {
		return nil, fmt.Errorf("subscription_tasks: claim: %w", err)
	}
	defer rows.Close()
	var out []subtask.Task
	for rows.Next() {
		t, err := scanSubTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *SubscriptionTaskRepo) RecordSuccess(ctx context.Context, id string, status string, batchID string, at time.Time) error {
	_, err := r.c.db.ExecResult(ctx, `
		UPDATE subscription_tasks SET
			last_run_status = $2,
			last_batch_id = COALESCE(NULLIF($3, ''), last_batch_id),
			last_error = '',
			last_success_at = $4,
			updated_at = now()
		WHERE id = $1`,
		id, status, batchID, at,
	)
	if err != nil {
		return fmt.Errorf("subscription_tasks: record_success %s: %w", id, err)
	}
	return nil
}

func (r *SubscriptionTaskRepo) RecordFailure(ctx context.Context, id string, errMsg string) error {
	_, err := r.c.db.ExecResult(ctx, `
		UPDATE subscription_tasks SET
			last_run_status = 'failed',
			last_error = $2,
			updated_at = now()
		WHERE id = $1`,
		id, errMsg,
	)
	if err != nil {
		return fmt.Errorf("subscription_tasks: record_failure %s: %w", id, err)
	}
	return nil
}

func scanSubTask(row rowScanner) (subtask.Task, error) {
	var (
		t               subtask.Task
		templateVersion sql.NullInt64
		scheduling      []byte
		lastRunAt       sql.NullTime
		lastRunStatus   sql.NullString
		lastBatchID     sql.NullString
		lastError       sql.NullString
		lastSuccessAt   sql.NullTime
		createdBy       sql.NullString
	)
	err := row.Scan(
		&t.ID, &t.Name, &t.Enabled, &t.TemplateID, &templateVersion, &t.TargetID, &scheduling,
		&t.ProjectID, &t.SubscriptionID, &t.PullIntervalSec, &t.MaxMessagesPerPull,
		&lastRunAt, &lastRunStatus, &lastBatchID, &lastError, &lastSuccessAt,
		&createdBy, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return subtask.Task{}, nil
		}
		return subtask.Task{}, err
	}
	if templateVersion.Valid {
		v := int(templateVersion.Int64)
		t.TemplateVersion = &v
	}
	if lastRunAt.Valid {
		tt := lastRunAt.Time
		t.LastRunAt = &tt
	}
	if lastRunStatus.Valid {
		t.LastRunStatus = lastRunStatus.String
	}
	if lastBatchID.Valid {
		t.LastBatchID = lastBatchID.String
	}
	if lastError.Valid {
		t.LastError = lastError.String
	}
	if lastSuccessAt.Valid {
		tt := lastSuccessAt.Time
		t.LastSuccessAt = &tt
	}
	if createdBy.Valid {
		t.CreatedBy = createdBy.String
	}
	return t, nil
}

func nullableInt(v *int) any {
	if v == nil {
		return nil
	}
	return *v
}

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
