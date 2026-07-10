# Tasks — CYB-3279

## Context files
- `Frontend/src/pages/AssetDetailPage.tsx:412` — `<LineageTab assetId={asset.asset_id} />`（当前只传 assetId）
- `Frontend/src/components/asset-detail/LineageTab.tsx:55-59` — `LineageTabProps { assetId }`；`205-230` 上游区渲染
- `backend/internal/handlers/asset/lineage_response.go:20` — 上游 = `asset.mcap_file_id → mcap_files`（本 issue 不改后端，仅参考）

## Implementation
- [ ] [frontend] `LineageTab.tsx` — `LineageTabProps` 加 `parentAssetId?: string` + `assetType?: string`
- [ ] [frontend] `LineageTab.tsx` — 上游区:当 `parentAssetId` 存在时,在 raw MCAP 卡片**之上**渲染「父资产」节点(`react-router` `Link` → `/assets/<parentAssetId>`,展示 id;label 「父资产」)。无 `parentAssetId` 时行为不变
- [ ] [frontend] `AssetDetailPage.tsx:412` — `<LineageTab assetId={asset.asset_id} parentAssetId={asset.parent_asset_id} assetType={asset.asset_type} />`

## Local verification (Tier M — 前端 2 文件,无共享组件/路由改动)
- [ ] `cd Frontend && npm run lint`（biome）
- [ ] `cd Frontend && npm run build`（vite;`tsc -b` 在 dev 上有历史无关红,以 build 为准 — 见 CYB-3268 decisions）

## Deploy verification (diff 含 `Frontend/` → 强制 Chrome DevTools MCP)
- [ ] merge 后 CICD `deploy-dev` 部署前端 Worker(或本地 `wrangler deploy --env dev`)
- [ ] Chrome DevTools MCP:打开一个 `action` 资产详情 → 血缘 → 上游出现「父资产」节点、点击跳父 segment;再开一个 `segment` 确认上游不变(仅 MCAP);Console 无 error;截图存 `openspec/changes/CYB-3279-*/deploy-verify-*.png`
- [ ] 回归:资产详情页其它 Tab(概览/算法处理/血缘 for segment)正常(§6.1 #3 + #2)

## PR
- [ ] PR 标题:`fix(frontend): show parent asset in lineage upstream for child assets (cyb-3279)`
- [ ] PR body:`## Linear` 链 CYB-3279;`## OpenSpec` 链本 change;`## 部署验证` 贴 MCP 截图 + revision;注明后端未改、无 API 契约变更
- [ ] 无 API contract sync(纯前端展示,无 HTTP 变更)
