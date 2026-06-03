-- 050: Add active_version to pipeline_templates.
-- Allows pinning a default run version that differs from the latest.

ALTER TABLE pipeline_templates
  ADD COLUMN IF NOT EXISTS active_version INT NOT NULL DEFAULT 0;
