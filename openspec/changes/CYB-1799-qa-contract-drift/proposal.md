# Proposal — CYB-1799

## Why
Deep QA found that several UI-visible API behaviors are missing from OpenAPI/SDK/docs, and one list page hides backend failures as an empty result set.

## What Changes

### New Capabilities
- None.

### Modified Capabilities
- API contract artifacts SHALL document the filters and operations already exposed by the backend and used by the frontend for asset events, MCAP files, algo runs, and pipeline operations.
- SDK managers SHALL expose the same filter/operation surface for the corrected MCAP, algo run, and pipeline routes.
- The algo runs list UI SHALL show a visible failure state when the list API fails instead of silently rendering an empty table.

## Impact
- **Affected code**: `api/openapi.yaml`, `docs/review/api-guide.md`, `sdk/src/cyber_databrew_sdk/`, `sdk/tests/unit/`, `Frontend/src/pages/AlgoRunsPage.tsx`
- **New APIs**: none; this syncs documented/SDK/frontend behavior to existing backend routes.
- **Dependencies**: none.

## Scope
- **In scope**: OpenAPI/doc/SDK sync for existing routes and frontend error-state fix for algo runs.
- **Out of scope**: data model changes, new backend routes, auth/session changes, Cloudflare/MCP profile handling, and broad UI redesign.

## Success Criteria
- [ ] OpenAPI includes the query params and response semantics used by asset event filters.
- [ ] OpenAPI includes existing pipeline active-version, promote, and batch-template-run endpoints.
- [ ] OpenAPI and SDK expose MCAP `owner` / `ingest_state` filters.
- [ ] OpenAPI, frontend, backend docs, and SDK align on algo run list filters.
- [ ] Algo runs list displays an actionable error when its list API fails.
