# Tasks — CYB-2178 Node Config Binding

## Context files
- `Frontend/src/components/pipeline/types.ts` # pipeline/node DSL types
- `Frontend/src/components/pipeline/NodeConfigPanel.tsx` # node edit surface
- `Frontend/src/components/pipeline/PipelineNode.tsx` # node card summary
- `Frontend/src/lib/pipeline-design/canvas-to-dsl.ts` # save template JSON
- `Frontend/src/lib/pipeline-design/dsl-to-canvas.ts` # reopen template JSON
- `Frontend/src/api/pipelineConfigs.ts` # config-library client
- `Frontend/src/components/pipeline/DeployPanel.tsx` # existing deploy-level config UI to de-emphasize
- `backend/internal/usecase/pipeline/usecase.go` # deploy-time config resolution/projection
- `backend/internal/k8s/runtime_config.go` # K8s ConfigMap projection
- `backend/internal/transpiler/pipeline.go` / `backend/internal/transpiler/transpiler.go` # node-local mounts/env
- `api/openapi.yaml` # payload/schema documentation
- `docs/review/api-guide.md` # user-facing API examples

## Implementation
- [x] [Frontend] Add `PipelineNodeRuntimeConfig` / node-level config binding types.
- [x] [Frontend] Load ready config-library entries for the node config panel.
- [x] [Frontend] Add a node config section that lets users select/clear a saved config, mount path, and target filename.
- [x] [Frontend] Show selected config metadata and ready-only empty/error states in the node config panel.
- [x] [Frontend] Show a compact configured badge/summary on canvas nodes.
- [x] [Frontend] Persist node `runtimeConfig` through `canvasToDesignDSL`.
- [x] [Frontend] Restore node `runtimeConfig` through `designDSLToCanvas`.
- [x] [Frontend] De-emphasize/remove deploy-level config selection from the normal saved-pipeline run modal.
- [x] [Backend] Add node-level runtime config structs to transpiler/pipeline model.
- [x] [Backend] Resolve every node saved-config reference during deploy.
- [x] [Backend] Reject missing, inaccessible, draft, deprecated, or invalid node config references before Argo submission.
- [x] [Backend] Create runtime config projection containing all node config file contents for one run.
- [x] [Backend] Apply config volume mounts and `PIPELINE_CONFIG_*` env vars only to the configured node.
- [x] [Backend] Preserve deploy-level `configSelection` compatibility and define node-level precedence.
- [x] [Backend] Preserve deploy-level asset env injection into every node Pod (`ASSET_IDS`, `ASSET_COUNT`, indexed asset env vars).
- [x] [Frontend] Keep asset selection in the deploy/run dialog and avoid presenting asset IDs as per-node configuration.

## API Contract Sync
- [x] [api] Update `api/openapi.yaml` for optional node `runtimeConfig` fields in pipeline template JSON and deploy examples.
- [x] [docs] Update `docs/review/api-guide.md` with node-level config binding examples and compatibility notes.
- [x] [smoke] Add or extend smoke coverage for a node-configured pipeline deploy.
- [x] [Frontend] Ensure typed frontend API/DSL types match the OpenAPI contract.

## Verification
- [ ] [Frontend] Add tests for node config panel selecting, clearing, and summary rendering.
- [x] [Frontend] Add save/load round-trip tests for node `runtimeConfig`.
- [ ] [Frontend] Add tests that deploy-level config controls are not the primary normal run path.
- [x] [Backend] Add unit tests for resolving multiple node config bindings.
- [x] [Backend] Add workflow manifest tests proving two nodes get different mounted config files and env vars.
- [x] [Backend] Add workflow manifest tests proving every node still receives deploy-selected asset env vars.
- [x] [Backend] Add error-path tests for missing/not-ready config references.
- [x] [Frontend] Run targeted Vitest plus `npm run build`.
- [x] [Backend] Run targeted Go tests for pipeline usecase/transpiler/k8s runtime config.

## Deploy verification
- [x] [deploy] Apply any required migration first; none expected for this slice.
- [x] [deploy] Deploy backend dev if backend/runtime code changed.
- [ ] [deploy] Deploy frontend to `https://cyber-databrew-dev.cyberorigin.ai/` through Wrangler/CF.
- [ ] [Chrome MCP] Open Pipeline designer, bind different ready configs to two nodes, save, reopen, and confirm both bindings persist.
- [x] [Chrome MCP] Deploy the saved pipeline and confirm no console errors or unexpected deploy-level config UI confusion.
- [ ] [runtime] Inspect generated Workflow/Pod YAML to verify node A and node B have separate config mounts/env vars.
- [ ] [runtime] Verify a node without config does not receive another node's config env vars.
- [x] [runtime] Verify every node Pod receives the selected run asset IDs through env vars.

## 2026-06-18 Runtime Probe Evidence
- Backend dev revision: `cyber-databrew-backend-dev-00898-kf2`.
- Local frontend: `http://127.0.0.1:5177`, pointed at Cloud Run dev backend for faster debugging per user request.
- Probe image: `us-central1-docker.pkg.dev/green-valley-442103/cyber-databrew-images/cyb2178-probe-20260618115410:latest`.
- UI-owned ready config: `cyb2178-probe-ui-20260618115702.yaml` / `4cba82c8-00cb-4642-a1f5-1fa505e36e20`.
- UI-created saved template: `974f426d-711c-4e1d-9b11-5d3b0e14243f` v53; deploy auto-saved run template `45ad7bec-1516-4c60-b109-f07459ecb109` v1.
- Workflow/run: `pipeline-1781755203826-65750f` / `a30f5be5-8370-4303-816f-9ec1d6650a8f`, status `Succeeded`, asset `SDKT0202`.
- Pod log confirmed: `ASSET_IDS=SDKT0202`, `ASSET_COUNT=1`, `PIPELINE_CONFIG_PATH=/workspace/configs/cyb2178-probe-ui-20260618115702.yaml`, `PIPELINE_CONFIG_ID=4cba82c8-00cb-4642-a1f5-1fa505e36e20`, `CONFIG_FILE_EXISTS=true`, and config content contained `expected_asset_id: SDKT0202`.
- K8s confirmed runtime ConfigMap `runtime-config-a30f5be5-8370-4303-816f-9ec1d6650a8f` has ownerReference `pipeline-1781755203826-65750f` / workflow UID `b9bfdeee-86f7-4130-b823-b4edb7e38a0a`.
- Regression fix verified: after reopening template `974f426d-711c-4e1d-9b11-5d3b0e14243f`, selecting the canvas node and switching to the executions tab no longer triggers the false "unsaved changes" modal.
- Verification commands passed after the dirty-state fix: `npm run test -- --run src/components/pipeline/DeployPanel.test.tsx src/lib/pipelineContract.test.ts src/components/pipeline/PipelineNode.test.tsx`, `npm run build`, and `go test ./internal/usecase/pipeline ./internal/k8s ./internal/transpiler`.
- `npm run lint` is still blocked by pre-existing full-repo Biome diagnostics outside the focused change set, including historical formatting/import ordering, CSS `!important`, and existing hook dependency warnings in the large designer file.
