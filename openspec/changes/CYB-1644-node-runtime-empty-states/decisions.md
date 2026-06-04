## 2026-06-04 — OpenSpec checkpoint approved

- **Context**: CYB-1644 changes frontend runtime surfaces in `WorkflowNodeDetailPanel` and `WorkflowDetailPage`, so repository rules require OpenSpec artifacts before editing application code.
- **Decision**: Proceed with runtime frontend edits after the user approved the CYB-1644 OpenSpec checkpoint with "ok".
- **Rationale**: This keeps the change aligned with the required spec-first workflow while allowing the UI bugfix to continue in the isolated worktree.

## 2026-06-04 — Chrome MCP verification boundary

- **Context**: Repository rules require Chrome DevTools MCP verification for frontend changes after dev deployment.
- **Decision**: Use Chrome MCP to verify the deployed `frontend-dev` revision loads real execution data on `/pipeline?tab=executions`, capture console/network state, and supplement node-drawer copy coverage with focused Vitest assertions.
- **Alternatives**: Block on interactive MCP actions for node-card clicks, or claim drawer-level manual verification that did not happen.
- **Rationale**: The exposed Chrome MCP toolset in this session supports navigation, snapshots, screenshots, console, and network inspection, but not click/type interactions. The deployed revision was still validated in-browser, while the exact branch copy permutations remained covered by targeted component/page tests.

## 2026-06-04 — Local Argo port-forward for detail-page smoke

- **Context**: Local workflow detail pages initially failed with `connect: connection refused` because `ARGO_SERVER_URL=http://localhost:12746` had no active port-forward.
- **Decision**: Recreate the local `kubectl port-forward` to `svc/argo-server` in the `argo` namespace for detail-page smoke validation.
- **Alternatives**: Treat the local detail failure as an application regression or skip detail-page verification entirely.
- **Rationale**: This isolates infrastructure reachability from frontend behavior. Once the port-forward was restored, local workflow detail routes could resolve against the Argo API again.
