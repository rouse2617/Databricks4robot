# Use Cases 清单

> 平台 2.0 目标态需要支持的全部用户场景。按业务域分组，每条标角色 + API + 关键约束。
> 设计上下文与 6 个代表性场景叙述见 `data-platform-design.md` §5.8.2；架构 / 数据流 / 事件契约见同文档其它章节。

## 编号约定

- A 资产 / B Tag / C MCAP / D 算法 / E 交付 / F 检索 / G 事件审计 / H 数据集 / I 训练任务 / J Lakehouse / K SDK / L 前端组合页 / M 运维
- 同一编号下的子项按"用户动作"排，与开发任务可一一对应

## 全景视图

### 角色 × 业务域

```mermaid
flowchart LR
    subgraph Roles[角色]
        AlgoEng[算法工程师]
        AlgoWorker[算法 Worker<br>外部编排]
        BizOps[业务运营 / 数据 owner]
        TrainEng[训练工程师]
        FE[前端用户]
        SRE[SRE / Admin]
        Customer[客户系统]
    end

    subgraph Domains[业务域]
        A[A 资产]
        B[B Tag]
        C[C MCAP]
        D[D 算法]
        E[E 交付]
        F[F 检索]
        G[G 事件审计]
        H[H 数据集]
        I[I 训练]
        J[J Lakehouse]
        M[M 运维]
    end

    AlgoEng --> A & D & F & J
    AlgoWorker --> D & B & G
    BizOps --> A & B & E & G
    TrainEng --> H & I & J
    FE --> A & C & F
    SRE --> G & M
    Customer --> E
```

### 核心写 API → 事件 → 下游消费

平台的跨域协作通过 `asset_events` outbox 串起来——下图展示哪些写 API 产事件、哪个域消费：

```mermaid
flowchart LR
    subgraph Writes[核心写 API]
        A1w[A1 注册资产]
        A4w[A4 更新资产]
        A7w[A7 切换 lifecycle]
        B1w[B1 打 tag]
        B3w[B3 批量打 tag]
        C1w[C1 登记 MCAP]
        D2w[D2 提交算法结果]
        E1w[E1 创建交付]
        H4w[H4 触发快照]
    end

    Events[("asset_events<br>outbox")]

    subgraph Consumers[下游消费]
        ES[(Elasticsearch<br>F 检索)]
        Lake[(Iceberg<br>J Lakehouse)]
        Audit[(审计<br>G 审计)]
        Train[(训练 lineage<br>I 训练)]
    end

    A1w --> Events
    A4w --> Events
    A7w --> Events
    B1w --> Events
    B3w --> Events
    C1w --> Events
    D2w --> Events
    E1w --> Events
    H4w --> Events

    Events --> ES
    Events --> Lake
    Events --> Audit
    Lake -.-> Train
```

### Phase 上线节奏

```mermaid
flowchart LR
    subgraph P1[Phase 1 MVP<br>上线必备]
        P1A["A1/A3/A4/A5/A7/A10"]
        P1B["B1/B2/B4"]
        P1C["C1/C2/C4"]
        P1D["D1/D2/D3/D4/D7"]
        P1E["E1-E5"]
        P1G["G1/G2"]
        P1K["K1/K3/K5/K6"]
        P1L["L1/L2/L4/L8"]
        P1M["M1/M2"]
    end

    subgraph P15[Phase 1.5<br>上线后立刻补]
        P15A["A2/A6/A8/A9/A11"]
        P15BFGM["B3/B5/B6 · F1/F2/F4 · G3/G4 · M3/M5/M6/M7"]
        P15CDE["C3/C5/C6 · D5/D6/D8/D9 · E6/E7"]
    end

    subgraph P2[Phase 2<br>数据集 / 训练域上线]
        P2H["H1-H9"]
        P2I["I1-I5"]
        P2J["J1-J4"]
        P2KL["K2/K4 · L3/L5/L6/L7/L9 · M4"]
    end

    subgraph P3[Phase 3+<br>候选]
        P3F["F3 聚合 · F5 向量召回"]
    end

    P1 --> P15 --> P2 --> P3
```

---

## A. 资产（Asset）—— 平台一等公民

| # | 场景 | API | 角色 | 关键约束 |
|---|------|-----|------|---------|
| A1 | 注册新资产（数采链路完成后） | `POST /api/v1/assets` | 系统 / 数据管道 | 写入同事务追加 `asset_created` 事件 |
| A2 | 批量导入资产（历史 / 离线作业） | `POST /api/v1/assets:batch` | Admin / 数据工程 | 单次 ≤ 1000 条；超出走 chunk |
| A3 | 资产详情（基础 + tags + algo 状态合并） | `GET /api/v1/assets/{id}` | 全部 | PG 三表 fan-out；< 100 ms P99 |
| A4 | 部分更新资产（多字段 PATCH） | `PATCH /api/v1/assets/{id}` | 业务 / 算法 | OCC：带 `If-Match: version`，冲突 `409` |
| A5 | 列表筛选 + 分页 | `GET /api/v1/assets?...` | 全部 | 走 ES，PG fallback；`page_size ≤ 100` |
| A6 | 软删 / 归档资产 | `DELETE /api/v1/assets/{id}` 或 `POST /:archive` | Admin | 物理删除由后台 retention job 跑 |
| A7 | 生命周期切换（draft → ready → archived） | `PATCH /api/v1/assets/{id}/lifecycle` | 业务 | 仅允许状态机定义的转移 |
| A8 | 还原归档资产 | `POST /api/v1/assets/{id}:restore` | Admin | retention 期内有效 |
| A9 | 查资产关联的 MCAP | `GET /api/v1/assets/{id}/mcap` | 算法 / 业务 | 一资产可关联多 MCAP |
| A10 | 资产事件历史 / 时间线 | `GET /api/v1/assets/{id}/events` | 业务 / 审计 | 按 `event_seq` 排，支持游标分页 |
| A11 | 资产血缘（父子 / 派生） | `GET /api/v1/assets/{id}/lineage` | 训练 / 算法 | 走 `asset_relations` 表 |

---

## B. Tag 管理

| # | 场景 | API | 角色 | 关键约束 |
|---|------|-----|------|---------|
| B1 | 给资产打 tag | `POST /api/v1/assets/{id}/tags` | 业务 / 算法 | 校验 `tag_registry`：key + value 必须在白名单 |
| B2 | 删 tag | `DELETE /api/v1/assets/{id}/tags/{key}` | 业务 | 同事务追加 `tag_deleted` 事件 |
| B3 | 批量打 tag | `POST /api/v1/tags:bulk` | 业务 | 单次 ≤ 500 资产 |
| B4 | 查 tag 注册表（key / values 枚举） | `GET /api/v1/tags/registry` | 全部 | 来自 `tag_registry.yaml`，支持热重载 |
| B5 | 按 tag 反查资产（A5 的 filter 特例） | `GET /api/v1/assets?tags.scene=highway` | 全部 | 走 ES `nested` 查询 |
| B6 | tag 修改历史 | `GET /api/v1/assets/{id}/tags/history` | 审计 | 走 `asset_events` 过滤 type |

---

## C. MCAP 文件（原始数采）

| # | 场景 | API | 角色 | 关键约束 |
|---|------|-----|------|---------|
| C1 | 登记 MCAP 元数据（上传完成回调） | `POST /api/v1/mcap` | 数采链路 | `raw_hash_md5` UNIQUE，重复直接复用 |
| C2 | 查 MCAP 元数据（含 manifest） | `GET /api/v1/mcap/{id}` | 全部 | 含字节偏移索引 |
| C3 | 列表 MCAP | `GET /api/v1/mcap` | 业务 / Admin | 按时间 / size / source 过滤 |
| C4 | MCAP segment signed URL | `GET /api/v1/mcap/{id}/segment-url?start_ns=&end_ns=` | 前端 / SDK | 10 min 过期；浏览器直拉 GCS |
| C5 | MCAP 关联到资产 | `POST /api/v1/mcap/{id}/assets` | 数采 / Admin | 多对多关系 |
| C6 | 查 MCAP topic / message 统计 | `GET /api/v1/mcap/{id}/topics` | SDK / 算法 | 不下载文件 |

---

## D. 算法处理

| # | 场景 | API | 角色 | 关键约束 |
|---|------|-----|------|---------|
| D1 | 拉待处理资产（pending） | `GET /api/v1/algo/{name}/pending?since=&limit=` | 算法 worker | 按 `depends_on` 链动态算 pending；游标 since |
| D2 | 提交算法运行结果 | `POST /api/v1/algo/{name}/runs` | 算法 worker | 同事务写 `asset_algo_latest` + 追加 `algo_completed` |
| D3 | 查算法 run 详情 | `GET /api/v1/algo/runs/{run_id}` | 全部 | 含 metrics / artifact_uri |
| D4 | 资产的所有算法状态 | `GET /api/v1/assets/{id}/algo` | 业务 | 投影表读，毫秒级 |
| D5 | 列某算法的所有 run | `GET /api/v1/algo/{name}/runs?status=ok` | 算法 / Admin | 大量数据走 Trino 而非 PG |
| D6 | 重跑算法 / 回放 | `POST /api/v1/algo/{name}:replay` | Admin / 算法 | 按 asset_id 列表 / 时间区间 |
| D7 | 算法注册表查询 | `GET /api/v1/algo/registry` | 全部 | 来自 `algo_registry.yaml` |
| D8 | 算法依赖（depends_on） | `GET /api/v1/algo/{name}/dependencies` | 算法 worker | 用于 worker 自调度 |
| D9 | 标记算法状态 blocked / reset | `PATCH /api/v1/assets/{id}/algo/{name}` | Admin | 应急运维入口 |

---

## E. 交付（Delivery）

| # | 场景 | API | 角色 | 关键约束 |
|---|------|-----|------|---------|
| E1 | 创建交付 | `POST /api/v1/deliveries` | 业务 | **必须带 `Idempotency-Key`** |
| E2 | 查交付详情 | `GET /api/v1/deliveries/{id}` | 业务 | 含 SLA / 渠道 / 当前进度 |
| E3 | 列出交付 | `GET /api/v1/deliveries` | 业务 | 按 owner / status / 时间过滤 |
| E4 | 列交付项明细 | `GET /api/v1/deliveries/{id}/items` | 业务 | 大交付走分页 |
| E5 | 取消交付 | `POST /api/v1/deliveries/{id}:cancel` | 业务 | 仅 `pending / in_progress` 可取消 |
| E6 | 重试失败项 | `POST /api/v1/deliveries/{id}:retry` | 业务 | 只重试 failed 状态项 |
| E7 | 客户回执 | `POST /api/v1/deliveries/{id}/ack` | 客户系统 | 外部受信任入口（独立鉴权） |
| E8 | 资产被交付历史 | A3 详情已含 `last_delivered_to` 等 | 业务 | — |

---

## F. 检索 / 搜索（ES）

| # | 场景 | API | 角色 | 关键约束 |
|---|------|-----|------|---------|
| F1 | 全文检索（关键字） | `GET /api/v1/search?q=` | 业务 / 算法 | `multi_match` + 高亮 |
| F2 | 多维 filter + 分页 | `GET /api/v1/search?q=&filter=...` | 业务 | filter 走 ES `term`，无评分 |
| F3 | 聚合统计（按 tag / lifecycle / owner） | `GET /api/v1/search/agg?group_by=` | 看板 | ES `aggregation` |
| F4 | 自动补全 / suggest | `GET /api/v1/search/suggest?prefix=` | 前端搜索框 | ES `completion suggester` |
| F5 | 相似资产召回（向量，3.x 候选） | `GET /api/v1/search/similar?asset_id=` | 算法 | 不在 2.0 范围 |

---

## G. 事件 / 审计 / 历史

| # | 场景 | API | 角色 | 关键约束 |
|---|------|-----|------|---------|
| G1 | 资产事件流（与 A10 同） | `GET /api/v1/assets/{id}/events` | 业务 / 审计 | 按 `event_seq` 范围 |
| G2 | 全局事件流（按 seq 拉取） | `GET /api/v1/events?from_seq=` | Admin / 下游 | 长游标，天然支持 outbox 风格订阅 |
| G3 | 审计查询 | `GET /api/v1/audit?actor=&action=` | 合规 / Admin | 审计表 INSERT-only |
| G4 | 重放事件区间 | `POST /api/v1/events:replay` | Admin / SRE | 配合 sink watermark 重置 |

---

## H. 数据集 / Dataset（训练数据治理）

| # | 场景 | API | 角色 | 关键约束 |
|---|------|-----|------|---------|
| H1 | 创建数据集定义 | `POST /api/v1/datasets` | 训练工程师 | 含 query 语义 + owner / 描述 |
| H2 | 列数据集 | `GET /api/v1/datasets` | 训练 / 业务 | — |
| H3 | 数据集详情 | `GET /api/v1/datasets/{id}` | 训练 | — |
| H4 | 触发构建快照 | `POST /api/v1/datasets/{id}/snapshots` | 训练 | 异步，状态 building → sealed |
| H5 | 查快照详情 | `GET /api/v1/datasets/{id}/snapshots/{version}` | 训练 | 含 `manifest_uri / row_count` |
| H6 | 列快照 | `GET /api/v1/datasets/{id}/snapshots` | 训练 | — |
| H7 | 快照预览（采样） | `GET /api/v1/datasets/{id}/snapshots/{version}/preview?limit=` | 训练 | Trino → Iceberg time travel |
| H8 | 归档快照 | `POST /api/v1/datasets/{id}/snapshots/{version}:archive` | 训练 / Admin | **不可硬删，仅 archived** |
| H9 | 快照关联的训练任务 | `GET /api/v1/datasets/{id}/snapshots/{version}/training-runs` | 训练 / 审计 | 走 `training_runs` 反查 |

---

## I. 训练任务记录（Training Runs）

| # | 场景 | API | 角色 | 关键约束 |
|---|------|-----|------|---------|
| I1 | 注册训练任务 | `POST /api/v1/training-runs` | 训练系统 | 引用 `dataset_id + snapshot_id` |
| I2 | 上报状态 / metrics / artifact | `PATCH /api/v1/training-runs/{id}` | 训练系统 | 增量 patch |
| I3 | 查训练任务详情 | `GET /api/v1/training-runs/{id}` | 训练 / 业务 | — |
| I4 | 列训练任务 | `GET /api/v1/training-runs?model=&status=` | 训练 | — |
| I5 | 训练任务的数据快照（lineage） | I3 详情已含 | 训练 / 审计 | 自包含 catalog 引用 |

---

## J. Lakehouse / OLAP（Trino + Iceberg）

| # | 场景 | API | 角色 | 关键约束 |
|---|------|-----|------|---------|
| J1 | 受限 SQL 自助查询 | `POST /api/v1/lakehouse/query` | 业务 / 数据分析 | 黑名单某些函数；超时 30 s |
| J2 | 重算候选清单 | `GET /api/v1/lakehouse/recompute-candidates` | 算法 | 跑 Trino SQL，结果缓存 |
| J3 | 跨字段聚合 | `GET /api/v1/lakehouse/agg?...` | 看板 | 走 Iceberg gold 层 |
| J4 | 表 schema 查询 | `GET /api/v1/lakehouse/tables/{name}/schema` | SDK / 训练 | 走 Polaris catalog |

---

## K. Python SDK（grace_sdk）

| # | 场景 | SDK API | 内部走 | 说明 |
|---|------|--------|------|------|
| K1 | 拿单资产 | `asset.get(id)` | A3 | 一行返回 dataclass |
| K2 | 流式读 MCAP topic | `asset.stream(topic, asset_id)` | C4 + GCS Range | 不下整文件 |
| K3 | 列资产（生成器，自动分页） | `asset.list(filter=, iter=True)` | A5 | 自动 cursor 翻页 |
| K4 | 流式拉训练数据 | `dataset.iter(snapshot_id=)` | PyIceberg 直读 | 不经 backend |
| K5 | 给资产打 tag | `asset.tag(id, key, value)` | B1 | — |
| K6 | 提交算法结果 | `algo.complete(asset_id, name, result)` | D2 | Worker 友好包装 |

---

## L. 前端组合页面（多 API 拼接）

| # | 页面 | 拼接的 API |
|---|------|----------|
| L1 | 工作台首页（资产 + run + 交付 + 趋势） | A5 + D5 + E3 + J3 |
| L2 | 资产详情页（基础 + tags + algo + events + lineage + MCAP） | A3 + A10 + A11 + A9 |
| L3 | 算法监控页 | D5 + J3 |
| L4 | 交付管理页 | E1–E6 |
| L5 | 数据集管理页 | H1–H9 + I3 |
| L6 | 训练任务监控页 | I3 + I4 + H9 |
| L7 | 搜索页（全文 + facet + 高亮） | F1 + F2 + F3 |
| L8 | MCAP 在线预览（嵌入 Webviz / Foxglove） | C4 + C2 |
| L9 | 审计追踪页 | G1 + G3 |

---

## M. 运维 / Admin / 系统层

| # | 场景 | API | 角色 |
|---|------|-----|------|
| M1 | 健康检查 | `GET /healthz` `/readyz` | K8s |
| M2 | Prometheus 指标 | `GET /metrics` | GMP |
| M3 | 强制重建 ES 索引 | `POST /admin/search/reindex?from_seq=` | SRE |
| M4 | 触发 PyIceberg MERGE（手动） | `POST /admin/lakehouse/merge` | SRE |
| M5 | 重置 outbox watermark / DLQ 处理 | `POST /admin/outbox/{sink}/watermark` | SRE |
| M6 | 查 outbox queue 状态 | `GET /admin/outbox/status` | SRE |
| M7 | tag / algo 注册表热重载 | `POST /admin/registry:reload` | Admin |

---

## 角色 → 主要场景速查

| 角色 | 主要场景 |
|------|---------|
| 算法工程师 | A3 / A5 / D2 / D4 / D5 / F1 / J1 / K1–K6 / H7 |
| 算法 Worker（自动化） | D1 / D2 / D7 / D8 / B1 / G2 |
| 业务运营 / 数据 owner | A1 / A4 / A5 / B1 / B5 / E1–E6 / G1 |
| 训练工程师 | H1–H9 / I1–I5 / J1 / K4 |
| 客户系统 | E7（仅回执） |
| Admin / SRE | A6 / A8 / D6 / D9 / G4 / M1–M7 |
| 前端用户 | A3 / A5 / F1–F4 / L1–L9 |

---

## 开发优先级建议

> 仅作 MVP 范围参考，最终由产品 / 工程联合排期。

### Phase 1 MVP（必备，上线前）

A1 / A3 / A4 / A5 / A7 / A10
B1 / B2 / B4
C1 / C2 / C4
D1 / D2 / D3 / D4 / D7
E1 / E2 / E3 / E4 / E5
G1 / G2
K1 / K3 / K5 / K6
L1 / L2 / L4 / L8
M1 / M2

### Phase 1.5（上线后立刻补）

A2 / A6 / A8 / A9 / A11
B3 / B5 / B6
C3 / C5 / C6
D5 / D6 / D8 / D9
E6 / E7
F1 / F2 / F4
G3 / G4
M3 / M5 / M6 / M7

### Phase 2（数据集 / 训练域上线时）

H1–H9
I1–I5
J1 / J2 / J3 / J4
K2 / K4
L3 / L5 / L6 / L7 / L9
M4

### Phase 3+（候选）

F3（聚合）/ F5（向量召回）

---

## 维护规则

- 新增 API 端点必须先在本文件登记一行（编号 + 角色 + API + 关键约束）
- API 退役时不删行，把"关键约束"列改为 `~~已废弃 vYY.MM~~`，保留 1 个版本周期后归档到附录
- 编号一旦分配不复用；废弃编号留作历史档案
- 与 `data-platform-design.md` §5.8.2 的 6 个代表场景同步——叙述变更要回写到本文件
