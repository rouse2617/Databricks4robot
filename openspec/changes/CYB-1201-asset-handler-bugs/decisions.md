# CYB-1201 Decisions

## 2026-05-25 — Batch of 6 fixes for asset handler bugs
- **Context**: 7 bugs found in asset handler code review, all Medium severity
- **Decision**: Fix 6 of 7 bugs in one PR; skip Bug 6 (design choice to not bloat response)
- **Rationale**: All touch adjacent files; single deploy + verification cycle
