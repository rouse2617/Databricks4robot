# Tasks — CYB-1135

## Context files
- `scripts/api-guide-smoke.sh` — smoke test script, add cancel/retry/ack section.
- `docs/review/api-guide.md` — already has cancel/retry/ack curl examples and error table (no changes needed).
- `openspec/changes/CYB-1135-delivery-ops-api-sync/` — this directory.

## Implementation
- [ ] [smoke] Add cancel happy-path test: commit a delivery, cancel it, assert status=cancelled.
- [ ] [smoke] Add retry happy-path test: retry the cancelled delivery, assert status=pending.
- [ ] [smoke] Add cancel error-path: cancel non-existent UUID → 404.
- [ ] [smoke] Add ack error-path: ack a pending delivery → 422.
- [ ] [openspec] Write proposal.md, tasks.md, specs/behavior/spec.md.

## API contract sync
- `docs/review/api-guide.md` already has cancel/retry/ack curl examples and error table (verified at lines 1409-1439). No additional changes needed.

## Local verification
- [ ] `bash scripts/api-guide-smoke.sh` against local or dev server (requires RUN_WRITES=1, valid CUST_ID, and an asset).

## Deploy verification
- [ ] Run smoke on dev after merge: `RUN_WRITES=1 BASE=<dev-url> TOKEN=<token> bash scripts/api-guide-smoke.sh`

## PR
- [ ] PR template filled; Linear `CYB-1135` linked.
