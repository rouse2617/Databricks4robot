# Tasks — CYB-3797

## Code

- [ ] `handlers/mcap/handler.go` — add `assetTagRepo repository.AssetTagRepository` field to `Handler` struct
- [ ] Add `SetAssetTagRepo(repo)` setter (mirrors `SetAssetRepo` pattern)
- [ ] In `createFileTx`, after successful `assetRepo.InsertNew`, call new `extractAndInsertMetadataTags(txCtx, placeholder.AssetID, f.Metadata, f.TenantID, f.ProjectID)` when `assetTagRepo != nil`
- [ ] New file `handlers/mcap/metadata_tags.go`:
    - `func extractMetadataTags(assetID, tenantID, projectID string, meta map[string]any) []repository.AssetTagUpsertInput`
    - `extractVibecapTasks(v any) []string` — nil / array / JSON string parse
    - `extractStringField(v any) string`
    - `extractChineseCity(addr string) string` — comma-split reverse, POI-suffix filter, min-length 3
    - Constants: `metadataTagSourceType = "system"`, `metadataTagSourceName = "mcap_ingest"`

## Wiring

- [ ] `cmd/server/core.go` — after `mcapHandler := mcap.New(mcapRepo)`, call `mcapHandler.SetAssetTagRepo(assetTagRepo)` (the repo is already constructed on line 48)

## Tests (Tier L required — new handler branch, shared types)

- [ ] `handlers/mcap/metadata_tags_test.go`:
    - `extractMetadataTags` — pangzi-shaped metadata → 3 known tags (task multi-value, source, city)
    - Unknown metadata keys (weather, collector_height) → no extra tags
    - `vibecap_tasks` shape variants: `nil` → 0, `[]string{}` → 0, `[]any{"a","b"}` → 2, `string("[\"a\",\"b\"]")` → 2, `map[string]any{}` (bad shape) → 0 (skip cleanly)
    - `location.address` variants: single segment `"惠乐超市"` → skip city (no city detected), `"惠乐超市, 合肥市, 安徽省"` → `合肥市`, missing key → skip
- [ ] `handlers/mcap/handler_test.go` — extend `TestCreateFile` (or `handler_metadata_tags_test.go`) to assert `AssetTagRepo.Upsert` is called with expected inputs when mcap POST has full pangzi-shaped metadata; use a stub `assetTagRepo` mirroring the existing `stubAssetRepo` pattern

## Verification

- [ ] `go build -ldflags "-w -s" -o /dev/null ./...`
- [ ] `go test ./...` — all packages green
- [ ] Off-limits self-audit: grep clean for middleware/auth, outbox, migrations, .env, schemas/pg-phase0.sql

## Post-deploy verify

- [ ] Create a new mcap on dev via `POST /api/v1/mcap-files` with pangzi-shape metadata (unique mcap_file_id, mocked vibecap_tasks + source_platform + location.address)
- [ ] `GET /api/v1/assets/{mcap_file_id}` → `tags` map contains `task`, `source`, `city`
- [ ] `POST /queries/run where: tags.source=vibecap` — count increased by 1
- [ ] Follow-up: rerun `scripts/backfill_pangzi_metadata_tags_dev.sh` one final time to catch the ~200 mcap uploaded between backfill and this PR — after that new uploads self-tag, no more script runs needed

## Contract sync

- [ ] `docs/review/api-guide.md` — add one line noting mcap-files POST auto-tags from metadata

## Deploy

- [ ] Merge to `dev`
- [ ] `deploy-dev.yml` completes
- [ ] Rerun post-deploy verify curls
- [ ] Update Linear CYB-3797 → Done with PR link
