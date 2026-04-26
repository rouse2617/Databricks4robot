-- 003_mock_10k.sql — Generate 10,000 mock assets for testing.
-- Run manually: docker exec local-postgres-1 psql -U postgres -d data4cyber -f /docker-entrypoint-initdb.d/003_mock_10k.sql

-- Step 1: Insert 50 additional mcap_files
INSERT INTO mcap_files (mcap_file_id, raw_hash_md5, cf_meta, cf_process, created_at, updated_at, version)
SELECT
  gen_random_uuid(),
  md5(random()::text),
  jsonb_build_object(
    'gcs_path', 'gs://grace-raw/mcap/2026-04/recording-' || lpad(i::text, 4, '0') || '.mcap',
    'size_bytes', (50 + floor(random() * 200))::int * 1048576,
    'ingest_state', (ARRAY['pending','summarized','summarized','summarized'])[1 + floor(random()*4)::int],
    'start_timestamp_ns', 1700000000000000000 + i * 100000000000,
    'end_timestamp_ns',   1700000000000000000 + i * 100000000000 + 60000000000,
    'channel_count', 4 + floor(random()*16)::int,
    'chunk_count', 10 + floor(random()*90)::int,
    'owner', (ARRAY['urn:grace:team:bay-area','urn:grace:team:shanghai','urn:grace:team:tokyo','urn:grace:team:berlin'])[1 + floor(random()*4)::int]
  ),
  jsonb_build_object(
    'hand_tracking', (ARRAY['completed','pending','failed'])[1 + floor(random()*3)::int],
    'head_tracking', (ARRAY['completed','pending'])[1 + floor(random()*2)::int]
  ),
  now() - (random() * interval '30 days'),
  now() - (random() * interval '15 days'),
  1
FROM generate_series(1, 50) AS i;

-- Step 2: Insert 10,000 assets referencing random mcap_files
INSERT INTO assets (
  asset_id, mcap_file_id, start_timestamp_ns, end_timestamp_ns,
  status, segment_locator, cf_meta, cf_algo, cf_tag, cf_files,
  created_at, updated_at, version
)
SELECT
  gen_random_uuid() AS asset_id,
  mcap_ids.mcap_file_id,
  base_ts + (i % 200) * 5000000000 AS start_ns,
  base_ts + (i % 200) * 5000000000 + (1 + floor(random()*30))::bigint * 1000000000 AS end_ns,
  (ARRAY['approved','approved','approved','rejected','superseded','archived','pending','disputed'])[1 + floor(random()*8)::int],
  encode(digest(
    mcap_ids.mcap_file_id::text ||
    (base_ts + (i % 200) * 5000000000)::text ||
    (base_ts + (i % 200) * 5000000000 + (1 + floor(random()*30))::bigint * 1000000000)::text,
    'sha1'
  ), 'hex'),
  jsonb_build_object(
    'duration_sec', 1 + floor(random()*30)::int,
    'reviewer', (ARRAY['alice','bob','charlie','diana','eve','frank'])[1 + floor(random()*6)::int],
    'owner', (ARRAY[
      'urn:grace:user:alice','urn:grace:user:bob','urn:grace:user:charlie',
      'urn:grace:user:diana','urn:grace:user:eve','urn:grace:user:frank'
    ])[1 + floor(random()*6)::int],
    'type', (ARRAY['task_demo','calibration','free_motion','manipulation','navigation'])[1 + floor(random()*5)::int],
    'env', (ARRAY['kitchen','warehouse','office','outdoor','lab','factory'])[1 + floor(random()*6)::int],
    'task', (ARRAY['cook_pasta','pick_item','calibrate','wash_dishes','sort_packages','navigate_room','assemble_part'])[1 + floor(random()*7)::int],
    'delivery_count', floor(random()*5)::int,
    'last_delivered_to', CASE WHEN random() > 0.5 THEN 'urn:grace:customer:' || chr(65 + floor(random()*5)::int) ELSE '' END,
    'retention_tier', (ARRAY['standard','archive','critical'])[1 + floor(random()*3)::int],
    'archive_after_days', (ARRAY[30,60,90,180])[1 + floor(random()*4)::int],
    'delete_after_days', (ARRAY[180,365,730])[1 + floor(random()*3)::int],
    'total_size_bytes', floor(random() * 50000000)::bigint
  ),
  -- cf_algo: random algo results
  CASE floor(random()*4)::int
    WHEN 0 THEN jsonb_build_object(
      'hand_tracking@1.2.0:status', (ARRAY['ok','failed','running','pending'])[1 + floor(random()*4)::int],
      'hand_tracking@1.2.0:method', 'ray_batch',
      'hand_tracking@1.2.0:run_id', 'dagster-run-' || lpad(floor(random()*9999)::text, 4, '0')
    )
    WHEN 1 THEN jsonb_build_object(
      'deface@2.0.0:status', (ARRAY['ok','failed','running','pending'])[1 + floor(random()*4)::int],
      'deface@2.0.0:method', 'local',
      'deface@2.0.0:output_uri', 'gs://grace-algo/deface/seg-' || i || '.mcap'
    )
    WHEN 2 THEN jsonb_build_object(
      'sam2@1.0.0:status', (ARRAY['ok','failed','pending'])[1 + floor(random()*3)::int],
      'tracker@3.0.0:status', (ARRAY['ok','running','pending'])[1 + floor(random()*3)::int]
    )
    ELSE jsonb_build_object(
      'hand_tracking@1.2.0:status', (ARRAY['ok','ok','failed','pending'])[1 + floor(random()*4)::int],
      'deface@2.0.0:status', (ARRAY['ok','ok','running'])[1 + floor(random()*3)::int],
      'sam2@1.0.0:status', (ARRAY['ok','failed','pending','blocked'])[1 + floor(random()*4)::int]
    )
  END,
  -- cf_tag: random tags
  jsonb_build_object(
    'priority', (ARRAY['critical','high','medium','low'])[1 + floor(random()*4)::int],
    'quality', (ARRAY['excellent','good','acceptable','poor','unusable'])[1 + floor(random()*5)::int],
    'scene', (ARRAY['indoor','outdoor','warehouse','office','kitchen','lab'])[1 + floor(random()*6)::int]
  ) || CASE WHEN random() > 0.7 THEN jsonb_build_object('note', (ARRAY[
    '数据质量优秀','需要重新标注','光照条件差','遮挡严重','动作流畅',
    '背景杂乱','标注完成','待审核','已修正','测试数据'
  ])[1 + floor(random()*10)::int]) ELSE '{}'::jsonb END,
  -- cf_files
  jsonb_build_object(
    'raw_mcap', 'gs://grace-raw/mcap/2026-04/recording-' || lpad((i % 50 + 1)::text, 4, '0') || '.mcap'
  ) || CASE WHEN random() > 0.5 THEN jsonb_build_object(
    'thumbnail', 'gs://grace-derived/thumbnails/seg-' || i || '.jpg',
    'preview_manifest', 'gs://grace-derived/previews/seg-' || i || '/manifest.json'
  ) ELSE '{}'::jsonb END,
  -- timestamps
  now() - (random() * interval '60 days'),
  now() - (random() * interval '30 days'),
  1 + floor(random()*5)::int
FROM generate_series(1, 10000) AS i
CROSS JOIN LATERAL (
  SELECT mcap_file_id, 1700000000000000000 + (row_number() OVER ())::bigint * 100000000000 AS base_ts
  FROM mcap_files
  ORDER BY random()
  LIMIT 1
) AS mcap_ids;

-- Step 3: Insert some deliveries and delivery_items for the new assets
INSERT INTO deliveries (delivery_id, customer_id, status, delivered_at, cf_meta, created_at, updated_at, version)
SELECT
  gen_random_uuid(),
  'urn:grace:customer:' || chr(65 + (i % 5)),
  (ARRAY['pending','delivered','accepted','rejected'])[1 + floor(random()*4)::int],
  CASE WHEN random() > 0.3 THEN now() - (random() * interval '20 days') ELSE NULL END,
  jsonb_build_object(
    'manifest_uri', 'gs://deliveries/batch-' || i || '/manifest.json',
    'contract_id', 'CT-2026-' || lpad(i::text, 4, '0'),
    'note', 'Mock delivery batch ' || i,
    'asset_count', 10 + floor(random()*50)::int,
    'owner', (ARRAY['urn:grace:user:alice','urn:grace:user:bob','urn:grace:user:charlie'])[1 + floor(random()*3)::int]
  ),
  now() - (random() * interval '30 days'),
  now() - (random() * interval '15 days'),
  1
FROM generate_series(1, 20) AS i;

-- Step 4: Link some assets to deliveries
INSERT INTO delivery_items (delivery_id, asset_id)
SELECT d.delivery_id, a.asset_id
FROM (SELECT delivery_id, row_number() OVER (ORDER BY random()) AS rn FROM deliveries) d
CROSS JOIN LATERAL (
  SELECT asset_id FROM assets ORDER BY random() LIMIT (3 + floor(random()*10)::int)
) a
WHERE d.rn <= 20
ON CONFLICT DO NOTHING;

-- Step 5: Insert algo events for a sample of assets
INSERT INTO asset_algo_events (event_id, asset_id, algo_key, prev_status, new_status, run_id, reason, created_at)
SELECT
  gen_random_uuid(),
  a.asset_id,
  (ARRAY['hand_tracking@1.2.0','deface@2.0.0','sam2@1.0.0','tracker@3.0.0'])[1 + floor(random()*4)::int],
  (ARRAY[NULL,'pending','running','pending'])[1 + floor(random()*4)::int],
  (ARRAY['pending','running','ok','failed'])[1 + floor(random()*4)::int],
  'dagster-run-' || lpad(floor(random()*9999)::text, 4, '0'),
  CASE WHEN random() > 0.7 THEN 'timeout' WHEN random() > 0.5 THEN 'oom' ELSE NULL END,
  now() - (random() * interval '30 days')
FROM (SELECT asset_id FROM assets ORDER BY random() LIMIT 2000) a;

-- Summary
SELECT 'assets' AS tbl, count(*) FROM assets
UNION ALL SELECT 'mcap_files', count(*) FROM mcap_files
UNION ALL SELECT 'deliveries', count(*) FROM deliveries
UNION ALL SELECT 'delivery_items', count(*) FROM delivery_items
UNION ALL SELECT 'algo_events', count(*) FROM asset_algo_events;
