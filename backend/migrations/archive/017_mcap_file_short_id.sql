-- 017_mcap_file_short_id.sql
-- mcap_file_id: UUID -> 8-char [0-9A-Za-z] (same format as asset_id).
-- Idempotent: no-op if mcap_files.mcap_file_id is already non-uuid.

DO $body$
DECLARE
  col_udt text;
  u         uuid;
  sid       text;
  alphabet  text := '0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz';
  i         int;
  attempts  int;
  chars_len int := length(alphabet);
BEGIN
  SELECT c.udt_name INTO col_udt
  FROM information_schema.columns c
  WHERE c.table_schema = 'public'
    AND c.table_name = 'mcap_files'
    AND c.column_name = 'mcap_file_id';

  IF col_udt IS NULL THEN
    RAISE NOTICE '017: mcap_files missing, skip';
    RETURN;
  END IF;

  IF col_udt <> 'uuid' THEN
    RAISE NOTICE '017: mcap_file_id already % (not uuid), skip', col_udt;
    RETURN;
  END IF;

  ALTER TABLE asset_events DROP CONSTRAINT IF EXISTS asset_events_mcap_file_id_fkey;
  ALTER TABLE asset_eval_results DROP CONSTRAINT IF EXISTS asset_eval_results_mcap_file_id_fkey;

  CREATE TEMP TABLE _mcap_uuid_map (
    old_uuid uuid PRIMARY KEY,
    new_id   text NOT NULL UNIQUE
      CHECK (new_id ~ '^[0-9A-Za-z]{8}$')
  ) ON COMMIT DROP;

  FOR u IN
    SELECT x FROM (
      SELECT mcap_file_id AS x FROM mcap_files
      UNION
      SELECT mcap_file_id FROM assets
      UNION
      SELECT mcap_file_id FROM asset_events WHERE mcap_file_id IS NOT NULL
      UNION
      SELECT mcap_file_id FROM asset_eval_results WHERE mcap_file_id IS NOT NULL
      UNION
      SELECT mcap_file_id FROM outbox_dlq WHERE mcap_file_id IS NOT NULL
    ) s
  LOOP
    attempts := 0;
    <<genloop>>
    LOOP
      sid := '';
      FOR i IN 1..8 LOOP
        sid := sid || substr(alphabet, 1 + floor(random() * chars_len)::int, 1);
      END LOOP;
      BEGIN
        INSERT INTO _mcap_uuid_map (old_uuid, new_id) VALUES (u, sid);
        EXIT genloop;
      EXCEPTION
        WHEN unique_violation THEN
          attempts := attempts + 1;
          IF attempts > 200 THEN
            RAISE EXCEPTION '017: could not allocate unique mcap id for %', u;
          END IF;
      END;
    END LOOP;
  END LOOP;

  ALTER TABLE assets
    ALTER COLUMN mcap_file_id TYPE text
    USING ( (SELECT m.new_id FROM _mcap_uuid_map m WHERE m.old_uuid = assets.mcap_file_id) );

  ALTER TABLE assets
    ADD CONSTRAINT assets_mcap_file_id_chk CHECK (mcap_file_id ~ '^[0-9A-Za-z]{8}$');

  ALTER TABLE outbox_dlq
    ALTER COLUMN mcap_file_id TYPE text
    USING (
      CASE
        WHEN outbox_dlq.mcap_file_id IS NULL THEN NULL
        ELSE (SELECT m.new_id FROM _mcap_uuid_map m WHERE m.old_uuid = outbox_dlq.mcap_file_id)
      END
    );

  ALTER TABLE asset_events
    ALTER COLUMN mcap_file_id TYPE text
    USING (
      CASE
        WHEN asset_events.mcap_file_id IS NULL THEN NULL
        ELSE (SELECT m.new_id FROM _mcap_uuid_map m WHERE m.old_uuid = asset_events.mcap_file_id)
      END
    );

  ALTER TABLE asset_eval_results
    ALTER COLUMN mcap_file_id TYPE text
    USING (
      CASE
        WHEN asset_eval_results.mcap_file_id IS NULL THEN NULL
        ELSE (SELECT m.new_id FROM _mcap_uuid_map m WHERE m.old_uuid = asset_eval_results.mcap_file_id)
      END
    );

  ALTER TABLE mcap_files
    ALTER COLUMN mcap_file_id TYPE text
    USING ( (SELECT m.new_id FROM _mcap_uuid_map m WHERE m.old_uuid = mcap_files.mcap_file_id) );

  ALTER TABLE mcap_files
    ADD CONSTRAINT mcap_files_mcap_file_id_chk CHECK (mcap_file_id ~ '^[0-9A-Za-z]{8}$');

  ALTER TABLE assets
    ADD CONSTRAINT fk_assets_mcap_file
    FOREIGN KEY (mcap_file_id) REFERENCES mcap_files (mcap_file_id);

  ALTER TABLE asset_events
    ADD CONSTRAINT asset_events_mcap_file_id_fkey
    FOREIGN KEY (mcap_file_id) REFERENCES mcap_files (mcap_file_id);

  ALTER TABLE asset_eval_results
    ADD CONSTRAINT asset_eval_results_mcap_file_id_fkey
    FOREIGN KEY (mcap_file_id) REFERENCES mcap_files (mcap_file_id);

  RAISE NOTICE '017: mcap_file_id migrated to 8-char text';
END
$body$;
