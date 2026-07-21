# Tasks — CYB-3714

## Backend / config

- [x] Add `source` (string) + `city` (string, max_length 32) to `backend/config/tag_registry.yaml` under `tags:`
- [x] No handler change; `POST /assets/:id/tags` already accepts arbitrary keys with source_type=system

## Backfill script

- [x] Write `scripts/backfill_pangzi_metadata_tags_dev.sh`:
  - Idempotent (POST /tags is Upsert)
  - Iterates all `owner=pangzi-consumer` mcap
  - Extracts `vibecap_tasks[]`, `source_platform`, `location.address` city
  - Posts one tag row per (mcap, key, value)
  - Summary line at end

## Contract sync

- [ ] `docs/review/api-guide.md` — one line noting `source` / `city` as baseline tags
- [ ] `api/openapi.yaml` — tag-registry schema doesn't enumerate keys; no change needed
- [ ] `scripts/api-guide-smoke.sh` — assertion that `GET /tag-registry` returns `source` + `city` entries (post-deploy)

## Verification (Tier M — YAML config + shell script, not runtime Go code paths)

- [ ] `cd backend && go test ./internal/handlers/asset/... ./internal/config/...` — YAML load parity, no regression
- [ ] Local dry-run: `bash scripts/backfill_pangzi_metadata_tags_dev.sh --dry-run` shows planned tags without POSTing
- [ ] Post-deploy on dev:
  - `curl GET /api/v1/tag-registry` → confirm `source` + `city` present
  - Run backfill script live: `bash scripts/backfill_pangzi_metadata_tags_dev.sh` (targets dev)
  - `POST /queries/run where: {pred:{field:"tags.task",op:"eq",value:"备餐操作"}}` → > 0 hits
  - `POST /queries/run where: {pred:{field:"tags.source",op:"eq",value:"vibecap"}}` → >= 140 hits
  - `POST /queries/run where: {pred:{field:"tags.city",op:"eq",value:"合肥市"}}` → some hits (address parsing may vary)

## Off-limits check

- [x] Confirm grep-clean of `middleware/auth`, `outbox/`, `migrations/`, `.env*`, `schemas/pg-phase0.sql`

## Deploy

- [ ] Merge PR to `dev`
- [ ] `deploy-dev.yml` reloads config + refreshes tag_registry
- [ ] Run backfill script targeting dev
- [ ] Update Linear CYB-3714 → Done with PR link + backfill summary
