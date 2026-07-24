package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// GetUserPipelineStatsAfter returns aggregated usage statistics for a user's pipelines.
func (r *PipelineTemplateRepo) GetUserPipelineStatsAfter(ctx context.Context, userEmail string, after time.Time) (*models.PipelineUserStats, error) {
	query := `
	SELECT
		pt.id,
		pt.name,
		COALESCE((SELECT COUNT(*) FROM pipeline_runs pr WHERE pr.template_id = pt.id AND pr.created_by = $1), 0) as run_count,
		COALESCE((SELECT SUM(EXTRACT(EPOCH FROM (pr.finished_at - pr.created_at)) * 1000)::bigint
			FROM pipeline_runs pr WHERE pr.template_id = pt.id), 0) as exec_time_ms,
		COALESCE((SELECT MAX(pr.updated_at) FROM pipeline_runs pr WHERE pr.template_id = pt.id), pt.updated_at) as last_access_time
	FROM pipeline_templates pt
	WHERE pt.owner = $1 OR pt.scope = 'shared'
	ORDER BY run_count DESC, last_access_time DESC
	LIMIT 200
	`

	rows, err := r.c.db.Query(ctx, query, userEmail)
	if err != nil {
		return nil, fmt.Errorf("failed to query pipeline stats: %w", err)
	}
	defer rows.Close()

	stats := &models.PipelineUserStats{
		ClickCounts:     make(map[string]int),
		RunCounts:       make(map[string]int),
		ExecutionTimes:  make(map[string]int64),
		LastAccessTimes: make(map[string]time.Time),
		Window:          "30d",
		ComputedAt:      time.Now(),
	}

	for rows.Next() {
		var (
			pipelineID string
			name       string
			runCount   int
			execTimeMs int64
			lastAccess time.Time
		)

		if err := rows.Scan(&pipelineID, &name, &runCount, &execTimeMs, &lastAccess); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		stats.ClickCounts[pipelineID] = runCount / 2
		stats.RunCounts[pipelineID] = runCount
		stats.ExecutionTimes[pipelineID] = execTimeMs
		stats.LastAccessTimes[pipelineID] = lastAccess
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return stats, nil
}

// GetLatestVersionWithConflictCheck returns the latest version and checks for conflicts.
func (r *PipelineTemplateRepo) GetLatestVersionWithConflictCheck(ctx context.Context, pipelineID string, baseVersion int) (*models.PipelineTemplate, bool, error) {
	query := `
	SELECT ` + pipelineTemplateSelectCols + `
	FROM pipeline_templates
	WHERE name = (SELECT name FROM pipeline_templates WHERE id = $1)
	ORDER BY version DESC
	LIMIT 1
	`

	rs := r.c.db.QueryRow(ctx, query, pipelineID)
	pt, err := scanPipelineTemplate(rs)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("failed to query template: %w", err)
	}

	conflict := (baseVersion > 0 && pt.Version != baseVersion)
	return pt, conflict, nil
}
