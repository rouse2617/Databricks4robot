# Tasks — CYB-1532

## Context files
- `backend/internal/usecase/pipeline/usecase.go` — deploy, asset injection, deployment status refresh, lineage side effects
- `backend/internal/handlers/pipeline/handler.go` — pipeline/deployment HTTP surface
- `backend/internal/models/pipeline.go` — current template and deployment model
- `Frontend/src/pages/PipelinePage.tsx` — design, template, execution, component tabs
- `Frontend/src/components/pipeline/DeployPanel.tsx` — saved templates, run history, asset modal
- `Frontend/src/components/pipeline/AssetPicker.tsx` — asset search and multi-select
- `Frontend/src/components/workflow` and `Frontend/src/pages/WorkflowDetailPage.tsx` — workflow node logs/resource UI

## Discovery
- [x] Run component registration -> drag/drop -> save template on dev.
- [x] Run a saved template without selected assets and inspect failure mode.
- [x] Create a valid output-writing component and run it with three assets.
- [x] Verify node detail exposes pod, host, status, logs, and resource duration.
- [x] Record UI/UX findings in `flow-audit.md`.

## OpenSpec Checkpoint
- [x] Write `proposal.md`.
- [x] Write `design.md`.
- [x] Write `specs/pipeline/spec.md`.
- [x] Write `context-files.md`.
- [x] Stop for user confirmation before runtime code edits.

## Implementation
- [x] [backend] Add execution target read model and default dev target.
- [x] [backend] Add pipeline run-compatible API shape and map current deployment data into it.
- [x] [backend] Preserve selected asset IDs in existing deployment payloads and return target metadata for new runs.
- [x] [backend] Reuse existing Argo refresh/node detail path for pod/log/resource metadata.
- [x] [backend] Add validation and clear error responses for unknown targets.
- [x] [Frontend] Split the pipeline page into clear component library, template management, run records, and run submission surfaces.
- [x] [Frontend] Replace hidden dropdown asset run with explicit run modal including assets and execution target.
- [x] [Frontend] Display asset count, target, workflow, run status, and linked asset list in executions.
- [x] [Frontend] Keep node logs/resource details available from the workflow/run detail flow.
- [x] [Frontend] Add component authoring hints for required output files when outputs are declared.
- [x] [backend/frontend] Mark active deployments as `Expired` when the Argo Workflow CR has been TTL-cleaned, and allow retry.
- [x] [Frontend] Support legacy `?tab=templates` links by routing them to the pipeline management tab.
- [x] [Frontend] Move run history out of the saved-pipeline management tab and keep execution operations in the execution records tab.
- [x] [Frontend] Expand the execution records tab to use the full desktop work area instead of a narrow centered panel.
- [x] [Frontend] Fix first-drop reliability by allowing the canvas wrapper to accept component drag/drop and using a matching copy drop effect.
- [ ] [follow-up, migration approval required] Persist first-class `execution_targets`, `pipeline_runs`, and `pipeline_run_nodes` tables instead of compatibility mapping over `pipeline_deployments`.
- [ ] [follow-up] Add first-class asset existence validation once the run API owns asset batch submission end-to-end.
- [ ] [follow-up] Replace whole-log fetch with bounded tail/pagination/streaming for large pod logs.

## API Contract Sync
- [x] Update `api/openapi.yaml` for any new/changed run or execution-target endpoints.
- [x] Update `docs/review/api-guide.md` with curl examples and error cases.
- [x] Update `sdk/src/cyber_databrew_sdk/` if the new REST surface is public SDK scope.
- [x] Add or update `sdk/tests/unit/` if SDK methods are added.
- [x] Add targeted smoke coverage in `scripts/api-guide-smoke.sh` or a feature smoke script.
- [x] Update frontend API types and clients before handler implementation for new endpoints.

## Verification
- [x] Backend targeted tests for target listing, run creation, target validation, and deployment mapping.
- [x] Frontend tests for run modal, asset selection, no-asset run, target rendering, execution list/detail, and output contract hint.
- [x] Tier L local verification before dev deploy because this affects API and frontend workflow.
- [ ] Deploy dev and verify with Chrome DevTools MCP: create component, save pipeline, run with assets, inspect execution, inspect logs/resources.

### Local verification — 2026-06-02
- `cd backend && go test ./...` — PASS
- `cd Frontend && npm run lint` — PASS with 2 existing warnings (`WorkflowYamlViewer.test.tsx`, `WorkflowDagNode.css`)
- `cd Frontend && npm run test -- --run src/components/pipeline/ComponentManager.test.tsx src/components/pipeline/DeployPanel.test.tsx src/pages/PipelinePage.test.tsx` — PASS, 56 tests
- `cd Frontend && npm run build` — PASS with existing large chunk warning
- `cd sdk && uv run pytest tests/unit/ -q` — PASS, 178 tests
- `git diff --check` — PASS
- `go test ./internal/usecase/pipeline/...` — PASS after `Expired` status fix
- `cd Frontend && npm run test -- --run src/pages/PipelinePage.test.tsx src/components/pipeline/DeployPanel.test.tsx` — PASS, 41 tests after tab alias fix
- `cd backend && go test ./...` — PASS after follow-up fixes
- `cd Frontend && npm run lint` — PASS with 2 existing warnings (`WorkflowYamlViewer.test.tsx`, `WorkflowDagNode.css`)
- `cd Frontend && npm run test -- --run src/components/pipeline/DeployPanel.test.tsx src/pages/PipelinePage.test.tsx` — PASS, 39 tests after run-history/tab-width UX fix
- `cd Frontend && npm run build` — PASS with existing large chunk warning after run-history/tab-width UX fix
- Chrome DevTools MCP on local `http://localhost:5176/pipeline?tab=pipelines` — PASS, saved-pipeline tab no longer renders `运行历史`; the toolbar `执行记录` button navigates to the execution records tab.
- Chrome DevTools MCP on local `http://localhost:5176/pipeline?tab=executions` — PASS, execution content uses the full desktop work area (`3180px` at `3440px` viewport) instead of the prior centered narrow panel.
- Chrome DevTools MCP on local `http://localhost:5176/pipeline?tab=pipelines`, `?tab=components`, and `?tab=executions` — PASS, management tab content now uses the full main work area (`3180px` at `3440px` viewport) instead of the centered `1280px` panel.
- Chrome DevTools MCP on local `http://localhost:5176/pipeline` — PASS, simulated component drag/drop added three nodes, toolbar deploy became enabled, `POST /api/v1/pipelines` returned `201`, and the saved template appeared in the pipeline management tab.

### Dev deploy record — 2026-06-02
| Service | Image tag | Cloud Run revision | URL |
|---------|-----------|-------------------|-----|
| backend-dev | `cyber-databrew-backend:8d4ccd1-cyb1532-r2` | `cyber-databrew-backend-dev-00413-887` | `https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app` |
| frontend-dev | `cyber-databrew-frontend:8d4ccd1` | `cyber-databrew-frontend-dev-00297-c9q` | `https://cyber-databrew-frontend-dev-wtttm6suaq-uc.a.run.app` |

### Dev verification notes — 2026-06-02
- `GET /api/v1/execution-targets` on backend dev — PASS, returned default target `default/cyber-databrew-dev`.
- Chrome MCP opened `/pipeline`, verified the run modal exposes asset selection and execution target selection, and verified execution history renders asset count, target, workflow name, and asset list links.
- Screenshots captured:
  - `openspec/changes/CYB-1532-asset-driven-pipeline-mvp/deploy-verify-pipeline-initial.png`
  - `openspec/changes/CYB-1532-asset-driven-pipeline-mvp/deploy-verify-asset-list-modal.png`
- Console had no JavaScript runtime errors; one existing browser issue reported a form field without `id`/`name`.
- Remaining workflow detail/log/resource verification is blocked in dev because backend Cloud Run points Argo requests at `ARGO_BASE_URL=https://cyber-databrew-pipeline-ui-dev-234851712830.us-central1.run.app`, which returns `UNAUTHORIZED`. Direct port-forward to `svc/argo-server` in `cyber-databrew-dev` returns 200 for `GET /api/v1/workflows/cyber-databrew-dev`, confirming the Argo Server and namespace are healthy.
- Per user request on 2026-06-02, the PR is opened before completing the remaining dev workflow verification so the user can test.
