# CYB-3230 env structured search: enum → string

## Problem
Assets structured search rejects `env:家庭` with "字段 env 不支持值 家庭" and never
runs the query. `AssetsSearchBar.tsx` defines `env` as `type:"enum"` with a
hardcoded English value list `["kitchen","outdoor","warehouse","office","factory"]`,
but real env values are data-driven Chinese labels (家庭/工厂/学校/…). The value
validation blocks anything not in the stale list. Backend supports it (API:
`env:家庭` → 6 results).

## Scope
- `AssetsSearchBar.tsx`: change `env` from `type:"enum"` (hardcoded values) to
  `type:"string"` (like scene/task/batch). Any current/future env value passes to
  the backend filter — zero maintenance.

## Out of Scope
- Dynamic env dropdown (backend env terms aggregation + frontend suggestions) —
  recorded on the issue as a future enhancement; not this change.
- No backend / schema change.
