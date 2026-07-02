# Decisions — CYB-1569

## 2026-06-03 — OpenSpec checkpoint approved and branch aligned
- **Context**: CYB-1569 already had proposal, design, tasks, and spec delta in the worktree. The user asked to rebase on dev and start implementation.
- **Decision**: Reset the worktree to latest `origin/dev` (`7825a5f`) and proceed with implementation from the existing OpenSpec.
- **Alternatives**: Stop again at the OpenSpec checkpoint.
- **Rationale**: The user's explicit instruction "rebase dev, 开干" approves the checkpoint and starts runtime work.

## 2026-06-03 — Runtime slice uses memory sessions and backend-owned exec
- **Context**: Durable terminal session persistence requires a migration, which is an off-limits zone without separate approval. Kubernetes exec must remain backend-owned and policy-gated.
- **Decision**: Implement the runtime slice with default-disabled policy, backend-owned in-memory sessions, one-time attach tokens, create/status/terminate APIs, frontend terminal UI, SDK/docs/smoke, and a WebSocket attach path that proxies allowed command output through client-go `remotecommand` when the backend has a Kubernetes exec client configured. If exec is not configured, attach returns a controlled `POD_EXEC_UNAVAILABLE` frame.
- **Alternatives**: Add the session table migration and full Kubernetes exec proxy immediately.
- **Rationale**: This ships the safe product/API/UX boundary without exposing credentials or enabling broad shell access. Durable persistence and fully interactive stdin/xterm remain explicit follow-up tasks.

## 2026-06-03 — WebSocket auth caveat
- **Context**: Browser WebSocket clients cannot set arbitrary `X-Databrew-Token` headers. The attach URL carries a scoped token, but the current route still sits under the existing authenticated `/api/v1` group.
- **Decision**: Keep attach under current auth for this first slice and document the caveat before enabling real exec.
- **Alternatives**: Move attach outside the auth group and rely only on session attach token validation.
- **Rationale**: Avoid weakening auth boundaries before real exec is enabled; revisit routing when enabling live Kubernetes exec streams.

## 2026-06-03 — Session first, websocket second
- **Context**: Browser WebSocket connections cannot safely carry arbitrary custom auth headers, and the browser must never receive Kubernetes credentials.
- **Decision**: Require a backend-created terminal session with short-lived attach metadata before websocket attach.
- **Alternatives**: Open a direct WebSocket with workflow/node query params, or expose a Kubernetes proxy credential to the browser.
- **Rationale**: A session-first flow gives DataBrew a durable audit anchor, scoped authorization, expiry, and one-time attach controls.

## 2026-06-03 — Terminal disabled by default
- **Context**: Pod exec is operationally sensitive and can mutate runtime state.
- **Decision**: Treat terminal policy as disabled unless explicitly enabled on the execution target.
- **Alternatives**: Enable terminal globally once backend credentials can exec into Pods.
- **Rationale**: Execution target policy is the correct boundary for cluster/namespace isolation and future multi-cluster support.
