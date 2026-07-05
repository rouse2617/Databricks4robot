-- CYB-3059: source-video duration lookup for the batch subtask runs list.
-- Batch asset_ids are Grace video_ids (not DataBrew assets), so duration has no
-- assets row to live on. This table is keyed by video_id and is populated by an
-- external system; DataBrew only reads it (LEFT JOIN on asset_id) to show a
-- "video duration" column in the "子任务执行记录" list.
CREATE TABLE IF NOT EXISTS video_durations (
  video_id     TEXT PRIMARY KEY,
  duration_sec DOUBLE PRECISION NOT NULL,
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
