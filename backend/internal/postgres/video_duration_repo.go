package postgres

import (
	"context"
	"fmt"
)

// VideoDurationRepo reads source-video durations (CYB-3059) from the
// video_durations table, which is keyed by video_id (== batch asset_id) and
// populated by an external system. DataBrew only reads it.
type VideoDurationRepo struct {
	c *Client
}

// NewVideoDurationRepo constructs a VideoDurationRepo.
func NewVideoDurationRepo(c *Client) *VideoDurationRepo {
	return &VideoDurationRepo{c: c}
}

// GetByVideoIDs returns a map of video_id -> duration_sec for the given ids.
// Missing ids are simply absent from the map (some videos have no duration).
func (r *VideoDurationRepo) GetByVideoIDs(ctx context.Context, videoIDs []string) (map[string]float64, error) {
	out := make(map[string]float64)
	if len(videoIDs) == 0 {
		return out, nil
	}
	const q = `SELECT video_id, duration_sec FROM video_durations WHERE video_id = ANY($1)`
	db := dbFromCtx(ctx, r.c.db)
	rows, err := db.Query(ctx, q, videoIDs)
	if err != nil {
		return nil, fmt.Errorf("postgres VideoDurationRepo.GetByVideoIDs: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var dur float64
		if err := rows.Scan(&id, &dur); err != nil {
			return nil, fmt.Errorf("postgres VideoDurationRepo.GetByVideoIDs scan: %w", err)
		}
		out[id] = dur
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres VideoDurationRepo.GetByVideoIDs rows: %w", err)
	}
	return out, nil
}

// Upsert writes video_id -> duration_sec rows idempotently (CYB-3072). Called by
// the Grace sync loop; existing rows are updated in place. Batched to keep the
// statement size bounded for large syncs.
func (r *VideoDurationRepo) Upsert(ctx context.Context, durations map[string]float64) error {
	if len(durations) == 0 {
		return nil
	}
	db := dbFromCtx(ctx, r.c.db)
	const q = `
	INSERT INTO video_durations (video_id, duration_sec)
	SELECT unnest($1::text[]), unnest($2::float8[])
	ON CONFLICT (video_id) DO UPDATE
	  SET duration_sec = EXCLUDED.duration_sec, updated_at = now()`
	const chunk = 1000
	ids := make([]string, 0, len(durations))
	for id := range durations {
		ids = append(ids, id)
	}
	for i := 0; i < len(ids); i += chunk {
		end := i + chunk
		if end > len(ids) {
			end = len(ids)
		}
		batchIDs := ids[i:end]
		batchDur := make([]float64, len(batchIDs))
		for j, id := range batchIDs {
			batchDur[j] = durations[id]
		}
		if err := db.Exec(ctx, q, batchIDs, batchDur); err != nil {
			return fmt.Errorf("postgres VideoDurationRepo.Upsert: %w", err)
		}
	}
	return nil
}
