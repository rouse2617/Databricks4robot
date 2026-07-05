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
