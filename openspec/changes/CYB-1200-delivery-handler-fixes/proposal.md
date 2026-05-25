# CYB-1200: Fix delivery handler remaining bugs

## Summary

Fix 5 remaining bugs in `backend/internal/handlers/delivery/handler.go`:

1. **HandleCancel**: Non-TxRunner fallback (same pattern as CYB-1199 Bug 3 — data inconsistency on partial failure)
2. **HandleCancel**: Missing `cancelled_by` validation (HandleAck validates `acknowledged_by`, but HandleCancel doesn't validate `cancelled_by`)
3. **HandleCancel**: Optimistic lock conflict returns 500 instead of 409
4. **HandleAck**: Optimistic lock conflict returns 500 instead of 409
5. **One-step Commit**: Missing `CompletedAt` timestamp (delivery goes directly to Delivered status but CompletedAt is not set)

## Fixes

### Bug 1: HandleCancel non-TxRunner fallback
- Remove non-TxRunner fallback, always use `h.repo.(repository.TxRunner)` 
- Match the pattern from CYB-1199 HandleAddItems fix

### Bug 2: HandleCancel missing cancelled_by validation
- Add `strings.TrimSpace` + empty check before using `req.CancelledBy`, matching HandleAck's pattern for `acknowledged_by`

### Bug 3 & 4: Optimistic lock 500→409 in HandleCancel and HandleAck
- Wrap `h.repo.Update()` error check with `errors.Is(err, repository.ErrOptimisticLock)` → return 409 Conflict
- Match the pattern from CYB-1199 HandleAddItems fix

### Bug 5: One-step commit missing CompletedAt
- Set `d.CompletedAt = &now` when creating delivery in the one-step commit handler (alongside `DeliveredAt`)

## Verification

1. `go build ./...` passes
2. `go test ./internal/handlers/delivery/...` passes
3. Deploy dev + curl verify
