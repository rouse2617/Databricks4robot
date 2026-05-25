# CYB-1199 Tasks

- [x] Create Linear issue CYB-1199
- [x] Create branch `fix/CYB-1199-delivery-bugfixes` from origin/dev
- [x] Fix 1: Add `idempotency_key` to SDK delivery.commit()
- [x] Fix 2: Swap HandleDraft RequestedBy to prefer req.Owner
- [x] Fix 3: Remove non-TxRunner fallback in HandleAddItems, add 409 on optimistic lock
- [x] `go build ./...` passes
- [x] `go test ./internal/handlers/delivery/...` passes
- [x] `cd sdk && uv run ruff check src/` passes
- [x] Deploy backend dev + verify
- [ ] Commit + merge to dev
