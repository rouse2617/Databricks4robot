# Proposal — CYB-2178

## Why
Pipeline runtime config was modeled at deploy level, but the intended user model is node scoped: configs from the config library should be attached to individual pipeline nodes/components.

## What Changes

### New Capabilities
- Pipeline designer nodes can bind an optional saved config-library entry to that node.
- Saved pipeline templates persist each node's config binding and restore it when reopened.
- Pipeline deploy resolves node-level config bindings into immutable config content snapshots.
- Generated Argo workflows mount each node's selected config only into that node's container/script template.
- Node runtime env vars describe the mounted config path and selected config metadata.
- Deploy-selected asset IDs remain run-level context and are injected into every workflow node Pod through environment variables by default.

### Modified Capabilities
- Deploy-level config selection is no longer the primary UI model for normal pipeline runs.
- Existing deploy-level `configSelection` payload remains backend-compatible for callers that already use it, but node-level bindings take precedence for configured nodes.
- Config-library entries are used as reusable task/node parameters instead of one global file for the whole DAG.
- The runtime model separates two scopes: assets are selected once at deploy/run time for the whole pipeline, while config files are selected per node/component.

## Impact
- **Affected code**: `Frontend/src/components/pipeline/NodeConfigPanel.tsx`, `Frontend/src/components/pipeline/PipelineNode.tsx`, `Frontend/src/components/pipeline/types.ts`, `Frontend/src/lib/pipeline-design/*`, `Frontend/src/api/pipelineConfigs.ts`, `Frontend/src/components/pipeline/DeployPanel.tsx`, `backend/internal/transpiler/*`, `backend/internal/usecase/pipeline/usecase.go`, `backend/internal/k8s/runtime_config.go`, `api/openapi.yaml`, `docs/review/api-guide.md`
- **New APIs**: no new endpoint; existing pipeline template/deploy payload schemas gain optional node-level config binding fields.
- **Dependencies**: existing `/api/v1/pipeline-configs` config library and existing runtime ConfigMap projection infrastructure.

## Scope
- **In scope**: saved config-library binding per node, node config UI, save/load round trip, backend saved-config resolution, node-specific ConfigMap mount projection, node-local config env vars, preservation of run-level asset env injection into every Pod, tests and dev browser regression.
- **Out of scope**: per-node local file upload, per-node inline draft editing, multi-file config bundles per node, DB schema changes, arbitrary parsing of config content into env vars, exposing Kubernetes mount internals as normal user concepts.

## Success Criteria
- [ ] A user can open a node/组件 configuration panel and choose a ready config from the config library for that node.
- [ ] The selected node shows enough summary to make the binding visible before deploy.
- [ ] Saving and reopening a pipeline preserves each node's config binding.
- [ ] A pipeline with two nodes can bind two different configs without either config leaking to the other node.
- [ ] Deploying a node-configured pipeline creates runtime config projections and mounts each config only on its bound node.
- [ ] The node container receives env vars such as `PIPELINE_CONFIG_PATH`, `PIPELINE_CONFIG_ID`, and `PIPELINE_CONFIG_VERSION` for its own selected config.
- [ ] Asset IDs selected at deploy time are still injected into every node Pod as run-level env vars such as `ASSET_IDS`, `ASSET_COUNT`, and per-asset indexed IDs.
- [ ] Node-level config binding does not change which assets a run processes; asset selection remains a deploy/run-level decision.
- [ ] If a saved config is missing, not ready, or not accessible, deploy fails before Argo submission with a clear validation error.
- [ ] Existing deploy-level config payloads remain accepted for compatibility, but the main UI no longer suggests that one deploy config is the normal per-component configuration model.

## Goals (SLO)
- **Latency**: opening the node config panel should load ready config choices without a full-page refresh.
- **Concurrency**: deploying a pipeline with many configured nodes should create at most one runtime config projection object per run where practical.
- **Quality**: frontend round-trip tests and backend workflow-manifest tests cover at least one multi-node, multi-config scenario.
