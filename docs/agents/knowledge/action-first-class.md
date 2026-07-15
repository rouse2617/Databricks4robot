# Action as a first-class asset (`asset_type='action'`) — CYB-3268

> ⚠️ **v2 / BREAKING (2026-07-10).** `/assets/:id/actions` (POST / GET / PATCH /
> DELETE) is now served as a **first-class asset** on the `assets` table
> (`asset_type='action'`), replacing the standalone `actions` table for all new
> writes. Response shape changed: legacy action-table rows (`action_id` /
> `start_ns` / top-level `primary_label`) → asset rows (`asset_id` /
> `asset_type='action'` / `parent_asset_id` / `start_timestamp_ns` /
> `metadata.*`). The old `actions` table is **no longer exposed via the API**;
> its historical data stays in PG until a separate backfill issue.

## Why

The `actions` table was an annotation overlay: actions didn't appear in the
asset list API, ES facets, deliveries, or the lineage graph. After the Grace
migration, users began discovering actions (search by `primary_label=pickup`,
by description, lineage drill-down) — the overlay abstraction no longer fit.
Actions are now first-class assets: facetable, searchable, and visible in the
lineage graph (L2/L3 edges `segment → action` / `task → action`).

## Request / response shape (v2)

**Create** — `POST /assets/{seg}/actions`, body is `createChildAssetRequest`:

```json
{
  "start_timestamp_ns": 1640056114776298435,
  "end_timestamp_ns":   1640056115776298435,
  "split_method": "manual",
  "split_run_id": "",
  "metadata": {
    "primary_label": "pickup",
    "labels": ["pickup", "left_hand"],
    "description": "操作员从料盒中取出零件",
    "source_type": "human",
    "source_name": "annotator-001"
  }
}
```

Returns `201` + an `Asset` row (`asset_id`, `asset_type='action'`,
`parent_asset_id`, `start_timestamp_ns`, `metadata.*`, …).

- **List** `GET /assets/{seg}/actions?limit=&offset=` → `{items:[Asset], asset_id, total}`.
  Label filtering is client-side (server takes only `limit`/`offset`).
- **Update** `PATCH /assets/{seg}/actions/{aid}` `{metadata:{…}}` → merges metadata; 404 if `aid` is missing or not under `{seg}`.
- **Delete** `DELETE /assets/{seg}/actions/{aid}` → soft-delete; 404 on cross-parent / missing.

## Field mapping — legacy `actions` row → `asset_type='action'`

| Legacy `actions` column | First-class asset field |
| -- | -- |
| `action_id` | `asset_id` (8-char) |
| `asset_id` (seg ref) | `parent_asset_id` (+ inherited `root_asset_id`) |
| `start_ns` / `end_ns` | `start_timestamp_ns` / `end_timestamp_ns` |
| `primary_label` | `metadata.primary_label` (validated vs `config/action_label_registry.yaml`) |
| `labels[]` | `metadata.labels` |
| `description` | `metadata.description` |
| `source_type` / `source_name` / `source_version` / `run_id` | `metadata.source_*` / `metadata.run_id` |
| `confidence` | `metadata.confidence` |
| `external_id` | `metadata.external_id` |
| `action_index` | `metadata.action_index` |
| `attrs` (JSONB) | folded into `metadata` |
| — | `lifecycle_state` hard-coded `'ready'` (PG NOT NULL + CHECK; not written to ES) |

Validation lives in the `CreateChildAsset` usecase (label registry + the shared
`validateAssetTimeRange` ≥1ms floor). Hierarchy (`parent ∈ {segment, task}`) is
enforced by the existing `deliveryrules` validator (`checkAction`).

## ES / search

- `GET /search/assets?where=asset_type:eq:action` → all actions.
- `GET /search/assets?where=metadata.primary_label:eq:pickup` → by label (flattened metadata).
- `GET /search/assets?where=metadata.description:ilike:拿手机` → by description.
- Action docs **omit** `lifecycle_state` (no meaningless facet bucket) and no
  longer carry the seg's nested `actions[]` array (removed; wiped from old docs
  by a one-off `_update_by_query` at deploy).

## Querying the legacy `actions` table (ops, until backfill)

Historical actions written before CYB-3268 remain in the `actions` table and are
**not** returned by the API. To inspect them directly in PG (dev tool pod):

```sql
-- all legacy actions for a segment
SELECT action_id, start_ns, end_ns, primary_label, labels, source_type, run_id
FROM actions
WHERE asset_id = '<seg_id>' AND is_deleted = FALSE
ORDER BY start_ns;

-- count legacy actions not yet migrated to first-class
SELECT COUNT(*) FROM actions WHERE is_deleted = FALSE;
```

A separate backfill issue will migrate these into `assets`
(`asset_type='action'`) or drop the table per business need.

## Frontend

The action tab (`Frontend/src/components/asset-detail/ActionsTimelineTab.tsx`)
is unchanged; `Frontend/src/api/actions.ts` maps the first-class asset row ↔ the
`Action` view model (and the create body). Because there is no backfill, the tab
shows only actions created after CYB-3268 until the backfill issue lands.
