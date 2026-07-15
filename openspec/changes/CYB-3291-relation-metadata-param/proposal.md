# CYB-3291 Fix InsertRelationWithMetadata missing metadata arg (P0 regression)

## Problem
Regression from CYB-3281 (#349). `InsertRelationWithMetadata` (repos.go) has a SQL
with 5 placeholders ($5::jsonb = metadata) but `db.Exec` passes only 4 args
(metadata omitted) → pgx "mismatched param and argument count" → 500. Since
CYB-3281 also made `InsertRelation` delegate to it, EVERY asset_relations insert
is broken: child-asset creation (clips/actions/frames/tasks) and InsertRevisionOf
(version promote). Verified on dev: `POST /assets/{seg}/actions` → 500.

## Scope
- `postgres/repos.go` `InsertRelationWithMetadata`: marshal metadata to jsonb and
  pass as the 5th arg (nil → `{}`). No SQL / arg-order change.
- Unit test asserting 5 args (incl metadata) are passed.

## Out of Scope
- parent/child arg-order convention (pre-existing, unchanged).
