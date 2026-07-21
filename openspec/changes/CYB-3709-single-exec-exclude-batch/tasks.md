# Tasks — CYB-3709

## Context files

- `Frontend/src/pages/WorkflowExecutionList.tsx` — `refresh()` non-batch branch (~line 891) `listRuns` call
- `Frontend/src/api/runApi.ts` — `excludeBatch` / `excludeBatchParents` params (unchanged)
- `backend/internal/postgres/pipeline_repo.go` — both filters already implemented (unchanged)

## Implementation

- [x] [frontend] Non-batch `listRuns`: `excludeBatchParents: true` → `excludeBatch: true`; replace the stale cyb-3392b comment with the CYB-3709 rationale
- [x] [frontend] Confirm the batch-scope branch (`isBatchScope && batchJobId`) is untouched

## API contract sync

No API change — both params already supported by `/runs`. No OpenAPI/SDK change.

## Local verification (Tier S/M)

- [ ] [frontend] `npm run lint` on the touched file
- [ ] [frontend] `npm run build`
- [ ] [frontend] related test run if any (`WorkflowExecutionList.test.tsx`)

## Deploy verification (post-merge via CI — user prefers CI deploy)

- [ ] Frontend dev auto-deploys on merge to dev
- [ ] Chrome DevTools MCP on dev: single-execution tab total ≈ 4.4k, no batch rows; batch tab still lists children
- [ ] Console: no new errors

## PR

- [ ] PR template filled; Linear CYB-3709 linked; before/after row counts
