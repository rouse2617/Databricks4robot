# Tasks — CYB-3707

## Context files

- `backend/internal/usecase/pipeline/usecase.go` — `persistRunObservation` (2598), `reconcileMisclassifiedRunFromArgo` (2869), `applyWorkflowToRun` (2321), `reconcileTerminalRunFromLedger` (3013), `inferTerminalRunFromAssetNodes` (3072), `GetRun` (4774)
- `backend/internal/usecase/pipeline/runstate/` — run status normalization helpers

## Implementation

- [ ] [backend] Add `isTerminalFailureRunStatus` helper (Failed/Error, terminal) if not already present
- [ ] [backend] `persistRunObservation`: reject direct `Failed/Error → Succeeded` when `!existingWasActive` (keep existing failure); leave `Failed→Running→Succeeded` and `Succeeded→Failed` paths intact
- [ ] [backend] `applyWorkflowToRun` / `reconcileMisclassifiedRunFromArgo`: never map to `Succeeded` when the workflow is active (phase Running/empty) or has a Failed/Error leaf node → map to Running (re-observe)
- [ ] [backend] Recovery: in `GetRun` add bounded `terminalSucceededButLedgerHasFailure(run)` check → run `reconcileTerminalRunFromLedger` to re-infer Failed from the ledger
- [ ] [backend] Ensure `reconcileTerminalRunFromLedger` trigger/gate admits the new "Succeeded-with-failed-leaf" case

## API contract sync

No new/changed HTTP API — response shape unchanged. Skip OpenAPI/api-guide/SDK.

## Local verification (Tier M/L — touches status core)

- [ ] [backend] `make fmt && make vet`
- [ ] [backend] Unit: `persistRunObservation` guard (direct Failed→Succeeded blocked; Failed→Running→Succeeded allowed; genuine Succeeded unaffected)
- [ ] [backend] Unit: `applyWorkflowToRun` never derives Succeeded from active/failed-leaf workflow
- [ ] [backend] Unit: recovery re-infers Failed for Succeeded-with-failed-leaf run
- [ ] [backend] `go test ./internal/usecase/pipeline/...` (status core → run whole package)
- [ ] [backend] `go test ./...` before PR (Tier L: status derivation is cross-cutting)

## Deploy verification (post-merge via CI — user-directed)

Per user instruction ("直接 pr 到 dev,然后 merge 走 cicd 部署"), deploy verification runs after merge + CI auto-deploy (see decisions.md).

- [ ] After CI deploy: `a45cc195-…` open run detail → status corrects Succeeded→Failed, retry button appears
- [ ] Click retry → `smpl-body-fit` re-runs; run stays Failed after it re-fails (no self-lock); retry clickable again
- [ ] Regression: a genuinely Succeeded run still shows Succeeded (not flipped)

## PR

- [ ] PR template filled; Linear CYB-3707 linked; before/after status + retry evidence
