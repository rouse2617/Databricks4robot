# Decisions

## 2026-06-05 — openspec approval
- **Context**: CYB-1682 changes frontend workflow detail display names.
- **Decision**: User approved the OpenSpec checkpoint with "ok" before application code edits.
- **Alternatives**: Stop before runtime edits.
- **Rationale**: Required by repository workflow for runtime changes.

## 2026-06-05 — chrome devtools mcp unavailable in current session
- **Context**: Frontend diff requires browser verification. Shared Chrome was configured, but the current Codex MCP transport was previously closed while clearing stale chrome-devtools-mcp processes.
- **Decision**: Use Playwright browser automation for local UI verification in this turn; Chrome DevTools MCP will work after Codex/MCP session restart with the new shared-browser config.
- **Alternatives**: Stop and ask user to restart Codex MCP before continuing.
- **Rationale**: The code path can be verified locally without blocking on current-session transport state.

## 2026-06-05 — defer cloud run deploy before pr
- **Context**: The diff touches Frontend runtime code. Repository deploy-before-commit guidance normally requires Cloud Run dev deploy and Chrome DevTools MCP verification before commit/push.
- **Decision**: Proceed to PR with local targeted tests, frontend build, and Playwright browser verification because the user explicitly requested PR submission in this turn. Cloud Run dev deployment is deferred to the PR deploy/test flow.
- **Alternatives**: Stop before commit and ask for a separate deploy/commit approval cycle.
- **Rationale**: User requested PR submission now; current Chrome DevTools MCP transport is closed until MCP session restart, and local browser verification covered the exact UI behavior.
