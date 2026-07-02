# Proposal — CYB-1535 Bounded Logs And Resources

## Problem
Workflow node log reads can load full pod logs into memory. A workflow with very large logs can exhaust backend memory or block the UI. The SSE log handler exists but is not registered consistently, and resource usage currently mixes Argo resource duration with manifest requests/limits while the API guide describes live usage.

## Goals
- Make log reads bounded by default using `tailLines` and `limitBytes`.
- Register a canonical log stream endpoint.
- Avoid loading full pod logs in fallback paths.
- Clarify resource usage semantics and add workflow/node resource endpoints suitable for run detail panels.
- Keep existing workflow log and deployment resource endpoints compatible.

## Non-Goals
- Persisting arbitrary historical log pages in the first pass.
- Implementing a full log warehouse.
- Replacing Argo workflow detail rendering.
- Frontend redesign beyond API compatibility updates.

## Acceptance Criteria
- `GET /api/v1/workflows/{name}/logs` defaults to a bounded tail response.
- Server-side max values prevent unbounded log reads.
- `GET /api/v1/workflows/{name}/logs/stream` is registered and documented.
- SSE events are structured and cancellation-safe.
- Resource endpoints distinguish live Argo status, manifest-derived requests/limits, and unavailable live metrics.
- OpenAPI and api-guide no longer imply live CPU/memory metrics unless the backend actually provides them.
