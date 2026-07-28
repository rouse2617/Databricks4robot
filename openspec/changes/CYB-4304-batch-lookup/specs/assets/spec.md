# BatchAssetLookup spec — CYB-4304

## Component shape

`<BatchAssetLookup preset={preset} />` — a single component that renders the entire batch-lookup UX driven by `preset`. Consumers do not pass state; the component owns its own local state and calls `preset.fetch` when the user submits.

### Layout (top → bottom)

1. Paragraph secondary text describing the paste rules + max cap (from preset.maxIds)
2. Form:
   - Textarea: paste ids, split on whitespace/comma. Below it a small hint `N ids (K duplicates ignored)` in secondary color, or `— 超过上限 M` in danger color when over cap
   - id_type Radio: auto / asset_id / grace_video_id (fixed for the component; presets that don't want the choice can pass a preset-level `hideIdType: true` — durations shows it, lineage does too, costs adds a date-range row)
   - Preset filters slot: `preset.filters?.render(filters, setFilters)` renders in a horizontal Space or vertical Form.Item strip below the textarea
   - Submit button (disabled when parsed.ids.length === 0 || parsed.ids.length > preset.maxIds || filter.validate() returns error)

3. After submit → results area:
   - Row of statistic cards: matched / missing / filtered_out (always), then whatever `preset.extraStats(response)` returns (e.g. durations shows total/mean/p50/p90)
   - Histogram: `preset.histogram(items)` → flexbox bar chart (same style as CYB-4294). Preset may omit; component skips the row.
   - Results Table: `columns={preset.columns}`, default sort by first sortable column desc, page size 20, virtualization not needed at 5000 cap
   - Export CSV button (top-right of the table): triggers download via `preset.csv.filename(matched.length)` + header + rows
   - Missing ids Collapse: copyable textarea, count in the header
   - Filtered-out ids Collapse: same

### API contract

```ts
interface BatchLookupRequest<Filters> {
  ids: string[];
  id_type: "auto" | "asset_id" | "grace_video_id";
  filters: Filters;    // preset-specific extras — durations passes {min_duration_ms, max_duration_ms}
}

interface BatchLookupResponse<Item> {
  items: Item[];
  missing_ids: string[];
  filtered_out_ids?: string[];   // presets that don't range-filter server-side can omit
  stats: Record<string, number>; // arbitrary numeric bag — preset.extraStats interprets it
}
```

`preset.fetch` returns raw response; the component does not transform. Errors surface via antd `message.error` (message string comes from thrown Error).

## Durations migration

`pages/presets/durations.ts` — exports `durationsPreset: BatchLookupPreset<DurationLookupItem, {min_duration_ms?: number; max_duration_ms?: number}>`:
- `fetch = (req) => assetsApi.lookupDurations({ids, id_type, ...req.filters})`
- `columns` = current AssetDurationLookup columns (5 columns, input_id / asset_id / grace_video_id / duration_ms / formatted)
- `extraStats(res) = [{label:"总时长", value: formatDuration(res.stats.total_ms)}, {label:"均值", ...}, {label:"P50",...}, {label:"P90",...}]`
- `histogram(items)` = existing 5-bucket function
- `csv.filename(n) = 'asset-durations-' + n + '.csv'`, header/row unchanged

## Non-preset behavior

The framework does NOT decide bucket boundaries, stat aggregation, or CSV format — those live in the preset. This keeps the framework preset-agnostic; adding a fifth preset never requires editing the framework.
