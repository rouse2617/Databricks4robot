# Tasks — CYB-3305

## A. 资产类型 facet 完整性

- [ ] `Frontend/src/components/assets/AssetsFacetSidebar.tsx`
  - [ ] `ASSET_TYPE_OPTIONS` 加入 `raw_mcap`, `action`, `frame`(与后端一致)
  - [ ] 新增导出 `mergeAssetTypeOptions(counts?: Record<string,number>): string[]`——合并硬编码 + backend counts 中出现的 key,保序去重
  - [ ] 「资产类型」组渲染改用 `mergeAssetTypeOptions(facets?.asset_type_agg)`,若 count > 0 显示 `type (N)`
- [ ] `Frontend/src/components/assets/AddFilterPopover.tsx`
  - [ ] 删除自己那份 `["segment","clip","frame_set","derived_asset"]` 硬编码,`import { ASSET_TYPE_OPTIONS } from './AssetsFacetSidebar'`(或抽到 `lib/assets/assetTypes.ts` 供两处 import)
- [ ] `Frontend/src/components/assets/AssetsFacetSidebar.test.tsx`
  - [ ] 更新 `ASSET_TYPE_OPTIONS` 断言到新集合
  - [ ] 新增测试:backend counts 里出现「新类型」应被 `mergeAssetTypeOptions` 追加显示

## B. UI/UX 小问题 6 条

- [ ] `Frontend/src/components/AppLayout.tsx` `resolveSelectedKey`
  - [ ] 加 `if (pathname.startsWith("/api-keys")) return "/api-keys";`(放在 `/algo` 之前避免歧义)
- [ ] `Frontend/src/pages/DashboardPage.tsx` 「主力事件类型」KPI 卡
  - [ ] 值元素加 `title` + `text-overflow: ellipsis` + 字号夹到 `clamp(1rem, 2.4vw, 1.5rem)` 或 `Statistic` valueStyle `maxWidth`
  - [ ] 「事件分布」ECharts option `legend.type = "scroll"`;或缩短第 4 项 legend label
- [ ] `Frontend/src/components/asset-detail/TagsTab.tsx` 来源 badge
  - [ ] 按 `tag.source_type` 三档渲染:`human` (蓝) / `algo` (紫) / `system` (灰);优先使用 `source_type`,回退 `source_name`
- [ ] `Frontend/src/pages/McapFilesPage.tsx` 大小列
  - [ ] 汇总条目(.mp4 且 `size_bytes>0`)展示 `formatBytes`;`size_bytes==0` 显示「未记录」而非「—」
  - [ ] 若后端不下发汇总 size,标注 TODO 并沿用「未记录」
- [ ] `Frontend/src/pages/RegistryCenterPage.tsx` 资源池
  - [ ] 顶部说明「受 ResourceQuota 约束」改为条件文案:所有行 CPU/MEM 都 `—` 时显示「当前命名空间未配置 ResourceQuota」

## C. 数据源口径可见化

- [ ] `Frontend/src/pages/DashboardPage.tsx`
  - [ ] KPI 面板小字副标题:`数据来源:湖仓(Iceberg via BigQuery)· 按日聚合`(项目已有类似文案,统一到 KPI 卡片附近)
- [ ] `Frontend/src/pages/AssetsPage.tsx` 或列表工具条
  - [ ] 统计条「共 N 条」后加小灰字 `· PG 实时`

## Backend(视 A 项前端合并策略而定)

- [ ] `backend/internal/queryexec/elasticsearch/compile.go` 或 `internal/config/query_field_registry.go`
  - [ ] 若 asset_type facet bucket 白名单过滤了 `raw_mcap`/`action`/`frame`,把这三类加入白名单
  - [ ] 若 backend 是「不限白名单,ES agg 直出」则**零改动**(A 项 mergeAssetTypeOptions 已够)
- [ ] `cd backend && go test ./internal/queryexec/... ./internal/handlers/query/...`

## 本地验证(Tier M)

- [ ] `cd Frontend && npm run lint && npm run test -- --run && npm run build`
- [ ] `cd backend && make fmt && make vet`(有后端改动才跑 `go test`)

## Deploy dev + 验收

- [ ] `USE_EXISTING_IMAGE=true bash deploy/cloudrun/frontend-dev.sh`(必)
- [ ] `USE_EXISTING_IMAGE=true bash deploy/cloudrun/backend-dev.sh`(仅当动了 backend)
- [ ] Chrome DevTools MCP 在 dev 上逐条验证:
  - [ ] `/assets` facet 显示 `raw_mcap (N)` / `action (N)` / `frame (N)`;count 合计 ≈ 「共 X 条」
  - [ ] `/api-keys` 侧栏高亮「API 密钥」
  - [ ] `/dashboard` 主力事件类型 KPI 卡文字不溢出;图例不截断
  - [ ] `/assets/<有 human 标签的 asset>` 标签 tab 来源 badge = human
  - [ ] `/mcap-files` 已汇总条目显示「未记录」或 size 数字
  - [ ] `/registry` 资源池文案与实际 quota 一致
  - [ ] `/dashboard` 有数据源副标题;`/assets` 列表统计有「· PG 实时」
- [ ] 每条截图存 `openspec/changes/CYB-3305-ui-consistency-cleanups/deploy-verify-<n>.png`

## PR

- [ ] `gh pr create` 填 `.github/pull_request_template.md`,Linear=CYB-3305,贴 revision + 截图路径
