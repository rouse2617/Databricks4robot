# CYB-1194 Decisions

## 2026-05-25 — Loop-mode bugfix
- **Context**: /loop autonomous mode, second iteration
- **Decision**: Remove URL-sync effect entirely rather than adding ref-based gating. Simpler, fewer moving parts.
- **Rationale**: Page state is initialized from URL on mount. Pagination onChange already writes URL. Browser back/forward isn't handled by the current code anyway (searchParams doesn't update on popstate). Removing dead code is cleaner than patching around it.
