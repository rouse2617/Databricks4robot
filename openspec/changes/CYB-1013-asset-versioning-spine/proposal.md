# Proposal — CYB-1013

## Why

Assets need a first-class multi-version model (`logical_assets` + per-revision rows) so algorithm outputs can promote new revisions without overwriting prior materialized versions.

## What Changes

### Modified Capabilities

- asset-management: schema for `logical_assets` and version columns on `assets`; write contract on create/promote; `version_promoted` events

## Impact

- **Affected code**: `backend/migrations/`, `backend/internal/models/`, `backend/internal/postgres/`, `backend/internal/repository/`, `backend/internal/usecase/asset/`, `backend/internal/handlers/asset/`
- **New APIs**: optional `logical_asset_id` on `POST /api/v1/assets` to create a promoted revision
- **Dependencies**: none

## Scope

- **In scope**: DDL, repos, create + promote write path, `version_promoted` event, unit/integration tests on new writes
- **Out of scope**: ES/search (CYB-1016), provenance API/UI (CYB-1017), backfill of existing `assets` rows, `POST /assets/{id}/revisions` dedicated route

## Success Criteria

- [ ] New asset without `logical_asset_id` gets `logical_asset_id=asset_id`, `revision=1`, `is_current=true` and a `logical_assets` row
- [ ] Create with existing `logical_asset_id` inserts next revision, flips prior `is_current`, updates `logical_assets` cache, emits `version_promoted`
- [ ] Partial unique index prevents two current revisions per logical asset

## Context files

- `docs/review/unified-asset-catalog/design/asset-versioning.md`
- `docs/review/unified-asset-catalog/schema.md` (§3 logical_assets, §3.1 B-route)
