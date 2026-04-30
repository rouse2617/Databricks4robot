-- 002_seed.sql — Representative test data for local development.
-- Executed automatically on first docker-compose up after 001_init.sql.

-- ============================================================
-- McapFiles (3 records)
-- ============================================================
INSERT INTO mcap_files (mcap_file_id, raw_hash_md5, cf_meta, cf_process, created_at, updated_at, version) VALUES
(
    'aaaaaaaa-1111-4000-8000-000000000001',
    'd41d8cd98f00b204e9800998ecf8427e',
    '{"gcs_path":"gs://grace-raw/mcap/2026-04/recording-001.mcap","size_bytes":104857600,"ingest_state":"summarized","start_timestamp_ns":1700000000000000000,"end_timestamp_ns":1700000060000000000,"channel_count":12,"chunk_count":48,"owner":"urn:grace:team:bay-area"}'::jsonb,
    '{"hand_tracking":"completed","head_tracking":"completed","deface":"completed"}'::jsonb,
    '2026-04-20T08:00:00Z', '2026-04-20T09:00:00Z', 1
),
(
    'aaaaaaaa-1111-4000-8000-000000000002',
    'a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6',
    '{"gcs_path":"gs://grace-raw/mcap/2026-04/recording-002.mcap","size_bytes":52428800,"ingest_state":"summarized","start_timestamp_ns":1700000100000000000,"end_timestamp_ns":1700000200000000000,"channel_count":8,"chunk_count":24,"owner":"urn:grace:team:shanghai"}'::jsonb,
    '{"hand_tracking":"completed","body_tracking":"pending"}'::jsonb,
    '2026-04-20T10:00:00Z', '2026-04-20T11:00:00Z', 1
),
(
    'aaaaaaaa-1111-4000-8000-000000000003',
    'ff00ff00ff00ff00ff00ff00ff00ff00',
    '{"gcs_path":"gs://grace-raw/mcap/2026-04/recording-003.mcap","size_bytes":209715200,"ingest_state":"pending","start_timestamp_ns":1700000300000000000,"end_timestamp_ns":1700000400000000000,"channel_count":16,"chunk_count":64,"owner":"urn:grace:team:bay-area"}'::jsonb,
    '{}'::jsonb,
    '2026-04-21T06:00:00Z', '2026-04-21T06:00:00Z', 1
);

-- ============================================================
-- Assets (5 records spanning 2+ McapFiles)
-- segment_locator = sha1(mcap_file_id || start_ns || end_ns)
-- ============================================================
INSERT INTO assets (asset_id, mcap_file_id, start_timestamp_ns, end_timestamp_ns, status, segment_locator, cf_meta, cf_algo, cf_tag, cf_files, created_at, updated_at, version) VALUES
(
    'bbbbbbbb-2222-4000-8000-000000000001',
    'aaaaaaaa-1111-4000-8000-000000000001',
    1700000000000000000, 1700000005000000000,
    'approved',
    encode(digest('aaaaaaaa-1111-4000-8000-000000000001' || '1700000000000000000' || '1700000005000000000', 'sha1'), 'hex'),
    '{"duration_sec":5.0,"reviewer":"alice","owner":"urn:grace:user:alice","type":"task_demo","env":"kitchen","task":"cook_pasta","delivery_count":1,"last_delivered_to":"urn:grace:customer:A","retention_tier":"standard","archive_after_days":90,"delete_after_days":365,"total_size_bytes":10485760}'::jsonb,
    '{"hand_tracking@1.2.0:status":"ok","hand_tracking@1.2.0:started_at":"2026-04-20T09:00:00Z","hand_tracking@1.2.0:finished_at":"2026-04-20T09:05:00Z","hand_tracking@1.2.0:method":"ray_batch","hand_tracking@1.2.0:run_id":"dagster-run-001","hand_tracking@1.2.0:output_uri":"gs://grace-algo/ht/seg01.npz"}'::jsonb,
    '{"priority":"high","quality":"excellent","scene":"indoor"}'::jsonb,
    '{"raw_mcap":"gs://grace-raw/mcap/2026-04/recording-001.mcap","hand_tracking@1.2.0":"gs://grace-algo/ht/seg01.npz"}'::jsonb,
    '2026-04-20T09:00:00Z', '2026-04-20T09:10:00Z', 2
),
(
    'bbbbbbbb-2222-4000-8000-000000000002',
    'aaaaaaaa-1111-4000-8000-000000000001',
    1700000005000000000, 1700000010000000000,
    'approved',
    encode(digest('aaaaaaaa-1111-4000-8000-000000000001' || '1700000005000000000' || '1700000010000000000', 'sha1'), 'hex'),
    '{"duration_sec":5.0,"reviewer":"bob","owner":"urn:grace:user:bob","type":"task_demo","env":"kitchen","task":"cook_pasta","delivery_count":0,"retention_tier":"standard","archive_after_days":90,"delete_after_days":365,"total_size_bytes":0}'::jsonb,
    '{"hand_tracking@1.2.0:status":"pending"}'::jsonb,
    '{"priority":"medium","scene":"indoor"}'::jsonb,
    '{"raw_mcap":"gs://grace-raw/mcap/2026-04/recording-001.mcap"}'::jsonb,
    '2026-04-20T09:00:00Z', '2026-04-20T09:00:00Z', 1
),
(
    'bbbbbbbb-2222-4000-8000-000000000003',
    'aaaaaaaa-1111-4000-8000-000000000001',
    1700000010000000000, 1700000015000000000,
    'rejected',
    encode(digest('aaaaaaaa-1111-4000-8000-000000000001' || '1700000010000000000' || '1700000015000000000', 'sha1'), 'hex'),
    '{"duration_sec":5.0,"reviewer":"alice","owner":"urn:grace:user:alice","type":"task_demo","env":"warehouse","task":"pick_item","delivery_count":0,"retention_tier":"standard","archive_after_days":90,"delete_after_days":365,"total_size_bytes":0}'::jsonb,
    '{}'::jsonb,
    '{"quality":"poor","scene":"warehouse"}'::jsonb,
    '{}'::jsonb,
    '2026-04-20T09:05:00Z', '2026-04-20T09:05:00Z', 1
),
(
    'bbbbbbbb-2222-4000-8000-000000000004',
    'aaaaaaaa-1111-4000-8000-000000000002',
    1700000100000000000, 1700000108000000000,
    'approved',
    encode(digest('aaaaaaaa-1111-4000-8000-000000000002' || '1700000100000000000' || '1700000108000000000', 'sha1'), 'hex'),
    '{"duration_sec":8.0,"reviewer":"charlie","owner":"urn:grace:user:charlie","type":"calibration","env":"office","task":"calibrate","delivery_count":2,"last_delivered_to":"urn:grace:customer:B","retention_tier":"standard","archive_after_days":90,"delete_after_days":365,"total_size_bytes":20971520}'::jsonb,
    '{"deface@2.0.0:status":"ok","deface@2.0.0:started_at":"2026-04-20T11:00:00Z","deface@2.0.0:finished_at":"2026-04-20T11:02:00Z","deface@2.0.0:method":"local","deface@2.0.0:output_uri":"gs://grace-algo/deface/seg04.mcap"}'::jsonb,
    '{"priority":"low","quality":"good","scene":"office"}'::jsonb,
    '{"raw_mcap":"gs://grace-raw/mcap/2026-04/recording-002.mcap","deface@2.0.0":"gs://grace-algo/deface/seg04.mcap"}'::jsonb,
    '2026-04-20T11:00:00Z', '2026-04-20T11:05:00Z', 3
),
(
    'bbbbbbbb-2222-4000-8000-000000000005',
    'aaaaaaaa-1111-4000-8000-000000000002',
    1700000108000000000, 1700000120000000000,
    'approved',
    encode(digest('aaaaaaaa-1111-4000-8000-000000000002' || '1700000108000000000' || '1700000120000000000', 'sha1'), 'hex'),
    '{"duration_sec":12.0,"reviewer":"alice","owner":"urn:grace:user:alice","type":"task_demo","env":"kitchen","task":"wash_dishes","delivery_count":1,"last_delivered_to":"urn:grace:customer:A","retention_tier":"archive","archive_after_days":30,"delete_after_days":180,"total_size_bytes":5242880}'::jsonb,
    '{"hand_tracking@1.2.0:status":"running","hand_tracking@1.2.0:started_at":"2026-04-20T12:00:00Z","hand_tracking@1.2.0:method":"ray_batch","hand_tracking@1.2.0:run_id":"dagster-run-002"}'::jsonb,
    '{"priority":"critical","quality":"acceptable","scene":"indoor"}'::jsonb,
    '{"raw_mcap":"gs://grace-raw/mcap/2026-04/recording-002.mcap"}'::jsonb,
    '2026-04-20T12:00:00Z', '2026-04-20T12:05:00Z', 2
);

-- ============================================================
-- Deliveries (2 records)
-- item_count / manifest_uri / contract_id / delivered_by / metadata are what the
-- API reads; cf_meta kept for legacy parity (includes redundant asset_count).
-- ============================================================
INSERT INTO deliveries (
    delivery_id, customer_id, status, delivered_at,
    contract_id, delivered_by, manifest_uri,
    item_count, metadata, cf_meta,
    created_at, updated_at, version
) VALUES
(
    'cccccccc-3333-4000-8000-000000000001',
    'urn:grace:customer:A',
    'delivered',
    '2026-04-20T14:00:00Z',
    'CT-2026-0401',
    'urn:grace:user:alice',
    'gs://deliveries/cc-01/manifest.json',
    2,
    '{"note":"Q1 batch delivery"}'::jsonb,
    '{"manifest_uri":"gs://deliveries/cc-01/manifest.json","contract_id":"CT-2026-0401","note":"Q1 batch delivery","asset_count":2,"owner":"urn:grace:user:alice"}'::jsonb,
    '2026-04-20T13:55:00Z', '2026-04-20T14:00:00Z', 2
),
(
    'cccccccc-3333-4000-8000-000000000002',
    'urn:grace:customer:B',
    'pending',
    NULL,
    'CT-2026-0502',
    'urn:grace:user:charlie',
    'gs://deliveries/cc-02/manifest.json',
    2,
    '{"note":"Calibration data"}'::jsonb,
    '{"manifest_uri":"gs://deliveries/cc-02/manifest.json","contract_id":"CT-2026-0502","note":"Calibration data","asset_count":2,"owner":"urn:grace:user:charlie"}'::jsonb,
    '2026-04-21T08:00:00Z', '2026-04-21T08:00:00Z', 1
);

-- ============================================================
-- DeliveryItems (4 records)
-- ============================================================
INSERT INTO delivery_items (delivery_id, asset_id) VALUES
('cccccccc-3333-4000-8000-000000000001', 'bbbbbbbb-2222-4000-8000-000000000001'),
('cccccccc-3333-4000-8000-000000000001', 'bbbbbbbb-2222-4000-8000-000000000005'),
('cccccccc-3333-4000-8000-000000000002', 'bbbbbbbb-2222-4000-8000-000000000004'),
('cccccccc-3333-4000-8000-000000000002', 'bbbbbbbb-2222-4000-8000-000000000005');

-- ============================================================
-- AlgoEvents (3 records: pending → running → ok)
-- ============================================================
INSERT INTO asset_algo_events (event_id, asset_id, algo_key, prev_status, new_status, run_id, reason, created_at) VALUES
(
    'dddddddd-4444-4000-8000-000000000001',
    'bbbbbbbb-2222-4000-8000-000000000001',
    'hand_tracking@1.2.0',
    NULL,
    'pending',
    NULL,
    NULL,
    '2026-04-20T09:00:00Z'
),
(
    'dddddddd-4444-4000-8000-000000000002',
    'bbbbbbbb-2222-4000-8000-000000000001',
    'hand_tracking@1.2.0',
    'pending',
    'running',
    'dagster-run-001',
    NULL,
    '2026-04-20T09:01:00Z'
),
(
    'dddddddd-4444-4000-8000-000000000003',
    'bbbbbbbb-2222-4000-8000-000000000001',
    'hand_tracking@1.2.0',
    'running',
    'ok',
    'dagster-run-001',
    NULL,
    '2026-04-20T09:05:00Z'
);

-- ============================================================
-- Outbox Sink Cursors (ES sink bootstrap)
-- ============================================================
INSERT INTO outbox_sink_cursors (sink_name, last_published_seq)
VALUES ('es_assets', 0)
ON CONFLICT DO NOTHING;
