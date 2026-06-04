# Tasks — CYB-1631

## Implementation

- [x] Update execution-list mapping so `/api/v1/pipeline-runs` is the primary source.
- [x] Merge live `/api/v1/workflows` data into ledger rows when workflow names match.
- [x] Keep live-only Argo workflows visible for workflows without ledger records.
- [x] Display `totalEstimatedCost` from ledger rows as the list total cost.
- [x] Improve cost unavailable tooltips for historical ledger rows.
- [x] Extend generated Argo workflow TTL from 1 hour to 30 days for new runs.

## Verification

- [x] Add or update frontend tests for ledger-first cost display.
- [x] Update backend transpiler TTL test for 30-day manifests.
- [x] Run `cd Frontend && npm run lint`.
- [x] Run related frontend tests.
- [x] Run related backend tests.
- [x] Run `git diff --check`.
- [x] Verify dev/local UI with Chrome DevTools MCP after deployment.
