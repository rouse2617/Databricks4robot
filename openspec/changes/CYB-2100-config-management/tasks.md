# Tasks — CYB-2100 Config Management

## Context files
- `Frontend/src/components/pipeline/DeployPanel.tsx` # deploy-time control surface
- `Frontend/src/components/pipeline/DeployPanel.test.tsx` # deploy panel regression coverage
- `Frontend/src/api/pipelineConfigs.ts` # saved config data source
- `Frontend/src/api/pipelineApi.ts` # single-run deploy payload
- `Frontend/src/api/deployPipelineRun.ts` # deploy orchestration
- `Frontend/src/api/batchJobApi.ts` # batch deploy payload
- `backend/internal/handlers/pipeline/handler.go` # single-run deploy handler
- `backend/internal/handlers/backfill/handler.go` # batch deploy handler
- `backend/internal/usecase/pipeline/usecase.go` # runtime config resolution and workflow creation
- `backend/internal/usecase/backfill/usecase.go` # batch propagation
- `backend/internal/transpiler/*` # workflow mount/env projection
- `openspec/changes/CYB-2100-config-management/design.md` # source-mode and mount-path decisions

## Implementation
- [ ] [Frontend] Add a deploy-panel config section with three source modes: saved config, local upload, and inline editor.
- [ ] [Frontend] Add saved-config picker UI that shows selected config name, lifecycle, current version, and owner summary.
- [ ] [Frontend] Add local upload UI with filename, size, and replace/remove actions in draft state.
- [ ] [Frontend] Add inline editor UI with helper copy that the content is request-scoped runtime input.
- [ ] [Frontend] Add mount-path and target filename inputs in the same config section.
- [ ] [Frontend] Add summary copy so users can review source type and mount target before deploy.
- [ ] [Frontend] Serialize the selected config source into single-run deploy payloads.
- [ ] [Frontend] Serialize the selected config source into batch-job create payloads.
- [ ] [Backend] Accept the deploy-time config selection contract on single-run and batch-create endpoints.
- [ ] [Backend] Resolve saved-config references to immutable version content and validate upload/inline draft content.
- [ ] [Backend] Project the selected config into workflow generation as a mounted runtime file.
- [ ] [Backend] Inject companion environment variables that describe the mounted config path and selected source.
- [ ] [Backend] Propagate the same config selection through batch subtasks.

## API Contract Sync
- [ ] [backend] Update `api/openapi.yaml` for deploy and batch-create payload changes.
- [ ] [docs] Update `docs/review/api-guide.md` with deploy examples for saved/upload/inline config sources.
- [ ] [smoke] Add or extend smoke coverage for the updated deploy contract.

## Verification
- [ ] [Frontend] Add deploy-panel tests for switching source modes and rendering the correct editor.
- [ ] [Frontend] Add deploy-panel tests for saved-config summary, upload draft metadata, and inline draft summary.
- [ ] [Frontend] Add tests that the selected config is serialized into deploy requests.
- [ ] [Backend] Add unit tests for config snapshot resolution and workflow mount/env projection.
- [ ] [Frontend] Run frontend build and targeted tests.
- [ ] [Backend] Run targeted Go tests for pipeline/backfill/transpiler paths.

## Deploy verification
- [ ] [deploy] Run local latest-dev frontend against dev backend and verify the deploy panel shows all three config source modes.
- [ ] [deploy] Verify saved-config mode can load platform config metadata without console errors.
- [ ] [deploy] Verify upload and inline modes render local draft state and mount-path summary without breaking existing deploy controls.
- [ ] [deploy] Launch a pipeline with each config source mode and verify in browser/runtime evidence that the Pod spec contains the mounted file and companion environment variables.
- [ ] [deploy] Verify batch-created subtasks inherit the same config mount contract.
