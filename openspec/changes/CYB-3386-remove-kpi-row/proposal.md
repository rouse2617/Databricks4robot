# CYB-3386 — 移除 Assets 页顶部 KPI Row

## Why
CYB-3382 #3 加了 4 张 KPI 卡片(总数 / ready:created / 算法失败 / 30 天即将过期),但只有「总数」正常显示,其他 3 张长期显示 `—`:

- **ready:created**:`bucketCount(agg, "lifecycle_state", ...)` 的 aggregation key 错了(实际 key 带 `_agg` 后缀,见 `mapFacetFieldToAggregationKey`)
- **算法失败**:`algo_status` 根本没在 `ASSET_DISCOVERY_FACETS` 里请求
- **30 天即将过期**:后端没有 `expiring_30d` aggregation 出口,是 CYB-3382 时留的 stub `(未来加)`

3/4 显示 `—` 长期误导用户;概览页(Dashboard)本身也承担类似指标,资产列表页的 KPI Row 属于冗余 + 半成品。 用户明确选择:**"都不要好了,概览界面其实有的,去除吧"**。

## What Changes
- 从 `AssetsPage.tsx` 移除 `<AssetsKpiRow>` 元素及其 import
- 删除 `AssetsKpiRow.tsx` + `AssetsKpiRow.test.tsx`(若存在)
- 保留 `bucketCount` 若他处引用(检查后决定)

## Impact
- **面向用户**:资产页顶部空间回归给筛选 chips row + 搜索栏;不再有 `—` 尴尬占位
- **数据准确性关切消除**:不虚构指标
- **无功能损失**:概览页(`/dashboard`)已有指标,资产列表的核心操作(筛选 / 列表 / 预览)完全不受影响
- 无后端 / API 变更

## 验证
- Tier L:tsc + biome + reducer 相关 test 全绿(不涉及 hook 逻辑改动)
- dev 回归:Chrome MCP → /assets → 页面顶部无 KPI Row,总数在表格头部「共 X 条」处仍可见
