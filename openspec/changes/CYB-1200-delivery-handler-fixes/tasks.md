# CYB-1200 Tasks

- [x] Create Linear issue CYB-1200
- [x] Create branch `fix/CYB-1200-delivery-handler-fixes` from origin/dev
- [x] Fix 1: Remove non-TxRunner fallback in HandleCancel
- [x] Fix 2: Add cancelled_by validation in HandleCancel
- [x] Fix 3: 409 on optimistic lock in HandleCancel
- [x] Fix 4: 409 on optimistic lock in HandleAck
- [x] Fix 5: Set CompletedAt in one-step commit
- [x] `go build ./...` passes
- [x] `go test ./internal/handlers/delivery/...` passes
- [ ] Deploy backend dev + verify
- [ ] Commit + merge to dev
