# Design — CYB-1559

## Architecture Context
- DataBrew submits and reads workflows through Argo Workflows.
- Argo workflow nodes include enough information to identify the runtime Pod for a container step, but they do not provide the full Kubernetes describe/events view needed for debugging.
- The frontend must never call Kubernetes directly or receive kubeconfig/token material.
- The deployment environment is GCP/Cloud Run talking to GKE/Argo in the dev namespace.

## Affected Modules
- `backend/internal/handlers/workflow` — validates request, loads workflow from Argo, resolves node Pod, maps Kubernetes errors to DataBrew HTTP responses.
- `backend/internal/k8s` — builds Kubernetes client and reads Pod/container/event diagnostics.
- `backend/routes` — registers the node Pod diagnostics route.
- `Frontend/src/api/workflowApi.ts` — typed client method and response shape.
- `Frontend/src/components/pipeline/WorkflowNodeDetailPanel.tsx` — Pod tab consumption, fallback, and unavailable/error messaging.
- `api/openapi.yaml`, `docs/review/api-guide.md`, `scripts/` — API contract and smoke coverage.

## API Shape

`GET /api/v1/workflows/{name}/nodes/{nodeId}/pod`

Response:
- `cluster`: optional display label for the configured Kubernetes target.
- `namespace`: Kubernetes namespace.
- `podName`: Kubernetes Pod name.
- `podIp`: Kubernetes Pod IP.
- `serviceAccountName`: Pod service account.
- `restartCount`: summed restart count across containers.
- `containers`: name, image, readiness, restart count, and state.
- `podConditions`: condition type/status/reason/message/transition time.
- `podEvents`: event type/reason/message/count/first/last timestamp.

## Error Semantics
- `400 INVALID_ARGUMENT`: missing workflow name or node ID.
- `404 WORKFLOW_NOT_FOUND`: Argo workflow does not exist.
- `404 NODE_NOT_FOUND`: workflow exists but node ID is absent or cannot resolve a Pod.
- `404 POD_NOT_FOUND`: resolved Pod does not exist, usually because it has not been created or has been cleaned up.
- `403 K8S_FORBIDDEN`: Kubernetes credentials lack pod/event read permission for the namespace.
- `503 K8S_UNAVAILABLE`: Kubernetes API connectivity or client configuration is unavailable.
- `500 INTERNAL_ERROR`: unexpected backend failure.

## GCP/GKE Configuration
- The Cloud Run backend must use backend-held Kubernetes credentials and RBAC; the browser never receives direct cluster credentials.
- Preferred production path is GCP identity based access and explicit namespace-scoped Kubernetes RBAC for Pod and Event read operations.
- Long-lived bearer token environment variables are acceptable only as a temporary dev bridge if documented and stored as secrets.
- The client should be optional at startup: if Kubernetes access is not configured, normal workflow pages continue to work and the Pod diagnostics endpoint returns an unavailable response.

## Frontend Behavior
- The Pod tab fetches diagnostics when opened for a node.
- Existing Argo node fields remain visible if the Pod diagnostics API fails.
- Conditions and events use diagnostics when available, otherwise fallback to existing node data.
- The UI shows a compact warning with the error category and a refresh affordance rather than an empty table with no explanation.

## Risks / Trade-offs

| Risk | Impact | Mitigation |
|------|--------|------------|
| Cloud Run cannot reach private GKE API | Pod diagnostics appear unavailable | Return `K8S_UNAVAILABLE`; document required GCP networking/auth setup |
| RBAC only allows Pods but not Events | Conditions work, events empty or unavailable | Keep events optional and surface partial failures when needed |
| Pod is deleted by TTL | Detail page cannot show live Pod data | Preserve Argo node fallback and use future run ledger for durable history |
| Endpoint shape drifts from frontend type | UI silently drops fields | Update OpenAPI, frontend type, and smoke in same PR |
