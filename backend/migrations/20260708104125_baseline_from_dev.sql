-- Enable required extensions (live has pg_trgm + pgcrypto; atlas migrate diff does not emit these)
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Create "actions" table
CREATE TABLE "actions" (
  "action_id" text NOT NULL,
  "asset_id" text NOT NULL,
  "start_ns" bigint NOT NULL,
  "end_ns" bigint NOT NULL,
  "action_index" integer NULL,
  "primary_label" text NULL,
  "labels" text[] NOT NULL DEFAULT '{}',
  "description" text NULL,
  "attrs" jsonb NOT NULL DEFAULT '{}',
  "source_type" text NOT NULL DEFAULT 'human',
  "source_name" text NULL,
  "source_version" text NULL,
  "run_id" text NULL,
  "confidence" double precision NULL,
  "external_id" text NULL,
  "tenant_id" text NULL,
  "project_id" text NULL,
  "is_deleted" boolean NOT NULL DEFAULT false,
  "version" bigint NOT NULL DEFAULT 1,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  "task_id" text NULL,
  PRIMARY KEY ("action_id"),
  CONSTRAINT "actions_id_chk" CHECK (action_id ~ '^[0-9A-Za-z]{8}$'::text),
  CONSTRAINT "actions_range_chk" CHECK (end_ns >= start_ns)
);
-- Create index "idx_actions_asset_start" to table: "actions"
CREATE INDEX "idx_actions_asset_start" ON "actions" ("asset_id", "start_ns") WHERE (is_deleted = false);
-- Create index "idx_actions_labels_gin" to table: "actions"
CREATE INDEX "idx_actions_labels_gin" ON "actions" USING GIN ("labels") WHERE (is_deleted = false);
-- Create index "idx_actions_primary_label" to table: "actions"
CREATE INDEX "idx_actions_primary_label" ON "actions" ("primary_label") WHERE ((is_deleted = false) AND (primary_label IS NOT NULL));
-- Create index "idx_actions_run_id" to table: "actions"
CREATE INDEX "idx_actions_run_id" ON "actions" ("source_name", "run_id") WHERE ((is_deleted = false) AND (run_id IS NOT NULL));
-- Create index "uq_actions_external" to table: "actions"
CREATE UNIQUE INDEX "uq_actions_external" ON "actions" ("asset_id", "source_name", "external_id") WHERE ((external_id IS NOT NULL) AND (is_deleted = false));
-- Set comment to table: "actions"
COMMENT ON TABLE "actions" IS 'seg 内时间分段标注 (mcap → seg → action 第三层)。docs/review/data-platform-design.md §5.2.15。';
-- Set comment to column: "task_id" on table: "actions"
COMMENT ON COLUMN "actions"."task_id" IS 'P1.5: FK → annotation_tasks(task_id), nullable placeholder, not yet enforced';
-- Create "algo_runs" table
CREATE TABLE "algo_runs" (
  "run_id" text NOT NULL,
  "algo_name" text NOT NULL,
  "algo_version" text NOT NULL,
  "algo_kind" text NOT NULL,
  "triggered_by" text NOT NULL,
  "status" text NOT NULL DEFAULT 'pending',
  "started_at" timestamptz NULL,
  "finished_at" timestamptz NULL,
  "duration_ns" bigint NULL GENERATED ALWAYS AS (
CASE
    WHEN ((finished_at IS NOT NULL) AND (started_at IS NOT NULL)) THEN ((EXTRACT(epoch FROM (finished_at - started_at)))::bigint * 1000000000)
    ELSE NULL::bigint
END) STORED,
  "input_filter" jsonb NOT NULL DEFAULT '{}',
  "input_asset_ids" text[] NULL,
  "params" jsonb NOT NULL DEFAULT '{}',
  "code_commit" text NULL,
  "image_digest" text NULL,
  "pipeline_name" text NULL,
  "pipeline_version" text NULL,
  "assets_processed" integer NULL,
  "assets_succeeded" integer NULL,
  "assets_failed" integer NULL,
  "actions_created" integer NULL,
  "metrics_written" integer NULL,
  "outputs" jsonb NOT NULL DEFAULT '{}',
  "cpu_seconds" bigint NULL,
  "gpu_seconds" bigint NULL,
  "cost_usd_micros" bigint NULL,
  "error_class" text NULL,
  "error_message" text NULL,
  "tenant_id" text NULL,
  "project_id" text NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  "row_version" bigint NOT NULL DEFAULT 1,
  "external_runtime" text NULL,
  "external_url" text NULL,
  "pipeline_id" text NULL,
  PRIMARY KEY ("run_id"),
  CONSTRAINT "algo_runs_algo_kind_check" CHECK (algo_kind = ANY (ARRAY['processing'::text, 'split'::text, 'qa'::text, 'enrichment'::text, 'pipeline'::text])),
  CONSTRAINT "algo_runs_run_id_check" CHECK (run_id ~ '^[0-9A-Za-z]{16}$'::text),
  CONSTRAINT "algo_runs_status_check" CHECK (status = ANY (ARRAY['pending'::text, 'running'::text, 'ok'::text, 'failed'::text, 'cancelled'::text]))
);
-- Create index "idx_algo_runs_algo" to table: "algo_runs"
CREATE INDEX "idx_algo_runs_algo" ON "algo_runs" ("algo_name", "algo_version", "started_at" DESC);
-- Create index "idx_algo_runs_pipeline_id" to table: "algo_runs"
CREATE INDEX "idx_algo_runs_pipeline_id" ON "algo_runs" ("pipeline_id") WHERE (pipeline_id IS NOT NULL);
-- Create index "idx_algo_runs_started" to table: "algo_runs"
CREATE INDEX "idx_algo_runs_started" ON "algo_runs" ("started_at" DESC);
-- Create index "idx_algo_runs_status" to table: "algo_runs"
CREATE INDEX "idx_algo_runs_status" ON "algo_runs" ("status") WHERE (status = ANY (ARRAY['pending'::text, 'running'::text, 'failed'::text]));
-- Create index "idx_algo_runs_triggered" to table: "algo_runs"
CREATE INDEX "idx_algo_runs_triggered" ON "algo_runs" ("triggered_by") WHERE (triggered_by ~~ 'manual:%'::text);
-- Create "asset_algo_events" table
CREATE TABLE "asset_algo_events" (
  "event_id" uuid NOT NULL,
  "asset_id" text NOT NULL,
  "algo_key" character varying(128) NOT NULL,
  "prev_status" character varying(16) NULL,
  "new_status" character varying(16) NOT NULL,
  "run_id" character varying(64) NULL,
  "reason" character varying(256) NULL,
  "created_at" timestamptz NOT NULL,
  PRIMARY KEY ("event_id")
);
-- Create index "idx_algo_events_algo_status" to table: "asset_algo_events"
CREATE INDEX "idx_algo_events_algo_status" ON "asset_algo_events" ("algo_key", "new_status", "created_at" DESC);
-- Create index "idx_algo_events_asset_created" to table: "asset_algo_events"
CREATE INDEX "idx_algo_events_asset_created" ON "asset_algo_events" ("asset_id", "created_at" DESC);
-- Create index "idx_algo_events_run_id" to table: "asset_algo_events"
CREATE INDEX "idx_algo_events_run_id" ON "asset_algo_events" ("run_id") WHERE (run_id IS NOT NULL);
-- Create "asset_algo_latest" table
CREATE TABLE "asset_algo_latest" (
  "asset_id" text NOT NULL,
  "algo_name" text NOT NULL,
  "algo_version" text NOT NULL,
  "status" text NOT NULL,
  "result_tag" text NULL,
  "result_score" double precision NULL,
  "result_summary" jsonb NOT NULL DEFAULT '{}',
  "run_id" text NULL,
  "method" text NULL,
  "model_uri" text NULL,
  "output_uri" text NULL,
  "error_code" text NULL,
  "error_message" text NULL,
  "started_at" timestamptz NULL,
  "finished_at" timestamptz NULL,
  "tenant_id" text NULL,
  "project_id" text NULL,
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("asset_id", "algo_name")
);
-- Create index "idx_asset_algo_latest_algo_status" to table: "asset_algo_latest"
CREATE INDEX "idx_asset_algo_latest_algo_status" ON "asset_algo_latest" ("algo_name", "status");
-- Create index "idx_asset_algo_latest_algo_version" to table: "asset_algo_latest"
CREATE INDEX "idx_asset_algo_latest_algo_version" ON "asset_algo_latest" ("algo_name", "algo_version");
-- Create index "idx_asset_algo_latest_run" to table: "asset_algo_latest"
CREATE INDEX "idx_asset_algo_latest_run" ON "asset_algo_latest" ("run_id") WHERE (run_id IS NOT NULL);
-- Create index "idx_asset_algo_latest_updated" to table: "asset_algo_latest"
CREATE INDEX "idx_asset_algo_latest_updated" ON "asset_algo_latest" ("updated_at" DESC);
-- Create "set_updated_at" function
CREATE FUNCTION "set_updated_at" () RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$;
-- Create trigger "trg_asset_algo_latest_updated_at"
CREATE TRIGGER "trg_asset_algo_latest_updated_at" BEFORE UPDATE ON "asset_algo_latest" FOR EACH ROW EXECUTE FUNCTION "set_updated_at"();
-- Create "asset_eval_results" table
CREATE TABLE "asset_eval_results" (
  "eval_result_id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "asset_id" text NOT NULL,
  "mcap_file_id" text NULL,
  "target_type" text NOT NULL DEFAULT 'segment',
  "target_id" text NOT NULL DEFAULT '',
  "eval_name" text NOT NULL,
  "eval_version" text NOT NULL,
  "parameter_version" text NULL,
  "run_id" text NULL,
  "status" text NOT NULL DEFAULT 'ok',
  "result_payload" jsonb NOT NULL DEFAULT '{}',
  "output_uri" text NULL,
  "summary_uri" text NULL,
  "source_type" text NOT NULL DEFAULT 'algo',
  "source_name" text NULL,
  "source_version" text NULL,
  "started_at" timestamptz NULL,
  "finished_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("eval_result_id")
);
-- Create index "idx_eval_results_asset" to table: "asset_eval_results"
CREATE INDEX "idx_eval_results_asset" ON "asset_eval_results" ("asset_id", "target_type", "target_id", "created_at" DESC);
-- Create index "idx_eval_results_name_ver" to table: "asset_eval_results"
CREATE INDEX "idx_eval_results_name_ver" ON "asset_eval_results" ("eval_name", "eval_version", "parameter_version", "created_at" DESC);
-- Create index "idx_eval_results_run_id" to table: "asset_eval_results"
CREATE INDEX "idx_eval_results_run_id" ON "asset_eval_results" ("run_id") WHERE (run_id IS NOT NULL);
-- Create trigger "trg_asset_eval_results_updated_at"
CREATE TRIGGER "trg_asset_eval_results_updated_at" BEFORE UPDATE ON "asset_eval_results" FOR EACH ROW EXECUTE FUNCTION "set_updated_at"();
-- Create "asset_events" table
CREATE TABLE "asset_events" (
  "event_id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "event_seq" bigserial NOT NULL,
  "event_type" text NOT NULL,
  "aggregate_type" text NOT NULL DEFAULT 'asset',
  "payload_schema_version" text NOT NULL DEFAULT 'v1',
  "asset_id" text NULL,
  "mcap_file_id" text NULL,
  "tenant_id" text NULL,
  "project_id" text NULL,
  "event_source" text NOT NULL,
  "actor_type" text NULL,
  "actor_id" text NULL,
  "request_id" text NULL,
  "idempotency_key" text NULL,
  "run_id" text NULL,
  "occurred_at" timestamptz NOT NULL DEFAULT now(),
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "publish_state" text NOT NULL DEFAULT 'pending',
  "published_at" timestamptz NULL,
  "event_payload" jsonb NOT NULL DEFAULT '{}',
  "retry_count" integer NOT NULL DEFAULT 0,
  "last_error" text NULL,
  CONSTRAINT "asset_events_pkey1" PRIMARY KEY ("event_seq", "occurred_at")
) PARTITION BY RANGE ("occurred_at");
-- Create index "idx_asset_events_aggregate_time" to table: "asset_events"
CREATE INDEX "idx_asset_events_aggregate_time" ON "asset_events" ("aggregate_type", "occurred_at" DESC);
-- Create index "idx_asset_events_asset" to table: "asset_events"
CREATE INDEX "idx_asset_events_asset" ON "asset_events" ("asset_id", "occurred_at" DESC) WHERE (asset_id IS NOT NULL);
-- Create index "idx_asset_events_publish_pending" to table: "asset_events"
CREATE INDEX "idx_asset_events_publish_pending" ON "asset_events" ("publish_state", "event_seq") WHERE (publish_state = 'pending'::text);
-- Create index "idx_asset_events_retention" to table: "asset_events"
CREATE INDEX "idx_asset_events_retention" ON "asset_events" ("created_at") WHERE (publish_state = 'published'::text);
-- Create index "idx_asset_events_retry_high" to table: "asset_events"
CREATE INDEX "idx_asset_events_retry_high" ON "asset_events" ("retry_count") WHERE ((retry_count > 5) AND (publish_state = 'pending'::text));
-- Create index "idx_asset_events_run" to table: "asset_events"
CREATE INDEX "idx_asset_events_run" ON "asset_events" ("run_id") WHERE (run_id IS NOT NULL);
-- Create index "idx_asset_events_tenant_project" to table: "asset_events"
CREATE INDEX "idx_asset_events_tenant_project" ON "asset_events" ("tenant_id", "project_id");
-- Create index "idx_asset_events_type_time" to table: "asset_events"
CREATE INDEX "idx_asset_events_type_time" ON "asset_events" ("event_type", "occurred_at" DESC);
-- Create "assets" table
CREATE TABLE "assets" (
  "asset_id" text NOT NULL,
  "mcap_file_id" text NULL,
  "start_timestamp_ns" bigint NOT NULL,
  "end_timestamp_ns" bigint NOT NULL,
  "status" character varying(32) NOT NULL DEFAULT 'approved',
  "is_deleted" boolean NULL DEFAULT false,
  "segment_locator" character(40) NULL,
  "asset_type" text NOT NULL DEFAULT 'segment',
  "lifecycle_state" text NOT NULL DEFAULT 'created',
  "duration_ms" bigint NOT NULL DEFAULT 0,
  "owner" text NOT NULL DEFAULT '',
  "reviewer" text NOT NULL DEFAULT '',
  "storage_uri" text NOT NULL DEFAULT '',
  "thumb_uri" text NOT NULL DEFAULT '',
  "retention_tier" text NOT NULL DEFAULT '',
  "expire_at" timestamptz NULL,
  "asset_level" integer NOT NULL DEFAULT 0,
  "parent_asset_id" text NULL,
  "root_asset_id" text NULL,
  "delivery_count" integer NOT NULL DEFAULT 0,
  "last_delivered_at" timestamptz NULL,
  "last_delivered_to" text NOT NULL DEFAULT '',
  "segment_index" integer NULL,
  "parent_start_offset_ms" bigint NULL,
  "parent_end_offset_ms" bigint NULL,
  "split_method" text NULL,
  "split_algo_name" text NULL,
  "split_algo_version" text NULL,
  "split_run_id" text NULL,
  "split_reason" text NULL,
  "metadata" jsonb NOT NULL DEFAULT '{}',
  "files" jsonb NOT NULL DEFAULT '{}',
  "algo_inputs_uris" jsonb NOT NULL DEFAULT '{}',
  "annot_inputs_uris" jsonb NOT NULL DEFAULT '{}',
  "tenant_id" text NULL,
  "project_id" text NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "version" bigint NULL DEFAULT 1,
  "logical_asset_id" text NULL,
  "revision" bigint NULL,
  "is_current" boolean NULL,
  PRIMARY KEY ("asset_id"),
  CONSTRAINT "assets_parent_asset_id_fkey" FOREIGN KEY ("parent_asset_id") REFERENCES "assets" ("asset_id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "assets_root_asset_id_fkey" FOREIGN KEY ("root_asset_id") REFERENCES "assets" ("asset_id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "assets_asset_id_check" CHECK ((asset_id ~ '^[0-9A-Za-z]{8}$'::text) OR (asset_id ~ '^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$'::text)),
  CONSTRAINT "assets_logical_asset_id_check" CHECK ((logical_asset_id IS NULL) OR (logical_asset_id ~ '^[0-9A-Za-z]{8}$'::text) OR (logical_asset_id ~ '^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$'::text)),
  CONSTRAINT "assets_mcap_file_id_check" CHECK ((mcap_file_id IS NULL) OR (mcap_file_id = ''::text) OR (mcap_file_id ~ '^[0-9A-Za-z]{8}$'::text) OR (mcap_file_id ~ '^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$'::text)),
  CONSTRAINT "assets_revision_check" CHECK ((revision IS NULL) OR (revision >= 1)),
  CONSTRAINT "chk_lifecycle_state" CHECK (lifecycle_state = ANY (ARRAY['created'::text, 'processing'::text, 'ready'::text, 'delivered'::text, 'archived'::text, 'superseded'::text, 'failed'::text, 'rejected'::text])),
  CONSTRAINT "chk_mcap_file_required" CHECK ((asset_type = ANY (ARRAY['derived_asset'::text, 'dataset'::text, 'annotation_result'::text, 'ml_model'::text, 'evaluation_report'::text, 'grace_video'::text])) OR (mcap_file_id IS NOT NULL))
);
-- Create index "idx_assets_active_status_updated_at" to table: "assets"
CREATE INDEX "idx_assets_active_status_updated_at" ON "assets" ("status", "updated_at" DESC) WHERE (is_deleted = false);
-- Create index "idx_assets_active_updated_at" to table: "assets"
CREATE INDEX "idx_assets_active_updated_at" ON "assets" ("updated_at" DESC) WHERE (is_deleted = false);
-- Create index "idx_assets_algo_inputs_uris_gin" to table: "assets"
CREATE INDEX "idx_assets_algo_inputs_uris_gin" ON "assets" USING GIN ("algo_inputs_uris");
-- Create index "idx_assets_annot_inputs_uris_gin" to table: "assets"
CREATE INDEX "idx_assets_annot_inputs_uris_gin" ON "assets" USING GIN ("annot_inputs_uris");
-- Create index "idx_assets_asset_id_trgm" to table: "assets"
CREATE INDEX "idx_assets_asset_id_trgm" ON "assets" USING GIN ("asset_id" gin_trgm_ops);
-- Create index "idx_assets_asset_type" to table: "assets"
CREATE INDEX "idx_assets_asset_type" ON "assets" ("asset_type") WHERE (is_deleted = false);
-- Create index "idx_assets_asset_type_trgm" to table: "assets"
CREATE INDEX "idx_assets_asset_type_trgm" ON "assets" USING GIN ("asset_type" gin_trgm_ops);
-- Create index "idx_assets_created_at" to table: "assets"
CREATE INDEX "idx_assets_created_at" ON "assets" ("created_at" DESC);
-- Create index "idx_assets_lifecycle" to table: "assets"
CREATE INDEX "idx_assets_lifecycle" ON "assets" ("lifecycle_state") WHERE (is_deleted = false);
-- Create index "idx_assets_lifecycle_state_trgm" to table: "assets"
CREATE INDEX "idx_assets_lifecycle_state_trgm" ON "assets" USING GIN ("lifecycle_state" gin_trgm_ops);
-- Create index "idx_assets_logical" to table: "assets"
CREATE INDEX "idx_assets_logical" ON "assets" ("logical_asset_id");
-- Create index "idx_assets_mcap_file_id" to table: "assets"
CREATE INDEX "idx_assets_mcap_file_id" ON "assets" ("mcap_file_id");
-- Create index "idx_assets_mcap_file_id_trgm" to table: "assets"
CREATE INDEX "idx_assets_mcap_file_id_trgm" ON "assets" USING GIN ("mcap_file_id" gin_trgm_ops);
-- Create index "idx_assets_metadata_gin" to table: "assets"
CREATE INDEX "idx_assets_metadata_gin" ON "assets" USING GIN ("metadata");
-- Create index "idx_assets_owner_trgm" to table: "assets"
CREATE INDEX "idx_assets_owner_trgm" ON "assets" USING GIN ("owner" gin_trgm_ops);
-- Create index "idx_assets_parent" to table: "assets"
CREATE INDEX "idx_assets_parent" ON "assets" ("parent_asset_id") WHERE (parent_asset_id IS NOT NULL);
-- Create index "idx_assets_reviewer_trgm" to table: "assets"
CREATE INDEX "idx_assets_reviewer_trgm" ON "assets" USING GIN ("reviewer" gin_trgm_ops);
-- Create index "idx_assets_root" to table: "assets"
CREATE INDEX "idx_assets_root" ON "assets" ("root_asset_id") WHERE (root_asset_id IS NOT NULL);
-- Create index "idx_assets_segment_locator" to table: "assets"
CREATE INDEX "idx_assets_segment_locator" ON "assets" ("segment_locator");
-- Create index "idx_assets_status" to table: "assets"
CREATE INDEX "idx_assets_status" ON "assets" ("status");
-- Create index "idx_assets_tenant_project" to table: "assets"
CREATE INDEX "idx_assets_tenant_project" ON "assets" ("tenant_id", "project_id") WHERE (is_deleted = false);
-- Create index "uq_assets_current_per_logical" to table: "assets"
CREATE UNIQUE INDEX "uq_assets_current_per_logical" ON "assets" ("logical_asset_id") WHERE ((is_current = true) AND (is_deleted = false));
-- Create "mcap_files" table
CREATE TABLE "mcap_files" (
  "mcap_file_id" text NOT NULL,
  "raw_hash_md5" character varying(32) NULL,
  "is_deleted" boolean NULL DEFAULT false,
  "mcap_uri" text NOT NULL DEFAULT '',
  "size_bytes" bigint NOT NULL DEFAULT 0,
  "file_duration_ms" bigint NOT NULL DEFAULT 0,
  "start_timestamp_ns" bigint NOT NULL DEFAULT 0,
  "end_timestamp_ns" bigint NOT NULL DEFAULT 0,
  "channel_count" integer NOT NULL DEFAULT 0,
  "chunk_count" integer NOT NULL DEFAULT 0,
  "ingest_state" text NOT NULL DEFAULT 'pending',
  "vendor_id" text NULL,
  "collector_id" text NULL,
  "task_id" text NULL,
  "device_id" text NULL,
  "camera_model" text NULL,
  "data_source" text NULL,
  "location_id" text NULL,
  "scene_id" text NULL,
  "environment_id" text NULL,
  "collection_method" text NULL,
  "owner" text NOT NULL DEFAULT '',
  "retention_tier" text NULL,
  "expire_at" timestamptz NULL,
  "metadata" jsonb NOT NULL DEFAULT '{}',
  "process_state" jsonb NOT NULL DEFAULT '{}',
  "raw_hash_sha256" text NULL,
  "tenant_id" text NULL,
  "project_id" text NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "version" bigint NULL DEFAULT 1,
  PRIMARY KEY ("mcap_file_id"),
  CONSTRAINT "mcap_files_mcap_file_id_check" CHECK (mcap_file_id ~ '^[0-9A-Za-z]{8}$'::text)
);
-- Create index "idx_mcap_files_ingest_state" to table: "mcap_files"
CREATE INDEX "idx_mcap_files_ingest_state" ON "mcap_files" ("ingest_state") WHERE (is_deleted = false);
-- Create index "idx_mcap_files_metadata_gin" to table: "mcap_files"
CREATE INDEX "idx_mcap_files_metadata_gin" ON "mcap_files" USING GIN ("metadata");
-- Create index "idx_mcap_files_tenant_project" to table: "mcap_files"
CREATE INDEX "idx_mcap_files_tenant_project" ON "mcap_files" ("tenant_id", "project_id") WHERE (is_deleted = false);
-- Create index "uq_mcap_files_hash_md5" to table: "mcap_files"
CREATE UNIQUE INDEX "uq_mcap_files_hash_md5" ON "mcap_files" ("raw_hash_md5") WHERE ((raw_hash_md5 IS NOT NULL) AND (is_deleted = false));
-- Modify "asset_events" table
ALTER TABLE "asset_events" ADD CONSTRAINT "asset_events_asset_id_fkey1" FOREIGN KEY ("asset_id") REFERENCES "assets" ("asset_id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "asset_events_mcap_file_id_fkey1" FOREIGN KEY ("mcap_file_id") REFERENCES "mcap_files" ("mcap_file_id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Create "asset_events_default" partition
CREATE TABLE "asset_events_default" PARTITION OF "asset_events" DEFAULT;
-- Create "asset_events_p_202605" partition
CREATE TABLE "asset_events_p_202605" PARTITION OF "asset_events" FOR VALUES FROM ('2026-05-01 00:00:00+00') TO ('2026-06-01 00:00:00+00');
-- Create "asset_events_p_202606" partition
CREATE TABLE "asset_events_p_202606" PARTITION OF "asset_events" FOR VALUES FROM ('2026-06-01 00:00:00+00') TO ('2026-07-01 00:00:00+00');
-- Create "asset_metrics" table
CREATE TABLE "asset_metrics" (
  "asset_id" text NOT NULL,
  "target_type" text NOT NULL DEFAULT 'segment',
  "target_id" text NOT NULL DEFAULT '',
  "metric_key" text NOT NULL,
  "metric_type" text NOT NULL DEFAULT 'float',
  "metric_unit" text NULL,
  "metric_value" double precision NULL,
  "metric_value_int" bigint NULL,
  "metric_value_text" text NULL,
  "metric_value_bool" boolean NULL,
  "eval_name" text NOT NULL,
  "eval_version" text NOT NULL,
  "parameter_version" text NULL,
  "run_id" text NULL,
  "source_type" text NOT NULL DEFAULT 'algo',
  "source_name" text NULL,
  "confidence" double precision NULL,
  "recorded_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("asset_id", "target_type", "target_id", "metric_key", "eval_name", "eval_version")
);
-- Create index "idx_asset_metrics_asset" to table: "asset_metrics"
CREATE INDEX "idx_asset_metrics_asset" ON "asset_metrics" ("asset_id", "target_type", "target_id");
-- Create index "idx_asset_metrics_key_value" to table: "asset_metrics"
CREATE INDEX "idx_asset_metrics_key_value" ON "asset_metrics" ("metric_key", "metric_value");
-- Create trigger "trg_asset_metrics_updated_at"
CREATE TRIGGER "trg_asset_metrics_updated_at" BEFORE UPDATE ON "asset_metrics" FOR EACH ROW EXECUTE FUNCTION "set_updated_at"();
-- Create "asset_relations" table
CREATE TABLE "asset_relations" (
  "parent_asset_id" text NOT NULL,
  "child_asset_id" text NOT NULL,
  "relation_type" text NOT NULL,
  "method" text NULL,
  "algo_name" text NULL,
  "algo_version" text NULL,
  "run_id" text NULL,
  "parent_start_offset_ms" bigint NULL,
  "parent_end_offset_ms" bigint NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "metadata" jsonb NOT NULL DEFAULT '{}',
  PRIMARY KEY ("parent_asset_id", "child_asset_id", "relation_type"),
  CONSTRAINT "chk_relation_type" CHECK (relation_type = ANY (ARRAY['split_from'::text, 'derived_from'::text, 'contains'::text, 'sampled_from'::text, 'merged_from'::text, 'revision_of'::text, 'annotated_from'::text, 'materialized_from'::text, 'trained_from'::text, 'evaluated_on'::text, 'validated_on'::text, 'configured_by'::text, 'fine_tuned_from'::text, 'features_from'::text, 'tested_on'::text, 'evaluates'::text, 'compares_to'::text, 'calibrated_from'::text, 'generated_by'::text]))
);
-- Create index "idx_asset_relations_child" to table: "asset_relations"
CREATE INDEX "idx_asset_relations_child" ON "asset_relations" ("child_asset_id", "relation_type");
-- Create "asset_tags" table
CREATE TABLE "asset_tags" (
  "asset_id" text NOT NULL,
  "tag_key" text NOT NULL,
  "tag_value" text NOT NULL,
  "tag_value_num" double precision NULL,
  "tag_value_bool" boolean NULL,
  "tag_type" text NOT NULL DEFAULT 'string',
  "source_type" text NOT NULL DEFAULT 'human',
  "source_name" text NULL,
  "source_version" text NULL,
  "run_id" text NULL,
  "confidence" double precision NULL,
  "tenant_id" text NULL,
  "project_id" text NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  "id" bigserial NOT NULL,
  "applied_at" timestamptz NOT NULL DEFAULT now(),
  "source_version_norm" text NULL GENERATED ALWAYS AS (COALESCE(source_version, ''::text)) STORED,
  PRIMARY KEY ("id"),
  CONSTRAINT "uq_asset_tags_identity" UNIQUE ("asset_id", "tag_key", "tag_value", "source_type", "source_version_norm")
);
-- Create index "idx_asset_tags_key_value_bool" to table: "asset_tags"
CREATE INDEX "idx_asset_tags_key_value_bool" ON "asset_tags" ("tag_key", "tag_value_bool", "asset_id") WHERE (tag_type = 'bool'::text);
-- Create index "idx_asset_tags_key_value_num" to table: "asset_tags"
CREATE INDEX "idx_asset_tags_key_value_num" ON "asset_tags" ("tag_key", "tag_value_num", "asset_id") WHERE (tag_type = 'number'::text);
-- Create index "idx_asset_tags_key_value_str" to table: "asset_tags"
CREATE INDEX "idx_asset_tags_key_value_str" ON "asset_tags" ("tag_key", "tag_value", "asset_id");
-- Create index "idx_asset_tags_tag_key" to table: "asset_tags"
CREATE INDEX "idx_asset_tags_tag_key" ON "asset_tags" ("tag_key");
-- Create index "idx_asset_tags_tag_value_trgm" to table: "asset_tags"
CREATE INDEX "idx_asset_tags_tag_value_trgm" ON "asset_tags" USING GIN ("tag_value" gin_trgm_ops);
-- Create index "idx_asset_tags_tenant_project" to table: "asset_tags"
CREATE INDEX "idx_asset_tags_tenant_project" ON "asset_tags" ("tenant_id", "project_id");
-- Create index "idx_atags_lookup" to table: "asset_tags"
CREATE INDEX "idx_atags_lookup" ON "asset_tags" ("tag_key", "tag_value", "asset_id");
-- Create index "idx_atags_propagation" to table: "asset_tags"
CREATE INDEX "idx_atags_propagation" ON "asset_tags" ("tag_key") WHERE (tag_key ~~ 'compliance.%'::text);
-- Create index "idx_atags_run" to table: "asset_tags"
CREATE INDEX "idx_atags_run" ON "asset_tags" ("run_id") WHERE (run_id IS NOT NULL);
-- Create index "idx_atags_source" to table: "asset_tags"
CREATE INDEX "idx_atags_source" ON "asset_tags" ("source_type", "source_name", "source_version");
-- Create trigger "trg_asset_tags_updated_at"
CREATE TRIGGER "trg_asset_tags_updated_at" BEFORE UPDATE ON "asset_tags" FOR EACH ROW EXECUTE FUNCTION "set_updated_at"();
-- Create "asset_usage_stats" table
CREATE TABLE "asset_usage_stats" (
  "id" bigserial NOT NULL,
  "asset_id" text NOT NULL,
  "logical_asset_id" text NULL,
  "view_count" integer NOT NULL DEFAULT 0,
  "last_viewed_at" timestamptz NULL,
  "favorite_count" integer NOT NULL DEFAULT 0,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "asset_usage_stats_asset_id_key" UNIQUE ("asset_id")
);
-- Create index "idx_asset_usage_stats_logical" to table: "asset_usage_stats"
CREATE INDEX "idx_asset_usage_stats_logical" ON "asset_usage_stats" ("logical_asset_id");
-- Create "nullify_empty_mcap" function
CREATE FUNCTION "nullify_empty_mcap" () RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF NEW.mcap_file_id = '' THEN NEW.mcap_file_id = NULL; END IF;
  RETURN NEW;
END;
$$;
-- Create trigger "trg_nullify_empty_mcap"
CREATE TRIGGER "trg_nullify_empty_mcap" BEFORE INSERT OR UPDATE ON "assets" FOR EACH ROW EXECUTE FUNCTION "nullify_empty_mcap"();
-- Create "audit_events" table
CREATE TABLE "audit_events" (
  "event_id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "actor" text NOT NULL,
  "action" text NOT NULL,
  "resource_type" text NOT NULL,
  "resource_ids" text[] NOT NULL,
  "request_summary" jsonb NULL DEFAULT '{}',
  "request_id" text NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("event_id")
);
-- Create index "idx_audit_events_action_created" to table: "audit_events"
CREATE INDEX "idx_audit_events_action_created" ON "audit_events" ("action", "created_at" DESC);
-- Create index "idx_audit_events_actor_created" to table: "audit_events"
CREATE INDEX "idx_audit_events_actor_created" ON "audit_events" ("actor", "created_at" DESC);
-- Create "backfill_items" table
CREATE TABLE "backfill_items" (
  "id" text NOT NULL,
  "job_id" text NOT NULL,
  "asset_id" text NOT NULL,
  "status" text NOT NULL DEFAULT 'pending',
  "workflow_name" text NULL,
  "error_message" text NULL,
  "started_at" timestamptz NULL,
  "finished_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "pipeline_run_id" text NULL,
  "attempts" integer NOT NULL DEFAULT 0,
  PRIMARY KEY ("id")
);
-- Create index "idx_backfill_items_job_id" to table: "backfill_items"
CREATE INDEX "idx_backfill_items_job_id" ON "backfill_items" ("job_id");
-- Create index "idx_backfill_items_pipeline_run_id" to table: "backfill_items"
CREATE INDEX "idx_backfill_items_pipeline_run_id" ON "backfill_items" ("pipeline_run_id") WHERE (pipeline_run_id IS NOT NULL);
-- Create index "idx_backfill_items_status" to table: "backfill_items"
CREATE INDEX "idx_backfill_items_status" ON "backfill_items" ("status");
-- Create "backfill_jobs" table
CREATE TABLE "backfill_jobs" (
  "id" text NOT NULL,
  "name" text NOT NULL,
  "template_id" text NOT NULL,
  "filter_json" jsonb NULL,
  "total_count" integer NOT NULL DEFAULT 0,
  "completed_count" integer NOT NULL DEFAULT 0,
  "failed_count" integer NOT NULL DEFAULT 0,
  "status" text NOT NULL DEFAULT 'running',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  "pipeline_run_id" text NULL,
  "template_version" integer NULL,
  "pilot_count" integer NOT NULL DEFAULT 0,
  "pilot_phase" text NOT NULL DEFAULT 'none',
  "created_by" text NULL,
  "finished_at" timestamptz NULL,
  "notification_sent_at" timestamptz NULL,
  PRIMARY KEY ("id")
);
-- Create index "idx_backfill_jobs_created_at" to table: "backfill_jobs"
CREATE INDEX "idx_backfill_jobs_created_at" ON "backfill_jobs" ("created_at" DESC);
-- Create index "idx_backfill_jobs_created_by" to table: "backfill_jobs"
CREATE INDEX "idx_backfill_jobs_created_by" ON "backfill_jobs" ("created_by");
-- Create index "idx_backfill_jobs_finished_at" to table: "backfill_jobs"
CREATE INDEX "idx_backfill_jobs_finished_at" ON "backfill_jobs" ("finished_at" DESC NULLS LAST);
-- Create index "idx_backfill_jobs_pipeline_run_id" to table: "backfill_jobs"
CREATE INDEX "idx_backfill_jobs_pipeline_run_id" ON "backfill_jobs" ("pipeline_run_id") WHERE (pipeline_run_id IS NOT NULL);
-- Create index "idx_backfill_jobs_status" to table: "backfill_jobs"
CREATE INDEX "idx_backfill_jobs_status" ON "backfill_jobs" ("status");
-- Create "customers" table
CREATE TABLE "customers" (
  "customer_id" text NOT NULL,
  "display_name" text NOT NULL,
  "legal_name" text NULL,
  "status" text NOT NULL DEFAULT 'active',
  "region" text NULL,
  "sla_tier" text NOT NULL DEFAULT 'standard',
  "account_owner" text NULL,
  "compliance_tags" jsonb NOT NULL DEFAULT '[]',
  "exclude_tags" jsonb NOT NULL DEFAULT '[]',
  "metadata" jsonb NOT NULL DEFAULT '{}',
  "extra" jsonb NOT NULL DEFAULT '{}',
  "onboarded_at" timestamptz NULL,
  "offboarded_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  "row_version" bigint NOT NULL DEFAULT 1,
  PRIMARY KEY ("customer_id"),
  CONSTRAINT "customers_sla_tier_check" CHECK (sla_tier = ANY (ARRAY['standard'::text, 'premium'::text, 'enterprise'::text])),
  CONSTRAINT "customers_status_check" CHECK (status = ANY (ARRAY['active'::text, 'trial'::text, 'suspended'::text, 'offboarded'::text]))
);
-- Create index "idx_customers_account_owner" to table: "customers"
CREATE INDEX "idx_customers_account_owner" ON "customers" ("account_owner") WHERE (account_owner IS NOT NULL);
-- Create index "idx_customers_region" to table: "customers"
CREATE INDEX "idx_customers_region" ON "customers" ("region") WHERE (region IS NOT NULL);
-- Create index "idx_customers_sla" to table: "customers"
CREATE INDEX "idx_customers_sla" ON "customers" ("sla_tier");
-- Create index "idx_customers_status" to table: "customers"
CREATE INDEX "idx_customers_status" ON "customers" ("status");
-- Create "deliveries" table
CREATE TABLE "deliveries" (
  "delivery_id" uuid NOT NULL,
  "customer_id" text NOT NULL,
  "status" character varying(16) NOT NULL DEFAULT 'pending',
  "delivered_at" timestamptz NULL,
  "is_deleted" boolean NULL DEFAULT false,
  "contract_id" text NULL,
  "delivery_type" text NOT NULL DEFAULT 'asset_set',
  "requested_by" text NULL,
  "approved_by" text NULL,
  "delivered_by" text NULL,
  "manifest_uri" text NULL,
  "replay_manifest_uri" text NULL,
  "item_count" bigint NOT NULL DEFAULT 0,
  "total_size_bytes" bigint NULL,
  "completed_at" timestamptz NULL,
  "metadata" jsonb NOT NULL DEFAULT '{}',
  "tenant_id" text NULL,
  "project_id" text NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "version" bigint NULL DEFAULT 1,
  PRIMARY KEY ("delivery_id")
);
-- Create index "idx_deliveries_customer_created" to table: "deliveries"
CREATE INDEX "idx_deliveries_customer_created" ON "deliveries" ("customer_id", "created_at" DESC);
-- Create index "idx_deliveries_tenant_project" to table: "deliveries"
CREATE INDEX "idx_deliveries_tenant_project" ON "deliveries" ("tenant_id", "project_id") WHERE (is_deleted = false);
-- Create "delivery_items" table
CREATE TABLE "delivery_items" (
  "delivery_id" uuid NOT NULL,
  "asset_id" text NOT NULL,
  "created_at" timestamptz NULL DEFAULT now(),
  PRIMARY KEY ("delivery_id", "asset_id")
);
-- Create index "idx_delivery_items_asset_id" to table: "delivery_items"
CREATE INDEX "idx_delivery_items_asset_id" ON "delivery_items" ("asset_id");
-- Create "delivery_rules" table
CREATE TABLE "delivery_rules" (
  "rule_id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "name" text NOT NULL,
  "owner" text NOT NULL,
  "customer_id" text NULL,
  "query_dsl" jsonb NOT NULL,
  "dsl_version" text NOT NULL DEFAULT 'v1',
  "enforce_mode" text NOT NULL DEFAULT 'block',
  "rating_scope" text NOT NULL DEFAULT 'current',
  "is_active" boolean NOT NULL DEFAULT true,
  "version" bigint NOT NULL DEFAULT 1,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("rule_id"),
  CONSTRAINT "delivery_rules_enforce_mode_check" CHECK (enforce_mode = ANY (ARRAY['block'::text, 'warn'::text, 'tag_only'::text])),
  CONSTRAINT "delivery_rules_rating_scope_check" CHECK (rating_scope = ANY (ARRAY['current'::text, 'logical'::text]))
);
-- Create index "idx_drules_active" to table: "delivery_rules"
CREATE INDEX "idx_drules_active" ON "delivery_rules" ("is_active") WHERE (is_active = true);
-- Create index "idx_drules_customer" to table: "delivery_rules"
CREATE INDEX "idx_drules_customer" ON "delivery_rules" ("customer_id", "is_active");
-- Create "es_sync_checkpoint" table
CREATE TABLE "es_sync_checkpoint" (
  "shard_id" integer NOT NULL,
  "applied_seq" bigint NOT NULL,
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("shard_id")
);
-- Set comment to table: "es_sync_checkpoint"
COMMENT ON TABLE "es_sync_checkpoint" IS 'ES subscriber per-shard high-water mark of applied event_seq (see consumer_lag in /search/sync-progress).';
-- Set comment to column: "shard_id" on table: "es_sync_checkpoint"
COMMENT ON COLUMN "es_sync_checkpoint"."shard_id" IS 'FNV(routing_key) % OUTBOX_ES_CHECKPOINT_SHARDS; stable for the lifetime of the deployment.';
-- Set comment to column: "applied_seq" on table: "es_sync_checkpoint"
COMMENT ON COLUMN "es_sync_checkpoint"."applied_seq" IS 'GREATEST(applied_seq, batch_max_seq) — only advances on successful ES bulk.';
-- Create "execution_targets" table
CREATE TABLE "execution_targets" (
  "id" text NOT NULL,
  "name" text NOT NULL,
  "description" text NOT NULL DEFAULT '',
  "cluster" text NOT NULL,
  "namespace" text NOT NULL,
  "service_account" text NOT NULL DEFAULT '',
  "argo_server_url" text NOT NULL DEFAULT '',
  "argo_auth_secret_ref" text NOT NULL DEFAULT '',
  "argo_insecure_skip_verify" boolean NOT NULL DEFAULT false,
  "argo_ca_cert_ref" text NOT NULL DEFAULT '',
  "enabled" boolean NOT NULL DEFAULT true,
  "status" text NOT NULL DEFAULT 'available',
  "is_default" boolean NOT NULL DEFAULT false,
  "resource_defaults" jsonb NOT NULL DEFAULT '{}',
  "quota_policy" jsonb NOT NULL DEFAULT '{}',
  "labels" jsonb NOT NULL DEFAULT '{}',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  "image_pull_secrets" text[] NOT NULL DEFAULT '{}',
  PRIMARY KEY ("id")
);
-- Create index "idx_execution_targets_one_default" to table: "execution_targets"
CREATE UNIQUE INDEX "idx_execution_targets_one_default" ON "execution_targets" ("is_default") WHERE is_default;
-- Create "idempotency_keys" table
CREATE TABLE "idempotency_keys" (
  "scope" character varying(64) NOT NULL,
  "idem_key" character varying(128) NOT NULL,
  "request_hash" character varying(64) NULL,
  "status_code" integer NULL,
  "response_json" jsonb NULL,
  "created_at" timestamptz NULL,
  PRIMARY KEY ("scope", "idem_key")
);
-- Create "lakehouse_bronze_checkpoint" table
CREATE TABLE "lakehouse_bronze_checkpoint" (
  "id" integer NOT NULL,
  "applied_seq" bigint NOT NULL,
  "ingested_at" timestamptz NOT NULL,
  "run_id" text NULL,
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "lakehouse_bronze_checkpoint_singleton" CHECK (id = 1)
);
-- Set comment to table: "lakehouse_bronze_checkpoint"
COMMENT ON TABLE "lakehouse_bronze_checkpoint" IS 'PG→Iceberg Bronze high-water mark; one row (id=1) updated by Cloud Run Job bronze-incremental after each successful run.';
-- Set comment to column: "applied_seq" on table: "lakehouse_bronze_checkpoint"
COMMENT ON COLUMN "lakehouse_bronze_checkpoint"."applied_seq" IS 'MAX(event_seq) the job confirmed committed to Bronze on its latest run.';
-- Set comment to column: "ingested_at" on table: "lakehouse_bronze_checkpoint"
COMMENT ON COLUMN "lakehouse_bronze_checkpoint"."ingested_at" IS 'Wall-clock time of the job that produced applied_seq (matches Bronze _ingested_at for that batch).';
-- Set comment to column: "run_id" on table: "lakehouse_bronze_checkpoint"
COMMENT ON COLUMN "lakehouse_bronze_checkpoint"."run_id" IS 'Cloud Run Job execution name (e.g. bronze-incremental-jvzkr); useful for cross-referencing logs.';
-- Create "logical_assets" table
CREATE TABLE "logical_assets" (
  "logical_asset_id" text NOT NULL,
  "asset_type" text NOT NULL,
  "display_name" text NULL,
  "description" text NULL,
  "owner" text NULL,
  "status" text NOT NULL DEFAULT 'active',
  "current_revision" bigint NOT NULL DEFAULT 1,
  "total_revisions" bigint NOT NULL DEFAULT 1,
  "metadata" jsonb NOT NULL DEFAULT '{}',
  "extra" jsonb NOT NULL DEFAULT '{}',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  "row_version" bigint NOT NULL DEFAULT 1,
  PRIMARY KEY ("logical_asset_id"),
  CONSTRAINT "logical_assets_check" CHECK (total_revisions >= current_revision),
  CONSTRAINT "logical_assets_current_revision_check" CHECK (current_revision >= 1),
  CONSTRAINT "logical_assets_logical_asset_id_check" CHECK ((logical_asset_id ~ '^[0-9A-Za-z]{8}$'::text) OR (logical_asset_id ~ '^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$'::text)),
  CONSTRAINT "logical_assets_status_check" CHECK (status = ANY (ARRAY['active'::text, 'archived'::text]))
);
-- Create index "idx_lassets_owner" to table: "logical_assets"
CREATE INDEX "idx_lassets_owner" ON "logical_assets" ("owner") WHERE (owner IS NOT NULL);
-- Create index "idx_lassets_status" to table: "logical_assets"
CREATE INDEX "idx_lassets_status" ON "logical_assets" ("status") WHERE (status <> 'active'::text);
-- Create index "idx_lassets_type" to table: "logical_assets"
CREATE INDEX "idx_lassets_type" ON "logical_assets" ("asset_type", "status");
-- Create index "idx_lassets_updated" to table: "logical_assets"
CREATE INDEX "idx_lassets_updated" ON "logical_assets" ("updated_at" DESC);
-- Create "trg_logical_assets_type_immutable" function
CREATE FUNCTION "trg_logical_assets_type_immutable" () RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF NEW.asset_type IS DISTINCT FROM OLD.asset_type THEN
    RAISE EXCEPTION 'logical_assets.asset_type is immutable (LA1)';
  END IF;
  RETURN NEW;
END;
$$;
-- Create trigger "logical_assets_type_immutable"
CREATE TRIGGER "logical_assets_type_immutable" BEFORE UPDATE ON "logical_assets" FOR EACH ROW EXECUTE FUNCTION "trg_logical_assets_type_immutable"();
-- Create "node_attempts" table
CREATE TABLE "node_attempts" (
  "id" text NOT NULL,
  "node_run_id" text NOT NULL,
  "attempt_no" integer NOT NULL,
  "argo_node_id" text NOT NULL DEFAULT '',
  "pod_name" text NOT NULL DEFAULT '',
  "container_name" text NOT NULL DEFAULT '',
  "phase" text NOT NULL DEFAULT '',
  "exit_code" integer NULL,
  "reason" text NOT NULL DEFAULT '',
  "message" text NOT NULL DEFAULT '',
  "log_uri" text NOT NULL DEFAULT '',
  "started_at" timestamptz NULL,
  "finished_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "node_attempts_attempt_no_positive" CHECK (attempt_no > 0)
);
-- Create index "idx_node_attempts_pod" to table: "node_attempts"
CREATE INDEX "idx_node_attempts_pod" ON "node_attempts" ("pod_name") WHERE (pod_name <> ''::text);
-- Create index "idx_node_attempts_unique" to table: "node_attempts"
CREATE UNIQUE INDEX "idx_node_attempts_unique" ON "node_attempts" ("node_run_id", "attempt_no");
-- Create "node_runs" table
CREATE TABLE "node_runs" (
  "id" text NOT NULL,
  "pipeline_run_id" text NOT NULL,
  "shard_id" text NULL,
  "asset_id" text NOT NULL,
  "node_key" text NOT NULL,
  "node_name" text NOT NULL DEFAULT '',
  "component_id" text NOT NULL DEFAULT '',
  "component_version" text NOT NULL DEFAULT '',
  "image" text NOT NULL DEFAULT '',
  "image_digest" text NOT NULL DEFAULT '',
  "config_hash" text NOT NULL DEFAULT '',
  "input_hash" text NOT NULL DEFAULT '',
  "upstream_output_hash" text NOT NULL DEFAULT '',
  "interface_hash" text NOT NULL DEFAULT '',
  "cache_key" text NOT NULL DEFAULT '',
  "status" text NOT NULL DEFAULT 'pending',
  "quality_status" text NOT NULL DEFAULT 'unknown',
  "taint_status" text NOT NULL DEFAULT 'clean',
  "cache_hit" boolean NOT NULL DEFAULT false,
  "output_uri" text NOT NULL DEFAULT '',
  "output_hash" text NOT NULL DEFAULT '',
  "output_asset_id" text NOT NULL DEFAULT '',
  "superseded_by_node_run_id" text NULL,
  "error_class" text NOT NULL DEFAULT '',
  "error_message" text NOT NULL DEFAULT '',
  "started_at" timestamptz NULL,
  "finished_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "node_runs_superseded_by_node_run_id_fkey" FOREIGN KEY ("superseded_by_node_run_id") REFERENCES "node_runs" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "node_runs_asset_id_not_blank" CHECK (btrim(asset_id) <> ''::text),
  CONSTRAINT "node_runs_node_key_not_blank" CHECK (btrim(node_key) <> ''::text),
  CONSTRAINT "node_runs_quality_status_contract" CHECK (lower(quality_status) = ANY (ARRAY['unknown'::text, 'pending'::text, 'passed'::text, 'failed'::text, 'quality_failed'::text, 'warning'::text])),
  CONSTRAINT "node_runs_status_contract" CHECK (lower(status) = ANY (ARRAY['pending'::text, 'queued'::text, 'created'::text, 'planning'::text, 'submitting'::text, 'submitted'::text, 'running'::text, 'succeeded'::text, 'success'::text, 'completed'::text, 'failed'::text, 'error'::text, 'skipped'::text, 'omitted'::text, 'cached'::text])),
  CONSTRAINT "node_runs_taint_status_contract" CHECK (lower(taint_status) = ANY (ARRAY['clean'::text, 'none'::text, 'unknown'::text, 'tainted'::text, 'stale'::text, 'dirty'::text]))
);
-- Create index "idx_node_runs_asset_updated_run" to table: "node_runs"
CREATE INDEX "idx_node_runs_asset_updated_run" ON "node_runs" ("asset_id", "updated_at" DESC, "pipeline_run_id");
-- Create index "idx_node_runs_cache_key" to table: "node_runs"
CREATE INDEX "idx_node_runs_cache_key" ON "node_runs" ("cache_key") WHERE (cache_key <> ''::text);
-- Create index "idx_node_runs_run_node_status_asset" to table: "node_runs"
CREATE INDEX "idx_node_runs_run_node_status_asset" ON "node_runs" ("pipeline_run_id", "node_key", "status", "asset_id");
-- Create index "idx_node_runs_run_shard_status" to table: "node_runs"
CREATE INDEX "idx_node_runs_run_shard_status" ON "node_runs" ("pipeline_run_id", "shard_id", "status");
-- Create index "idx_node_runs_unique" to table: "node_runs"
CREATE UNIQUE INDEX "idx_node_runs_unique" ON "node_runs" ("pipeline_run_id", "asset_id", "node_key");
-- Create "outbox_dlq" table
CREATE TABLE "outbox_dlq" (
  "dlq_id" bigserial NOT NULL,
  "event_id" uuid NOT NULL,
  "event_seq" bigint NOT NULL,
  "event_type" text NOT NULL,
  "aggregate_type" text NOT NULL DEFAULT 'asset',
  "asset_id" text NULL,
  "mcap_file_id" text NULL,
  "event_source" text NOT NULL DEFAULT 'backend',
  "event_payload" jsonb NOT NULL DEFAULT '{}',
  "retry_count" integer NOT NULL DEFAULT 0,
  "last_error" text NULL,
  "original_created_at" timestamptz NOT NULL,
  "moved_at" timestamptz NOT NULL DEFAULT now(),
  "resolved_at" timestamptz NULL,
  "resolution" text NULL,
  PRIMARY KEY ("dlq_id")
);
-- Create index "idx_outbox_dlq_asset" to table: "outbox_dlq"
CREATE INDEX "idx_outbox_dlq_asset" ON "outbox_dlq" ("asset_id") WHERE (resolved_at IS NULL);
-- Create index "idx_outbox_dlq_unresolved" to table: "outbox_dlq"
CREATE INDEX "idx_outbox_dlq_unresolved" ON "outbox_dlq" ("moved_at" DESC) WHERE (resolved_at IS NULL);
-- Create "pipeline_component_releases" table
CREATE TABLE "pipeline_component_releases" (
  "id" text NOT NULL,
  "component_id" text NOT NULL,
  "task_name" text NOT NULL,
  "task_path" text NOT NULL DEFAULT '',
  "display_name" text NOT NULL DEFAULT '',
  "owner" text NOT NULL DEFAULT '',
  "release_label" text NOT NULL,
  "channel" text NOT NULL DEFAULT 'dev',
  "source_repo" text NOT NULL DEFAULT '',
  "source_ref" text NOT NULL DEFAULT '',
  "source_commit" text NOT NULL DEFAULT '',
  "build_id" text NOT NULL DEFAULT '',
  "image_repo" text NOT NULL DEFAULT '',
  "image_tag" text NOT NULL DEFAULT '',
  "image_digest" text NOT NULL DEFAULT '',
  "runtime_image" text NOT NULL DEFAULT '',
  "status" text NOT NULL DEFAULT 'failed',
  "selectable" boolean NOT NULL DEFAULT false,
  "validation_status" text NOT NULL DEFAULT 'failed',
  "validation_errors" jsonb NOT NULL DEFAULT '[]',
  "runtime_snapshot" jsonb NOT NULL DEFAULT '{}',
  "technical_metadata" jsonb NOT NULL DEFAULT '{}',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  "last_synced_at" timestamptz NULL,
  PRIMARY KEY ("id")
);
-- Create index "idx_pipeline_component_releases_component_label" to table: "pipeline_component_releases"
CREATE UNIQUE INDEX "idx_pipeline_component_releases_component_label" ON "pipeline_component_releases" ("component_id", "release_label");
-- Create index "idx_pipeline_component_releases_selectable" to table: "pipeline_component_releases"
CREATE INDEX "idx_pipeline_component_releases_selectable" ON "pipeline_component_releases" ("selectable");
-- Create index "idx_pipeline_component_releases_source_commit" to table: "pipeline_component_releases"
CREATE INDEX "idx_pipeline_component_releases_source_commit" ON "pipeline_component_releases" ("source_commit");
-- Create index "idx_pipeline_component_releases_status" to table: "pipeline_component_releases"
CREATE INDEX "idx_pipeline_component_releases_status" ON "pipeline_component_releases" ("status");
-- Create index "idx_pipeline_component_releases_task_name" to table: "pipeline_component_releases"
CREATE INDEX "idx_pipeline_component_releases_task_name" ON "pipeline_component_releases" ("task_name");
-- Create "pipeline_components" table
CREATE TABLE "pipeline_components" (
  "id" text NOT NULL,
  "name" text NOT NULL,
  "description" text NOT NULL DEFAULT '',
  "image" text NOT NULL,
  "tag" text NOT NULL DEFAULT 'latest',
  "source" text NOT NULL DEFAULT 'custom',
  "input_ports" jsonb NOT NULL DEFAULT '[]',
  "output_ports" jsonb NOT NULL DEFAULT '[]',
  "resources" jsonb NULL,
  "env_vars" jsonb NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  "scope" character varying(16) NOT NULL DEFAULT 'dev',
  "owner" character varying(255) NOT NULL DEFAULT '',
  PRIMARY KEY ("id")
);
-- Create index "idx_pipeline_components_name" to table: "pipeline_components"
CREATE INDEX "idx_pipeline_components_name" ON "pipeline_components" ("name");
-- Create index "idx_pipeline_components_source" to table: "pipeline_components"
CREATE INDEX "idx_pipeline_components_source" ON "pipeline_components" ("source");
-- Create "pipeline_config_versions" table
CREATE TABLE "pipeline_config_versions" (
  "id" text NOT NULL,
  "config_id" text NOT NULL,
  "version" integer NOT NULL,
  "status" text NOT NULL DEFAULT 'draft',
  "content" text NOT NULL,
  "content_sha256" text NOT NULL,
  "content_size_bytes" integer NOT NULL,
  "summary" text NOT NULL DEFAULT '',
  "author" text NOT NULL DEFAULT '',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "uq_pipeline_config_versions_config_version" UNIQUE ("config_id", "version"),
  CONSTRAINT "chk_pipeline_config_versions_size" CHECK ((content_size_bytes >= 0) AND (content_size_bytes <= 1048576)),
  CONSTRAINT "chk_pipeline_config_versions_status" CHECK (status = ANY (ARRAY['draft'::text, 'ready'::text, 'deprecated'::text]))
);
-- Create index "idx_pipeline_config_versions_config_id" to table: "pipeline_config_versions"
CREATE INDEX "idx_pipeline_config_versions_config_id" ON "pipeline_config_versions" ("config_id", "version" DESC);
-- Create "pipeline_configs" table
CREATE TABLE "pipeline_configs" (
  "id" text NOT NULL,
  "name" text NOT NULL,
  "description" text NOT NULL DEFAULT '',
  "owner" text NOT NULL DEFAULT '',
  "scope" character varying(16) NOT NULL DEFAULT 'dev',
  "tags" jsonb NOT NULL DEFAULT '[]',
  "file_type" text NOT NULL,
  "lifecycle" text NOT NULL DEFAULT 'draft',
  "current_version" integer NOT NULL DEFAULT 1,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "chk_pipeline_configs_file_type" CHECK (file_type = ANY (ARRAY['yaml'::text, 'json'::text])),
  CONSTRAINT "chk_pipeline_configs_lifecycle" CHECK (lifecycle = ANY (ARRAY['draft'::text, 'ready'::text, 'deprecated'::text]))
);
-- Create index "idx_pipeline_configs_lifecycle" to table: "pipeline_configs"
CREATE INDEX "idx_pipeline_configs_lifecycle" ON "pipeline_configs" ("lifecycle");
-- Create index "idx_pipeline_configs_owner_scope" to table: "pipeline_configs"
CREATE INDEX "idx_pipeline_configs_owner_scope" ON "pipeline_configs" ("owner", "scope");
-- Create "pipeline_definitions" table
CREATE TABLE "pipeline_definitions" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "tenant_id" text NOT NULL,
  "project_id" text NULL,
  "name" text NOT NULL,
  "description" text NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "uq_pipeline_def_name" UNIQUE ("tenant_id", "name")
);
-- Set comment to table: "pipeline_definitions"
COMMENT ON TABLE "pipeline_definitions" IS 'Pipeline definitions — top-level container for versioned DAG revisions.';
-- Create trigger "trg_pipeline_definitions_updated_at"
CREATE TRIGGER "trg_pipeline_definitions_updated_at" BEFORE UPDATE ON "pipeline_definitions" FOR EACH ROW EXECUTE FUNCTION "set_updated_at"();
-- Create "pipeline_deployments" table
CREATE TABLE "pipeline_deployments" (
  "id" text NOT NULL,
  "template_id" text NULL,
  "pipeline_name" text NOT NULL,
  "workflow_name" text NOT NULL,
  "status" text NOT NULL DEFAULT 'Pending',
  "node_count" integer NOT NULL DEFAULT 0,
  "manifest" text NULL,
  "pipeline_json" jsonb NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  "finished_at" timestamptz NULL,
  "scope" character varying(16) NOT NULL DEFAULT 'dev',
  "owner" character varying(255) NOT NULL DEFAULT '',
  "batch_run_id" uuid NULL,
  "trigger_source" text NOT NULL DEFAULT 'manual',
  "trigger_context" jsonb NOT NULL DEFAULT '{}',
  PRIMARY KEY ("id")
);
-- Create index "idx_pipeline_deployments_batch_run_id" to table: "pipeline_deployments"
CREATE INDEX "idx_pipeline_deployments_batch_run_id" ON "pipeline_deployments" ("batch_run_id") WHERE (batch_run_id IS NOT NULL);
-- Create "pipeline_revisions" table
CREATE TABLE "pipeline_revisions" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "definition_id" uuid NOT NULL,
  "version" integer NOT NULL,
  "status" text NOT NULL DEFAULT 'draft',
  "dag" jsonb NOT NULL,
  "argo_wft_name" text NULL,
  "publish_error" text NULL,
  "published_at" timestamptz NULL,
  "published_by" text NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "created_by" text NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "uq_pipeline_rev_def_ver" UNIQUE ("definition_id", "version"),
  CONSTRAINT "chk_pipeline_rev_status" CHECK (status = ANY (ARRAY['draft'::text, 'published'::text, 'archived'::text]))
);
-- Create index "idx_pipeline_rev_def_status" to table: "pipeline_revisions"
CREATE INDEX "idx_pipeline_rev_def_status" ON "pipeline_revisions" ("definition_id", "status");
-- Set comment to table: "pipeline_revisions"
COMMENT ON TABLE "pipeline_revisions" IS 'Pipeline revisions — versioned DAG snapshots with status machine. openspec/changes/CYB-1237-pipeline-revisions/.';
-- Create "pipeline_run_asset_nodes" table
CREATE TABLE "pipeline_run_asset_nodes" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "run_id" text NOT NULL,
  "asset_id" text NOT NULL,
  "pipeline_node_id" text NOT NULL,
  "argo_node_id" text NULL,
  "display_name" text NULL,
  "status" text NULL,
  "message" text NULL,
  "pod_name" text NULL,
  "log_ref" text NULL,
  "estimated_cost_usd" numeric(16,8) NULL,
  "cost_source" text NOT NULL DEFAULT 'not_available',
  "started_at" timestamptz NULL,
  "finished_at" timestamptz NULL,
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "pipeline_run_asset_nodes_asset_id_not_blank" CHECK (btrim(asset_id) <> ''::text),
  CONSTRAINT "pipeline_run_asset_nodes_node_id_not_blank" CHECK (btrim(pipeline_node_id) <> ''::text)
);
-- Create index "idx_pipeline_run_asset_nodes_run_status" to table: "pipeline_run_asset_nodes"
CREATE INDEX "idx_pipeline_run_asset_nodes_run_status" ON "pipeline_run_asset_nodes" ("run_id", "status");
-- Create index "idx_pipeline_run_asset_nodes_unique" to table: "pipeline_run_asset_nodes"
CREATE UNIQUE INDEX "idx_pipeline_run_asset_nodes_unique" ON "pipeline_run_asset_nodes" ("run_id", "asset_id", "pipeline_node_id");
-- Create "pipeline_run_events" table
CREATE TABLE "pipeline_run_events" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "run_id" text NOT NULL,
  "workflow_name" text NULL,
  "event_type" text NOT NULL,
  "subject_type" text NOT NULL,
  "subject_id" text NOT NULL,
  "status" text NULL,
  "message" text NULL,
  "reason" text NULL,
  "payload" jsonb NOT NULL DEFAULT '{}',
  "idempotency_key" text NOT NULL,
  "sequence" bigserial NOT NULL,
  "occurred_at" timestamptz NOT NULL DEFAULT now(),
  "observed_at" timestamptz NOT NULL DEFAULT now(),
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "pipeline_run_events_event_type_not_blank" CHECK (btrim(event_type) <> ''::text),
  CONSTRAINT "pipeline_run_events_idempotency_key_not_blank" CHECK (btrim(idempotency_key) <> ''::text),
  CONSTRAINT "pipeline_run_events_subject_id_not_blank" CHECK (btrim(subject_id) <> ''::text),
  CONSTRAINT "pipeline_run_events_subject_type_not_blank" CHECK (btrim(subject_type) <> ''::text)
);
-- Create index "idx_pipeline_run_events_run_filters" to table: "pipeline_run_events"
CREATE INDEX "idx_pipeline_run_events_run_filters" ON "pipeline_run_events" ("run_id", "event_type", "subject_type", "status", "occurred_at");
-- Create index "idx_pipeline_run_events_run_id_idempotency" to table: "pipeline_run_events"
CREATE UNIQUE INDEX "idx_pipeline_run_events_run_id_idempotency" ON "pipeline_run_events" ("run_id", "idempotency_key");
-- Create index "idx_pipeline_run_events_run_timeline" to table: "pipeline_run_events"
CREATE INDEX "idx_pipeline_run_events_run_timeline" ON "pipeline_run_events" ("run_id", "sequence");
-- Create index "idx_pipeline_run_events_workflow_time" to table: "pipeline_run_events"
CREATE INDEX "idx_pipeline_run_events_workflow_time" ON "pipeline_run_events" ("workflow_name", "occurred_at");
-- Create "pipeline_run_nodes" table
CREATE TABLE "pipeline_run_nodes" (
  "id" text NOT NULL,
  "run_id" text NOT NULL,
  "pipeline_node_id" text NOT NULL,
  "argo_node_id" text NOT NULL DEFAULT '',
  "argo_node_name" text NOT NULL DEFAULT '',
  "display_name" text NOT NULL DEFAULT '',
  "template_name" text NOT NULL DEFAULT '',
  "type" text NOT NULL DEFAULT '',
  "phase" text NOT NULL DEFAULT '',
  "message" text NOT NULL DEFAULT '',
  "pod_name" text NOT NULL DEFAULT '',
  "host_node_name" text NOT NULL DEFAULT '',
  "children" text[] NOT NULL DEFAULT '{}',
  "inputs" jsonb NOT NULL DEFAULT '{}',
  "outputs" jsonb NOT NULL DEFAULT '{}',
  "resources_duration" jsonb NOT NULL DEFAULT '{}',
  "resource_summary" jsonb NOT NULL DEFAULT '{}',
  "log_ref" text NOT NULL DEFAULT '',
  "started_at" timestamptz NULL,
  "finished_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  "estimated_cost_usd" numeric(16,8) NULL,
  PRIMARY KEY ("id")
);
-- Create index "idx_pipeline_run_nodes_run_argo_node" to table: "pipeline_run_nodes"
CREATE UNIQUE INDEX "idx_pipeline_run_nodes_run_argo_node" ON "pipeline_run_nodes" ("run_id", "argo_node_id") WHERE (argo_node_id <> ''::text);
-- Create index "idx_pipeline_run_nodes_run_phase" to table: "pipeline_run_nodes"
CREATE INDEX "idx_pipeline_run_nodes_run_phase" ON "pipeline_run_nodes" ("run_id", "phase");
-- Create "pipeline_run_notification_candidates" table
CREATE TABLE "pipeline_run_notification_candidates" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "run_id" text NOT NULL,
  "event_id" text NOT NULL,
  "event_type" text NOT NULL,
  "subject_type" text NOT NULL,
  "subject_id" text NOT NULL,
  "status" text NULL,
  "message" text NULL,
  "sink_type" text NOT NULL DEFAULT 'candidate',
  "delivery_status" text NOT NULL DEFAULT 'pending',
  "idempotency_key" text NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "pipeline_run_notification_candidates_key_not_blank" CHECK (btrim(idempotency_key) <> ''::text)
);
-- Create index "idx_pipeline_run_notification_candidates_key" to table: "pipeline_run_notification_candidates"
CREATE UNIQUE INDEX "idx_pipeline_run_notification_candidates_key" ON "pipeline_run_notification_candidates" ("idempotency_key");
-- Create index "idx_pipeline_run_notification_candidates_run" to table: "pipeline_run_notification_candidates"
CREATE INDEX "idx_pipeline_run_notification_candidates_run" ON "pipeline_run_notification_candidates" ("run_id", "created_at");
-- Create "pipeline_run_watcher_state" table
CREATE TABLE "pipeline_run_watcher_state" (
  "id" text NOT NULL,
  "last_synced_at" timestamptz NULL,
  "active_scan_limit" integer NOT NULL DEFAULT 100,
  "last_error" text NULL,
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  "last_scan_started_at" timestamptz NULL,
  "last_scan_finished_at" timestamptz NULL,
  "last_success_at" timestamptz NULL,
  "last_error_at" timestamptz NULL,
  "last_synced_run_count" integer NOT NULL DEFAULT 0,
  "consecutive_failures" integer NOT NULL DEFAULT 0,
  "total_scans" bigint NOT NULL DEFAULT 0,
  "total_errors" bigint NOT NULL DEFAULT 0,
  "scan_lag_seconds" bigint NULL,
  PRIMARY KEY ("id")
);
-- Create "pipeline_runs" table
CREATE TABLE "pipeline_runs" (
  "id" text NOT NULL,
  "template_id" text NULL,
  "pipeline_name" text NOT NULL,
  "template_version" integer NULL,
  "workflow_name" text NOT NULL,
  "execution_target_id" text NOT NULL,
  "target_snapshot" jsonb NOT NULL DEFAULT '{}',
  "status" text NOT NULL DEFAULT 'Pending',
  "node_count" integer NOT NULL DEFAULT 0,
  "asset_ids" text[] NOT NULL DEFAULT '{}',
  "asset_count" integer NOT NULL DEFAULT 0,
  "no_asset_run" boolean NOT NULL DEFAULT false,
  "manifest" text NULL,
  "pipeline_json" jsonb NOT NULL DEFAULT '{}',
  "argo_namespace" text NOT NULL,
  "argo_workflow_uid" text NOT NULL DEFAULT '',
  "message" text NOT NULL DEFAULT '',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  "started_at" timestamptz NULL,
  "finished_at" timestamptz NULL,
  "admission_status" text NOT NULL DEFAULT 'admitted',
  "queue_reason" text NOT NULL DEFAULT '',
  "priority" text NOT NULL DEFAULT 'normal',
  "resource_overrides" jsonb NOT NULL DEFAULT '{}',
  "admission_snapshot" jsonb NOT NULL DEFAULT '{}',
  "estimated_cost_usd" numeric(16,8) NULL,
  "scope" character varying(16) NOT NULL DEFAULT 'dev',
  "owner" character varying(255) NOT NULL DEFAULT '',
  "ledger_state" text NOT NULL DEFAULT 'pending',
  "batch_run_id" uuid NULL,
  "parameters" jsonb NOT NULL DEFAULT '{}',
  "trigger_source" text NOT NULL DEFAULT 'manual',
  "trigger_context" jsonb NOT NULL DEFAULT '{}',
  "input_filter" jsonb NOT NULL DEFAULT '{}',
  "resource_pool" text NOT NULL DEFAULT '',
  "queue_name" text NOT NULL DEFAULT '',
  "runtime_backend" text NOT NULL DEFAULT 'argo',
  "total_shards" integer NOT NULL DEFAULT 0,
  "submitted_shards" integer NOT NULL DEFAULT 0,
  "running_shards" integer NOT NULL DEFAULT 0,
  "completed_shards" integer NOT NULL DEFAULT 0,
  "failed_shards" integer NOT NULL DEFAULT 0,
  "total_nodes" bigint NOT NULL DEFAULT 0,
  "completed_nodes" bigint NOT NULL DEFAULT 0,
  "failed_nodes" bigint NOT NULL DEFAULT 0,
  "cached_nodes" bigint NOT NULL DEFAULT 0,
  "tainted_nodes" bigint NOT NULL DEFAULT 0,
  "parent_run_id" text NULL,
  "repair_reason" text NOT NULL DEFAULT '',
  "batch_job_id" text NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "pipeline_runs_workflow_name_key" UNIQUE ("workflow_name"),
  CONSTRAINT "pipeline_runs_parent_run_id_fkey" FOREIGN KEY ("parent_run_id") REFERENCES "pipeline_runs" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "pipeline_runs_priority_contract" CHECK (lower(priority) = ANY (ARRAY['low'::text, 'normal'::text, 'high'::text, 'urgent'::text])),
  CONSTRAINT "pipeline_runs_runtime_backend_contract" CHECK (lower(runtime_backend) = 'argo'::text)
);
-- Create index "idx_pipeline_runs_admission_status_created_at" to table: "pipeline_runs"
CREATE INDEX "idx_pipeline_runs_admission_status_created_at" ON "pipeline_runs" ("admission_status", "created_at");
-- Create index "idx_pipeline_runs_asset_ids" to table: "pipeline_runs"
CREATE INDEX "idx_pipeline_runs_asset_ids" ON "pipeline_runs" USING GIN ("asset_ids");
-- Create index "idx_pipeline_runs_batch_job_id" to table: "pipeline_runs"
CREATE INDEX "idx_pipeline_runs_batch_job_id" ON "pipeline_runs" ("batch_job_id") WHERE (batch_job_id IS NOT NULL);
-- Create index "idx_pipeline_runs_batch_run_id" to table: "pipeline_runs"
CREATE INDEX "idx_pipeline_runs_batch_run_id" ON "pipeline_runs" ("batch_run_id") WHERE (batch_run_id IS NOT NULL);
-- Create index "idx_pipeline_runs_created_at" to table: "pipeline_runs"
CREATE INDEX "idx_pipeline_runs_created_at" ON "pipeline_runs" ("created_at" DESC);
-- Create index "idx_pipeline_runs_ledger_state" to table: "pipeline_runs"
CREATE INDEX "idx_pipeline_runs_ledger_state" ON "pipeline_runs" ("ledger_state") WHERE (ledger_state <> 'has_ledger'::text);
-- Create index "idx_pipeline_runs_parent_run_id" to table: "pipeline_runs"
CREATE INDEX "idx_pipeline_runs_parent_run_id" ON "pipeline_runs" ("parent_run_id") WHERE (parent_run_id IS NOT NULL);
-- Create index "idx_pipeline_runs_priority_created_at" to table: "pipeline_runs"
CREATE INDEX "idx_pipeline_runs_priority_created_at" ON "pipeline_runs" ("priority", "created_at");
-- Create index "idx_pipeline_runs_status_updated_at" to table: "pipeline_runs"
CREATE INDEX "idx_pipeline_runs_status_updated_at" ON "pipeline_runs" ("status", "updated_at" DESC);
-- Create index "idx_pipeline_runs_template_created_at" to table: "pipeline_runs"
CREATE INDEX "idx_pipeline_runs_template_created_at" ON "pipeline_runs" ("template_id", "created_at" DESC);
-- Create index "idx_pipeline_runs_trigger_source_created_at" to table: "pipeline_runs"
CREATE INDEX "idx_pipeline_runs_trigger_source_created_at" ON "pipeline_runs" ("trigger_source", "created_at" DESC);
-- Set comment to column: "ledger_state" on table: "pipeline_runs"
COMMENT ON COLUMN "pipeline_runs"."ledger_state" IS '
  pending    — never scanned by watcher (default for new runs)
  has_ledger — watcher confirmed events exist (COUNT(events) > 0)
  no_ledger  — watcher scanned but found no events
  backfilling— watcher actively backfilling events from Argo
';
-- Set comment to column: "parameters" on table: "pipeline_runs"
COMMENT ON COLUMN "pipeline_runs"."parameters" IS 'Resolved pipeline-level parameter snapshot for this run. Secret values are masked before persistence.';
-- Create "pipeline_shards" table
CREATE TABLE "pipeline_shards" (
  "id" text NOT NULL,
  "pipeline_run_id" text NOT NULL,
  "shard_index" integer NOT NULL,
  "asset_ids" text[] NOT NULL DEFAULT '{}',
  "asset_count" integer NOT NULL DEFAULT 0,
  "asset_filter" jsonb NOT NULL DEFAULT '{}',
  "status" text NOT NULL DEFAULT 'pending',
  "priority" text NOT NULL DEFAULT 'normal',
  "deployment_id" text NULL,
  "workflow_name" text NOT NULL DEFAULT '',
  "argo_namespace" text NOT NULL DEFAULT '',
  "queue_name" text NOT NULL DEFAULT '',
  "resource_pool" text NOT NULL DEFAULT '',
  "message" text NOT NULL DEFAULT '',
  "error_message" text NOT NULL DEFAULT '',
  "claimed_at" timestamptz NULL,
  "admitted_at" timestamptz NULL,
  "submitted_at" timestamptz NULL,
  "started_at" timestamptz NULL,
  "finished_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "pipeline_shards_asset_count_nonnegative" CHECK (asset_count >= 0),
  CONSTRAINT "pipeline_shards_priority_contract" CHECK (lower(priority) = ANY (ARRAY['low'::text, 'normal'::text, 'high'::text, 'urgent'::text])),
  CONSTRAINT "pipeline_shards_shard_index_nonnegative" CHECK (shard_index >= 0),
  CONSTRAINT "pipeline_shards_status_contract" CHECK (lower(status) = ANY (ARRAY['pending'::text, 'queued'::text, 'submitting'::text, 'submitted'::text, 'running'::text, 'succeeded'::text, 'completed'::text, 'failed'::text, 'error'::text, 'paused'::text, 'cancelled'::text, 'canceled'::text, 'terminated'::text]))
);
-- Create index "idx_pipeline_shards_deployment" to table: "pipeline_shards"
CREATE INDEX "idx_pipeline_shards_deployment" ON "pipeline_shards" ("deployment_id") WHERE (deployment_id IS NOT NULL);
-- Create index "idx_pipeline_shards_dispatch" to table: "pipeline_shards"
CREATE INDEX "idx_pipeline_shards_dispatch" ON "pipeline_shards" ("status", "priority", "created_at", "shard_index") WHERE (status = ANY (ARRAY['pending'::text, 'queued'::text]));
-- Create index "idx_pipeline_shards_dispatch_priority_case" to table: "pipeline_shards"
CREATE INDEX "idx_pipeline_shards_dispatch_priority_case" ON "pipeline_shards" ("status", (
CASE lower(priority)
    WHEN 'urgent'::text THEN 0
    WHEN 'high'::text THEN 1
    WHEN 'normal'::text THEN 2
    WHEN 'low'::text THEN 3
    ELSE 4
END), "created_at", "shard_index") WHERE (status = ANY (ARRAY['pending'::text, 'queued'::text]));
-- Create index "idx_pipeline_shards_problem_summary" to table: "pipeline_shards"
CREATE INDEX "idx_pipeline_shards_problem_summary" ON "pipeline_shards" ("pipeline_run_id", "status", "error_message", "message") WHERE ((status = ANY (ARRAY['failed'::text, 'error'::text, 'paused'::text])) OR (error_message <> ''::text));
-- Create index "idx_pipeline_shards_run_index" to table: "pipeline_shards"
CREATE UNIQUE INDEX "idx_pipeline_shards_run_index" ON "pipeline_shards" ("pipeline_run_id", "shard_index");
-- Create index "idx_pipeline_shards_run_status" to table: "pipeline_shards"
CREATE INDEX "idx_pipeline_shards_run_status" ON "pipeline_shards" ("pipeline_run_id", "status", "shard_index");
-- Create "pipeline_templates" table
CREATE TABLE "pipeline_templates" (
  "id" text NOT NULL,
  "name" text NOT NULL,
  "pipeline" jsonb NOT NULL,
  "node_count" integer NOT NULL DEFAULT 0,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  "version" integer NOT NULL DEFAULT 1,
  "active_version" integer NOT NULL DEFAULT 0,
  "scope" character varying(16) NOT NULL DEFAULT 'dev',
  "owner" character varying(255) NOT NULL DEFAULT '',
  "automation" jsonb NULL,
  PRIMARY KEY ("id")
);
-- Create index "idx_pipeline_templates_automation" to table: "pipeline_templates"
CREATE INDEX "idx_pipeline_templates_automation" ON "pipeline_templates" USING GIN ("automation") WHERE (automation IS NOT NULL);
-- Create index "idx_pipeline_templates_name_version" to table: "pipeline_templates"
CREATE UNIQUE INDEX "idx_pipeline_templates_name_version" ON "pipeline_templates" ("name", "version");
-- Create "run_inputs" table
CREATE TABLE "run_inputs" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "run_id" text NOT NULL,
  "node_id" text NOT NULL DEFAULT '',
  "type" text NOT NULL,
  "ref_id" text NOT NULL DEFAULT '',
  "ref_version" text NOT NULL DEFAULT '',
  "file_name" text NOT NULL DEFAULT '',
  "mount_path" text NOT NULL DEFAULT '',
  "target_filename" text NOT NULL DEFAULT '',
  "content_hash" text NOT NULL DEFAULT '',
  "projection_key" text NOT NULL DEFAULT '',
  "source" text NOT NULL DEFAULT '',
  "snapshot" jsonb NOT NULL DEFAULT '{}',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "run_inputs_run_not_blank" CHECK (btrim(run_id) <> ''::text),
  CONSTRAINT "run_inputs_type_not_blank" CHECK (btrim(type) <> ''::text)
);
-- Create index "idx_run_inputs_run_created" to table: "run_inputs"
CREATE INDEX "idx_run_inputs_run_created" ON "run_inputs" ("run_id", "created_at");
-- Create index "idx_run_inputs_unique_fact" to table: "run_inputs"
CREATE UNIQUE INDEX "idx_run_inputs_unique_fact" ON "run_inputs" ("run_id", "type", "node_id", "ref_id", "ref_version", "mount_path", "target_filename", "projection_key");
-- Create "run_relations" table
CREATE TABLE "run_relations" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "parent_run_id" text NOT NULL,
  "child_run_id" text NOT NULL,
  "relation_type" text NOT NULL,
  "asset_id" text NOT NULL DEFAULT '',
  "source" text NOT NULL DEFAULT '',
  "snapshot" jsonb NOT NULL DEFAULT '{}',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "run_relations_child_not_blank" CHECK (btrim(child_run_id) <> ''::text),
  CONSTRAINT "run_relations_no_self_edge" CHECK (parent_run_id <> child_run_id),
  CONSTRAINT "run_relations_parent_not_blank" CHECK (btrim(parent_run_id) <> ''::text),
  CONSTRAINT "run_relations_type_not_blank" CHECK (btrim(relation_type) <> ''::text)
);
-- Create index "idx_run_relations_child_created" to table: "run_relations"
CREATE INDEX "idx_run_relations_child_created" ON "run_relations" ("child_run_id", "created_at" DESC);
-- Create index "idx_run_relations_parent_child_type" to table: "run_relations"
CREATE UNIQUE INDEX "idx_run_relations_parent_child_type" ON "run_relations" ("parent_run_id", "child_run_id", "relation_type");
-- Create index "idx_run_relations_parent_created" to table: "run_relations"
CREATE INDEX "idx_run_relations_parent_created" ON "run_relations" ("parent_run_id", "created_at" DESC);
-- Create "saved_queries" table
CREATE TABLE "saved_queries" (
  "saved_query_id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "name" text NOT NULL,
  "description" text NULL,
  "resource" text NOT NULL DEFAULT 'assets',
  "schema_version" text NOT NULL DEFAULT 'v1',
  "query_ir_json" jsonb NOT NULL,
  "owner" text NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("saved_query_id")
);
-- Create index "idx_saved_queries_owner_updated" to table: "saved_queries"
CREATE INDEX "idx_saved_queries_owner_updated" ON "saved_queries" ("owner", "updated_at" DESC) WHERE (owner IS NOT NULL);
-- Create index "idx_saved_queries_resource_updated" to table: "saved_queries"
CREATE INDEX "idx_saved_queries_resource_updated" ON "saved_queries" ("resource", "updated_at" DESC);
-- Create trigger "trg_saved_queries_updated_at"
CREATE TRIGGER "trg_saved_queries_updated_at" BEFORE UPDATE ON "saved_queries" FOR EACH ROW EXECUTE FUNCTION "set_updated_at"();
-- Create "search_reindex_jobs" table
CREATE TABLE "search_reindex_jobs" (
  "id" text NOT NULL,
  "status" text NOT NULL,
  "dry_run" boolean NOT NULL DEFAULT false,
  "page_size" integer NOT NULL DEFAULT 200,
  "next_page" integer NOT NULL DEFAULT 1,
  "stop_requested" boolean NOT NULL DEFAULT false,
  "total_assets" bigint NOT NULL DEFAULT 0,
  "assets_scanned" bigint NOT NULL DEFAULT 0,
  "documents_indexed" bigint NOT NULL DEFAULT 0,
  "documents_deleted" bigint NOT NULL DEFAULT 0,
  "failed" bigint NOT NULL DEFAULT 0,
  "error" text NOT NULL DEFAULT '',
  "error_samples" jsonb NOT NULL DEFAULT '[]',
  "elasticsearch_doc_count" bigint NULL,
  "index_cleared" boolean NOT NULL DEFAULT false,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  "started_at" timestamptz NULL,
  "finished_at" timestamptz NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "search_reindex_jobs_status_check" CHECK (status = ANY (ARRAY['queued'::text, 'running'::text, 'paused'::text, 'succeeded'::text, 'failed'::text]))
);
-- Create index "idx_search_reindex_jobs_created" to table: "search_reindex_jobs"
CREATE INDEX "idx_search_reindex_jobs_created" ON "search_reindex_jobs" ("created_at" DESC);
-- Create index "idx_search_reindex_jobs_status_updated" to table: "search_reindex_jobs"
CREATE INDEX "idx_search_reindex_jobs_status_updated" ON "search_reindex_jobs" ("status", "updated_at" DESC);
-- Create "sync_watermarks" table
CREATE TABLE "sync_watermarks" (
  "table_name" text NOT NULL,
  "watermark" timestamptz NOT NULL,
  "dagster_run_id" text NULL,
  "synced_rows" bigint NULL DEFAULT 0,
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("table_name")
);
-- Create "video_durations" table
CREATE TABLE "video_durations" (
  "video_id" text NOT NULL,
  "duration_sec" double precision NOT NULL,
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("video_id")
);
-- Create "asset_algo_statuses" function
CREATE FUNCTION "asset_algo_statuses" ("algo" jsonb) RETURNS text[] LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
SELECT COALESCE(array_agg(value ORDER BY key), ARRAY[]::TEXT[])
  FROM jsonb_each_text(COALESCE(algo, '{}'::JSONB))
  WHERE key LIKE '%:status';
$$;
-- Create "event_retention_cleanup" function
CREATE FUNCTION "event_retention_cleanup" ("retention_interval" interval) RETURNS bigint LANGUAGE plpgsql AS $$
DECLARE
  deleted_count BIGINT;
BEGIN
  DELETE FROM asset_events
  WHERE publish_state IN ('published', 'dlq')
    AND created_at < now() - retention_interval;
  GET DIAGNOSTICS deleted_count = ROW_COUNT;
  RETURN deleted_count;
END;
$$;
-- Modify "actions" table
ALTER TABLE "actions" ADD CONSTRAINT "fk_act_run" FOREIGN KEY ("run_id") REFERENCES "algo_runs" ("run_id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- Modify "asset_algo_events" table
ALTER TABLE "asset_algo_events" ADD CONSTRAINT "fk_algo_events_asset" FOREIGN KEY ("asset_id") REFERENCES "assets" ("asset_id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "asset_algo_latest" table
ALTER TABLE "asset_algo_latest" ADD CONSTRAINT "asset_algo_latest_asset_id_fkey" FOREIGN KEY ("asset_id") REFERENCES "assets" ("asset_id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "fk_aal_run" FOREIGN KEY ("run_id") REFERENCES "algo_runs" ("run_id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- Modify "asset_eval_results" table
ALTER TABLE "asset_eval_results" ADD CONSTRAINT "fk_eval_run" FOREIGN KEY ("run_id") REFERENCES "algo_runs" ("run_id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- Modify "asset_relations" table
ALTER TABLE "asset_relations" ADD CONSTRAINT "asset_relations_child_asset_id_fkey" FOREIGN KEY ("child_asset_id") REFERENCES "assets" ("asset_id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "asset_relations_parent_asset_id_fkey" FOREIGN KEY ("parent_asset_id") REFERENCES "assets" ("asset_id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "asset_tags" table
ALTER TABLE "asset_tags" ADD CONSTRAINT "asset_tags_asset_id_fkey" FOREIGN KEY ("asset_id") REFERENCES "assets" ("asset_id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "fk_atags_run" FOREIGN KEY ("run_id") REFERENCES "algo_runs" ("run_id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- Modify "asset_usage_stats" table
ALTER TABLE "asset_usage_stats" ADD CONSTRAINT "asset_usage_stats_asset_id_fkey" FOREIGN KEY ("asset_id") REFERENCES "assets" ("asset_id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "assets" table
ALTER TABLE "assets" ADD CONSTRAINT "fk_assets_logical_asset" FOREIGN KEY ("logical_asset_id") REFERENCES "logical_assets" ("logical_asset_id") ON UPDATE NO ACTION ON DELETE NO ACTION DEFERRABLE INITIALLY DEFERRED, ADD CONSTRAINT "fk_assets_mcap" FOREIGN KEY ("mcap_file_id") REFERENCES "mcap_files" ("mcap_file_id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "backfill_items" table
ALTER TABLE "backfill_items" ADD CONSTRAINT "backfill_items_job_id_fkey" FOREIGN KEY ("job_id") REFERENCES "backfill_jobs" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- Modify "backfill_jobs" table
ALTER TABLE "backfill_jobs" ADD CONSTRAINT "backfill_jobs_pipeline_run_id_fkey" FOREIGN KEY ("pipeline_run_id") REFERENCES "pipeline_runs" ("id") ON UPDATE NO ACTION ON DELETE SET NULL, ADD CONSTRAINT "backfill_jobs_template_id_fkey" FOREIGN KEY ("template_id") REFERENCES "pipeline_templates" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "deliveries" table
ALTER TABLE "deliveries" ADD CONSTRAINT "fk_deliveries_customer" FOREIGN KEY ("customer_id") REFERENCES "customers" ("customer_id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "delivery_items" table
ALTER TABLE "delivery_items" ADD CONSTRAINT "delivery_items_asset_id_fkey" FOREIGN KEY ("asset_id") REFERENCES "assets" ("asset_id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "fk_delivery_items_asset" FOREIGN KEY ("asset_id") REFERENCES "assets" ("asset_id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "fk_delivery_items_delivery" FOREIGN KEY ("delivery_id") REFERENCES "deliveries" ("delivery_id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "delivery_rules" table
ALTER TABLE "delivery_rules" ADD CONSTRAINT "delivery_rules_customer_id_fkey" FOREIGN KEY ("customer_id") REFERENCES "customers" ("customer_id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "mcap_files" table
ALTER TABLE "mcap_files" ADD CONSTRAINT "fk_mcap_asset" FOREIGN KEY ("mcap_file_id") REFERENCES "assets" ("asset_id") ON UPDATE NO ACTION ON DELETE NO ACTION DEFERRABLE INITIALLY DEFERRED;
-- Modify "node_attempts" table
ALTER TABLE "node_attempts" ADD CONSTRAINT "node_attempts_node_run_id_fkey" FOREIGN KEY ("node_run_id") REFERENCES "node_runs" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- Modify "node_runs" table
ALTER TABLE "node_runs" ADD CONSTRAINT "node_runs_pipeline_run_id_fkey" FOREIGN KEY ("pipeline_run_id") REFERENCES "pipeline_runs" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, ADD CONSTRAINT "node_runs_shard_id_fkey" FOREIGN KEY ("shard_id") REFERENCES "pipeline_shards" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- Modify "pipeline_config_versions" table
ALTER TABLE "pipeline_config_versions" ADD CONSTRAINT "pipeline_config_versions_config_id_fkey" FOREIGN KEY ("config_id") REFERENCES "pipeline_configs" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT;
-- Modify "pipeline_deployments" table
ALTER TABLE "pipeline_deployments" ADD CONSTRAINT "pipeline_deployments_template_id_fkey" FOREIGN KEY ("template_id") REFERENCES "pipeline_templates" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "pipeline_revisions" table
ALTER TABLE "pipeline_revisions" ADD CONSTRAINT "pipeline_revisions_definition_id_fkey" FOREIGN KEY ("definition_id") REFERENCES "pipeline_definitions" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- Modify "pipeline_run_nodes" table
ALTER TABLE "pipeline_run_nodes" ADD CONSTRAINT "pipeline_run_nodes_run_id_fkey" FOREIGN KEY ("run_id") REFERENCES "pipeline_runs" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- Modify "pipeline_runs" table
ALTER TABLE "pipeline_runs" ADD CONSTRAINT "pipeline_runs_execution_target_id_fkey" FOREIGN KEY ("execution_target_id") REFERENCES "execution_targets" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "pipeline_runs_template_id_fkey" FOREIGN KEY ("template_id") REFERENCES "pipeline_templates" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- Modify "pipeline_shards" table
ALTER TABLE "pipeline_shards" ADD CONSTRAINT "pipeline_shards_deployment_id_fkey" FOREIGN KEY ("deployment_id") REFERENCES "pipeline_deployments" ("id") ON UPDATE NO ACTION ON DELETE SET NULL, ADD CONSTRAINT "pipeline_shards_pipeline_run_id_fkey" FOREIGN KEY ("pipeline_run_id") REFERENCES "pipeline_runs" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
