# Batch Lookup Hub spec — CYB-4333

## URL contract

| URL                                            | Behavior                                                    |
| ---------------------------------------------- | ----------------------------------------------------------- |
| `/assets`                                      | 资产列表 (existing AssetsPage, unchanged)                     |
| `/assets?view=lookup`                          | 批量查询 tab, preset = `DEFAULT_PRESET_KEY` (durations)       |
| `/assets?view=lookup&preset=durations`         | 批量查询 tab + durations preset                              |
| `/assets?view=lookup&preset=costs`             | 批量查询 tab + costs preset                                  |
| `/assets?view=lookup&preset=<unknown>`         | Fallback to `DEFAULT_PRESET_KEY`; URL corrected via replace  |
| `/assets?view=durations`                       | Redirect (replace) → `/assets?view=lookup&preset=durations`  |
| `/assets?view=costs`                           | Redirect (replace) → `/assets?view=lookup&preset=costs`      |

Redirects preserve any additional query params (unlikely, but safe).

## Registry API

```ts
// Frontend/src/pages/presets/index.ts

export type BatchLookupPresetKey = "durations" | "costs";

export const BATCH_LOOKUP_PRESETS: Record<BatchLookupPresetKey, BatchLookupPreset<any, any>> = {
  durations: durationsPreset,
  costs:     costsPreset,
};

export const DEFAULT_PRESET_KEY: BatchLookupPresetKey = "durations";

export const PRESET_OPTIONS: Array<{
  key: BatchLookupPresetKey;
  label: string;
  icon: ReactNode;
}> = [
  { key: "durations", label: "时长",  icon: <ClockCircleOutlined /> },
  { key: "costs",     label: "成本",  icon: <DollarOutlined /> },
];
```

Adding a preset later = import module + append one line each to the map and the options array; the type union grows too.

## Hub component

`BatchLookupPage` layout:

1. `<Segmented>` at top with `PRESET_OPTIONS` (icons + labels).
2. Below the Segmented: `<BatchAssetLookup preset={BATCH_LOOKUP_PRESETS[activeKey]} />`.
3. URL sync:
   - Read `?preset=` on mount + on searchParams change
   - Resolve to `activeKey` via a `resolvePresetKey(raw): BatchLookupPresetKey` helper that maps `null | unknown → DEFAULT_PRESET_KEY`
   - When Segmented changes, `setSearchParams(prev => { const next = new URLSearchParams(prev); next.set("preset", newKey); return next; }, { replace: true })`
   - `view` param is left untouched (workbench owns it)

## Workbench redirect

Inside `AssetsWorkbenchPage`, before the Tabs render:

```tsx
const view = params.get("view");
if (view === "durations" || view === "costs") {
  const target = new URLSearchParams(params);
  target.set("view", "lookup");
  target.set("preset", view);
  return <Navigate to={`/assets?${target.toString()}`} replace />;
}
```

Placed BEFORE the tabs so a stale bookmark doesn't briefly flash the wrong tab state.

## Non-URL surface

- No sidebar change.
- No backend change.
- `BatchAssetLookup` component itself unchanged.
- Preset configs (durations.tsx / costs.tsx) unchanged.

## Failure modes

- `?preset=whatever_garbage` → resolves to durations. Segmented UI shows the durations tab selected. Optionally show a small `message.warning` "未知 preset 已回退到时长" — spec does not require it; keep it silent to avoid noise on typos users will fix.
- Preset registry ever empty → this is a build-time invariant (TypeScript's `Record<Key, Preset>` enforces it). No runtime handling needed.
