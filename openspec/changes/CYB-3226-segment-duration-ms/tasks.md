# CYB-3226 Tasks

- [x] Create Linear issue CYB-3226
- [x] Create branch `fix/CYB-3226-segment-duration-ms` from origin/dev
- [x] **OpenSpec checkpoint** — user confirmed「开做吧」(2026-07-09)

## Fix
- [x] Canonicalize `duration_ms` before `SyncLegacyFields()` in `prepAssetForWrite`
- [x] Unit test: seconds-only → ms; span-only → ms; ms already set → unchanged

## Write gates
- [x] Shared range validator `validateAssetTimeRange`: reject `end <= start` and span `< 1ms`
- [x] Wire validator into all write entry points (Create, CreateChildAsset, CommitSegments)
- [x] Parent-range consistency: warn-only log in CreateChildAsset (child vs loaded parent); segment-vs-mcap deferred (decisions.md)
- [x] Unit tests for validator (end<=start → err; sub-1ms → err; valid → ok)
- [x] Map `ErrDurationTooSmall` → 422 INVALID_STATE in both handler error sites
- [x] Update existing test fixtures to valid (>=1ms) spans (ingest/projection/versioning/handler/server)

## API contract sync (validation tightening on existing endpoints)
- [x] `docs/review/api-guide.md` — note new range/duration validation + 422
- [x] `api/openapi.yaml` — document constraint on end_timestamp_ns (Create + Child requests)
- [x] smoke: `scripts/smoke-asset-duration-validation-dev.sh` — reject paths (end<=start, sub-1ms → 422)
- [x] `specs/asset-write/spec.md` delta — Given/When/Then for duration + validation
- [x] Frontend: N/A — no `assetsApi.create(` usage in the UI; segments are created by pipeline/SDK, not the assets UI

## Verify / ship
- [x] Tier L: gofmt + `go vet` + `go test ./...` — all pass except pre-existing CYB-3155 pipeline cost tests
- [x] Commit + PR #331 (CYB-3226 + OpenSpec change-id + API sync rows) + merged to dev (`a89944aa`)
- [x] Deploy-verify on dev (deployed revision): valid 60s create → `duration_ms=59999` (non-zero); end<=start → 422; sub-1ms → 422. (segment-create needs admin-token commit path scoped to an empty tenant, so the identical `Create`→`prepAssetForWrite` path was driven via `derived_asset` under the real user principal; verify asset soft-deleted)
- [x] Update Linear CYB-3226 → Done with commit hash `a89944aa`
