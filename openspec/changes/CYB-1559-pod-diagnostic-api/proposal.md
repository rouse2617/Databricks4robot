# Proposal — CYB-1559

## Why
Workflow detail already exposes node status and logs, but the Pod tab cannot reliably show Kubernetes-level diagnostics such as container readiness, restarts, Pod conditions, and Pod events. Users need this information to debug Argo steps without leaving DataBrew or using direct cluster access.

## What Changes

### New Capabilities
- Add a backend-mediated Pod diagnostics endpoint for a workflow node.
- Resolve the Kubernetes Pod from the Argo workflow node before querying Kubernetes.
- Return Pod metadata, container statuses, Pod conditions, and Kubernetes events in a frontend-friendly response.
- Update the workflow node detail Pod tab to consume the endpoint while preserving existing Argo node data as fallback.

### Modified Capabilities
- Pod diagnostics failures are surfaced as unavailable/RBAC/network states instead of being collapsed into a generic "Pod not found".
- API contract files and smoke coverage document the new endpoint in the same change.
- The branch is rebased onto latest `dev` so pipeline template version snapshots are preserved.

## Impact
- **Affected code**: `backend/internal/handlers/workflow`, `backend/internal/k8s`, `backend/routes`, `api/openapi.yaml`, `docs/review/api-guide.md`, `scripts/`, `Frontend/src/api/workflowApi.ts`, `Frontend/src/components/pipeline/WorkflowNodeDetailPanel.tsx`
- **New API**: `GET /api/v1/workflows/{name}/nodes/{nodeId}/pod`
- **Dependencies**: Kubernetes client-go for Pod/event reads
- **No schema changes**: this does not add pipeline run ledger persistence

## Scope
- **In scope**: Pod diagnostic read API, GCP/GKE-compatible configuration guidance, route registration, typed frontend consumption, fallback UI, API guide/OpenAPI/smoke updates, and targeted tests.
- **Out of scope**: Pod exec terminal, live log streaming changes, metrics/cost APIs, multi-cluster execution target persistence, and database migrations.

## Success Criteria
- [ ] A workflow node Pod tab can show Pod namespace, name, IP, service account, restart count, container statuses, conditions, and events when Kubernetes access is available.
- [ ] If Kubernetes access is unavailable or unauthorized, the UI still shows existing node metadata and a clear unavailable state.
- [ ] Backend errors distinguish workflow missing, node missing, Pod missing, Kubernetes auth/RBAC failure, and Kubernetes connectivity/configuration failure.
- [ ] The feature branch does not remove or regress pipeline template version snapshot behavior from current `dev`.
- [ ] OpenAPI, API guide, and smoke coverage describe the new endpoint.
