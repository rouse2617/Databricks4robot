-- CYB-3490: workflow-level progress projection ("done/total" from Argo
-- status.progress). Continuous batch progress reads this single column
-- instead of per-node rows (which are now archived once at terminal).
-- IF NOT EXISTS: dev was hand-applied (scripts/apply-migration-dev.sh) on
-- 2026-07-15 after the deploy pipeline reused a stale migrate image and
-- no-op'd; atlas has no revision row there, so this must be re-runnable.
ALTER TABLE "pipeline_runs" ADD COLUMN IF NOT EXISTS "progress" text NOT NULL DEFAULT '';
