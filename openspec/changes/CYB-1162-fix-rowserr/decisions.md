## 2026-05-24 — OpenSpec approved
- **Context**: Code review of dev branch found missing `rows.Err()` in `buildLineageResponse` (lineage_response.go)
- **Decision**: Fix by adding `rows.Err()` checks and scan-error logging after each `rows.Next()` loop; add unit tests
- **Alternatives**: Refactor to use shared row-iteration helper — deferred; scope is too small
- **Rationale**: Matches the same fix pattern from commit `7f0cf5e` for audit search; minimal change with no HTTP API impact
