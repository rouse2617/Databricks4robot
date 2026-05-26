-- 000_initial.sql — Full current schema (squashed from 36 incremental migrations).
-- Single source of truth for fresh deployments. Executed automatically on first
-- docker-compose up via /docker-entrypoint-initdb.d mount.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE FUNCTION asset_algo_statuses(algo jsonb) RETURNS text[]
    LANGUAGE sql IMMUTABLE PARALLEL SAFE
    AS $$
  SELECT COALESCE(array_agg(value ORDER BY key), ARRAY[]::TEXT[])
  FROM jsonb_each_text(COALESCE(algo, '{}'::JSONB))
  WHERE key LIKE '%:status';
$$;

CREATE FUNCTION event_retention_cleanup(retention_interval interval) RETURNS bigint
    LANGUAGE plpgsql
    AS $$
DECLARE
BEGIN
  DELETE FROM asset_events
  WHERE publish_state IN ('published', 'dlq')
    AND created_at < now() - retention_interval;
END;
$$;

CREATE FUNCTION set_updated_at() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$;

CREATE FUNCTION trg_logical_assets_type_immutable() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
  IF NEW.asset_type IS DISTINCT FROM OLD.asset_type THEN
    RAISE EXCEPTION 'logical_assets.asset_type is immutable (LA1)';
  END IF;
  RETURN NEW;
END;
$$;

SET default_tablespace = '';

SET default_table_access_method = heap;

CREATE TABLE actions (
    action_id text NOT NULL,
    asset_id text NOT NULL,
    start_ns bigint NOT NULL,
    end_ns bigint NOT NULL,
    action_index integer,
    primary_label text,
    labels text[] DEFAULT '{}'::text[] NOT NULL,
    description text,
    attrs jsonb DEFAULT '{}'::jsonb NOT NULL,
    source_type text DEFAULT 'human'::text NOT NULL,
    source_name text,
    source_version text,
    run_id text,
    confidence double precision,
    external_id text,
    tenant_id text,
    project_id text,
    version bigint DEFAULT 1 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    task_id text,
    CONSTRAINT actions_id_chk CHECK ((action_id ~ '^[0-9A-Za-z]{8}$'::text)),
    CONSTRAINT actions_range_chk CHECK ((end_ns >= start_ns))
);

CREATE TABLE algo_runs (
    run_id text NOT NULL,
    algo_name text NOT NULL,
    algo_version text NOT NULL,
    algo_kind text NOT NULL,
    triggered_by text NOT NULL,
    status text DEFAULT 'pending'::text NOT NULL,
    started_at timestamp with time zone,
    finished_at timestamp with time zone,
    duration_ns bigint GENERATED ALWAYS AS (
CASE
    WHEN ((finished_at IS NOT NULL) AND (started_at IS NOT NULL)) THEN ((EXTRACT(epoch FROM (finished_at - started_at)))::bigint * 1000000000)
    ELSE NULL::bigint
END) STORED,
    input_filter jsonb DEFAULT '{}'::jsonb NOT NULL,
    input_asset_ids text[],
    params jsonb DEFAULT '{}'::jsonb NOT NULL,
    code_commit text,
    image_digest text,
    pipeline_name text,
    pipeline_version text,
    assets_processed integer,
    assets_succeeded integer,
    assets_failed integer,
    actions_created integer,
    metrics_written integer,
    outputs jsonb DEFAULT '{}'::jsonb NOT NULL,
    cpu_seconds bigint,
    gpu_seconds bigint,
    cost_usd_micros bigint,
    error_class text,
    error_message text,
    tenant_id text,
    project_id text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    external_runtime text,
    external_url text,
    CONSTRAINT algo_runs_algo_kind_check CHECK ((algo_kind = ANY (ARRAY['processing'::text, 'split'::text, 'qa'::text, 'enrichment'::text]))),
    CONSTRAINT algo_runs_run_id_check CHECK ((run_id ~ '^[0-9A-Za-z]{16}$'::text)),
    CONSTRAINT algo_runs_status_check CHECK ((status = ANY (ARRAY['pending'::text, 'running'::text, 'ok'::text, 'failed'::text, 'cancelled'::text])))
);

CREATE TABLE asset_algo_latest (
    asset_id text NOT NULL,
    algo_name text NOT NULL,
    algo_version text NOT NULL,
    status text NOT NULL,
    result_tag text,
    result_score double precision,
    result_summary jsonb DEFAULT '{}'::jsonb NOT NULL,
    run_id text,
    method text,
    model_uri text,
    output_uri text,
    error_code text,
    error_message text,
    started_at timestamp with time zone,
    finished_at timestamp with time zone,
    tenant_id text,
    project_id text,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE TABLE asset_eval_results (
    eval_result_id uuid DEFAULT gen_random_uuid() NOT NULL,
    asset_id text NOT NULL,
    mcap_file_id text,
    target_type text DEFAULT 'segment'::text NOT NULL,
    target_id text DEFAULT ''::text NOT NULL,
    eval_name text NOT NULL,
    eval_version text NOT NULL,
    parameter_version text,
    run_id text,
    status text DEFAULT 'ok'::text NOT NULL,
    result_payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    output_uri text,
    summary_uri text,
    source_type text DEFAULT 'algo'::text NOT NULL,
    source_name text,
    source_version text,
    started_at timestamp with time zone,
    finished_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE TABLE asset_events (
    event_id uuid DEFAULT gen_random_uuid() NOT NULL,
    event_seq bigint NOT NULL,
    event_type text NOT NULL,
    aggregate_type text DEFAULT 'asset'::text NOT NULL,
    payload_schema_version text DEFAULT 'v1'::text NOT NULL,
    asset_id text,
    mcap_file_id text,
    tenant_id text,
    project_id text,
    event_source text NOT NULL,
    actor_type text,
    actor_id text,
    request_id text,
    idempotency_key text,
    run_id text,
    occurred_at timestamp with time zone DEFAULT now() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    publish_state text DEFAULT 'pending'::text NOT NULL,
    published_at timestamp with time zone,
    event_payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    retry_count integer DEFAULT 0 NOT NULL,
    last_error text
)
PARTITION BY RANGE (occurred_at);

CREATE SEQUENCE asset_events_event_seq_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE asset_events_event_seq_seq OWNED BY asset_events.event_seq;

CREATE TABLE asset_events_default (
    event_id uuid DEFAULT gen_random_uuid() NOT NULL,
    event_seq bigint DEFAULT nextval('asset_events_event_seq_seq'::regclass) NOT NULL,
    event_type text NOT NULL,
    aggregate_type text DEFAULT 'asset'::text NOT NULL,
    payload_schema_version text DEFAULT 'v1'::text NOT NULL,
    asset_id text,
    mcap_file_id text,
    tenant_id text,
    project_id text,
    event_source text NOT NULL,
    actor_type text,
    actor_id text,
    request_id text,
    idempotency_key text,
    run_id text,
    occurred_at timestamp with time zone DEFAULT now() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    publish_state text DEFAULT 'pending'::text NOT NULL,
    published_at timestamp with time zone,
    event_payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    retry_count integer DEFAULT 0 NOT NULL,
    last_error text
);

CREATE TABLE asset_metrics (
    asset_id text NOT NULL,
    target_type text DEFAULT 'segment'::text NOT NULL,
    target_id text DEFAULT ''::text NOT NULL,
    metric_key text NOT NULL,
    metric_type text DEFAULT 'float'::text NOT NULL,
    metric_unit text,
    metric_value double precision,
    metric_value_int bigint,
    metric_value_text text,
    metric_value_bool boolean,
    eval_name text NOT NULL,
    eval_version text NOT NULL,
    parameter_version text,
    run_id text,
    source_type text DEFAULT 'algo'::text NOT NULL,
    source_name text,
    confidence double precision,
    recorded_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE TABLE asset_relations (
    parent_asset_id text NOT NULL,
    child_asset_id text NOT NULL,
    relation_type text NOT NULL,
    method text,
    algo_name text,
    algo_version text,
    run_id text,
    parent_start_offset_ms bigint,
    parent_end_offset_ms bigint,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_relation_type CHECK ((relation_type = ANY (ARRAY['split_from'::text, 'derived_from'::text, 'contains'::text, 'sampled_from'::text, 'merged_from'::text, 'revision_of'::text])))
);

CREATE TABLE asset_tags (
    asset_id text NOT NULL,
    tag_key text NOT NULL,
    tag_value text NOT NULL,
    tag_value_num double precision,
    tag_value_bool boolean,
    tag_type text DEFAULT 'string'::text NOT NULL,
    source_type text DEFAULT 'human'::text NOT NULL,
    source_name text,
    source_version text,
    run_id text,
    confidence double precision,
    tenant_id text,
    project_id text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    id bigint NOT NULL,
    applied_at timestamp with time zone DEFAULT now() NOT NULL,
    source_version_norm text GENERATED ALWAYS AS (COALESCE(source_version, ''::text)) STORED
);

CREATE SEQUENCE asset_tags_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE asset_tags_id_seq OWNED BY asset_tags.id;

CREATE TABLE asset_usage_stats (
    id bigint NOT NULL,
    asset_id text NOT NULL,
    logical_asset_id text,
    view_count integer DEFAULT 0 NOT NULL,
    last_viewed_at timestamp with time zone,
    favorite_count integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE SEQUENCE asset_usage_stats_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE asset_usage_stats_id_seq OWNED BY asset_usage_stats.id;

CREATE TABLE assets (
    asset_id text NOT NULL,
    mcap_file_id text,
    start_timestamp_ns bigint NOT NULL,
    end_timestamp_ns bigint NOT NULL,
    segment_locator character(40),
    asset_type text DEFAULT 'segment'::text NOT NULL,
    lifecycle_state text DEFAULT 'created'::text NOT NULL,
    duration_ms bigint DEFAULT 0 NOT NULL,
    owner text DEFAULT ''::text NOT NULL,
    reviewer text DEFAULT ''::text NOT NULL,
    storage_uri text DEFAULT ''::text NOT NULL,
    thumb_uri text DEFAULT ''::text NOT NULL,
    retention_tier text DEFAULT ''::text NOT NULL,
    expire_at timestamp with time zone,
    asset_level integer DEFAULT 0 NOT NULL,
    parent_asset_id text,
    root_asset_id text,
    delivery_count integer DEFAULT 0 NOT NULL,
    last_delivered_at timestamp with time zone,
    last_delivered_to text DEFAULT ''::text NOT NULL,
    segment_index integer,
    parent_start_offset_ms bigint,
    parent_end_offset_ms bigint,
    split_method text,
    split_algo_name text,
    split_algo_version text,
    split_run_id text,
    split_reason text,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    files jsonb DEFAULT '{}'::jsonb NOT NULL,
    algo_inputs_uris jsonb DEFAULT '{}'::jsonb NOT NULL,
    annot_inputs_uris jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id text,
    project_id text,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    version bigint DEFAULT 1,
    logical_asset_id text,
    revision bigint,
    is_current boolean,
    CONSTRAINT assets_asset_id_check CHECK ((asset_id ~ '^[0-9A-Za-z]{8}$'::text)),
    CONSTRAINT assets_logical_asset_id_check CHECK (((logical_asset_id IS NULL) OR (logical_asset_id ~ '^[0-9A-Za-z]{8}$'::text))),
    CONSTRAINT assets_mcap_file_id_check CHECK ((mcap_file_id ~ '^[0-9A-Za-z]{8}$'::text)),
    CONSTRAINT assets_revision_check CHECK (((revision IS NULL) OR (revision >= 1))),
    CONSTRAINT chk_lifecycle_state CHECK ((lifecycle_state = ANY (ARRAY['created'::text, 'processing'::text, 'ready'::text, 'delivered'::text, 'archived'::text, 'superseded'::text, 'failed'::text, 'rejected'::text]))),
    CONSTRAINT chk_mcap_file_required CHECK (((asset_type = 'derived_asset'::text) OR (mcap_file_id IS NOT NULL)))
);

CREATE TABLE audit_events (
    event_id uuid DEFAULT gen_random_uuid() NOT NULL,
    actor text NOT NULL,
    action text NOT NULL,
    resource_type text NOT NULL,
    resource_ids text[] NOT NULL,
    request_summary jsonb DEFAULT '{}'::jsonb,
    request_id text,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE TABLE customers (
    customer_id text NOT NULL,
    display_name text NOT NULL,
    legal_name text,
    status text DEFAULT 'active'::text NOT NULL,
    region text,
    sla_tier text DEFAULT 'standard'::text NOT NULL,
    account_owner text,
    compliance_tags jsonb DEFAULT '[]'::jsonb NOT NULL,
    exclude_tags jsonb DEFAULT '[]'::jsonb NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    extra jsonb DEFAULT '{}'::jsonb NOT NULL,
    onboarded_at timestamp with time zone,
    offboarded_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT customers_sla_tier_check CHECK ((sla_tier = ANY (ARRAY['standard'::text, 'premium'::text, 'enterprise'::text]))),
    CONSTRAINT customers_status_check CHECK ((status = ANY (ARRAY['active'::text, 'trial'::text, 'suspended'::text, 'offboarded'::text])))
);

CREATE TABLE deliveries (
    delivery_id uuid NOT NULL,
    customer_id text NOT NULL,
    status character varying(16) DEFAULT 'pending'::character varying NOT NULL,
    delivered_at timestamp with time zone,
    contract_id text,
    delivery_type text DEFAULT 'asset_set'::text NOT NULL,
    requested_by text,
    approved_by text,
    delivered_by text,
    manifest_uri text,
    replay_manifest_uri text,
    item_count bigint DEFAULT 0 NOT NULL,
    total_size_bytes bigint,
    completed_at timestamp with time zone,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id text,
    project_id text,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    version bigint DEFAULT 1
);

CREATE TABLE delivery_items (
    delivery_id uuid NOT NULL,
    asset_id text NOT NULL,
    created_at timestamp with time zone DEFAULT now()
);

CREATE TABLE delivery_rules (
    rule_id uuid DEFAULT gen_random_uuid() NOT NULL,
    name text NOT NULL,
    owner text NOT NULL,
    customer_id text,
    query_dsl jsonb NOT NULL,
    dsl_version text DEFAULT 'v1'::text NOT NULL,
    enforce_mode text DEFAULT 'block'::text NOT NULL,
    rating_scope text DEFAULT 'current'::text NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    version bigint DEFAULT 1 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT delivery_rules_enforce_mode_check CHECK ((enforce_mode = ANY (ARRAY['block'::text, 'warn'::text, 'tag_only'::text]))),
    CONSTRAINT delivery_rules_rating_scope_check CHECK ((rating_scope = ANY (ARRAY['current'::text, 'logical'::text])))
);

CREATE TABLE es_sync_checkpoint (
    shard_id integer NOT NULL,
    applied_seq bigint NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE TABLE idempotency_keys (
    scope character varying(64) NOT NULL,
    idem_key character varying(128) NOT NULL,
    request_hash character varying(64),
    status_code integer,
    response_json jsonb,
    created_at timestamp with time zone
);

CREATE TABLE lakehouse_bronze_checkpoint (
    id integer NOT NULL,
    applied_seq bigint NOT NULL,
    ingested_at timestamp with time zone NOT NULL,
    run_id text,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT lakehouse_bronze_checkpoint_singleton CHECK ((id = 1))
);

CREATE TABLE logical_assets (
    logical_asset_id text NOT NULL,
    asset_type text NOT NULL,
    display_name text,
    description text,
    owner text,
    status text DEFAULT 'active'::text NOT NULL,
    current_revision bigint DEFAULT 1 NOT NULL,
    total_revisions bigint DEFAULT 1 NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    extra jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT logical_assets_check CHECK ((total_revisions >= current_revision)),
    CONSTRAINT logical_assets_current_revision_check CHECK ((current_revision >= 1)),
    CONSTRAINT logical_assets_logical_asset_id_check CHECK ((logical_asset_id ~ '^[0-9A-Za-z]{8}$'::text)),
    CONSTRAINT logical_assets_status_check CHECK ((status = ANY (ARRAY['active'::text, 'archived'::text])))
);

CREATE TABLE mcap_files (
    mcap_file_id text NOT NULL,
    raw_hash_md5 character varying(32),
    mcap_uri text DEFAULT ''::text NOT NULL,
    size_bytes bigint DEFAULT 0 NOT NULL,
    file_duration_ms bigint DEFAULT 0 NOT NULL,
    start_timestamp_ns bigint DEFAULT 0 NOT NULL,
    end_timestamp_ns bigint DEFAULT 0 NOT NULL,
    channel_count integer DEFAULT 0 NOT NULL,
    chunk_count integer DEFAULT 0 NOT NULL,
    ingest_state text DEFAULT 'pending'::text NOT NULL,
    vendor_id text,
    collector_id text,
    task_id text,
    device_id text,
    camera_model text,
    data_source text,
    location_id text,
    scene_id text,
    environment_id text,
    collection_method text,
    owner text DEFAULT ''::text NOT NULL,
    retention_tier text,
    expire_at timestamp with time zone,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    process_state jsonb DEFAULT '{}'::jsonb NOT NULL,
    raw_hash_sha256 text,
    tenant_id text,
    project_id text,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    version bigint DEFAULT 1,
    summary_index_state text DEFAULT 'pending'::text NOT NULL,
    summary_index_version text,
    summary_indexed_at timestamp with time zone,
    summary_index_attempts integer DEFAULT 0 NOT NULL,
    summary_index_error text,
    CONSTRAINT chk_mcap_files_summary_index_state CHECK ((summary_index_state = ANY (ARRAY['pending'::text, 'running'::text, 'done'::text, 'failed'::text]))),
    CONSTRAINT mcap_files_mcap_file_id_check CHECK ((mcap_file_id ~ '^[0-9A-Za-z]{8}$'::text))
);

CREATE TABLE outbox_dlq (
    dlq_id bigint NOT NULL,
    event_id uuid NOT NULL,
    event_seq bigint NOT NULL,
    event_type text NOT NULL,
    aggregate_type text DEFAULT 'asset'::text NOT NULL,
    asset_id text,
    mcap_file_id text,
    event_source text DEFAULT 'backend'::text NOT NULL,
    event_payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    retry_count integer DEFAULT 0 NOT NULL,
    last_error text,
    original_created_at timestamp with time zone NOT NULL,
    moved_at timestamp with time zone DEFAULT now() NOT NULL,
    resolved_at timestamp with time zone,
    resolution text
);

CREATE SEQUENCE outbox_dlq_dlq_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE outbox_dlq_dlq_id_seq OWNED BY outbox_dlq.dlq_id;

CREATE TABLE saved_queries (
    saved_query_id uuid DEFAULT gen_random_uuid() NOT NULL,
    name text NOT NULL,
    description text,
    resource text DEFAULT 'assets'::text NOT NULL,
    schema_version text DEFAULT 'v1'::text NOT NULL,
    query_ir_json jsonb NOT NULL,
    owner text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE TABLE search_reindex_jobs (
    id text NOT NULL,
    status text NOT NULL,
    dry_run boolean DEFAULT false NOT NULL,
    page_size integer DEFAULT 200 NOT NULL,
    next_page integer DEFAULT 1 NOT NULL,
    stop_requested boolean DEFAULT false NOT NULL,
    total_assets bigint DEFAULT 0 NOT NULL,
    assets_scanned bigint DEFAULT 0 NOT NULL,
    documents_indexed bigint DEFAULT 0 NOT NULL,
    failed bigint DEFAULT 0 NOT NULL,
    error text DEFAULT ''::text NOT NULL,
    error_samples jsonb DEFAULT '[]'::jsonb NOT NULL,
    elasticsearch_doc_count bigint,
    index_cleared boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    started_at timestamp with time zone,
    finished_at timestamp with time zone,
    CONSTRAINT search_reindex_jobs_status_check CHECK ((status = ANY (ARRAY['queued'::text, 'running'::text, 'paused'::text, 'succeeded'::text, 'failed'::text])))
);

CREATE TABLE sync_watermarks (
    table_name text NOT NULL,
    watermark timestamp with time zone NOT NULL,
    dagster_run_id text,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

ALTER TABLE ONLY asset_events ATTACH PARTITION asset_events_default DEFAULT;

ALTER TABLE ONLY asset_events ALTER COLUMN event_seq SET DEFAULT nextval('asset_events_event_seq_seq'::regclass);

ALTER TABLE ONLY asset_tags ALTER COLUMN id SET DEFAULT nextval('asset_tags_id_seq'::regclass);

ALTER TABLE ONLY asset_usage_stats ALTER COLUMN id SET DEFAULT nextval('asset_usage_stats_id_seq'::regclass);

ALTER TABLE ONLY outbox_dlq ALTER COLUMN dlq_id SET DEFAULT nextval('outbox_dlq_dlq_id_seq'::regclass);

ALTER TABLE ONLY actions
    ADD CONSTRAINT actions_pkey PRIMARY KEY (action_id);

ALTER TABLE ONLY algo_runs
    ADD CONSTRAINT algo_runs_pkey PRIMARY KEY (run_id);

ALTER TABLE ONLY asset_algo_latest
    ADD CONSTRAINT asset_algo_latest_pkey PRIMARY KEY (asset_id, algo_name);

ALTER TABLE ONLY asset_eval_results
    ADD CONSTRAINT asset_eval_results_pkey PRIMARY KEY (eval_result_id);

ALTER TABLE ONLY asset_events
    ADD CONSTRAINT asset_events_pkey1 PRIMARY KEY (event_seq, occurred_at);

ALTER TABLE ONLY asset_events_default
    ADD CONSTRAINT asset_events_default_pkey PRIMARY KEY (event_seq, occurred_at);

ALTER TABLE ONLY asset_metrics
    ADD CONSTRAINT asset_metrics_pkey PRIMARY KEY (asset_id, target_type, target_id, metric_key, eval_name, eval_version);

ALTER TABLE ONLY asset_relations
    ADD CONSTRAINT asset_relations_pkey PRIMARY KEY (parent_asset_id, child_asset_id, relation_type);

ALTER TABLE ONLY asset_tags
    ADD CONSTRAINT asset_tags_pkey PRIMARY KEY (id);

ALTER TABLE ONLY asset_usage_stats
    ADD CONSTRAINT asset_usage_stats_asset_id_key UNIQUE (asset_id);

ALTER TABLE ONLY asset_usage_stats
    ADD CONSTRAINT asset_usage_stats_pkey PRIMARY KEY (id);

ALTER TABLE ONLY assets
    ADD CONSTRAINT assets_pkey PRIMARY KEY (asset_id);

ALTER TABLE ONLY audit_events
    ADD CONSTRAINT audit_events_pkey PRIMARY KEY (event_id);

ALTER TABLE ONLY customers
    ADD CONSTRAINT customers_pkey PRIMARY KEY (customer_id);

ALTER TABLE ONLY deliveries
    ADD CONSTRAINT deliveries_pkey PRIMARY KEY (delivery_id);

ALTER TABLE ONLY delivery_items
    ADD CONSTRAINT delivery_items_pkey PRIMARY KEY (delivery_id, asset_id);

ALTER TABLE ONLY delivery_rules
    ADD CONSTRAINT delivery_rules_pkey PRIMARY KEY (rule_id);

ALTER TABLE ONLY es_sync_checkpoint
    ADD CONSTRAINT es_sync_checkpoint_pkey PRIMARY KEY (shard_id);

ALTER TABLE ONLY idempotency_keys
    ADD CONSTRAINT idempotency_keys_pkey PRIMARY KEY (scope, idem_key);

ALTER TABLE ONLY lakehouse_bronze_checkpoint
    ADD CONSTRAINT lakehouse_bronze_checkpoint_pkey PRIMARY KEY (id);

ALTER TABLE ONLY logical_assets
    ADD CONSTRAINT logical_assets_pkey PRIMARY KEY (logical_asset_id);

ALTER TABLE ONLY mcap_files
    ADD CONSTRAINT mcap_files_pkey PRIMARY KEY (mcap_file_id);

ALTER TABLE ONLY outbox_dlq
    ADD CONSTRAINT outbox_dlq_pkey PRIMARY KEY (dlq_id);

ALTER TABLE ONLY saved_queries
    ADD CONSTRAINT saved_queries_pkey PRIMARY KEY (saved_query_id);

ALTER TABLE ONLY search_reindex_jobs
    ADD CONSTRAINT search_reindex_jobs_pkey PRIMARY KEY (id);

ALTER TABLE ONLY sync_watermarks
    ADD CONSTRAINT sync_watermarks_pkey PRIMARY KEY (table_name);

ALTER TABLE ONLY asset_tags
    ADD CONSTRAINT uq_asset_tags_identity UNIQUE (asset_id, tag_key, tag_value, source_type, source_version_norm);

CREATE INDEX idx_asset_events_aggregate_time ON ONLY asset_events USING btree (aggregate_type, occurred_at DESC);

CREATE INDEX asset_events_default_aggregate_type_occurred_at_idx ON asset_events_default USING btree (aggregate_type, occurred_at DESC);

CREATE INDEX idx_asset_events_asset ON ONLY asset_events USING btree (asset_id, occurred_at DESC) WHERE (asset_id IS NOT NULL);

CREATE INDEX asset_events_default_asset_id_occurred_at_idx ON asset_events_default USING btree (asset_id, occurred_at DESC) WHERE (asset_id IS NOT NULL);

CREATE INDEX idx_asset_events_retention ON ONLY asset_events USING btree (created_at) WHERE (publish_state = 'published'::text);

CREATE INDEX asset_events_default_created_at_idx ON asset_events_default USING btree (created_at) WHERE (publish_state = 'published'::text);

CREATE INDEX idx_asset_events_type_time ON ONLY asset_events USING btree (event_type, occurred_at DESC);

CREATE INDEX asset_events_default_event_type_occurred_at_idx ON asset_events_default USING btree (event_type, occurred_at DESC);

CREATE INDEX idx_asset_events_publish_pending ON ONLY asset_events USING btree (publish_state, event_seq) WHERE (publish_state = 'pending'::text);

CREATE INDEX asset_events_default_publish_state_event_seq_idx ON asset_events_default USING btree (publish_state, event_seq) WHERE (publish_state = 'pending'::text);

CREATE INDEX idx_asset_events_retry_high ON ONLY asset_events USING btree (retry_count) WHERE ((retry_count > 5) AND (publish_state = 'pending'::text));

CREATE INDEX asset_events_default_retry_count_idx ON asset_events_default USING btree (retry_count) WHERE ((retry_count > 5) AND (publish_state = 'pending'::text));

CREATE INDEX idx_asset_events_run ON ONLY asset_events USING btree (run_id) WHERE (run_id IS NOT NULL);

CREATE INDEX asset_events_default_run_id_idx ON asset_events_default USING btree (run_id) WHERE (run_id IS NOT NULL);

CREATE INDEX idx_asset_events_tenant_project ON ONLY asset_events USING btree (tenant_id, project_id);

CREATE INDEX asset_events_default_tenant_id_project_id_idx ON asset_events_default USING btree (tenant_id, project_id);

CREATE INDEX idx_algo_runs_algo ON algo_runs USING btree (algo_name, algo_version, started_at DESC);

CREATE INDEX idx_algo_runs_started ON algo_runs USING btree (started_at DESC);

CREATE INDEX idx_algo_runs_status ON algo_runs USING btree (status) WHERE (status = ANY (ARRAY['pending'::text, 'running'::text, 'failed'::text]));

CREATE INDEX idx_algo_runs_triggered ON algo_runs USING btree (triggered_by) WHERE (triggered_by ~~ 'manual:%'::text);

CREATE INDEX idx_asset_algo_latest_algo_status ON asset_algo_latest USING btree (algo_name, status);

CREATE INDEX idx_asset_algo_latest_algo_version ON asset_algo_latest USING btree (algo_name, algo_version);

CREATE INDEX idx_asset_algo_latest_run ON asset_algo_latest USING btree (run_id) WHERE (run_id IS NOT NULL);

CREATE INDEX idx_asset_algo_latest_updated ON asset_algo_latest USING btree (updated_at DESC);

CREATE INDEX idx_asset_metrics_asset ON asset_metrics USING btree (asset_id, target_type, target_id);

CREATE INDEX idx_asset_metrics_key_value ON asset_metrics USING btree (metric_key, metric_value);

CREATE INDEX idx_asset_relations_child ON asset_relations USING btree (child_asset_id, relation_type);

CREATE INDEX idx_asset_tags_key_value_bool ON asset_tags USING btree (tag_key, tag_value_bool, asset_id) WHERE (tag_type = 'bool'::text);

CREATE INDEX idx_asset_tags_key_value_num ON asset_tags USING btree (tag_key, tag_value_num, asset_id) WHERE (tag_type = 'number'::text);

CREATE INDEX idx_asset_tags_key_value_str ON asset_tags USING btree (tag_key, tag_value, asset_id);

CREATE INDEX idx_asset_tags_tenant_project ON asset_tags USING btree (tenant_id, project_id);

CREATE INDEX idx_asset_usage_stats_logical ON asset_usage_stats USING btree (logical_asset_id);

CREATE INDEX idx_assets_algo_inputs_uris_gin ON assets USING gin (algo_inputs_uris);

CREATE INDEX idx_assets_annot_inputs_uris_gin ON assets USING gin (annot_inputs_uris);

CREATE INDEX idx_assets_created_at ON assets USING btree (created_at DESC);

CREATE INDEX idx_assets_logical ON assets USING btree (logical_asset_id);

CREATE INDEX idx_assets_mcap_file_id ON assets USING btree (mcap_file_id);

CREATE INDEX idx_assets_metadata_gin ON assets USING gin (metadata);

CREATE INDEX idx_assets_parent ON assets USING btree (parent_asset_id) WHERE (parent_asset_id IS NOT NULL);

CREATE INDEX idx_assets_root ON assets USING btree (root_asset_id) WHERE (root_asset_id IS NOT NULL);

CREATE INDEX idx_assets_segment_locator ON assets USING btree (segment_locator);

CREATE INDEX idx_atags_lookup ON asset_tags USING btree (tag_key, tag_value, asset_id);

CREATE INDEX idx_atags_propagation ON asset_tags USING btree (tag_key) WHERE (tag_key ~~ 'compliance.%'::text);

CREATE INDEX idx_atags_run ON asset_tags USING btree (run_id) WHERE (run_id IS NOT NULL);

CREATE INDEX idx_atags_source ON asset_tags USING btree (source_type, source_name, source_version);

CREATE INDEX idx_audit_events_action_created ON audit_events USING btree (action, created_at DESC);

CREATE INDEX idx_audit_events_actor_created ON audit_events USING btree (actor, created_at DESC);

CREATE INDEX idx_customers_account_owner ON customers USING btree (account_owner) WHERE (account_owner IS NOT NULL);

CREATE INDEX idx_customers_region ON customers USING btree (region) WHERE (region IS NOT NULL);

CREATE INDEX idx_customers_sla ON customers USING btree (sla_tier);

CREATE INDEX idx_customers_status ON customers USING btree (status);

CREATE INDEX idx_deliveries_customer_created ON deliveries USING btree (customer_id, created_at DESC);

CREATE INDEX idx_delivery_items_asset_id ON delivery_items USING btree (asset_id);

CREATE INDEX idx_drules_active ON delivery_rules USING btree (is_active) WHERE (is_active = true);

CREATE INDEX idx_drules_customer ON delivery_rules USING btree (customer_id, is_active);

CREATE INDEX idx_eval_results_asset ON asset_eval_results USING btree (asset_id, target_type, target_id, created_at DESC);

CREATE INDEX idx_eval_results_name_ver ON asset_eval_results USING btree (eval_name, eval_version, parameter_version, created_at DESC);

CREATE INDEX idx_eval_results_run_id ON asset_eval_results USING btree (run_id) WHERE (run_id IS NOT NULL);

CREATE INDEX idx_lassets_owner ON logical_assets USING btree (owner) WHERE (owner IS NOT NULL);

CREATE INDEX idx_lassets_status ON logical_assets USING btree (status) WHERE (status <> 'active'::text);

CREATE INDEX idx_lassets_type ON logical_assets USING btree (asset_type, status);

CREATE INDEX idx_lassets_updated ON logical_assets USING btree (updated_at DESC);

CREATE INDEX idx_mcap_files_metadata_gin ON mcap_files USING gin (metadata);

CREATE INDEX idx_outbox_dlq_asset ON outbox_dlq USING btree (asset_id) WHERE (resolved_at IS NULL);

CREATE INDEX idx_outbox_dlq_unresolved ON outbox_dlq USING btree (moved_at DESC) WHERE (resolved_at IS NULL);

CREATE INDEX idx_saved_queries_owner_updated ON saved_queries USING btree (owner, updated_at DESC) WHERE (owner IS NOT NULL);

CREATE INDEX idx_saved_queries_resource_updated ON saved_queries USING btree (resource, updated_at DESC);

CREATE INDEX idx_search_reindex_jobs_created ON search_reindex_jobs USING btree (created_at DESC);

CREATE INDEX idx_search_reindex_jobs_status_updated ON search_reindex_jobs USING btree (status, updated_at DESC);

ALTER INDEX idx_asset_events_aggregate_time ATTACH PARTITION asset_events_default_aggregate_type_occurred_at_idx;

ALTER INDEX idx_asset_events_asset ATTACH PARTITION asset_events_default_asset_id_occurred_at_idx;

ALTER INDEX idx_asset_events_retention ATTACH PARTITION asset_events_default_created_at_idx;

ALTER INDEX idx_asset_events_type_time ATTACH PARTITION asset_events_default_event_type_occurred_at_idx;

ALTER INDEX asset_events_pkey1 ATTACH PARTITION asset_events_default_pkey;

ALTER INDEX idx_asset_events_publish_pending ATTACH PARTITION asset_events_default_publish_state_event_seq_idx;

ALTER INDEX idx_asset_events_retry_high ATTACH PARTITION asset_events_default_retry_count_idx;

ALTER INDEX idx_asset_events_run ATTACH PARTITION asset_events_default_run_id_idx;

ALTER INDEX idx_asset_events_tenant_project ATTACH PARTITION asset_events_default_tenant_id_project_id_idx;

CREATE TRIGGER logical_assets_type_immutable BEFORE UPDATE ON logical_assets FOR EACH ROW EXECUTE FUNCTION trg_logical_assets_type_immutable();

CREATE TRIGGER trg_asset_algo_latest_updated_at BEFORE UPDATE ON asset_algo_latest FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_asset_eval_results_updated_at BEFORE UPDATE ON asset_eval_results FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_asset_metrics_updated_at BEFORE UPDATE ON asset_metrics FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_asset_tags_updated_at BEFORE UPDATE ON asset_tags FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_saved_queries_updated_at BEFORE UPDATE ON saved_queries FOR EACH ROW EXECUTE FUNCTION set_updated_at();

ALTER TABLE ONLY asset_algo_latest
    ADD CONSTRAINT asset_algo_latest_asset_id_fkey FOREIGN KEY (asset_id) REFERENCES assets(asset_id);

ALTER TABLE asset_events
    ADD CONSTRAINT asset_events_asset_id_fkey1 FOREIGN KEY (asset_id) REFERENCES assets(asset_id);

ALTER TABLE asset_events
    ADD CONSTRAINT asset_events_mcap_file_id_fkey1 FOREIGN KEY (mcap_file_id) REFERENCES mcap_files(mcap_file_id);

ALTER TABLE ONLY asset_relations
    ADD CONSTRAINT asset_relations_child_asset_id_fkey FOREIGN KEY (child_asset_id) REFERENCES assets(asset_id);

ALTER TABLE ONLY asset_relations
    ADD CONSTRAINT asset_relations_parent_asset_id_fkey FOREIGN KEY (parent_asset_id) REFERENCES assets(asset_id);

ALTER TABLE ONLY asset_tags
    ADD CONSTRAINT asset_tags_asset_id_fkey FOREIGN KEY (asset_id) REFERENCES assets(asset_id);

ALTER TABLE ONLY asset_usage_stats
    ADD CONSTRAINT asset_usage_stats_asset_id_fkey FOREIGN KEY (asset_id) REFERENCES assets(asset_id);

ALTER TABLE ONLY assets
    ADD CONSTRAINT assets_parent_asset_id_fkey FOREIGN KEY (parent_asset_id) REFERENCES assets(asset_id);

ALTER TABLE ONLY assets
    ADD CONSTRAINT assets_root_asset_id_fkey FOREIGN KEY (root_asset_id) REFERENCES assets(asset_id);

ALTER TABLE ONLY delivery_items
    ADD CONSTRAINT delivery_items_asset_id_fkey FOREIGN KEY (asset_id) REFERENCES assets(asset_id);

ALTER TABLE ONLY delivery_rules
    ADD CONSTRAINT delivery_rules_customer_id_fkey FOREIGN KEY (customer_id) REFERENCES customers(customer_id);

ALTER TABLE ONLY asset_algo_latest
    ADD CONSTRAINT fk_aal_run FOREIGN KEY (run_id) REFERENCES algo_runs(run_id) ON DELETE SET NULL;

ALTER TABLE ONLY actions
    ADD CONSTRAINT fk_act_run FOREIGN KEY (run_id) REFERENCES algo_runs(run_id) ON DELETE SET NULL;

ALTER TABLE ONLY assets
    ADD CONSTRAINT fk_assets_logical_asset FOREIGN KEY (logical_asset_id) REFERENCES logical_assets(logical_asset_id) DEFERRABLE INITIALLY DEFERRED;

ALTER TABLE ONLY assets
    ADD CONSTRAINT fk_assets_mcap FOREIGN KEY (mcap_file_id) REFERENCES mcap_files(mcap_file_id);

ALTER TABLE ONLY asset_tags
    ADD CONSTRAINT fk_atags_run FOREIGN KEY (run_id) REFERENCES algo_runs(run_id) ON DELETE SET NULL;

ALTER TABLE ONLY deliveries
    ADD CONSTRAINT fk_deliveries_customer FOREIGN KEY (customer_id) REFERENCES customers(customer_id);

ALTER TABLE ONLY delivery_items
    ADD CONSTRAINT fk_delivery_items_asset FOREIGN KEY (asset_id) REFERENCES assets(asset_id);

ALTER TABLE ONLY delivery_items
    ADD CONSTRAINT fk_delivery_items_delivery FOREIGN KEY (delivery_id) REFERENCES deliveries(delivery_id);

ALTER TABLE ONLY asset_eval_results
    ADD CONSTRAINT fk_eval_run FOREIGN KEY (run_id) REFERENCES algo_runs(run_id) ON DELETE SET NULL;

ALTER TABLE ONLY mcap_files
    ADD CONSTRAINT fk_mcap_asset FOREIGN KEY (mcap_file_id) REFERENCES assets(asset_id) DEFERRABLE INITIALLY DEFERRED;
