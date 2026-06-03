# CYB-1569 — Pod Terminal Exec And Debug Workspace

## Problem

Pipeline users can now inspect node logs and Pod diagnostics, but they still cannot perform controlled in-Pod debugging from DataBrew. When a node is stuck or fails due to environment, file, dependency, or runtime state, the current workflow forces engineers to leave DataBrew and use direct Kubernetes tooling. That breaks the product model and does not give DataBrew a durable audit trail for who debugged which run/node/pod.

A raw cluster shell is not acceptable. DataBrew must provide a narrow, auditable Pod terminal experience that is scoped to one workflow node Pod, enabled only by execution target policy, and safe for GCP/GKE multi-namespace operation.

## Goals

- Add a controlled terminal/debug workspace for a selected workflow node Pod.
- Require execution target policy to explicitly enable Pod exec.
- Create a backend-owned terminal session before any WebSocket attach; the browser never receives Kubernetes credentials.
- Proxy stdin/stdout/stderr/resize over WebSocket to Kubernetes Pod exec.
- Audit session lifecycle and commands at DataBrew level.
- Surface clear unavailable/forbidden/expired states in the node runtime panel.
- Enforce timeout, max session duration, and command policy.
- Keep cluster/namespace isolation aligned with the selected run execution target.

## Non-Goals

- No unrestricted cluster shell.
- No browser-side Kubernetes token, kubeconfig, or service-account credential exposure.
- No terminal for completed/deleted Pods in the first slice.
- No generic namespace pod picker in the first slice.
- No multi-cluster agent architecture in this slice; use the existing backend-to-GKE access path and execution target policy.
- No approval workflow UI in the first slice; approval may be a later extension of command policy.

## Proposed Scope

### Backend

- Add terminal session lifecycle APIs:
  - create a Pod exec session for workflow name + node id + optional container + command preset
  - return session metadata and one-time attach URL
  - attach WebSocket stream for an existing session
  - terminate a session
- Resolve workflow node -> Pod name using existing workflow/detail and Pod diagnostics logic.
- Validate session against run execution target snapshot:
  - target exists
  - target namespace matches workflow/run namespace
  - target exec policy is enabled
  - target command policy allows the requested command
- Implement Kubernetes exec proxy with backend-owned credentials.
- Persist session metadata for audit and UX recovery.
- Append run events for session started/attached/ended/failed where a pipeline run is known.
- Write audit events for user/session/pod/command lifecycle.

### Frontend

- Replace the current runtime-tab placeholder with a terminal launch area.
- Show target policy state: enabled, disabled, forbidden, Pod unavailable, completed Pod unsupported.
- Create a terminal session before opening a terminal.
- Use xterm.js or equivalent terminal emulator only in the terminal panel, not as a general log viewer.
- Show session timer, command preset, target pod/container, and disconnect/terminate controls.
- Make errors actionable and non-technical where possible.

### API / SDK / Docs

- Add OpenAPI paths and schemas for session create, session attach URL metadata, terminate, and status.
- Add API guide examples for allowed, disabled, forbidden, and unavailable cases.
- Add smoke script for disabled-target response and one happy-path preflight/status response.
- Add SDK methods for session create/status/terminate if the API is public.

## Risks

- Exec is high-risk operationally; policy and audit must be part of the first slice, not follow-up.
- Browser WebSocket auth cannot rely on arbitrary headers, so session attach needs a one-time backend-issued token or equivalent scoped attach id.
- Kubernetes exec protocols are streaming and stateful; backend must clean up sessions reliably when clients disconnect.
- Finished Pods may not support exec; first slice should disable terminal for terminal Pods and point users to logs/diagnostics.

## Success Criteria

- For a running allowed Pod, the user can open a terminal, run allowed commands, resize, and terminate the session.
- For disabled/forbidden/unavailable states, UI gives a clear reason and no credential leaks.
- Every session creates durable audit/run-event records.
- Sessions are scoped to one workflow node Pod and cannot be retargeted by changing client payloads.
- Sessions auto-expire and cannot be reused after termination or timeout.
