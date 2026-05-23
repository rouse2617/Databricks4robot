# Proposal — CYB-1123

## Why

Recent dev merges introduced contract and state-flow regressions in the algo-runs and delivery APIs. The current behavior breaks frontend pagination, over-counts deliveries before commit, and leaves delivery acknowledgement semantics inconsistent with the model and docs.

## What Changes

### New Capabilities
- None.

### Modified Capabilities
- **asset-management**: `GET /api/v1/algo-runs` returns a pagination contract that is consistent between backend, frontend, OpenAPI, and api-guide.
- **asset-management**: duplicate `run_id` creation behavior for `POST /api/v1/algo-runs` becomes explicit and consistent across code and contract.
- **delivery**: C2 draft and retry flows stop mutating asset delivery indexes before final commit.
- **delivery**: delivery acknowledgement semantics align with the delivery state model and documented workflow.

## Impact
- **Affected code**: `backend/internal/handlers/algorun/`, `backend/internal/postgres/algo_runs.go`, `backend/internal/handlers/delivery/`, `backend/internal/postgres/repos.go`, `Frontend/src/api/algoRuns.ts`, `Frontend/src/pages/AlgoRunsPage.tsx`
- **New APIs**: None; existing HTTP behavior is corrected
- **Dependencies**: None

## Scope
- **In scope**: algo-runs list pagination fix, duplicate `run_id` contract alignment, delivery draft/retry indexing fix, delivery ack semantics alignment, API contract sync, targeted tests and smoke coverage
- **Out of scope**: new delivery features, new algo-runs search features, SDK expansion beyond any contract-following touch needed by the same API change

## Success Criteria
- [ ] Algo-runs list pagination works from the UI and API with aligned request params and response shape
- [ ] Algo-runs list `total` reflects the full result set semantics defined by the API contract
- [ ] Duplicate algo-run creation returns the documented and tested outcome
- [ ] Pending draft or retry deliveries do not increment `delivery_count` or `last_delivered_*` before commit
- [ ] Delivery acknowledgement behavior is consistent across handler, state model, OpenAPI, and api-guide
- [ ] OpenAPI, api-guide, and smoke/tests are updated in the same change
