# Design — CYB-3385

## 候选方案

### 方案 A(采用):前端 reducer 分组编译

在 `buildStructuredQueryWhere` 里按 `(field, op)` 分组 chip,同组 OR,跨组 AND。

**Pro**
- 单点改动(1 函数 ~30 行)
- 无 chip 数据结构变更 → UI 展示 / URL hydration / saved views 全兼容
- 无后端改动 → 单侧发布,dev 直接生效
- 满足 memory `feedback_prefer_minimal_infra`

**Con**
- 编译成 `{or:[...]}` 会让 debug_plan 里 where clause 深度增加(可以接受,已有 or/and 表达)

### 方案 B(不采用):合并同 field chip 为单个 chip with array value + 引入 op="in"

`FACET_TOGGLE` 检测同 field 已有 chip → 更新 value 为数组;后端 `buildScalarClause` 加 `case "in"` 支持。

**Con**
- chip 数据结构变更 → ActiveFilterChipsRow 渲染逻辑要改(测试里显示 `field = value` 变 `field in [v1,v2]`)
- 后端 op="in" 需要 ES `terms` + PG `IN (...)` 两个 executor 都改
- 涉及后端 → 需要 dev deploy + 回归 → 与 前端修改捆绑
- 收益仅是 payload 稍紧凑,不值得

### 方案 C(不采用):FACET_TOGGLE reducer 生成 or predicate

在 dispatch 层直接把同 field chip 合并生成 `{or:[...]}` chip data。

**Con**
- 混合 UI state 和 query IR 语义,面向未来不清晰

## 关键决策

| 决策 | 选择 | 原因 |
|---|---|---|
| 分组 key | `(field, op)` | eq 和 ne 需分开(`alice OR bob` AND `NOT charlie`)|
| 单 chip 返回 shape | 保持原有 `{pred:...}` | 不给单值情况多加一层 or 包裹,便于 backend 编译效率 |
| 未参与分组的 chip(range/date)| 也走同一分组机制 | 一致性;range 通常单 chip 一 field,不会被误 OR |
| 前后端接口 | 不改 | IR schema `{or:...}` 早已支持 |

## 影响面

**修改**
- `Frontend/src/hooks/assets/useAssetsDiscoveryReducer.ts` — `buildStructuredQueryWhere` 单函数
- `Frontend/src/hooks/assets/useAssetsDiscoveryReducer.test.ts` — 补 5 用例

**不改**
- ActiveFilterChipsRow / QuickFiltersRow / AssetsFacetSidebar / saved views / URL hydration
- 后端 planner / executor / IR / postgres SQL / elasticsearch DSL
- 任何 config / env / migration
