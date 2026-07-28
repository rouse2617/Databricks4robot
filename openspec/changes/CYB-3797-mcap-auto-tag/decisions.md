# Decisions — CYB-3797

## 2026-07-21 — Direct repo injection over usecase call

- **Context**: `POST /assets/:id/tags` handler calls `assetUC.UpsertTag` which does tag_registry validation + customer namespace lint. mcap `createFileTx` could do the same, but that introduces a cross-layer dependency (mcap handler → asset usecase).
- **Decision**: Inject `AssetTagRepository` directly into mcap Handler; call `Upsert` from `extractAndInsertMetadataTags`. Skip tag_registry validation for the 3 auto-extracted tags.
- **Alternatives**: Call `assetUC.UpsertTag` from mcap handler — cleaner semantic layering, but adds a hard dep on the asset usecase for a well-defined 3-tag surface.
- **Rationale**: The 3 keys (`task`, `source`, `city`) are already registered in `tag_registry.yaml` (CYB-3714). Validation would only re-check something the code path already guarantees. Direct repo mirrors the existing `assetRepo.InsertNew` pattern in the same function.

## 2026-07-21 — All 3 rules hardcoded, no config file

- **Context**: Rule set could live in Go code, in a YAML config, or in a DB table.
- **Decision**: Hardcode in Go (`extractMetadataTags` function).
- **Alternatives**: YAML config with parser registry — flexible for admin-added rules, but overkill for 3 rules.
- **Rationale**: 3 rules is well below the "worth designing a config DSL" threshold. Adding a new rule = one PR touching one file + one test. When the rule set grows past ~5-10 or when different producers need different mappings, revisit config-driven design.

## 2026-07-21 — Whitelist over dynamic mapping

- **Context**: Producer might send unknown metadata keys (`weather`, `collector_height`, `custom_x`). We could auto-tag them (dynamic) or ignore them (whitelist).
- **Decision**: Whitelist — only the 3 known keys are extracted. Everything else is ignored (but preserved in the JSONB blob on `mcap_files`).
- **Alternatives**: Auto-tag every top-level metadata key. Would let VibeCap add fields without databrew PR; also lets any producer pollute the tag namespace.
- **Rationale**: Tag namespace hygiene beats convenience. `tag_registry` is intentionally a curated catalog of business-meaningful labels. Adding a new producer or field = deliberate databrew PR to extend the rule set; the metadata JSONB always stays intact for future backfill or ad-hoc query.

## 2026-07-21 — Tag write failure rolls back mcap create

- **Context**: Should a tag insert failure inside `extractAndInsertMetadataTags` abort the mcap POST, or silently swallow?
- **Decision**: Roll back. The extraction runs inside the same tx as the mcap + asset + outbox writes; any error returned bubbles up and aborts everything.
- **Alternatives**: Log-and-continue — extract failures never fail the API call. Simpler for producers but leaves the mcap silently untagged, defeating the ticket's purpose.
- **Rationale**: If the tag repo is broken (DB unavailable, unique-constraint bug), we want to know via a 500 on the POST, not by discovering days later that new mcap have no tags. The 3-rule set is small enough that write errors are almost always infrastructure issues, not per-record.
- **Exception**: **Shape-mismatch skip** (e.g. `vibecap_tasks` is a number). That does NOT return an error from `extractMetadataTags` — the skip is silent, a WARN log line records it, and the mcap create proceeds. This is per-record data quality tolerance, distinct from tag repo failure.

## 2026-07-21 — Source name = `mcap_ingest`

- **Context**: `AssetTag` has a `source_name` field. Backfill script uses `backfill_cyb_3714`; POST /tags default = `human`. Auto-extract needs its own tag.
- **Decision**: `source_type="system"`, `source_name="mcap_ingest"`.
- **Rationale**: Distinguishes auto-tags from manual tags for audit; enables future selective cleanup (`DELETE ... WHERE source_name='mcap_ingest'`) if the extract rules change.
