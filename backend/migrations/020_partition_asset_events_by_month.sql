-- 020_partition_asset_events_by_month.sql
-- Convert asset_events into a monthly range-partitioned table on occurred_at.
-- Existing rows are migrated into month partitions and a default partition.

DO $$
DECLARE
  is_partitioned bool;
  min_ts timestamptz;
  max_ts timestamptz;
  part_start timestamptz;
  part_end timestamptz;
  part_name text;
BEGIN
  SELECT EXISTS (
    SELECT 1
    FROM pg_partitioned_table pt
    JOIN pg_class c ON c.oid = pt.partrelid
    WHERE c.relname = 'asset_events'
  ) INTO is_partitioned;
  IF is_partitioned THEN
    RAISE NOTICE 'asset_events is already partitioned; skipping migration 020';
    RETURN;
  END IF;

  IF NOT EXISTS (
    SELECT 1
    FROM information_schema.tables
    WHERE table_schema = 'public' AND table_name = 'asset_events'
  ) THEN
    RAISE EXCEPTION 'asset_events table not found';
  END IF;

  IF EXISTS (
    SELECT 1 FROM pg_class
    WHERE relkind = 'S' AND relname = 'asset_events_event_seq_seq'
  ) THEN
    EXECUTE 'ALTER SEQUENCE asset_events_event_seq_seq OWNED BY NONE';
  ELSE
    CREATE SEQUENCE asset_events_event_seq_seq;
  END IF;

  ALTER TABLE asset_events RENAME TO asset_events_legacy;

  CREATE TABLE asset_events (
    event_id                UUID         NOT NULL DEFAULT gen_random_uuid(),
    event_seq               BIGINT       NOT NULL DEFAULT nextval('asset_events_event_seq_seq'),
    event_type              TEXT         NOT NULL,
    aggregate_type          TEXT         NOT NULL DEFAULT 'asset',
    payload_schema_version  TEXT         NOT NULL DEFAULT 'v1',
    asset_id                TEXT         REFERENCES assets(asset_id),
    mcap_file_id            TEXT         REFERENCES mcap_files(mcap_file_id),
    tenant_id               TEXT,
    project_id              TEXT,
    event_source            TEXT         NOT NULL,
    actor_type              TEXT,
    actor_id                TEXT,
    request_id              TEXT,
    idempotency_key         TEXT,
    run_id                  TEXT,
    occurred_at             TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_at              TIMESTAMPTZ  NOT NULL DEFAULT now(),
    publish_state           TEXT         NOT NULL DEFAULT 'pending',
    published_at            TIMESTAMPTZ,
    event_payload           JSONB        NOT NULL DEFAULT '{}'::jsonb,
    retry_count             INT          NOT NULL DEFAULT 0,
    last_error              TEXT,
    PRIMARY KEY (event_seq, occurred_at)
  ) PARTITION BY RANGE (occurred_at);

  SELECT min(occurred_at), max(occurred_at)
    INTO min_ts, max_ts
  FROM asset_events_legacy;
  IF min_ts IS NULL THEN
    min_ts := date_trunc('month', now());
    max_ts := min_ts + interval '1 month';
  ELSE
    min_ts := date_trunc('month', min_ts);
    max_ts := date_trunc('month', max_ts) + interval '1 month';
  END IF;

  part_start := min_ts;
  WHILE part_start < max_ts + interval '1 month' LOOP
    part_end := part_start + interval '1 month';
    part_name := format('asset_events_p_%s', to_char(part_start, 'YYYYMM'));
    EXECUTE format(
      'CREATE TABLE IF NOT EXISTS %I PARTITION OF asset_events FOR VALUES FROM (%L) TO (%L)',
      part_name, part_start, part_end
    );
    part_start := part_end;
  END LOOP;

  CREATE TABLE IF NOT EXISTS asset_events_default PARTITION OF asset_events DEFAULT;

  INSERT INTO asset_events (
    event_id, event_seq, event_type, aggregate_type, payload_schema_version,
    asset_id, mcap_file_id, tenant_id, project_id,
    event_source, actor_type, actor_id, request_id, idempotency_key, run_id,
    occurred_at, created_at, publish_state, published_at,
    event_payload, retry_count, last_error
  )
  SELECT
    event_id, event_seq, event_type, aggregate_type, payload_schema_version,
    asset_id, mcap_file_id, tenant_id, project_id,
    event_source, actor_type, actor_id, request_id, idempotency_key, run_id,
    occurred_at, created_at, publish_state, published_at,
    event_payload, retry_count, last_error
  FROM asset_events_legacy
  ORDER BY event_seq;

  PERFORM setval(
    'asset_events_event_seq_seq',
    COALESCE((SELECT max(event_seq) FROM asset_events), 1),
    true
  );
  EXECUTE 'ALTER SEQUENCE asset_events_event_seq_seq OWNED BY asset_events.event_seq';

  DROP TABLE asset_events_legacy;
END $$;

CREATE INDEX IF NOT EXISTS idx_asset_events_type_time
  ON asset_events (event_type, occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_asset_events_asset
  ON asset_events (asset_id, occurred_at DESC) WHERE asset_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_asset_events_run
  ON asset_events (run_id) WHERE run_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_asset_events_publish_pending
  ON asset_events (publish_state, event_seq) WHERE publish_state = 'pending';
CREATE INDEX IF NOT EXISTS idx_asset_events_tenant_project
  ON asset_events (tenant_id, project_id);
CREATE INDEX IF NOT EXISTS idx_asset_events_aggregate_time
  ON asset_events (aggregate_type, occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_asset_events_retention
  ON asset_events (created_at) WHERE publish_state = 'published';
CREATE INDEX IF NOT EXISTS idx_asset_events_retry_high
  ON asset_events (retry_count) WHERE retry_count > 5 AND publish_state = 'pending';
