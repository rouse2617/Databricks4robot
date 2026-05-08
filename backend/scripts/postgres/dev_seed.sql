-- dev_seed.sql — optional local demo data (3 McapFiles, 5 Assets, deliveries, etc.)
-- mcap_file_id values are 8-char [0-9A-Za-z] (same as asset_id / McapFile primary key).
-- Schema target: migrations through 017+. Apply manually:
--   make local-dev-seed   /   bash backend/scripts/apply_dev_seed.sh
-- Re-applying fails with PK conflicts unless tables are truncated.

-- ============================================================
-- McapFiles (3 records)
-- ============================================================
INSERT INTO mcap_files (
    mcap_file_id,
    raw_hash_md5,
    is_deleted,
    mcap_uri,
    size_bytes,
    file_duration_ms,
    start_timestamp_ns,
    end_timestamp_ns,
    channel_count,
    chunk_count,
    ingest_state,
    owner,
    metadata,
    process_state,
    created_at,
    updated_at,
    version
) VALUES
(
    'demoMc01',
    'd41d8cd98f00b204e9800998ecf8427e',
    FALSE,
    'gs://grace-raw/mcap/2026-04/recording-001.mcap',
    104857600,
    60000,
    1700000000000000000,
    1700000060000000000,
    12,
    48,
    'summarized',
    'urn:grace:team:bay-area',
    '{"ingest_note":"seed demo"}'::jsonb,
    '{"hand_tracking":"completed","head_tracking":"completed","deface":"completed"}'::jsonb,
    '2026-04-20T08:00:00Z',
    '2026-04-20T09:00:00Z',
    1
),
(
    'demoMc02',
    'a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6',
    FALSE,
    'gs://grace-raw/mcap/2026-04/recording-002.mcap',
    52428800,
    100000,
    1700000100000000000,
    1700000200000000000,
    8,
    24,
    'summarized',
    'urn:grace:team:shanghai',
    '{}'::jsonb,
    '{"hand_tracking":"completed","body_tracking":"pending"}'::jsonb,
    '2026-04-20T10:00:00Z',
    '2026-04-20T11:00:00Z',
    1
),
(
    'demoMc03',
    'ff00ff00ff00ff00ff00ff00ff00ff00',
    FALSE,
    'gs://grace-raw/mcap/2026-04/recording-003.mcap',
    209715200,
    100000000,
    1700000300000000000,
    1700000400000000000,
    16,
    64,
    'pending',
    'urn:grace:team:bay-area',
    '{}'::jsonb,
    '{}'::jsonb,
    '2026-04-21T06:00:00Z',
    '2026-04-21T06:00:00Z',
    1
);

-- ============================================================
-- Assets (5 records spanning 2+ McapFiles)
-- ============================================================
INSERT INTO assets (
    asset_id,
    mcap_file_id,
    start_timestamp_ns,
    end_timestamp_ns,
    status,
    segment_locator,
    asset_type,
    lifecycle_state,
    duration_ms,
    owner,
    reviewer,
    delivery_count,
    last_delivered_at,
    last_delivered_to,
    retention_tier,
    metadata,
    files,
    created_at,
    updated_at,
    version
) VALUES
(
    'aset0001',
    'demoMc01',
    1700000000000000000,
    1700000005000000000,
    'approved',
    encode(digest('demoMc01' || '1700000000000000000' || '1700000005000000000', 'sha1'), 'hex'),
    'segment',
    'ready',
    5000,
    'urn:grace:user:alice',
    'alice',
    1,
    '2026-04-20T14:00:00Z',
    'urn:grace:customer:A',
    'standard',
    '{"type":"task_demo","env":"kitchen","task":"cook_pasta","archive_after_days":90,"delete_after_days":365,"total_size_bytes":10485760}'::jsonb,
    '{"raw_mcap":"gs://grace-raw/mcap/2026-04/recording-001.mcap","hand_tracking@1.2.0":"gs://grace-algo/ht/seg01.npz"}'::jsonb,
    '2026-04-20T09:00:00Z',
    '2026-04-20T09:10:00Z',
    2
),
(
    'aset0002',
    'demoMc01',
    1700000005000000000,
    1700000010000000000,
    'approved',
    encode(digest('demoMc01' || '1700000005000000000' || '1700000010000000000', 'sha1'), 'hex'),
    'segment',
    'ready',
    5000,
    'urn:grace:user:bob',
    '',
    0,
    NULL,
    '',
    'standard',
    '{"type":"task_demo","env":"kitchen","task":"cook_pasta","archive_after_days":90,"delete_after_days":365,"total_size_bytes":0}'::jsonb,
    '{"raw_mcap":"gs://grace-raw/mcap/2026-04/recording-001.mcap"}'::jsonb,
    '2026-04-20T09:00:00Z',
    '2026-04-20T09:00:00Z',
    1
),
(
    'aset0003',
    'demoMc01',
    1700000010000000000,
    1700000015000000000,
    'approved',
    encode(digest('demoMc01' || '1700000010000000000' || '1700000015000000000', 'sha1'), 'hex'),
    'segment',
    'rejected',
    5000,
    'urn:grace:user:alice',
    'alice',
    0,
    NULL,
    '',
    'standard',
    '{"type":"task_demo","env":"warehouse","task":"pick_item","archive_after_days":90,"delete_after_days":365,"total_size_bytes":0}'::jsonb,
    '{}'::jsonb,
    '2026-04-20T09:05:00Z',
    '2026-04-20T09:05:00Z',
    1
),
(
    'aset0004',
    'demoMc02',
    1700000100000000000,
    1700000108000000000,
    'approved',
    encode(digest('demoMc02' || '1700000100000000000' || '1700000108000000000', 'sha1'), 'hex'),
    'segment',
    'ready',
    8000,
    'urn:grace:user:charlie',
    '',
    2,
    NULL,
    '',
    'standard',
    '{"type":"calibration","env":"office","task":"calibrate","archive_after_days":90,"delete_after_days":365,"total_size_bytes":20971520}'::jsonb,
    '{"raw_mcap":"gs://grace-raw/mcap/2026-04/recording-002.mcap","deface@2.0.0":"gs://grace-algo/deface/seg04.mcap"}'::jsonb,
    '2026-04-20T11:00:00Z',
    '2026-04-20T11:05:00Z',
    3
),
(
    'aset0005',
    'demoMc02',
    1700000108000000000,
    1700000120000000000,
    'approved',
    encode(digest('demoMc02' || '1700000108000000000' || '1700000120000000000', 'sha1'), 'hex'),
    'segment',
    'ready',
    12000,
    'urn:grace:user:alice',
    '',
    1,
    NULL,
    'urn:grace:customer:A',
    'archive',
    '{"type":"task_demo","env":"kitchen","task":"wash_dishes","archive_after_days":30,"delete_after_days":180,"total_size_bytes":5242880}'::jsonb,
    '{"raw_mcap":"gs://grace-raw/mcap/2026-04/recording-002.mcap"}'::jsonb,
    '2026-04-20T12:00:00Z',
    '2026-04-20T12:05:00Z',
    2
);

-- ============================================================
-- Deliveries (2 records)
-- ============================================================
INSERT INTO deliveries (
    delivery_id,
    customer_id,
    status,
    delivered_at,
    contract_id,
    delivered_by,
    manifest_uri,
    item_count,
    metadata,
    created_at,
    updated_at,
    version
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
    '{"note":"Q1 batch delivery","legacy_cf":{"asset_count":2}}'::jsonb,
    '2026-04-20T13:55:00Z',
    '2026-04-20T14:00:00Z',
    2
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
    '{"note":"Calibration data","legacy_cf":{"asset_count":2}}'::jsonb,
    '2026-04-21T08:00:00Z',
    '2026-04-21T08:00:00Z',
    1
);

-- ============================================================
-- DeliveryItems (4 records)
-- ============================================================
INSERT INTO delivery_items (delivery_id, asset_id) VALUES
('cccccccc-3333-4000-8000-000000000001', 'aset0001'),
('cccccccc-3333-4000-8000-000000000001', 'aset0005'),
('cccccccc-3333-4000-8000-000000000002', 'aset0004'),
('cccccccc-3333-4000-8000-000000000002', 'aset0005');

-- ============================================================
-- AlgoEvents (3 records: pending → running → ok)
-- ============================================================
INSERT INTO asset_algo_events (event_id, asset_id, algo_key, prev_status, new_status, run_id, reason, created_at) VALUES
(
    'dddddddd-4444-4000-8000-000000000001',
    'aset0001',
    'hand_tracking@1.2.0',
    NULL,
    'pending',
    NULL,
    NULL,
    '2026-04-20T09:00:00Z'
),
(
    'dddddddd-4444-4000-8000-000000000002',
    'aset0001',
    'hand_tracking@1.2.0',
    'pending',
    'running',
    'dagster-run-001',
    NULL,
    '2026-04-20T09:01:00Z'
),
(
    'dddddddd-4444-4000-8000-000000000003',
    'aset0001',
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
