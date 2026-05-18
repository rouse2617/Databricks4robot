# 查询平台设计（Query IR / Planner / Executors）

> **状态**：提案 / 部分实现（桥接态）  
> **目标版本**：作为 1.0 之后的统一查询内核，为后续 `saved query / slice / export / scenario test / 多引擎检索` 提供底盘。  
> **适用前提**：当前项目尚未正式上线，允许废弃现有 `backend/internal/filter` 的 `field:op:value` 方案，不做兼容包袱。

---

## 状态分层（Now / Next / Later）

为避免“提案和现状混在一起”，本设计按交付时序划分：

- **Now（v1）**：必须在第一阶段落地，且作为上线门槛
- **Next（v2）**：依赖 v1 稳定后迭代
- **Later（v3+）**：长期能力，不阻塞主线

| 章节 | 状态层 | 说明 |
|---|---|---|
| `3. Query IR v1` | Now | 仅 `assets` 资源，JSON 协议 |
| `4. 字段注册中心` | Now | 字段语义、操作符和引擎绑定单一真源 |
| `5. Planner` | Now | 规则规划（不做复杂 cost-based） |
| `6. Executors / Compilers` | Now | `PostgreSQL` 主执行，`Elasticsearch` 辅执行 |
| `7. API 设计` | Now | 仅 `queries/validate` / `queries/run` |
| `8. Saved Query / Slice / Export / Scenario Test` | Next | 建立在 Query API 之上 |
| `9~12. 前端状态与对齐` | Next | 分步迁移到 Query Workbench |
| `13. 框架选型` | Now | 约束边界，避免错误依赖路径 |
| `14. 与当前 filter 关系` | Now | 明确未上线场景可直接切换 |
| `15. 分阶段落地` | Now/Next/Later | 执行计划与验收标准 |

---

## 0.1 当前实现进度（代码现状，不等于目标架构）

为避免“文档目标”与“仓库现状”混淆，这里明确当前状态：

1. 已有 `queries/validate`、`queries/run` 接口与 `queryir` 包雏形；
2. 当前主链路仍是 **`Query IR -> 旧 filter 字符串 -> 旧 PG 查询`** 的桥接；
3. 字段注册中心已有最小配置与加载器雏形，但尚未形成 planner/executor 的统一真源；
4. 已有最小 `queryplan` 与 `queryexec/postgres` 雏形，但仍以桥接旧 filter 路径为主；
5. 运行时已进入 `ES recall + PG refine` 过渡主路径，纯 PG 作为回退而非默认主链。

当前与目标协议的主要分叉（非常重要：不要把目标态当成已落地）：

- **表达式树**：当前 IR 形状允许 `and/or/not/pred`，但 **v1 bridge 执行仅支持 `pred` / `and`**（`or/not` 返回 `UNPLANNABLE_QUERY`）。
- **分页**：目标分页是 `offset/limit`；当前实现是 `page/page_size`。
- **返回字段**：目标返回是 `normalized_query / field_capabilities / warnings / debug_plan / facets`；当前实现覆盖其中子集（以 `api/openapi.yaml` 为准）。

本文档后续章节以“目标架构”为准；上述现状仅用于排期与上线前收敛评估。

---

## 0.2 章节现状对照（目标 vs 当前）

下表用于快速回答“已经做到哪一步”，避免把目标态误读为已落地能力。

| 章节 | 目标主张 | 当前状态（2026-05） | 结论 |
|---|---|---|---|
| 3. Query IR v1 | `and/or/not/pred` 表达式树 + 统一协议 | 当前已切到树形协议，但 v1 bridge **仅 `pred/and` 可执行**（`or/not` 返回 `UNPLANNABLE_QUERY`） | 半实现（协议仍有历史兼容字段） |
| 4. 字段注册中心 | 单一真源（字段能力 + 引擎绑定） | 已有最小 YAML/loader，但字段语义仍未完全从 filter/ES/前端收口 | 半实现（未成为统一真源） |
| 5. Planner | 独立 `queryplan`，产出可解释 plan | 已有最小 planner，支持 ES recall + PG refine 路由，但规则仍偏保守 | 半实现（规则未充分展开） |
| 6. Executors/Compilers | `queryexec/*` 分层执行 | 已有 `queryexec/postgres` 与 `queryexec/elasticsearch`，PG refine 仍部分复用旧 filter 体系 | 半实现（仍在桥接） |
| 7. Query API | `validate/run` 返回 normalized/debug/facets 能力 | `validate/run` 已返回 normalized/debug/facets/warnings；当前主路径已统一到 structured/keyword，semantic/similar 仍是工作台模式标签并映射到上述能力 | 半实现 |
| 8~12 业务能力 | saved query/slice/export/scenario + 前端全接入 | `saved_queries` 已有后端 CRUD、工作台接入和独立管理页；slice/export/scenario 仍未实现 | 半实现 |
| 14. 旧 filter 关系 | 停止扩展并迁移到 Query API 主路径 | Query API 已成为工作台主路径，但 PG refine 仍桥接旧 filter 编译 | 偏离（收口后期） |
| 15. 分阶段任务 | QP-001~015 有序推进 | 目前完成少量协议/API 壳层 | 半实现 |

---

## 0. 背景与结论

当前仓库已经出现典型信号：

1. 资产筛选逻辑开始跨 `PostgreSQL` 与 `Elasticsearch`
2. 后续产品明确会有 `saved query / slice / export / test-run`
3. 未来存在 `Qdrant` / 向量检索 / 多模态检索的接入空间

继续在 handler / usecase / repo 中手工拼接 SQL、ES DSL 和布尔逻辑，会让查询语义散落在各层代码里，后续任何新能力都会复制一遍路由和过滤逻辑。

**结论**：

- 不继续扩展现有 `field:op:value`
- 不先发明一门文本 DSL
- 直接引入新的统一查询内核：

```text
Frontend / SDK / CLI
        |
   Query API (JSON)
        |
     Query IR
        |
      Planner
   /      |      \
 PG   Elasticsearch  Qdrant/Trino
```

其中：

- `Query IR` 负责表达用户意图
- `Planner` 负责决定执行计划
- `Executors` 负责和具体引擎打交道

---

## 1. 设计目标

### 1.1 必须解决

1. 用**统一协议**表达结构化查询
2. 让查询逻辑**不再绑定** `SQL` / `ES DSL`
3. 支持后续 `saved query / slice / export / scenario test`
4. 允许未来新增 `Qdrant`、替换 `ES`、增加 `Trino`

### 1.2 当前不做

1. 不做自然语言查询
2. 不做多模态向量检索
3. 不做可视化 query builder 细节
4. 不做跨资源 join 查询语言
5. 不做 `saved query / slice / export / scenario test` 的完整产品化接口（放到 v2）
6. 不做 `Qdrant / Trino` 生产执行（v1 只保留接口位）

---

## 2. 核心原则

### 2.1 引擎无关

对外协议中禁止出现任何引擎细节：

- 禁止 `sql_where`
- 禁止 `es_query`
- 禁止 `qdrant_filter`

API 只接受业务字段与布尔结构。

### 2.2 字段语义先于物理字段

查询使用业务字段，例如：

- `owner`
- `lifecycle_state`
- `tag.priority`
- `algo.hand_tracking.status`
- `action.label`

不得在查询协议中暴露：

- `asset_tags.tag_value`
- `metadata.env`
- `tags_flat.priority.keyword`

物理映射由字段注册中心负责。

### 2.3 Planner 与 Compiler 分离

- `Planner` 决定走哪个引擎、先后顺序、是否 merge / post-filter
- `Compiler` 只把某个 plan step 翻译成某个引擎的查询

Planner 不写 SQL，Compiler 不决定业务语义。

### 2.4 Saved Query / Slice 存 Query IR，不存引擎语法

数据库里保存的是 `JSON Query IR + schema version`，不是 SQL 文本，也不是 ES DSL。

这样以后更换执行引擎，历史 query 不需要重写。

---

## 3. Query IR v1

> **实现注记（当前）**：本章定义的是目标协议；当前代码已切到树形 `where` 协议，但整体仍处于桥接执行阶段（`Query IR -> 旧 filter/sort -> PG`，分页仍为 `page/page_size`）。  
> 上线前策略：不做双轨长期兼容，直接收敛到唯一目标协议，并删除桥接协议与转换代码。

### 3.1 选择 JSON，而不是文本 DSL

`v1` 采用 **JSON Query IR**，原因：

1. SDK 最好写
2. 前端 query builder 最容易生成
3. 校验和版本演进最稳
4. 不需要先投入 parser 维护成本

文本 DSL 可以作为 `v2` 的语法糖入口，先不作为主协议。

### 3.2 顶层结构

```json
{
  "schema_version": "v1",
  "scope": { "resource": "assets" },
  "select": {
    "fields": ["asset_id", "owner", "created_at"]
  },
  "where": {
    "and": [
      { "pred": { "field": "owner", "op": "eq", "value": "team_alpha" } },
      { "pred": { "field": "tag.priority", "op": "in", "value": ["high", "critical"] } },
      { "pred": { "field": "lifecycle_state", "op": "ne", "value": "archived" } }
    ]
  },
  "sort": [
    { "field": "created_at", "direction": "desc" }
  ],
  "page": { "page": 1, "page_size": 50 },
  "facets": [
    { "field": "tag.priority", "size": 10 }
  ],
  "debug": { "explain": true }
}
```

### 3.3 节点模型

`where` 由布尔表达式树组成：

- `and`
- `or`
- `not`
- `pred`

谓词节点：

```json
{ "pred": { "field": "owner", "op": "eq", "value": "team_alpha" } }
```

### 3.4 v1 支持的操作符

- `eq`
- `ne`
- `gt`
- `gte`
- `lt`
- `lte`
- `in`
- `between`
- `like`
- `ilike`
- `exists`

### 3.5 v1 支持的资源范围

`v1` 仅支持：

- `resource = assets`

后续（v2/v3）资源可以扩展：

- `actions`
- `deliveries`
- `dataset_snapshots`
- `training_runs`

### 3.6 Go 类型草案

后端建议定义独立的 `queryir` 包，不复用现有 `filter.Filter`：

```go
package queryir

type Query struct {
    SchemaVersion int     `json:"schema_version"`
    Scope         Scope   `json:"scope"`
    Select        *Select `json:"select,omitempty"`
    Where         *Expr   `json:"where,omitempty"`
    Sort          []Sort  `json:"sort,omitempty"`
    Page          *Page   `json:"page,omitempty"`
    Facets        []Facet `json:"facets,omitempty"`
    Debug         *Debug  `json:"debug,omitempty"`
}

type Scope struct {
    Resource string `json:"resource"` // "assets"
}

type Select struct {
    Fields []string `json:"fields,omitempty"`
}

type Expr struct {
    And  []Expr     `json:"and,omitempty"`
    Or   []Expr     `json:"or,omitempty"`
    Not  *Expr      `json:"not,omitempty"`
    Pred *Predicate `json:"pred,omitempty"`
}

type Predicate struct {
    Field string `json:"field"`
    Op    string `json:"op"`
    Value any    `json:"value,omitempty"`
}

type Sort struct {
    Field string `json:"field"`
    Desc  bool   `json:"desc,omitempty"`
}

type Page struct {
    Offset int `json:"offset"`
    Limit  int `json:"limit"`
}

type Facet struct {
    Field string `json:"field"`
    Size  int    `json:"size,omitempty"`
}

type Debug struct {
    Explain bool `json:"explain,omitempty"`
}
```

建议同时定义统一返回体：

```go
type ValidationResult struct {
    Valid             bool                   `json:"valid"`
    NormalizedQuery   *Query                 `json:"normalized_query,omitempty"`
    Warnings          []string               `json:"warnings,omitempty"`
    FieldCapabilities []FieldCapabilityBrief `json:"field_capabilities,omitempty"`
}

type FieldCapabilityBrief struct {
    Field   string   `json:"field"`
    Engines []string `json:"engines"`
}

type QueryRunResult struct {
    Columns   []ResultColumn             `json:"columns"`
    Items     []map[string]any           `json:"items"`
    Total     int64                      `json:"total"`
    Facets    map[string][]FacetBucket   `json:"facets,omitempty"`
    Page      Page                       `json:"page"`
    DebugPlan *DebugPlan                 `json:"debug_plan,omitempty"`
}

type ResultColumn struct {
    Name     string `json:"name"`
    Type     string `json:"type"` // string, enum, numeric, timestamp, boolean, json
    Nullable bool   `json:"nullable"`
}

type FacetBucket struct {
    Value string `json:"value"`
    Count int64  `json:"count"`
}

type DebugPlan struct {
    Steps []DebugPlanStep `json:"steps"`
}

type DebugPlanStep struct {
    Engine string `json:"engine"`
    Mode   string `json:"mode"`
}
```

---

## 4. 字段注册中心（Field Registry）

字段注册中心是解耦的关键。它负责回答：

1. 这个字段是什么类型
2. 支持哪些操作符
3. 支持哪些引擎执行
4. 在不同引擎里映射到什么物理字段
5. 是否支持排序 / facet / post-filter

### 4.1 建议结构

```go
type FieldCapability struct {
    Field            string
    ValueType        string   // string, enum, numeric, timestamp, boolean
    SupportedOps     []string
    FilterEngines    []string // postgres, elasticsearch, qdrant, trino
    SortEngines      []string
    FacetEngines     []string
    PostFilterOnly   bool
}

type EngineBinding struct {
    Engine      string
    Source      string // table/index/logical-view
    FieldPath   string // assets.owner / owner.keyword / tags_flat.priority.keyword
    ValueType   string // keyword, text, int64, timestamp, jsonb
    JoinHint    string // optional: asset_tags(tag_key=priority)
    AccessMode  string // direct, join, nested, exists
}
```

### 4.2 例子

| 业务字段 | PG | ES | 说明 |
|----------|----|----|------|
| `owner` | `assets.owner` | `owner.keyword` | 可 filter / sort |
| `tag.priority` | `asset_tags` 投影 | `tags_flat.priority.keyword` | 可 filter / facet |
| `action.label` | `actions.labels` / EXISTS | `actions.primary_label` nested | v1 可先只走 PG |
| `lifecycle_state` | `assets.lifecycle_state` | `lifecycle_state.keyword` | 可 filter / facet |

字段注册中心必须与执行器分离，避免在 compiler 内部写死业务字段语义。

### 4.3 建议落盘格式（YAML）

建议新增独立配置（例如 `backend/config/query_field_registry.yaml`）：

```yaml
schema_version: 1
resources:
  assets:
    fields:
      - field: owner
        type: string
        operators: [eq, ne, in, like, ilike]
        filter_engines: [postgres, elasticsearch]
        sort_engines: [postgres, elasticsearch]
        facet_engines: [elasticsearch]
        post_filter_only: false
        bindings:
          postgres:
            source: table
            field_path: assets.owner
            value_type: text
            access_mode: direct
          elasticsearch:
            source: index
            field_path: owner.keyword
            value_type: keyword
            access_mode: direct

      - field: lifecycle_state
        type: enum
        enum_values: [draft, ready, delivered, archived]
        operators: [eq, ne, in]
        filter_engines: [postgres, elasticsearch]
        sort_engines: [postgres, elasticsearch]
        facet_engines: [postgres, elasticsearch]
        post_filter_only: false
        bindings:
          postgres:
            source: table
            field_path: assets.lifecycle_state
            value_type: text
            access_mode: direct
          elasticsearch:
            source: index
            field_path: lifecycle_state.keyword
            value_type: keyword
            access_mode: direct

      - field: tag.priority
        type: enum
        enum_values: [low, medium, high, critical]
        operators: [eq, ne, in]
        filter_engines: [postgres, elasticsearch]
        sort_engines: []
        facet_engines: [elasticsearch]
        post_filter_only: false
        bindings:
          postgres:
            source: table
            field_path: asset_tags.tag_value
            value_type: text
            join_hint: "asset_tags(tag_key='priority')"
            access_mode: join
          elasticsearch:
            source: index
            field_path: tags_flat.priority.keyword
            value_type: keyword
            access_mode: direct

      - field: action.label
        type: string
        operators: [eq, ne, in]
        filter_engines: [postgres]
        sort_engines: []
        facet_engines: []
        post_filter_only: false
        bindings:
          postgres:
            source: table
            field_path: actions.labels
            value_type: text[]
            access_mode: exists
```

字段约束建议：

1. `field` 全局唯一（按 resource 命名空间）
2. `operators` 必须是协议白名单子集
3. `bindings` 必须覆盖 `filter_engines` 中声明的引擎
4. `sort_engines/facet_engines` 只能引用已存在引擎
5. `post_filter_only=true` 时，不允许出现在任一 `*_engines` 列表中
6. `bindings` 内禁止伪 SQL 字符串，必须为结构化对象

---

## 5. Planner

### 5.1 Planner 的职责

Planner 负责：

1. 校验 Query IR 是否可执行
2. 根据字段能力选择执行引擎
3. 产出执行计划
4. 决定哪些条件是 pushdown，哪些只能 post-filter

Planner 不做：

1. SQL 生成
2. ES DSL 拼装
3. 结果序列化

### 5.2 Plan 输出

```go
type Plan struct {
    Resource string
    Steps    []PlanStep
    Explain  []string
}

type PlanStep struct {
    Engine string // postgres, elasticsearch, qdrant, trino
    Mode   string // filter, recall, sort, facet, aggregate, post_filter
    Input  any
}
```

### 5.3 v1 规划策略

`v1` 先采用可解释的规则规划，不做复杂成本模型：

1. 纯结构化过滤，默认优先 `PostgreSQL`
2. 需要 facet / keyword search / 大规模召回时优先 `Elasticsearch`
3. 如果 sort 字段不被 ES 支持，则：
   - `ES` 先召回 `asset_id`
   - `PG` 再精过滤与排序
4. `PostFilterOnly` 字段一律放到后置过滤

### 5.4 为什么 v1 不上复杂 cost-based planner

当前项目还没正式上线，先做确定性、可 explain 的规则规划更合适：

- 更容易调试
- 更容易写测试
- 更容易观察 planner 是否把字段路由错了

---

## 6. Executors / Compilers

### 6.1 统一接口

```go
type QueryExecutor interface {
    Name() string
    Compile(step PlanStep) (CompiledQuery, error)
    Execute(ctx context.Context, q CompiledQuery) (ResultSet, error)
}
```

### 6.2 目录建议

```text
backend/internal/queryexec/
  interfaces.go
  postgres/
    compile.go
    execute.go
  elasticsearch/
    compile.go
    execute.go
  qdrant/
    compile.go
    execute.go
  trino/
    compile.go
    execute.go
```

### 6.3 目标阶段（v1 终态）实现范围

- `postgresExecutor`
- `elasticsearchExecutor`

`Qdrant` 与 `Trino` 只预留接口，不在 v1 实现。

---

## 7. API 设计

> **实现注记（当前）**：`/queries/validate` 与 `/queries/run` 已存在，但返回体目前仍是桥接期最小集合。  
> `normalized_query / field_capabilities / debug_plan / facets` 以阶段任务为准逐步补齐。

### 7.1 核心端点

- `POST /api/v1/queries/validate`
- `POST /api/v1/queries/run`

### 7.2 validate 返回

```json
{
  "valid": true,
  "normalized_query": { "...": "..." },
  "warnings": [],
  "field_capabilities": [
    { "field": "tag.priority", "engines": ["postgres", "elasticsearch"] }
  ]
}
```

### 7.3 run 返回（当前实现）

```json
{
  "items": [],
  "total": 0,
  "facets": {},
  "page": 1,
  "page_size": 50,
  "debug_plan": {
    "steps": [
      { "engine": "postgres", "mode": "filter" }
    ]
  }
}
```

### 7.4 后续建立在 Query API 之上的能力

- `saved_queries`
- `slices`
- `exports`
- `scenario_tests`

它们都引用同一份 `Query IR`。

### 7.5 OpenAPI 契约（当前实现以 `api/openapi.yaml` 为准）

建议新增两个主端点：

```yaml
/api/v1/queries:validate:
  post:
    summary: Validate Query IR
    requestBody:
      required: true
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/QueryValidateRequest'
    responses:
      "200":
        description: Query validation result
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/QueryValidateResponse'

/api/v1/queries:run:
  post:
    summary: Run Query IR
    requestBody:
      required: true
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/QueryRunRequest'
    responses:
      "200":
        description: Query execution result
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/QueryRunResponse'
```

组件 schema（目标态草案，当前实现不要照抄；当前请以 `api/openapi.yaml` 为准）：

```yaml
QueryIR (target):
  type: object
  required: [schema_version, scope]
  properties:
    schema_version:
      type: integer
      minimum: 1
      enum: [1]
    scope:
      $ref: '#/components/schemas/QueryScope'
    select:
      $ref: '#/components/schemas/QuerySelect'
    where:
      $ref: '#/components/schemas/QueryExpr'
    sort:
      type: array
      items:
        $ref: '#/components/schemas/QuerySort'
    page:
      $ref: '#/components/schemas/QueryPage'
    facets:
      type: array
      items:
        $ref: '#/components/schemas/QueryFacet'
    debug:
      $ref: '#/components/schemas/QueryDebug'

QueryExpr:
  oneOf:
    - type: object
      required: [and]
      additionalProperties: false
      properties:
        and:
          type: array
          minItems: 1
          items:
            $ref: '#/components/schemas/QueryExpr'
    - type: object
      required: [or]
      additionalProperties: false
      properties:
        or:
          type: array
          minItems: 1
          items:
            $ref: '#/components/schemas/QueryExpr'
    - type: object
      required: [not]
      additionalProperties: false
      properties:
        not:
          $ref: '#/components/schemas/QueryExpr'
    - type: object
      required: [pred]
      additionalProperties: false
      properties:
        pred:
          $ref: '#/components/schemas/QueryPredicate'

QueryValidateRequest:
  type: object
  required: [query]
  properties:
    query:
      $ref: '#/components/schemas/QueryIR'

QueryValidateResponse:
  type: object
  required: [valid]
  properties:
    valid:
      type: boolean
    normalized_query:
      $ref: '#/components/schemas/QueryIR'
    warnings:
      type: array
      items: { type: string }
    field_capabilities:
      type: array
      items:
        $ref: '#/components/schemas/FieldCapabilityBrief'

QueryRunRequest:
  type: object
  required: [query]
  properties:
    query:
      $ref: '#/components/schemas/QueryIR'

QueryRunResponse:
  type: object
  required: [columns, items, total, page]
  properties:
    columns:
      type: array
      items:
        $ref: '#/components/schemas/ResultColumn'
    items:
      type: array
      items:
        type: object
        additionalProperties: true
    total:
      type: integer
      format: int64
    facets:
      type: object
      additionalProperties:
        type: array
        items:
          $ref: '#/components/schemas/FacetBucket'
    page:
      $ref: '#/components/schemas/QueryPage'
    debug_plan:
      $ref: '#/components/schemas/DebugPlan'

ResultColumn:
  type: object
  required: [name, type, nullable]
  properties:
    name:
      type: string
    type:
      type: string
      enum: [string, enum, numeric, timestamp, boolean, json]
    nullable:
      type: boolean

FieldCapabilityBrief:
  type: object
  required: [field, engines]
  properties:
    field:
      type: string
    engines:
      type: array
      items:
        type: string

QuerySort (target):
  type: object
  required: [field]
  properties:
    field:
      type: string
    desc:
      type: boolean

QueryPage (target):
  type: object
  required: [offset, limit]
  properties:
    offset:
      type: integer
      minimum: 0
    limit:
      type: integer
      minimum: 1

QueryScope:
  type: object
  required: [resource]
  properties:
    resource:
      type: string
      enum: [assets]

QuerySelect:
  type: object
  properties:
    fields:
      type: array
      items:
        type: string

QueryFacet:
  type: object
  required: [field]
  properties:
    field:
      type: string
    size:
      type: integer
      minimum: 1

QueryDebug:
  type: object
  properties:
    explain:
      type: boolean

QueryPredicate:
  type: object
  required: [field, op]
  properties:
    field:
      type: string
    op:
      type: string
      enum: [eq, ne, gt, gte, lt, lte, in, between, like, ilike, exists]
    value:
      nullable: true

FacetBucket:
  type: object
  required: [value, count]
  properties:
    value:
      type: string
    count:
      type: integer
      format: int64

DebugPlan:
  type: object
  required: [steps]
  properties:
    steps:
      type: array
      items:
        $ref: '#/components/schemas/DebugPlanStep'

DebugPlanStep:
  type: object
  required: [engine, mode]
  properties:
    engine:
      type: string
      enum: [postgres, elasticsearch, qdrant, trino]
    mode:
      type: string
      enum: [filter, recall, sort, facet, aggregate, post_filter]
```

请求示例：

```json
{
  "query": {
    "schema_version": 1,
    "scope": { "resource": "assets" },
    "where": {
      "and": [
        { "pred": { "field": "owner", "op": "eq", "value": "team_alpha" } },
        { "pred": { "field": "tag.priority", "op": "in", "value": ["high", "critical"] } }
      ]
    },
    "sort": [{ "field": "created_at", "desc": true }],
    "page": { "offset": 0, "limit": 50 }
  }
}
```

### 7.6 错误语义分层（必须统一）

`queries/validate` 与 `queries/run` 建议统一错误层级：

| 场景 | HTTP | code | 说明 |
|---|---:|---|---|
| JSON 结构非法 / 缺字段 | 400 | `INVALID_ARGUMENT` | 请求体不满足基础 schema |
| 操作符或节点不支持 | 400 | `INVALID_QUERY_SYNTAX` | 协议层错误 |
| 字段不存在 / 资源不支持 | 422 | `UNSUPPORTED_FIELD` | 语义层错误 |
| 字段-操作符组合不支持 | 422 | `UNSUPPORTED_OPERATOR` | 能力层错误 |
| Planner 无法生成可执行计划 | 422 | `UNPLANNABLE_QUERY` | 规划层错误 |
| 依赖引擎不可用（ES/PG） | 503 | `DEPENDENCY_UNAVAILABLE` | 运行时依赖错误 |
| 执行超时 | 504 | `QUERY_TIMEOUT` | 运行超时 |
| 内部异常 | 500 | `INTERNAL_ERROR` | 未分类服务端异常 |

错误返回保持现有统一 envelope：

```json
{
  "code": "UNSUPPORTED_FIELD",
  "message": "field 'foo.bar' is not registered for resource 'assets'",
  "request_id": "..."
}
```

---

## 8. Saved Query / Slice / Export / Scenario Test

### 8.1 Saved Query

保存：

- `name`
- `query_ir_json`
- `schema_version`
- `owner`
- `description`

### 8.2 Slice

`slice` 本质上是：

- 命名 query
- 一次物化结果
- 或两者结合

推荐结构：

- `slice_id`
- `query_ir_json`
- `materialized_asset_ids`
- `row_count`
- `status`

### 8.3 Export

`export` 不接受 ad-hoc 手工拼条件，而是引用：

- `query_id`
- `slice_id`
- 或直接提交 `Query IR`

### 8.4 Scenario Test

`scenario test` 引用：

- `query_ir_json` 或 `slice_id`
- 再绑定规则、阈值、baseline

这就是为什么 Query IR 必须独立存在。

---

## 9. 前端状态模型与页面形态

前端不应继续围绕零散的 `activeFilters[] + queryText + sort + page` 组织状态，而应升级为 **Query Workbench State**。

### 9.1 顶层状态域

建议前端至少拆成 6 个状态域：

1. `QueryIRState`
2. `SavedQueryState`
3. `SliceState`
4. `QueryExecutionState`
5. `SelectionState`
6. `InspectorState`

其中前 3 个是对象模型核心，后 3 个是工作台运行时状态。

### 9.2 QueryIRState

```ts
type QueryExpr =
  | { kind: "and"; children: QueryExpr[] }
  | { kind: "or"; children: QueryExpr[] }
  | { kind: "not"; child: QueryExpr }
  | {
      kind: "pred";
      field: string;
      op: "eq" | "ne" | "gt" | "gte" | "lt" | "lte" | "in" | "between" | "like" | "ilike" | "exists";
      value?: unknown;
    };

type QueryIRDraft = {
  schemaVersion: 1;
  scope: { resource: "assets" };
  where?: QueryExpr;
  sort?: Array<{ field: string; desc?: boolean }>;
  page?: { offset: number; limit: number };
  facets?: Array<{ field: string; size?: number }>;
  select?: { fields?: string[] };
  debug?: { explain?: boolean };
};

type QueryWorkbenchUiState = {
  searchText?: string; // UI-only, 不属于 Query IR 协议
  viewMode: "table" | "grid"; // UI-only
};

type QueryIRState = {
  draft: QueryIRDraft;
  validated?: {
    normalizedQuery: QueryIRDraft;
    warnings: string[];
    fieldCapabilities: Array<{ field: string; engines: string[] }>;
    validatedAt: string;
  };
  dirty: boolean;
  lastRunQueryHash?: string;
};
```

### 9.3 SavedQueryState

```ts
type SavedQueryItem = {
  id: string;
  name: string;
  description?: string;
  query: QueryIRDraft;
  owner?: string;
  createdAt: string;
  updatedAt: string;
  pinned?: boolean;
};

type SavedQueryState = {
  items: SavedQueryItem[];
  activeId?: string;
  status: "idle" | "loading" | "saving" | "error";
  saveDialogOpen: boolean;
  renameDialogOpen: boolean;
  error?: string;
};
```

### 9.4 SliceState

```ts
type SliceItem = {
  id: string;
  name: string;
  sourceQueryId?: string;
  sourceQuery: QueryIRDraft;
  mode: "logical" | "materialized";
  rowCount?: number;
  materializedAssetIds?: string[];
  status: "draft" | "building" | "ready" | "failed";
  createdAt: string;
  updatedAt: string;
};

type SliceState = {
  items: SliceItem[];
  activeId?: string;
  createDialogOpen: boolean;
  materializeDialogOpen: boolean;
  status: "idle" | "loading" | "creating" | "materializing" | "error";
  error?: string;
};
```

### 9.5 QueryExecutionState

```ts
type QueryExecutionState = {
  status: "idle" | "validating" | "running" | "success" | "error";
  result?: {
    items: unknown[];
    total: number;
    facets?: Record<string, Array<{ value: string; count: number }>>;
    debugPlan?: unknown;
  };
  error?: string;
};
```

### 9.6 其余运行时状态

```ts
type SelectionState = {
  selectedIds: Set<string>;
  activeAssetId?: string;
};

type InspectorState = {
  mode: "preview" | "compare";
  compareAssetIds: string[];
  drawerOpen: boolean;
};
```

### 9.7 为什么要这样拆

- `QueryIRState`：定义“当前想查什么”
- `SavedQueryState`：定义“保存下来的 view”
- `SliceState`：定义“可复用 / 可物化子集”
- `QueryExecutionState`：定义“这次跑出来什么”

这四者职责不同，不能混在一个旧式 `queryState` 里。

### 9.8 TypeScript 类型文件草案

建议新增：

```text
Frontend/src/lib/query/
  queryIrTypes.ts
  queryWorkbenchTypes.ts
  queryWorkbenchActions.ts
  queryWorkbenchReducer.ts
```

`queryIrTypes.ts`：

```ts
export type QueryOperator =
  | "eq"
  | "ne"
  | "gt"
  | "gte"
  | "lt"
  | "lte"
  | "in"
  | "between"
  | "like"
  | "ilike"
  | "exists";

export type QueryExpr =
  | { kind: "and"; children: QueryExpr[] }
  | { kind: "or"; children: QueryExpr[] }
  | { kind: "not"; child: QueryExpr }
  | { kind: "pred"; field: string; op: QueryOperator; value?: unknown };

export type QueryIRDraft = {
  schemaVersion: 1;
  scope: { resource: "assets" };
  select?: { fields?: string[] };
  where?: QueryExpr;
  sort?: Array<{ field: string; desc?: boolean }>;
  page?: { offset: number; limit: number };
  facets?: Array<{ field: string; size?: number }>;
  debug?: { explain?: boolean };
};
```

`queryWorkbenchTypes.ts`：

```ts
import type { QueryIRDraft } from "./queryIrTypes";

export type QueryIRState = {
  draft: QueryIRDraft;
  validated?: {
    normalizedQuery: QueryIRDraft;
    warnings: string[];
    fieldCapabilities: Array<{ field: string; engines: string[] }>;
    validatedAt: string;
  };
  dirty: boolean;
  lastRunQueryHash?: string;
};

export type SavedQueryItem = {
  id: string;
  name: string;
  description?: string;
  query: QueryIRDraft;
  owner?: string;
  createdAt: string;
  updatedAt: string;
  pinned?: boolean;
};

export type SliceItem = {
  id: string;
  name: string;
  sourceQueryId?: string;
  sourceQuery: QueryIRDraft;
  mode: "logical" | "materialized";
  rowCount?: number;
  materializedAssetIds?: string[];
  status: "draft" | "building" | "ready" | "failed";
  createdAt: string;
  updatedAt: string;
};
```

---

## 10. Frontend Reducer Actions

前端 reducer 不应再围绕 `ADD_FILTER_CHIP / REMOVE_FILTER_CHIP` 组织，而应围绕 Query Workbench 的对象状态组织。

### 10.1 QueryIR actions

- `QUERY_SET_DRAFT`
- `QUERY_PATCH_WHERE`
- `QUERY_SET_SORT`
- `QUERY_SET_PAGE`
- `QUERY_VALIDATE_START`
- `QUERY_VALIDATE_SUCCESS`
- `QUERY_VALIDATE_ERROR`
- `QUERY_RUN_START`
- `QUERY_RUN_SUCCESS`
- `QUERY_RUN_ERROR`
- `QUERY_RESET_DIRTY`

### 10.2 SavedQuery actions

- `SAVED_QUERY_LIST_START`
- `SAVED_QUERY_LIST_SUCCESS`
- `SAVED_QUERY_LIST_ERROR`
- `SAVED_QUERY_SELECT`
- `SAVED_QUERY_SAVE_START`
- `SAVED_QUERY_SAVE_SUCCESS`
- `SAVED_QUERY_SAVE_ERROR`
- `SAVED_QUERY_DELETE_SUCCESS`
- `SAVED_QUERY_RENAME_SUCCESS`
- `SAVED_QUERY_DIALOG_TOGGLE`

### 10.3 Slice actions

- `SLICE_LIST_START`
- `SLICE_LIST_SUCCESS`
- `SLICE_LIST_ERROR`
- `SLICE_SELECT`
- `SLICE_CREATE_START`
- `SLICE_CREATE_SUCCESS`
- `SLICE_CREATE_ERROR`
- `SLICE_MATERIALIZE_START`
- `SLICE_MATERIALIZE_SUCCESS`
- `SLICE_MATERIALIZE_ERROR`
- `SLICE_DIALOG_TOGGLE`

### 10.4 Workspace actions

- `RESULT_SELECTION_SET`
- `RESULT_SELECTION_TOGGLE`
- `RESULT_SELECTION_CLEAR`
- `WORKBENCH_UI_SET_SEARCH_TEXT`
- `WORKBENCH_UI_SET_VIEW_MODE`
- `INSPECTOR_SET_ACTIVE_ASSET`
- `INSPECTOR_SET_MODE`
- `INSPECTOR_TOGGLE_DRAWER`

### 10.5 上线前收敛原则

现有前端中的：

- `activeFilters`
- `queryText`（映射到 `QueryWorkbenchUiState.searchText`）
- `sort`
- `page`
- `viewMode`（映射到 `QueryWorkbenchUiState.viewMode`）

其中仅协议字段归并进 `QueryIRState.draft`；UI 状态留在 `QueryWorkbenchUiState`。

### 10.6 TypeScript action union 草案

`queryWorkbenchActions.ts`：

```ts
import type { QueryIRDraft, QueryExpr } from "./queryIrTypes";
import type { SavedQueryItem, SliceItem } from "./queryWorkbenchTypes";

export type QueryWorkbenchAction =
  | { type: "QUERY_SET_DRAFT"; payload: { draft: QueryIRDraft } }
  | { type: "QUERY_PATCH_WHERE"; payload: { where?: QueryExpr } }
  | { type: "QUERY_SET_SORT"; payload: { sort: QueryIRDraft["sort"] } }
  | { type: "QUERY_SET_PAGE"; payload: { page: QueryIRDraft["page"] } }
  | { type: "WORKBENCH_UI_SET_SEARCH_TEXT"; payload: { searchText: string } }
  | { type: "WORKBENCH_UI_SET_VIEW_MODE"; payload: { viewMode: "table" | "grid" } }
  | { type: "QUERY_VALIDATE_START" }
  | { type: "QUERY_VALIDATE_SUCCESS"; payload: { normalizedQuery: QueryIRDraft; warnings: string[]; fieldCapabilities: Array<{ field: string; engines: string[] }> } }
  | { type: "QUERY_VALIDATE_ERROR"; payload: { error: string } }
  | { type: "QUERY_RUN_START" }
  | { type: "QUERY_RUN_SUCCESS"; payload: { items: unknown[]; total: number; facets?: Record<string, Array<{ value: string; count: number }>>; debugPlan?: unknown } }
  | { type: "QUERY_RUN_ERROR"; payload: { error: string } }
  | { type: "SAVED_QUERY_LIST_SUCCESS"; payload: { items: SavedQueryItem[] } }
  | { type: "SAVED_QUERY_SELECT"; payload: { id: string } }
  | { type: "SAVED_QUERY_SAVE_SUCCESS"; payload: { item: SavedQueryItem } }
  | { type: "SLICE_LIST_SUCCESS"; payload: { items: SliceItem[] } }
  | { type: "SLICE_SELECT"; payload: { id: string } }
  | { type: "SLICE_CREATE_SUCCESS"; payload: { item: SliceItem } };
```

### 10.7 Reducer 设计稿

`queryWorkbenchReducer.ts`：

```ts
export function queryWorkbenchReducer(
  state: QueryWorkbenchState,
  action: QueryWorkbenchAction
): QueryWorkbenchState {
  switch (action.type) {
    case "QUERY_SET_DRAFT":
      return {
        ...state,
        queryIR: {
          ...state.queryIR,
          draft: action.payload.draft,
          dirty: true,
        },
      };
    case "QUERY_VALIDATE_SUCCESS":
      return {
        ...state,
        queryIR: {
          ...state.queryIR,
          validated: {
            normalizedQuery: action.payload.normalizedQuery,
            warnings: action.payload.warnings,
            fieldCapabilities: action.payload.fieldCapabilities,
            validatedAt: new Date().toISOString(),
          },
        },
      };
    case "QUERY_RUN_SUCCESS":
      return {
        ...state,
        queryIR: {
          ...state.queryIR,
          dirty: false,
        },
        execution: {
          status: "success",
          result: {
            items: action.payload.items,
            total: action.payload.total,
            facets: action.payload.facets,
            debugPlan: action.payload.debugPlan,
          },
        },
      };
    default:
      return state;
  }
}
```

建议保留现有 `useAssetsDiscoveryReducer` 的页面骨架，但逐步替换为 `useQueryWorkbenchReducer`。

---

## 11. AssetsPage 演进形态

当前 `AssetsPage` 已经具备三栏 workbench 的雏形，应保留这个布局，只升级其语义。

### 11.1 目标页面结构

```text
AssetsPage
  QueryWorkbenchProvider
    TopBar
      QueryBar
      QueryActions
      SavedQuerySelector
      SliceSelector
    MainLayout
      LeftRail
        SavedQueriesPanel
        SlicesPanel
        FacetsPanel
      ResultsWorkspace
        ResultToolbar
        ActiveQuerySummary
        ResultsPane
      InspectorPane
        PreviewTab
        CompareTab
    DialogLayer
      SaveQueryDialog
      CreateSliceDialog
      MaterializeSliceDialog
      ExportDialog
```

### 11.2 模块职责

#### `QueryBar`

- 编辑当前 `QueryIRDraft`
- 触发 validate / run

#### `QueryActions`

- Save Query
- Create Slice
- Export

#### `SavedQueriesPanel`

- 浏览、选择、重命名、删除 saved query

#### `SlicesPanel`

- 浏览 logical / materialized slices
- 触发物化

#### `FacetsPanel`

- 根据结果 facets 反向编辑 `QueryIRDraft`

#### `ResultsWorkspace`

- 显示 `queries/run` 的 items / total / facets / debug plan

#### `InspectorPane`

- 展示 Preview
- 后续承载 Compare

### 11.3 对当前前端代码的影响

当前前端不需要推倒重来，但需要升级这些概念：

- `SavedViewSelector` -> `SavedQuerySelector`
- URL sync 从零散 `queryState` -> 序列化后的 `Query IR`
- `useAssetsDiscoveryReducer` -> `useQueryWorkbenchReducer`
- `AssetsFacetSidebar` 从“拼 filter” -> “编辑 query draft”

---

## 12. 前后端一一对齐关系

前端和后端必须共享同一套查询语义，不允许两边各自约定。

### 12.1 Query IR 对齐

| 前端状态 | 后端协议 |
|----------|----------|
| `QueryIRState.draft` | `POST /api/v1/queries/validate` / `run` 的 `query` |
| `validated.normalizedQuery` | validate 返回的 `normalized_query` |
| `QueryExecutionState.result` | run 返回的 `items / total / facets / debug_plan` |

### 12.2 Saved Query 对齐

| 前端状态 | 后端模型 |
|----------|----------|
| `SavedQueryItem.query` | `query_ir_json` |
| `SavedQueryItem.name` | `name` |
| `SavedQueryItem.owner` | `owner` |
| `SavedQueryItem.updatedAt` | `updated_at` |

### 12.3 Slice 对齐

| 前端状态 | 后端模型 |
|----------|----------|
| `SliceItem.sourceQuery` | `query_ir_json` |
| `SliceItem.mode` | `logical` / `materialized` |
| `SliceItem.materializedAssetIds` | 物化结果 |
| `SliceItem.rowCount` | `row_count` |
| `SliceItem.status` | `draft / building / ready / failed` |

### 12.4 对齐规则

1. 前端不得自行发明额外查询语义
2. 前端本地状态中的 `QueryExpr` 节点类型与后端 `Query IR` 节点类型一一对应
3. `Saved Query` 与 `Slice` 都只保存 / 引用 `Query IR`，不保存任何 SQL 或 ES DSL
4. URL 同步使用 `Query IR` 的压缩/编码结果，而不是零散 filter 参数

### 12.5 字段命名对齐策略

建议统一约定：

- 前端内部状态：`camelCase`
- API / Go / JSON 协议：`snake_case`

例如：

| 前端 | API / Go |
|------|----------|
| `schemaVersion` | `schema_version` |
| `normalizedQuery` | `normalized_query` |
| `fieldCapabilities` | `field_capabilities` |
| `debugPlan` | `debug_plan` |

命名转换应放在 API client 层完成，不应混入 reducer 与页面组件。

---

## 13. 是否有现成框架可以直接替代

结论：**没有成熟框架能直接替代 Query IR + Planner + Compiler 这一整层。**

### 13.1 可复用组件

#### `google/cel-go`

用途：

- 未来做文本 DSL
- `text expression -> AST -> Query IR`

优点：

- 安全
- 有类型检查
- 社区成熟

限制：

- 不负责跨引擎 planning
- 不适合作为内部长期协议

参考：<https://github.com/google/cel-go>

#### `alecthomas/participle`

用途：

- 如果坚持自定义一门 DSL，可以用它 parse 成 AST

限制：

- 只解决解析，不解决 planner / compiler

参考：<https://github.com/alecthomas/participle>

#### `goqu` / `squirrel`

用途：

- 辅助构建 SQL

限制：

- 是 SQL builder，不是统一查询层

参考：

- <https://github.com/doug-martin/goqu>
- <https://github.com/Masterminds/squirrel>

### 13.2 不建议作为核心方案

#### OData / RSQL / JsonLogic

这些都可以作为“外部语法”参考，但都不适合作为本项目的内部统一协议：

- OData 语法适合公开标准 API，但对你当前资产模型并不自然
- RSQL 比较偏 REST 过滤语法，不解决多引擎规划
- JsonLogic 适合规则求值，不适合作为数据库 pushdown 的核心 IR

#### OmniQL / Calcite

- `OmniQL` 路线接近，但生态和成熟度不足，不适合作为当前核心依赖
- `Apache Calcite` 很强，但它是 Java 生态，接入你当前 Go 后端成本过高

### 13.3 最终建议

| 层 | 方案 |
|----|------|
| 外部协议 | `JSON Query IR` |
| 文本 DSL（未来） | `cel-go` |
| Planner | 自研 |
| Compiler / Executor | 自研 |
| SQL builder | 先手写参数化 SQL；必要时局部引入 `goqu` |

---

## 14. 与当前 filter 的关系

当前 `backend/internal/filter` 方案：

- 绑定 `field:op:value`
- 默认多条件 `AND`
- 主要目标是生成 SQL

如果项目尚未正式上线，建议：

1. 不继续扩展现有 filter
2. 新建 `queryir / queryplan / queryexec`
3. 新页面、新 SDK、新 CLI 全部走新 Query API
4. 旧 filter 可以直接删除，不做对外双轨兼容

本项目当前前提是“尚未正式上线”，因此旧 filter 可直接下线，不额外引入对外兼容门槛。
如需做质量对拍，仅作为上线前内部验证手段，不构成长期双轨承诺。

当前过渡现实（必须承认）：

1. `queries/*` 的 PG refine 仍复用旧 filter 编译；
2. 前端主查询路径已切到 `queries/validate|run`，旧查询接口只剩兼容/包装层；
3. 旧 filter 当前仍是“运行时依赖”，不是“仅测试工件”。

因此本章的删除目标应理解为**阶段终态**，不是当前事实陈述。

---

## 15. 分阶段落地

### 15.0 任务拆分（可直接排期）

建议每个 phase 都拆成 6 类任务：`协议` / `后端` / `前端` / `测试` / `文档` / `验收`。

### 15.0.1 Task ID 实施状态（快照）

| Task ID | 状态 | 备注 |
|---|---|---|
| QP-001 | 半实现 | OpenAPI 已有，但与目标协议仍有分叉 |
| QP-002 | 半实现 | `queryir` 与基础校验已存在，normalize 能力仍薄 |
| QP-003 | 半实现 | 已有最小 `queryplan` 雏形，尚无真实多引擎规划 |
| QP-004 | 半实现 | 已有最小 `queryexec/postgres`，仍复用旧 filter 体系 |
| QP-005 | 已实现 | `queries/validate`、`queries/run` 已可调用 |
| QP-006 | 未实现 | 缺少系统性对拍矩阵与报告 |
| QP-007 | 半实现 | 已有最小 ES executor，专项测试与能力覆盖仍不足 |
| QP-008 | 半实现 | 已进入 ES recall + PG refine 主链，但规则/降级仍可完善 |
| QP-009 | 半实现 | `saved_queries` 服务端 CRUD、工作台接入与独立管理页已落地 |
| QP-010 | 半实现 | 工作台已接入 validate/run；structured/keyword 主路径已统一，semantic/similar 仍通过兼容包装映射到主查询能力 |

| Task ID | Phase | 类别 | 任务 | 输出物 |
|---|---|---|---|---|
| QP-001 | 1 | 协议 | 固化 Query IR v1 schema（JSON/OpenAPI） | `api/openapi.yaml` + 示例请求 |
| QP-002 | 1 | 后端 | 实现 `queryir` types/validate/normalize | `backend/internal/queryir/*` |
| QP-003 | 1 | 后端 | 实现规则 planner（PG-only） | `backend/internal/queryplan/planner.go` |
| QP-004 | 1 | 后端 | 实现 PG compiler/executor | `backend/internal/queryexec/postgres/*` |
| QP-005 | 1 | API | 落地 `queries/validate/run` | `handlers/query/*` + 路由注册 |
| QP-006 | 1 | 测试 | 20 组查询对拍（仅上线前内部基线，不对外兼容） | 回归测试报告 |
| QP-007 | 2 | 后端 | 实现 ES compiler/executor | `backend/internal/queryexec/elasticsearch/*` |
| QP-008 | 2 | 后端 | planner 支持 `ES recall + PG refine` | explain 中出现双引擎 step |
| QP-009 | 2 | API | `saved_queries` CRUD | `handlers/query/saved_queries.go` |
| QP-010 | 2 | 前端 | Query Workbench 接入 validate/run | `Frontend/src/lib/query/*` |
| QP-011 | 3 | API | `slices` + materialize | `handlers/query/slices.go` |
| QP-012 | 3 | API | `exports` / `scenario_tests` 基础接口 | 对应 handler + usecase |
| QP-013 | 3 | 前端 | Saved Query / Slice / Export 页面联动 | `AssetsPage` 工作台改造 |
| QP-014 | 4 | 协议 | 文本 DSL -> Query IR（CEL） | parser + 转换器 |
| QP-015 | 4 | 后端 | Qdrant executor + 多模态节点预留 | `queryexec/qdrant/*` |

执行建议：

1. 每个 Task ID 独立 PR，避免一次性大改。
2. 每个 PR 必须带“验证步骤 + 回归结果”。
3. 每个 phase 完成后做一次 freeze（只修 bug，不扩 scope）。

### Phase 1

- 定义 `Query IR v1`
- 定义字段注册中心
- 实现 `queries/validate`
- 实现 `queries/run`
- 实现 `PostgreSQL` compiler + executor

**DoD（Phase 1）**

- [x] `resource=assets` 的 v1 查询可用（PG 主路径）
- [x] `queries/validate/run` OpenAPI 契约落地
- [ ] P95 延迟（仅 PG） <= 300ms（limit=50，典型过滤）
- [ ] 至少 20 组查询对拍通过（新旧路径结果一致）
- [ ] `debug_plan` 输出可读且字段路由可解释
- [ ] Planner / Compiler / Handler 单测覆盖主分支

**Phase 1 子任务（建议顺序）**

- [x] P1-1：新增 `queryir` 包（types + validate + normalize）
- [ ] P1-2：新增字段注册中心配置与加载器
- [ ] P1-3：新增 `queryplan` 规则 planner（PG-only）
- [ ] P1-4：新增 PG compiler/executor
- [x] P1-5：落地 `POST /api/v1/queries/validate`
- [x] P1-6：落地 `POST /api/v1/queries/run`
- [x] P1-7：补 OpenAPI 与 `docs/review/api-guide.md`
- [ ] P1-8：补 20 组查询对拍与性能基线

### Phase 2

- 增加 `Elasticsearch` compiler + executor
- planner 支持 `ES 召回 + PG 精过滤`
- 增加 `saved_queries`

**DoD（Phase 2）**

- [ ] ES 可用场景默认走 `ES recall + PG refine`
- [ ] ES 不可用时自动降级 PG（并返回 warning）
- [ ] P95 延迟（ES+PG 混合） <= 500ms（limit=50）
- [ ] `saved_queries` 能持久化 `query_ir_json`
- [ ] 至少 30 组混合查询对拍通过

**Phase 2 子任务（建议顺序）**

- [ ] P2-1：新增 ES compiler/executor
- [ ] P2-2：planner 增加 `ES recall + PG refine` 路由
- [ ] P2-3：`saved_queries` 表结构与 CRUD API
- [ ] P2-4：前端 query workbench 接入 validate/run
- [ ] P2-5：前端 saved query 列表/选择/保存
- [ ] P2-6：补 30 组混合查询对拍与降级测试

### Phase 3

- 增加 `slices`
- 增加 `exports`
- 增加 `scenario_tests`

**DoD（Phase 3）**

- [ ] `slice` 支持 logical + materialized 两种模式
- [ ] `export` 支持引用 query/slice，不接受 ad-hoc 私有语法
- [ ] `scenario_tests` 可稳定复跑并产出可比较报告
- [ ] 关键链路（slice/export/test）具备失败重试与可观测性

**Phase 3 子任务（建议顺序）**

- [ ] P3-1：`slices` API（create/get/list）
- [ ] P3-2：`slices/:id:materialize` 任务化执行
- [ ] P3-3：`exports` API（引用 query/slice）
- [ ] P3-4：`scenario_tests` API（run/list/result）
- [ ] P3-5：前端 slices/export/test-run 工作流联动
- [ ] P3-6：补可观测性（失败重试、耗时、错误分层）

### Phase 4

- 文本 DSL（`CEL -> Query IR`）
- `Qdrant` executor
- 多模态查询节点

**DoD（Phase 4）**

- [ ] DSL 仅作为语法糖入口，内部统一编译到 Query IR
- [ ] Qdrant 接入不影响既有 Query IR 协议
- [ ] 多模态节点在 planner 中可 explain、可回退
- [ ] 安全与配额策略（限流/超时/复杂度）明确

**Phase 4 子任务（建议顺序）**

- [ ] P4-1：CEL 文本 DSL -> Query IR 转换
- [ ] P4-2：Qdrant compiler/executor
- [ ] P4-3：多模态节点（vector_search/fusion/rerank）协议扩展
- [ ] P4-4：planner explain 扩展（多引擎融合路径）
- [ ] P4-5：复杂度限制、限流、超时策略落地

---

## 16. 最终裁决

当前项目的长期正确方向是：

1. **弃用现有轻量 filter**
2. **引入 JSON Query IR 作为唯一查询协议**
3. **自研 Planner 与 Executors**
4. **只复用 parser / builder 这类局部组件，不指望存在现成统一查询框架**

这条路线的价值不在于“抽象更漂亮”，而在于它能稳定承载：

- `saved query`
- `slice`
- `export`
- `scenario test`
- `PG / ES / Qdrant / Trino` 多引擎执行

而不让查询语义散落到 handler、repo、前端、SDK 和脚本里。
