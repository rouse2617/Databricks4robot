# Tasks — CYB-3388

## 前端
- [ ] `LineageTab.tsx` — 接受 `assetType` prop
- [ ] 上游卡:raw_mcap 时改标题「存储」+ 隐藏 mcap_file_id/ingest_state 行 + 保留 URI
- [ ] 下游卡:仅保留 children,标题「子资产」,移除 algo/delivery/eval 三节
- [ ] 布局:删 `maxWidth: 720`,改两栏 `<Row><Col span=10>...</Col><Col span=14>...</Col></Row>`
- [ ] `AssetDetailPage.tsx` — 传 `assetType={asset.asset_type}` 到 `<LineageTab>`
- [ ] `AssetPreviewHero.tsx` — Badge text 「视频预览」→「已就绪」

## 测试
- [ ] `LineageTab.test.tsx`(如无则新建)— 场景:
  - raw_mcap:上游卡隐藏 mcap_file_id + ingest_state
  - segment:上游卡显示 mcap_file_id / URI / ingest_state
  - 无论何种,下游卡只显示 children
- [ ] Tier L:tsc + biome check

## PR
- [ ] commit + push branch `fix/CYB-3388-lineage-tab-ambiguity`
- [ ] PR → base dev

## dev 回归
- [ ] Chrome MCP 走查 raw_mcap 页面(qrZQGxpm)
- [ ] Chrome MCP 走查 segment 页面
- [ ] 视觉:右侧空白消除;绿点标签清晰

## 存量数据(可选)
- [ ] 附 SQL 脚本 `scripts/backfill_mcap_ingest_state.sql`(不合并到主分支,只在 PR 描述里贴给运维参考)
