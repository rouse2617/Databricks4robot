# Decisions

## 2026-06-05 — Implement direct UI follow-up
- **Context**: User reported two visible UI issues while reviewing the local/dev pipeline pages.
- **Decision**: Implement the small UI follow-up in a new branch based on latest dev and keep the change frontend-only.
- **Alternatives**: Split into two separate PRs.
- **Rationale**: Both changes are narrow pipeline UI readability fixes and can be validated together with frontend lint, tests, and build.

## 2026-06-05 — Chrome DevTools MCP unavailable
- **Context**: Chrome DevTools MCP snapshot failed with transport closed.
- **Decision**: Use local Vite against dev backend, HTTP smoke, frontend lint, targeted tests, build, and pre-commit for verification.
- **Alternatives**: Wait for browser MCP transport recovery.
- **Rationale**: The dev server is available locally; automated checks cover the changed helpers and rendering tests.
