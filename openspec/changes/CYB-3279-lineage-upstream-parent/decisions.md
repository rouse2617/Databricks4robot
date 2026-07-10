# Decisions — CYB-3279

## 2026-07-10 — Gate on `parentAssetId !== mcap_file_id` (not pure presence); drop `assetType` prop

- **Context**: The proposal proposed gating the 父资产 node on presence of
  `parent_asset_id`. During implementation I checked a real segment
  (`XlPAwMz0`) and found **`segment.parent_asset_id == mcap_file_id`**
  (`6EDE33F6`) — a segment's parent is the raw_mcap asset, whose id equals the
  MCAP already rendered in the 上游 card.
- **Decision**: Gate on `parentAssetId && parentAssetId !== data.upstream.mcap_file_id`.
  - action / clip / frame / task: parent = segment/clip (≠ inherited mcap) → 父资产 node shown. ✅ "一起收益" (user-approved).
  - segment: parent == mcap_file_id → node hidden (no duplicate row) → segment 上游 unchanged. ✅
- **Consequence**: The `assetType` prop the proposal mentioned is **not needed**
  (the mcap-equality guard subsumes it), so `LineageTab` takes only the new
  `parentAssetId?` prop. Keeps the change to one meaningful prop.
- **Approved design (user, 2026-07-10)**: generic child-asset handling ("一起收益");
  parent rendered as a plain id `Link` to `/assets/<id>` with no extra fetch
  ("按照你的来").

## Deploy path

Following the CYB-3268 precedent (user: "提交 pr 到 dev 就走 cicd"): commit → PR →
CICD `deploy-dev` builds+deploys the frontend Worker on merge → Chrome DevTools
MCP verification on the deployed dev revision post-merge. Local gate: `biome
check` clean on the 2 touched files + `npm run build` clean (`tsc -b` is
pre-existing-red on dev in unrelated files — see CYB-3268 decisions).
