# CYB-3516 — Tasks

- [x] Trace self-perpetuating loop: `runObservedRecently` → `UpdatedAt` → `Save` → repeat
- [x] Remove `UpdatedAt` from `runObservedRecently` ref chain
- [x] Add unit test covering 4 scenarios (aged+recent-UpdatedAt, nil-FinishedAt, fresh run, no_ledger)
- [x] Full `go test ./internal/usecase/pipeline/...` passes
- [ ] Deploy to dev and observe log output ~20 min to confirm WARN noise drops
- [ ] Open PR
