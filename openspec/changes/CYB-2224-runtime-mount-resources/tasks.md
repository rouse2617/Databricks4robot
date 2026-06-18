# Tasks — CYB-2224

## Context files
- `backend/internal/transpiler/pipeline.go` — current pipeline DSL and `VolumeMount` model.
- `backend/internal/transpiler/transpiler.go` — Argo Workflow volume and container/script mount generation.
- `backend/internal/usecase/pipeline/usecase.go` — deploy path, runtime config resolution, asset env injection, target validation.
- `backend/internal/handlers/pipeline/handler.go` — deploy handlers and request/response mapping.
- `Frontend/src/components/pipeline/NodeConfigPanel.tsx` — node editor form surface.
- `Frontend/src/components/pipeline/types.ts` — canvas node data types.
- `Frontend/src/lib/pipeline-design/canvas-to-dsl.ts` — canvas to saved DSL conversion.
- `Frontend/src/lib/pipeline-design/dsl-to-canvas.ts` — saved DSL to canvas conversion.
- `Frontend/src/pages/WorkflowDetailPage.tsx` — execution detail DAG and node selection state.
- `Frontend/src/components/pipeline/WorkflowNodeDetailPanel.tsx` — execution node detail drawer.
- `backend/internal/usecase/pipeline/runtime_mounts.go` — DataBrew-owned runtime mount catalog and deployment-time binding resolver.

## Implementation
- [x] [backend] Add typed pipeline DSL fields for node secret mounts and storage mounts.
- [x] [backend] Extend transpiler volume model for SecretProviderClass CSI volumes.
- [x] [backend] Add unit tests for CSI secret, PVC, emptyDir, container node, and script node manifest output.
- [x] [backend] Add read-only runtime mount catalog types and resolver.
- [x] [backend] Add validation for unknown IDs, unsafe mount paths, duplicate volume names, write-mode mismatch, and unsupported target capabilities.
- [x] [backend] Add `GET /api/v1/pipeline/runtime-mounts`.
- [x] [backend] Wire deploy/template deploy to resolve node secret/storage bindings before workflow submission.
- [x] [Frontend] Add runtime mount catalog API client and types.
- [x] [Frontend] Extend pipeline node data types with secret/storage bindings.
- [x] [Frontend] Round-trip secret/storage bindings through canvas-to-DSL and DSL-to-canvas.
- [x] [Frontend] Add NodeConfigPanel sections: Configuration, Secrets, Storage.
- [x] [Frontend] Move secret mounts into a collapsed advanced section so normal node editing stays focused on config, storage, env, and resources.
- [x] [backend] Support generic environment-provided runtime mount catalogs via `PIPELINE_RUNTIME_MOUNT_CATALOG_JSON`, `PIPELINE_RUNTIME_SECRET_RESOURCES_JSON`, and `PIPELINE_RUNTIME_STORAGE_RESOURCES_JSON`.
- [x] [Frontend] Show compact node badges for selected secret/storage mounts without exposing low-level details.
- [x] [Frontend] Show selected node runtime config, secret mounts, storage mounts, and injected env names in the execution detail drawer.
- [x] [Frontend] Move deprecated pipeline configs out of the default config workspace and into an archive shelf for trace/view/diff only.
- [x] [Frontend] Add an explicit order-only edge mode for CyberPipe-style pipelines where edges control sequencing without consuming `/tmp/outputs` files.
- [x] [Frontend] Align execution detail asset-node table names with the DAG/component labels while keeping runtime step IDs as secondary diagnostics.
- [x] [backend] Revalidate generated component release digests on sync/list/get so short or dummy `sha256` values cannot stay selectable.
- [x] [Frontend] Surface asset-node runtime messages in the execution detail status column so Pending pods expose reasons such as `InvalidImageName`.
- [x] [backend] Keep pipeline deploy input as `asset_ids` while allowing UUID-shaped CyberPipe video IDs as compatible external run inputs.
- [x] [backend] Inject `REQUEST_ID` for every run and `VIDEO_ID` for single-asset/single-video pipeline runs.
- [x] [backend] Map GPU component resources to Kubernetes GPU scheduling hints, and allow execution targets to provide hidden template scheduling defaults.
- [x] [Frontend] Render non-canonical execution asset IDs as text instead of broken asset-detail links.
- [x] [Frontend] Add tests for selecting/clearing secret and storage bindings. Covered by DSL round-trip, node badge unit test, and NodeConfigPanel storage interaction test.
- [ ] [Frontend] Component release detail displays task directory from `technicalMetadata.taskDir` when `taskPath` is absent.
- [ ] [Frontend] Component release list distinguishes image tag from digest/hash identity and shows full image metadata in tooltips.
- [ ] [docs] Record component release CI ingest token as a required backend dev deploy binding.

## API contract sync
See [`docs/agents/AI-RULES.md` § API contract sync](../../docs/agents/AI-RULES.md#api-contract-sync-mandatory).

- [x] `api/openapi.yaml` — add runtime mount catalog path and response schemas.
- [x] `docs/review/api-guide.md` — add curl example and error behavior.
- [x] `scripts/api-guide-smoke.sh` — smoke the runtime mount catalog.
- [x] `openspec/changes/CYB-2224-runtime-mount-resources/specs/pipeline/spec.md` — behavior delta.
- [x] [Frontend] `Frontend/src/api` runtime mount catalog types and client.
- [x] SDK client was updated because `pipelines` is already a public SDK manager.

## Local verification
- [x] [backend] `go test ./internal/transpiler/...`
- [x] [backend] `go test ./internal/usecase/pipeline/...`
- [x] [backend] `go test ./internal/handlers/pipeline/...`
- [x] [backend] `go test ./...` before deploy.
- [x] [Frontend] `npx biome check` on touched frontend files.
- [x] [Frontend] targeted Vitest for pipeline contract and pipeline node display.
- [x] [Frontend] targeted Vitest for execution detail runtime mount display and workflow-node to pipeline-node mapping.
- [x] [Frontend] targeted Vitest for Registry Center active/archive config shelf behavior.
- [x] [Frontend] targeted Vitest for pipeline order-only edge import/export and run validation.
- [x] [backend] `go test ./internal/usecase/pipeline_component ./internal/handlers/pipeline_component`.
- [x] [Frontend] targeted Vitest for execution detail asset-node message display and workflow-node mapping.
- [x] [Frontend] targeted Vitest for execution detail external video ID rendering.
- [x] [backend] targeted Go tests for GPU scheduling hints and execution target scheduling defaults.
- [x] [Frontend] `npm run build`.
- [x] [SDK] `uv run pytest tests/unit/test_managers.py -q`.
- [x] [repo] `git diff --check`.
- [x] [repo] `pre-commit run --files` on touched files before commit.

## Deploy verification
- [ ] Apply migrations first if any are introduced; none expected for P0.
- [x] Deploy backend dev with SHA-tagged image; record image, revision, URL.
  - Image: `us-central1-docker.pkg.dev/green-valley-442103/cyber-databrew-images/cyber-databrew-backend:bbe2509-cyb2224-20260618082338`
  - Revision: `cyber-databrew-backend-dev-00904-fd2`
  - URL: `https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app`
  - Image: `us-central1-docker.pkg.dev/green-valley-442103/cyber-databrew-images/cyber-databrew-backend:049fad0-cyb2224-release-digest-20260618224454`
  - Revision: `cyber-databrew-backend-dev-00908-q6q`
  - URL: `https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app`
  - Image: `us-central1-docker.pkg.dev/green-valley-442103/cyber-databrew-images/cyber-databrew-backend:e6e120a-videoid-compat-20260618155644`
  - Revision: `cyber-databrew-backend-dev-00914-s6q`
  - URL: `https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app`
  - Image: `us-central1-docker.pkg.dev/green-valley-442103/cyber-databrew-images/cyber-databrew-backend:e6e120a-cyb2224-no-fixed-target-cap-20260618164727`
  - Revision: `cyber-databrew-backend-dev-00916-kxz`
  - URL: `https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app`
- [x] Deploy frontend dev/CF if frontend code changed; record version/revision.
  - Cloudflare Worker: `cyber-databrew-dev`
  - URL: `https://cyber-databrew-dev.cyberorigin.ai`
  - Wrangler version ID: `b1c732ca-37c5-48bf-b681-86af2f3266d8`
  - Created: `2026-06-18T08:28:56.013Z`
  - UI build ref: `vcyb-2224-dev-202606180826-runtime-mounts (feat/CYB-2224-runtime-mount-resources#bbe2509+runtime-mounts)`
  - Wrangler version ID: `e944c4b7-1ffc-49dc-afe8-eedc6bace54d`
  - UI build ref: `v049fad0 (feat/CYB-2224-runtime-mount-resources#049fad0+release-digest-message)`
  - Wrangler version ID: `49cd0b9d-2303-4ee5-9945-86725feeb907`
  - UI build ref: `ve6e120a (feat/CYB-2224-runtime-mount-resources#e6e120a)`
- [x] API smoke `GET /api/v1/pipeline/runtime-mounts` on dev.
- [x] Chrome DevTools MCP: open pipeline designer, bind one secret and one storage mount, save, and verify node badge persists as `密钥 1 / 存储 1`. Evidence: `openspec/changes/CYB-2224-runtime-mount-resources/cf-designer-secret-storage-badge.png`.
- [x] Chrome DevTools MCP: deploy a test pipeline and verify no console errors.
- [x] Chrome DevTools MCP: open execution detail drawer and verify node runtime mount visibility. Evidence: `openspec/changes/CYB-2224-runtime-mount-resources/workflow-detail-runtime-mounts.png`.
- [x] Chrome DevTools MCP on CF domain: execution detail drawer shows runtime storage mount and env names. Evidence: `openspec/changes/CYB-2224-runtime-mount-resources/cf-workflow-detail-runtime-mounts-runtime-tab.png`.
- [x] Chrome DevTools MCP on CF domain: designer node config loads runtime mount catalog and node badge shows storage count. Evidence: `openspec/changes/CYB-2224-runtime-mount-resources/cf-designer-storage-mount-badge.png`.
- [x] Chrome DevTools MCP on CF domain: Registry Center default config workspace hides archived configs; archive shelf shows only archived records with trace/view/diff actions. Evidence: `openspec/changes/CYB-2224-runtime-mount-resources/cf-registry-config-archive-shelf.png`.
- [x] Chrome DevTools MCP on CF domain: execution detail asset-node table uses the same friendly component labels as the DAG and keeps `step-node-*` as secondary diagnostics. Worker version `f3861623-58ee-4ecf-91dc-fc50c311e9ed`; UI build ref `v049fad0 (dev#049fad0+workflow-asset-node-name-fix)`. Evidence: `openspec/changes/CYB-2224-runtime-mount-resources/cf-workflow-detail-asset-node-name-alignment.png`.
- [x] Chrome DevTools MCP on CF domain: execution detail asset-node table maps real `node-*` template IDs to `step-node-*` runtime rows. Worker version `68bd48ac-40a7-4f19-ab76-44cb3aac4043`; UI build ref `v049fad0 (feat/CYB-2224-runtime-mount-resources#049fad0+workflow-asset-node-name-fix2)`. Evidence: `openspec/changes/CYB-2224-runtime-mount-resources/cf-workflow-detail-asset-node-name-alignment-fix2.png`.
- [x] Chrome DevTools MCP on CF domain: execution detail asset-node status column shows `InvalidImageName` for a Pending main container, making it clear the algorithm step is not actually running. Worker version `e944c4b7-1ffc-49dc-afe8-eedc6bace54d`; UI build ref `v049fad0 (feat/CYB-2224-runtime-mount-resources#049fad0+release-digest-message)`. Evidence: `openspec/changes/CYB-2224-runtime-mount-resources/cf-workflow-detail-pending-message.snapshot.txt`; screenshot tool timed out twice, see `decisions.md`.
- [x] Backend API smoke: `GET /api/v1/pipeline-component-releases?selectable=true&q=hand-detect-yolov26m` returns `{"items":[]}`; direct search shows the historical dummy release revalidated to `validationStatus=failed` with `imageDigest is required and must be sha256:<64 hex>`.
- [x] Runtime smoke: UI deployed `cyb2178-probe-20260618115702` as `pipeline-1781771851723-ae379d`; run succeeded in 4s and Pod diagnostics showed `Service Account = cyber-databrew-backend-argo`, mounted secret/storage rows, and normal container events. Evidence: `openspec/changes/CYB-2224-runtime-mount-resources/cf-runtime-mount-pod-detail-success.png`.
- [x] Runtime smoke: the actual Pod spec contains `PIPELINE_SECRET_DATABREW_SMOKE_SECRET_PATH`, `PIPELINE_STORAGE_SCRATCH_EMPTYDIR_PATH`, CSI driver `secrets-store-gke.csi.k8s.io`, SecretProviderClass `databrew-dev-runtime-mount-smoke`, and `emptyDir` storage mount.
- [x] Runtime smoke: verify unbound nodes do not receive another node's secret or storage mount. Dry-run two-node manifest shows `step-node-b` has only `PIPELINE_DEPLOYMENT_ID` and `volumemounts: []`.
- [x] Runtime negative smoke: unknown mount resource fails before workflow creation with `400 INVALID_ARGUMENT` and message `node node-a runtime secret does-not-exist is not available`.
- [x] Backend API smoke: `POST /api/v1/deploy?dryRun=true` with `asset_ids=["019dabf3-5685-769f-8ec3-3992767ebe65"]` returns `Preview` and manifest contains `VIDEO_ID`, `REQUEST_ID`, and `ASSET_0_ID`; the same smoke with `asset_ids=["missing-2"]` still returns `400 INVALID_ARGUMENT asset_ids contain unknown assets`.
- [x] Runtime smoke: `videoid-compat-live-smoke-dcd098` succeeded with run `287b4663-b61f-4e92-95ec-681c3a9f5ff1`; Pod logs showed `VIDEO_ID=019dabf3-5685-769f-8ec3-3992767ebe65`, `REQUEST_ID=287b4663-b61f-4e92-95ec-681c3a9f5ff1`, and `ASSET_0_ID=019dabf3-5685-769f-8ec3-3992767ebe65`.
- [x] Chrome DevTools MCP on CF domain: execution detail for `videoid-compat-live-smoke-dcd098` shows the UUID input as text, not an `/assets/<uuid>` link; all workflow/run/asset-node/cost APIs returned 200. Evidence: `openspec/changes/CYB-2224-runtime-mount-resources/cf-videoid-compat-workflow-detail.png`.
- [x] Backend API smoke after revision `00916-kxz`: `/readyz` healthy, `/api/v1/pipeline/runtime-mounts` returned configured runtime mounts, and `/api/v1/execution-targets` returned `video-proc-dev` without a hard-coded quota policy.
- [ ] Chrome DevTools MCP on CF domain: component registry shows `hand-track-stereo-databrew-test` task path, image tag, digest identity, and friendlier version label.
- [ ] DataBrew sync smoke with `X-Databrew-CI-Token` returns `{"items":[]}` after the backend revision binds `COMPONENT_RELEASE_INGEST_TOKEN` or `DATABREW_CI_INGEST_TOKEN`.
- [ ] Chrome DevTools MCP on CF domain: designer order-only edge mode exports naked edge refs and no longer shows `/tmp/outputs/output` consumption warnings.

## PR
- [ ] PR template filled with Linear `CYB-2224` and OpenSpec change-id.
- [ ] Include deploy record, Chrome MCP evidence, API smoke output, and runtime log evidence.
- [ ] Do not commit generated `site/` assets unless explicitly requested.
