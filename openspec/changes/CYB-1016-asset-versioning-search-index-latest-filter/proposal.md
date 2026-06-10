# Proposal — CYB-1016

## Why

CYB-1013 added `logical_asset_id`, `revision`, and `is_current` on `assets`, but search still indexes and lists every revision. Users expect catalog/search to show one row per logical asset unless they explicitly ask for history.

## What Changes

### Modified Capabilities

- asset-search: ES documents carry version identity fields; `version_promoted` also re-indexes the demoted prior revision
- query-api: `POST /api/v1/queries/run` defaults to current revisions only; `?include_history=true` (or `scope.include_history`) returns all revisions

## Impact

- **Affected code**: `backend/internal/searchindex/`, `backend/internal/elasticsearch/`, `backend/internal/queryir/`, `backend/internal/queryexec/postgres/`, `backend/internal/handlers/query/`, `backend/internal/usecase/asset/versioning.go`, `backend/config/query_field_registry.yaml`, `deploy/local/elasticsearch/init-index.sh`
- **API**: optional query param `include_history` on `POST /queries/run` and `POST /queries/validate`
- **Dependencies**: CYB-1013 (done)

## Scope

- **In scope**: ES mapping + builder fields, default current-only filter (PG + ES), prior-revision re-index on promote, registry fields, contract docs + smoke
- **Out of scope**: provenance UI (CYB-1017), full dev ES reindex automation in CI, SDK

## Success Criteria

- [ ] ES `_source` includes `logical_asset_id`, `revision`, `is_current` for versioned assets; legacy rows index without `logical_asset_id` and remain searchable
- [ ] `queries/run` without `include_history` excludes `is_current=false` rows (PG: `COALESCE(is_current, TRUE)`; ES: current or non-versioned)
- [ ] `?include_history=true` returns historical revisions
- [ ] `version_promoted` emits a follow-up event so the demoted prior `asset_id` is re-indexed

## Context files

- `openspec/changes/CYB-1013-asset-versioning-spine/decisions.md`
- `backend/migrations/028_asset_versioning.sql`
