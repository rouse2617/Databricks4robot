-- Legacy backfill repro: left eye only (CYB-2041).
-- Idempotent manifest insert for report.project@1.0.0-backfill-left-eye-only.
-- Run via: bash scripts/test2-backfill-results-legacy.sh left_eye

INSERT INTO report_manifests (
  report_id,
  version,
  schema_json,
  created_at,
  updated_at
)
VALUES (
  'report.project@1.0.0-backfill-left-eye-only',
  '1.0.0',
  '{"crop":"left_eye_only","description":"Legacy repro — left eye crop only"}'::jsonb,
  NOW(),
  NOW()
)
ON CONFLICT (report_id) DO UPDATE SET
  schema_json = EXCLUDED.schema_json,
  updated_at = NOW();
