# Tasks — CYB-3382

## #1 搜索 DSL 加 asset_type

- [ ] `AssetsSearchBar.tsx` `keys` 数组(约 :33-70)加 `{ key: "asset_type" }`(参考现有 lifecycle_state / algo_status 项的结构;若各项还有 label/values 需同步)
- [ ] :520 附近「可用字段」帮助浮层文案加 `asset_type`
- [ ] Vitest:`splitRespectingQuotes("asset_type:segment algo_status:ok")` → 两个 structured token(而非 fulltext)

## #2 顶部 facet 折叠

- [ ] 找到 facet 面板容器组件(可能是 `AssetsFacetSidebar` 顶层 + Assets 页 layout wrapper),加 `collapsed` state
- [ ] 折叠 UI:显示当前 active filter 摘要行(如 `env=warehouse · algo_status=failed · (+2)`)+ 右侧展开按钮
- [ ] 展开 UI:现有完整 facet 面板
- [ ] 状态存 `localStorage['assets:facet-collapsed']`(string `"true"`/`"false"`);默认 `true`
- [ ] Vitest:toggle 后 localStorage 值正确;初次加载时读取值

## #3 KPI 概览卡片

- [ ] 新组件 `Frontend/src/components/assets/AssetsKpiRow.tsx`
- [ ] 4 个 antd `Statistic` card(总数 / ready:created / algo failure 30d / expire 30d)
- [ ] 数据源:复用 `useAssetsDiscovery` 现有 fetch 结果(facet counts + algo_status);无额外 API
- [ ] 若某个 metric 数据不可用(如 expire_at 无 facet),显示 `—`,不阻塞其他 card
- [ ] 挂在 `AssetsPage` 顶部,列表上方(facet 面板下方)
- [ ] Vitest 覆盖:mock discovery result,断言 4 个 card 数值

## #4 asset_id 副标题

- [ ] `AssetsResultsPane`(表格)`asset_id` 列 `customRender`:主体 `<Text code>{id}</Text>` + 下面一行 `<Text type="secondary" size="xs">{owner} · {asset_type} · {formatShort(created_at)}</Text>`
- [ ] `AssetsCardView` 卡片视图同款处理(asset_id 主 + owner/type/date 副)
- [ ] 短日期格式(如 `07-13`),复用 `Frontend/src/lib/dateTime.ts` 现有 helper 若有
- [ ] Vitest:snapshot 或结构断言两列内容都在

## #5 预览侧栏默认态

- [ ] `AssetQuickPreviewPane`:`selectedAssetId == null` 时渲染新 `<QuickPreviewEmptyState />` 子组件
- [ ] 新 hook `useRecentViewedAssets()`:读写 `localStorage['assets:recent-viewed']`(JSON array, 最多 5 项,LRU),暴露 `recent` + `pushRecent(id)`
- [ ] Empty state UI:
  - 上部保留「点击行/卡片查看预览」文案
  - 下部「最近查看」列表(每项:id + owner + short date,点击 → 调用现有 `onSelectAsset(id)` 或 `navigate(/assets/${id})`)
- [ ] 每次 `AssetDetail` 或 preview 打开一个新 id 时调 `pushRecent(id)`(单一入口)
- [ ] Vitest:pushRecent LRU 行为(5+1 挤掉最旧;重复 push 提到最前)

## 通用测试 / verification

- [ ] `cd Frontend && ./node_modules/.bin/biome check src/`(所有触及文件)
- [ ] `cd Frontend && npx tsc --noEmit`(全库)
- [ ] `cd Frontend && npm run build`(Tier L)
- [ ] 前端 Vitest 触及包(注意 memory `project_frontend_tests_bitrot` — 若 @testing-library/dom peer dep 漂移则同 CYB-3379 处理:decisions.md 记录,不阻塞 PR)

## Deploy + 验证

- [ ] merge 到 dev 后 deploy-dev workflow 自动触发 frontend 部署
- [ ] Chrome DevTools MCP 打开 `/assets`:
  - [ ] **#1**:搜索栏输入 `asset_type:segment`,断言 chip 显示结构化条件(不是 fulltext);列表显示 segment 资产
  - [ ] **#2**:首次访问 facet 折叠;点开;刷新页面后仍展开(localStorage 记住)
  - [ ] **#3**:4 个 KPI card 顶部可见,数字合理(总数 ≥1)
  - [ ] **#4**:表格首行 asset_id 下方有副标题(owner + type + date)
  - [ ] **#5**:未点击行时,右侧显示「最近查看」列表(初次访问为空 → 提示);点几个后有累加

## PR

- [ ] `gh pr create --base dev`,Linear ID = CYB-3382,贴 Tier L 输出 + Chrome MCP 截图
- [ ] **等用户 review 后由用户合并**
