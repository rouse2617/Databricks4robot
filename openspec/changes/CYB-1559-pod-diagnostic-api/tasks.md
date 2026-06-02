# Tasks — CYB-1559

## OpenSpec
- [x] Create proposal, design, tasks, and spec delta.
- [x] Record OpenSpec checkpoint approval.

## Backend
- [x] Rebase/port Pod diagnostic implementation onto latest `origin/dev` without reverting pipeline template version snapshots.
- [x] Add Kubernetes Pod client with optional startup configuration and namespace-scoped reads.
- [x] Add workflow node Pod diagnostics handler.
- [x] Register `GET /api/v1/workflows/{name}/nodes/{nodeId}/pod`.
- [x] Map Kubernetes not found, forbidden/unauthorized, and unavailable errors to distinct HTTP responses.
- [x] Add targeted handler and Kubernetes client tests.

## API Contract
- [x] Add OpenAPI path and schemas for node Pod diagnostics.
- [x] Update API guide with curl examples and error behavior.
- [x] Add smoke coverage for at least one unavailable/error path and document the happy path.
- [x] Defer Python SDK client for this endpoint; recorded in `decisions.md`.

## Frontend
- [x] Add typed API client for node Pod diagnostics.
- [x] Update Pod tab to fetch diagnostics, preserve node fallback data, and surface unavailable states clearly.
- [x] Add targeted frontend tests for fallback/error rendering where practical.

## Verification
- [x] `cd backend && go test ./internal/handlers/workflow ./internal/k8s`
- [x] `cd backend && go test ./...`
- [x] `cd Frontend && npm run test -- --run src/components/pipeline/WorkflowNodeDetailPanel.test.tsx`
- [x] `cd Frontend && npm run lint`
- [x] `cd Frontend && npm run build`
- [ ] Dev UI/API verification after deploy if user asks this agent to deploy.
