# CYB-1199 Decisions

## 2026-05-25 — Fix 3 bugs together in one PR
- **Context**: 3 bugs found in same code review cycle, all in delivery handler/SDK
- **Decision**: Single PR for all 3 fixes
- **Rationale**: All touch adjacent code, single deploy + verification cycle

## 2026-05-25 — SDK auto-generates idempotency key
- **Context**: Caller shouldn't be forced to provide an idempotency key
- **Decision**: `commit()` generates UUID if caller doesn't provide `idempotency_key`
- **Rationale**: Backend requires it, but SDK callers shouldn't have to manage it manually
