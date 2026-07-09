# Tasks — CYB-3203: Asset metadata endpoint

## 0. Setup
- [ ] Confirm `CYB-3203` Linear issue exists; create if not.
- [ ] Branch `feat/CYB-3203-asset-metadata-endpoint` (already cut from latest dev).

## 1. Backend
- [ ] Add handler `GetAssetMetadata(c *gin.Context)` near existing asset
      handlers (`backend/internal/handlers/asset/handler.go` or new file).
- [ ] Read asset by id; 404 if missing.
- [ ] Read mcap_files by `mcap_file_id` (may be empty for grace_video without
      mcap — handle gracefully).
- [ ] Build response: `asset_id`, top-level `segment_locator` and
      `lifecycle_state` for convenience, `asset_metadata` (raw JSONB),
      `mcap_metadata` (raw JSONB or null), and convenience fields extracted
      from `mcap_metadata.storage_meta.*` / `process_info` / `video_info` /
      `collection_meta`, and `asset_metadata.grace_video_snapshot`.
- [ ] Wire route `GET /api/v1/assets/:id/metadata` in `backend/routes/routes.go`
      (no extra `RequireScope`).

## 2. API contract sync (mandatory)
- [ ] `api/openapi.yaml`: add `GET /api/v1/assets/{id}/metadata` path + 200/404
      response schemas.
- [ ] `docs/review/api-guide.md`: add curl example + auth + error paths.
- [ ] SDK: add typed method in `sdk/src/cyber_databrew_sdk/...` + export.
- [ ] SDK unit test: happy path + 404.
- [ ] `scripts/api-guide-smoke.sh` (or new `scripts/smoke-asset-metadata-dev.sh`):
      happy + error path against dev.

## 3. Frontend
- [ ] Add typed client (extend `Frontend/src/api/...` or new `assets.ts`).
- [ ] Add `react-json-view` to `Frontend/package.json` (or substitute a
      zero-dep recursive `<JsonNode>`; declare choice in PR description per
      AI-RULES dep rule).
- [ ] Add single `<Collapse>` panel "高级 / 元数据" to the asset detail page,
      wrapping `<JsonViewer collapsed={1}>`. **No tabs.**
- [ ] Read-only; no inline editor. Enable standard clipboard copy.
- [ ] Graceful handling of `mcap_metadata: null` (don't render empty subtrees
      that confuse the user).

## 4. Verification
- [ ] Local backend `make fmt && make vet` (Tier S) + `go test` packages
      touched (Tier M).
- [ ] Local frontend `npm run lint` + `npm run test -- --run` (Tier M).
- [ ] SDK: `cd sdk && uv run pytest tests/unit/` (Tier L for SDK touch).
- [ ] Curl smoke against deployed dev: confirm 200 with both metadata trees
      for a known asset id (e.g. 91022781) and 404 for a missing one.
- [ ] Chrome DevTools MCP: render the asset detail page in dev, click the
      "高级 / 元数据" section, confirm tree shows metadata.

## 5. Deploy + PR
- [ ] PR to dev; required reviewers ≥ 1 (off-limits: `backend/routes/` is
      reviewed per AI-RULES).
- [ ] Wait for `db-migrate-lint` + `commitlint` (or skip non-CYB head) + checks.
- [ ] Merge → deploy-dev green; no DB migration needed (no schema change).
- [ ] Update Linear status to Done with commit hash.

## Rollback
- Revert commit. No DB migration to reverse. Smoke endpoint can return 404
  (route gone) without affecting other endpoints.
