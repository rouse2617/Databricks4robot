# Decisions

## 2026-06-02 — Separate UI polish branch

- **Context**: CYB-1559 is already carrying backend and API contract work for pod diagnostics. The user requested any new UI interaction changes to happen on a new branch in a worktree.
- **Decision**: Create `feat/CYB-1561-workflow-detail-interactions` in `/Users/rick/cyber-databrew-cyb1561` from latest `origin/dev`.
- **Alternatives**: Continue in CYB-1559 and include the UI polish there.
- **Rationale**: Keeps the UX polish independent from pod diagnostics API work and avoids mixing unrelated review scope.

## 2026-06-02 — OpenSpec approved

- **Context**: Runtime UI code requires an OpenSpec checkpoint before editing `Frontend/`.
- **Decision**: User approved with "继续".
- **Alternatives**: Stop before runtime edits.
- **Rationale**: Approval satisfies the project OpenSpec checkpoint for CYB-1561.

## 2026-06-02 — Submit PR before Cloud Run deploy

- **Context**: The frontend runtime change normally requires Cloud Run dev deployment verification before commit/push.
- **Decision**: User explicitly requested "ok提个pr 到dev"; submit the PR with local build/test and MCP evidence, without deploying Cloud Run dev in this step.
- **Alternatives**: Build and deploy a frontend dev Cloud Run revision before opening the PR.
- **Rationale**: The user wants PR review first; local MCP verification was completed on the CYB-1561 worktree at `localhost:5177`.
