# Design — CYB-1569 Pod Terminal Exec

## User Model

The user opens an execution detail page, selects a node, opens the runtime tab, and sees a debug area:

- Pod diagnostics remains the primary context.
- Terminal is a secondary action gated by policy.
- If enabled, user selects an allowed command preset such as `sh`, `pwd`, `ls`, or `env` and opens a session.
- The terminal is visibly bound to one run, node, pod, container, namespace, and execution target.

The UI must avoid implying broad cluster access. Copy should be about "Pod terminal for this node", not "cluster shell".

## Backend Session Flow

1. Browser calls `POST /api/v1/workflows/{workflowName}/nodes/{nodeId}/terminal-sessions`.
2. Backend resolves:
   - workflow detail
   - node id
   - Pod name and container
   - matching `pipeline_run` when available
   - execution target snapshot / namespace
3. Backend validates:
   - Pod is running
   - execution target has terminal policy enabled
   - command preset is allowed
   - current user/request is authorized for this run/target
4. Backend creates a session row with status `created` and an attach token with short TTL.
5. Browser opens WebSocket `GET /api/v1/pod-terminal/sessions/{sessionId}/attach?token=...`.
6. Backend upgrades to WebSocket, marks session `attached`, and starts Kubernetes exec proxy.
7. Client sends JSON frames for input and resize; backend sends JSON frames for stdout/stderr/status.
8. Backend marks session `ended`, `timed_out`, or `failed` and writes audit/run events.

## WebSocket Frame Shape

Client to server:

```json
{"type":"stdin","data":"ls -la\n"}
{"type":"resize","cols":120,"rows":32}
{"type":"close"}
```

Server to client:

```json
{"type":"stdout","data":"..."}
{"type":"stderr","data":"..."}
{"type":"status","status":"attached"}
{"type":"exit","exitCode":0,"reason":"completed"}
{"type":"error","code":"POD_EXEC_FORBIDDEN","message":"..."}
```

Frames are intentionally small and typed. Raw binary transport can be considered later, but JSON frames are easier to audit and test in the first slice.

## Policy Model

Add terminal policy to execution targets, either as first-class columns or a JSON policy snapshot:

```json
{
  "terminal": {
    "enabled": true,
    "maxSessionSeconds": 900,
    "idleTimeoutSeconds": 120,
    "allowedCommands": ["sh", "pwd", "ls", "env"],
    "deniedPatterns": ["kubectl", "docker", "crictl", "rm -rf /", "mkfs"],
    "allowCompletedPods": false
  }
}
```

First slice should default to disabled. Dev can enable terminal on the default execution target explicitly.

## Persistence

Expected new table, name finalizable during implementation:

`pipeline_pod_terminal_sessions`

Core fields:

- `id`
- `run_id`
- `workflow_name`
- `node_id`
- `pod_name`
- `container_name`
- `execution_target_id`
- `cluster`
- `namespace`
- `command`
- `status`
- `actor`
- `created_at`
- `attached_at`
- `ended_at`
- `expires_at`
- `exit_code`
- `error_code`
- `error_message`

Do not persist terminal output in the first slice. Persist metadata and audit only. If transcript retention is later required, it should be policy-controlled and redacted.

## Audit And Run Events

Run events when run is known:

- `pod_terminal_session_created`
- `pod_terminal_session_attached`
- `pod_terminal_session_ended`
- `pod_terminal_session_failed`

Audit event examples:

- actor opened terminal for run/node/pod
- actor attached terminal websocket
- actor terminated session
- session timed out or failed

Command content should be recorded as preset/initial command only. Interactive keystrokes are not stored in first slice.

## Error Semantics

- `400 POD_EXEC_INVALID_REQUEST`: invalid command/container/node payload.
- `403 POD_EXEC_FORBIDDEN`: policy or RBAC blocks terminal.
- `404 POD_NOT_FOUND`: Pod or node cannot be resolved.
- `409 POD_EXEC_UNAVAILABLE`: Pod is completed, deleted, or not in a phase that supports exec.
- `503 K8S_UNAVAILABLE`: backend cannot reach Kubernetes API.

UI maps these to clear states and does not show raw Kubernetes credentials or bearer-token details.

## GCP / GKE Notes

The implementation uses the backend's existing GKE access mode. For Cloud Run dev/prod, the backend service account must have enough permission for Kubernetes `pods/exec` in the target namespace. This should be namespace-scoped RBAC, not cluster-admin.

For multi-cluster future work, each execution target should declare whether terminal is supported and which backend/agent path handles exec. This spec does not introduce a cluster agent, but it keeps the API target-scoped so an agent path can be added later.

## Frontend UX

In `WorkflowNodeDetailPanel` runtime tab:

- Keep Pod diagnostics at top.
- Add compact terminal policy status.
- Show command preset buttons only when enabled.
- Disable terminal for terminal Pods with a direct path to logs and Pod events.
- Terminal opens inside a drawer/panel with stable height, monospace display, session timer, and terminate button.
- After disconnect, show final status and allow reopening only by creating a new session.

## Testing Strategy

Backend:

- unit tests for policy allow/deny
- handler tests for create session error states
- fake Kubernetes exec proxy tests for frame relay and close semantics
- repository tests for session persistence if migration is added

Frontend:

- runtime tab shows disabled/forbidden/unavailable states
- create session calls API with workflow/node/container
- terminal component handles stdout/stderr/exit frames
- terminate button closes websocket and calls terminate API if needed

Dev smoke:

- disabled target returns a clear 403/409 without credentials
- allowed dev target can open a short `sh -lc 'pwd'` or equivalent preflight command if policy enables it
