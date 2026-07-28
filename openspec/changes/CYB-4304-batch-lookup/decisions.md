# CYB-4304 decisions

## Why preset-driven rather than inheritance

The 4-5 anticipated lookups all share ~80% of the shell (paste/parse/submit/stats/CSV/missing) and differ ~20% (columns, filter inputs, histogram buckets, response shape). Composition via a props-config beats subclassing because JSX doesn't reward inheritance and Preset lets the framework stay a pure function of its input.

## Why keep `AssetDurationLookup.tsx` as a shim

Two reasons:
1. `AssetsWorkbenchPage` currently `lazy(() => import('./AssetDurationLookup'))` — keeping the file preserves the chunk boundary and the URL contract (`?view=durations` still lands there)
2. Migration is safer when the old export stays default-exportable; a git blame-friendly one-line replacement keeps all upstream imports valid without touching them

## Why filters live in the preset (not the framework)

Durations has 2 optional number inputs; costs will have 2 required date pickers; lineage has a depth radio. Each preset knows its shape, its validation, and its default. If the framework tried to be generic-enough for all of them it'd become a mini form-builder — YAGNI for now.

## Why `stats` is a bag (Record<string, number>) not typed per preset

The framework only needs to render numbers. Each preset's `extraStats(response)` picks fields from that bag with its own copy — the framework doesn't need compile-time knowledge of what's inside. This lets the backend evolve (add new stat fields) without touching the framework.

## What we're not doing yet

- Server-side polymorphism (one `/assets/batch-lookup` endpoint with a `kind` field) — every preset stays on its own endpoint. Reason: makes each backend team-owned and independently versionable. Bundling later is easy; splitting later hurts.
- URL sync of the input state — no requirement to bookmark a lookup query; if it becomes one, add a `?state=...` URL contract later without breaking the current one.
