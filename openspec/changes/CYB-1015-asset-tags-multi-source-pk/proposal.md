# Proposal — CYB-1015

## Why

`asset_tags` PK is `(asset_id, tag_key)`. Same key from different sources
(human / algo_sdk / rule_engine / compliance / …) cannot coexist — every
new `Upsert` overwrites the previous source identity. This contradicts
[`docs/review/unified-asset-catalog/design/asset-tagging.md`](../../../docs/review/unified-asset-catalog/design/asset-tagging.md)
§3, blocks CYB-1020 delivery-rule evaluation (needs multi-source reads),
and silently drops `source_name` / `source_version` / `run_id` (already
columns on the table, but `Upsert` does not write them — design doc §3.3).

## What Changes

- **DDL** (`030_asset_tags_multisource.sql`):
  - Drop `PRIMARY KEY (asset_id, tag_key)`
  - Add generated column `source_version_norm TEXT GENERATED ALWAYS AS (COALESCE(source_version, '')) STORED`
  - Add `UNIQUE (asset_id, tag_key, tag_value, source_type, source_version_norm)` — `uq_asset_tags_identity`
  - New surrogate PK `id BIGSERIAL` (or `tag_row_id UUID`) so FK targets remain stable
  - Indexes per design §3.1 (`idx_atags_lookup`, `idx_atags_source`, `idx_atags_propagation`, `idx_atags_run`)
- **Repo**:
  - `AssetTagRepo.Upsert` writes `source_name`, `source_version`, `run_id`, `applied_at`; `ON CONFLICT` on new unique key
  - `AssetTagRepo.Delete(asset_id, tag_key, sourceType *string)` — optional source filter
- **Usecase / Handler**:
  - `UpsertTagInput` adds `SourceType`, `SourceName`, `SourceVersion`, `RunID` (`source_type` defaults to `human`)
  - `POST /api/v1/assets/{id}/tags` body accepts `source_type` / `source_name` / `source_version`
  - `DELETE /api/v1/assets/{id}/tags/{key}` accepts `?source_type=` (omit → delete all sources for key)
  - `GET /api/v1/assets/{id}` returns both `tags` (flat map, last-write-wins per key for backward compat) **and** `tags_detailed: [{key,value,source_type,source_name,source_version,run_id,applied_at}]`
- **Events**: `tag_upserted` / `tag_deleted` payloads include `source_name`, `source_version`, `run_id`
- **Search**: ES `tags` nested already carries `source_type` / `source_name`; flat `tags_flat[key]` keeps last-write-wins (documented)
- **Frontend `TagsTab`**: group rows by `tag_key`, show each source as a separate chip with source badge; delete acts on `(key, source)`
- **Registry**: extend `tag_registry.yaml` with `tag_sources[]` block per design §5; validator checks `writable_by` / `requires_source_name` / `requires_source_version` (P0 minimum — `human` / `algo_sdk` / `rule_engine` / `system` / `compliance`)

## Impact

- `backend/migrations/030_asset_tags_multisource.sql` (off-limits exception, same pattern as CYB-1013 / 1014)
- `backend/internal/postgres/repos.go` — `AssetTagRepo`
- `backend/internal/repository/asset_tag.go` — interface signatures
- `backend/internal/usecase/asset/usecase.go` — `UpsertTagInput`, `upsertTagProjection`, `DeleteTag`, `hydrateTags`, `findTag`
- `backend/internal/handlers/asset/handler.go` — request/response shapes, `UpsertTag` / `DeleteTag`
- `backend/internal/models/asset.go` — `Asset.Tags` shape (add `TagsDetailed []AssetTag` or rename)
- `backend/internal/searchindex/builder.go` — already iterates rows, but `tags_flat` semantics documented
- `backend/config/tag_registry.yaml`
- `api/openapi.yaml`, `docs/review/api-guide.md`, `scripts/api-guide-smoke.sh`
- `Frontend/src/components/asset-detail/TagsTab.tsx` + `api/assets.ts` types
- `backend/internal/postgres/repos_test.go`, usecase / handler tests

## Out of scope

- Tag propagation along lineage (P1.5 — design §8)
- LLM / vendor / crowdsource sources (registry left extensible; no writer yet)
- ES mapping change for `tags_flat` — single-value semantics retained
- Backfill of historical `source_name` / `source_version` for legacy rows (stays NULL)
- Action-label / metric / eval refactors (different tickets)

## Success Criteria

- Two rows with same `(asset_id, tag_key)` but different `source_type` coexist
- `Upsert` is idempotent on `(asset_id, tag_key, tag_value, source_type, source_version)`
- `GET /assets/{id}` returns all rows in `tags_detailed`
- `DELETE .../tags/{key}?source_type=human` removes only human row; other sources remain
- Delivery-rule prototype query `EXISTS (... tag_key=$1 AND source_type='compliance' ...)` works
- `tag_registry.yaml` rejects `human` write without `source_name` (422)
- Dev migration completes without data loss; pre/post row counts match plus expected splits

## Context files

- `docs/review/unified-asset-catalog/design/asset-tagging.md` §3 (PK design + bug-fix snippet)
- `docs/review/unified-asset-catalog/schema.md` (canonical DDL)
- `docs/review/sql.md` §4.3 (multi-source intent)
- Linear: [CYB-1015](https://linear.app/cyberorigin/issue/CYB-1015) — HITL (destructive migration)

## HITL gate

Destructive migration on `asset_tags`. Stop after this proposal + `tasks.md` +
`decisions.md` + `migration-plan.md` for user approval before touching
`backend/migrations/` or runtime code.
