# Tasks — CYB-1013

- [x] OpenSpec proposal + tasks
- [x] Migration `028_asset_versioning.sql` (logical_assets + assets columns + indexes/trigger)
- [x] Model + `LogicalAssetRepository` + postgres wiring
- [x] Extend `AssetRepository` insert/select for version fields
- [x] Usecase: first-version create + promote via `logical_asset_id`
- [x] Handler: accept optional `logical_asset_id` on create
- [x] Tests: first version + promote + event type
- [x] `go test ./internal/usecase/asset/...` passed; decisions recorded
- [x] Deploy dev + API smoke (per deploy-before-commit.md)

## Deploy record — CYB-1013

| Item | Value |
|------|-------|
| Cloud SQL | `cyber-databrew-pg-dev` (`172.27.160.7`) / `cyber_databrew_dev` |
| Migration | `028_asset_versioning.sql` applied via GKE psql pod (2026-05-22) |
| Service | `cyber-databrew-backend-dev` |
| Image | `us-central1-docker.pkg.dev/green-valley-442103/rick-cyber-databrew-images/cyber-databrew-backend:81e0813-cyb1013-dev-fix3` |
| Revision | `cyber-databrew-backend-dev-00141-649` |
| URL | https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app |

### API verification matrix (fix3, 2026-05-22 — all PASS)

| Fix | What broke / changed | API to hit | Expected |
|-----|----------------------|------------|----------|
| **Core (CYB-1013)** | First version seed | `POST /api/v1/mcap-files` → `POST /api/v1/assets` (no `logical_asset_id`) | 201; `logical_asset_id=asset_id`, `revision=1`, `is_current=true` |
| **fix1** | Promote ignored when `asset_id` omitted | `POST /api/v1/assets` with `logical_asset_id` only (no `asset_id`) | 201; `revision=2`, new `asset_id` ≠ v1 |
| **fix2** | `revision_of` FK 500 (tx leak) | Same promote POST + `GET /api/v1/assets/{v2}/events` | 201; `event_type=version_promoted`, `prior_asset_id=v1` |
| **fix3** | `omitempty` hid `is_current:false` | `GET /api/v1/assets/{v1}` | 200; JSON has `"is_current": false` |
| **fix3** | current revision readable | `GET /api/v1/assets/{v2}` | 200; `"is_current": true`, `revision=2` |
| **read paths** | version fields on reads | `POST /api/v1/assets:batch_get` | both rows; v1 `is_current=false`, v2 `true` |
| **errors** | promote guards | `POST /api/v1/assets` unknown `logical_asset_id` | 422 |
| **errors** | type match | `POST /api/v1/assets` promote with wrong `asset_type` | 422 |
| **infra** | DB reachable from Cloud Run | `GET /readyz` | `pg.status=healthy` |

Prerequisite for asset creates: `POST /api/v1/mcap-files` (FK `fk_assets_mcap`).

**Out of scope for this change** (no dedicated route): `POST /assets/{id}/revisions`, list-by-logical, ES/search.

### Post-deploy code fixes (same branch, not yet committed)

- fix1: `allocateNewAssetID(..., promoteLogicalID)`
- fix2: `InsertRevisionOf` → `dbFromCtx`
- fix3: `Asset.is_current` without `omitempty`
