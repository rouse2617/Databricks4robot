# Tasks — CYB-1015

- [x] OpenSpec proposal + tasks + decisions + migration-plan
- [x] **HITL checkpoint** — user approved (2026-05-22 "可以，开整")
- [x] Migration `030_asset_tags_multisource.sql`
- [x] `backend/internal/repository/common.go` — `AssetTagUpsertInput` + new interface signatures
- [x] `backend/internal/postgres/repos.go` — `AssetTagRepo.Upsert` writes all source columns; `Delete` optional `sourceType`; new conflict target
- [x] `backend/internal/models/asset.go` — `Asset.TagsDetailed` (keep `Tags` map for back-compat); `AssetTag.AppliedAt`
- [x] `backend/internal/usecase/asset/usecase.go` — `UpsertTagInput` fields; `hydrateTags` (last-applied-wins + detailed); `DeleteTag(assetID, key, sourceType)`; events carry source identity; `tagSource` + `assertNotImmutable`
- [x] `backend/internal/handlers/asset/handler.go` — request body source fields; delete `?source_type=`; 409 `TAG_IMMUTABLE` / 422 `TAG_SOURCE_INVALID`
- [x] `backend/config/tag_registry.yaml` — `tag_sources[]` (human/algo_sdk/rule_engine/system/compliance)
- [x] `backend/internal/config/tag_registry.go` — `TagSourceDef`, `ValidateSource`, `SourceDef`
- [x] `backend/internal/httpresp/codes.go` — new codes
- [x] Unit tests — `TestUpsertTag_MultiSourceCoexists`, `TestUpsertTag_RejectsMissingSourceName`, `TestUpsertTag_ImmutableSourceCannotBeRewritten`, `TestDeleteTag_SourceScopedRemovesOnlyMatching`; fix all existing test mocks
- [x] API contract sync — `api/openapi.yaml`, `docs/review/api-guide.md` §2.4, `scripts/api-guide-smoke.sh`
- [x] `Frontend/src/components/asset-detail/TagsTab.tsx` — source chips + per-source delete + add panel with source identity
- [x] `Frontend/src/api/assets.ts`, `Frontend/src/api/types.ts` — `AssetTagDetail` + new request/query params
- [ ] Dev pre-flight (§3.5 zero-dup check) on `cyber-databrew-pg-dev`
- [ ] `scripts/apply-migration-dev.sh 030`
- [ ] Dev deploy backend + smoke (multi-source upsert, source-scoped delete, 422 on missing `source_name` for human)
- [ ] Linear update + branch merge to `dev`

## Dev verification

| Check | Result |
|-------|--------|
| Migration applied on `cyber-databrew-pg-dev` | _pending_ |
| Pre/post row counts (`asset_tags`) | _pending_ |
| Two-source coexistence test | _pending_ |
| `DELETE ?source_type=human` removes only one row | _pending_ |
| Registry rejects `human` w/o `source_name` (422) | _pending_ |
| Smoke `scripts/api-guide-smoke.sh` (tags section) | _pending_ |
