# Decisions — CYB-1559

## 2026-06-02 — OpenSpec checkpoint approved
- **Context**: Runtime feature adds a workflow node Pod diagnostics API and frontend Pod tab integration.
- **Decision**: User approved the OpenSpec checkpoint with "ok".
- **Alternatives**: Continue without checkpoint, which would violate repository workflow.
- **Rationale**: This keeps CYB-1559 traceable before editing backend/frontend runtime code.

## 2026-06-02 — Defer SDK client for pod diagnostics
- **Context**: CYB-1559 adds a backend-mediated Pod diagnostics endpoint used by the DataBrew frontend runtime environment tab.
- **Decision**: Defer Python SDK coverage for this endpoint in this PR; keep OpenAPI, API guide, smoke script, backend tests, and frontend typed client in scope.
- **Alternatives**: Add `client.workflows.pod_diagnostics(...)` and SDK unit tests now.
- **Rationale**: The current user-facing need is UI debugging inside DataBrew. SDK access is optional and can be added later if external automation needs Pod diagnostics.

## 2026-06-02 — Preserve workflow/node lookup before Kubernetes availability checks
- **Context**: PR #90 review suggested checking `podClient == nil` before loading the Argo workflow to avoid an unnecessary Argo request.
- **Decision**: Keep the existing order: validate workflow and node first, then return `K8S_UNAVAILABLE` only after a pod-backed node is resolved.
- **Alternatives**: Return `K8S_UNAVAILABLE` immediately whenever the Kubernetes client is not configured.
- **Rationale**: The endpoint contract is more useful when missing workflows/nodes return `WORKFLOW_NOT_FOUND` / `NODE_NOT_FOUND` even in local or degraded Kubernetes setups. `TestGetNodePodDiagnostics_WorkflowNotFoundBeforeKubernetesAvailability` locks this behavior.

## 2026-06-02 — Support CA data for the default Kubernetes target
- **Context**: Cloud Run dev could reach the GKE API endpoint with the configured bearer token, but Pod diagnostics failed TLS verification because file-mounted CA configuration is brittle in Cloud Run.
- **Decision**: Add `K8S_CA_DATA` support for the current default Kubernetes target while keeping `K8S_CA_FILE` as a fallback.
- **Alternatives**: Continue requiring `K8S_CA_FILE`, or skip TLS verification in dev.
- **Rationale**: Inline CA data is easier to supply through Secret Manager and keeps TLS verification enabled. This is a compatibility step for the default target, not the long-term multi-cluster model; future `execution_targets` should store target-specific endpoint/auth/CA secret references and construct clients per target.
