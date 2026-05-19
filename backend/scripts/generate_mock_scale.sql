-- Generate scale-test mock data for local Postgres.
--
-- Usage:
--   psql postgresql://postgres:postgres@localhost:5432/cyber_databrew_dev \
--     -v row_count=100000 \
--     -v batch_id=scale_100k \
--     -f backend/scripts/generate_mock_scale.sql
--
-- The script is repeatable per batch_id. It deletes rows previously generated
-- for the same batch before inserting fresh mock data.

\set ON_ERROR_STOP on

BEGIN;

-- Clean up an existing batch so local experiments are repeatable.
DELETE FROM delivery_items di
USING deliveries d
WHERE di.delivery_id = d.delivery_id
  AND d.cf_meta ->> 'mock_batch' = :'batch_id';

DELETE FROM delivery_items di
USING assets a
WHERE di.asset_id = a.asset_id
  AND a.cf_meta ->> 'mock_batch' = :'batch_id';

DELETE FROM asset_algo_events e
USING assets a
WHERE e.asset_id = a.asset_id
  AND a.cf_meta ->> 'mock_batch' = :'batch_id';

DELETE FROM deliveries
WHERE cf_meta ->> 'mock_batch' = :'batch_id';

DELETE FROM assets
WHERE cf_meta ->> 'mock_batch' = :'batch_id';

DELETE FROM mcap_files
WHERE cf_meta ->> 'mock_batch' = :'batch_id';

CREATE TEMP TABLE tmp_mock_mcap (
  rn integer PRIMARY KEY,
  mcap_file_id uuid NOT NULL,
  gcs_path text NOT NULL,
  base_ts bigint NOT NULL
) ON COMMIT DROP;

INSERT INTO tmp_mock_mcap (rn, mcap_file_id, gcs_path, base_ts)
SELECT
  i,
  gen_random_uuid(),
  'gs://grace-raw/mcap/scale/' || :'batch_id' || '/recording-' || lpad(i::text, 6, '0') || '.mcap',
  1700000000000000000 + i::bigint * 100000000000
FROM generate_series(1, GREATEST(100, LEAST(5000, CEIL(:'row_count'::numeric / 200)::int))) AS i;

INSERT INTO mcap_files (mcap_file_id, raw_hash_md5, cf_meta, cf_process, created_at, updated_at, version)
SELECT
  mcap_file_id,
  md5(mcap_file_id::text),
  jsonb_build_object(
    'mock_batch', :'batch_id',
    'gcs_path', gcs_path,
    'size_bytes', (50 + (rn % 200))::bigint * 1048576,
    'ingest_state', CASE WHEN rn % 10 = 0 THEN 'pending' ELSE 'summarized' END,
    'start_timestamp_ns', base_ts,
    'end_timestamp_ns', base_ts + 60000000000,
    'channel_count', 4 + (rn % 16),
    'chunk_count', 10 + (rn % 90),
    'owner', (ARRAY['urn:grace:team:bay-area','urn:grace:team:shanghai','urn:grace:team:tokyo','urn:grace:team:berlin'])[1 + (rn % 4)]
  ),
  jsonb_build_object(
    'hand_tracking', CASE WHEN rn % 7 = 0 THEN 'failed' WHEN rn % 3 = 0 THEN 'pending' ELSE 'completed' END,
    'head_tracking', CASE WHEN rn % 5 = 0 THEN 'pending' ELSE 'completed' END
  ),
  now() - ((rn % 60) * interval '1 day'),
  now() - ((rn % 30) * interval '1 day'),
  1
FROM tmp_mock_mcap;

CREATE TEMP TABLE tmp_mock_assets (
  seq bigint PRIMARY KEY,
  asset_id uuid NOT NULL,
  mcap_file_id uuid NOT NULL,
  start_ns bigint NOT NULL,
  end_ns bigint NOT NULL,
  status text NOT NULL,
  env text NOT NULL,
  task text NOT NULL,
  quality text NOT NULL,
  algo_status text NOT NULL,
  created_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL
) ON COMMIT DROP;

INSERT INTO tmp_mock_assets (
  seq, asset_id, mcap_file_id, start_ns, end_ns, status, env, task, quality, algo_status, created_at, updated_at
)
SELECT
  i::bigint,
  gen_random_uuid(),
  m.mcap_file_id,
  m.base_ts + ((i % 2000)::bigint * 5000000000),
  m.base_ts + ((i % 2000)::bigint * 5000000000) + ((1 + (i % 30))::bigint * 1000000000),
  (ARRAY['approved','approved','approved','rejected','superseded','archived','pending','disputed'])[1 + (i % 8)],
  (ARRAY['kitchen','warehouse','office','outdoor','lab','factory'])[1 + (i % 6)],
  (ARRAY['cook_pasta','pick_item','calibrate','wash_dishes','sort_packages','navigate_room','assemble_part'])[1 + (i % 7)],
  (ARRAY['excellent','good','acceptable','poor','unusable'])[1 + (i % 5)],
  (ARRAY['ok','ok','failed','running','pending'])[1 + (i % 5)],
  now() - ((i % 60) * interval '1 day'),
  now() - ((i % 30) * interval '1 day')
FROM generate_series(1, :'row_count'::bigint) AS i
JOIN tmp_mock_mcap m
  ON m.rn = 1 + ((i - 1) % (SELECT count(*) FROM tmp_mock_mcap));

INSERT INTO assets (
  asset_id, mcap_file_id, start_timestamp_ns, end_timestamp_ns,
  status, segment_locator, cf_meta, cf_algo, cf_tag, cf_files,
  created_at, updated_at, version
)
SELECT
  asset_id,
  mcap_file_id,
  start_ns,
  end_ns,
  status,
  encode(digest(mcap_file_id::text || start_ns::text || end_ns::text, 'sha1'), 'hex'),
  jsonb_build_object(
    'mock_batch', :'batch_id',
    'duration_sec', ((end_ns - start_ns) / 1000000000.0),
    'reviewer', (ARRAY['alice','bob','charlie','diana','eve','frank'])[1 + (seq % 6)],
    'owner', (ARRAY[
      'urn:grace:user:alice','urn:grace:user:bob','urn:grace:user:charlie',
      'urn:grace:user:diana','urn:grace:user:eve','urn:grace:user:frank'
    ])[1 + (seq % 6)],
    'type', (ARRAY['task_demo','calibration','free_motion','manipulation','navigation'])[1 + (seq % 5)],
    'env', env,
    'task', task,
    'delivery_count', (seq % 5),
    'last_delivered_to', CASE WHEN seq % 3 = 0 THEN 'urn:grace:customer:' || chr(65 + (seq % 5)::int) ELSE '' END,
    'retention_tier', (ARRAY['standard','archive','critical'])[1 + (seq % 3)],
    'archive_after_days', (ARRAY[30,60,90,180])[1 + (seq % 4)],
    'delete_after_days', (ARRAY[180,365,730])[1 + (seq % 3)],
    'total_size_bytes', (seq % 50000000)
  ),
  CASE seq % 4
    WHEN 0 THEN jsonb_build_object(
      'hand_tracking@1.2.0:status', algo_status,
      'hand_tracking@1.2.0:method', 'ray_batch',
      'hand_tracking@1.2.0:run_id', 'scale-run-' || :'batch_id'
    )
    WHEN 1 THEN jsonb_build_object(
      'deface@2.0.0:status', algo_status,
      'deface@2.0.0:method', 'local',
      'deface@2.0.0:output_uri', 'gs://grace-algo/deface/' || :'batch_id' || '/seg-' || seq || '.mcap'
    )
    WHEN 2 THEN jsonb_build_object(
      'sam2@1.0.0:status', algo_status,
      'tracker@3.0.0:status', (ARRAY['ok','running','pending'])[1 + (seq % 3)]
    )
    ELSE jsonb_build_object(
      'hand_tracking@1.2.0:status', algo_status,
      'deface@2.0.0:status', (ARRAY['ok','ok','running'])[1 + (seq % 3)],
      'sam2@1.0.0:status', (ARRAY['ok','failed','pending','blocked'])[1 + (seq % 4)]
    )
  END,
  jsonb_build_object(
    'priority', (ARRAY['critical','high','medium','low'])[1 + (seq % 4)],
    'quality', quality,
    'scene', (ARRAY['indoor','outdoor','warehouse','office','kitchen','lab'])[1 + (seq % 6)]
  ) || CASE WHEN seq % 10 = 0 THEN jsonb_build_object('note', 'scale-test') ELSE '{}'::jsonb END,
  jsonb_build_object(
    'raw_mcap', 'gs://grace-raw/mcap/scale/' || :'batch_id' || '/recording-' || lpad(((seq % (SELECT count(*) FROM tmp_mock_mcap)) + 1)::text, 6, '0') || '.mcap'
  ) || CASE WHEN seq % 2 = 0 THEN jsonb_build_object(
    'thumbnail', 'gs://grace-derived/thumbnails/' || :'batch_id' || '/seg-' || seq || '.jpg',
    'preview_manifest', 'gs://grace-derived/previews/' || :'batch_id' || '/seg-' || seq || '/manifest.json'
  ) ELSE '{}'::jsonb END,
  created_at,
  updated_at,
  1 + (seq % 5)
FROM tmp_mock_assets;

INSERT INTO deliveries (delivery_id, customer_id, status, delivered_at, cf_meta, created_at, updated_at, version)
SELECT
  gen_random_uuid(),
  'urn:grace:customer:' || chr(65 + (i % 5)),
  (ARRAY['pending','delivered','accepted','rejected'])[1 + (i % 4)],
  CASE WHEN i % 3 = 0 THEN NULL ELSE now() - ((i % 20) * interval '1 day') END,
  jsonb_build_object(
    'mock_batch', :'batch_id',
    'manifest_uri', 'gs://deliveries/' || :'batch_id' || '/batch-' || i || '/manifest.json',
    'contract_id', 'CT-' || :'batch_id' || '-' || lpad(i::text, 6, '0'),
    'note', 'Scale test delivery batch ' || i,
    'asset_count', 20,
    'owner', (ARRAY['urn:grace:user:alice','urn:grace:user:bob','urn:grace:user:charlie'])[1 + (i % 3)]
  ),
  now() - ((i % 30) * interval '1 day'),
  now() - ((i % 15) * interval '1 day'),
  1
FROM generate_series(1, GREATEST(20, LEAST(1000, CEIL(:'row_count'::numeric / 1000)::int))) AS i;

INSERT INTO delivery_items (delivery_id, asset_id)
SELECT d.delivery_id, a.asset_id
FROM (
  SELECT delivery_id, row_number() OVER (ORDER BY delivery_id) AS rn
  FROM deliveries
  WHERE cf_meta ->> 'mock_batch' = :'batch_id'
) d
JOIN tmp_mock_assets a ON a.seq BETWEEN ((d.rn - 1) * 20 + 1) AND (d.rn * 20)
ON CONFLICT DO NOTHING;

-- item_count must match junction rows (API reads item_count → asset_count).
UPDATE deliveries d
SET
  item_count = s.cnt,
  contract_id = COALESCE(NULLIF(TRIM(contract_id), ''), d.cf_meta->>'contract_id'),
  manifest_uri = COALESCE(NULLIF(TRIM(manifest_uri), ''), d.cf_meta->>'manifest_uri'),
  delivered_by = COALESCE(NULLIF(TRIM(delivered_by), ''), d.cf_meta->>'owner')
FROM (
  SELECT delivery_id, COUNT(*)::bigint AS cnt
  FROM delivery_items
  GROUP BY delivery_id
) s
WHERE d.delivery_id = s.delivery_id
  AND d.cf_meta ->> 'mock_batch' = :'batch_id';

UPDATE deliveries d
SET item_count = 0
WHERE d.cf_meta ->> 'mock_batch' = :'batch_id'
  AND NOT EXISTS (SELECT 1 FROM delivery_items di WHERE di.delivery_id = d.delivery_id);

INSERT INTO asset_algo_events (event_id, asset_id, algo_key, prev_status, new_status, run_id, reason, created_at)
SELECT
  gen_random_uuid(),
  asset_id,
  (ARRAY['hand_tracking@1.2.0','deface@2.0.0','sam2@1.0.0','tracker@3.0.0'])[1 + (seq % 4)],
  (ARRAY[NULL,'pending','running','pending'])[1 + (seq % 4)],
  (ARRAY['pending','running','ok','failed'])[1 + (seq % 4)],
  'scale-run-' || :'batch_id',
  CASE WHEN seq % 11 = 0 THEN 'timeout' WHEN seq % 13 = 0 THEN 'oom' ELSE NULL END,
  updated_at
FROM tmp_mock_assets
WHERE seq % 5 = 0;

COMMIT;

SELECT 'assets' AS tbl, count(*) FROM assets
UNION ALL SELECT 'mcap_files', count(*) FROM mcap_files
UNION ALL SELECT 'deliveries', count(*) FROM deliveries
UNION ALL SELECT 'delivery_items', count(*) FROM delivery_items
UNION ALL SELECT 'algo_events', count(*) FROM asset_algo_events
UNION ALL SELECT 'mock_assets_' || :'batch_id', count(*) FROM assets WHERE cf_meta ->> 'mock_batch' = :'batch_id';
