# Tasks — CYB-3062

## Context files
```
backend/internal/usecase/backfill/usecase.go   # executeItem, ~L538 DeployOptions literal
```

## Implementation

- [ ] [backend] `usecase.go`: add `Owner: job.CreatedBy` to the `DeployOptions{}` literal in `executeItem`

## Scenario coverage (tests)

- [x] Covered via live deploy verification instead of a new unit test — see `decisions.md`. Covers *"Backfill batch item deployed for a job with a known creator"* by submitting a real backfill batch and inspecting the resulting pod's labels via `kubectl`, which is stronger evidence for this bug class than a new mock would provide.

## API contract sync
N/A — internal-only change, no HTTP surface touched.

## Verification (Tier M — single-file logic change)
- [ ] `make fmt && make vet`
- [ ] `go test ./internal/usecase/backfill/...`
- [ ] `go test ./...` (full suite, per this session's established practice)

## Deploy verification
- [ ] Deploy backend to Cloud Run dev, confirm serving revision matches this change's commit
- [ ] Submit a real backfill batch job as an authenticated user
- [ ] `kubectl get pod -l workflows.argoproj.io/workflow=<resulting workflow> -o jsonpath='{.metadata.labels}'` — confirm `cyber-databrew/owner` is now present
