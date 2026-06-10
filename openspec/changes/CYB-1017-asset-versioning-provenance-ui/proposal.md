# Proposal — CYB-1017

## Why

Asset versioning spine (CYB-1013/1016) is in PG/ES but invisible to users. Operators need version history and a way to switch revisions on the detail page.

## What Changes

- asset-management: `GET /api/v1/assets/{id}/provenance` with `version_history`, `revisions`, and structural `lineage`
- Frontend: asset detail version selector when multiple revisions exist
- UI design spec (wireframes / visual / Figma structure): [`design-ui.md`](design-ui.md) → [`docs/review/unified-asset-catalog/design/asset-versioning-provenance-ui.md`](../../../docs/review/unified-asset-catalog/design/asset-versioning-provenance-ui.md)

## Scope

- **In scope**: provenance read API, OpenAPI/api-guide/smoke, detail page version dropdown
- **Out of scope**: full PRD timeline/facets, `GET /logical-assets/*`, SDK (defer per CYB-1016 pattern)

## Success Criteria

- [ ] `GET /assets/{id}/provenance` includes `version_history[]` with `version`, `promoted_at`, `by_run_id`, `reason`
- [ ] Detail page lists revisions and navigates on version change
