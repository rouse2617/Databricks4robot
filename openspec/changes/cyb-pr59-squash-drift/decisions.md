## 2026-06-03 — C1: row_version dropped from squashed baseline

- **Context**: PR #59 (`feat/may-integration`) review by ryanzhao9459 identified that `000_initial.sql` (squash of 36 archive migrations) lost `row_version` columns on 3 tables, and `event_retention_cleanup` function lost its `RETURN` statement. See `docs/review/pr-59-findings.md` (not yet committed to repo).

- **Decision**: Directly edit `backend/migrations/000_initial.sql` — an off-limits zone per AI-RULES §Off-limits — to backfill the lost columns and fix the function. This is not a hotfix against production; it's addressing review findings on an open, unmerged PR.

- **Changes applied**:
  1. `algo_runs`: added `row_version bigint NOT NULL DEFAULT 1`
  2. `customers`: added `row_version bigint NOT NULL DEFAULT 1`
  3. `logical_assets`: added `row_version bigint NOT NULL DEFAULT 1`
  4. `event_retention_cleanup`: added `deleted_count` variable, `GET DIAGNOSTICS`, and `RETURN deleted_count`

- **Verification**: On a fresh PG DB (no prior schema), the patched `000_initial.sql`:
  - Creates all 3 tables with `row_version` → `column_name` / `data_type` / `column_default` / `is_nullable` all match expected
  - Creates function with correct body → `GET DIAGNOSTICS deleted_count = ROW_COUNT; RETURN deleted_count;` present
  - Function executes → returns `0` (empty table, expected)
  - INSERTs get `row_version = 1` default → verified on all 3 tables

- **Alternatives considered**: Adding it as a new delta migration (`044_hotfix_row_version.sql`) instead of editing the baseline. Rejected because (a) fresh deploys must not rely on "safe" ordering of post-044 deltas, and (b) the archive approach already proved fragile — the squash was the source of the drift, so the fix belongs at the source.

- **Rationale**: Two DB build paths (fresh via `000_initial.sql` vs incremental via `archive/*.sql`) must produce identical schemas. The squash was the point of divergence; patching the baseline aligns both paths. Requires ≥2 PR reviewers per off-limits policy.

- **CI follow-up** (not implemented in this commit): A schema-drift CI guard that builds a DB from `000_initial.sql` and another from the full archive chain, then diffs `pg_dump --schema-only` — recommended by reviewer as permanent prevention.
