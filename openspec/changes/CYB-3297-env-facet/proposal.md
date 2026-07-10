# CYB-3297 Phase B — env 可发现（动态 facet）

## Why
用户最初痛点：「不知道数据里有哪些 env」。现状：env 存 `metadata.env`（flattened，已索引）；env **过滤已可用**（PG `assets_fields.go:60` env→metadata.env，前端也已暴露 env 过滤）。但 env **不作为 facet 出现**：
- 后端 `query_ir.go facetFieldPath` 对 `env` 报 `unsupported facet field`；
- 前端 facet 侧栏用**写死的 `ENV_OPTIONS`**（5 个英文值），且被 `fieldCounts.env` 空值 gate 住 → 环境筛选组根本不渲染。
真实数据是自由中文值（如 `工作区`/`家庭`），写死清单永远对不上。

## What Changes
纯「让 env facet 通起来 + 选项由聚合动态生成」，不改索引内容：
1. 后端 `internal/elasticsearch/query_ir.go`：`facetFieldPath` 增加 `case "env": return "metadata.env"`（flattened 子字段 terms agg）。
2. 后端 `config/query_field_registry.yaml`：登记 `env`（`filter_engines:[postgres,elasticsearch]`、`facet_engines:[elasticsearch]`），与实际路由一致。
3. 前端 `useAssetsDiscoveryReducer.ts`：`ASSET_DISCOVERY_FACETS` 增加 `{ field:"env", size:50 }`；`mapFacetFieldToAggregationKey` 增加 env→`env_agg`。
4. 前端 `AssetsFacetSidebar.tsx`：`AGG_KEY_MAP` 增加 `env_agg→env`；环境 facet **选项由 `fieldCounts.env` 的 key 动态生成**（不再用写死的 `ENV_OPTIONS`），保留空值 gate（现在会被填充）。

## Impact
- Affected code: query_ir.go / query_field_registry.yaml / useAssetsDiscoveryReducer.ts / AssetsFacetSidebar.tsx
- 无 schema / 无索引重建（metadata.env 已索引）。
- **验证不受 CYB-3298(b) 影响**：facet 由服务 revision（新代码，/version 已确认）在查询时聚合，不依赖订阅者写 doc。
- 风险低：只新增一个 facet 字段与动态选项；env 过滤本就走 PG，点选即生效。

## 验证（dev）
- AssetsPage「采集」组出现「环境」，选项为真实值（如 `工作区`/`家庭`）带计数。
- 点某 env 值 → 结果按该 env 过滤（PG）。
