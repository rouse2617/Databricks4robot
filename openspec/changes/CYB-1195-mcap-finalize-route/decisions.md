# CYB-1195 Decisions

## 2026-05-25 — Keep old route for backward compat
- **Context**: The old `/mcap/upload/finalize` route exists and the SDK may be in use.
- **Decision**: Add new RESTful route alongside old one; handler reads from URL param first, falls back to body.
- **Rationale**: The old route is documented in OpenAPI and may be used by external callers. Removing it would be a breaking change.
