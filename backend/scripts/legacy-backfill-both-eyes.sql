-- Legacy backfill repro: both eyes (CYB-2041).
-- Idempotent manifest insert for report.project@1.0.0-backfill-both-eyes.
-- Run via: bash scripts/test2-backfill-results-legacy.sh both_eyes

INSERT INTO report_manifests (
  report_id,
  version,
  schema_json,
  created_at,
  updated_at
)
VALUES (
  'report.project@1.0.0-backfill-both-eyes',
  '1.0.0',
  '{"crop":"both_eyes","description":"Legacy repro — both eyes crop"}'::jsonb,
  NOW(),
  NOW()
)
ON CONFLICT (report_id) DO UPDATE SET
  schema_json = EXCLUDED.schema_json,
  updated_at = NOW();
