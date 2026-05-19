-- 019_drop_outbox_sink_cursors.sql
-- Multi-sink cursor table is superseded by transactional outbox + Pub/Sub relay
-- (single publish path; no per-sink checkpoint in PostgreSQL).

DROP TABLE IF EXISTS outbox_sink_cursors;
