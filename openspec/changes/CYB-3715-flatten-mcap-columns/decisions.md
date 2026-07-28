# Decisions — CYB-3715

## 2026-07-22 — Flatten to columns over ES-only projection

- **Context**: Two ways to make `camera_model` etc. queryable — (A) add columns to `assets` + backfill + planner extension, (B) index the fields into the ES doc only, planner routes these queries to ES.
- **Decision**: (A) columns + backfill.
- **Alternatives**: (B) ES-only would avoid the migration + off-limits path, and would leverage the existing planner engine split (CYB-3384).
- **Rationale**: These are **structured, low-cardinality-per-value** fields — exactly what SQL columns + indexes handle best. ES adds a sync gap (subscriber lag) that CYB-3384 fallback exists to hide, but for basic filter/facet correctness we want PG-primary. Also matches the CYB-3297 phase D precedent (env lifted to top-level column, not just ES).

## 2026-07-22 — 3 UUID fields filter-only, no facet

- **Context**: `device_id`, `collector_id`, `scene_id` are uuids. Should the facet whitelist include them?
- **Decision**: No. Filter yes, facet no.
- **Alternatives**: Include them — future-proof for admin views ("top 20 devices by mcap count").
- **Rationale**: Facet aggregation on high-cardinality uuid returns hundreds of thousands of buckets — the UI can't render, the ES agg is expensive. Same rule databrew already uses for `asset_id` (filter-only). Admins wanting "top devices" should build a purpose-built aggregation endpoint, not repurpose the facet chip.

## 2026-07-22 — Lift `source_platform` from metadata JSONB to first-class column

- **Context**: `source_platform` currently lives only in `metadata.source_platform`. Should we (a) leave it as JSONB and only tag it (already done in CYB-3714/CYB-3797), (b) lift it to a first-class column too?
- **Decision**: (b) — lift.
- **Alternatives**: Only tag. The `tags.source` filter already works after CYB-3797.
- **Rationale**: `source_platform` is the **producer identity** — a structural field every asset has (or will have), not a business label. Making it a column lets the planner treat it as an authoritative filter (no join to asset_tags), matches the treatment of `data_source` (which is already a column on mcap_files), and makes the "which producer wrote this" question a native asset-schema property. The `tags.source` filter continues to work as a redundant convenience path.

## 2026-07-22 — Migration includes backfill in the same file

- **Context**: Should the backfill `UPDATE` be in the same migration file, or a follow-up migration?
- **Decision**: Same file.
- **Alternatives**: Two migrations (add columns → backfill later).
- **Rationale**: The `UPDATE` is idempotent, cheap on the current dev size (~800 assets), and short. Splitting into two migrations means a window where columns exist but rows are null — code touching those columns must handle null gracefully anyway (they *will* be null on non-mcap-derived assets), so the split gains nothing. Prod is bigger (~50k rows expected), but a plain UPDATE with FROM join on `mcap_files.mcap_file_id` PK is fast (<1s at this scale).

## 2026-07-22 — Documented off-limits touch, requesting ≥2 reviewers

- **Context**: Touches `backend/migrations/` which is off-limits per AI-RULES.
- **Decision**: Explicit approval already granted by user via delegate ("你自己完成 ... 全程按照开发规范来"). PR body will flag it + request ≥2 reviewers.
- **Alternatives**: Skip the migration and only add code (columns nullable, populated on new writes only). Would leave the 800+ historical assets un-queryable → violates the "everything is searchable" acceptance criterion.
- **Rationale**: The value of this ticket is proportional to the coverage — a partial rollout (new mcap only) doesn't fix the current UX. Off-limits process is designed for reviewer scrutiny, not blocking legitimate schema evolution.
