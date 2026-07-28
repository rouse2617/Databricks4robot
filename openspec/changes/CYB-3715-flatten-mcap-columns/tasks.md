# Tasks — CYB-3715 flatten mcap columns to assets

## Migration (off-limits `backend/migrations/` — needs ≥2 reviewers)

- [ ] Create `backend/migrations/<YYYYMMDDHHMMSS>_flatten_mcap_columns_to_assets.sql` (timestamp = `date +%Y%m%d%H%M%S`)
- [ ] `ALTER TABLE assets ADD COLUMN` for the 7 new fields (all nullable)
- [ ] Partial indexes on `camera_model` / `device_id` / `scene_id` / `source_platform` (skip `data_source`, `collection_method`, `collector_id`)
- [ ] Backfill `UPDATE assets FROM mcap_files` in same migration
- [ ] `cd backend && make db-migrate-hash` — recompute `atlas.sum`
- [ ] `atlas migrate validate` — fresh-PG17 replay of every migration + checksum

## Backend handler

- [ ] `backend/internal/handlers/mcap/handler.go` `createFileTx`: extend the raw_mcap placeholder builder to copy the 7 fields from `f` onto the asset (`camera_model`, `device_id`, `collector_id`, `scene_id`, `data_source`, `collection_method`, `source_platform` — the last one from `metadata.source_platform`)
- [ ] Add matching fields to `models.Asset` if not already present
- [ ] `backend/internal/postgres/assets_repo.go` `InsertNew` / `Set` — include the 7 new columns in INSERT / UPDATE
- [ ] Ensure `PATCH /assets/:id` also accepts + persists these fields (if they should be updatable), or explicitly document that they're set-once at ingest

## Subscriber (ES sync)

- [ ] `backend/internal/searchindex/` — extend asset ES doc projection to include the 7 fields
- [ ] Trigger admin reindex on dev after migration lands (mostly for backfilled rows)

## Planner facet whitelist

- [ ] `backend/internal/queryplan/planner.go` — add to `PGSupportedFacetFields` (or equivalent) — `camera_model`, `data_source`, `collection_method`, `source_platform` (facet-able); `device_id`, `collector_id`, `scene_id` (filter-only)
- [ ] Verify structured filter also allows these fields (should be automatic once column exists + query plan uses columns list)

## Frontend

- [ ] `Frontend/src/components/assets/AssetsFacetSidebar.tsx` — add 4 chips (camera_model / data_source / collection_method / source_platform)
- [ ] Assets page filter drop-down — include the 7 new fields for `where` predicate use
- [ ] Update relevant `AssetSummary` / detail-view type + display

## Tests (Tier L required — migration + shared assets + planner)

- [ ] mcap handler test — extend `createFileTx` test to assert new fields propagate to asset
- [ ] postgres assets_repo test — INSERT with 7 new fields round-trips
- [ ] planner test — `{field: "camera_model"}` produces a valid PG plan, no 422
- [ ] `db-migrate-lint` CI — Atlas fresh-DB replay must pass
- [ ] Frontend unit test — chip renders + selecting it filters query
- [ ] `go build -ldflags "-w -s" ./...` + `go test ./...` all green

## Contract sync (per AI-RULES §API contract sync)

- [ ] `api/openapi.yaml` — Asset schema gets 7 new optional fields
- [ ] `docs/review/api-guide.md` — new section under `/queries/run` documenting facet-able mcap columns
- [ ] `scripts/api-guide-smoke.sh` — assertion that `field: "camera_model"` no longer 422s, actually returns hits

## Deploy

- [ ] `deploy-dev.yml deploy-migrate` step applies the migration
- [ ] `deploy-backend` picks up handler + subscriber changes
- [ ] Post-deploy: `curl -X POST /queries/run where camera_model=CyberCap2` returns hits (should be non-zero — pangzi mcap all have `CyberCap2`)
- [ ] Post-deploy: facet endpoint returns non-empty buckets for the 4 facetable fields
- [ ] Trigger ES reindex if the subscriber didn't already project historical rows
- [ ] Frontend deploy — chips visible + working

## Off-limits declaration

- [ ] `backend/migrations/` touched: 1 new file + `atlas.sum` update
- [ ] PR body flags off-limits + requests ≥2 reviewers
- [ ] Verify grep clean for `middleware/auth`, `outbox/`, `.env*`, `schemas/pg-phase0.sql`
