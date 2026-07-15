-- CYB-3490: workflow-level progress projection ("done/total" from Argo
-- status.progress). Continuous batch progress reads this single column
-- instead of per-node rows (which are now archived once at terminal).
ALTER TABLE "pipeline_runs" ADD COLUMN "progress" text NOT NULL DEFAULT '';
