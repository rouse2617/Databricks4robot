-- 021_search_reindex_jobs.sql
-- Async reindex job state for stop/resume/progress tracking.

CREATE TABLE IF NOT EXISTS search_reindex_jobs (
  id                      TEXT PRIMARY KEY,
  status                  TEXT NOT NULL CHECK (status IN ('queued', 'running', 'paused', 'succeeded', 'failed')),
  dry_run                 BOOLEAN NOT NULL DEFAULT FALSE,
  page_size               INT NOT NULL DEFAULT 200,
  next_page               INT NOT NULL DEFAULT 1,
  stop_requested          BOOLEAN NOT NULL DEFAULT FALSE,
  total_assets            BIGINT NOT NULL DEFAULT 0,
  assets_scanned          BIGINT NOT NULL DEFAULT 0,
  documents_indexed       BIGINT NOT NULL DEFAULT 0,
  documents_deleted       BIGINT NOT NULL DEFAULT 0,
  failed                  BIGINT NOT NULL DEFAULT 0,
  error                   TEXT NOT NULL DEFAULT '',
  error_samples           JSONB NOT NULL DEFAULT '[]'::jsonb,
  elasticsearch_doc_count BIGINT,
  index_cleared           BOOLEAN NOT NULL DEFAULT FALSE,
  created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
  started_at              TIMESTAMPTZ,
  finished_at             TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_search_reindex_jobs_status_updated
  ON search_reindex_jobs (status, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_search_reindex_jobs_created
  ON search_reindex_jobs (created_at DESC);
