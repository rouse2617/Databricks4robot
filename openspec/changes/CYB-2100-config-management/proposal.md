# Proposal — CYB-2100

## Why
The current config-management work stops at library CRUD plus a deploy-panel draft UI. Users still cannot carry a chosen config into the runtime Pod, which leaves the main deploy workflow blocked even though configs already exist.

## What Changes

### New Capabilities
- The deploy panel adds a config-file section with three source modes: pick a saved platform config, upload a local file, or edit inline content.
- Users can review the selected config source, file name, version summary, and mount target before deploy.
- Deploy requests carry a concrete config snapshot so the backend can materialize a runtime file even for upload/inline draft sources.
- The selected config is projected into the workflow Pod through a mounted file and companion environment variables that describe the selected source and target path.
- Components and deploy-time runtime settings expose a mount-path contract so the chosen file has a clear in-container destination.

### Modified Capabilities
- Deploy-time config selection is no longer limited to previously saved config records.
- The deploy UX becomes a guided file-source workflow instead of a future-placeholder action.
- Batch deploys reuse the same config selection so every subtask run receives the same runtime file contract.

## Impact
- **Affected code**: `Frontend/src/components/pipeline/DeployPanel.tsx`, `Frontend/src/components/pipeline/DeployPanel.test.tsx`, `Frontend/src/api/pipelineConfigs.ts`, `Frontend/src/api/pipelineApi.ts`, `Frontend/src/api/deployPipelineRun.ts`, `Frontend/src/api/batchJobApi.ts`, `Frontend/src/pages/PipelinePage.tsx`, `backend/internal/handlers/pipeline/handler.go`, `backend/internal/handlers/backfill/handler.go`, `backend/internal/usecase/pipeline/usecase.go`, `backend/internal/usecase/backfill/usecase.go`, `backend/internal/transpiler/*`, `api/openapi.yaml`, `docs/review/api-guide.md`
- **New APIs**: deploy and batch-create payloads gain config selection fields
- **Dependencies**: existing standalone config library under `/registry`; existing Kubernetes client wiring already used for Pod diagnostics/exec; current Argo workflow generation path

## Scope
- **In scope**: deploy-panel config-source switcher, saved-config picker UI, local upload UI, inline editor UI, mount-path input UI, draft validation/copy, preview summary in deploy panel, deploy payload contract, backend config snapshot resolution, runtime file mount injection, companion env var projection, batch-job propagation for the same config selection
- **Out of scope**: multi-file mounts per run, persistent storage of ad hoc upload/inline configs into the shared library, component-manager schema changes, arbitrary key-value env templating from config file content

## Success Criteria
- [ ] The deploy panel shows a clear "配置文件" section with three mutually exclusive source modes.
- [ ] A user can choose an existing saved config from the platform library and see its metadata summary.
- [ ] A user can choose a local file for upload and see file name and size before deploy.
- [ ] A user can edit config content inline and see the draft content source reflected in the summary.
- [ ] The deploy panel captures a mount path / target file location in the same UI flow.
- [ ] A deploy request resolves the chosen source into a concrete runtime config snapshot that the backend can mount into the Workflow Pod.
- [ ] The mounted file is visible in the Pod spec through a dedicated volume mount and the Pod also receives environment variables describing the selected config and target path.
- [ ] Batch deploys preserve the same config selection across all subtasks.
- [ ] The UI makes it obvious which parts are draft local input versus previously saved platform configs.

## Goals (SLO)
- **Latency**: switching between config source modes should feel immediate on the client with no full-page reload
- **Concurrency**: changing config source mode should not trigger duplicate deploy submissions
- **Quality**: deploy-panel config source interactions are covered by focused frontend tests for mode switching and validation states
