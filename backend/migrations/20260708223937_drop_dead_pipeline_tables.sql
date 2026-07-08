-- Reconciliation (idempotent): drop superseded pipeline-execution tables from
-- the old design (replaced by pipeline_run_nodes / pipeline_run_events etc.).
-- All are 0 code references and referenced only among themselves (no external
-- FK / view depends on them). node_runs/node_attempts/pipeline_shards were
-- empty; pipeline_definitions/pipeline_revisions held 7 rows each of old dev
-- test data (backed up to scratch CSV before this drop).
--
-- KEPT (not dropped): asset_algo_events — still referenced by backend code.
--
-- IF EXISTS => no-op on prod (which never had these). Dropped in FK order
-- (referencing table before referenced).
DROP TABLE IF EXISTS node_attempts;
DROP TABLE IF EXISTS node_runs;
DROP TABLE IF EXISTS pipeline_shards;
DROP TABLE IF EXISTS pipeline_revisions;
DROP TABLE IF EXISTS pipeline_definitions;
