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
  "is_deleted" boolean NOT NULL DEFAULT false,
  "external_id" text NULL,
  "tenant_id" text NULL,
  "project_id" text NULL,
  "version" bigint NOT NULL DEFAULT 1,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  "task_id" text NULL,
  PRIMARY KEY ("action_id")
);
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
  "duration_ns" bigint NULL,
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
  "external_runtime" text NULL,
  "external_url" text NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("run_id")
);
-- Create "asset_algo_latest" table
CREATE TABLE "asset_algo_latest" (
  "asset_id" text NOT NULL,
  "algo" text NOT NULL,
  "algo_version" text NOT NULL,
  "status" text NOT NULL,
  "score" double precision NULL,
  "result" jsonb NULL,
  "started_at" timestamptz NOT NULL,
  "completed_at" timestamptz NOT NULL,
  "recorded_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("asset_id", "algo")
);
-- Create "asset_eval_results" table
CREATE TABLE "asset_eval_results" (
  "eval_id" bigserial NOT NULL,
  "asset_id" text NOT NULL,
  "eval_type" text NOT NULL,
  "score" double precision NULL,
  "threshold" double precision NULL,
  "passed" boolean NULL,
  "metrics" jsonb NULL,
  "started_at" timestamptz NULL,
  "completed_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("eval_id")
);
-- Create "asset_metrics" table
CREATE TABLE "asset_metrics" (
  "asset_id" text NOT NULL,
  "metric_name" text NOT NULL,
  "bucket_start" timestamptz NOT NULL,
  "value" double precision NOT NULL,
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("asset_id", "metric_name", "bucket_start")
);
-- Create "asset_relations" table
CREATE TABLE "asset_relations" (
  "relation_id" bigserial NOT NULL,
  "from_asset_id" text NOT NULL,
  "to_asset_id" text NOT NULL,
  "relation_type" text NOT NULL,
  "algo_name" text NULL,
  "algo_version" text NULL,
  "confidence" double precision NULL,
  "metadata" jsonb NOT NULL DEFAULT '{}',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("relation_id")
);
-- Create "asset_tags" table
CREATE TABLE "asset_tags" (
  "asset_id" text NOT NULL,
  "source" text NOT NULL,
  "key" text NOT NULL,
  "value" text NOT NULL,
  "confidence" double precision NULL,
  "producer_id" text NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("asset_id", "source", "key")
);
-- Create "asset_usage_stats" table
CREATE TABLE "asset_usage_stats" (
  "asset_id" text NOT NULL,
  "view_count" bigint NOT NULL DEFAULT 0,
  "download_count" bigint NOT NULL DEFAULT 0,
  "query_count" bigint NOT NULL DEFAULT 0,
  "delivery_count" bigint NOT NULL DEFAULT 0,
  "last_used_at" timestamptz NULL,
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("asset_id")
);
-- Create "assets" table
CREATE TABLE "assets" (
  "asset_id" text NOT NULL,
  "mcap_file_id" text NULL,
  "segment_locator" character(40) NULL,
  "start_timestamp_ns" bigint NOT NULL,
  "end_timestamp_ns" bigint NOT NULL,
  "reviewer" text NOT NULL DEFAULT '',
  "owner" text NOT NULL DEFAULT '',
  "last_delivered_at" timestamptz NULL,
  "last_delivered_to" text NOT NULL DEFAULT '',
  "delivery_count" integer NOT NULL DEFAULT 0,
  "files" jsonb NOT NULL DEFAULT '{}',
  "asset_type" text NOT NULL DEFAULT 'segment',
  "lifecycle_state" text NOT NULL DEFAULT 'created',
  "is_deleted" boolean NULL DEFAULT false,
  "duration_ms" bigint NOT NULL DEFAULT 0,
  "storage_uri" text NOT NULL DEFAULT '',
  "thumb_uri" text NOT NULL DEFAULT '',
  "retention_tier" text NOT NULL DEFAULT '',
  "expire_at" timestamptz NULL,
  "asset_level" integer NOT NULL DEFAULT 0,
  "parent_asset_id" text NULL,
  "root_asset_id" text NULL,
  "segment_index" integer NULL,
  "parent_start_offset_ms" bigint NULL,
  "parent_end_offset_ms" bigint NULL,
  "split_method" text NULL,
  "split_algo_name" text NULL,
  "split_algo_version" text NULL,
  "split_run_id" text NULL,
  "split_reason" text NULL,
  "metadata" jsonb NOT NULL DEFAULT '{}',
  "algo_inputs_uris" jsonb NOT NULL DEFAULT '{}',
  "annot_inputs_uris" jsonb NOT NULL DEFAULT '{}',
  "tenant_id" text NULL,
  "project_id" text NULL,
  "logical_asset_id" text NULL,
  "revision" bigint NULL,
  "is_current" boolean NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "version" bigint NULL DEFAULT 1,
  PRIMARY KEY ("asset_id")
);
-- Create "audit_events" table
CREATE TABLE "audit_events" (
  "event_id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "actor" text NOT NULL,
  "action" text NOT NULL,
  "resource_type" text NOT NULL,
  "resource_ids" text[] NOT NULL,
  "request_summary" jsonb NOT NULL DEFAULT '{}',
  "request_id" text NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("event_id")
);
-- Create "backfill_items" table
CREATE TABLE "backfill_items" (
  "id" text NOT NULL,
  "job_id" text NOT NULL,
  "asset_id" text NOT NULL,
  "status" text NOT NULL DEFAULT 'pending',
  "workflow_name" text NULL,
  "run_id" text NULL,
  "attempts" integer NOT NULL DEFAULT 0,
  "last_error" text NULL,
  "error_message" text NULL,
  "started_at" timestamptz NULL,
  "finished_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);
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
  "created_by" text NULL,
  "finished_at" timestamptz NULL,
  "notification_sent_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);
-- Create "component_build_runs" table
CREATE TABLE "component_build_runs" (
  "run_id" text NOT NULL,
  "component_id" text NOT NULL DEFAULT '',
  "repo_url" text NOT NULL DEFAULT '',
  "git_ref" text NOT NULL DEFAULT '',
  "commit_sha" text NOT NULL DEFAULT '',
  "dockerfile" text NOT NULL DEFAULT 'Dockerfile',
  "build_context" text NOT NULL DEFAULT '.',
  "image_repository" text NOT NULL DEFAULT '',
  "image_tag" text NOT NULL DEFAULT '',
  "image_digest" text NOT NULL DEFAULT '',
  PRIMARY KEY ("run_id")
);
-- Create "component_releases" table
CREATE TABLE "component_releases" (
  "id" text NOT NULL,
  "component_id" text NOT NULL,
  "source_commit" text NOT NULL DEFAULT '',
  "image" text NOT NULL DEFAULT '',
  "image_tag" text NOT NULL DEFAULT '',
  "image_digest" text NOT NULL DEFAULT '',
  "release_label" text NOT NULL DEFAULT '',
  "build_run_id" text NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);
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
  PRIMARY KEY ("customer_id")
);
-- Create "databrew_runs" table
CREATE TABLE "databrew_runs" (
  "id" text NOT NULL,
  "type" text NOT NULL,
  "name" text NOT NULL,
  "status" text NOT NULL DEFAULT 'Pending',
  "runtime" text NOT NULL DEFAULT 'argo',
  "runtime_namespace" text NOT NULL DEFAULT '',
  "runtime_resource_name" text NOT NULL,
  "runtime_uid" text NOT NULL DEFAULT '',
  "owner" text NOT NULL DEFAULT '',
  "created_by" text NOT NULL DEFAULT '',
  "message" text NOT NULL DEFAULT '',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "started_at" timestamptz NULL,
  "finished_at" timestamptz NULL,
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);
-- Create "deliveries" table
CREATE TABLE "deliveries" (
  "delivery_id" uuid NOT NULL,
  "customer_id" text NOT NULL,
  "status" character varying(16) NOT NULL DEFAULT 'pending',
  "delivered_at" timestamptz NULL,
  "is_deleted" boolean NOT NULL DEFAULT false,
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
  "version" bigint NOT NULL DEFAULT 1,
  PRIMARY KEY ("delivery_id")
);
-- Create "delivery_items" table
CREATE TABLE "delivery_items" (
  "delivery_id" uuid NOT NULL,
  "asset_id" text NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("delivery_id", "asset_id")
);
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
  PRIMARY KEY ("rule_id")
);
-- Create "es_sync_checkpoint" table
CREATE TABLE "es_sync_checkpoint" (
  "shard_id" bigserial NOT NULL,
  "applied_seq" bigint NOT NULL,
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("shard_id")
);
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
  PRIMARY KEY ("id")
);
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
  "id" bigserial NOT NULL,
  "applied_seq" bigint NOT NULL,
  "ingested_at" timestamptz NOT NULL,
  "run_id" text NULL,
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);
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
  PRIMARY KEY ("logical_asset_id")
);
-- Create "mcap_files" table
CREATE TABLE "mcap_files" (
  "mcap_file_id" text NOT NULL,
  "gcs_path" text NOT NULL,
  "size_bytes" bigint NOT NULL,
  "raw_hash_md5" text NOT NULL,
  "raw_hash_sha256" text NULL,
  "ingest_state" text NOT NULL DEFAULT 'pending',
  "file_duration_ms" bigint NULL,
  "start_timestamp_ns" bigint NULL,
  "end_timestamp_ns" bigint NULL,
  "channel_count" integer NULL,
  "chunk_count" integer NULL,
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
  "owner" text NULL,
  "retention_tier" text NULL,
  "expire_at" timestamptz NULL,
  "tenant_id" text NULL,
  "project_id" text NULL,
  "metadata" jsonb NULL,
  "process_state" jsonb NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "version" bigint NULL DEFAULT 1,
  PRIMARY KEY ("mcap_file_id")
);
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
  "scope" text NOT NULL DEFAULT 'dev',
  "owner" text NOT NULL DEFAULT '',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);
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
  PRIMARY KEY ("id")
);
-- Create index "idx_pipeline_config_versions_config_version" to table: "pipeline_config_versions"
CREATE UNIQUE INDEX "idx_pipeline_config_versions_config_version" ON "pipeline_config_versions" ("version");
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
  PRIMARY KEY ("id")
);
-- Create index "idx_pipeline_configs_name" to table: "pipeline_configs"
CREATE UNIQUE INDEX "idx_pipeline_configs_name" ON "pipeline_configs" ("name");
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
  "scope" text NOT NULL DEFAULT 'dev',
  "owner" text NOT NULL DEFAULT '',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  "finished_at" timestamptz NULL,
  PRIMARY KEY ("id")
);
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
  PRIMARY KEY ("id")
);
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
  PRIMARY KEY ("id")
);
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
  "estimated_cost_usd" numeric(16,8) NULL,
  "started_at" timestamptz NULL,
  "finished_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);
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
  PRIMARY KEY ("id")
);
-- Create "pipeline_run_watcher_state" table
CREATE TABLE "pipeline_run_watcher_state" (
  "id" text NOT NULL,
  "last_synced_at" timestamptz NULL,
  "active_scan_limit" integer NOT NULL DEFAULT 100,
  "last_error" text NULL,
  "health_status" text NULL DEFAULT 'healthy',
  "health_message" text NULL,
  "health_checked_at" timestamptz NULL,
  "updated_at" timestamptz NOT NULL DEFAULT now(),
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
  "scope" text NOT NULL DEFAULT 'dev',
  "owner" text NOT NULL DEFAULT '',
  "ledger_state" text NOT NULL DEFAULT 'pending',
  "backfill_job_id" text NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  "started_at" timestamptz NULL,
  "finished_at" timestamptz NULL,
  PRIMARY KEY ("id")
);
-- Create index "idx_pipeline_runs_workflow_name" to table: "pipeline_runs"
CREATE UNIQUE INDEX "idx_pipeline_runs_workflow_name" ON "pipeline_runs" ("workflow_name");
-- Create "pipeline_templates" table
CREATE TABLE "pipeline_templates" (
  "id" text NOT NULL,
  "name" text NOT NULL,
  "version" integer NOT NULL DEFAULT 1,
  "pipeline" jsonb NOT NULL,
  "node_count" integer NOT NULL DEFAULT 0,
  "scope" text NOT NULL DEFAULT 'dev',
  "owner" text NOT NULL DEFAULT '',
  "active_version" integer NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);
-- Create "rag_build_runs" table
CREATE TABLE "rag_build_runs" (
  "run_id" text NOT NULL,
  "knowledge_base_id" text NOT NULL DEFAULT '',
  "datasource_snapshot" jsonb NOT NULL DEFAULT '{}',
  "embedding_model" text NOT NULL DEFAULT '',
  "vector_index_name" text NOT NULL DEFAULT '',
  "release_version" text NOT NULL DEFAULT '',
  PRIMARY KEY ("run_id")
);
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
  PRIMARY KEY ("id")
);
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
  PRIMARY KEY ("id")
);
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
  "failed" bigint NOT NULL DEFAULT 0,
  "error" text NOT NULL DEFAULT '',
  "error_samples" jsonb NOT NULL DEFAULT '[]',
  "elasticsearch_doc_count" bigint NULL,
  "index_cleared" boolean NOT NULL DEFAULT false,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  "started_at" timestamptz NULL,
  "finished_at" timestamptz NULL,
  PRIMARY KEY ("id")
);
-- Create "sync_watermarks" table
CREATE TABLE "sync_watermarks" (
  "table_name" text NOT NULL,
  "watermark" timestamptz NOT NULL,
  "dagster_run_id" text NULL,
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
