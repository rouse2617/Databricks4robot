# Decisions — CYB-1015

## 2026-05-22 — PK shape

- **Context**: Linear loosely says `(asset_id, key, source)`. Design doc
  [`asset-tagging.md`](../../../docs/review/unified-asset-catalog/design/asset-tagging.md)
  §3.1 says **`(asset_id, tag_key, tag_value, source_type, source_version_norm)`**
  via generated column + UNIQUE (Postgres disallows COALESCE in PK).
- **Decision**: Follow design doc. `tag_value` is part of identity so the same
  source can legitimately record two competing assertions (`human:labeler_A says
  scenario=kitchen` vs `human:labeler_B says scenario=warehouse`) without
  overwrite. Idempotency of re-runs is preserved because `source_version` /
  `source_name` are part of the key.
- **Rationale**: Algo SDK re-running version 2.0 must be idempotent; humans
  disagreeing must be visible (UI shows both); compliance is immutable so
  identity must include source.
- **Trade-off**: A row count can grow faster than `(asset_id, tag_key)` would
  suggest; acceptable because growth is bounded by `(sources × versions)` per
  key per asset and indexes cover the lookup.

## 2026-05-22 — Surrogate PK

- **Context**: New unique key has 5 columns; FKs from future tables (e.g.
  `asset_tag_history` if ever added) and ES doc references benefit from a
  stable single-column ID.
- **Decision**: Add `id BIGSERIAL PRIMARY KEY`. Surrogate, not exposed in API.
  (Alternative: skip PK entirely and only rely on UNIQUE — rejected because
  some tooling assumes a PK exists.)

## 2026-05-22 — Touch `backend/migrations/` (off-limits exception)

- **Context**: Destructive DDL on `asset_tags`. Same exception pattern as
  CYB-1013 / 1014.
- **Decision**: Single migration `030_asset_tags_multisource.sql`. No reverse
  migration script (down path documented in `migration-plan.md` §5).

## 2026-05-22 — `GET /assets/{id}` response shape

- **Context**: Current `tags: map[string]string` cannot represent multi-source.
  Two options: (a) replace with array, breaking; (b) keep map + add
  `tags_detailed: []`.
- **Decision**: Option (b) — additive. Map is last-write-wins by
  `applied_at DESC`. Detailed list is the source of truth. Frontend `TagsTab`
  uses `tags_detailed`.
- **Rationale**: Avoid breaking existing SDK / UI callers that read the flat
  map. Cost is one extra field on the wire; size impact negligible.

## 2026-05-22 — `tag_registry.yaml` scope for this slice

- **Context**: Design §5 has 8 sources + `tag_keys` rules. Full schema is
  large.
- **Decision**: Land only the P0 sources — `human`, `algo_sdk`, `rule_engine`,
  `system`, `compliance` — plus existing `tag_keys` enum rules. `llm` /
  `vendor` / `crowdsource` left out (no writer yet). Propagation rules left
  for P1.5.

## 2026-05-22 — Default source on legacy `POST /tags`

- **Context**: Current handler body only has `{key, value}`; no source.
  Frontend `TagsTab` calls this path.
- **Decision**: Default `source_type=human`. If `human` requires
  `source_name` and the caller is authenticated, fill from auth subject; if
  unauthenticated dev path, fall back to `system`. Documented in api-guide.

## Open questions (to confirm before code)

1. Backfill of `source_name` / `source_version` for legacy rows — leave NULL,
   or set `source_name='legacy'` so future queries can filter them out?
2. Should `DELETE .../tags/{key}` without `?source_type` delete **all** sources
   (current behavior) or be rejected as ambiguous (`400`)?
3. ES `tags_flat` — keep last-write-wins, or drop it in favor of nested-only
   queries in `tags`? (Affects facet sidebar performance.)
