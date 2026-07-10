# Proposal — CYB-3279

## Why

CYB-3268 made `action` a first-class asset. In 资产详情 → 血缘 (LineageTab), the
「上游」card is built by `buildLineageResponse` from a single query
(`asset.mcap_file_id → mcap_files`), so for an action it shows the **inherited raw
MCAP** and **skips the immediate parent segment**. An action and its parent
segment therefore show the *same* upstream MCAP, and you can't see which segment
an action belongs to.

The data is correct — `action.parent_asset_id = <segment>` and the
`asset_relations` edge `action → segment` are written by `CreateChildAsset`. The
gap is purely presentational: the parent asset is never fed into `LineageTab`
nor rendered. The parent id is already loaded on the page (`AssetDetailPage` has
the full `asset`), so this is fixable **frontend-only**.

## What Changes

### Modified Capabilities
- **frontend — asset lineage** — `LineageTab`「上游」区对 child asset(有
  `parent_asset_id`,如 `action`/`clip`/`frame`/`task`)先渲染一个「父资产」节点
  (链到 `/assets/<parent_asset_id>`),现有 raw MCAP 卡片保留在其下作为更上一层
  来源。非 child asset(segment 等,无 `parent_asset_id`)行为不变。
- `AssetDetailPage` 把已加载的 `asset.parent_asset_id` + `asset.asset_type` 作为
  props 传给 `LineageTab`(当前只传 `asset.asset_id`)。

### Design decisions
- **Generic over child-asset types**, not action-only: gate on presence of
  `parent_asset_id`, so clips/frames/tasks benefit too.
- **No extra fetch**: render the parent as a clickable id link labeled 「父资产」.
  We do not fetch the parent to show its exact type — the link navigates to it.
  (Showing the resolved parent type/label is a possible later enhancement.)

## Impact
- **Affected code**:
  - `Frontend/src/pages/AssetDetailPage.tsx` — pass `parentAssetId` + `assetType` to `<LineageTab>`.
  - `Frontend/src/components/asset-detail/LineageTab.tsx` — add the two optional props; render a 父资产 node in the 上游 section when `parentAssetId` is set.
- **New APIs**: none. No backend change; uses `asset.parent_asset_id` already present client-side + the existing lineage response.
- **Dependencies**: none.

## Scope
- **In scope**: frontend display of the immediate parent asset in the lineage 上游 section for child assets.
- **Out of scope**:
  - Backend `buildLineageResponse` returning a server-resolved full chain (`action → segment → raw_mcap → grace_video`) — separate follow-up.
  - Any API / OpenAPI / SDK change (there is none).

## Success Criteria
- [ ] On an `action` asset detail → 血缘: 上游 shows a 「父资产」node linking to the parent segment, **above** the raw MCAP card.
- [ ] Clicking the 父资产 node navigates to `/assets/<parent_asset_id>`.
- [ ] A `segment` (no `parent_asset_id`) 上游 is unchanged (MCAP card only).
- [ ] No console errors; `npm run build` clean.
