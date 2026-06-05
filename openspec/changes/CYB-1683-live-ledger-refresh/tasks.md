# Tasks — CYB-1683

## Implementation

- [x] Refresh run events, asset-node rows, and cost summary while active workflows poll.
- [x] Add terminal catch-up refreshes for ledger data after workflow completion.
- [x] Merge live Argo node status into asset-node table display when ledger rows are stale.
- [x] Preserve durable run IDs when opening details from the execution list.

## Verification

- [x] Add focused frontend tests for live ledger polling and stale asset-node fallback.
- [x] Run targeted frontend tests.
- [x] Run targeted Biome check.
- [x] Run frontend build.
- [x] Run authenticated local browser smoke against Cloud Run dev backend.
