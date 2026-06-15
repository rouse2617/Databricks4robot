package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

// BackfillResultRepo reads report manifests and upserts algo_run_results staging rows.
type BackfillResultRepo struct {
	c *Client
}

func NewBackfillResultRepo(c *Client) *BackfillResultRepo {
	return &BackfillResultRepo{c: c}
}

var _ repository.BackfillResultRepository = (*BackfillResultRepo)(nil)

func (r *BackfillResultRepo) GetReportManifest(ctx context.Context, reportID string) (*repository.ReportManifest, error) {
	const q = `
SELECT report_id, version, COALESCE(schema_json, '{}'::jsonb)
FROM report_manifests
WHERE report_id = $1`
	db := dbFromCtx(ctx, r.c.db)
	var manifest repository.ReportManifest
	var schemaJSON []byte
	if err := db.QueryRow(ctx, q, reportID).Scan(&manifest.ReportID, &manifest.Version, &schemaJSON); err != nil {
		if errors.Is(err, errNoRows) {
			return nil, repository.ErrReportManifestNotFound
		}
		return nil, fmt.Errorf("postgres BackfillResultRepo.GetReportManifest: %w", err)
	}
	manifest.SchemaJSON = schemaJSON
	return &manifest, nil
}

func (r *BackfillResultRepo) UpsertAlgoRunResult(ctx context.Context, in repository.AlgoRunResultWriteInput) error {
	payload, err := json.Marshal(in.ResultPayload)
	if err != nil {
		return fmt.Errorf("postgres BackfillResultRepo.UpsertAlgoRunResult marshal: %w", err)
	}
	now := time.Now().UTC()
	const q = `
INSERT INTO algo_run_results (
  asset_id, algo_key, version, report_id, result_payload, created_at, updated_at
) VALUES (
  $1, $2, $3, $4, $5::jsonb, $6, $7
)
ON CONFLICT (asset_id, algo_key, version) DO UPDATE SET
  report_id = EXCLUDED.report_id,
  result_payload = EXCLUDED.result_payload,
  updated_at = EXCLUDED.updated_at`
	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, q,
		in.AssetID, in.AlgoKey, in.Version, in.ReportID, payload, now, now,
	); err != nil {
		return fmt.Errorf("postgres BackfillResultRepo.UpsertAlgoRunResult: %w", err)
	}
	return nil
}

func (r *BackfillResultRepo) HasAlgoRunResult(ctx context.Context, assetID, algoKey, version string) (bool, error) {
	const q = `
SELECT 1
FROM algo_run_results
WHERE asset_id = $1 AND algo_key = $2 AND version = $3
LIMIT 1`
	db := dbFromCtx(ctx, r.c.db)
	var one int
	if err := db.QueryRow(ctx, q, assetID, algoKey, version).Scan(&one); err != nil {
		if errors.Is(err, errNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("postgres BackfillResultRepo.HasAlgoRunResult: %w", err)
	}
	return true, nil
}
