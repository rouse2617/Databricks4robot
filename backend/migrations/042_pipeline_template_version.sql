-- Add versioning support to pipeline_templates.
-- Each save creates a new row with an auto-incremented version per template name.

ALTER TABLE pipeline_templates ADD COLUMN IF NOT EXISTS version INT NOT NULL DEFAULT 1;

-- Backfill: assign sequential versions to existing rows grouped by name.
WITH numbered AS (
  SELECT id, name,
         ROW_NUMBER() OVER (PARTITION BY name ORDER BY created_at ASC) AS seq
  FROM pipeline_templates
)
UPDATE pipeline_templates t
SET version = n.seq
FROM numbered n
WHERE t.id = n.id;

-- Guarantee no duplicate versions for the same name.
CREATE UNIQUE INDEX IF NOT EXISTS idx_pipeline_templates_name_version ON pipeline_templates(name, version);
