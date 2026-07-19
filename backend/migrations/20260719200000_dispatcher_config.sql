-- CYB-3679: per-cluster dispatcher tuning, editable online. One row per
-- cluster; absent row = compiled defaults. The submitter reads this each
-- cycle, so a PUT takes effect within one tick without a deploy.
CREATE TABLE "dispatcher_configs" (
  "cluster_id" text NOT NULL,
  "max_concurrency" integer NOT NULL DEFAULT 20,
  "submit_batch" integer NOT NULL DEFAULT 20,
  "rate_per_sec" double precision NOT NULL DEFAULT 10,
  "paused" boolean NOT NULL DEFAULT false,
  "updated_by" text NOT NULL DEFAULT '',
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("cluster_id")
);
