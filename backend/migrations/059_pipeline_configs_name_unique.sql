-- 059: Add UNIQUE constraint on pipeline_configs.name to prevent duplicates.

ALTER TABLE pipeline_configs ADD CONSTRAINT uq_pipeline_configs_name UNIQUE (name);
