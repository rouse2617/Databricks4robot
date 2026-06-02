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
