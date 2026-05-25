# CYB-1199: Fix delivery critical & high bugs

## Summary

Fix 3 bugs found in code review:

1. SDK `delivery.commit()` never sends `Idempotency-Key` header → all calls fail with 400
2. `HandleDraft` sets `RequestedBy` to X-Request-ID (UUID trace ID) instead of actual requester
3. `HandleAddItems` non-TxRunner path commits items before Update leaves data inconsistent on version conflict

## Fixes

### Bug 1: SDK missing Idempotency-Key
- Add `idempotency_key` parameter to `commit()` method
- Generate a UUID if caller doesn't provide one
- Send as `Idempotency-Key` header

### Bug 2: HandleDraft RequestedBy is X-Request-ID
- Swap: use `req.Owner` first, fall back to X-Request-ID
- Fix the comment to match the code

### Bug 3: HandleAddItems non-TxRunner inconsistency
- Simply use `h.repo` as TxRunner directly without the non-TxRunner fallback, or make the non-TxRunner path use a single atomic operation
- The simplest fix: always use `h.repo.(repository.TxRunner)` since the repo already implements it

## Verification

1. `cd backend && go test ./...` passes
2. `cd sdk && uv run ruff check src/` passes
3. `cd sdk && uv run pytest tests/unit/` passes (if delivery tests exist)
4. Deploy dev + curl verify
