# Tasks — CYB-1344

## Context files
- `backend/internal/handlers/workflow/handler.go` — workflow detail response shaping
- `backend/internal/handlers/workflow/handler_test.go` — workflow handler tests
- `Frontend/src/pages/WorkflowDagView.tsx` — React Flow DAG builder and rendering
- `Frontend/src/api/workflowApi.ts` — workflow API types
- `api/openapi.yaml` — workflow detail API contract
- `docs/review/api-guide.md` — public API usage guide
- `databrew-pipeline/argo-ui/src/workflows/components/workflow-dag/workflow-dag.tsx` — upstream reference for Argo DAG graph construction

## Implementation
- [x] [backend] Add `WorkflowDagEdge` response model and normalized edge generation for workflow detail.
- [x] [backend] Preserve runtime `children` edges and add logical DAG edges for omitted/skipped downstream nodes.
- [x] [backend] Filter or compress edges so both endpoints reference nodes the frontend can render.
- [x] [backend] Add focused handler tests for failed upstream plus omitted downstream workflow nodes.
- [x] [Frontend] Add `WorkflowDagEdge` and `WorkflowDetail.edges` types.
- [x] [Frontend] Update `WorkflowDagView` to prefer backend-provided edges and keep local inference as fallback.
- [x] [Frontend] Add focused DAG rendering/builder tests for backend-provided edges.

## API contract sync (mandatory if HTTP API added/changed — same PR)
See [`docs/agents/AI-RULES.md` § API contract sync](../../docs/agents/AI-RULES.md#api-contract-sync-mandatory).

- [x] `api/openapi.yaml` — add `WorkflowDagEdge` and `WorkflowDetail.edges`
- [x] `docs/review/api-guide.md` — document workflow detail edges and omitted-step behavior
- [x] `sdk/src/cyber_databrew_sdk/` — confirm no SDK client impact or update if workflow detail is exposed
- [x] `scripts/api-guide-smoke.sh` or targeted smoke — verify workflow detail response includes `edges`
- [x] `openspec/changes/CYB-1344-workflow-dag-edges/specs/pipeline/spec.md` — behavior delta
- [x] [Frontend] `src/api/workflowApi.ts` — typed client response aligned with OpenAPI

## Local verification (Tier L per AI-RULES: HTTP API + Frontend)
- [x] Backend targeted tests: `cd backend && go test ./internal/handlers/workflow/...`
- [x] Backend full tests: `cd backend && go test ./...`
- [x] Frontend targeted tests: `cd Frontend && npm run test -- --run WorkflowDagView.test.tsx`
- [x] Frontend build: `cd Frontend && npm run build`
- [x] Frontend touched-file lint: `cd Frontend && npx biome check src/pages/WorkflowDagView.tsx src/pages/WorkflowDagView.test.tsx src/api/workflowApi.ts src/pages/WorkflowDetailPage.tsx`
- [x] SDK tests: `cd sdk && uv run pytest tests/unit/ -q`
- [x] OpenAPI YAML parse
- [ ] Full frontend test suite: attempted; blocked by pre-existing unrelated failures in PipelinePage, ComponentManager, SettingsPage, and other tests.
- [ ] Full frontend lint: attempted; blocked by pre-existing unrelated Biome issues outside touched files.
- [ ] SDK full ruff on `src/`: attempted; blocked by pre-existing `search.py` builtin shadowing.

## Deploy verification (before commit — runtime only)
- [ ] Deploy backend dev after code change.
- [ ] Deploy frontend dev after code change.
- [ ] Chrome DevTools MCP on dev: open failed two-step workflow detail and verify the failed node connects to the omitted node.
- [ ] Screenshot: `deploy-verify-workflow-dag-edges.png` in this change dir.
- [ ] Console: no new errors.
- [ ] Backend smoke: fetch `GET /api/v1/workflows/{name}` and assert non-empty `edges` for the target workflow.

## PR
- [ ] PR template filled; Linear CYB-1344 linked
