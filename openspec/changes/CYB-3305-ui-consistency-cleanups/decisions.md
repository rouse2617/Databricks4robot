# Decisions — CYB-3305

## 2026-07-11 — C1 数据源副标题合并到页面级说明

- **Context**: proposal 里 C 项承诺给 Dashboard KPI 卡加数据源副标题/tooltip;实际发现 `DashboardPage.tsx:1335` 已有页面级说明「数据来源:湖仓(Iceberg via BigQuery)·」。
- **Decision**: 不再在每张 KPI 卡里重复标注数据源,只在 `/assets` 列表统计条加「· 数据源:PostgreSQL 实时 (+ title tooltip 交叉指向 Dashboard 湖仓)」;两处口径已可对照解释。
- **Alternatives**: (a) 每张 KPI 卡加副标题 —— 视觉冗余;(b) 在 Dashboard 顶部再加一句 —— 与已有 line 1335 重复。
- **Rationale**: 满足「口径可见」目标,不新增 UI 噪声。

## 2026-07-11 — 后端 facet 白名单零改动

- **Context**: proposal A.10 项写「后端 facet 聚合白名单加 raw_mcap/action/frame」。摸抓手发现前端 `AssetsFacetSidebar.tsx` 的 `fieldCounts.asset_type` 直接从 ES aggregation buckets 生成,并非按前端硬编码白名单过滤;后端 ES agg 也不限白名单,只要 index 里有 doc 就出 bucket。
- **Decision**: 后端零改动。前端新增 `mergeAssetTypeOptions` 合并「硬编码兜底 + backend counts 里出现的所有 key」,即使有新 asset_type 出现也能自动上屏。
- **Alternatives**: 后端加白名单常量 —— 与前端职责重叠,反而多一处需要跟着 asset_type 集合演化的地方。
- **Rationale**: 单一真相源在 `Frontend/src/lib/assets/assetTypes.ts`;后端 ES 不管、直传 bucket。

## 2026-07-11 — Baseline test drift 不动

- **Context**: 修改前跑 `npm run test -- --run`,基线 **8 fail / 646 pass**,6 个 test file 各有 pre-existing fail(包括 `OverviewTab.test.tsx` 加载报错、`useAssetsDiscoveryReducer.test.ts` 3 条,与 CYB-3301 刚合入的 decouple facets fetch 相关)。
- **Decision**: 本 PR 只修与 CYB-3305 直接相关的 test drift(`AssetsFacetSidebar.test.tsx` 的 `GROUP_KEYS` 5→7、`ASSET_TYPE_OPTIONS` 断言更新 + 新加 `mergeAssetTypeOptions` 用例)。其它 5 个 baseline fail 不动。
- **Alternatives**: 顺手扫全部 test drift —— 与 CYB-3305 范围无关,违反最小化方案原则(memory: 拒绝越权 refactor)。
- **Rationale**: 遵循「不 refactor 或清理无关代码」的 AI-RULES,避免 PR 变大;baseline 漂移交给单独 CYB tracking。
