# Decisions — CYB-1013

## 2026-05-22 — Column naming vs Linear issue text

- **Context**: CYB-1013 acceptance text mentions `version` / `is_latest`; PRD design uses `revision` / `is_current`; `assets.version` already stores optimistic-lock row version.
- **Decision**: Implement `logical_asset_id`, `revision`, `is_current` per `asset-versioning.md` / `schema.md`.
- **Alternatives**: Rename OCC column; alias `is_latest` in API only.
- **Rationale**: Avoid breaking existing Set() OCC semantics and match PRD DDL.

## 2026-05-22 — No backfill of existing assets

- **Context**: User explicitly excluded存量回填 migration from CYB-1013 scope.
- **Decision**: New columns nullable; only new inserts/promotes populate version fields. Legacy rows remain NULL.
- **Alternatives**: One-shot backfill `logical_asset_id = asset_id`.
- **Rationale**: Smaller blast radius; backfill can be a dedicated migration issue later.

## 2026-05-22 — Touch `backend/migrations/` (off-limits exception)

- **Context**: `AI-RULES.md` lists `backend/migrations/` as off-limits; CYB-1013 cannot ship without DDL.
- **Decision**: Add `028_asset_versioning.sql` only; no edits to `001_init.sql` or `schemas/pg-phase0.sql`.
- **Rationale**: User asked to start CYB-1013 implementation; schema slice requires one forward migration.

## 2026-05-22 — Deploy-time fixes (dev smoke)

- **Context**: First Cloud Run deploy passed `/readyz` but promote smoke failed.
- **Decision**: (1) When `asset_id` is omitted, still honor `logical_asset_id` in `allocateNewAssetID` → `persistNewAsset(..., promoteLogicalID)`. (2) `InsertRevisionOf` must use `dbFromCtx(ctx, r.c.db)` so `revision_of` edge inserts see the new asset row in the same transaction. (3) Drop `omitempty` on `Asset.is_current` so `false` appears in API JSON.
- **Alternatives**: Defer relation insert to post-commit hook; document “omit field means false”.
- **Rationale**: Promote is the CYB-1013 spine contract; silent `omitempty` broke verification and client semantics.
