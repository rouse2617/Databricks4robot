# CYB-3382 — Assets 页 UX 优化包

## Why

Assets Discovery 页 UX 走查发现 5 个真实短板。**#1 是真 bug**(搜索 DSL 认不出 `asset_type`,把它降级为 fulltext,导致「输了明明存在的过滤条件却找不到资产」—— 信任杀手);#2–5 是明显 UX gap(信息密度过密、缺 KPI、id 难识读、预览侧栏死区)。

按用户明确要求打包一起改,不再拆 5 个 PR(避免每次都单独走 CI/deploy 循环)。

## What Changes(纯前端,后端零改动)

**#1** `Frontend/src/components/assets/AssetsSearchBar.tsx` `keys` 数组补 `asset_type`(1 行);帮助浮层「可用字段」列表(:520 附近)同步加。

**#2** Assets 页顶部 facet 面板默认折叠成「筛选摘要行 + 展开按钮」。状态存 `localStorage['assets:facet-collapsed']`,初值 `true`(默认收起)。

**#3** `AssetsPage` 顶部加 4 个 KPI card:
- 总数(现有 `total`)
- ready:created 比例(facet counts)
- 近 30 天算法失败数(现有 algo_status facet)
- 近 30 天即将过期数(现有 `metadata.expire_at` / retention 逻辑,若无数据则跳过卡片)

**#4** 表格 / 卡片视图里 `asset_id` 显示层次调整:主体仍是 8 位 id(可拷贝、URL 友好),**旁边加副标题** `owner · asset_type · created_at`(短格式)。

**#5** `AssetQuickPreviewPane` 未选中状态显示两部分:
- 顶部保留「点击行/卡片查看预览」提示
- 下方「最近查看的 5 个资产」列表 —— 用 `localStorage['assets:recent-viewed']` 记 LRU,点击回跳。有选中时不变。

## Impact

- 纯 UI 层改动,无 API / schema / URL 结构变化
- 向后兼容:localStorage 缺失时按初值(#2 collapsed=true, #5 recent=空列表)
- 不引入新依赖 / 新抽象 —— 全用现有 antd + 现有 hook

## Out of scope

- ❌ 通用 KPI 组件抽象(第一个真实场景,先硬编码 4 个 card;真需要通用抽象等第二个 KPI 页面)
- ❌ 后端改动
- ❌ AssetsPage 整体布局重构
- ❌ asset_id 完全弱化(讨论过:id 是产品的可寻址标识,弱化会误导)
- ❌ URL 结构改动(现 `/assets` 已有 query state,不动)

## 验证

- Vitest 覆盖新增逻辑(#1 keys 解析、#2 折叠 toggle + localStorage、#5 recent LRU)
- Chrome DevTools MCP 手动跑 5 个 acceptance(见 tasks.md 「Deploy + 验证」段)
- 合并前 Tier L:tsc / biome / build
