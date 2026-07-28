# Asset lineage batch spec — CYB-4305

## Endpoint

`POST /api/v1/assets/lineage-batch` under `assets:read`.

### Request validation

- `ids` non-empty, ≤5000 (400 `ID_LIST_REQUIRED` / `ID_LIST_TOO_LARGE`)
- `id_type ∈ {"", "auto", "asset_id", "grace_video_id"}` — server ignores it beyond echoing; matching is always both-column OR
- `depth ∈ {1, "1", "all"}` normalized to `1` or `"all"`; anything else → 400 `INVALID_LINEAGE_DEPTH`

### Server pipeline

1. **Dedup + resolve** — reuse the CYB-4306 `resolveAssetIDs(ctx, ids) ([]ResolvedAsset, missing []string, err)` helper. Any input that doesn't match an asset_id or grace_video_id row goes to `missing_ids`; the rest carry `{input_id, asset_id, grace_video_id}`.
2. **Depth=1 rows** — single SELECT on `assets` fetching parent/root/logical/is_current/revision for the resolved asset_id set.
3. **Depth=all overlay** — when `depth="all"`, also call `LineageDocsByAssetID(ctx, resolvedAssetIDs)` which runs one ES `_mget` (batched at ES's 10000-id limit) fetching only the three projected fields. Merge upstream/downstream/relation_types into the same item.
4. **Item order** — walk the request `ids` in insertion order (post-dedup). A resolved-but-not-in-DB case shouldn't happen (resolve step already dropped those) but if it does, treat as missing.
5. **Stats** — computed after items assembled:
   - `matched_count = len(items)`
   - `missing_count = len(missing_ids)`
   - `has_parent_count`: `parent_asset_id IS NOT NULL`
   - `is_root_count`: `root_asset_id IS NULL || root_asset_id == asset_id` (item IS a root — no ancestry above)
   - `orphan_count`: `parent_asset_id IS NULL AND root_asset_id IS NULL` (typically stale/broken asset)
   - `is_current_count`: `is_current == true`
   - `relation_type_counts`: `map[string]int` counting union of every item's `relation_types` array (depth=all only; absent from response body when depth=1)

### Response shape

Follows proposal.md — pointer fields on `LineageBatchItem` for parent/root/logical/upstream/downstream so Go serializes nulls not empty strings.

### Semantic notes

- `parent_asset_id` NULL — no known parent (could be a root, could be orphan)
- `root_asset_id` NULL — no known root (implies this IS the root, semantically same as `root_asset_id == asset_id`)
- **is_root_count** counts BOTH the "root_asset_id is null" case AND the "root_asset_id == asset_id" case: both mean "this asset is at the top of its lineage tree"
- **orphan_count** is the strict "neither known" case, usually a data-quality signal

### Depth=all fields

When `depth="all"`, each item additionally carries:
- `upstream_ids: []string` (may be empty for a root)
- `downstream_ids: []string` (may be empty for a leaf)
- `relation_types: []string` — union of Argo-style relation strings ("derive", "split", "merge", "clip", "promote") that led from any upstream to this asset OR from this asset to any downstream

For depth=1 responses these three fields are ABSENT (not null/empty) — keeps the payload smaller.

## Frontend preset

### Filter row

Single `Segmented` for `depth`:
```
[直接父/根 (1)] [全链 (all)]
```
Icons: `NodeIndexOutlined` for 1 (direct-hop), `PartitionOutlined` for all (subtree).

### Columns

Callback based on current filters.depth:

**depth=1** (7 cols):
| input_id | asset_id | grace_video_id | parent_asset_id | root_asset_id | is_current | revision |

**depth=all** (10 cols): above + 3 more:
| upstream_count | downstream_count | relation_types |

`relation_types` renders as a comma-joined string; `upstream_count`/`downstream_count` are `.length` of the arrays.

Default sort: `input_id` ascending (keeps output stable across queries so users can diff).

### extraStats cards

5 cards, always shown (values 0 for empty batches):

1. **已匹配** — matched_count
2. **有 parent 的** — has_parent_count (追溯上游可行)
3. **是 root 的** — is_root_count (顶层无上游)
4. **孤儿** — orphan_count (parent + root 都空,通常是异常)
5. **当前版本** — is_current_count

### Histogram

Depth=all only: categorical bar chart of `relation_type_counts`, one bar per relation type (dynamic, whatever's in the response). Skipped for depth=1.

### CSV

filename `asset-lineage-depth${d}-${n}.csv`, d ∈ {1, all}.

Header + row switch with the current depth. Multi-value fields (relation_types) joined with `|` inside a single CSV cell (matches CYB-4306 asset_algo convention).

## Non-goals surface

- No 数值 histogram (lineage has no numeric distribution — categorical only)
- No lineage tree graph render (single-asset lineage tab already does this)
- No date range (lineage is a static property of the asset, not time-windowed)
