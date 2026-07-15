# CYB-3294 Asset detail: tab↔URL sync + clickable algo summary → algo tab

## Problem (UX walkthrough, group A items 1 & 4)
1. The asset detail `Tabs` uses `defaultActiveKey` (uncontrolled) and never writes
   the active tab to the URL, so refresh/share loses the current tab.
2. The header `Algo: N pending / M blocked` is plain text; to triage a blocked
   algo the user must manually switch to the 算法处理 tab and hunt for it.

## Scope
- **A4**: make `AssetDetailPage` Tabs controlled — `activeKey` from `?tab=`
  (validated against the known tab keys; invalid → `overview`); `onChange` writes
  `?tab=` back, preserving existing params (preview/time). Refresh/share keeps the tab.
- **A1**: make `AssetPreviewHero`'s algo summary (`AlgoSummaryInline`) clickable →
  calls a new `onJumpToAlgo` prop; `AssetDetailPage` passes
  `onJumpToAlgo = () => setTab("algo")`.

## Out of Scope
- Anchoring/scrolling to the specific blocked node inside AlgoTab (follow-up).
- No backend change.

## Verify (dev)
- switch tabs → URL shows `?tab=xxx`; refresh keeps the tab; clicking the header
  algo status jumps to the 算法处理 tab.
