-- CYB-3744: scheduled auto-dispatch rules ("定时任务"). Each row is a rule that,
-- on its own interval, fetches asset-ids from a configured data source
-- (v1: generic REST-JSON) and creates a batch for the given pipeline template
-- on the given execution target. Replaces the external grace-sync Cloud Run
-- Job + Cloud Scheduler, moving scheduled dispatch into the DataBrew backend
-- as a self-service, in-app feature.
--
-- Design details: openspec/changes/CYB-3744-scheduled-tasks/design.md
--
-- Multi-replica safety: the scheduler loop claims a rule via a conditional
-- UPDATE (see backend usecase), so at most one replica executes a rule per
-- cycle. No cross-row transactional coupling here.
CREATE TABLE "scheduled_tasks" (
  "id"                 text        NOT NULL,
  "name"               text        NOT NULL,
  "enabled"            boolean     NOT NULL DEFAULT true,

  -- pipeline + resource pool
  "template_id"        text        NOT NULL,
  "template_version"   integer,
  "target_id"          text        NOT NULL,
  -- resource_defaults.scheduling shape (nodeSelector/tolerations), free-form
  -- so we can evolve the pool config without another migration.
  "scheduling"         jsonb       NOT NULL DEFAULT '{}',

  -- data source (v1: source_type='rest'; source_config is the generic REST
  -- source config: base_url, auth{type, secret_ref}, query, paging, id_path).
  -- Credentials are ONLY a secret-manager reference (secret_ref); never
  -- plaintext.
  "source_type"        text        NOT NULL,
  "source_config"      jsonb       NOT NULL DEFAULT '{}',

  -- trigger:
  --   trigger_mode ∈ ('incremental','rolling','range','ids')
  --   trigger_config carries mode-specific fields:
  --     incremental: {interval_seconds, time_field, initial_lookback_seconds}
  --     rolling:     {interval_seconds, lookback_seconds, time_field}
  --     range:       {from, to, time_field}
  --     ids:         {ids: [...]}
  "trigger_mode"       text        NOT NULL,
  "trigger_config"     jsonb       NOT NULL DEFAULT '{}',

  -- watermark cursor for incremental mode (opaque string, usually the last
  -- observed time_field value). NULL means "never run" -> first run uses
  -- trigger_config.initial_lookback_seconds.
  "cursor"             text,

  -- observation
  "last_run_at"        timestamptz,
  "last_run_status"    text,               -- 'succeeded' | 'failed' | 'empty' | 'skipped'
  "last_batch_id"      text,               -- backfill_jobs.id of the last created batch
  "last_error"         text,
  "last_success_at"    timestamptz,        -- for "stuck" detection (alerts)
  -- run_now_requested_at: user clicked 立即运行. Set to now() to bypass the
  -- interval check on the next scheduler tick; the loop clears it after
  -- claiming the rule.
  "run_now_requested_at" timestamptz,

  -- audit
  "created_by"         text        NOT NULL DEFAULT '',
  "created_at"         timestamptz NOT NULL DEFAULT now(),
  "updated_at"         timestamptz NOT NULL DEFAULT now(),

  PRIMARY KEY ("id"),

  CONSTRAINT scheduled_tasks_name_not_blank      CHECK (btrim(name) <> ''),
  CONSTRAINT scheduled_tasks_template_not_blank  CHECK (btrim(template_id) <> ''),
  CONSTRAINT scheduled_tasks_target_not_blank    CHECK (btrim(target_id) <> ''),
  CONSTRAINT scheduled_tasks_source_type_v1      CHECK (source_type IN ('rest')),
  CONSTRAINT scheduled_tasks_trigger_mode_valid  CHECK (trigger_mode IN ('incremental','rolling','range','ids')),
  CONSTRAINT scheduled_tasks_last_run_status_valid
    CHECK (last_run_status IS NULL OR last_run_status IN ('succeeded','failed','empty','skipped'))
);

-- Unique display name so the UI list + audit references stay unambiguous.
CREATE UNIQUE INDEX "scheduled_tasks_name_uniq" ON "scheduled_tasks" ("name");

-- Scheduler loop scans enabled rules to find due ones; keep it cheap.
CREATE INDEX "scheduled_tasks_enabled_last_run_at_idx"
  ON "scheduled_tasks" ("enabled", "last_run_at")
  WHERE enabled = true;
