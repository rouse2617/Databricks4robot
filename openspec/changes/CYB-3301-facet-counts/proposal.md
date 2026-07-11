# CYB-3301 — 修复 facet 计数不上屏

## Why
AssetsPage facet 侧栏所有 facet 都不显示计数（连始终可见的「资产类型」也没有 `(144)`），因此 env 组（渲染门槛=计数非空）也不显示。dev 实测：facets 请求已发出并返回完整 buckets（asset_type/env/lifecycle/owner/tag.priority 都有值），但 UI 无计数。

## Root cause
facets 请求原来**链在 list 请求之后、同一个 effect 里**。list 成功 dispatch `RESULTS_SUCCESS` → `isStale` 由 true 翻 false → 该 effect 依赖变化被重跑 → 触发其 cleanup（`cancelled=true`）—— 而此时同一 run 的 facets fetch 还在飞行中。facets 响应到达后命中 `if (cancelled) return`，**FACETS_SUCCESS 从未 dispatch**，aggregations 永不进 state → 所有 facet 无计数。

## What Changes
`useAssetsDiscoveryReducer.ts`：把 facets 抓取**抽成独立 useEffect**，只按 filter key（`facetsRequestKey`）触发，与 list 的 `isStale` 无关；用 `facetsRequestKeyRef` 判定「filters 变了就丢弃」。list effect 不再链 facets。

## Impact
- 纯前端，行为等价（list 请求仍不带 facets；facets 仍是单独一次请求，只是不再被 list 的 isStale churn 取消）。
- 无后端变更。修好后 env / asset_type / lifecycle / owner / tag.priority 计数都会上屏。
- 现有 reducer 单测（list-then-facets、page 变不重取 facets）仍成立。

## 验证
- dev：全新加载 /assets → 「资产类型」显示 `segment (N)` 等计数；展开「采集」→「环境」出现真实值带计数。
