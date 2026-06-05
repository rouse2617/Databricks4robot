# Tasks — CYB-1684

## Implementation

- [x] Audit current `pipeline_runs` persisted fields against the traceability requirements.
- [x] Decide whether existing run columns are sufficient or a migration is required for trigger/snapshot metadata.
- [x] Update backend run creation flow to persist stable execution-origin metadata.
- [x] Return the traceability fields from `/api/v1/pipeline-runs` and `/api/v1/pipeline-runs/:id`.
- [x] Sync API contract artifacts in the same change:
  - [x] `api/openapi.yaml`
  - [x] `docs/review/api-guide.md`
  - [x] frontend API types/hooks as needed
- [x] Update execution list row layout to show pipeline name/version clearly.
- [x] Add execution-list summary polish tied to the same information architecture:
  - [x] collapse empty label filter row
  - [x] add status/total summary
  - [x] improve disabled bulk-delete affordance
  - [x] surface failure summary or failure entry affordance
- [x] Update execution detail header summary to show pipeline/version/snapshot/trigger source.

## Verification

- [x] Add or update backend tests for run list/detail traceability fields.
- [x] Add or update frontend tests for execution list/detail traceability rendering.
- [x] Run targeted backend tests.
- [x] Run targeted frontend tests.
- [ ] Run `cd Frontend && npm run lint`. (fails on unrelated dev baseline issues; see `decisions.md`)
- [x] Run `cd Frontend && npm run build`.
- [x] Run `git diff --check`.
- [ ] Verify execution list and detail flows on dev with browser tooling after deploy.
