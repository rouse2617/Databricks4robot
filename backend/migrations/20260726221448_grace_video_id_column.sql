-- CYB-4011: reserve an explicit column for the Grace video id on mcap assets.
--
-- DataBrew mcap files (8-char id) and Grace videos (UUID) currently have no
-- explicit link column; the only reliable join key is raw_hash_md5 (content
-- hash == GCS filename). This adds a nullable text column `grace_video_id`
-- to mcap_files (source of truth) and mirrors it onto assets (following the
-- CYB-3715 flatten pattern in 20260722100000) so /queries/run and the mcap
-- detail page can read/filter/display it.
--
-- Scope: reserve the interface only. No backfill — existing rows keep
-- grace_video_id NULL until a later change populates it (by md5 backfill /
-- Grace sync / ingest-time write). Stored as text (not uuid) so a legacy /
-- malformed value never aborts a write; the value is semantically a UUID.

ALTER TABLE mcap_files
  ADD COLUMN grace_video_id text;

ALTER TABLE assets
  ADD COLUMN grace_video_id text;

-- Partial index for exact filter lookups on populated rows only (mirrors the
-- CYB-3715 flatten indexes). High-cardinality, filter-only — never faceted.
CREATE INDEX idx_assets_grace_video_id ON assets (grace_video_id)
  WHERE is_deleted = FALSE AND grace_video_id IS NOT NULL;
