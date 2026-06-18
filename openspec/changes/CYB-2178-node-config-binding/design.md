# Design — CYB-2178

## Architecture Context
- **Constraints**: pipeline templates currently persist arbitrary pipeline JSON; node data has no first-class config binding; backend deploy currently resolves one optional deploy-level `ConfigSelection` and injects it through global env vars plus mounts on every node. Deploy-selected asset IDs are already run-level context and should continue to be injected into every node Pod through global env vars.
- **Goals**: move the primary config UX to node/component instances, keep template snapshots self-contained, keep asset selection deploy-level, and keep runtime behavior deterministic by resolving saved config versions at deploy time.
- **Non-Goals**: storing new config content from the node panel, replacing the config library CRUD model, or adding Kubernetes-specific scheduling/config concepts to the normal node form.

## Affected Modules
- `Frontend/src/components/pipeline/NodeConfigPanel.tsx` — add saved-config selector, mount destination defaults, clear state, and validation copy.
- `Frontend/src/components/pipeline/PipelineNode.tsx` — show compact config-bound indicator on configured nodes.
- `Frontend/src/components/pipeline/types.ts` — add node-level config binding types.
- `Frontend/src/lib/pipeline-design/canvas-to-dsl.ts` / `dsl-to-canvas.ts` — preserve config binding across save/load.
- `Frontend/src/components/pipeline/DeployPanel.tsx` — de-emphasize or remove deploy-level config controls from the normal run path.
- `backend/internal/usecase/pipeline/usecase.go` — resolve node-level config references and project them before transpilation.
- `backend/internal/k8s/runtime_config.go` — support multi-file runtime config projection if one ConfigMap per run is used.
- `backend/internal/transpiler/*` — keep mounts/env node-local rather than workflow-global.

## Architecture Decisions

### Decision 1: Store config binding on the pipeline node DSL
- **Approach**: add an optional node field, tentatively `runtimeConfig`, with saved-config identity and mount destination:
  - `mode: "saved"`
  - `configId`
  - `version`
  - `fileName`
  - `mountPath`
  - `targetFilename`
- **Alternative**: store config binding in deploy request state only.
- **Rationale**: users configure a node/component instance while designing the pipeline; the binding must be versioned with the template snapshot and visible before deploy.
- **Trade-off**: template JSON schema expands, so old templates need tolerant parsing.

### Decision 2: MVP supports saved config-library entries per node
- **Approach**: node config UI lists ready saved configs from `/api/v1/pipeline-configs` and persists selected config metadata.
- **Alternative**: also support per-node upload/inline editing in the first slice.
- **Rationale**: the corrected requirement explicitly says config-library configs should attach to each node. Per-node upload/inline would add draft lifecycle ambiguity and should wait until the saved-config model is stable.
- **Trade-off**: one-off per-node config files still need to be saved into the library first.

### Decision 3: Resolve saved config versions at deploy time
- **Approach**: backend validates every node-level saved config reference and reads immutable version content during deploy. Missing, deprecated, draft, or inaccessible config references fail before Argo submission.
- **Alternative**: frontend snapshots config content into the pipeline template.
- **Rationale**: config files already have a versioned library; templates should reference the version, not duplicate file content.
- **Rollback**: ignore node `runtimeConfig` in deploy and fall back to previous deploy-level behavior.

### Decision 4: Node-local env vars and mounts, not global env
- **Approach**: append config env vars and volume mounts only to the configured node's transpiler node. Use `PIPELINE_CONFIG_PATH` for container compatibility, and include metadata vars such as `PIPELINE_CONFIG_ID`, `PIPELINE_CONFIG_VERSION`, and `PIPELINE_CONFIG_SOURCE`.
- **Alternative**: keep one global `PIPELINE_CONFIG_PATH` for every node.
- **Rationale**: a DAG can contain multiple components with different config files; global env makes that impossible and was the root product mismatch.
- **Risk**: components that expected one global config may need migration if users move to node-level bindings.

### Decision 5: Keep asset IDs as deploy-level global runtime context
- **Approach**: asset IDs selected in the deploy/run dialog stay outside node config. Backend continues to inject asset context into every node Pod through global env vars, including `ASSET_IDS`, `ASSET_COUNT`, and indexed values such as `ASSET_0_ID`.
- **Alternative**: attach asset IDs to individual nodes.
- **Rationale**: assets answer "what is this pipeline run processing", while node config answers "how should this component process it". Splitting asset context per node would add user burden and break the normal batch/pipeline run model.
- **Trade-off**: every node sees the full selected asset set. Components that only need one asset must choose the relevant item from env context or input files.

### Decision 6: Preserve deploy-level configSelection as compatibility only
- **Approach**: backend continues to accept the current deploy-level `configSelection` request. If a node has `runtimeConfig`, node-level config wins for that node. The main UI should avoid presenting deploy-level config as the normal component-configuration model.
- **Alternative**: remove deploy-level config from API immediately.
- **Rationale**: preserving the API avoids breaking smoke scripts and existing callers while correcting the primary product flow.

## Data Flow

```text
User opens node config panel
        ↓
Frontend lists ready saved configs from config library
        ↓
User selects one config for this node and saves the node
        ↓
Pipeline template JSON stores nodes[i].runtimeConfig
        ↓
User deploys saved pipeline
        ↓
Backend resolves deploy-selected assets as run-level env context
        ↓
Backend resolves each node runtimeConfig to immutable config version content
        ↓
Backend creates runtime ConfigMap projection for this run
        ↓
Transpiler mounts the selected file only on the matching node template
        ↓
Every node container receives ASSET_* env vars
        ↓
Configured node containers also read PIPELINE_CONFIG_PATH for their own config
```

## Data Model Changes
- **Table**: none planned.
- **Change**: pipeline template JSON gains optional node-level `runtimeConfig`; existing config library tables remain the source of config content/version metadata.
- **Migration**: none.

## Risks / Trade-offs

| Risk | Impact | Mitigation |
|------|--------|------------|
| Existing deploy-level config UI creates product confusion | Users keep configuring the wrong scope | Move normal UI to node panel and label old deploy-level path as compatibility/advanced if retained |
| Asset context and config context are conflated | Users may think assets must be selected per node | Keep asset selection in deploy/run dialog and document that it is injected into every Pod by default |
| Config list only returns draft configs for a user | Saved-config selector appears empty | Filter and explain ready-only behavior; guide users to publish/mark configs ready |
| Multiple nodes use same target filename | ConfigMap keys can collide | Use unique internal projection keys while preserving per-node target filename through `subPath` |
| Old templates lack `runtimeConfig` | Backward compatibility risk | Treat missing node config as valid and deploy unchanged |
| Config becomes deprecated after template save | Deploy surprises users | Fail early with clear message and show stale/deprecated state in node UI when possible |
