# Tasks — CYB-1618

## Phase 1: Version History + Execution Filter (P0)

- [ ] 1.1 Add `GET /api/v1/pipelines/:name/versions` route + handler
- [ ] 1.2 Add `getPipelineVersions(name)` to `pipelineApi.ts`
- [ ] 1.3 `VersionBadge` — clickable version chip on pipeline cards in DeployPanel
- [ ] 1.4 `VersionHistoryDrawer` — Ant Design Drawer + Timeline listing all versions
- [ ] 1.5 Version filter dropdown in `WorkflowExecutionList` filter bar
- [ ] 1.6 API contract sync: OpenAPI + api-guide + SDK

## Phase 2: Run Parameter & Metric Recording (P0)

- [ ] 2.1 Extend `PipelineDeployment` model with `params` (JSON) and `metrics` (JSON) fields
- [ ] 2.2 Capture params on pipeline submit (from template+DAG inputs)
- [ ] 2.3 Capture metrics on workflow completion (cost, cpu/memory, duration)
- [ ] 2.4 Expose in `GET /api/v1/workflows` and `GET /api/v1/workflows/:name` responses
- [ ] 2.5 Display params + metrics in workflow detail page (metadata card)

## Phase 3: Multi-Run Comparison View (P1)

- [ ] 3.1 Add checkbox selection to execution list rows
- [ ] 3.2 `RunComparisonPanel` — side-by-side table comparing selected runs
- [ ] 3.3 Compare: version, params, metrics, cost, duration, status, node count
- [ ] 3.4 "Compare selected" action button in toolbar

## Phase 4: One-Click Rollback (P1)

- [ ] 4.1 Add `POST /api/v1/pipelines/:name/rollback` endpoint (sets active version)
- [ ] 4.2 "Rollback to v{N}" button in `VersionHistoryDrawer`
- [ ] 4.3 Confirmation modal with version info preview
- [ ] 4.4 After rollback: show success toast, next run uses rolled-back version
- [ ] 4.5 API contract sync for rollback endpoint

## Phase 5: Verification

- [ ] 5.1 `cd backend && go test ./...`
- [ ] 5.2 `cd Frontend && npm run lint && npm run build`
- [ ] 5.3 Full browser smoke test: version history → filter → comparison → rollback
- [ ] 5.4 `cd sdk && uv run pytest tests/unit/ -q`
