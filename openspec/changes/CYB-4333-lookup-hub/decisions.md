# CYB-4333 decisions

## Why a hub tab, not a hub page

Two hop-levels (workbench tabs → preset picker) beats one very wide tab strip (`资产列表 | 时长 | 成本 | 血缘 | 采集 | 交付 | ...`) at 4+ presets. Two levels also fits users' mental model — top level is "what mode of the assets page am I in?", inner is "which lens?".

## Why Segmented, not Dropdown or Tabs inside

- Tabs-inside-Tabs is confusing (two rows of horizontal indicators competing).
- Dropdown hides the options — a new user can't discover what the workbench offers.
- Segmented is visible, keyboard-navigable, and scales to ~6 items cleanly. At >6 we can revisit.

## Why `?view=lookup&preset=<key>` instead of `?preset=<key>` alone

Keep `view` as the workbench's single source of truth for "which tab is active." Preset is a sub-state of that tab; nesting it under `view=lookup` makes both the redirect (`view=durations → view=lookup+preset=durations`) and the workbench's tab detection obvious.

## Why redirect old URLs instead of just removing them

`?view=durations` shipped on 2026-07-27 (CYB-4304 merge) and `?view=costs` shipped 2026-07-28 (CYB-4306). Any Slack/doc/browser bookmark from that window would 404-tab silently otherwise. Redirects are one-line with `<Navigate replace />`, cheap insurance.

## Why delete the shim pages instead of keeping them

`AssetDurationLookup.tsx` / `AssetCostsLookup.tsx` were 5-line shims meant to serve as URL landing points. The hub owns the URL now. Keeping the shims would duplicate the render tree and confuse future readers (which file is "the durations page"?).

## Why an `unknown preset` falls back silently

Users don't typo preset keys in the URL directly — the Segmented sets them. The only way to hit `?preset=bogus` is a hand-edited URL, a stale link from a rename, or a Slack copy-paste with a truncated string. Landing them on the default view (with the URL replaced so refresh gives a clean state) is the least surprising behavior.

## Why DEFAULT_PRESET_KEY = durations, not first-of-map

`Object.keys` order isn't specified across JS engines; explicitly naming durations keeps the default deterministic. Durations is also the first preset users hit (it's what the whole framework was extracted from), so it's the least-surprise landing tile.

## What we're not doing

- Preset icons in the response payload — they live in the frontend config, not the backend.
- URL sync of preset-specific filters (e.g. date range for costs) — those stay local component state. Bookmarking a filled-out cost query is out of scope; if a user needs it later we add `?filters=<encoded>` on top of the existing shape.
- Recent-presets memory — no user has asked for it; YAGNI.
