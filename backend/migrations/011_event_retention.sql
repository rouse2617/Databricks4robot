-- 011_event_retention.sql
-- Event retention: clean up old published events.
-- This migration adds a partial index to efficiently find old published events
-- and provides a function for scheduled cleanup.

CREATE INDEX IF NOT EXISTS idx_asset_events_retention
  ON asset_events (created_at)
  WHERE publish_state = 'published';

-- Retention cleanup function: deletes published events older than the given interval.
-- Usage: SELECT event_retention_cleanup('90 days');
CREATE OR REPLACE FUNCTION event_retention_cleanup(retention_interval INTERVAL)
RETURNS BIGINT AS $$
DECLARE
  deleted_count BIGINT;
BEGIN
  DELETE FROM asset_events
  WHERE publish_state IN ('published', 'dlq')
    AND created_at < now() - retention_interval;
  GET DIAGNOSTICS deleted_count = ROW_COUNT;
  RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;
