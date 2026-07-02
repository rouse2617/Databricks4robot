# Decisions — CYB-1609

## 2026-06-03 — PR before full deploy verification
- **Context**: The change touches backend, frontend, SDK, and API docs. Local automated checks passed, and Chrome MCP verified the new frontend log viewer state against the existing local backend. Starting a full current-branch local stack was interrupted by the user, who then requested a PR.
- **Decision**: Open the PR with automated test/build evidence and partial MCP evidence, and mark full current-branch local/Cloud Run deploy verification as pending.
- **Alternatives**: Continue local full-stack restart and complete MCP validation before committing.
- **Rationale**: User explicitly asked to submit the PR now; the PR body will call out the remaining verification gap.
