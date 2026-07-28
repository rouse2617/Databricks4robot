-- CYB-3715 Phase 1 of 3: flatten mcap-file top-level fields onto assets so
-- /queries/run can filter/facet by camera_model / device_id / scene_id /
-- data_source / collection_method / source_platform. Producer identity
-- fields become first-class asset columns instead of living only inside
-- mcap_files (invisible to the query planner).
--
-- Note: 7 columns nullable. Uuid fields coerced from mcap_files.<field>
-- via NULLIF(x,'')::uuid — the mcap-files text columns are user-supplied
-- and may be empty/malformed for legacy rows. The migration falls back
-- to NULL on bad uuids by using a per-row exception via a DO block
-- (see Phase 2 backfill below).

ALTER TABLE assets
  ADD COLUMN camera_model      text,
  ADD COLUMN device_id         uuid,
  ADD COLUMN collector_id      uuid,
  ADD COLUMN scene_id          uuid,
  ADD COLUMN data_source       text,
  ADD COLUMN collection_method text,
  ADD COLUMN source_platform   text;

-- Partial indexes on the fields we expect to filter/facet frequently. Skip
-- data_source / collection_method (low cardinality, small enough to seq
-- scan) and collector_id (per-uuid, filter-only, use rows are rare).
CREATE INDEX idx_assets_camera_model    ON assets (camera_model)    WHERE is_deleted = FALSE AND camera_model    IS NOT NULL;
CREATE INDEX idx_assets_device_id       ON assets (device_id)       WHERE is_deleted = FALSE AND device_id       IS NOT NULL;
CREATE INDEX idx_assets_scene_id        ON assets (scene_id)        WHERE is_deleted = FALSE AND scene_id        IS NOT NULL;
CREATE INDEX idx_assets_source_platform ON assets (source_platform) WHERE is_deleted = FALSE AND source_platform IS NOT NULL;

-- Phase 2 backfill: pull the values from mcap_files onto the raw_mcap asset
-- rows. Every raw_mcap asset has mcap_file_id == asset_id (CYB-1217). Uuid
-- coercion is per-row so a single bad string (e.g. legacy free-form text
-- in device_id) leaves that ONE column NULL rather than aborting the
-- migration.
DO $$
DECLARE
  r RECORD;
  v_device_id       uuid;
  v_collector_id    uuid;
  v_scene_id        uuid;
BEGIN
  FOR r IN
    SELECT a.asset_id,
           m.camera_model,
           m.device_id       AS device_text,
           m.collector_id    AS collector_text,
           m.scene_id        AS scene_text,
           m.data_source,
           m.collection_method,
           m.metadata ->> 'source_platform' AS source_platform
      FROM assets a
      JOIN mcap_files m ON m.mcap_file_id = a.mcap_file_id
     WHERE a.is_deleted = FALSE
       AND m.is_deleted = FALSE
  LOOP
    BEGIN
      v_device_id := NULLIF(r.device_text, '')::uuid;
    EXCEPTION WHEN invalid_text_representation THEN v_device_id := NULL;
    END;
    BEGIN
      v_collector_id := NULLIF(r.collector_text, '')::uuid;
    EXCEPTION WHEN invalid_text_representation THEN v_collector_id := NULL;
    END;
    BEGIN
      v_scene_id := NULLIF(r.scene_text, '')::uuid;
    EXCEPTION WHEN invalid_text_representation THEN v_scene_id := NULL;
    END;

    UPDATE assets
       SET camera_model      = NULLIF(r.camera_model, ''),
           device_id         = v_device_id,
           collector_id      = v_collector_id,
           scene_id          = v_scene_id,
           data_source       = NULLIF(r.data_source, ''),
           collection_method = NULLIF(r.collection_method, ''),
           source_platform   = NULLIF(r.source_platform, '')
     WHERE asset_id = r.asset_id;
  END LOOP;
END $$;
