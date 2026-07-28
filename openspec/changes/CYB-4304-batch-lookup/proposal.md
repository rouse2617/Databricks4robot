# CYB-4304: BatchAssetLookup shared component; migrate durations onto it

## Context

CYB-4294 shipped 时长批量查询 as a monolithic ~450-line `AssetDurationLookup.tsx`. Follow-ups (CYB-4305 血缘, CYB-4306 成本) will need the same shell: paste ids → parse → submit → stats + histogram + table + missing/filtered-out + CSV. Copying that shell 3-5 times duplicates parse/CSV/error-handling bugs.

## Change

Introduce `Frontend/src/components/batch-lookup/BatchAssetLookup.tsx` as a preset-driven generic component. `AssetDurationLookup.tsx` becomes a thin re-export that plugs a `durationsPreset` into `BatchAssetLookup`.

### Preset shape

```ts
interface BatchLookupPreset<Item, Filters = {}> {
  key: string;                            // "durations" | "lineage" | "costs"
  placeholder: string;                    // textarea example ids
  maxIds: number;                         // hard cap
  fetch: (req: BatchLookupRequest<Filters>) => Promise<BatchLookupResponse<Item>>;
  filters?: {                             // extra inputs beside id_type
    initial: Filters;
    render: (f: Filters, set: (f: Filters) => void) => ReactNode;
    validate?: (f: Filters) => string | null;
  };
  columns: ColumnsType<Item>;             // antd table columns for results
  extraStats?: (res: BatchLookupResponse<Item>) => StatEntry[];
  histogram?: (items: Item[]) => BucketRow[]; // preset-specific bucket boundaries
  csv: {
    filename: (matchedCount: number) => string;
    header: string[];
    row: (item: Item) => (string | number)[];
  };
}
```

### Migration

- `AssetDurationLookup.tsx` — reduced to preset config + `<BatchAssetLookup preset={...} />`; histogram bucket constants + `formatDuration` helpers moved to `presets/durations.ts`
- `AssetsWorkbenchPage.tsx` — untouched (still lazy-loads `AssetDurationLookup` which now dispatches to the shared component)

### Non-goals

- Configurable bucket boundaries — presets still hard-code their own histograms
- Backend changes — the shape stays exactly what CYB-4294 shipped (`AssetDurationsResponse` shape)
- Adding lineage/cost presets — they land in CYB-4305 / CYB-4306

## Testability

Existing `AssetDurationLookup.test.tsx` continues to pass (integration): render the durations preset, submit a mocked fetch, assert stats + histogram + CSV filename. New: `BatchAssetLookup.test.tsx` covers the preset-agnostic paths (parseIdBlob edge cases, disabled Submit at cap, missing/filtered-out render even when items empty).
