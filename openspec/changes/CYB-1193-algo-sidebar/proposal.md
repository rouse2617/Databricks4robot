# CYB-1193: fix(ui): AppLayout sidebar /algo route missing

## Summary

The sidebar "算法处理" menu item (key: `/algo`) is never highlighted because `resolveSelectedKey()` in `AppLayout.tsx` does not handle the `/algo` route. This is a regression from CYB-1167.

## Impact

Users navigating to the Algo Processing page see no sidebar highlight — it always falls back to `/assets`.

## Fix

Add `if (pathname.startsWith("/algo")) return "/algo";` to `resolveSelectedKey()` after the `/algo-runs` check.

## Verification

1. `npm run build` passes
2. Navigate to `/algo` → sidebar highlights "算法处理"
