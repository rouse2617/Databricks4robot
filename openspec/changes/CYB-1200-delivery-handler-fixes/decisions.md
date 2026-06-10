# CYB-1200 Decisions

## 2026-05-25 — Fix 5 bugs together in one PR
- **Context**: 5 bugs found in same code review cycle, all in delivery handler
- **Decision**: Single PR for all 5 fixes
- **Rationale**: All touch adjacent code in the same file, single deploy + verification cycle

## 2026-05-25 — Skipped HandleRetry TenantID/ProjectID bug
- **Context**: Previous review flagged HandleRetry losing TenantID/ProjectID
- **Decision**: Not a real bug — these model fields exist but are never set by any HTTP handler path (draft, commit, etc.), so HandleRetry not copying them is consistent behavior
- **Rationale**: Fixing HandleRetry to copy fields that no handler sets would be meaningless; the root issue (field not set at creation) affects all handlers equally
