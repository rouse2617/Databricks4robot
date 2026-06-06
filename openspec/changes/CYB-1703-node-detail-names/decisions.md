# Decisions — CYB-1703

## 2026-06-06 — Browser target blocked by Argo timeout
- **Context**: Frontend dev Worker deployed successfully and Chrome DevTools MCP could log in and open the target route.
- **Decision**: Record targeted browser verification as blocked rather than marking it passed.
- **Alternatives**: Keep retrying the same page indefinitely, or verify against mocked browser state.
- **Rationale**: The page repeatedly failed before rendering workflow detail because the backend returned `加载工作流失败` with `dial tcp 10.2.1.211:2746: i/o timeout` from the Argo API. This is outside the frontend display-name change. Unit, lint, build, and Worker deploy verification passed.

## 2026-06-06 — Avoid transient external workflow state
- **Context**: After the Argo timeout was fixed, the target route initially rendered before DataBrew run context finished loading, briefly showing `外部 Workflow` and the no-asset-details alert.
- **Decision**: Treat run events, asset-node rows, and their loading flags as a three-state DataBrew context: linked, syncing, or external.
- **Alternatives**: Leave the transient state as-is, or block the entire page until all DataBrew context APIs finish.
- **Rationale**: The workflow DAG can render from Argo first, but the UI should not show a negative external-workflow conclusion while the run ledger and asset-node calls are still in flight.
