# Proposal — CYB-1135

## Why

Delivery cancel/retry/ack endpoints (CYB-1104/1105/1106) were implemented in `cb205c0` and merged to dev, but the API contract sync was never completed. Per project policy, any new or changed HTTP API must have matching documentation, smoke tests, and an OpenSpec change record before it is considered fully delivered.

## What Changes

### Modified Capabilities
- **api-guide.md**: Add curl examples and error-path reference for cancel/retry/ack endpoints (already present from prior work).
- **api-guide-smoke.sh**: Add standalone smoke tests covering cancel (happy + 404), retry (happy), and ack error-path (pending → 422).
- **OpenSpec**: Record the change delta in `openspec/changes/CYB-1135-delivery-ops-api-sync/`.

## Impact
- **Affected code**: `scripts/api-guide-smoke.sh`, `openspec/changes/CYB-1135-delivery-ops-api-sync/`.
- **New APIs**: None (documenting existing endpoints).
- **Dependencies**: None — all endpoint code is already on dev.

## Scope
- **In scope**: smoke tests for cancel/retry happy paths, cancel 404 error path, ack 422 error path; OpenSpec proposal/tasks/spec.
- **Out of scope**: endpoint implementation changes, new API routes, UI changes.

## Success Criteria
- [ ] Smoke tests pass on dev for cancel (delivered → cancelled), retry (cancelled → pending), cancel 404, ack 422.
- [ ] OpenSpec change dir contains proposal.md, tasks.md, specs/behavior/spec.md.
- [ ] PR linked to CYB-1135.

## Goals (SLO)
- **Coverage**: Every endpoint permutation in api-guide.md has a corresponding smoke assertion.
- **Documentation**: OpenSpec delta is discoverable for future audit.
