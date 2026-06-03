# Tasks — CYB-1565

## Implementation
- [x] [backend] Add asset-node, cost summary, notification candidate, and watcher state models.
- [x] [backend] Add approved migration for asset-node, notification candidate, and watcher state tables.
- [x] [backend] Add repository interfaces and Postgres implementations.
- [x] [backend] Derive/upsert asset-node snapshots during run refresh/watcher sync.
- [x] [backend] Add cost summary usecase based on node and asset-node estimated costs.
- [x] [backend] Add notification candidate creation for failed/error run events.
- [x] [backend] Add persisted watcher state and bounded active-run scan behavior.
- [x] [backend] Add `GET /api/v1/pipeline-runs/{id}/asset-nodes`.
- [x] [backend] Add `GET /api/v1/pipeline-runs/{id}/cost-summary`.
- [x] [backend] Extend run events API with search/status/time filters.
- [x] [Frontend] Add typed clients/hooks for asset-node and cost summary APIs.
- [x] [Frontend] Replace asset-node placeholder with usable table/matrix and node action links.
- [x] [Frontend] Upgrade run events UI with filters, search, and load-more.
- [x] [Frontend] Show estimated cost source labels clearly.
- [x] [Frontend] Compact execution list and run detail observability UI to reduce redundant cards, disabled actions, and engineering copy.
- [x] [sdk] Add pipeline asset-node/cost/event filter methods and tests.

## API contract sync
- [x] `api/openapi.yaml` — new paths, schemas, and added event filters.
- [x] `docs/review/api-guide.md` — curl examples and error paths.
- [x] `sdk/src/cyber_databrew_sdk/` and `sdk/tests/unit/` — public methods.
- [x] `scripts/smoke-pipeline-observability-dev.sh` — asset-node, cost summary, events filters.
- [x] `openspec/changes/CYB-1565-pipeline-observability/specs/pipeline/spec.md` — behavior delta.
- [x] [Frontend] API types/hooks aligned with OpenAPI.

## Local verification
- [x] `cd backend && go test ./internal/usecase/pipeline/... ./internal/handlers/pipeline/... ./internal/postgres/...`
- [x] `cd backend && go test ./...`
- [x] `cd Frontend && npm run lint`
- [x] `cd Frontend && npm run test -- --run WorkflowDetail pipelineApi useWorkflowDetail`
- [x] `cd Frontend && npm run test -- --run WorkflowExecutionList useWorkflowDetail WorkflowNodeDetailPanel`
- [x] `cd Frontend && npm run build`
- [x] `cd sdk && uv run pytest tests/unit/`
- [x] `pre-commit run --all-files`

## Deploy verification
- [ ] Apply CYB-1564 and CYB-1565 migrations to dev.
- [ ] Deploy backend and frontend dev.
- [ ] Run smoke script against dev.
- [ ] Chrome DevTools MCP: verify run detail timeline filters/load-more and asset-node table actions.
- [x] Chrome DevTools MCP local verification: `http://127.0.0.1:5179/pipeline?tab=executions` and `my-pipeline-306ecf` detail show compact status filters, inline label filter, cost `$0.0013`, no-asset node row, timeline toggle, node detail drawer, log viewer, IO tab, and no UI regression.
- [x] Chrome DevTools MCP local verification: Failed workflow `databrew-pl-6d523d56-c1-test-pipeline-njt24` shows invalid spec prominently with non-blank DAG/asset-node empty states.
- [ ] Follow-up: Pod diagnostics endpoint still returns 503 for a valid Pod node in local/dev verification; UI shows a fallback, but backend/K8s connectivity or RBAC needs separate investigation.
- [ ] Follow-up: Event search currently searches backend event message/ID fields, not translated UI labels such as “节点”; placeholder now says “搜索消息/ID”, but richer subject/type search should be designed separately.

## PR
- [ ] Include CYB-1564 and CYB-1565 in PR body.
- [ ] Explain estimated-vs-billing cost limitation.
- [ ] Mark exact dev migration and deploy verification evidence.
