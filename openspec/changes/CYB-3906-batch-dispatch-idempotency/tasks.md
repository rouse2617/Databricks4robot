# Tasks — CYB-3906

## Context files

- `backend/internal/usecase/pipeline/batch_subtask.go` — placeholder/run identity creation and legacy name handling.
- `backend/internal/usecase/pipeline/usecase.go` — actual Argo workflow naming and AlreadyExists refresh.
- `backend/internal/usecase/backfill/submitter.go` — automatic recovery and stale-attempt decisions.
- `backend/internal/usecase/backfill/usecase.go` — materialization, reconciliation, and manual rerun paths.
- `backend/internal/usecase/backfill/submitter_test.go` — submitter fake currently masks production identity mismatch.
- `backend/internal/usecase/pipeline/batch_subtask_test.go` — run identity and explicit rerun coverage.
- `openspec/changes/CYB-3677-batch-dispatch-submitter/decisions.md` — original deterministic-name invariant.
- `openspec/changes/CYB-3677-batch-dispatch-submitter/specs/pipeline/spec.md` — existing restart/no-duplicate behavior.
- `docs/agents/deploy-verification.md` — backend deploy and regression verification.

## Implementation

- [x] [backend] Add failing tests proving a materialized UID-less placeholder is reused rather than replaced on first submit.
- [x] [backend] Add failing tests for create-success/UID-persistence-failure followed by same-name AlreadyExists convergence.
- [x] [backend] Add a concurrent-submitter regression test proving both callers resolve one initial run UUID and one Argo workflow identity.
- [x] [backend] Make the initial non-rerun batch attempt ID deterministic from the full `(batchJobID, assetID)` key.
- [x] [backend] Persist the run UUID as the placeholder `workflow_name`, matching Deploy and watcher lookup behavior.
- [x] [backend] Remove automatic `ForceNewAttempt` for UID-less placeholders and terminal linked attempts; retain it only for explicit rerun paths.
- [x] [backend] Replace `-batch-` placeholder-name heuristics with structural pending-batch predicates where lifecycle correctness depends on them.
- [x] [backend] Normalize legacy UID-less business-name placeholders to their existing run UUID before submission.
- [x] [backend] Update submitter fakes so successful Deploy returns the production UUID workflow name instead of the placeholder business name.

## API contract sync

- [x] N/A — no HTTP route, request/response, status code, or public SDK behavior changes.

## Local verification

- [x] Run `gofmt` on changed Go files.
- [x] Run `cd backend && make vet`.
- [x] Run `cd backend && go test ./internal/usecase/backfill ./internal/usecase/pipeline ./internal/postgres ./internal/argo`.
- [x] Run `cd backend && go test ./...` before PR/deploy.

## Deploy verification

- [x] Build the backend dev image; deployment was explicitly waived by the user after the shared dev revision failed pre-existing token/JWT validation.
- [x] N/A by explicit user waiver — dev batch identity smoke deferred.
- [x] Covered locally by the UID-less placeholder + AlreadyExists regression test.
- [x] N/A by explicit user waiver — dev L1 smoke deferred.

## PR

- [x] Replace the temporary Linear placeholder in the branch and OpenSpec directory with `CYB-3906` before commit/PR.
- [ ] Fill the PR template with Linear ID, OpenSpec change ID, test evidence, and deploy evidence.
