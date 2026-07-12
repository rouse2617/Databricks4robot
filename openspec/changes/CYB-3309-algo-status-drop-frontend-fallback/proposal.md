# CYB-3309 — algo_status 检索:删除前端冗余降级,信任后端精确筛选

## Why

`/assets` 结构化检索里 `algo_status`(算法状态)筛选表现为"点了不准":点快捷 chip「算法失败」后列表/总数不对、还弹"仅当前页生效"黄条。

**运行时实测(dev `a1fac565`)推翻了"后端筛不了"的最初假设**:

- `POST /api/v1/queries/run`,`where:{pred:{field:"algo_status",op:"eq",value:"failed"}}`
  - list 查询 → `debug_plan: engine=postgres, mode=filter`、`total: 0`、`items: null`
  - facets 查询(同 where)→ `total: 0`
- dev 上确无 algo_status=failed 的资产(全 pending/blocked),`total:0` 是精确正确的。

后端早已支持:`algo_status` 是注册表里 `filter_engines:[postgres]` 的虚拟字段,`backend/internal/filter/types.go` 编译成 `EXISTS(SELECT 1 FROM asset_algo_latest ... status = ?)`,且 `handlers/query/handler.go` 在 `where != nil` 时强制走 PG COUNT → total 精确。

**病根 100% 在前端** `Frontend/src/hooks/assets/useAssetsDiscoveryReducer.ts` 的一套过时降级:

1. 对**已被后端筛过**的结果再跑一遍 client 端 `assetMatchesAlgoStatusFilter`——而它读的 `asset.algo_results` **不在 list 查询的 select 字段里**(select 仅 `asset_id/asset_type/lifecycle_state/owner/duration/updated_at` 6 列),故恒为空、逻辑本身即坏。
2. 无条件 push 黄条「algo_status 筛选仅在当前页生效…总数仍为全量」——而后端 total 其实精确(实测 total=0 仍弹此条,自证矛盾)。

## What Changes

**纯前端**,删掉 algo_status 专属降级,让它和其它服务端字段一样直接信任后端结果:

`Frontend/src/hooks/assets/useAssetsDiscoveryReducer.ts`
1. 删 `assetMatchesAlgoStatusFilter` 函数
2. 删 `activeAlgoStatusFilters` useMemo 及其在 results-fetch effect 依赖数组中的引用
3. results-fetch:`items = fetchedItems`(去掉 `hasAlgoStatusFallback` client 过滤分支)
4. 去掉 algo_status 黄条 push(`warnings` 直接用 `data.warnings`)
5. `total` 的 `cachedAuthoritativeTotal`(facets count)逻辑**保持不动**——facets 查询同样带 algo_status where,实测 total 精确;它是通用的 ES-fallback 兜底,与本次无关。

`Frontend/src/pages/AssetsPage.tsx`
6. 删 `hasAlgoStatusWarning` 变量(源头 warning 不再产生 → 死代码)
7. 删 Alert `message` 三元里的 `hasAlgoStatusWarning` 分支(保留 searchFallback + 默认两分支)

## Impact

- 行为:`algo_status:eq:*` 走后端精确过滤,列表 / total / 分页三者一致;不再有误导性黄条。
- 语义:后端为 **EXISTS-any**(`failed` = 该资产至少有一个算法处于 failed),对"找失败/运行中"正是所需,保留。
- 零后端改动、零 migration、零 ES reindex、不触 off-limits。契合最小化原则。
- `algo_status` facet 侧栏无服务端计数——这是**现状**(algo_status 未进 facet 聚合白名单),删降级不引入新回归。
- 组合 fulltext + algo_status 时 ES 返 0 候选、后端静默回退全量 PG 扫描(结果仍对,丢 ES recall 收益)——既有行为,不在本次范围。

## Out of scope

- 物化列 / 触发器(原方案 A2):数据量用不上,过度工程,不做。
- 整体 rollup 语义(全 ok 才 ok / failed 优先):当前 EXISTS-any 已满足用户需求,不改。
- facet 面板 UI/UX(手风琴/占位符黑话等)→ CYB-3310。
