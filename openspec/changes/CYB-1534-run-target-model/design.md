# Design — CYB-1534 Run Target Model

## Current State
`pipeline_templates` and `pipeline_deployments` are the only persisted pipeline tables. `ExecutionTarget` is synthesized from the process-wide Argo client and namespace. Deploy calls write a `PipelineDeployment`, inject selected asset IDs into `pipeline_json`, and refresh status by reading Argo.

## Data Model

### `execution_targets`
- `id TEXT PRIMARY KEY`
- `name TEXT NOT NULL`
- `description TEXT NOT NULL DEFAULT ''`
- `cluster TEXT NOT NULL`
- `namespace TEXT NOT NULL`
- `service_account TEXT NOT NULL DEFAULT ''`
- `argo_server_url TEXT NOT NULL DEFAULT ''`
- `argo_auth_secret_ref TEXT NOT NULL DEFAULT ''`
- `argo_insecure_skip_verify BOOLEAN NOT NULL DEFAULT FALSE`
- `argo_ca_cert_ref TEXT NOT NULL DEFAULT ''`
- `enabled BOOLEAN NOT NULL DEFAULT TRUE`
- `status TEXT NOT NULL DEFAULT 'available'`
- `is_default BOOLEAN NOT NULL DEFAULT FALSE`
- `resource_defaults JSONB NOT NULL DEFAULT '{}'`
- `quota_policy JSONB NOT NULL DEFAULT '{}'`
- `labels JSONB NOT NULL DEFAULT '{}'`
- `created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`
- `updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`

Use a partial unique index to enforce one default target.

### `pipeline_runs`
- `id TEXT PRIMARY KEY`
- `template_id TEXT REFERENCES pipeline_templates(id) ON DELETE SET NULL`
- `pipeline_name TEXT NOT NULL`
- `template_version INT`
- `workflow_name TEXT NOT NULL UNIQUE`
- `execution_target_id TEXT NOT NULL REFERENCES execution_targets(id)`
- `target_snapshot JSONB NOT NULL DEFAULT '{}'`
- `status TEXT NOT NULL DEFAULT 'Pending'`
- `node_count INT NOT NULL DEFAULT 0`
- `asset_ids TEXT[] NOT NULL DEFAULT '{}'`
- `asset_count INT NOT NULL DEFAULT 0`
- `no_asset_run BOOLEAN NOT NULL DEFAULT FALSE`
- `manifest TEXT`
- `pipeline_json JSONB NOT NULL DEFAULT '{}'`
- `argo_namespace TEXT NOT NULL`
- `argo_workflow_uid TEXT NOT NULL DEFAULT ''`
- `message TEXT NOT NULL DEFAULT ''`
- `created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`
- `updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`
- `started_at TIMESTAMPTZ`
- `finished_at TIMESTAMPTZ`

Indexes: `created_at DESC`, `status, updated_at`, `template_id, created_at DESC`, GIN on `asset_ids`.

### `pipeline_run_nodes`
- `id TEXT PRIMARY KEY`
- `run_id TEXT NOT NULL REFERENCES pipeline_runs(id) ON DELETE CASCADE`
- `pipeline_node_id TEXT NOT NULL`
- `argo_node_id TEXT NOT NULL DEFAULT ''`
- `argo_node_name TEXT NOT NULL DEFAULT ''`
- `display_name TEXT NOT NULL DEFAULT ''`
- `template_name TEXT NOT NULL DEFAULT ''`
- `type TEXT NOT NULL DEFAULT ''`
- `phase TEXT NOT NULL DEFAULT ''`
- `message TEXT NOT NULL DEFAULT ''`
- `pod_name TEXT NOT NULL DEFAULT ''`
- `host_node_name TEXT NOT NULL DEFAULT ''`
- `children TEXT[] NOT NULL DEFAULT '{}'`
- `inputs JSONB NOT NULL DEFAULT '{}'`
- `outputs JSONB NOT NULL DEFAULT '{}'`
- `resources_duration JSONB NOT NULL DEFAULT '{}'`
- `resource_summary JSONB NOT NULL DEFAULT '{}'`
- `log_ref TEXT NOT NULL DEFAULT ''`
- `started_at TIMESTAMPTZ`
- `finished_at TIMESTAMPTZ`
- `created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`
- `updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`

Indexes: unique `(run_id, argo_node_id)` when `argo_node_id <> ''`, and `(run_id, phase)`.

## API
- Keep `GET /api/v1/execution-targets` and back it with the repository.
- Add `POST /api/v1/pipeline-runs`.
- Add `GET /api/v1/pipeline-runs`.
- Add `GET /api/v1/pipeline-runs/{id}`.
- Add `POST /api/v1/pipeline-runs/{id}/retry`.
- Add `POST /api/v1/pipeline-runs/{id}/stop`.
- Add `DELETE /api/v1/pipeline-runs/{id}`.

Existing deploy/deployment endpoints stay registered and either write both compatibility and run rows or project deployment-shaped responses from run rows.

## Execution Flow
1. Resolve target ID. Empty or `default` uses the enabled default target.
2. Validate assets through the shared validator once `CYB-1536-asset-validation` lands.
3. Transpile with target namespace, service account, workflow TTL, and resource defaults.
4. Insert a `pipeline_runs` row before Argo submission.
5. Submit the workflow to the target's Argo endpoint.
6. Update run status, workflow UID if available, and compatibility deployment data.
7. Refresh active run nodes from Argo with a bounded cap.

## Compatibility
- `/api/v1/deploy` and `/api/v1/deploy/template/{id}` keep accepting `asset_ids` and `target_id`.
- `/api/v1/deployments` keeps returning `PipelineDeployment`-shaped JSON for current frontend compatibility.
- `PIPELINE_DEPLOYMENT_ID` remains injected until containers can use a new `PIPELINE_RUN_ID`.
- `POST /api/v1/pipeline-assets` accepts the old deployment ID and resolves the corresponding run.

## Risks
- `backend/migrations/` requires explicit approval.
- Node identity mapping is not one-to-one across pipeline node IDs, Argo node IDs, and pod names.
- Status refresh can create excessive Argo calls; keep capped refresh behavior.
- Target secrets must remain references, not raw API response values.
