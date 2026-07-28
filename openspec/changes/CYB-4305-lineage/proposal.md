# CYB-4305: batch asset lineage lookup — API + preset

## Context

Recurring lineage-reconciliation needs today require hand-run psql:
- "these 5000 segments came from which root mcaps?" — bulk root reverse lookup
- "which of these assets have no lineage at all?" — data-quality orphan check
- "are these all current-revision variants?" — logical-asset chain
- "everything derived from this mcap" — forward descendant sweep

All the underlying data is on-hand (assets table has parent/root columns; ES doc carries the projection from CYB-3268), the framework is on-hand (CYB-4304 BatchAssetLookup shell + CYB-4333 hub), so shipping this is a matter of one endpoint + one preset registration.

## Change

### Backend

New endpoint `POST /api/v1/assets/lineage-batch` (scope: `assets:read`).

Request:
```json
{
  "ids":     ["uYN6qys6", "019eda...uuid", ...],
  "id_type": "auto",                              // hint; server matches both cols
  "depth":   1 | "all"                            // default 1
}
```

Response:
```json
{
  "items": [
    {
      "input_id":         "uYN6qys6",
      "asset_id":         "uYN6qys6",
      "grace_video_id":   "019eda...",
      "parent_asset_id":  "aBcD1234",             // null if orphan / self-root
      "root_asset_id":    "root1234",             // null when self is root
      "logical_asset_id": "log_abc",              // versioned family, if any
      "is_current":       true,
      "revision":         3,
      "upstream_ids":     ["parent1","root1"],    // depth="all" only
      "downstream_ids":   ["child1","child2"],    // depth="all" only
      "relation_types":   ["derive","split"]      // depth="all" only
    }
  ],
  "missing_ids":      ["019eda...not-in-db"],
  "filtered_out_ids": [],                          // not used here, but keeps shape symmetric
  "stats": {
    "matched_count":      42,
    "missing_count":      3,
    "has_parent_count":   38,
    "is_root_count":      4,
    "orphan_count":       0,                       // parent AND root both null
    "is_current_count":   40,
    "relation_type_counts": {                      // depth="all" only, otherwise absent/empty
      "derive": 32, "split": 10, "merge": 5, "clip": 3
    }
  }
}
```

### SQL / ES paths

**depth=1** (default): one SELECT on `assets`, all columns are already direct:

```sql
WITH resolved AS (
  SELECT asset_id, grace_video_id, parent_asset_id, root_asset_id,
         logical_asset_id, is_current, COALESCE(revision, 0) AS revision
  FROM assets
  WHERE (asset_id = ANY($1) OR grace_video_id = ANY($1))
    AND is_deleted = FALSE
)
SELECT * FROM resolved;
```

No JOIN, no CTE beyond input-resolution. ~50ms for 5000 ids on dev.

**depth="all"**: after the same resolve, one ES `_mget` for the resolved asset_ids fetching `lineage_upstream_ids` / `lineage_downstream_ids` / `lineage_relation_types` (already projected onto every doc by `searchindex/builder.go` at CYB-3268 landing time). Merge with the SQL rows.

### Frontend

New preset `lineage` under the existing hub (`CYB-4333` — appears as a new option in the Segmented picker at `/assets?view=lookup&preset=lineage`).

- **Filter row**: single `Segmented` `depth: 直接父/根 (1) / 全链 (all)` — default `1`. That's it; no date range, no thresholds.
- **Columns (depth=1)**: `input_id / asset_id / grace_video_id / parent_asset_id / root_asset_id / is_current / revision`
- **Columns (depth=all)**: + `upstream_count / downstream_count / relation_types (joined string)`
- **extraStats** cards (5): `已匹配 / 有 parent 的 / 是 root 的 / 孤儿 / is_current 的`
- **Histogram**: **categorical bar chart of `relation_types`** (only when depth=all; otherwise skip). Buckets are dynamic (whatever types the batch actually contains).
- **CSV filename**: `asset-lineage-depth1-N.csv` / `asset-lineage-depthAll-N.csv`

## Non-goals

- Recursive multi-hop SQL — depth=all uses the ES projection to skip recursion entirely
- New backend indexes — CYB-3268 already indexed the projection fields
- Rendering a lineage tree diagram — that's the single-asset detail page's job; batch page is a table
- Cycles / cycle detection — projection is pre-computed acyclic; no runtime concern

## Success criteria

- `POST /api/v1/assets/lineage-batch` returns correct shape end-to-end
- 5000-id batch under 500ms depth=1, under 2s depth=all
- Hub Segmented picker shows 血缘 as a third preset alongside 时长 + 成本
- `?view=lookup&preset=lineage` loads with the new preset selected
