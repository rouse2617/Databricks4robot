# PRD: 统一资产目录、血缘、版本、标签与可追溯（PG + ES + GCS）

| 字段 | 值 |
|------|---|
| 状态 | **Superseded by rev.12 SoT documents**（本 PRD 进入历史决策追溯模式）|
| 当前版本 | rev.11（2026-05-21 自洽 cleanup）|
| **rev.12 业务建模 SoT** | **[`../README.md`](../README.md)** — 业务实体优先 |
| **rev.12 schema SoT** | **[`../schema.md`](../schema.md)** — PG 物理 schema |
| 关联 Issue | CYB-983 |
| 关联文档 | `data-platform-design.md`、`schema-reference.md`、`eval-metrics-design.md`、roadmap P1-E2 |
| 参考架构 | DataHub **产品模式**（SystemMetadata / Actions Framework）；**不**引入 DataHub runtime / Kafka / Neo4j / PDL |
| 架构分层 | **L1** 学 DataHub 产品模式，落地 PG+ES+GCS+Go；**L2** 湖仓 catalog 联邦（Gravitino）不在本 PRD |

---

## 🔄 rev.12 重要变更通知（2026-05-21）

> **本 PRD 已被 rev.12 SoT 文档接管业务建模主线**。新人读者优先读：
>
> 1. **[`../README.md`](../README.md)** —— 4 类一等实体 / 7 个 asset_type / algo_runs / customers / 版本对应规则
> 2. **[`../schema.md`](../schema.md)** —— 13 张 P1 表完整 DDL / 索引 / 约束 / trigger
>
> **rev.12 核心翻转**（与本 PRD §3.8 / §12.4 等冲突时以 rev.12 为准）：
>
> - ✗ **砍** Entity-Aspect 4 张通用 Aspect 拆分（asset_lineage / asset_content / asset_governance / asset_usage_stats）
> - ✗ **砍** asset_full VIEW + HydrateScope + AssetWriter 拆分器
> - ✗ **砍** asset_algo_latest 的 algo_kind 多 kind 共表
> - ✗ **砍** databrew:// URI scheme（HTTP URL 替代）
> - ✗ **砍** G3 GCS 双 commit（pending → materialized 状态机替代）
> - ✗ **砍** A2 PATCH 自动升 B（rev.11 改 422）
> - ✓ **新增** algo_runs 一等实体（执行事件类，run-level 元数据）
> - ✓ **新增** customers 一等实体（业务参考类，B2B 主数据）
> - ✓ **新增** raw_mcap 进 assets + mcap_files 作专用 Aspect（与 actions 对称）
> - ✓ **新增** asset_relations 的 derived_from 边（算法产出专用）
> - ✓ **不变式精简** 38 → 13 条
> - ✓ **命名子系统精简** 24 → 6 个
> - ✓ **没上线红利** Migration 7 步骤 → 1 步骤（无 dual-write / DROP COLUMN 延后 / OrphanAssertionJob）
>
> 详细决策路径见 [`../README.md` §16 决策矩阵](../README.md#16-决策矩阵rev12-锁定可追溯)。

**本 PRD 后续章节（§1-§16）保留为历史决策追溯**，但**业务建模 / schema 实现细节以 rev.12 SoT 为准**。

---

---

## 1. Problem Statement

### 1.1 业务背景

> 平台定性、用户角色、数据形态详见 `data-platform-design.md` §1。

DataBrew 是面向机器人 / 自动驾驶采集数据的 **B2B 数据资产化管理 + 处理与交付平台**：

- **数据形态**：MCAP（Foxglove 容器，多模态时序），从采集到算法处理都围绕 MCAP
- **用户角色**：**内部算法用户**（生产 + 消费）+ **外部客户**（接收处理后产物）
- **核心商业模式**：把 MCAP 母带切分、加工、标注为「可独立检索的资产片段」（segment / clip / action / frame / task），按片段卖给训练 / 评估 / 回放客户
- **核心竞争力**：**精确检索能力** + **交付质量的确定性承诺**（端到端业务流见 §17 待补）

业务层级（简，权威定义见 §3.1）：

```text
raw_mcap → segment → clip / frame / task / action(L2) → action(L3)
```

资产会经多轮算法处理、多版本迭代并可能落地 GCS 产物（亦可保持 virtual 无产物），可通过 `delivery_items` 交付客户。

### 1.2 Why Now

| 触发 | 影响 |
|------|------|
| 业务进入「卖给真客户」阶段 | 合规审计 / 客户 SLA / 链接稳定性等承诺开始硬约束 |
| 算法即将进入 v2 升级潮 | hand_track / action_detector 大规模重产将造成「客户拿到错文件」事故 |
| 行锁 / 静默漂移已出现真实问题 | 多算法并发写同一资产已观察到性能下降；老 GCS 对象被覆盖事故发生过 |
| AI Agent 自动消费目录的需求出现 | Cursor / Claude 直接查 DataBrew，需要稳定的元数据契约 |

### 1.3 用户感知缺口（5 类，业务问题）

每条均为「用户能直接感知的痛」，解决后**用户体感立即改善**。

| # | 缺口 | 用户视角的痛 | 主要 Solution 对应 |
|---|------|--------------|------------------|
| **U1** | **检索能力不足** | 运营无法回答「曾跑过 hand_track v2.0」「含 grasp 动作」「按 task 包含」等核心提数问题 | §10 / §4.4 |
| **U2** | **版本迭代无显式建模** | 算法升级 / 人工返工后，客户拿到的 GCS 文件可能被静默替换；老快照不可回溯；商业纠纷防御薄弱 | §4 Revisions |
| **U3** | **追溯不完整** | 合规审计无法一秒回答「PII 标签流向了哪些客户的 delivery」 | §3.6 / §4 / §11.5 |
| **U4** | **标签体系不可扩展** | 新增标签类型（LLM / 供应商 / 众包 / 合规）要改 schema；多算法多版本结果无并存模型 | §7 Tag Source Registry |
| **U5** | **资产模型割裂** | 物理表（`mcap_files` / `actions`）与业务资产并存；`frame_set` 命名混淆；前端 / SDK / 客户视角不统一 | §3 一等资产 |

### 1.4 工程内部债务（3 类，平台问题）

「用户感知不到，但不解决会让 §1.3 持续退化」。是 §1.3 的**结构性根因**，不应被忽略。

| # | 债务 | 不解决的代价 | 主要 Solution 对应 |
|---|------|--------------|------------------|
| **E1** | **`assets` 主表过胖**（30+ 列，业务字段持续累加）| 多算法并发写产生行锁竞争；新字段要改 schema；B 路由复制冗余 | §3.8 Entity-Aspect 拆分 |
| **E2** | **层级规则未 enforce** | schema 允许任意父子关系，存在数据腐化风险；血缘断裂用户最终感知到（变成 U3） | §3.7 Validator 不变式 |
| **E3** | **副作用编排散落** | reindex / 通知 / 投影逻辑分散，无统一 retry / watermark / DLQ；新 projector 重复造轮子，出错率高 | §4.11 Actions Framework |

### 1.5 成功指标

| 指标 | 目标 | 衡量来源 |
|------|------|---------|
| 任意 `asset_id` 血缘可追到 raw_mcap | ≥ 99% | 抽样脚本 + `asset_relations` 覆盖率 |
| 客户文件「静默改变」投诉数（同 URL 内容变化） | **0**（不可妥协）| 客服工单 |
| 新增标签类型上线时长 | < 1 小时（仅改 registry yaml）| PR → 部署 |
| 算法 v2 重产场景下「客户链接失效」率 | **0%** | delivery_items 引用稳定性巡检 |
| 合规审计「PII 流向查询」响应时间 | < 5 秒（10000 资产级）| `/audit/lineage-search` P95 |
| ES facet 检索 P95 延迟 | < 200ms | queries/run 监控 |
| 多算法并发写同一资产的行锁等待 | ≈ 0 | PG `pg_locks` 监控 |

### 1.6 Non-Goals（本 PRD 不解决）

> 完整非目标见 `data-platform-design.md` §3.1；本 PRD 额外明确：

- **跨数据平台元数据联邦**（不是 DataHub；不接 Snowflake / BigQuery 镜像）
- **BI / 报表 / 数据可视化**
- **实时算法编排**（属 Tekton / Airflow 调度层，不在本 PRD）
- **合规判定**（接受 `compliance.*` tag 输入，平台不做合规决策）
- **多租户 / RBAC 深度隔离**（P2+ 单独 ADR；本期单业务单实例）
- **Glossary Terms / Domains / Compliance Forms**（DataHub 治理产物，业务尚不需要）
- **Kafka 集群**（事件源用 PG events，P2 可切已有 GCP Pub/Sub）

### 1.7 缺口 → Solution 主线对照表

| 缺口 | Solution 主线（§2）| 关键章节 |
|------|------------------|---------|
| U1 检索 | 主线 3 + 主线 5 | §4.4 / §10 / §4.11 SearchReindexer |
| U2 版本 | 主线 3 | §4 Revisions + §4.10 URI |
| U3 追溯 | 主线 2 + 主线 3 | §3.6 / §11.5 lineage-search |
| U4 标签 | 主线 4 | §7 + §7.6 Registry |
| U5 资产模型 | 主线 1 | §3 一等资产 + §3.8 Entity-Aspect |
| E1 主表过胖 | 主线 1 | §3.8 Entity-Aspect |
| E2 层级 | 主线 2 | §3.7 不变式 + Validator |
| E3 副作用编排 | 主线 5 | §4.11 Actions Framework |

---

## 2. Solution（五条主线）

> 主线编号对应 §1.7 缺口对照表。每条主线列出**核心机制 + 解决的缺口 + 详见章节**；
> 完整 schema 见 `schema-entity-aspect.md`，完整 API 见 §11。
> **共享 schema 提示**：同一张 PG 表可被多主线协作使用（例：`asset_relations` 用 typed enum 同时承载结构血缘和版本血缘）——这是 Entity-Aspect 模式的工程红利，不是冗余。

### 主线 1 — 统一实体（Entity-Aspect 模式）

**核心：** `assets` 瘦身为 **9 列 Entity 壳** + 3 张通用 Aspect 表（lineage 含时间维度 / content / governance）+ P1.5 高频信号独立表（`asset_usage_stats`）+ 现有专用 Aspect（mcap_files / actions / asset_tags / asset_algo_latest / metrics / eval_results）。多算法并发写不同 Aspect **零行锁竞争**；新字段 = 新 Aspect，不改主表。
**解决：** U5（资产模型割裂）/ E1（主表过胖）
**详见：** §3.8 + `schema-entity-aspect.md`

### 主线 2 — 结构血缘（横向）

**核心：** 固定 `raw_mcap → segment → {clip / frame / task / action(L2)}`，`task → action(L3)`；`asset_relations` typed 边（`split_from / contains / merged_from / derived_from / sampled_from / validated_by / revision_of`）；写入契约强制写关系边。
**解决：** U3（追溯不完整）/ E2（层级未 enforce）
**详见：** §3.1 / §3.6 / §3.7

### 主线 3 — 非结构性血缘（处理 + 版本）

两个独立子能力，物理上分布在不同表 / 字段，但都属于「非结构性血缘」：

#### 3.1 处理血缘（横向：算法在资产上跑过什么）
- `asset_algo_latest`（当前态 per algo_name，支持 `is_pinned`）
- `asset_events` 中 `algo_*` 事件类型（全历史）
- ES `algos_current[]` + `algo_versions_seen`（当前 + 历史 facet）

#### 3.2 版本血缘（纵向：同逻辑资产的 v1 → v2 → v3）
- `logical_asset_id + revision + is_current`（Entity 壳 3 列；版本原因 `revision_reason` 走 `asset_relations(revision_of).metadata` JSONB，不进 Entity 壳）
- `asset_relations` 中 `revision_of` 边 + `logical_assets` 一等表
- `asset_events` 中 `asset_revised` 事件
- GCS `assets/<logical>/r<n>-<uuid4>/` 不可覆盖路径
- `GET /api/v1/logical-assets/{logical_id}/current` HTTP URL 给客户/SDK 跟随当前版本（rev.11 决策：不引入自定义 URI scheme）

**解决：** U1（检索曾跑过某版本）/ U2（版本迭代无显式建模）
**详见：** §4 全章

### 主线 4 — 标签 + 资产级 facet + 交付资格

**核心：**
- `asset_tags` 单表 + `source/source_name/source_version` 三列 + **Tag Source Registry** 开放扩展（新增 source 类只改 yaml，零 schema 变更）
- `actions` 一版一行（K1，每算法版本独立 `action_id` + `revision_of`）
- `asset_metrics` 标量投影（含人工评分 `rating.*`）
- `asset_eval_results` 评估原档
- **Amundsen 风格发现层信号**（view / download / trending_score / favorited_count）由 P1.5 独立 Aspect `asset_usage_stats` 承载（与 `asset_governance` 拆开，避免高频写阻塞低频治理写）
- `delivery_rules` 一等实体（DSL + commit C2 校验）

**解决：** U1（按 label / action / metric 检索）/ U4（标签可扩展）
**详见：** §7 / §3.8.4 / §9.2 / §11.5

### 主线 5 — 全量审计 + Actions Framework

#### 5.1 审计（同事务，强一致）
`asset_events` 三段式 = **顶层 4 字段独立列**（`actor / request_id / idempotency_key / event_time`，高频索引）+ **`system_metadata` JSONB**（对齐 DataHub SystemMetadata：run_id / pipeline / algo / registry）+ **`payload` JSONB**（只装 `before / after / changed_fields` 业务 diff）。append-only + Idempotency-Key + 乐观锁三重幂等。

#### 5.2 Actions Framework（异步，事件源可插拔）
统一 `ActionHandler` 接口 + `ActionRunner` 共享 watermark / retry / DLQ。P1 落 5 个 Handler：
- `SearchReindexer`（ES 文档更新）
- `ParentReindexer`（D1 父子摘要重建）
- `RevisionNotifier`（v2 发布后通知运营/客户）
- `AlgoVersionsProjector`（维护 ES `algo_versions_seen`）
- `OpenLineageEmitter`（占位，P2 启用）

事件源可插拔：**P1/P1.5 走 PG events polling**，P2 可切已有 **GCP Pub/Sub**（不引入 Kafka 集群）。

**解决：** E3（副作用编排散落）
**详见：** §4.9 / §4.11

---

### 读模型 / 写模型 契约

#### 读模型路由（强约束 IR1，§10.3）

| 路径 | 走 | 一致性 |
|------|----|------|
| `GET /assets/{id}` / `GET /provenance` / `GET /events` / `GET /logical-assets/{id}/current` | **PG**（asset_full VIEW） | 强一致（无 ES 延迟感知）|
| `POST /queries/run` / `GET /delivery-rules/.../candidates` / `GET /audit/search` | **ES** | 最终一致（1–3s）|

#### 写模型边界

- **PG 同事务原子写**：Entity 壳 + 4 Aspect + 专用 Aspect（如 actions） + `asset_events` + 关系边（缺则 422）
- **异步走 Actions Framework**：GCS 上传 / ES reindex / 通知 / 投影更新（**事务不跨外部 IO**）
- **三重幂等**：`Idempotency-Key`（同 key 重试返首次结果）+ 乐观锁（`parent_asset_version`）+ DB partial unique（`logical_asset_id` 唯一 is_current）
- **写入入口**：通用 `AssetWriter`（PRD §4 主线）+ Aspect 专用 API（`POST /asset-tags` / `POST /algo-finish` 等，仍走同一 Writer 契约）

---

## 2.1 设计哲学：多产品参考、轻量栈实现

> **定位**：DataBrew **主借 DataHub 整体架构模式**（Entity-Aspect / SystemMetadata / Actions Framework），**辅借**其他参考实现的特色能力（Atlas 治理 / MLflow 版本 / Amundsen 发现层 / Iceberg 不可变 / K8s 乐观锁）。用 **PG + ES + GCS + Go 4 个组件**实现，不引入 DataHub runtime / Kafka / Neo4j / PDL（详见 §16）。

### 各能力的最近似参考

> **说明**：「最近似参考」=「该能力概念上最像哪个产品」；不代表「只有该产品有」，也不代表「我们的实现与之相同」。

| 能力 | 最近似参考 | 落位 |
|------|----------|------|
| Entity-Aspect 模式 + 强字段 entity 壳 | **DataHub** + OpenMetadata | §3.8 Entity 壳 + 4 Aspect |
| SystemMetadata（run_id / pipeline / actor）| **DataHub** | §4.9 `system_metadata` 子结构 |
| 异构 metadata 写入（多 Aspect 表 + JSONB）| **DataHub** | mcap_files / actions / metrics / eval_results |
| 字段级 before/after diff 审计 | **OpenMetadata** `ChangeDescription` | §4.9 `payload` 三段式 |
| typed lineage relations（split_from / contains / derived_from / merged_from / validated_by） | **Atlas** + DataHub | §3.6 边类型 |
| Classification Propagation（治理标签沿血缘自动传播）| **Atlas**（独有特色）| §7.6.1 TagPropagator（P1.5） |
| Actions Framework（事件订阅 + 自动副作用）| **DataHub** | §4.11（PG events / 可切 Pub/Sub） |
| 发现层信号（trending / popularity / favorited）| **Amundsen**（起家能力）| §3.8.2 `asset_usage_stats` Aspect + §4.11 UsageStatsAccumulator（P1.5） |
| logical entity + 单调 revision | **MLflow Model Registry** | §4 Revisions |
| 不可变路径 + 防碰撞 | **Iceberg snapshot** + DVC | §4.3 GCS `r<n>-<uuid4>/` |
| 乐观锁（条件写 + 409）| **K8s** `resourceVersion` | §4.6 写入幂等 |
| sha pin / 客户引用稳定 | **HuggingFace Hub** | §4.7 S4 frozen + `GET /logical-assets/{id}/current` |
| Compliance lineage 一次拉清单 | **DataHub** `searchAcrossLineage` | §11.5 `GET /audit/lineage-search`（P1.5） |
| AI agent 接入 metadata | **DataHub MCP Server** | P2 候选（§16.4） |

### 没有抄的（已否决）

#### A. 工程成本太高 / 不必要的运维

| 否决项 | 理由 |
|-------|------|
| DataHub **Kafka MCL** + **Neo4j** + **Pegasus PDL** | 多组件部署 + 多语言栈 + DSL 工具链；我们 PG events 等价 |
| Atlas **JanusGraph + HBase** | Hadoop 时代重型方案；我们 PG recursive CTE 等价 |
| DataHub / Atlas / OpenMetadata **全栈集群部署** | 都需要独立运维团队；我们 Go 后端单体嵌入 |

#### B. 业务模型不匹配

| 否决项 | 理由 |
|-------|------|
| DataHub / OpenMetadata **80+ Ingestion Connectors** | 我们是数据真源，不是「跨平台元数据镜像」 |
| Amundsen **Databuilder ETL** | 同上 |
| DataHub **Glossary / Domains / Compliance Forms** | 通用治理产物；我们用 Tag Source Registry + Domain-by-tag 替代，更轻 |
| Atlas **Apache Ranger 集成** | 我们尚无 RBAC 需求（P2+ 单独 ADR） |

→ **「不抄」≠「不好」**，只是当前 DataBrew 不需要。出现业务需求时再评估。

---

## 3. 资产层级模型（横向）

### 3.1 层级树

```text
raw_mcap                          asset_level = 0
    └── segment                   asset_level = 1   ← MCAP 下唯一一层
            ├── clip              asset_level = 2   ← 连续时间子窗（常见交付单元）
            ├── action            asset_level = 2   ← segment 上的原子动作/标注
            ├── frame             asset_level = 2   ← 离散帧 / 抽帧集合
            └── task              asset_level = 2   ← 任务执行窗（如 t1–t2 扫地）
                    └── action    asset_level = 3   ← task 内子动作（t1+1 抬手等）
```

时间轴示例：

```text
segment  |==============================================|
task          [========  sweep_floor  t1 ─── t2  ========]
action(L3)         ·              ·                    ·
                 t1+1           ……                  t2−1
clip(L2)    [=======]                                    ← 可与 task 并存
```

#### 3.1.1 Cardinality 与重叠规则（早期约束宽松）

| 规则 | 说明 |
|------|------|
| 一个 raw_mcap | 0–N 个 segment（QA / commit-segments 决定）|
| 一个 segment | 0–N 个 clip / frame / task / action(L2)（无上限）|
| 一个 task | 0–N 个 action(L3)（无上限）|
| **同 segment 下子资产时间窗** | **允许任意重叠**（同一时间区间可同时挂 clip + task + 独立 action，互不干扰）|
| **同 task 下 action 时间窗** | **允许重叠**（不强制顺序）|
| **跨 task 时间窗** | **允许重叠**（一段时间可属多个并行 task）|

> 设计意图：早期不靠 schema 强约束时序关系，靠业务标签 / 算法层面去理解；避免给「同时含『骑车 task』和『扫地 task』的多语义并存场景」加门槛。未来若出现治理需求再收紧。

### 3.2 一等资产能力矩阵

| 能力 | raw_mcap | segment | clip | action | frame | task | derived_asset |
|------|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| 独立 `asset_id` | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| `asset_tags`（5 类来源） | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| `asset_algo_latest` + events | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| `queries/run` 检索 | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| `delivery_items` 交付 | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| Provenance + 版本链 | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |

> **结论**：所有 7 种 `asset_type` 在 `assets` 表里是平等的一等行；差异仅在「物理 aspect 表」与「默认 materialization」，详见 §3.3。

### 3.3 `asset_type` 完整属性矩阵

> 取代旧 §3.3 / §3.4 / §3.5 三处分散字段定义。**字段归属**：所有非 Entity 壳字段都在 Aspect 表，详见 §3.8 + `schema-entity-aspect.md`。

| `asset_type` | level | 合法父类型 | 时间形态 | materialization 默认 | 专用 Aspect 表 | 典型来源 / 用途 |
|--------------|:---:|------------|---------|---------|---------|---------|
| `raw_mcap` | 0 | —（顶层）| 文件全范围 | materialized（必） | `mcap_files` | ingest 后的 MCAP 物理文件元数据实体 |
| `segment` | 1 | `raw_mcap` only | 连续时间窗 | materialized 多 / virtual 少 | — | MCAP QA / commit-segments 切分出的业务一等单元 |
| `clip` | 2 | `segment` only | 连续时间窗 | virtual / materialized（视交付需求） | — | 交付单元 / 客户回放 / 预览 |
| `frame` | 2 | `segment` only | 离散点 / 集合（见 §3.5） | virtual / materialized | —（帧 URI 在 `asset_content.files`） | 抽帧 / 关键帧导出 |
| `task` | 2 | `segment` only | 连续任务窗 | virtual 主（任务窗是语义边界，不必产文件） | P1 `metadata` / P1.5+ `tasks` | 任务执行（扫地 / 拣选 / 等） |
| `action` (L2) | 2 | `segment` | 连续标注窗 | virtual 主 | `actions`（1:1） | segment 上的原子动作 / 标注 / 算法输出 |
| `action` (L3) | 3 | `task` | 连续标注窗（⊆ task） | virtual 主 | `actions`（1:1） | task 内子动作（与 L2 共用 actions 表，靠 `parent_asset_id` + `asset_level` 区分） |
| `derived_asset` | 任意 | 按规则 | 任意 | materialized 主 | — | 算法派生 / 多父融合（`merged_from`） / A/B 实验产物 |

**遗留迁移**：`frame_set → frame`；`task_demo → task`。filter 兼容旧名 ≤ 一个版本周期（详见 §12.3）。

### 3.4 物化（virtual / materialized）

**字段归属**：`materialization / storage_uri / files` 三列在 **`asset_content`** Aspect 表（不在 `assets` 主表）。详见 `schema-entity-aspect.md`。

| 状态 | 含义 | GCS 产物 |
|------|------|---------|
| `virtual` | 无 GCS 产物；仅 metadata（时间窗 + 父引用） | `storage_uri = NULL AND files = empty` |
| `materialized` | 有 GCS 产物 | `storage_uri` 必符合 `assets/<logical_asset_id>/r<n>-<uuid4>/...` 形态（§4.3） |

**转换规则**：

- **virtual → materialized**：单向；同 `asset_id` 走 A 路由原地升级（§4.1）；写 `asset_materialized` event；**revision 不变**
- **materialized → virtual**：**禁止**（已 materialized 的物理文件不可逆转回虚拟态）

### 3.5 frame 形态

**字段归属**：`frame_kind / frame_count` 在 **`asset_content`** Aspect 表。

| `frame_kind` | 含义 | 物理 |
|--------------|------|------|
| `single` | 单帧资产 | GCS：`r<n>-<uuid4>/frame.png` |
| `set` | 多帧集合（≤100 帧）| P1 用 `metadata.frame_items[]`；P1.5 视用量抽 `frame_set_items` 子表 |

→ 同一 segment 下 single frame 与 set frame 可并存，无互斥约束。

### 3.6 边类型约定（`asset_relations.relation_type`）

#### 3.6.0 边方向统一规则（ER1，rev.10 M1 决策）

> **`parent_asset_id` = 主语（subject），`child_asset_id` = 宾语（object），`relation_type` = 谓语。**
> **永远朝 parent 方向读：「parent {relation_type} child」是合法句子**。
>
> 历史包袱：列名 `parent / child` 在 `revision_of / derived_from / merged_from / sampled_from` 语境下反"时间父子"直觉（新版做 parent、源做 parent）。**rev.10 不改列名**（已有代码 / 索引依赖），用文档约定 + helper VIEW 解决。

| relation_type | parent_asset_id（主语） | child_asset_id（宾语） | 读法 |
|---|---|---|---|
| `split_from` | 切出来的 child | 源 parent | "clip_X is **split_from** seg_Y" |
| `contains` | 父容器 task | 内含 action | "task_T **contains** action_A" |
| `sampled_from` | 采样产物 frame | 源 segment | "frame_F is **sampled_from** seg_Y" |
| `merged_from` | 融合产物 derived | 源 asset | "merged_X is **merged_from** source_Y"（多父→多行） |
| `derived_from` | 派生产物 derived | 源 asset | "derived_X is **derived_from** source_Y" |
| `revision_of` | **新版** asset | **老版** asset | "clipA_v2 is **revision_of** clipA_v1" |

→ **统一规则一句话：「parent 是关系的发起方 / IS-A 的主体」**，朝 parent 读永远成立。

#### 3.6.0.1 helper VIEW（直觉化查询）

```sql
CREATE VIEW asset_relations_readable AS
SELECT
  parent_asset_id AS subject,         -- 主语：关系的起点 / IS-A 的主体
  relation_type   AS predicate,       -- 谓语
  child_asset_id  AS object,          -- 宾语：关系指向的对象
  metadata,
  created_at
FROM asset_relations;

-- 用法（ad-hoc / SQL 排障，自然语言风）：
-- SELECT subject, predicate, object
-- FROM asset_relations_readable
-- WHERE predicate = 'revision_of' AND object = 'clipA_v1';
--   → 谁是 clipA_v1 的 revision？答：clipA_v2
```

#### 3.6.0.2 反向查询模板（最常被搞错的两个）

```sql
-- ① 查「这个 asset 的新版本」
SELECT parent_asset_id AS new_version
FROM asset_relations
WHERE child_asset_id = 'clipA_v1' AND relation_type = 'revision_of';

-- ② 查「这个 asset 派生自哪些源」
SELECT child_asset_id AS source_asset
FROM asset_relations
WHERE parent_asset_id = 'derived_X' AND relation_type = 'derived_from';
```

#### 3.6.1 评估 / 质检关系 ≠ asset_relations 边

> 早期版本曾计划加 `validated_by` 边类型（asset → eval_result），**rev.9 移除**。

**理由**：`asset_relations` 表 `parent_asset_id / child_asset_id` 均 FK 到 `assets.asset_id`；`asset_eval_results` 不是 asset（是 metadata 派生表），强行入 `asset_relations` 会破坏 FK 约束。

**正确做法**：`asset_eval_results.asset_id` 列**已经**表达了「这次评估在哪个资产上」的关联，**无需另起 edge**。Provenance API 渲染血缘图时，从 `asset_eval_results` 派生「validated_by」**逻辑边**作为展现层概念（不持久化在 `asset_relations`）。

| 关系 | 真源表 | Provenance 展现 |
|------|------|------|
| 评估事实 | `asset_eval_results(eval_id, asset_id, eval_name, version, payload, ...)` | edge `validated_by`（虚线，标 eval_name@version） |
| 指标投影 | `asset_metrics(asset_id, metric_key, value, source, ...)` | node 摘要 |
| 时间线 | `asset_events` event_type=`eval_result_added` | Timeline 条目 |

#### 3.6.1 评估 / 质检关系 ≠ asset_relations 边

> 早期版本曾计划加 `validated_by` 边类型（asset → eval_result），**rev.9 移除**。

**理由**：`asset_relations` 表 `parent_asset_id / child_asset_id` 均 FK 到 `assets.asset_id`；`asset_eval_results` 不是 asset（是 metadata 派生表），强行入 `asset_relations` 会破坏 FK 约束。

**正确做法**：`asset_eval_results.asset_id` 列**已经**表达了「这次评估在哪个资产上」的关联，**无需另起 edge**。Provenance API 渲染血缘图时，从 `asset_eval_results` 派生「validated_by」**逻辑边**作为展现层概念（不持久化在 `asset_relations`）。

| 关系 | 真源表 | Provenance 展现 |
|------|------|------|
| 评估事实 | `asset_eval_results(eval_id, asset_id, eval_name, version, payload, ...)` | edge `validated_by`（虚线，标 eval_name@version） |
| 指标投影 | `asset_metrics(asset_id, metric_key, value, source, ...)` | node 摘要 |
| 时间线 | `asset_events` event_type=`eval_result_added` | Timeline 条目 |

### 3.7 不变式索引

> 本节是**索引页**。PRD 全文所有写入校验不变式集中编号；详细定义、payload、拒绝码留在来源章节，本表只**收录 + 链接**。Validator 实现按本表逐条强制；违反一律 422（个别注明 409 / 403）。

| 编号 | 不变式 | 来源 |
|:---:|------|------|
| **L1** | `segment` 父类型必须 = `raw_mcap` | §3.3 |
| **L2** | `clip / frame / task` 父类型必须 = `segment` | §3.3 |
| **L3** | `action(L3)` 父类型必须 = `task` 且时间窗 ⊆ task ⊆ 祖先 segment | §3.3 |
| **L4** | 禁止 `segment → segment` | §3.3 |
| **L5** | 禁止 task 下挂 clip / frame / task（P1）| §3.3 |
| **L6** | 子资产 `mcap_file_id` 必继承根 segment | §3.3 |
| **L7** | 有父或多父时必须写 `parent_asset_id` 或 `asset_relations` + event | §3.7 |
| **L8**（rev.10）| `asset_type ∈ {raw_mcap, segment, clip, action, frame, task}` ⇒ `asset_lineage.mcap_file_id NOT NULL`；`derived_asset` 允许 NULL（跨多 mcap 聚合）| §3.3 + schema §4.1 |
| **M1** | `materialization='virtual'` ⇒ `storage_uri IS NULL AND files = empty` | §3.4 |
| **M2** | `materialization='materialized'` ⇒ `storage_uri` 符合 `assets/<logical>/r<n>-<uuid4>/...` | §3.4 / §4.3 |
| **M3** | `materialized → virtual` 禁止 | §3.4 |
| **R1**–**R6** | Revision 不变式（同 logical 仅一条 is_current / 严格线性 / B 路由原子事务 等）| §4.5 |
| **AE1**–**AE5** | 审计契约（写业务表必写 event / append-only / 顶层 4 字段必填 / diff 必带 / hard-delete 留快照）| §4.9.7 |
| **I4**（=I1+I3）| 写入幂等三层（Idempotency-Key + 乐观锁 + DB partial unique）| §4.6 / §9.3 |
| **IR1.1–IR1.3** | 读写路由（点查走 PG / 列表走 ES / 合规校验走 PG）| §10.3 |
| **Tag1**–**Tag7** | Tag Source Registry 校验（source 必在 registry / writable_by / requires_source_name 等）| §7.6 |
| **Lifecycle R1**–**R6** | Lifecycle 删除约束（已交付禁删 / superseded 禁软删 / is_current 禁直接 archived 等）| §6.3 |
| **DR1**–**DR4** | Delivery Rules + commit C2 校验 | §9.2 / §11.4 |
| **LS1**–**LS4** | Lineage-search 不变式（角色门禁 / 审计-of-审计 / 分页 / 不缓存）| §11.5.4 |
| **AR1**（rev.11 修订）| `PATCH /assets/{id}` 命中 `revision_triggers` 任一字段 → **422 + 错误消息指引 `POST /assets/{id}/revisions`**（不再自动升级，避免 client 误触发版本生成）| §4.1.1 |
| **G3**（rev.9）| GCS PUT（ifGenerationMatch=0）必须在 PG 事务前；事务 commit 失败时 AssetWriter 立即 DELETE（带 etag）；24h 后 bucket lifecycle GC 兜底孤儿 | §4.3.1 |
| **D1**（rev.9）| Event payload diff = 列字段精准 before/after + JSONB 整字段 before/after；JSONB 内嵌套不做深 diff；> 50KB 时截断为 `{"__truncated__":true}` | §4.9.6.1 |
| **I5**（rev.9）| handler/projector 生成 event 时 `idempotency_key` 用新 UUID（不走 deterministic key）；去重责任在下游表 unique constraint | §4.9.6.2 |
| **TT3**（rev.9）| algo_sdk 写 `asset_algo_latest` 同事务必须双写 `asset_tags(source=algo_sdk, key=algo.<name>.version, value=<v>)`；缺则 422 | §7.0 + §7.6 |
| **TP1**（rev.9）| `propagation=descendants` 的 tag 不在 B 路由事务内复制；由 TagPropagator 订阅 `asset_revised / asset_created` 异步补；PII 类 tag 必须标 `critical=true` 触发告警 | §7.6.1 / §7.6.2 |
| **TP2**（rev.9）| delivery_rules 校验 PII 类 tag 时**按 `logical_asset_id` 聚合所有版本**（窗口期保护）；commit C2 在 PG 真源再校验一次 | §7.6.2 / §10 |
| **MR1**（rev.9）| `delivery_rules` 中 `metrics.rating.*` 默认 scope = 当前 is_current 版本（不跨 logical 聚合）；B 路由生新版后必须重评，否则 avg=NULL 阻交付 | §7.4.1 |
| **IR1.1b**（rev.10）| 详情面板可走 ES，OpenAPI 须标 `x-consistency: eventual`；主体禁止 ES | §10.3.1 |
| **ES1**（rev.10）| 单 ES doc 序列化 ≤ 1MB；超限自动 Top-50 + `*_truncated` | §10.2.1 |
| **WR1**（rev.10）| 子资产变更触发父 reindex 须经 Debounce 500ms；批量 ≥10 条须走 `actions:batch` | §10.5.1 |
| **SC1**（rev.10）| sync-check：`ready` 当 `es_last_indexed_event_seq >= expected_seq`；`lagging` ≥ 30s 告警 | §11.2.1 |
| **C2R1**（rev.10）| delivery commit 与 resolve-conflicts 分 API；commit 只认 `expected_revision`（业务版），不认 `row_version` | §11.4 |
| **QR1**（rev.10）| `POST /queries/run?wait_for_seq=N` 阻塞至 ES 水位或 408；不另起 run-strong | §10.3.2 |
| **R7**（rev.10）| `asset_algo_latest.is_pinned=true` 资产禁止 supersede；走 B → 422 | §6.3 |
| **R8**（rev.10）| `archived / superseded / soft-deleted` 资产拒收新 tag；直接写 422，propagator 静默丢弃 + metric | §6.3 |
| **DR5**（rev.10）| `delivery_rules` 必带 `dsl_version`；引擎升级保持旧版本前向兼容；不自动迁移 | §9.2 |
| **DR_ENF**（rev.10）| `enforce_mode`：block 阻 commit / warn 仅警告 / tag_only 仅打 `delivery_ready` tag 不阻 commit | §9.2 |
| **Tag1**（rev.10 提升）| `tag_key ∈ {label, labels, primary_label, action_id}` 一律 422（避免与 action label 混淆） | §9.1 |
| **AE6**（rev.10）| `algo_finished` event 必填 `asset_algo_latest.run_inputs`（params + input_asset_ids + code_commit + image_digest）| §4.4.3 / §4.9.7 |
| **AV1**（rev.10）| `asset_events.actor` 必须命中命名约定正则；机器类必带 `@<version>` | §4.9.2.1 |
| **AK1**（rev.10）| `asset_algo_latest.algo_kind ∈ {processing, qa, split, enrichment}`；PK 含 `algo_kind` 列 | §4.4.2 |
| **RL1**（rev.10）| `asset_full` VIEW 列必须覆盖所有 Aspect 表列；CI lint 缺列 fail PR | §12.4.6 |
| **RL2**（rev.10）| `assertion job` 24h 对账 Aspect 表与 `assets` 行数；孤儿/缺主体告警 | §12.4.6 |
| **RL3**（rev.10）| 新 handler 必须经 `usecase.GetWithScope` 显式声明 scope；禁止直接 `repo.Get` 取宽对象 | §12.4.6 |
| **LA1**（rev.10）| `logical_assets.asset_type` 首版决定，不可 UPDATE | §8.4 |
| **LA2**（rev.10）| `current_revision <= total_revisions`（CHECK 约束 DB 层兜底）| §8.4 |
| ~~**LA3**~~（rev.10 → rev.11 **已消除**）| ~~`current_asset_id` 必须满足 `is_current=true AND logical_asset_id=自身`~~ → 砍 `current_asset_id` 冗余指针后自然消失 | §8.4 |
| ~~**LA4**~~（rev.10 → rev.11 **已消除**）| ~~B 路由事务必须同时更新 `logical_assets` 5 字段~~ → 砍指针后只需更 `current_revision / total_revisions / updated_at` 3 字段 | §8.4 |
| **OPT1**（rev.10）| B 路由新 asset_id 行 `row_version=1` 起（DEFAULT），与老版无关；`parent_asset_version` 校验老版的 row_version | §4.6.1 |
| **AS1**（rev.10 H1）| 所有 Aspect 表统一含 `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()` + `idx_*_updated` 索引；AssetWriter 每次 UPDATE 必同步更新 | schema §4.1-§4.4 |
| **AS2**（rev.10 §16.7）| 所有 Aspect 表统一含 `extra JSONB NOT NULL DEFAULT '{}'` 作为实验区；新字段先进 extra → 注册 schema_registry yaml → 升列 | schema §4.1-§4.4 + §16.7 |
| **OPT3**（rev.10 J1）| `asset_content.storage_uri` UNIQUE（NULL 多 virtual 允许）；同 GCS 路径不可属两个 asset_id；G3 双 commit 协议兜底 | schema §4.2 |
| **IDX1**（rev.10 J1）| 查询性能索引：`asset_content.storage_uri UNIQUE`、`asset_usage_stats.{view_count, download_count, favorited_count} DESC`、`asset_governance.(tenant_id, project_id)`、`asset_metrics.(metric_key, scored_at DESC)` | schema §4 + §7.4 |
| **ER1**（rev.10 M1）| `asset_relations` 边方向统一：`parent_asset_id` = 主语 / IS-A 主体，`child_asset_id` = 宾语；朝 parent 读永远成立「parent {relation_type} child」 | §3.6.0 |

> **维护规则**：本表只**收录**不变式；新增 / 修改不变式时**同时**在此表加 / 改一行 + 在来源章节给出详细定义。reviewer 可凭此表一次 audit 所有 Validator 实现。


### 3.8 Entity-Aspect 落地（概览）

> ⚠️ **rev.12 已撤回此节方案**：4 张通用 Aspect 表（asset_lineage / asset_content / asset_governance / asset_usage_stats）+ asset_full VIEW + AssetWriter 拆分器整套设计**已废弃**。
>
> 撤回理由：现网 8 张 satellite 表（asset_tags / asset_algo_latest / asset_metrics / asset_eval_results / actions / asset_events / mcap_files / asset_relations）已经是 80% Entity-Aspect 模式，再拆 4 张表的并发收益接近零；algo 写路径早已不锁 assets 行（见 `algo_usecase.go:10-11` 代码注释）。
>
> 当前 schema SoT：**[`../schema.md`](../schema.md)** —— 保留 38 列 assets 宽表 + 8 张 satellite + 新增 algo_runs / customers。
>
> 本节以下内容仅作历史决策追溯。

#### 3.8.1 设计原则

> **「Entity 是身份证，Aspect 是档案袋。」**

`assets` 是一张**极薄 Entity 壳表**（9 列，永远不增加业务字段），所有业务/治理/产物字段都拆到 **Aspect 表**，靠 `asset_id` FK 关联。多系统并发写不同 Aspect 表互不阻塞。

#### 3.8.2 表清单

**Entity 壳（1 张）：**

| 表 | 用途 | 列数 |
|----|------|------|
| `assets` | 身份 + 版本链 + 系统字段（**永远不超过 9 列**）| 9 |

**通用 Aspect 表（4 张）：**

| 表 | 含义 | 写入特征 |
|----|------|---------|
| `asset_lineage` | **结构血缘 + 时间维度**（mcap_file_id / parent / root / asset_level / split_* / 偏移 / `start/end_timestamp_ns` / `duration_ns` / `segment_locator`） | 创建即写，几乎不改 |
| `asset_content` | **产物内容**（materialization / storage_uri / files / frame_kind / frame_count / summary_text） | A 路由触发原地改 / B 路由生新行 |
| `asset_governance` | **治理状态**（lifecycle / owner / reviewer / retention / expire_at / delivery_count / last_delivered_* / tenant / project） | 低频写（治理操作） |
| `asset_usage_stats`（P1.5）| **使用信号**（view_count / download_count / trending_score / favorited_count / last_viewed_at / last_downloaded_at） | **高频累加**（独立表避免与 governance 行锁竞争） |

**专用类型 Aspect 表（已有，不改）：**

| 表 | 关联类型 |
|----|---------|
| `mcap_files` | raw_mcap（1:1） |
| `actions` | action（1:1） |
| `asset_tags` | N:1（多源标签） |
| `asset_algo_latest` | N:1（每 algo_name 一行，支持 pin） |
| `asset_metrics` | N:1（标量指标） |
| `asset_eval_results` | N:1（评估原档，不进 ES） |
| `asset_relations` | N:N（结构边 + 版本边；`revision_of` 边 `metadata` JSONB 装版本原因） |
| `asset_events` | N:1（append-only 审计日志） |
| `logical_assets` | 1:1 via logical_asset_id |

#### 3.8.3 读 / 写路径

- **读路径**：通过 `asset_full` VIEW（assets LEFT JOIN 4 张通用 Aspect）一次返回完整对象；`repos.go` 仅改 `FROM` 目标，字段不变。点查走 PG，列表走 ES（§10.3 IR1）。
- **写路径**：**VIEW 只读**；写入由 `AssetWriter` 拆分到 Entity 壳 + 各 Aspect 同事务（详见 §4 + `schema-entity-aspect.md` 写入流程伪代码）。

#### 3.8.4 并发优势

```text
场景：clipA 同时被三个系统操作

算法 A 写 asset_algo_latest → 自己的行
算法 B 写 asset_metrics     → 自己的行
运营 C 写 asset_tags        → 自己的行
前端 D view_count += 1      → 自己的行（asset_usage_stats 独立表）

→ 4 个写操作同时发生，assets 壳表完全不参与，零行锁竞争
→ 数据库并发吞吐线性扩展
```

旧模型（所有字段在 `assets` 一行）下，4 个操作全部竞争同一行锁。Entity-Aspect 模式是消除该瓶颈的核心机制。

---

## 4. 版本与迭代（Revisions，纵向）

### 4.1 决策与路由

**两条路由，决定「这次写入要不要新建 asset_id」：**

| 路由 | 触发 | 行为 |
|------|------|------|
| **A 原地** | 元数据 / tag / algo status / confidence 微调 | 同 `asset_id` 改字段 + 写 event；不动 GCS |
| **B 新版** | `storage_uri` / `files` / 时间窗 / `primary_label` / `labels` 变；或显式标 `manual_fix` 原因（写入 `asset_relations(revision_of).metadata`） | 新 `asset_id`，老的 supersede + `revision_of` 边；GCS 落新 `r<n+1>-<uuid4>/...` |

**关键约束：**

- **版本在 logical 实体上做**：同 `logical_asset_id` 多 revision；严格线性 v1→v2→v3，禁止分叉；A/B 实验走 `derived_asset`
- **路由由服务端自动判定（决策 A2）**：触发字段清单进 `AssetTypeRegistry.revision_triggers[]`，按类型可配；业务方不手判
- **A 和 B 都写 event**：审计（§4.9）覆盖所有变更；confidence 0.91→0.93 走 A 也能在 timeline 看到 `before/after` diff

#### 4.1.1 显式路由：PATCH 拒绝触发 B（决策 A1，rev.11 修订）

> **rev.9 曾决策 A2 自动升级**：PATCH 命中 B 触发字段时服务端透明生成新 asset_id。
> **rev.11 翻转为 A1 显式拒绝**：避免 client 误触发版本生成的 footgun。

API 入口对 client **完全显式**：

```text
PATCH /assets/{id}        ──► 服务端 diff
   │
   ├─ changed_fields ∩ revision_triggers = ∅  → 走 A：原地 UPDATE，response 200
   │
   └─ 命中 revision_triggers                   → 422 + 错误指引：
                                                  {
                                                    "error": "revision_triggers_field_changed",
                                                    "fields": ["storage_uri", "primary_label"],
                                                    "hint": "use POST /assets/{id}/revisions"
                                                  }

POST /assets/{id}/revisions  ──► 显式 B 路由（client 主动声明）
```

**翻转理由（rev.11 决策）：**

- A2 自动升级是 magic：client 一次不小心改了 `storage_uri`，**真生产数据多了一行**，计费 / 监控 / 错误恢复全乱
- A2 第二次请求拿到 404（client 假设 asset_id 不变，但服务端偷偷换了）
- A1 显式像 `git commit --amend`：要做有意识的选择
- revision 是业务事件，必须 client 显式 → 平台留 audit trail（actor 故意而非误触）

**强约束：**

| 规则 | 说明 |
|------|------|
| `PATCH` 命中 B 触发字段 | **422**（不再升级；client 必须改用 `POST /assets/{id}/revisions`）|
| `POST /revisions` 缺 `Idempotency-Key` | 422 |
| `POST /revisions` 缺 `parent_asset_version` | 422（乐观锁必传）|
| response status | A 路由 200；B 路由（显式 POST）201 + Location 头 |

### 4.2 数据模型

#### 4.2.0 `logical_asset_id` 是什么？

**一句话：`logical_asset_id` 是「同一个东西的稳定身份证」——不管它被改了多少次、生成多少个版本、文件落在哪里，它永远不变。**

**类比：**

| 概念 | 类比 |
|------|------|
| `asset_id` | 一张实体身份证（每次换证都不同） |
| `logical_asset_id` | 身份证号（一辈子不变） |

**核心规则：**

- 首版资产：`logical_asset_id = asset_id` 自动设
- 走 B 路由生新版本：新 `asset_id` 不同，但 **继承父的 `logical_asset_id`**；`revision = 父.revision + 1`
- 同一 `logical_asset_id` 内只能有一条 `is_current = true`（partial unique index 强约束）
- GCS 路径前缀：`assets/<logical_asset_id>/r<revision>/...`

**`logical_asset_id` ≠ `parent_asset_id`（容易混）：**

| 维度 | 字段 | 表达 | 例 |
|------|------|------|-----|
| **横向血缘**（不同实体之间的切出关系） | `parent_asset_id` | clip 是从哪段 segment 切的 | clipA_v1/v2/v3 的 `parent_asset_id` **都**指向 `seg_Y` |
| **纵向版本**（同一实体的多次迭代） | `logical_asset_id` | clipA 的三个版本是同一物 | clipA_v1/v2/v3 的 `logical_asset_id` **都**等于 `L_clipA`，三个 `asset_id` 不同 |

```text
横向（parent_asset_id）：mcap_X ──split──► seg_Y ──split──► clipA_v?
纵向（logical_asset_id）：               clipA_v1 → clipA_v2 → clipA_v3   都 logical_asset_id = L_clipA
```

→ 两者**正交**，缺一不可：parent 回答「这玩意从哪来」；logical 回答「这玩意改过几次、当前是哪版」。

**实例（数据库行）：**

```sql
asset_id    parent_asset_id   logical_asset_id  revision  is_current
mcap_X      NULL              L_mcap_X          1         true
seg_Y       mcap_X            L_seg_Y           1         true
clipA_v1    seg_Y             L_clipA           1         false   ← 老版
clipA_v2    seg_Y             L_clipA           2         false   ← 老版
clipA_v3    seg_Y             L_clipA           3         true    ← 当前版
```

观察：clipA_v1/v2/v3 的 `parent_asset_id` 相同（都是 seg_Y），`logical_asset_id` 也相同（都是 L_clipA），但 `asset_id` 不同。

#### 4.2.1 `assets` Entity 壳新列（只 3 列）

```sql
logical_asset_id  TEXT  NOT NULL  -- 同一逻辑产物的版本共享；首版 = asset_id
revision          BIGINT NOT NULL DEFAULT 1   -- rev.10 E2：与 row_version / event_seq 类型统一
is_current        BOOL  NOT NULL DEFAULT true
```

> **`revision_reason` 不进 Entity 壳**。版本原因存于 `asset_relations(relation_type='revision_of').metadata` JSONB 字段（rev.9 决策；详见 `schema-entity-aspect.md` §3.3）：
> - 首版无 `revision_of` 边 → 无 revision_reason，自然合理
> - 查询「某资产这版怎么来的」= `SELECT metadata FROM asset_relations WHERE child_asset_id = X AND relation_type = 'revision_of'`
> - 审计权威记录仍在 `asset_events.system_metadata`（双视图：边带便于点查 / events 是历史 audit trail）

#### 4.2.2 Partial unique index（强约束）

```sql
CREATE UNIQUE INDEX uq_assets_current_per_logical
  ON assets (logical_asset_id) WHERE is_current = TRUE AND is_deleted = FALSE;
```

#### 4.2.3 `asset_relations.relation_type` 新枚举

`revision_of`（边的 `metadata` JSONB 字段装版本原因；见 §4.2.1）

#### 4.2.4 `asset_events.event_type` 新值

`asset_revised`、`asset_materialized`、`asset_pinned`、`asset_unpinned`

### 4.3 GCS 产物布局（强制）

```text
gs://<bucket>/assets/<logical_asset_id>/r<revision>-<uuid4>/<filename>
```

**例：**

```text
gs://cyb-prod/assets/L_clip_a1b2c3/r1-550e8400-e29b-41d4/video.mp4   ← v1
gs://cyb-prod/assets/L_clip_a1b2c3/r2-f47ac10b-58cc-4372/video.mp4   ← v2
```

**说明：**

- `r<revision>` 保留人类可读的版本语义；`-<uuid4>` 防止并发 writer 碰撞同一路径（无需 DB 做路径协调者）
- 任何 B 路由产物必须落新 URI；同 URI **不**覆盖
- GCS 写对象用 `ifGenerationMatch=0` 兜底幂等（重试不产生双份对象）
- 不依赖 GCS object versioning（避免与 IAM/lifecycle 策略耦合）
- UUID4 由 AssetWriter 在事务开始前生成，写入 `asset_content.storage_uri`

#### 4.3.1 GCS↔PG 双 commit（决策 G3，rev.9）

**问题：** UUID + ifGenerationMatch=0 在 PG 事务开始**前** PUT；若 PG 事务回滚（乐观锁冲突 / 校验失败），GCS object 已落地、无 DB 引用 = 孤儿。

**协议（G3 + G4 兜底）：**

```text
Step 1  AssetWriter 生成 UUID4 → 候选 URI: assets/<logical_id>/r<n>-<uuid>/file
Step 2  PUT GCS（ifGenerationMatch=0），保留 generation_number etag
Step 3  PG 事务（INSERT assets / INSERT asset_content / INSERT events ...）
        事务内 SELECT 校验 storage_uri 全局唯一（额外保险）
Step 4  事务 commit 成功 → 返回 client
        事务 commit 失败 → 立即 DELETE GCS（ifGenerationMatch=<etag> 保护：只删自己 PUT 的那个 generation）

兜底（极端：Step 4 DELETE 也失败 / 进程崩溃）：
        GCS bucket lifecycle 规则定期扫 r*/ 下 > 24h 且无 DB 引用的 object，offline GC
```

**实现要点：**

| 点 | 说明 |
|----|------|
| `defer cleanup` | AssetWriter 在 Go `defer` 中调 GCS DELETE；成功提交后 `cleanup` 设为 no-op |
| `ifGenerationMatch=<etag>` | DELETE 时带 generation，**不会误删**其他 writer 同名（理论上 UUID 已防碰撞，纯保险）|
| DELETE 失败 metrics | 上报到 `gcs_orphan_cleanup_failed` 指标；> 0 触发告警 |
| offline GC 周期 | bucket lifecycle `age > 1 day AND prefix r*/`，**且**对账 `assets/asset_content` 无该 storage_uri 才删 |
| 接受短窗孤儿 | 24h 内最多孤儿，可接受成本 |

### 4.4 算法当前态：`asset_algo_latest`（含 pin + 多 kind + 输入复现）

`asset_algo_latest` 是「这个资产在每种算法上当前跑到哪个版本」的真源。支持处理类 / 质检类 / 切分类等多种算法形态共用一张表（决策：扩 `algo_kind` 列而非开新表），并强约束输入复现（决策：加 `run_inputs JSONB`）。

#### 4.4.1 表结构（rev.10 新列）

```sql
asset_algo_latest (
  asset_id      TEXT  NOT NULL,
  algo_kind     TEXT  NOT NULL DEFAULT 'processing',  -- rev.10：processing|qa|split|enrichment
  algo_name     TEXT  NOT NULL,
  algo_version  TEXT  NOT NULL,
  status        TEXT  NOT NULL,                       -- ok|failed|running
  run_id        TEXT,
  run_inputs    JSONB NOT NULL DEFAULT '{}',          -- rev.10：reproducibility 强存储
  started_at    TIMESTAMPTZ,
  finished_at   TIMESTAMPTZ,
  is_pinned     BOOL  NOT NULL DEFAULT false,
  pinned_at     TIMESTAMPTZ,
  pinned_by     TEXT,
  metadata      JSONB NOT NULL DEFAULT '{}',
  extra         JSONB NOT NULL DEFAULT '{}',          -- §16.7 schema agility 实验区
  PRIMARY KEY (asset_id, algo_kind, algo_name)
);

CREATE INDEX idx_aalgo_kind_name ON asset_algo_latest(algo_kind, algo_name, algo_version);
CREATE INDEX idx_aalgo_pinned    ON asset_algo_latest(asset_id) WHERE is_pinned = true;
```

#### 4.4.2 `algo_kind` 多类型共用一张表（rev.10 决策）

| algo_kind | 业务含义 | 写入方 actor 前缀 | 例 |
|-----------|---------|----------------|----|
| `processing` | 处理类算法（生成 action / clip / frame）| `algo_sdk:` | hand_track@2.0 |
| `qa` | 质检类（评估 → 写 metrics + eval_results）| `qa_pipeline:` | action_completeness@1.2 |
| `split` | 切分类（生成子资产）| `split_pipeline:` | clip_segmenter@1.0 |
| `enrichment` | 元数据增强（贴 tag / 抽 summary）| `algo_sdk:` 或 `llm:` | scenario_classifier@1.0 |

**为什么不开 `asset_qa_latest` / `asset_split_latest` 单表：** 同一张表 + `algo_kind` 列 = 一套 pin 机制 / 一套 ES 投影 / 一套查询语法；扩新 kind 只加 enum 不加表（符合 §16.7 schema agility）。

#### 4.4.3 `run_inputs` 强存储（reproducibility 契约）

**强约束（rev.10 新不变式 AE6）：** 任何 `algo_finished` event 必须填 `run_inputs`（含 params + input_asset_ids + code_commit + image_digest）；缺则 422。

```jsonc
// run_inputs 标准字段
{
  "params": {                          // 算法参数 dict
    "confidence_threshold": 0.8,
    "model_size": "large"
  },
  "input_asset_ids": ["seg_X001", "seg_X002"],   // 输入资产快照（带 revision）
  "input_revisions":  {"seg_X001": 1, "seg_X002": 2},  // 复现时锁版用
  "code_commit": "git:abc123...",      // 算法代码 commit hash
  "image_digest": "sha256:..."         // 容器镜像 digest
}
```

**Reproducibility 收益：** 3 个月后追问「为什么 clipA 上 hand_track v2.0 跑出 confidence=0.91」→ `run_inputs` 完整还原参数 + 输入 + 代码版本 + 镜像。

**P1.5+ 演进：** 若业务需「按 run_id 反查影响的所有资产 + 批量重跑」→ 升 `algo_runs` 一等实体（OOS §14 → P1.5 候选）。

#### 4.4.4 Pin 机制（不变）

- 默认按 `finished_at desc` 隐式取 current
- `is_pinned=true`：新一次 finish **不**覆盖 latest 行；只追加 event + ES `algo_versions_seen`
- API：`PinAlgoVersion(asset_id, algo_kind, algo_name, version)` / `UnpinAlgoVersion(...)`；events 留 `asset_pinned/asset_unpinned`
- 用途：算法回滚、A/B 锁版、合规冻结
- **与 R7 联动**：pinned 资产禁止 supersede（§6.3）

### 4.5 Revision 不变式（Validator）


| 规则                                                                                                                          | 违反                 |
| --------------------------------------------------------------------------------------------------------------------------- | ------------------ |
| 同 `logical_asset_id` 仅一条 `is_current=true`                                                                                  | DB unique index 拒绝 |
| `CreateRevision(old_id)` 要求 `old.is_current = true`（N1 严格线性）                                                                | 422                |
| 新版本 `revision = old.revision + 1`                                                                                           | 422                |
| `revision_of.to` 必须是同 `logical_asset_id` 的上一版                                                                               | 422                |
| 走 B 时 `storage_uri` 必须新（与所有历史版本不同）                                                                                          | 422                |
| B 路由必同事务：插 new asset + `revision_of` 边 + `asset_revised` event + 更新 old.is_current=false + old.lifecycle_state='superseded' | 422 |
| 走 B 时 `storage_uri` 格式必须为 `assets/<logical_id>/r<n>-<uuid4>/<filename>` | 422 |


### 4.6 写入幂等

- 所有写入 API **必填** `Idempotency-Key` header（缺 → 400）
- 同 key 重试 → 返回首次结果；同 key 不同 payload → 409
- `CreateRevision` 必传 `parent_asset_version`（对应 `assets.row_version BIGINT`，乐观锁；**与业务 `revision` 不是同一字段**），服务端事务做条件写；不符 → 409
- GCS deterministic name + `ifGenerationMatch=0` 防双份对象
- 异步副作用（ES reindex）按 `event_seq` 推进，天然幂等

#### 4.6.1 B 路由 row_version 规则（OPT1 不变式，rev.10 F1 决策）

**OPT1：** B 路由生成的**新 `asset_id` 行 `row_version=1` 起**（DEFAULT），与生成前老版的 `row_version` **无关**。乐观锁是 per-row 概念，跨 asset_id 不传递。

**B 路由完整流程（含乐观锁校验）：**

```text
Client 请求:
  POST /assets/clipA_v1/revisions
  Idempotency-Key: <uuid>
  { parent_asset_version: 5,                ← 老版 clipA_v1 当前的 row_version
    changes: {storage_uri: "gs://.../r2-<uuid>/video.mp4"} }

Server 事务（同事务原子）:
  ① SELECT row_version FROM assets WHERE asset_id='clipA_v1' FOR UPDATE;
     若 row_version != 5  → ROLLBACK + 409 + body { actual_row_version: <真值> }

  ② INSERT assets (
       asset_id='clipA_v2', logical_asset_id='clipA_v1', revision=2,
       row_version=1,                       ← OPT1: 新行从 1 起（DEFAULT），不继承老版的 5
       is_current=true, ...);

  ③ UPDATE assets SET
       is_current=false,
       row_version=row_version+1            ← 老版 row_version: 5 → 6
     WHERE asset_id='clipA_v1' AND row_version=5;  ← 双保险条件写
     若 0 rows  → ROLLBACK 409（并发冲突）

  ④ UPDATE logical_assets SET                  -- rev.11：只更 3 字段（current_asset_id 列已砍）
       current_revision = 2,
       total_revisions = total_revisions + 1,
       updated_at = now()
     WHERE logical_asset_id='clipA_v1';

  ⑤ INSERT asset_relations (revision_of edge) + asset_events (asset_revised);
  COMMIT;
```

**关键规则：**

| # | 规则 |
|---|------|
| 1 | `parent_asset_version` 校验**老版 row_version**（不是新版） |
| 2 | 新版 `row_version` 总是 1（DEFAULT）—— OPT1 |
| 3 | 老版 `row_version` 在被 `UPDATE is_current=false` 时递增 +1（与所有 A 路由 UPDATE 规则一致：任何字段变都递增） |
| 4 | 新版第一次被 A 路由更新时，client 传 `parent_asset_version=1`，服务端 `WHERE row_version=1` 命中正常 |
| 5 | 跨 asset_id 比较 row_version **没有意义**（per-row 概念，不可跨行传递）|

### 4.7 交付与版本（默认快照 + 通知）

- `delivery_items.asset_id` 永远指**具体版本**；老链接稳定
- v2 完成（B 路由）→ projector 监听 `asset_revised` event → 找出引用 v1 的 delivery → 通知运营/客户「有新版本可续交付」
- 可选扩展（S3）：`delivery_items.follow_mode ∈ {frozen, follow}`，P1 仅 `frozen`

### 4.8 批量回灌

revision_of 边带相同 `run_id` + `algo_name@version`；可按 run_id 批量回滚、影响分析。`algo_runs` 一等实体 **P1.5+ 再做**。

### 4.9 全量审计契约（Audit Log）

**核心承诺：资产的任何变更，平台都必须留下可回放的审计日志。** 含 confidence 微调、tag 增删、pin/unpin、materialize、revision、归档、删除等所有写入。

#### 4.9.1 单一审计表：`asset_events`

| 性质 | 设计 |
|------|------|
| 表 | `asset_events`（已有；按月分区） |
| 写入特性 | **append-only**：禁止 UPDATE / DELETE（DB 权限 + AE2 兜底） |
| 单调 ID | `event_seq` BIGINT；全表单调；订阅者按 seq 推进 |
| 索引 | `(asset_id, event_seq)`、`(event_type, event_time)`、`(actor, event_time)`、`(idempotency_key)`、GIN(`payload->'changed_fields'`)、GIN(`system_metadata`) |
| 写入方 | 唯一通过 `AssetWriter`；缺 event → 事务回滚 + 422 |

#### 4.9.2 表结构（三段式）

```sql
asset_events (
  event_seq        BIGSERIAL PRIMARY KEY,        -- 全局单调
  asset_id         TEXT NOT NULL,
  event_type       TEXT NOT NULL,                -- 见 §4.9.3
  event_time       TIMESTAMPTZ NOT NULL,

  -- ① 顶层审计字段（高频查询，独立列 + 索引）
  actor            TEXT NOT NULL,                -- 严格命名约定见 §4.9.2.1（AV1 强约束）
  request_id       TEXT NOT NULL,                -- API 入口 X-Request-Id
  idempotency_key  TEXT NOT NULL,                -- Idempotency-Key（同 key 重试只一条 event）
  caller_ip        TEXT,

  -- ② system_metadata JSONB（变更上下文，对齐 DataHub SystemMetadata）
  system_metadata  JSONB NOT NULL DEFAULT '{}',
  -- 标准字段（可选）：
  --   run_id, pipeline_name, pipeline_version, algo_name, algo_version,
  --   registry_name (=databrew), registry_version, extra_properties

  -- ③ payload JSONB（业务 diff，只装变更内容）
  payload          JSONB NOT NULL DEFAULT '{}'
  -- 字段：changed_fields[], before{}, after{}
)
```

#### 4.9.2.1 actor 命名约定（AV1 强约束，rev.10）

`actor` 字段必须命中以下前缀之一，否则 Writer 拒绝（422）。约定让跨 actor 类型的审计 / 影响分析 / 合规反查可标准化执行。

```text
人 / 角色（无版本，user_id 即身份）:
  user:<user_id>              ← 通用人员
  ops:<user_id>               ← 运营
  reviewer:<user_id>          ← QA 评分员
  labeler:<user_id>           ← 标注员
  compliance_officer:<uid>    ← 合规

机器 / 流水线（强制 @<version>）:
  algo_sdk:<name>@<v>         ← 处理类算法（hand_track@2.0）→ asset_algo_latest algo_kind=processing
  qa_pipeline:<name>@<v>      ← 质检流水线 → asset_algo_latest algo_kind=qa
  split_pipeline:<name>@<v>   ← 切分流水线 → asset_algo_latest algo_kind=split
  rule_engine:<name>@<v>      ← 规则引擎
  llm:<model>@<v>             ← LLM 标签（gpt-4o@2024-08, claude-4@1.0）
  delivery_engine:<v>         ← 交付引擎（P1.5+）

系统 / 派生:
  system:<job_name>           ← 内部 cron / 自动化
  propagator:<handler_name>   ← ActionHandler 派生（如 propagator:TagPropagator）
```

**Lint 正则：** `^(user|ops|reviewer|labeler|compliance_officer):[^@]+$ | ^(algo_sdk|qa_pipeline|split_pipeline|rule_engine|llm|delivery_engine):[^@]+@[^@]+$ | ^(system|propagator):[^@]+$`

**为什么强制：** §11.5 lineage-search 按 actor / `system_metadata.algo_*` 反查影响时，actor 命名分裂会让查询无法聚合；统一前缀后 `WHERE actor LIKE 'algo_sdk:hand_track@%'` 一击中所有版本历史。

#### 4.9.3 事件类型清单

| 类别 | event_type |
|------|-----------|
| 资产生命周期 | `asset_created` / `asset_updated` / `asset_lifecycle_changed` / `asset_hard_deleted` |
| 物化 | `asset_materialized` |
| 版本 | `asset_revised` |
| 算法 | `algo_started` / `algo_finished` / `algo_failed` |
| 算法版本钉 | `asset_pinned` / `asset_unpinned` |
| 标签 | `tag_upserted` / `tag_deleted` |
| 指标 / 评分 | `metric_upserted` |
| 评估原档 | `eval_result_added` |
| Action 标注 | `action_upserted` / `action_deleted` |
| 交付 | `delivery_created` / `delivery_item_added` / `delivery_completed` |
| 数据集/训练 | `dataset_snapshot_created` / `training_run_started` / `training_run_finished` |
| Data Product（P1.5） | `data_product_created` / `data_product_updated` / `data_product_published` |
| 使用信号（P1.5）| `asset_viewed` / `asset_downloaded` / `asset_favorited` / `asset_unfavorited` |

#### 4.9.4 Event 完整结构（统一，所有事件类型）

**A 路由示例（原地：confidence 0.91 → 0.93）：**

```json
{
  "event_seq": 12345,
  "asset_id": "clipA_v2",
  "event_type": "algo_finished",
  "event_time": "2026-05-20T15:00:00Z",
  "actor": "algo_sdk:action_detector@2.0",
  "request_id": "req_7f8a...",
  "idempotency_key": "key_abc...",
  "caller_ip": "10.20.30.40",

  "system_metadata": {
    "run_id": "R789",
    "pipeline_name": "action_detection_pipeline",
    "pipeline_version": "v1.2.3",
    "algo_name": "action_detector",
    "algo_version": "2.0",
    "registry_name": "databrew",
    "registry_version": "rev.7",
    "extra_properties": {"triggered_by": "scheduled_cron"}
  },

  "payload": {
    "changed_fields": ["confidence"],
    "before": {"confidence": 0.91},
    "after":  {"confidence": 0.93}
  }
}
```

**B 路由示例（asset_revised：clipA_v1 → v2）：**

```json
{
  "event_seq": 12346,
  "asset_id": "clipA_v2",
  "event_type": "asset_revised",
  "event_time": "2026-05-20T16:00:00Z",
  "actor": "algo_sdk:hand_track@2.0",
  "request_id": "req_8g9b...",
  "idempotency_key": "key_def...",

  "system_metadata": {
    "run_id": "R789",
    "algo_name": "hand_track",
    "algo_version": "2.0",
    "revision_reason": {"type": "algo_rerun"}  // 与 asset_relations(revision_of).metadata 镜像
  },

  "payload": {
    "changed_fields": ["storage_uri", "revision"],
    "before": {
      "asset_id": "clipA_v1",
      "revision": 1,
      "storage_uri": "gs://.../r1-550e8400/video.mp4",
      "lifecycle_state": "superseded"
    },
    "after": {
      "asset_id": "clipA_v2",
      "revision": 2,
      "storage_uri": "gs://.../r2-f47ac10b/video.mp4",
      "lifecycle_state": "ready"
    }
  }
}
```

**B 路由 `payload.before` 只存关键识别字段**（asset_id / revision / storage_uri / lifecycle_state）。完整行状态通过 `before.asset_id` 查 `bronze_asset_mutations`（CDC 全量快照）。湖仓不需要 event payload 冗余全量字段。

#### 4.9.5 三段式职责（必读）

| 段 | 字段位置 | 职责 | 谁查 |
|----|---------|------|------|
| **顶层 4 字段** | 独立 PG 列 + 索引 | 「谁、什么时候、用什么链路」（高频审计）| 合规、运维 |
| **system_metadata** | JSONB 列 | 「这次变更属于哪个 run / 哪个 pipeline / 哪个算法」（变更上下文）| 算法回滚、影响分析 |
| **payload** | JSONB 列 | 「具体改了什么字段、改成什么」（业务 diff）| timeline UI、审计回放 |

设计依据：
- **DataHub `SystemMetadata` 对齐**：runId / lastObserved / properties 等都在 system_metadata，未来联邦零转换
- **`payload` 不再装 context**（之前在 payload.context 里的 run_id/algo_* 全部上提到 system_metadata）
- **顶层 4 字段不进 system_metadata**：actor/request_id/idempotency_key 是高频按 actor / 链路 / 幂等 key 查询的，列+索引比 JSONB 路径快几个数量级

#### 4.9.6 Writer 行为

- 事务内先读旧值 → 计算 `changed_fields` → 写 event；Writer 自动完成，调用方不手动填
- `changed_fields` 为空 → **不写** event（空更新视为 no-op）
- 创建类：`before=null, after={新行关键字段}, changed_fields=["__created__"]`
- 删除类：`before={删前关键字段}, after=null, changed_fields=["__deleted__"]`
- `system_metadata` 字段 Writer 按上下文自动注入（API 入口拿 request_id；algo_sdk 拿 run_id/algo_*）

##### 4.9.6.1 payload diff 策略（决策 D_HYBRID，rev.9）

字段级 diff 与 JSONB 整字段 diff 混合，平衡精度与 event 体积：

| 字段类型 | diff 方式 | payload 示例 |
|---------|---------|------------|
| **列字段**（lifecycle_state / confidence / revision / storage_uri 等强类型列）| 精准 before/after | `{"changed_fields":["confidence"], "before":{"confidence":0.91}, "after":{"confidence":0.93}}` |
| **JSONB 整字段**（`asset_governance.metadata` / `logical_assets.metadata` / `asset_relations.metadata` 等）| top-level key 算 changed；before/after **装整个 JSONB**（不再深入嵌套）| `{"changed_fields":["metadata"], "before":{"metadata":{...全量...}}, "after":{"metadata":{...全量...}}}` |
| **JSONB 内部嵌套字段** | **不**做深 diff（避免 JSON Patch 复杂度 + 上层 UI 解释成本） | 同上 |

**理由：**
- 列字段是上层 90% 关心的；精准 diff 让 timeline UI 直接读 changed_fields 渲染
- JSONB 体积通常 < 5KB（大于即视为设计问题，应提升为列），整存可接受
- > 50KB 的 JSONB（极端用户 metadata 滥用）触发 `large_jsonb_diff_warning` metric + 仅记录 changed_fields，before/after 留 `{"__truncated__":true}`，全量靠 CDC（`bronze_asset_mutations`）

##### 4.9.6.2 idempotency_key 生成（决策 I2，rev.9）

- **API 入口 actor**：client 必填 `Idempotency-Key` header（缺 → 400）
- **handler actor（projector / TagPropagator / DeliveryEligibilityProjector / RevisionNotifier 等）**：每次新生成 UUID（**不**走 deterministic key）
- **去重责任下放到下游表的 unique constraint**：
  - tag 类 → `asset_tags PK (asset_id, tag_key, tag_value, source, source_version)` 兜底
  - lifecycle 变更 → 状态机不允许重复变更（同状态再变 = no-op，不写 event）
  - delivery_ready 系统 tag → upsert 自带幂等
  - 仍写多条 event 是**可接受的代价**（event 是 append-only 流，多记 audit 不损害正确性，反而便于排查 handler 重试历史）
- **为什么不选 I1 deterministic key**：多 handler 共享同一 source event 时 key 冲突（A handler 触发 B handler，都用 `handler+source_seq` 派生 → 冲突）；I2 + 下游 constraint 更稳健

#### 4.9.7 审计不变式 AE1–AE5

| # | 规则 | 违反 |
|---|------|------|
| **AE1** | 任何写入业务表（`assets / asset_lineage / asset_content / asset_governance / asset_usage_stats / asset_tags / asset_algo_latest / actions / asset_metrics / asset_eval_results / asset_relations / deliveries / delivery_items / logical_assets`）必同事务写 `asset_events` | Writer 强制；缺则 422 |
| **AE2** | `asset_events` append-only：禁止 UPDATE / DELETE | DB 权限 GRANT 限制 |
| **AE3** | 每条 event 必填顶层 4 字段（`actor / request_id / idempotency_key / event_time`） | Writer 强制；缺则 422 |
| **AE4** | 字段更新类 event 必含 `payload.before / after / changed_fields` | Writer 强制 |
| **AE5** | hard-delete 资产前必须写 `asset_hard_deleted` event 含最后快照（`payload.before={final_snapshot}`） | retention job 兜底 |
| **AE6**（rev.10）| `algo_finished` event 必填 `asset_algo_latest.run_inputs`（含 params + input_asset_ids + code_commit + image_digest）；缺则 422 | Writer 强制 |
| **AV1**（rev.10）| `asset_events.actor` 必须命中 §4.9.2.1 命名约定正则；机器类必带 `@<version>` | Writer 强制 |

> 注：`system_metadata.run_id` **可选**，不强制（人工操作场景天然无 run_id）。Writer 能填就填，不强约束。
> 注：`run_inputs` 强制 (AE6) 只针对 `algo_finished`；人工操作 / 系统 cron 无需。

#### 4.9.8 湖仓消费路径（P2）

```text
PG assets / asset_*  ──CDC──► bronze_asset_mutations（Iceberg，全量行快照）
PG asset_events      ──CDC──► bronze_asset_events  （append-only event 流）
                                    ↓ join + enrich
                          silver_asset_events（统一 event 视图）
                                    ↓
       分析 SQL（按 actor / run_id / changed_field 维度聚合）
```

湖仓端示例查询：

```sql
-- 按 run_id 找一次算法回灌影响的所有资产
SELECT DISTINCT asset_id
FROM silver_asset_events
WHERE system_metadata['run_id']::text = 'R789';

-- 按 actor 找某算法跑过的所有资产
SELECT asset_id, event_time, payload['changed_fields'] AS fields
FROM silver_asset_events
WHERE actor LIKE 'algo_sdk:hand_track%';
```

#### 4.9.9 覆盖矩阵

| 变更 | 路由 | 新 asset_id | event_type | system_metadata 关键字段 | payload diff |
|------|------|-----------|------------|----------------------|------|
| confidence 0.91 → 0.93 | A | 否 | `algo_finished` | run_id, algo_* | ✓ |
| status ok → failed | A | 否 | `algo_failed` | run_id, algo_* | ✓ |
| 加/删 tag（人工） | A | 否 | `tag_upserted / tag_deleted` | — | ✓ |
| 加/删 tag（规则） | A | 否 | `tag_upserted` | run_id, rule_* | ✓ |
| algo pin / unpin | A | 否 | `asset_pinned / asset_unpinned` | — | ✓ |
| 人工评分 | A | 否 | `metric_upserted` | — | ✓ |
| 自由 review | A | 否 | `eval_result_added` | — | ✓ |
| 改 metadata 任意字段 | A | 否 | `asset_updated` | — | ✓ |
| materialize（virtual→materialized）| A | 否 | `asset_materialized` | run_id?, algo_*? | ✓ |
| 重切 mp4（storage_uri 变）| B | ✓ | `asset_revised` + `asset_created` | run_id, algo_*（revision_reason 镜像 asset_relations 边）| ✓ |
| 时间窗 / label 变 | B | ✓ | `asset_revised` | run_id, algo_* | ✓ |
| supersede / 归档 / 软删 | A | 否 | `asset_lifecycle_changed` | — | ✓ |
| hard-delete | — | 行没了 | `asset_hard_deleted`（含最后快照）| — | ✓ |

→ **100% 覆盖**；任意 `asset_id` timeline 可重建「谁在何时把什么字段从 X 改成 Y、属于哪个 run」。

#### 4.9.10 读取入口

| API | 用途 |
|-----|------|
| `GET /assets/{id}/provenance` `timeline[]` | 详情页变更时间线（默认 100 条）|
| `GET /assets/{id}/events?since=<seq>&limit=N` | SDK 增量订阅 |
| `GET /audit/search?actor=&time_range=&event_type=&changed_field=&run_id=` | 跨资产合规查询（P1.5）|
| Lakehouse `silver_asset_events` | 大规模分析（P2）|

#### 4.9.11 性能与存储

| 维度 | 估算 |
|------|------|
| 单条 event | ~1.5KB（含顶层 + system_metadata + payload）|
| 单资产年事件数 | 100–500 |
| 现网 30 万资产 × 200/年 | 6000 万行/年 ≈ 90GB |
| 分区 | 按月；在线 12 个月 → 归档 cold |
| 查询「该资产时间线」| `WHERE asset_id=X ORDER BY event_seq` | < 50ms |
| 查询「按 actor 查」| `WHERE actor='algo_sdk:hand_track@2.0'` | < 100ms（独立列+索引）|
| 查询「按 run_id 查」| `WHERE system_metadata @> '{"run_id":"R789"}'` | < 200ms（GIN）|

### 4.10 资产寻址：HTTP URL（rev.11 决策，砍 `databrew://`）

> **rev.9 曾设计 `databrew://` 自定义 URI scheme**。**rev.11 砍掉**：单实例不需要 namespace；HTTP URL 已能表达所有用例；自定义 scheme 引入 parser / serializer / SDK helper / 鉴权 / 文档维护成本但零业务收益。

资产两种寻址方式：

| 层 | 形式 | 用途 |
|----|------|------|
| **物理层** | `gs://<bucket>/assets/<logical_id>/r<n>-<uuid4>/<filename>` | GCS 实际存储，由 `asset_content.storage_uri` 持有 |
| **逻辑层（HTTP URL）** | `GET /api/v1/assets/{asset_id}` 或 `GET /api/v1/logical-assets/{logical_id}/current` | 客户 / SDK / 交付 manifest 引用 |

**两种 API 用法：**

```text
# 指定版本（delivery manifest 锁版用）
GET /api/v1/assets/clipA_v3
→ {
    "asset_id": "clipA_v3",
    "logical_asset_id": "clipA_v1",
    "revision": 3,
    "materialization": "materialized",
    "storage_uri": "gs://cyb-prod/assets/clipA_v1/r3-f47ac10b/video.mp4",
    "signed_url": "https://storage.googleapis.com/..."     ← 有效期 1h
  }

# 跟随当前版本（SDK 引用最新输入用）
GET /api/v1/logical-assets/clipA_v1/current
→ 同上结构，自动返回 is_current=true 的那一版
```

**使用场景：**

| 场景 | 用哪个 |
|------|------|
| delivery manifest（交付客户）| `/api/v1/assets/{asset_id}` 锁具体版本 |
| 算法 SDK 引用当前版本输入 | `/api/v1/logical-assets/{logical_id}/current` |
| 交付 changelog 通知新版 | `/api/v1/assets/{new_asset_id}`（带新 revision） |
| 内部系统资产引用 | `/api/v1/assets/{asset_id}` |

**约束：**

- `/current` 内部等价 `SELECT asset_id FROM assets WHERE logical_asset_id=? AND is_current=true AND is_deleted=false`（走 partial unique index `uq_assets_current_per_logical`）
- 不支持相对引用（`?revision=-1`）：避免语义歧义
- virtual 资产返 `materialization=virtual` + metadata（无 storage_uri / signed_url）
- 标准 HTTP 鉴权（Bearer token），无新机制
- **不引入自定义 URI scheme**：未来如需联邦（接 DataHub）再考虑 URN，当下 HTTP URL 足够

---

### 4.11 Actions Framework（事件订阅自动化）

> **设计来源：** DataHub Actions Framework 概念，**PG-based 实现**（不引入 Kafka）。

#### 4.11.1 解决什么问题

资产任何变更都会写 `asset_events`。但每个 event 通常需要触发**多个副作用**（reindex / 通知 / 投影更新 / 缓存失效）。**Actions Framework** 把所有「event → 副作用」抽象为统一的 Handler 接口，避免 6+ 个 projector 各写一套订阅 / watermark / retry / DLQ 代码。

#### 4.11.2 接口设计

```go
type ActionHandler interface {
    Name() string                         // 唯一标识，如 "ParentReindexer"
    Triggers() []EventType                // 订阅的 event 类型
    Handle(ctx context.Context, event AssetEvent) error
}

type ActionRunner struct {
    handlers   []ActionHandler
    pgEvents   EventRepo
    checkpoint CheckpointRepo             // per-handler watermark
    dlq        DLQRepo                    // 死信表
}
```

ActionRunner 共享逻辑：
- 每个 handler 一个 goroutine，独立 watermark（`action_checkpoints` 表，per-handler `last_event_seq`）
- 按 `event_seq` 顺序拉取（一次最多 N 条 = batch）
- **Polling 策略：自适应退避（决策 P2，rev.9）**
  - 拉到 batch（≥ 1 条）→ 立即下一轮拉取（连续追赶，无间隔）
  - 拉到 empty → 退避：`1s → 2s → 5s`（上限 5s）
  - 再次拉到非空 → 重置到 1s
  - 收益：忙时近实时（< 1s 延迟）；闲时 PG 负载 ≈ 8 handler × 0.2 qps = 1.6 qps，可忽略
- Handler 失败 → 指数退避重试（max 5 次）
- 仍失败 → 写 `action_dlq` 表，人工介入
- 优雅停止 / 重启续传

> P3 LISTEN/NOTIFY 与 P4 dispatcher fan-out 留作 P2 优化备选（当 handler 数超 20 个或 P99 延迟要 < 200ms 时再上）。

#### 4.11.3 P1 初始 Handler 清单（6 个）

| Handler | 订阅 | 副作用 |
|---------|------|------|
| **SearchReindexer** | `asset_created / asset_updated / asset_revised / asset_materialized` | 重建 ES 文档（取代现有 `search_reindex_jobs` 散写逻辑）|
| **ParentReindexer** | `action_upserted / action_deleted / asset_created`（子资产新建时） | 父 segment ES doc 的 `child_*[]` 摘要重建（D1 denormalize）|
| **RevisionNotifier** | `asset_revised` | 找引用旧版的 delivery_items → 发飞书/邮件通知运营/客户「有新版」|
| **AlgoVersionsProjector** | `algo_finished / algo_failed` | 维护 ES `algo_versions_seen` 字段 |
| **DeliveryEligibilityProjector** | `metric_upserted / tag_upserted / algo_finished` | 重评估 delivery_rules，自动打/撤 `delivery_ready` 系统 tag（P1.5）|
| **TagPropagator**（P1.5） | `tag_upserted / tag_deleted / asset_created` | 沿血缘自动传播 `propagation=descendants` 的 tag（§7.6.1，借鉴 Atlas Classification Propagation）|
| **UsageStatsAccumulator**（P1.5） | `asset_viewed / asset_downloaded / asset_favorited` | 批量累加到 `asset_usage_stats`（view/download/favorited_count）；每天重算 `trending_score` 投影到 ES `discovery_signals` |
| **OpenLineageEmitter** | `asset_created / asset_revised / algo_finished` | **占位 Handler，P1 启用 = false**；P2 真要导出血缘到 Marquez/DataHub 时启用 |

#### 4.11.4 与现有 `search_reindex_jobs` 的关系

现有 `search_reindex_jobs` 表是「手写的特殊 reindex action」。P1 改造：
- `SearchReindexer` ActionHandler 取代散写逻辑
- `search_reindex_jobs` 表保留作为 retry 视图（Handler DLQ + 现有 reindex 状态合并）
- 兼容老 admin reindex API（手动触发全量重建）

#### 4.11.5 事件源可插拔设计（PG 起步，未来可切 GCP Pub/Sub）

P1/P1.5 **PG events polling 是足够的**；但 Handler 接口设计要让事件源未来可平滑切换（DataBrew 已有 GCP Pub/Sub 基础设施，P2 可作为可选演进方向）：

```go
type EventSource interface {
    Subscribe(handlerName string, eventTypes []EventType) <-chan AssetEvent
    Ack(handlerName string, eventSeq int64) error
}

// 实现 1：P1 默认
type PGEventSource struct { ... }    // 基于 asset_events 表 + action_checkpoints watermark

// 实现 2：P2 可选
type PubSubEventSource struct { ... } // 基于 GCP Pub/Sub topic + subscription
```

ActionRunner 只依赖 `EventSource` 接口，**切换事件源对 Handler 零侵入**。

**P1/P1.5 vs P2 对比：**

| 维度 | P1/P1.5（PG polling）| P2 可选（GCP Pub/Sub）|
|------|-------------------|--------------------|
| 事件源 | PG `asset_events` 表 | Pub/Sub topic（PG outbox → Pub/Sub publisher）|
| 订阅 | polling + `action_checkpoints` watermark | Pub/Sub subscription + ack |
| 部署 | 零新组件 | 已有 GCP 基础设施，零新部署 |
| 延迟 | 1–3 秒 | 100ms–1s |
| 跨集群 / 跨语言订阅 | ✗ | ✓（GCP Pub/Sub 天然支持） |
| 适用场景 | 单 backend、内部使用 | 多 backend / 算法 SDK 独立服务 / 客户消费血缘 |

**对比 DataHub Kafka**：我们不引入 Kafka 集群（Zookeeper / Schema Registry 等运维成本），P2 选 Pub/Sub 是 **零新部署**（GCP 现有），与 Kafka 在「订阅 + ack」语义上等价。

**演进路径（不必现在做）：**

1. **P1/P1.5**：PG events 表 + polling，开发简单、零运维
2. **P2 触发点**：出现「跨服务订阅」（如算法 SDK 独立服务想订阅 events）/ 「秒级延迟不够」/ 「跨集群」需求时 → 加 `PubSubEventSource` 实现
3. **过渡期**：两套 EventSource 并存，新 Handler 走 Pub/Sub，老 Handler 留 PG polling，渐进迁移

#### 4.11.6 新表

```sql
action_checkpoints (
  handler_name        TEXT PRIMARY KEY,
  last_event_seq      BIGINT NOT NULL,
  last_processed_at   TIMESTAMPTZ NOT NULL,
  updated_at          TIMESTAMPTZ NOT NULL
);

action_dlq (
  id              UUID PRIMARY KEY,
  handler_name    TEXT NOT NULL,
  event_seq       BIGINT NOT NULL,
  asset_id        TEXT NOT NULL,
  event_type      TEXT NOT NULL,
  error_message   TEXT NOT NULL,
  retry_count     INT NOT NULL,
  first_failed_at TIMESTAMPTZ NOT NULL,
  last_failed_at  TIMESTAMPTZ NOT NULL,
  resolved_at     TIMESTAMPTZ
);
```

---

## 5. 横向 + 纵向血缘的统一视图


| 维度             | 字段 / 机制                                                                              | 何时必填  |
| -------------- | ------------------------------------------------------------------------------------ | ----- |
| 结构横向（怎么切出来的）   | `parent_asset_id`, offsets, `split_`*, `relation_type`                               | 有父时   |
| 处理横向（后来跑过什么）   | `asset_algo_latest`（含 pin）, `algo_*` events, `algo_versions_seen`                    | 跑算法时  |
| 版本纵向（同逻辑实体的迭代） | `logical_asset_id`, `revision`, `is_current`, `revision_of` 边, `asset_revised` event | 走 B 时 |


三者解耦：人工再切 clip 可以没有 `split_algo_*`；virtual 资产可以没有 `asset_algo_latest`；首版没有 `revision_of`。

---

## 6. Lifecycle、删除与归档

### 6.1 决策

- **决策 A**：不以 `lifecycle_state` 状态机作为交付硬门槛
- lifecycle 简化为 `ready / archived / superseded` 三态；过渡态（processing/failed/rejected）迁到 `asset_algo_latest.status` + events
- 删除/归档/supersede 严格区分

### 6.2 状态分类


| 状态           | 含义           | 谁触发              | PG 行                | GCS 文件         | 可检索                         | 进新交付 |
| ------------ | ------------ | ---------------- | ------------------- | -------------- | --------------------------- | ---- |
| 正常 ready     | 当前可用         | —                | ✓                   | ✓              | ✓                           | ✓    |
| superseded   | 被新版替代        | revision B 自动    | ✓                   | ✓ 永久           | 默认隐藏，`include_revisions` 可见 | ✗    |
| archived     | 长期不用，冷藏可恢复   | 运营 / retention   | ✓                   | ✓ cold storage | 默认隐藏                        | ✗    |
| soft-deleted | 标删除可恢复 ≤ N 天 | 运营               | ✓ `is_deleted=true` | ✓ 保留           | ✗                           | ✗    |
| hard-deleted | 物理删除         | retention / GDPR | ✗                   | ✗              | ✗                           | ✗    |


### 6.3 约束（R1–R8）

| #   | 规则                                                          | 违反   |
| --- | ----------------------------------------------------------- | ---- |
| R1  | 已被 `delivery_items` 引用的资产禁止直接 delete；需先「下架交付 → 解除引用 → 再删」   | 422  |
| R2  | superseded 不能 soft-delete（保留版本链审计）                          | 422  |
| R3  | hard-delete 必须先 soft-delete + N 天观察                         | 422  |
| R4  | hard-delete 前必须撤掉所有 `delivery_items` / `asset_relations` 引用 | 级联检查 |
| R5  | `is_current=true` 资产不能直接 archived（必须先 supersede 或 delete）   | 422  |
| R6  | archived ↔ ready 可双向；soft-deleted 保留期内可恢复                   | OK   |
| **R7** | 当前算法被 `is_pinned=true` 锁定的资产**禁止** supersede（B 路由 422）；需先 unpin | 422 |
| **R8** | `archived / superseded / soft-deleted` 资产**不接收**新 tag（含 TagPropagator 派生写入）；冻结视图 | 422（直接写）/ propagator 静默丢弃 + metric |

> **R7 用例：** 算法合规冻结某版本后，禁止任何重跑产物覆盖；必须显式 unpin。
> **R8 用例：** archived 资产不应被新 PII 标签触发回滚审核；propagator 静默跳过并上报 `propagation_skipped_frozen` metric。

### 6.4 Retention（Registry 可配，P1 仅声明）

```yaml
clip:
  retention_default_days: 365
  on_expire: archive_then_delete_after_180d
action:
  retention_default_days: unlimited       # 元数据轻
task:
  retention_default_days: unlimited
frame:
  retention_default_days: 90
  on_expire: archive_then_delete_after_90d
```

> **P1 行为（rev.10 决策）：** `retention_default_days` 仅写入 `asset_governance.expire_at`，**不启动**自动 archive/delete cron job。运营手动触发或等 P1.5 `RetentionJob` ActionHandler 上线。理由：自动删除是不可逆操作，P1 阶段优先保留人工 review；先观察 expire_at 命中情况再决定 SLA。

---

## 7. 标签体系（已知 5 类，可扩展 N 类）

> **设计原则**：5 类只是当下已知来源，**模型必须支持持续新增**（LLM 自动标签、第三方供应商、众包标注、合规审查、客户反馈、A/B 实验产物等）。新增类**不动 schema**，仅在 **Tag Source Registry** 注册 `source` 标识 + 命名空间 + 写入权限 + value 约束即可。

### 7.0 术语澄清：tag vs label vs annotation

代码与文档历史上 `tag` / `label` 混用。本 PRD 与后续 ADR / code review 统一规则：

| 词 | 语义 | 存储 | 例 |
|----|------|------|-----|
| **tag** | 贴在 **asset** 上、用于**目录 / 检索 / 治理 / 资格规则** 的 key-value 标记 | `asset_tags`（含 `source/source_name/source_version`） | `scenario=kitchen`, `customer=cust_X`, `quality=ok`, `algo.hand_track.version=2.0` |
| **label** | **action**（或未来 frame bbox）实体**内部**的**语义分类字段**，带定位 | `actions.primary_label / labels[]` | `primary_label=grasp`, `labels=[grasp, manipulation]` |
| **annotation** | label 的「带位置 / 带时间窗」版本，泛指；与 action 通用 | 同 `actions` | 时间窗 + label + source |

**用词规则**：

| 场景 | 使用 | 不使用 |
|------|------|--------|
| 资产级 facet / 治理 / 检索 / 交付资格 | **tag** | label |
| action 时间窗的分类 | **action label**（`primary_label / labels[]`） | tag |
| 泛指「打在数据上的分类信息」 | **「label 信息（tag 或 action label）」** | 单独「label」 |
| ES 文档字段 | `tags[]` ← `asset_tags`；`actions[]` ← `actions`（含 label 字段） | 同字段混并 |
| 「打标签」操作 | 资产级：**「打 tag」**；action 级：**「写 action label」** | 「打 label」 |

**底层数据流：**

```text
asset_tags（tag）         ──► ES tags[]            ← 治理 / 检索 / 交付资格
actions.primary_label    ──► ES actions[].label    ← 行为 / 标注 / 含某动作筛选
asset_algo_latest        ──► ES algos_current[]    ← 算法身份强类型字段（status/version/pin/finished_at）
   └─ 同事务双写 ─►  asset_tags(source=algo_sdk, key=algo.<name>.version, value=<v>)  ──► 投 ES tags[]
                                                   ← 决策 T3：双视图，asset_algo_latest 是强类型真源，
                                                     asset_tags 提供统一 facet（filter 一套语法）
```

三者在 ES 端**并列 facet**，filter 语法一致但路径不同；UI 按场景选合适的 facet。
algo_sdk **双写**保证：「algo.* 既能按状态字段强类型查（pinned / failed / 跑过哪些版本），也能在统一 tags[] facet 里 filter」。事务原子性由 AssetWriter 兜底。

> 旧文档若提及「类型 b 算法结果标签」实际指 **`actions.label`** 而非 `asset_tags`；rev.5 起统一表述为「**action label**」，仅在 ES 端可作为「tag-like facet」呈现。

### 7.0.1 决策树：何时用 tag，何时用 label

> **是否统一为一个词？不统一**。两者解决的不是一回事，硬合反而更乱。

**一句话区分：**

- **tag** = 贴在资产上的**外部分类标记**（附加信息，资产删了 tag 没了）
- **label** = 实体本身的**核心分类字段**（本质属性，没 label 实体不成立）

**决策树（新增字段时套）：**

```
新需求：要给数据加分类信息 X
│
├─ X 是 action 这种实体的「核心属性」？
│   （没有 X，实体不成立 / 实体定义就是为了表达 X）
│   ├─ 是 → label（写进 actions.primary_label / labels[]）
│   │
│   └─ 否 ↓
│
├─ X 是「跨实体类型」的分类（segment/clip/action/frame/task 都能打）？
│   ├─ 是 → tag（写进 asset_tags）
│   │
│   └─ 否 ↓
│
├─ X 是「治理 / 检索 / 交付资格」用的 facet？
│   ├─ 是 → tag
│   │
│   └─ 否 ↓
│
└─ X 是「带时间窗 / 带定位的语义事件」？
    └─ 新建一个 asset_type 或扩展 action，用 label 字段
```

**三个直觉判别：**

1. **「资产挂了多少 X」可以 0 到 N** → **tag**
   - clip 可以打 0 个或 20 个 tag 都行
   - action 必须有 `primary_label`，不然没意义 → label
2. **「X 是不是这个表的列定义」**：是列 → label；走 K-V 表 → tag
3. **「X 跨多种 asset_type」**：跨 → tag；只对某一类有意义 → 那一类的 aspect 表 label 字段

### 7.0.2 边界场景对照（容易搞错的）

| 场景 | 选 | 理由 |
|------|----|------|
| 这条 segment 是厨房场景 | **tag** `scenario=kitchen` | 资产 facet |
| 这条 segment 客户 cust_X 拥有 | **tag** `customer=cust_X` | 治理 |
| 这条 segment 已被 hand_track v2.0 ok | **tag** `algo.hand_track.version=2.0`（algo_sdk 投影） | 跨实体可筛选 |
| 这条 segment 通过质量门槛 | **tag** `quality=ok`（rule 派生） | facet，无定位 |
| **这个 action**（t=1000–2000）是 grasp 动作 | **label** `actions.primary_label=grasp` | 实体核心字段 |
| 这个 action 既是 grasp 也是 manipulation | **label** `actions.labels=[grasp, manipulation]` | 同上 |
| 这个 task 是 sweep_floor 任务 | **label** `task.task_kind=sweep_floor`（task 的 label 字段） | 实体核心属性 |
| 这个 task 客户优先级高 | **tag** `customer.cust_X.priority=high` | 治理 facet |
| 这条 segment 包含「危险动作」要审核 | **tag** `compliance.review_required=true` | 治理 facet |
| 算法 `action_detector` 在 segment 上跑出一段时间窗 + label=grasp | **新建 action 实体**，写 `actions.primary_label`（label） | 带定位的语义事件 |
| LLM 给整个 clip 生成 summary 长文本 | **不是 tag 也不是 label** → `asset_content.summary_text` 或 `asset_governance.metadata` | 不是分类，是描述 |
| 客户标记「这条不要交付」 | **tag** `customer.cust_X.exclude=true` | 治理 facet |
| 标注员标这个 clip 内有 3 个 grasp 动作 | **创建 3 个 action 实体**（label=grasp） | 每个 action 是独立实体 |
| 标注员标整个 clip「整体动作是 cooking」 | **tag** `scenario=cooking` 或 `asset_content.summary_text` | 资产级，无时间窗 |
| 算法人员给资产打分 4.5（数值） | **metric** `rating.quality_score=4.5`（`asset_metrics`，source=human）| 数值、可聚合、多人多次 |
| 算法人员附带自由文本 review | `asset_eval_results`（payload 引同一 reviewer_id） | 完整原档，不进 ES facet |
| 评分简单分类（通过 / 不通过） | **tag** `review.verdict=passed`（source=human） | 离散枚举，不需要数值聚合 |

### 7.0.3 为什么不能强行统一（反例）

| 假设统一为 tag | 出现的问题 |
|---------------|------------|
| `actions.primary_label=grasp` 改成 `tag(action_id, primary_label=grasp)` | action 核心字段变可有可无 K-V；查询 5000 万 action 性能崩；建模脱离实体直觉 |
| `actions.labels[]` 拆成 N 行 tag | 同 action 多 label 排序丢失；事务一致性弱 |

| 假设统一为 label | 出现的问题 |
|----------------|------------|
| `asset_tags(scenario=kitchen)` 改成 `assets.labels=[...]` | assets 表退化成 JSONB；治理/检索/资格规则全靠扫主表；丢失 `source/source_version` 多源信息 |
| 5 类 + 未来 N 类标签来源都塞进同一字段 | §7.6 Tag Source Registry 失效 |

### 7.0.4 衍生规则（lint / code review）

| 规则 | 强制度 |
|------|--------|
| `asset_tags.tag_key` 不允许等于 `label / labels / primary_label` | Lint 阻断（避免歧义） |
| `actions.primary_label / labels[]` 的值不再写一份进 `asset_tags`（单一真源） | code review |
| ES 端可同时投影 `tags[]` 和 `actions[].label`，但绝不放同一字段 | SearchDocumentBuilder 强约束 |
| UI 文案：资产级筛选称「按 tag」；含某动作筛选称「按 action label」 | 前端 PR |
| 文档/PR/注释里说「label」必须明确是 action label 还是 tag | code review |

---

### 7.1 当前已知 5 类标签来源（非穷尽）


| 类           | 含义                       | 例                                       | 存放                                                                   |
| ----------- | ------------------------ | --------------------------------------- | -------------------------------------------------------------------- |
| **a 算法身份**  | 资产被某算法某版本处理过             | `algo.hand_track@2.0 ok`                | `asset_algo_latest` + events + ES `algos_current/algo_versions_seen` |
| **b 算法结果**  | 算法跑出的结构化结果（带时间窗 + label） | `(t=1000–2000, label=grasp, conf=0.91)` | `actions` aspect（K1：一版一行）                                            |
| **c 规则派生**  | 规则引擎从 a/b/d/e 计算         | `quality:ok (rule=quality_check@1.0)`   | `asset_tags`（source=rule）                                            |
| **d 人工标注**  | 标注员/运营手动打                | `scenario:kitchen`                      | `asset_tags`（source=human）                                           |
| **e 业务/客户** | 运营或系统配置                  | `customer:cust_X`, `purpose:training`   | `asset_tags`（source=system）                                          |


### 7.2 多版本结果并存

- **X 并存**：同一算法多版本结果同时存在；按 `source_version` 区分
- **K1 一版一行**：`actions` 每个 (asset, source_name, source_version, label, time_window) 独立 `action_id`；走 §4 revision_of 串联
- **检索流程**：先按 label 搜资产 → 列出多版本候选 → 按 version 命中

### 7.3 `asset_tags` 表结构

```sql
asset_tags (
  asset_id        TEXT,
  tag_key         TEXT,
  tag_value       TEXT,
  source          TEXT,                -- 'human' | 'rule' | 'system'
  source_name     TEXT,                -- rule=规则名；human=user_id；system=配置源
  source_version  TEXT,                -- rule 才填
  applied_at      TIMESTAMPTZ,
  PRIMARY KEY (asset_id, tag_key, tag_value, source, COALESCE(source_version,''))
)
```

**多源共存**：人标 `quality:ok` 与规则标 `quality:ok` 可并存（两行，source 不同）；UI 显示「信任度叠加」。

**规则重算**：`DELETE FROM asset_tags WHERE source='rule' AND source_name=X AND source_version=V` 后由规则引擎重写。

### 7.4 算法结果 / 评估 / 指标 三表分工


| 表                    | 用途                                            | 何时写             |
| -------------------- | --------------------------------------------- | --------------- |
| `actions`            | 业务时间窗 + label（b 类）；asset_type=action 的 aspect | 算法/标注产**离散事件**  |
| `asset_metrics`      | 标量指标投影（K/V/source/version）                    | 算法/评估产**可聚合数值** |
| `asset_eval_results` | 一次评估 run 的完整原始 payload（不进 ES）                 | 评估流水线 run 结束    |


约束：

- `actions` 不存「整段评估指标」（如 mAP）；那进 `asset_metrics`
- `asset_metrics` 不存「带时间窗的离散事件」；那进 `actions`
- `asset_eval_results` 是审计/回放原档；UI 筛选不查它

#### 7.4.1 人工评分（算法人员对资产打分）

**决策：复用 `asset_metrics`（方案 A）+ 保留全部历史（R1）+ 跟随 revision（R4）**。不引入新表。

**核心约定：**

| 项 | 值 |
|----|----|
| 表 | `asset_metrics`（已有，扩 source 维度） |
| 命名空间 | `metric_key` 用前缀 `rating.*`（如 `rating.quality_score`、`rating.action_completeness`、`rating.label_correctness`） |
| 来源 | `source = human`；`source_name = reviewer_user_id` |
| 多次评 | 保留全部历史；同人多次评 = 多行（`scored_at` 区分） |
| 多人评 | 自然多行；ES 端聚合 |
| 与 revision | 评分挂在具体 `asset_id` 上；v1 评分不跟到 v2；可按 `logical_asset_id` 跨版本对比 |
| 自由文本 review | 进 `asset_eval_results` 的 payload，引用同 `reviewer_user_id` |

**数据示例：**

```text
asset_metrics 行
  asset_id        clipA_v2
  metric_key      rating.quality_score
  metric_value    4.5
  source          human                ← §7.6 已支持
  source_name     alg_zhang            ← reviewer_user_id
  source_version  NULL                 ← 人工评不带版本
  scored_at       2026-05-20T15:00:00Z
  metadata        {"dimension":"overall","comment":"夹爪动作不连贯"}
```

**ES 投影（D1 已有 `metrics[]`，新增聚合 facet）：**

```yaml
metrics:
  - {key: rating.quality_score, value: 4.5, source: human, source_name: alg_zhang, scored_at: ...}
  - {key: rating.quality_score, value: 4.0, source: human, source_name: alg_li,    scored_at: ...}
metrics_aggregates:                           # 新增
  rating.quality_score:
    avg: 4.25
    median: 4.25
    count: 2
    distinct_reviewers: 2
    last_scored_at: ...
```

**支持的查询：**

- 「rating.quality_score 平均 ≥ 4 且至少 2 人评过」→ `metrics_aggregates.rating.quality_score.avg >= 4 AND distinct_reviewers >= 2`
- 「alg_zhang 评过的所有资产」→ `metrics.source=human AND source_name=alg_zhang`
- 「v1 评 4.5，v2 评了多少」→ 按 `logical_asset_id` 聚合，对比各 `revision`
- 「最近一周被评过」→ `metrics.last_scored_at >= now - 7d`

**Validator / 治理：**

- `rating.*` 前缀进 §9.1 tag_registry（写入权限 `writable_by: [reviewer, algo_engineer]`）
- 评分值范围与维度由 `metric_registry`（评估文档约定）；超范围 422
- `source=human` 时 `source_name` 必填（对齐 §7.6 `requires_source_name`）

**衍生场景：**

- 评分驱动交付：`delivery_rules` 写 `metrics.rating.quality_score.avg < 3 → block`
- 评分驱动重产：projector 监听 `metric_upserted` + 阈值 → 触发 `algo_runs` 重产（P1.5）

**Rating 跨 revision 作用范围（决策 C1，rev.9）：**

| 维度 | 决策 |
|------|------|
| `delivery_rules` 中 `metrics.rating.*` 默认 scope | **仅当前 is_current 版本**（不跨 logical 聚合） |
| v1 评 4.5、v2 未评 → 走 delivery_rules `avg >= 4` | **失败**（v2 无 ratings → avg=NULL → 视为不通过）→ 阻交付 |
| 业务诉求 | **新版本必须重评**（B 路由通常 = 算法升级或人工 rework，旧分不能背书新产物）|
| ES 投影 | `metrics_aggregates` 只汇总当前版的 ratings；跨版本对比（v1 vs v2 评分趋势）走 `GET /logical-assets/{id}/ratings-history` 专门 API（P1.5）|
| 不做 C2 跨 logical 聚合的原因 | 算法或时间窗变了，旧评分语义不再适用；自动继承 = 隐患（合规风险）|
| 不做 C3 可配 scope 的原因 | delivery_rules 语义复杂度膨胀；运营不应自由切换「评分继承不继承」|
| 不做 C4 自动迁移的原因 | 把人工评分跨版本搬运是「数据捏造」；不留 audit trail 隐患 |

**「未评导致阻交付」的实际工作流：**

```text
v1 ─ ratings: alg_zhang=4.5, alg_li=4.0  → avg=4.25 → delivery 通过
   │
   └─ B 路由生 v2（算法 hand_track@2.0 升级）
         │
         ├─ 老 v1 supersede（仍保留 ratings 历史，可查）
         └─ v2 无 ratings → delivery_rules `avg>=4` 失败
                │
                └─ 运营见状态: 「rating 缺失」→ 派单给 alg_zhang 重评
                     │
                     └─ v2 评分 ≥ 4 → delivery 放行
```

→ **「重评」是 B 路由的强制门槛**，倒逼算法升级有质量验证，符合 §1 业务承诺。

### 7.5 「当前」与「历史」选

- `asset_algo_latest` 默认隐式 = 最后 finish 那版
- 显式 pin 可冻结当前 facet；events 与 `algo_versions_seen` 仍累积
- 时间线一览（资产 × 算法）：provenance `timeline[]` 按时间排序展示 `algo_started / algo_finished / asset_pinned / asset_unpinned`

### 7.6 标签来源开放扩展模型（Tag Source Registry）

**核心契约：** `asset_tags.source` 是开放字符串，不在 PG CHECK 限定；治理由 **Tag Source Registry** 承担。新增一类标签来源 = registry 加一行 + 可选 namespace 前缀，**完全不动 schema 和 ES mapping**。

**Registry 结构（与 §9.1 tag_registry 合并管理）：**

```yaml
tag_sources:
  # 已知 5 类
  - source: algo_sdk            # a 算法身份（由 algo_sdk 写）
    description: 算法运行身份标签
    prefix: algo.
    writable_by: [algo_sdk]
    immutable: true             # 不允许人工改/删
  - source: rule_engine         # c 规则派生
    prefix: rule.
    writable_by: [rule_engine]
    requires_source_version: true
    recomputable: true          # 规则重算时可批量重建
  - source: human               # d 人工标注
    writable_by: [labeler, ops]
    requires_source_name: true  # source_name = user_id
  - source: system              # e 业务/客户
    writable_by: [ops]
  - source: action_aspect       # b 算法结果（投影自 actions，仅作 ES facet）
    writable_by: [system]
    projection_only: true       # 不直接写 asset_tags，由 SearchDocumentBuilder 派生

  # 注：algo_sdk 的 algo.* 走双写（决策 T3，rev.9）——
  # asset_algo_latest 是强类型真源（status/version/pin/finished_at），
  # asset_tags 同事务再写一行 algo.<name>.version=<v>，提供统一 facet。
  # 不走 projection_only：避免 ES 端做复杂 join 才能拼出 algo.* tag。

  # ↓ 未来新增示例（按需追加，不改 schema）
  - source: llm                 # LLM 自动标签
    prefix: llm.
    writable_by: [llm_pipeline]
    requires_source_name: true   # = 模型名（如 gpt-4o, claude-4）
    requires_source_version: true
  - source: vendor              # 第三方供应商
    prefix: vendor.<vendor_id>.
    writable_by: [vendor_api]
    requires_source_name: true   # = vendor_id
  - source: crowdsource         # 众包平台标注
    prefix: crowd.
    writable_by: [crowd_pipeline]
  - source: compliance          # 合规审查
    prefix: compliance.
    writable_by: [compliance_officer]
    immutable: true
    propagation: descendants    # ← 沿血缘自动传播到子资产（PII 等）
  - source: customer_feedback   # 客户反馈回流
    prefix: feedback.
    writable_by: [feedback_pipeline]
```

#### 7.6.1 标签血缘传播（Classification Propagation，P1.5）

**借鉴 Apache Atlas 的 Classification Propagation 概念**：某些「治理性」标签应自动沿血缘传播到子资产，无需手动重复打。

| Registry 字段 | 取值 | 语义 |
|--------------|------|------|
| `propagation` | `none`（默认） | 不传播；标签仅作用于自身 |
| `propagation` | `descendants` | 沿 `split_from / contains` 边自动传播到所有下游 |
| `propagation_max_depth`（可选） | int | 限制传播深度（默认无限）|

**适用 vs 不适用的判断标准：**

| 标签类 | 应传播？ | 理由 |
|--------|--------|------|
| `compliance.pii=true` | ✓ | segment 含 PII，切出的所有 clip / action / frame 都含 PII |
| `customer.cust_X.exclude=true` | ✓ | 父被排除，子必须排除（避免误交付）|
| `scenario=kitchen` | ✓ | 业务场景天然继承 |
| `algo.hand_track.version=2.0` | ✗ | 算法在每个资产上独立运行 |
| `rule.quality_check=ok` | ✗ | 规则在每个资产上独立评估 |
| `quality=rejected` | ✗ | 子资产可能修复了，质量独立判定 |

**实现路径：** P1.5 加 `TagPropagator` ActionHandler（§4.11），订阅 `tag_upserted / tag_deleted / asset_created / asset_revised`：

- `tag_upserted` 命中 `propagation=descendants` → 递归找下游资产（`asset_relations` recursive CTE）→ 给每个下游写一份 tag，`source` 改为派生 source 并标 `system_metadata.propagated_from=<parent_asset_id>`
- `tag_deleted` → 反向回收下游同源派生 tag
- `asset_created` → 异步**补传播**父资产可传播的 tags（**决策 R2，rev.9**：B 路由事务**不**复制 tags，依靠 `asset_created` / `asset_revised` event 驱动异步补；接受 1–3s 的合规标签窗口）
- `asset_revised` → 同上，新版异步补父的可传播 tags

#### 7.6.2 B 路由与 PII 传播窗口（决策 R2 风险与缓解）

| 风险 | 缓解 |
|------|------|
| B 路由刚生新 clipA_v2 → 异步补 PII tag 前的 1–3s 窗口，新版无 PII tag | （1）delivery_rules 校验 PII 时**按 `logical_asset_id` 聚合所有版本 tags**（任一版本有 PII tag 即拒）；（2）`asset_created` event 进入 TagPropagator 优先队列（HighPriority handler），P95 < 500ms |
| TagPropagator handler 失败 / DLQ | PII 类 tag 在 registry 标 `critical=true`；handler 失败时**额外**写告警 metric `pii_propagation_failed` + Slack 通知 oncall |
| 异步补完前发生 delivery commit | C2 commit 协议（§10）在 PG 真源最后一次校验 PII（按 logical_asset_id），即使 ES 还没同步也能拦截 |

**为什么不选 R1（B 路由同事务复制）：**
- B 路由事务已经很重（写 9 列 entity + 4 aspect + relations + event + logical_assets 同步）；再加 N 个 tag 复制 = 锁面变大
- 父 tags 数量在某些 segment 上可能 50+，事务复制 50 行 = PG WAL 膨胀

**为什么不选 R3（派生到 logical 层）：**
- propagation 语义是「血缘继承」，logical 层是「版本聚合」；混层概念错位
- 老 ES 文档查询逻辑必须改（从 `tags[]` 查变成 join logical_assets.tags + 自身 tags）

**与 Atlas 的差异：** Atlas 用 JanusGraph + Kafka 实现，我们用 PG `asset_relations` + Actions Framework PG polling 实现，**架构成本降一个数量级**，能力等价。

**Validator 行为：**

| 校验 | 行为 |
|------|------|
| `source` 必须在 registry 内 | 不在 → 422 |
| `tag_key` 前缀必须与 source 的 `prefix`（若有）匹配 | 否则 → 422 |
| `writable_by` 校验 caller 身份 | 越权 → 403 |
| `requires_source_name=true` 时 `source_name` 必填 | 缺 → 422 |
| `requires_source_version=true` 时 `source_version` 必填 | 缺 → 422 |
| `immutable=true` 的 source 写后不允许 UPDATE/DELETE（仅可被 recomputable 流程整批 supersede） | 越权 → 403 |
| `projection_only=true` 的 source **不**走 `POST /asset-tags`，由 SearchDocumentBuilder 从 aspect 表派生 | 越权 → 403 |

**注册新来源流程：**

1. 在 `tag_sources.yaml` 加一行（PR + owner approve）
2. 若需新前缀，在 `tag_registry`（§9.1）也加 prefix 条目
3. SearchDocumentBuilder 自动从 `asset_tags` 拾取新 source（无需改 ES mapping —— `source/source_name/source_version` 都是 keyword）
4. 前端 facet UI 按 registry 自动渲染（按 source 分组显示，可勾选筛选）

**ES facet 形态保持稳定：**

```yaml
tags:
  - {key, value, source, source_name, source_version, applied_at}
  # source 可以是 algo_sdk / rule_engine / human / system / llm / vendor / crowdsource / ...
  # 都是同一字段，filter 一套语法：tags.source=llm AND tags.source_name=gpt-4o
```

**「类」不绑死在五个字母 a/b/c/d/e：** 文档与 UI 上「a/b/c/d/e」只是当前归纳的助记，不是 schema 概念；schema 里只有 `source` 字符串。新增类自然属于第 6/7/8 类，文档可顺序补充，**不需要重新定义** schema 与 ES 字段。

### 7.6.3 action 资产的 5 维标签全溯版本（rev.10）

> action 既是「action 实体」（带时间窗 + label）又是「一等资产」（享受所有 asset 通用机制），所以一个 action 资产可挂的「标签信息」有 5 个独立维度，**每一维都带完整 actor / source / version 血缘**。

| # | 维度 | 表 / 字段 | 多版本契约 |
|---|------|---------|-----------|
| 1 | **action 核心 label**（grasp / pickup） | `actions.primary_label` / `labels[]` | K1：一版一行（同窗多版本 = 不同 action_id + revision_of 串联）；`source_name + source_version` 记算法/标注员身份 |
| 2 | **action 资产开放 tag**（quality / scenario / customer / pii） | `asset_tags`（asset_id = action_id） | `source` + `source_name` + `source_version`；多源同 key 共存 |
| 3 | **后续算法当前态**（hand_track@2.0 在此 action 上跑过） | `asset_algo_latest`（含 `algo_kind`） | per (algo_kind, algo_name) 一行 + `is_pinned`；ES `algo_versions_seen` 累积全历史 |
| 4 | **action 标量评分**（confidence / rating） | `asset_metrics` | `source` + `source_name` + `source_version`；多人多次保留全部 |
| 5 | **QA 评估原档** | `asset_eval_results` | `source` + `source_version`；payload 装完整 reasoning |

**每一维的写入都同时记三个层面**（§4.9 契约）：
- `asset_events.actor`（含 actor 名 + 版本，AV1 强约束）
- `asset_events.system_metadata.run_id / algo_name / algo_version / pipeline_*`
- 表自身的 `source / source_name / source_version` 列

**ES 端复合查询示例：**

```text
"找所有：被 action_detector@2.0 标为 grasp + 被 hand_track@2.0 处理过 + QA 平均分 ≥ 4 + 标注员 labeler_007 贴过 scenario=kitchen 的 action 资产"

actions.source_name=action_detector AND actions.source_version=2.0 AND actions.label=grasp
AND algos_current.name=hand_track AND algos_current.version=2.0 AND algos_current.algo_kind=processing
AND metrics_aggregates.rating.quality_score.avg >= 4
AND tags.key=scenario AND tags.value=kitchen
AND tags.source_name=labeler_007
```

**时间线回放（events）：**

```text
2026-05-10T10:00  algo_sdk:action_detector@1.0  action_upserted  primary_label: null    → pickup
2026-05-15T09:00  algo_sdk:action_detector@2.0  asset_revised    revision: 1            → 2 (label pickup → grasp)
2026-05-15T10:30  algo_sdk:hand_track@2.0       algo_finished    status: ok, confidence=0.93
2026-05-16T14:00  labeler:labeler_007           tag_upserted     scenario: → kitchen
2026-05-17T11:00  reviewer:alg_zhang            metric_upserted  rating.quality_score: → 4.5
2026-05-18T16:00  qa_pipeline:action_compl@1.2  eval_result_added completeness=0.92
2026-05-19T08:00  propagator:TagPropagator      tag_upserted     compliance.pii: → true  (propagated_from: seg_001)
```

→ **任意一条变更都能追溯到「谁、什么版本、哪次 run、改了什么」**。

### 7.7 边界：什么时候新增 source vs 新增 aspect 表

| 新需求 | 路由 |
|--------|------|
| 标签是简单 key-value，无独立时间窗/payload | 新增 **tag_source**（仅 registry 改动） |
| 标签是「时间窗 + 离散事件」（类似 action） | 走 **`actions` 表新算法**（K1 一版一行）+ 自动投影到 ES `actions[]` |
| 标签是「标量数值」 | 走 **`asset_metrics`**（新 metric_key） |
| 标签需要完整 payload（评估 run） | 走 **`asset_eval_results`** |

→ 三表分工（§7.4）+ tag source registry（§7.6）一起，覆盖 **任意未来标签类**，schema 稳定。


---

## 8. logical_assets 一等表

### 8.1 结构

> **完整 DDL + 约束 + 索引详见 `schema-entity-aspect.md §3.5`**，本节仅给设计要点。

```sql
CREATE TABLE logical_assets (
  logical_asset_id    TEXT PRIMARY KEY CHECK (logical_asset_id ~ '^[0-9A-Za-z]{8}$'),
  asset_type          TEXT NOT NULL,                   -- 整组类型固定（LA1 不可改）
  display_name        TEXT,
  description         TEXT,
  owner               TEXT,
  status              TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','archived')),
  -- rev.11：砍掉 current_asset_id 冗余指针
  -- 查当前版本: SELECT asset_id FROM assets WHERE logical_asset_id=? AND is_current=true
  -- partial unique uq_assets_current_per_logical 保证唯一性
  current_revision    BIGINT NOT NULL DEFAULT 1 CHECK (current_revision >= 1),
  total_revisions     BIGINT NOT NULL DEFAULT 1 CHECK (total_revisions >= current_revision),
  metadata            JSONB NOT NULL DEFAULT '{}',
  extra               JSONB NOT NULL DEFAULT '{}',     -- §16.7 schema agility 实验区
  created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
  row_version         BIGINT NOT NULL DEFAULT 1       -- 乐观锁
);

-- 反向 FK（DEFERRABLE 让首版 INSERT 顺序自由）
ALTER TABLE assets ADD CONSTRAINT fk_assets_logical
  FOREIGN KEY (logical_asset_id) REFERENCES logical_assets(logical_asset_id)
  DEFERRABLE INITIALLY DEFERRED;
```

**关键设计点（rev.11）：**

- **砍 `current_asset_id` 列**：冗余指针引入 LA3/LA4 跨表 Validator；查询 `assets WHERE logical_asset_id=? AND is_current=true` 走 partial unique index `uq_assets_current_per_logical` 是 O(1)，性价比远超维护指针一致性
- **DEFERRABLE 反向 FK**：让首版创建事务可以 INSERT assets → INSERT logical_assets，FK 在 COMMIT 时统一校验
- **`row_version`** 与 `assets.row_version` 命名统一，消除「version 是业务版本还是乐观锁」混淆
- **CHECK 约束** 保留 PG 强约束：status enum + current_revision >= 1 + total_revisions >= current_revision
- **`extra JSONB`** 作为 schema agility 实验区（§16.7）

### 8.2 同步逻辑

- 创建首版资产时同事务建 `logical_assets` 行（`current_revision=1, total_revisions=1`）
- B 路由：同事务更新 **3 字段**：`current_revision / total_revisions + 1 / updated_at`（rev.11 砍指针后从 5 字段降到 3）
- 删除 logical_assets = 整组废弃（罕见，需 force flag）

### 8.4 不变式

| # | 规则 | enforce |
|---|------|---------|
| **LA1** | `logical_assets.asset_type` 首版决定，不可 UPDATE（防同一逻辑资产改类型）| PG trigger + Validator |
| **LA2** | `current_revision <= total_revisions` | CHECK 约束（DB 层）|
| ~~**LA3**~~ | ~~`current_asset_id` 指向的 asset 必须满足 `is_current=true AND logical_asset_id=自身`~~ | **rev.11 已消除**（砍 `current_asset_id` 列后无意义） |
| ~~**LA4**~~ | ~~B 路由事务必须同时更新 5 字段~~ | **rev.11 已消除**（无 `current_asset_id` 同步压力，B 路由只更 3 字段） |

### 8.3 用途

- 列表展示逻辑名（display_name）而非一串 hash
- v2 发布时的通知订阅挂在 logical 层（默认快照 + 通知 §4.7）
- owner 在 logical 层定义，可被 revision 重写

---

## 9. 治理：Tag Registry & Delivery Rules & 写入幂等

### 9.1 Tag Registry（单一权威 = §7.6 `tag_sources` + §9.1 本节合并管理）

> **rev.10 澄清：** §7.6 `tag_sources`（按 source 维度治理 = 写权限 / 命名空间 / propagation / requires_*）与本节按 key/prefix 的白名单**同属一份 `tag_registry.yaml`**，YAML 顶层两段；Validator 同时加载校验。

```yaml
# 段 ①：按 source 维度（§7.6，写权限 / 命名空间 / 传播）
tag_sources:
  - {source: algo_sdk,      prefix: algo.,        writable_by: [algo_sdk],      immutable: true}
  - {source: rule_engine,   prefix: rule.,        writable_by: [rule_engine],   requires_source_version: true, recomputable: true}
  - {source: human,         writable_by: [labeler, ops], requires_source_name: true}
  - {source: system,        writable_by: [ops]}
  - {source: compliance,    prefix: compliance.,  writable_by: [compliance_officer], immutable: true, propagation: descendants, critical: true}
  # 其他来源按需追加，详见 §7.6

# 段 ②：按 key/prefix 维度（本节，value 白名单 + 命名约束）
tag_keys:
  - prefix: algo.            # 命名前缀放行（value 自由）
  - prefix: rule.
  - prefix: customer.        # customer.<id>.<attr>
  - prefix: compliance.
  - key: scenario            # 严格 key + value 枚举
    allowed_values: [kitchen, warehouse, outdoor, lab, office, other]
  - key: purpose
    allowed_values: [training, eval, replay, demo, qa, other]
  - key: quality
    allowed_values: [ok, warn, rejected]
  - key: review.verdict
    allowed_values: [passed, failed, pending]
```

**Validator 检查顺序（rev.10 统一）：**

1. caller 解析 → 命中 `tag_sources[].source` → `writable_by / requires_source_name / requires_source_version / immutable` 校验
2. `tag_key` → 命中 `tag_keys[].key` 整条 → 校验 `allowed_values`
3. `tag_key` → 命中 `tag_keys[].prefix` → value 自由
4. 都没命中 → 422

**禁字段（Tag1，rev.10 提升不变式）：** `tag_key ∈ {label, labels, primary_label, action_id}` → 422（强制与 action label 区分，§7.0.4）。

**变更流程：** 改 `tag_registry.yaml` + git PR + owner approve；不允许运行时动态加。

### 9.2 Delivery Rules

```sql
CREATE TABLE delivery_rules (
  rule_id       UUID PRIMARY KEY,
  name          TEXT NOT NULL,
  owner         TEXT NOT NULL,
  customer_id   TEXT NULL,                       -- NULL=通用
  query_dsl     JSONB NOT NULL,                  -- 与 queries/run 同语法
  dsl_version   TEXT NOT NULL DEFAULT 'v1',      -- rev.10：DSL 版本号，引擎升级前向兼容
  enforce_mode  TEXT NOT NULL DEFAULT 'block',   -- 'block' | 'warn' | 'tag_only'
  rating_scope  TEXT NOT NULL DEFAULT 'current', -- rev.10：'current' | 'logical'（§7.4.1 MR1）
  is_active     BOOL NOT NULL DEFAULT true,
  version       BIGINT NOT NULL DEFAULT 1,
  created_at    TIMESTAMPTZ DEFAULT now(),
  updated_at    TIMESTAMPTZ DEFAULT now()
);
```

**`enforce_mode` 语义（rev.10 明确）：**

| 值 | commit 时行为 | projector 行为 |
|----|---------------|---------------|
| `block` | 不通过 → 进 `rule_failed` 阻 commit | 不打标 |
| `warn` | 不通过 → 进 `rule_failed` 但仅警告（运营可直接 commit 不需 force）| 不打标 |
| `tag_only` | 不参与 commit 校验 | DeliveryEligibilityProjector 仅按结果打/撤 `delivery_ready:{rule_id}` 系统 tag |

**DSL 版本（rev.10 决策 DR5）：** `dsl_version` 字段独立，引擎升级新增字段必须保持向后兼容旧 dsl_version 解析；不兼容时新规则强制 `dsl_version='v2'`，旧规则继续 `v1`。规则**不**自动迁移。

**执行时机：**

- **P1**：commit 时校验（C2 二次校验）；不通过项列出来由运营处理
- **P1.5**：被动 projector 自动给资产打 `delivery_ready:{rule_id}` 系统 tag（仅 `enforce_mode='tag_only'` 或 `'block'` 的规则）

### 9.3 写入幂等（I4 = 三层防线，rev.10 同步）

| 机制                                                        | 说明                                                         |
| --------------------------------------------------------- | ---------------------------------------------------------- |
| `Idempotency-Key` header                                  | API 入口写入必填；缺 → 400；同 key 同 payload 重试 → 首次结果；同 key 不同 payload → 409（**C2 commit 例外见 §11.4：每次 commit 尝试用新 key**） |
| handler/projector 生成 event                                | 不走 deterministic key，每次新 UUID；去重靠下游 unique constraint（§4.9.6.2 I5）|
| `parent_asset_version` 乐观锁（= `assets.row_version`）       | `CreateRevision` / `PATCH` 命中 B 触发字段时必传；事务条件写不符 → 409 |
| `(logical_asset_id) WHERE is_current AND NOT is_deleted` partial unique | DB 兜底，防双 current                              |
| GCS UUID4 path + `ifGenerationMatch=0` + G3 双 commit       | 防重试双份对象 + 防孤儿（§4.3.1）                                     |
| ES reindex 按 `event_seq`                                  | 天然幂等                                                       |
| sync-check / `wait_for_seq`                               | 写后强一致探测，避免 SDK 自旋（§11.2.1 / §10.3.2）                       |


---

## 10. 检索（ES）

### 10.1 入口

- 仍以 `POST /queries/run` + ES 为列表/筛选主入口
- PG 是真源；ES 由 SearchDocumentBuilder 投影；可随时重建
- `DeliveryEligibility` / commit 二次校验在 PG 重跑

### 10.2 ES asset 文档形态（一跳父带子摘要）

```yaml
asset_id: clipA
asset_type: clip
logical_asset_id: L_clip_a1b2c3
revision: 2
is_current: true
materialization: materialized
lifecycle_state: ready

# 标签（c/d/e）
tags:
  - {key: scenario, value: kitchen, source: human, source_name: labeler_007}
  - {key: customer, value: cust_X,  source: system}
  - {key: quality,  value: ok, source: rule, source_name: quality_check, source_version: "1.0"}

# 算法身份（a）
algos_current:
  - {name: hand_track, version: "2.0", status: ok, pinned: false, started_at, finished_at, run_id}
algo_versions_seen:
  hand_track: ["1.0", "2.0"]

# 直接子资产摘要（D1，L2）
actions:                                  # L2 直接子 action
  - {action_id, label: grasp, algo: action_detector, version: "2.0", is_current: true, start_ns, end_ns}
child_tasks:                              # L2 task
  - task_id: tsk1
    task_kind: sweep_floor
    start_ns, end_ns
    actions:                              # L3 actions（嵌一层）
      - {action_id, label: pickup, version: "2.0", is_current: true}
child_frames:                             # L2 frame
  - {frame_id, frame_kind: set, frame_count: 10, is_current: true}

# 评估/指标
metrics:
  - {key: good_frames_ratio, value: 0.92, source: eval_qa@1.0}

child_assets_summary:
  by_type: {action: 5, frame: 12, task: 2}

# 发现层信号（P1.5，借鉴 Amundsen）
discovery_signals:
  view_count: 1234
  download_count: 89
  trending_score: 0.78        # 0.0-1.0，时间衰减
  favorited_count: 5
  last_accessed_at: "2026-05-19T..."
  has_description: true
  owner_active: true

# 索引水位（sync-check / wait_for_seq 共用）
last_indexed_event_seq: 12346
actions_truncated: false      # §10.2.1 超 1MB 时为 true
actions_total_count: 847
```

### 10.3 读写路由分层（IR1，强约束）

> **设计来源：** DataHub 「点查走 PG，列表走 ES」契约。**根除 read-after-write 体感 bug**。

**核心规则：**

| 路径 | 走 | 理由 |
|------|----|----|
| `GET /assets/{id}`（详情**主体**） | **PG**（`asset_full` VIEW） | 强一致：标注员刚改完 tag 立即可见 |
| `GET /assets/{id}/panels/*`（反范式**面板**，见 §10.3.1）| **ES**（默认）或 PG（`?consistency=strong`）| D1 子摘要 / 引用计数；允许 1–3s 延迟 |
| `GET /assets/{id}/provenance`（血缘 + timeline）| **PG** | 强一致 |
| `GET /assets/{id}/events`（增量审计） | **PG** | 强一致 |
| `GET /assets/{id}/sync-check`（ES 就绪探测，§11.2）| **PG** + ES 水位比对 | SDK / 批量流水线卡点 |
| `GET /logical-assets/{id}/current`（跟随当前版本）| **PG** | 强一致（rev.11 砍 `databrew://` URI scheme）|
| `GET /logical-assets/{logical_id}` | **PG** | 强一致 |
| `POST /queries/run`（列表 / 筛选 / facet） | **ES**（默认）| 模糊检索 + 多维筛选 |
| `POST /queries/run?wait_for_seq=N` | **ES**（阻塞至水位，§10.3.2）| 写后立刻列表验证 |
| `GET /delivery-rules/{id}/candidates` | **ES** | 同上 |
| `GET /audit/search`（跨资产合规查）| ES | 跨资产聚合 |
| `GET /audit/lineage-search` | **PG** | 合规强一致（IR1.3）|
| `POST /deliveries/commit` C2 二次校验 | **PG** | commit 必须 PG 重跑规则 |

#### 10.3.1 详情页「主体 / 面板」二分（决策 IR-B，rev.10）

详情 API 拆成两层，避免为 D1 反范式数据破坏 IR1：

```text
GET /assets/{id}                    → PG 主体（Entity + 4 Aspect + tags + algo_latest + metrics）
GET /assets/{id}/panels/children    → ES 默认（actions[] / child_tasks[] / child_frames[] 摘要）
GET /assets/{id}/panels/references  → ES 默认（被哪些 delivery 引用、下游 asset 计数）
```

| 层 | 数据源 | UI 标注 |
|----|--------|---------|
| **主体** | PG | 无延迟提示 |
| **面板** | ES（默认）| 角标「数据可能延迟 1–3 秒」；运营可接受 |
| **面板强一致** | `?consistency=strong` → PG JOIN + 受限字段集 | 传播验证等场景；P95 < 200ms（单 asset_id 范围）|

**不变式 IR1（修订）：**

| # | 规则 | 违反 |
|---|------|------|
| **IR1.1** | 详情**主体**（身份 / tag / lifecycle / storage_uri / timeline 入口）不允许走 ES | code review 阻断 |
| **IR1.1b** | 详情**面板**允许走 ES；必须在 OpenAPI 标 `x-consistency: eventual` | 缺标注 → review 阻断 |
| **IR1.2** | 「未知集合中按条件检索」的列表必须走 ES（不能 PG 全表扫）| code review |
| **IR1.3** | delivery commit / 算法 finish / lineage-search 最终校验必须走 PG | Validator 强制 |

#### 10.3.2 列表写后验证：`wait_for_seq`（决策 QR3，rev.10）

不另起 `/queries/run-strong`（避免维护两套 query 引擎）。在现有 `POST /queries/run` 加可选参数：

```text
POST /queries/run
  ?wait_for_seq=12346        # 可选：阻塞直到 ES 对该 asset（或 logical）索引 event_seq >= N
  &wait_timeout=5s           # 默认 5s；超时 408 + body {pg_seq, es_seq, lag_seconds}
```

**适用场景：** 人工改父 tag 后立刻列表查「含该 tag 的子资产」验证 TagPropagator；算法 batch 写完后查候选池。

**实现：** SearchReindexer 在 ES doc 写入 `last_indexed_event_seq`；sync-check 与 wait_for_seq 共用同一水位字段。

**用户体感：**

> 详情**主体**永远反映 PG 真源；**面板**与**列表**接受最终一致，但提供 `wait_for_seq` / sync-check 给需要确定性的自动化场景。

### 10.4 检索能力（5 类核心需求）


| 需求           | filter 例                                                                                                 |
| ------------ | -------------------------------------------------------------------------------------------------------- |
| 按 label      | `tags.key=quality AND tags.value=ok AND tags.source=human`                                               |
| 含某类 action   | `actions.label=grasp AND actions.version=2.0 AND actions.is_current=true`                                |
| 含某 task_kind | `child_tasks.task_kind=sweep_floor`                                                                      |
| 按 asset_type | `asset_type IN [clip, action]`                                                                           |
| 按算法版本（处理过）   | `algos_current.name=hand_track AND version=2.0`（当前） / `algo_versions_seen.hand_track CONTAINS "2.0"`（历史） |


**默认 `is_current=true` 过滤**；`include_revisions=true` 翻转看历史。

#### 10.2.1 ES 文档体积上限（决策 E4，rev.10）

SearchDocumentBuilder 在组装 doc 时按**序列化后体积**自适应 cap（默认门限 **1 MB**）：

| 体积 | 行为 |
|------|------|
| ≤ 1 MB | 完整 D1：`actions[]` / `child_tasks[]` / `child_frames[]` 全装 |
| > 1 MB | 自动降级：各数组只保留 **最近 50 条**（按 `event_time` / `finished_at` desc）+ `*_total_count` + `*_truncated: true` |

**超限后的筛选路径：** 含某 action label / 全量子集 → `POST /queries/run` 加 `parent_asset_id` 或 `logical_asset_id` child filter（走 ES 子资产独立 doc，不依赖父 doc 内嵌数组）。

**不变式 ES1：** 任何单 doc 物理写入不得超过 ES `http.max_content_length` 的 80%（默认 100MB 集群配置下仍设 1MB 业务 cap，防止热点 segment 拖垮 bulk）。

### 10.5 投影触发

- 任意 PG 真源变化（assets / logical_assets / asset_tags / asset_algo_latest / actions / asset_metrics）写入 → 同事务 `asset_events` → SearchReindexer 异步 reindex 自身
- 子资产变化（action/task/frame）→ enqueue **父** reindex（D1 父带子摘要）

#### 10.5.1 父 reindex 写放大防护（决策 W4，rev.10）

**问题：** segment 下 1000 个 action 批量 upsert → 若无防护会 enqueue 1000 次父 reindex。

**三重防护（全部启用）：**

| 层 | 机制 | 效果 |
|----|------|------|
| **W1 Debounce** | ParentReindexer 收到 enqueue 后 **500ms** 窗口合并；同 `parent_asset_id` 多次 → 只执行 1 次最终 reindex | 1000 event → 1 次 ES write |
| **W2 Batch API** | `POST /assets/{seg_id}/actions:batch`（P1）一次请求 N 个 action；Writer **只 enqueue 1 次**父 reindex | 算法批量场景首选 |
| **W3 Watermark 合并** | Handler 拉 `[last_seq+1..now]` 后，同 parent + 同 event_type 族只保留 **最大 event_seq** 一条驱动 reindex | 兜底防 debounce 遗漏 |

**推荐调用：** 算法全量刷新 → **必须** batch API；零散单条 upsert → debounce 自动合并。

---

## 11. API 摘要

### 11.1 写入


| 方法                                     | 用途                                 | 必填                                        |
| -------------------------------------- | ---------------------------------- | ----------------------------------------- |
| `POST /assets`                         | 创建首版资产                             | `Idempotency-Key`                         |
| `POST /assets/{id}/revisions`          | **新增** — B 路由创建新版本                 | `Idempotency-Key`, `parent_asset_version` |
| `POST /assets/{id}/materialize`        | **新增** — virtual → materialized 升级 | `Idempotency-Key`                         |
| `POST /assets/{id}/actions`            | 创建 action（L2 segment 或 L3 task）    | `Idempotency-Key`                         |
| `POST /assets/{id}/actions:batch`      | **新增** — 批量创建/更新 action（算法全量刷新）；只触发 1 次父 reindex | `Idempotency-Key`                         |
| `PATCH /assets/{id}`                   | 更新资产（命中 revision_triggers → **422 指引** `POST /revisions`；显式路由 §4.1.1 AR1 rev.11）| `Idempotency-Key` |
| `POST /assets/{id}/tasks`              | 创建 task（或泛化 `POST /assets`）        | `Idempotency-Key`                         |
| `POST /assets/{id}/algo-finish`        | 算法完成（写 latest + events）            | `Idempotency-Key`                         |
| `POST /assets/{id}/algo-pin` / `unpin` | 显式钉/解钉算法版本                         | —                                         |
| `POST /asset-tags`                     | tag 写入；走 Validator 9.1             | —                                         |
| `POST /delivery-rules`                 | 创建/更新规则                            | —                                         |
| `POST /deliveries/{id}/resolve-conflicts` | **新增** — 运营处理 409 冲突（选替换/force/移除）；改 `delivery_items` 草稿，**不**提交 | — |
| `POST /deliveries/commit`              | C2 二次校验 + 提交（须 draft 已 resolve）| `Idempotency-Key`                         |


### 11.2 读取


| 方法                                    | 用途                                                                                                             |
| ------------------------------------- | -------------------------------------------------------------------------------------------------------------- |
| `GET /assets/{id}`                    | 详情**主体**（PG，`asset_full`）                                                                                    |
| `GET /assets/{id}/panels/{name}`      | 详情**面板**（ES 默认；`?consistency=strong` 走 PG，§10.3.1）                                                          |
| `GET /assets/{id}/provenance`         | 横向血缘 + 纵向版本 + timeline + facets（PG）                                                                          |
| `GET /logical-assets/{logical_id}`    | 列出该逻辑资产全部版本 + revision_of 链（PG）                                                                              |
| `GET /assets/{id}/lineage`            | 委托 provenance 子集（兼容，PG）                                                                                       |
| `GET /assets/{id}/sync-check`         | **新增** — ES 索引就绪探测（§11.2.1）                                                                                  |
| `GET /logical-assets/{logical_id}/current` | **rev.11** — 跟随当前版本（§4.10，砍 `databrew://` URI scheme，改 HTTP URL）（PG）                                |
| `POST /queries/run`                   | 列表/筛选（ES）；可选 `wait_for_seq` / `wait_timeout`（§10.3.2）                                                        |
| `GET /delivery-rules/{id}/candidates` | 用规则跑出资产候选（ES）                                                                                                 |
| `GET /assets/{id}/events`             | 增量拉取审计事件（§11.2.2）                                                                                            |
| `GET /audit/search`                   | **P1.5** — 跨资产审计查询                                                                                            |
| `GET /audit/lineage-search`           | **P1.5** — 合规血缘搜索（PG）。详见 §11.5                                                                              |

#### 11.2.1 Sync-check API（决策 S2+S3，rev.10）

```text
GET /assets/{id}/sync-check?expected_seq=12346&wait=5s
```

| 参数 | 说明 |
|------|------|
| `expected_seq` | 可选；写入 API 返回的 `event_seq`；缺则只报当前水位 |
| `wait` | 可选；如 `5s` — 服务端阻塞轮询直至 ES `last_indexed_event_seq >= expected_seq` 或超时 |

**响应：**

```yaml
asset_id: clipA_v2
pg_latest_event_seq: 12346
es_last_indexed_event_seq: 12346
status: ready          # ready | pending | lagging
lag_seconds: 0         # es 落后 pg 的秒数；ready 时为 0
```

| status | 条件 |
|--------|------|
| `ready` | `es_last_indexed_event_seq >= expected_seq`（或未传 expected_seq 且 es >= pg）|
| `pending` | es < expected_seq 且 lag < 30s |
| `lagging` | es < expected_seq 且 lag ≥ 30s（触发告警）|

`wait` 超时 → **408** + 同上 body。SDK helper：`client.WaitUntilIndexed(ctx, assetID, seq)` 封装。

> 点查走 PG 不依赖此 API；**列表 / 批量流水线**在 `POST /queries/run` 前调用。

#### 11.2.2 外部事件订阅协议（决策 SP4，rev.10）

| 阶段 | 协议 | 说明 |
|------|------|------|
| **P1** | **轮询** `GET /assets/{id}/events?since=<seq>&limit=500` | SDK 自存 watermark；指数退避；`event_seq` 单调 |
| **P1.5** | **SSE** `GET /assets/{id}/events/stream?since=<seq>` | 长连接 push；断线重连带 `Last-Event-ID` |
| **P2** | **GCP Pub/Sub**（可选）| PG outbox → topic；跨服务 / 跨语言订阅；与 §4.11 EventSource 切换对齐 |

**P1 SDK 契约（Python / Go）：**

```python
for event in client.subscribe_events(asset_id, since=0):
    handle(event)  # 内部：poll + backoff + 自动推进 since
```

**Ordering：** 单 `asset_id` 内按 `event_seq` 严格有序；跨 asset 不保证全局序。


### 11.3 Provenance 响应

```text
ProvenanceResponse {
  asset: AssetSummary { ..., materialization, logical_asset_id, revision, is_current, lifecycle_state }
  nodes: AssetNode[]
  edges: ProvenanceEdge[]           // split_from / contains / merged_from / derived_from / revision_of
  timeline: TimelineEntry[]         // asset_created/algo_*/tag_*/asset_revised/asset_materialized/asset_pinned/delivery_*
  revisions: RevisionEntry[]        // 同 logical_asset_id 所有版本
  facets: { tags, algos_current, metrics }
}
```

查询参数：`depth`（默认 1）、`root`、`include`（`deliveries` / `evals` / `revisions`）。

### 11.4 Delivery commit 协议（C2）

**状态机（决策 C2-D，rev.10）：** `draft → resolving → ready_to_commit → committed`。冲突处理与提交拆 API，避免 Idempotency-Key 与人工 override 打架。

```text
① POST /deliveries              → 创建 draft
② POST /deliveries/{id}/items   → 加资产行（可多次）
③ POST /deliveries/{id}/commit  → 首次校验
     ├─ 200 → committed
     └─ 409 → 进入 resolving（body 见下）
④ POST /deliveries/{id}/resolve-conflicts
     { resolutions: [{item_id, action: replace_latest|force|remove}, ...] }
     → 200，draft 更新，状态 ready_to_commit
⑤ POST /deliveries/{id}/commit  → 二次提交（新 Idempotency-Key）
```

**`items[]` 字段（决策 V4，rev.10）：**

| 字段 | 语义 |
|------|------|
| `expected_revision` | **业务版本号**（`assets.revision` INT）；非 `row_version` |
| `payload_mode` | `materialized` \| `virtual` |

**C2 只拦 B 路由：** commit 时若 `item.asset_id` 的 `revision != expected_revision` 或 `is_current=false` → `stale_revisions`。A 路由原地改 tag / metric **不**触发 stale（交付快照锁定的是「哪一版资产」，不是行锁）。

```yaml
POST /deliveries/{id}/commit
Idempotency-Key: <uuid>          # 每次 commit 尝试独立 key；同 payload 重试才复用
{
  delivery_id,
  items:
    - {asset_id, expected_revision, payload_mode}
}
```

**首次 commit 失败（409）：**

```yaml
{
  delivery_id,
  status: resolving,
  stale_revisions:  [{item_id, expected_revision, actual_asset_id, actual_revision, current_asset_id}, ...]
  deleted:          [{item_id, asset_id}, ...]
  rule_failed:      [{item_id, asset_id, rule_id, reason}, ...]
  not_materialized: [{item_id, asset_id}, ...]
}
```

运营在 UI 调 `resolve-conflicts` 选「替换最新 / force / 移除」→ 再 commit（**新** Idempotency-Key）。`force` 行在 `delivery_items` 记 `forced=true` + audit event。

**PG 二次校验（IR1.3）：** commit 事务内重跑 `delivery_rules` + PII 按 `logical_asset_id` 聚合（§7.6.2 TP2）；不读 ES。

### 11.5 合规血缘搜索 API（P1.5）

> **解决核心问题：** GDPR / PII 审计场景下，「打了某 tag 的资产流向了哪些下游 / 哪些客户的 delivery」一次性拉清单。借鉴 DataHub `searchAcrossLineage` + Atlas Classification Propagation。

#### 11.5.1 请求格式

```text
GET /api/v1/audit/lineage-search?
  source_tag=compliance.pii          ← 必填：找所有打了这个 tag（含传播来的）的资产
  &source_tag_value=true             ← 可选：限定 tag value
  &direction=downstream              ← upstream | downstream | both（默认 downstream）
  &target_kind=delivery              ← asset_type 名 / 'delivery' / 'all'（默认 all）
  &max_hops=10                       ← 血缘深度（默认 10）
  &include_propagated=true           ← 是否含 source='propagated' 的传播 tag（默认 true）
  &time_range=2026-01-01..now        ← 可选：限定 delivery 时间窗
  &customer_id=cust_X                ← 可选：限定客户
```

#### 11.5.2 响应格式

```yaml
{
  "search_summary": {
    "source_assets_count": 245,           # 直接打 tag 的资产数（含传播）
    "downstream_assets_count": 3812,      # 下游受影响资产数
    "downstream_deliveries_count": 27,    # 下游 delivery 数
    "customers_affected": ["cust_X", "cust_Y", "cust_Z"]
  },
  "source_assets": [                       # 源头资产（打 tag 的）
    {
      "asset_id": "seg_001",
      "asset_type": "segment",
      "tag": {"key": "compliance.pii", "value": "true", "source": "compliance_officer", "applied_at": "..."}
    },
    ...
  ],
  "downstream_assets": [                   # 下游受影响资产
    {
      "asset_id": "clipA_v3",
      "asset_type": "clip",
      "logical_asset_id": "L_clipA",
      "revision": 3,
      "is_current": true,
      "lineage_path": ["seg_001", "clipA_v3"],   # 从 source 到此资产的血缘路径
      "tag_origin": "propagated_from:seg_001"
    },
    ...
  ],
  "downstream_deliveries": [               # 下游 delivery 清单（合规审计核心）
    {
      "delivery_id": "del_42",
      "customer_id": "cust_X",
      "delivered_at": "2026-04-15T...",
      "follow_mode": "frozen",
      "affected_items": [
        {"asset_id": "clipA_v1", "asset_version": 1, "payload_mode": "materialized"}
      ]
    },
    ...
  ],
  "pagination": {"page": 1, "page_size": 100, "total_pages": 39}
}
```

#### 11.5.3 实现要点

| 部分 | 实现 |
|------|------|
| 找 source assets | `SELECT asset_id FROM asset_tags WHERE tag_key=? AND (tag_value=? OR ?=NULL)` |
| 下游血缘遍历 | PG recursive CTE on `asset_relations` WHERE `relation_type IN (split_from, contains, derived_from)`，按 `max_hops` 限深 |
| 关联 delivery | `JOIN delivery_items ON asset_id IN (downstream_set)` |
| 性能 | GIN index on `asset_tags(tag_key, tag_value, source)` + asset_relations `(parent_asset_id, child_asset_id)` 双向索引 |
| 走 ES 还是 PG | **走 PG**（合规审计要求强一致，IR1.3）|
| Resolver 复用 | 复用 `ProvenanceService` 的 graph 遍历逻辑（§4 / §5）|

#### 11.5.4 不变式

| # | 规则 |
|---|------|
| LS1 | 仅 `audit_role` / `compliance_officer` 可访问；普通运营 403 |
| LS2 | 每次调用写 `audit_events`（谁查的、查的什么、命中多少行）|
| LS3 | 大结果集（>10000）分页 + 警告 |
| LS4 | 不缓存（每次实时算，确保数据新鲜）|

#### 11.5.5 典型 SQL 骨架（实施参考）

```sql
WITH source_assets AS (
  SELECT asset_id
  FROM asset_tags
  WHERE tag_key = 'compliance.pii'
    AND (tag_value = 'true' OR $value IS NULL)
    AND ($include_propagated OR source <> 'propagated')
),
downstream AS (
  -- 递归找下游
  SELECT child_asset_id AS asset_id, ARRAY[child_asset_id]::text[] AS path, 1 AS hops
  FROM asset_relations
  WHERE parent_asset_id IN (SELECT asset_id FROM source_assets)
    AND relation_type IN ('split_from', 'contains', 'derived_from')
  UNION ALL
  SELECT ar.child_asset_id, d.path || ar.child_asset_id, d.hops + 1
  FROM asset_relations ar
  JOIN downstream d ON d.asset_id = ar.parent_asset_id
  WHERE d.hops < $max_hops
)
SELECT
  di.delivery_id,
  del.customer_id,
  del.created_at,
  array_agg(di.asset_id) AS affected_items
FROM delivery_items di
JOIN deliveries del USING (delivery_id)
WHERE di.asset_id IN (SELECT asset_id FROM downstream UNION SELECT asset_id FROM source_assets)
  AND ($time_from IS NULL OR del.created_at >= $time_from)
  AND ($customer_id IS NULL OR del.customer_id = $customer_id)
GROUP BY di.delivery_id, del.customer_id, del.created_at;
```

#### 11.5.6 与 DataHub `searchAcrossLineage` 的差异

| 维度 | DataHub | DataBrew |
|------|---------|---------|
| 入口 | GraphQL | REST（P1.5 GraphQL 后可加 GraphQL 入口）|
| 血缘存储 | Neo4j / ES | PG `asset_relations` |
| 大图性能 | 强（Neo4j 原生图）| PG recursive CTE，10 跳 / 万级 entity 内可控；超大走湖仓（P2）|
| 标签传播 | Classification Propagation 内建 | TagPropagator（§7.6.1，P1.5）|
| 合规审计 | searchAcrossLineage + Subscriptions | `/audit/lineage-search` + LS2 强制 audit_events |

→ 我们用 **「PG recursive CTE + asset_relations + 合规级审计」三件套**复刻 DataHub 的核心能力，零额外架构。

---

## 12. Implementation Decisions

### 12.1 架构分层

| 层 | 实现 |
|----|------|
| **Entity** | `assets`（9 列 Entity 壳）、`logical_assets`、`deliveries` |
| **Aspect（通用 4 张）** | `asset_lineage`（含时间维度）、`asset_content`、`asset_governance`、`asset_usage_stats`（P1.5） |
| **Aspect（类型专用）** | `mcap_files`（raw_mcap）、`actions`（action）、可选 P1.5 `tasks`（task） |
| **Tag/Relation** | `asset_tags`（+source/source_version）、`asset_relations`（+revision_of） |
| **Processing** | `asset_algo_latest`（+pin 字段） |
| **Audit/Timeline** | `asset_events`（append-only + unified payload）|
| **Read compat layer** | `asset_full` VIEW（JOIN Entity + 4 Aspect，repos.go 透明读）|
| **Search** | ES + `queries/run`；SearchDocumentBuilder 从 `asset_full` 构建 doc |
| **Read model** | `ProvenanceService`（横向+纵向+revisions+timeline） |
| **Write contract** | `AssetWriter`（Entity 壳 + 4 Aspect 原子写）+ `AssetWriteValidator` |
| **Type registry** | YAML 驱动 `AssetTypeRegistry` |
| **GCS layout** | `assets/<logical_asset_id>/r<n>-<uuid4>/` Validator 强制 |
| **Governance** | `tag_registry.yaml`（W2 前缀白名单）、`delivery_rules` 表 |


### 12.2 模块划分


| 模块                          | 职责                                                                           |
| --------------------------- | ---------------------------------------------------------------------------- |
| AssetTypeRegistry           | 类型元数据、合法父类型、revision_triggers、GCS 前缀                                         |
| AssetWriteValidator         | 层级 + 关系 + revision + GCS + tag 校验；A/B 路由判定                                   |
| AssetWriter                 | 原子 Create / Update(A) / CreateRevision(B) / Materialize / PinAlgo / WriteTag |
| RelationRepository          | `asset_relations` CRUD（含 revision_of）                                        |
| ProvenanceService           | graph + timeline + revisions + facets                                        |
| SearchDocumentBuilder       | ES 文档（含 D1 子摘要、tag source、algos_current pin）                                 |
| DeliveryRuleService         | 规则 CRUD、commit 二次校验                                                          |
| LogicalAssetWriter          | logical_assets 同步逻辑                                                          |
| MaterializationOrchestrator | virtual → materialized 异步 job（commit 触发）                                     |
| **ActionRunner（§4.11）**    | event 订阅 + watermark + 重试 + DLQ；嵌入 Go 后端，无 Kafka                             |
| **8 个 ActionHandler**       | SearchReindexer / ParentReindexer / RevisionNotifier / AlgoVersionsProjector / DeliveryEligibilityProjector(P1.5) / **TagPropagator(P1.5)** / **UsageStatsAccumulator(P1.5)** / OpenLineageEmitter（P2 启用）|
| **HydrateScope orchestrator**（usecase）| 按 scope 选择性 hydrate Aspect 到 `Asset`；handler 调用入口（§12.4.3）|
| **VIEW sync linter**（CI）| `scripts/check_view_sync.sh` 比对 `information_schema.columns` 与 `asset_full` VIEW；缺列 fail PR（§12.4.6 RL1）|
| **Orphan assertion job**（cron）| 24h 对账 Aspect 表与 `assets` 主表；监控孤儿 / 缺 Aspect（§12.4.6 RL2）|


### 12.3 Schema 迁移

> 完整 SQL 见 `docs/review/schema-entity-aspect.md §9`

**Entity-Aspect 拆分（本次最大变更）：**

| 变更 | 说明 |
|------|------|
| `assets` **瘦身为 9 列** | 删除 20+ 业务字段；只保留身份 + 版本链（logical_asset_id/revision/is_current）+ 系统字段（is_deleted/created_at/updated_at/row_version）|
| 新建 `asset_lineage` | 承接结构血缘字段（mcap_file_id / parent / root / asset_level / split_* / offsets）|
| ~~asset_temporal~~ | **rev.9 取消**：时间字段合并入 `asset_lineage`（创建即写、几乎不改，无独立 Aspect 必要） |
| 新建 `asset_content` | 承接产物字段（materialization / storage_uri / thumb_uri / files / frame_* / summary_text）|
| 新建 `asset_governance` | 承接治理字段（lifecycle_state / owner / reviewer / retention_* / delivery_*）|
| 新建 `asset_full` VIEW | assets LEFT JOIN 3 张通用 Aspect（lineage/content/governance）+ P1.5 含 usage_stats；`repos.go` 读路径透明切换；显式列名（避免 `SELECT *` 列名冲突）|
| GCS 路径格式 | `r<n>/` → `r<n>-<uuid4>/`；UUID4 由 AssetWriter 事务前生成 |

**其余变更（延续 rev.5）：**

| 变更 | 说明 |
|------|------|
| ~~`assets.revision_reason`~~ | **rev.9 删除**：信息走 `asset_relations(revision_of).metadata` JSONB + `asset_events.system_metadata`（双视图）|
| `assets.version` → `assets.row_version` | 改名，消除与业务 `revision` 命名冲突；注释明示「乐观锁，非业务版本」|
| `revision` / `current_revision` / `total_revisions` 改 INT → BIGINT（E2） | 与 `row_version` / `event_seq` 类型统一；零成本（8 字节 vs 4 字节差异微小） |
| `asset_lineage.duration_ms` → `duration_ns`（G1） | GENERATED 列改满精度（避免整数除法截断 off-by-one）；API 序列化层 `/ 1_000_000` 转 ms 保前端 wire format 兼容；**`sql.md` / `api-guide.md` / `data-platform-design.md` 等 30+ 文档的 `duration_ms` 引用 cleanup 在配套 PR 中处理（不在本 PRD 范围）** |
| `assets` partial unique | `(logical_asset_id) WHERE is_current AND NOT is_deleted` |
| `asset_type` CHECK | 含 `task`、`frame`；移除 `frame_set` |
| `asset_relations.relation_type` CHECK | 增 `revision_of` |
| `asset_events.event_type` | 增 `asset_revised / materialized / pinned / unpinned / metric_upserted / eval_result_added / asset_hard_deleted` |
| **`asset_events` 重构（§4.9.2）** | 改为三段式：顶层 4 列（actor / request_id / idempotency_key / event_time）+ `system_metadata JSONB` + `payload JSONB`（只装 before/after/changed_fields）；新增索引 `(actor, event_time)` / `(idempotency_key)` / GIN(`system_metadata`) |
| `asset_algo_latest` 新列 | `is_pinned`、`pinned_at`、`pinned_by` |
| `asset_tags` 改造 | 新列 `source / source_name / source_version`；PK 含 source 维度；老数据回填 `source='human'` |
| 新表 | `logical_assets`、`delivery_rules`、`asset_lineage`（含时间维度）、`asset_content`、`asset_governance`、`action_checkpoints`、`action_dlq`（§4.11）；**P1.5：`asset_usage_stats`、`asset_favorites`、`data_products`、`data_product_assets`** |
| **P1.5 governance 加列** | `view_count`、`download_count`、`last_viewed_at`、`last_downloaded_at`、`trending_score`、`favorited_count`（Amundsen 发现层信号）|
| **P1.5 asset_relations.relation_type 加** | `validated_by` |
| **P1.5 asset_events.event_type 加** | `asset_viewed`、`asset_downloaded`、`asset_favorited`、`asset_unfavorited` |
| 数据迁移（一次合并）| `UPDATE assets SET logical_asset_id=asset_id, revision=1, is_current=(lifecycle_state!='superseded')`；`frame_set→frame`；`task_demo→task`；按 assets 全表回填 logical_assets + 3 张通用 Aspect；删除 `assets.revision_reason`（信息走 `asset_relations(revision_of).metadata`） |
| Resolver API | 新增 `GET /api/v1/logical-assets/{id}/current` |
| **API 增量（rev.11）** | `PATCH /assets/{id}` 命中 revision_triggers → 422（AR1 rev.11）；`POST /assets/{id}/actions:batch`；`POST /deliveries/{id}/resolve-conflicts`；`GET /assets/{id}/sync-check`；`GET /assets/{id}/panels/{name}`；`POST /queries/run?wait_for_seq` |
| **ES 投影增量** | `materialization`、`logical_asset_id`、`revision`、`is_current`、`algo_versions_seen`、`actions[]`（≤1MB cap，超限 Top-50 + `*_truncated`）、`child_tasks[]`、`child_frames[]`、`metrics[]`、`tags[].source*`、`last_indexed_event_seq` |
| **delivery_rules 字段（rev.10）** | 加 `dsl_version`、`rating_scope`（§9.2）|
| 兼容 | filter 接受 `frame_set` / `task_demo` 旧名（deprecate 一版） |
| GCS | 新数据按 `assets/<logical>/r<n>-<uuid4>/...`；旧数据保留路径，仅在 PG 补 logical_asset_id |


### 12.4 读取层设计：VIEW + HydrateScope 按需（rev.10 决策，**rev.12 撤回**）

> ⚠️ **rev.12 撤回**：本节描述的 asset_full VIEW + HydrateScope + 三阶段实施方案**基于 Entity-Aspect 4 表拆分前提**。rev.12 既然砍掉 Aspect 拆分，VIEW + HydrateScope 也无需存在。
>
> 现网架构（`usecase.hydrateAssetReadModels`）已经是按 use case hydrate tags/algo 模式，**继续沿用即可**。无需重构成 HydrateScope 显式参数。
>
> 本节内容保留为历史决策追溯。

> 基于现有代码 (`AssetRepository / AspectRepo / usecase.hydrateAssetReadModels`) 调研结论：现有架构**已经 60% 是 DataHub-style aspect-per-Repo + hydrate 形态**，差的只是把 hydrate 从 hardcode 改按需。最佳设计 = **V0 VIEW 兼容层 + V2 渐进式按需 hydrate** 混合方案，最大化利用已有投资。

#### 12.4.1 三层结构

```text
Handler 层：按 use case 调 Usecase（已有）
   ├─ GetDetailMain   → 主体 + hydrate tags/algo
   ├─ McapLocator     → 只要 Lineage + Content（无需 tags/algo）
   ├─ AlgoList        → 只要 algo_latest 表
   └─ Children Panel  → 只要 child summaries（ES）

Usecase 层：hydrate 按 scope 按需
   ├─ GetWithScope(id, HydrateScope{Tags, AlgoLatest, Events, Children, UsageStats})
   ├─ GetEntityOnly(id)                 // 只壳
   └─ Get(id) → 内部转 GetWithScope(ALL)  // 向后兼容

Repo 层:
   ├─ AssetRepo.Get/GetAll → SELECT FROM asset_full（VIEW 主入口）
   └─ AspectXRepo.GetByAsset/BatchGet（已有，强化 BatchGet 防 N+1）
```

#### 12.4.2 `asset_full` VIEW 不可删的 5 个用途

| 用途 | 不可替代原因 |
|------|------------|
| `queries/run` PG fallback（filter EXISTS）| 跨表 filter 需要 VIEW 提供宽表形态 |
| `ListWithFilters` 动态 `ORDER BY` | 排序字段可能在任一 Aspect；VIEW 让 ORDER BY 透明 |
| `lakehouse/handler.go` 聚合查询 | 多个 `COUNT FILTER` 走 VIEW 不动 |
| Migration 兼容期 | repos.go 4 处 SQL 改 `FROM assets` → `FROM asset_full` 一行 |
| ad-hoc 排障 SQL | DBA 不需要记多张 Aspect 表名 |

→ **VIEW 是长期资产，不只是迁移期工具**。承担「逻辑宽表 / 兜底取数」职责，与 DataHub 单表「物理宽表」功能等价、性能更好（4 PK lookup vs 单表大行扫）。

#### 12.4.3 `HydrateScope` 契约

```go
type HydrateScope struct {
    Tags         bool
    AlgoLatest   bool
    Events       int  // 取最近 N 条；0 = 不取
    Children     bool // 子资产摘要（D1）
    UsageStats   bool // P1.5
    Metrics      bool
}

func (u *Usecase) GetWithScope(ctx, id string, scope HydrateScope) (*Asset, error) {
    a, err := u.repo.Get(ctx, id)   // VIEW 取主体 + Entity-Aspect 通用字段
    if err != nil { return nil, err }
    if scope.Tags         { u.hydrateTags(ctx, a) }
    if scope.AlgoLatest   { u.hydrateAlgoLatest(ctx, a) }
    if scope.Events > 0   { u.hydrateRecentEvents(ctx, a, scope.Events) }
    if scope.Children     { u.hydrateChildren(ctx, a) }
    if scope.UsageStats   { u.hydrateUsageStats(ctx, a) }
    if scope.Metrics      { u.hydrateMetrics(ctx, a) }
    return a, nil
}

// 兼容老代码
func (u *Usecase) Get(ctx, id string) (*Asset, error) {
    return u.GetWithScope(ctx, id, HydrateScope{Tags: true, AlgoLatest: true})
}
```

**收益示例：** `McapLocator handler` 当前每次额外读 `asset_tags + asset_algo_latest` 表（~10ms 浪费），改 `HydrateScope{}` 后零开销。

#### 12.4.4 `models.Asset` 不拆（前端零感知）

```go
type Asset struct {
    // Entity 壳（9 列）
    AssetID, AssetType, LogicalAssetID, Revision, IsCurrent, IsDeleted, ...

    // Aspect 字段（VIEW 自动 JOIN 注入；按字段一对一映射，不嵌套子结构）
    McapFileID, StartTimestampNs, EndTimestampNs, ...     // from asset_lineage
    Materialization, StorageURI, Files, FrameKind, ...    // from asset_content
    LifecycleState, Owner, Reviewer, RetentionTier, ...   // from asset_governance
    ViewCount, DownloadCount, TrendingScore, ...          // from asset_usage_stats (P1.5)

    // Hydrate-only 字段（usecase 内存填充）
    Tags         []*AssetTag         `json:"tags,omitempty"`
    AlgoResults  []*AssetAlgoLatest  `json:"algo_results,omitempty"`
}
```

→ **JSON 形态 100% 兼容 Frontend `types.ts`**；hydrate 字段在 scope 关闭时为 nil（`omitempty` 不序列化）。

#### 12.4.5 三阶段实施

| 阶段 | 工作 | 工作量 |
|------|------|--------|
| **P1 W1** | 落 VIEW + repos.go 4 处 SQL 改 FROM；hydrate 保持现状 | 1 天 |
| **P1 W2** | Usecase 加 `HydrateScope`；handler 渐进迁移（先 McapLocator / Foxglove） | 2 天 |
| **P1 W3** | 加 `assertion job`（每 Aspect 表对账 assets 主表，监控 VIEW LEFT JOIN 静默 NULL）+ VIEW 列同步 lint | 半天 |
| **P1.5 GraphQL 上线时** | resolver 用 `HydrateScope` 自动按字段集 build；VIEW 保留 | — |

#### 12.4.6 不变式与 lint

| # | 规则 |
|---|------|
| **RL1** | `asset_full` VIEW 列必须覆盖所有 Aspect 表的列；CI lint `scripts/check_view_sync.sh` 比对 `information_schema.columns` 与 VIEW 定义，缺列 fail PR |
| **RL2** | 每 24h `OrphanAssertionJob` 对账各 Aspect 表与 `assets` 表行数；孤儿 Aspect 行 / 缺 Aspect 主体 → metric 告警（具体 SQL + 阈值见下）|
| **RL3** | 新 handler 不允许直接 `repo.Get`；必须经 `usecase.GetWithScope` 显式声明 scope（code review 阻断）|
| **RL4** | `models.Asset` 字段增删 → 前后端 typed 同步检查（OpenAPI codegen 兜底）|

#### 12.4.7 LEFT JOIN 静默 NULL 三层防护（C1 决策，rev.10）

**为什么 LEFT JOIN 而非 INNER JOIN：** 一旦 Aspect 漏一行，INNER JOIN 让**整资产从详情页消失**，比 NULL 字段更难定位（用户看到 404 比看到 NULL 列更困惑）。LEFT JOIN 是默认安全网。

**三层防护：**

| 层 | 防护 | 已落地 |
|----|------|------|
| **写路径** | AssetWriter 同事务原子写 Entity + 4 Aspect；缺任一事务回滚 | AE1（§4.9.7）|
| **VIEW 层** | 保持 LEFT JOIN，bug 时仍能看到资产存在 | §12.4.2 |
| **巡检层** | `OrphanAssertionJob` 24h 跑 8 个对账 SQL（4 张通用 Aspect × 双向） | RL2 |
| **监控层** | 任一对账数 > 0 → metric `aspect_orphan_count{table=...,direction=...}` + Slack 告警 oncall | RL2 |

**对账 SQL（OrphanAssertionJob 实现参考）：**

```sql
-- 方向 ①：assets 主表有行但 Aspect 缺行（应恒为 0；> 0 = 写路径 bug）
SELECT 'asset_lineage_missing'  AS check_name, a.asset_id, a.asset_type
FROM assets a
LEFT JOIN asset_lineage l ON l.asset_id = a.asset_id
WHERE l.asset_id IS NULL AND a.is_deleted = FALSE

UNION ALL
SELECT 'asset_content_missing', a.asset_id, a.asset_type
FROM assets a
LEFT JOIN asset_content c ON c.asset_id = a.asset_id
WHERE c.asset_id IS NULL AND a.is_deleted = FALSE

UNION ALL
SELECT 'asset_governance_missing', a.asset_id, a.asset_type
FROM assets a
LEFT JOIN asset_governance g ON g.asset_id = a.asset_id
WHERE g.asset_id IS NULL AND a.is_deleted = FALSE;

-- 方向 ②：Aspect 表有行但 assets 主表没（孤儿；> 0 = 主表清理 bug）
SELECT 'orphan_lineage', l.asset_id
FROM asset_lineage l
LEFT JOIN assets a ON a.asset_id = l.asset_id
WHERE a.asset_id IS NULL;
-- （asset_content / asset_governance / asset_usage_stats 同理）
```

**告警分级：**

| 数量 | 行为 |
|------|------|
| 1–10 | Slack 告警 oncall（warning）；记 metric；不阻塞 |
| > 10 | PagerDuty critical；oncall 立刻 review；同时禁用相关 AssetWriter 路径直到修复 |
| > 100 | Migration 异常；触发 incident 流程 |

**为什么 OrphanRepairJob 默认不开（暂不走 C3）：** 自动补默认值会掩盖根因（写路径 bug 应该被发现，不是被静默修复）。需要时 P1.5 再加手动触发版（`POST /admin/repair-orphans?dry_run=true` 先看清单再修）。

### 12.5 分期


| 阶段       | 范围                                                                                                                                                                                                                                                                                                                                                                                                                     |
| -------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **P1** | **Entity-Aspect 拆分**（9 列壳 + 3 Aspect + `asset_full` VIEW）；层级 Validator；Revisions（**AR1 PATCH 422 显式（rev.11）** / G3 GCS 双 commit / `revision_reason` 走 relation 边）；**IR1 主体 PG + 面板 ES**；**ES 1MB 自适应 cap**；**ParentReindexer Debounce + `actions:batch`**；**sync-check + `queries/run?wait_for_seq`**；**C2 拆 resolve/commit + `expected_revision`**；**`GET /logical-assets/{id}/current` HTTP URL（rev.11，砍 databrew://）**；**events 轮询订阅**；Lifecycle；tag_registry；K1 actions；algo 双写 T3；ES D1；delivery_rules；Actions Framework（P2 polling）；logical_assets；**`asset_algo_latest.algo_kind`（processing/qa/split/enrichment 共表）+ `run_inputs` reproducibility（AE6）+ actor 命名约定（AV1）**；`thanos-io/objstore` / `hallgren/eventsourcing` / `ltree` |
| **P1.5** | DeliveryEligibilityProjector（被动打标 `delivery_ready`）；**TagPropagator**（沿血缘自动传播治理标签，§7.6.1，借鉴 Atlas）；**UsageStatsAccumulator + 发现层信号**（view/download/favorite/trending_score，借鉴 Amundsen）；**`asset_favorites` 表 + `validated_by` 边类型**；S3 follow_mode；`tasks` aspect 表；`algo_runs` 一等实体；`asset_metrics` 接入 ES facet；tag registry 增强（含 `propagation` 配置）；**GraphQL API**（详情页 / BI 接入）；**Data Products 一等表**（`data_products` / `data_product_assets`，订阅 / SLA / 文档 / owner）；**`GET /audit/lineage-search`**（合规血缘搜索 API，§11.5，借鉴 DataHub `searchAcrossLineage`）；**Python SDK**（包装现有 REST + Idempotency-Key + 重试 + 链路 ID + HTTP URL `GET /logical-assets/{id}/current` 解析，对齐 MLflow / HF Hub SDK 接入体验）|
| **P2** | Iceberg `silver_asset_lineage`；深度图；**启用 OpenLineageEmitter**（emit to Marquez/DataHub）；历史 GCS backfill；A/B 实验 derived_asset 工作流；Glossary Terms（合规需求出现时）；URN 规范化（联邦需求出现时）|


---

## 13. Testing Decisions

### 13.1 原则

- 测外部行为：API 形状、422/409、ES 字段、边列表、Entity-Aspect 事务原子性
- Fixture 小树（含 rev.9/10 新元素）：

```text
raw_mcap
  └── segment_A   (compliance.pii=true, propagation=descendants)
        ├── clip_1        v1 (materialization=materialized)
        ├── clip_1_v2     (B 路由 via POST /revisions，显式) --revision_of--> clip_1
        ├── clip_virtual  (materialization=virtual)
        ├── action_atomic (L2)
        ├── frame_set_1   (asset_type=frame, frame_kind=set, frame_count=10)
        └── task_sweep    (L2)
              ├── action_t1p1     (L3, algo v1.0, label=pickup)
              ├── action_t1p1_v2  (L3, algo v2.0, label=grasp) --revision_of--> action_t1p1
              └── action_t2m1     (L3)

→ propagator 异步把 PII tag 补到所有子（含 clip_1_v2）
```

### 13.2 建议覆盖


| 模块                    | 测什么                                                                                           |
| --------------------- | --------------------------------------------------------------------------------------------- |
| AssetWriteValidator | L1–L7 / M1–M3 / N1 严格线性 / 分叉拒绝 / revision_triggers 命中判定 |
| AssetWriter | 首版 / A / B / Materialize 原子；**Entity 壳 + 4 Aspect 同事务**；GCS UUID4 路径；**G3 PG 回滚触发 GCS DELETE**（mock GCS）|
| 写入幂等 I4 | 同 key 同 payload 首次结果；同 key 不同 payload 409；缺 `parent_asset_version` 走 B → 422；并发条件写 409 |
| **PATCH 拒绝 B 触发字段（AR1 rev.11）** | metadata 改 → A 200；storage_uri 改 → **422 + body 含 `fields:[storage_uri]` + `hint:"use POST /assets/{id}/revisions"`**；显式 POST /revisions 缺 Idempotency-Key / parent_asset_version → 422 |
| Tag Validator（§7.6 + §9.1 合并）| source 不在 registry 422；越权 403；`requires_source_name` 缺 422；immutable source 改写 403；`tag_key ∈ {label, primary_label}` 拒 |
| **algo_sdk 双写 TT3** | `algo_finished` 同事务写 `asset_algo_latest` + `asset_tags(source=algo_sdk)`；一方失败两边回滚 |
| asset_tags 多源 | 多 source 同 key 共存；rule 重算 DELETE+INSERT 不动 human |
| actions K1 | 同窗多版本独立 action_id；revision_of 串联；is_current 唯一 |
| asset_algo_latest pin + **R7** | finish 不覆盖 latest；pinned 资产走 B → 422 |
| **`algo_kind` 多种共表（AK1）** | 同 asset_id 可有 `(processing, hand_track@2.0)` + `(qa, completeness@1.2)` + `(split, segmenter@1.0)` 三行共存；PK 强制不冲突 |
| **`run_inputs` 必填（AE6）** | `algo_finished` 缺 `params / input_asset_ids / code_commit / image_digest` 任一 → 422 |
| **actor 命名（AV1）** | `algo_sdk:hand_track` 缺 `@version` → 422；`user:` / `system:` 不带版本是合法 |
| Lifecycle R1–R8 | 已交付禁删；superseded 禁软删；is_current 禁直接 archived；archived 拒收 tag（含 propagator）|
| logical_assets | 首版自动建；B 路由更新 current_*/total_revisions |
| ProvenanceService | 含 revision_of；include=revisions 返回链 |
| **Delivery commit C2（rev.10）** | `expected_revision` 不匹配 → stale_revisions；deleted / rule_failed / not_materialized；resolve-conflicts → ready_to_commit；二次 commit 必须用新 Idempotency-Key |
| **delivery_rules enforce_mode** | block 阻 commit；warn 仅警告；tag_only 不阻 commit 仅打 `delivery_ready` tag |
| **TagPropagator R2** | parent PII → 子异步补；B 路由新版触发 `asset_revised` 补传播；handler 失败 → DLQ + `pii_propagation_failed` metric |
| **IR1 主体/面板（rev.10）** | `GET /assets/{id}` 改 tag 立即可见（PG）；`/panels/children` ES 延迟可见；`?consistency=strong` 立即可见 |
| **sync-check** | expected_seq 命中 ready；< 30s pending；≥ 30s lagging；`?wait=5s` 超时 408 |
| **wait_for_seq** | `queries/run?wait_for_seq=N` 阻塞至水位；超时 408 + body 含 lag |
| **ES 1MB cap** | 1000 actions → doc Top-50 + `actions_truncated=true` + `actions_total_count=1000` |
| **ParentReindexer Debounce** | 1000 个 action 在 500ms 内 upsert → ES bulk ≤ 2 次 |
| **actions:batch** | 一次 1000 个 → 父 reindex enqueue 仅 1 次 |
| **Resolver `logical-assets/{id}/current`** | 精确 asset_id 直返；`/current` 跟随 is_current；virtual 资产返 `materialization=virtual` 无 signed_url |
| Migration | frame_set→frame；assets 回填 logical_asset_id；3 Aspect 回填；`revision_reason` 迁到 `asset_relations.metadata`；`row_version` 改名 |


### 13.3 不测

- ES 集群 perf、DataHub 本体、前端图渲染、GCS 大对象上传 perf（另开）

---

## 14. Out of Scope

- OpenMetadata / DataHub 集群部署
- **Kafka MCL**（DataHub 用 Kafka 做 MCE/MCL；我们用 PG `asset_events` + Actions Framework polling 起步；P2 若需要跨服务订阅，走已有的 GCP Pub/Sub，**不引入 Kafka 集群**）
- **Pegasus PDL / Avro Schema-First 工具链**（继续用 Go struct + JSON Schema codegen）
- Gravitino 作为 L1 主血缘库
- Neo4j / PG AGE 图数据库（用 `asset_relations` + recursive CTE + 湖仓足够）
- 替换 `queries/run`
- 删除 `mcap_files` / `actions` 物理表
- P1 N 跳图递归
- P1 强制废弃 `lifecycle_state` 列（保留为软标签）
- `asset_type=delivery`（交付批次仍是 `deliveries` 实体）
- GCS object versioning（用路径 `r<n>-<uuid4>` 替代）
- 跨 `logical_asset_id` 的合并
- **revision 分叉**（N1 严格线性；A/B 实验走 `derived_asset`）
- **多租户 / 权限边界**（本 PRD 不引入 tenant_id；后续单独 ADR）
- **跨 segment 父查询**（P1；P2 走湖仓）
- `algo_runs` 一等实体（P1.5+）
- **Glossary Terms**（P2 合规需求出现时再考虑；tag_registry 当前够用）
- **API URL 规范化**（`GET /api/v1/logical-assets/{id}/current`；保持 HTTP URL，联邦需求出现再考虑 URN）
- **GraphQL API**（P1.5 入池，P1 不做）
- **DataHub Actions 直接 fork**（不引入 Python Actions runtime；我们用 Go 实现 §4.11）
- **rev.10 新增 OOS：**
  - **自动 Retention Job**（P1.5+）：P1 仅声明 `retention_default_days` 写 `expire_at`，不跑自动 archive/delete cron
  - **revision 分支 / merge**（永远 OOS；N1 严格线性，多版本分流走 derived_asset）
  - **PG LISTEN/NOTIFY 替换 polling**（P2 备选，达到 20+ handler 或 P99 < 200ms 时启用）
  - **GraphQL 写入**（P1.5 GraphQL 只读；写入永远走 REST 保证 Idempotency-Key 链路）
  - **跨 logical_asset_id 的 rating 自动迁移**（C1 决策；语义不安全）
  - **JSONB 内部深 diff（JSON Patch RFC 6902）**（D_HYBRID：列字段精准 + JSONB 整字段，不深入嵌套）
  - **手动 commit_attempt_id**（C2-D 决策：状态机拆 API 自然分隔，不引入第三个标识）
  - **DataHub 风格 MetadataChangeProposal（MCP）+ Kafka envelope**（P1 不做；详见 §16.5。P1.5 触发条件出现时上 MCP-lite 同步端点；P2 视情况升 PubSub）

---

## 15. Further Notes

- 与 **roadmap P1-E2**（lineage Tab）对齐；本 PRD 扩大为统一目录 + 层级 + 版本 + 标签体系 + 写入契约 + 历史算法。
- 实现前新增 **ADR-002 Unified Asset Catalog & Revisions & Labels** 固化本 PRD 关键决策。
- 修订 `data-platform-design.md` / `schema-reference.md` / `eval-metrics-design.md` 中：`frame_set`、segment 子层级、`action` 进 assets、`lifecycle_state.superseded` 由 revision 驱动、`asset_tags` 多源、`asset_algo_latest` pin、三类算法产出（actions/metrics/eval_results）分工等旧述。
- **合规**：`asset_events` 分区与 retention 单独 ADR；GCS 老对象 retention 与 `is_current=false` 解耦。
- 后续依赖 `docs/review/schema-model-layers.md`（待写）统一新表/新列归属判定。

---

## 16. 参考实现对齐（调研结论）

本节记录 PRD 关键设计决策与业界参考实现的对齐结果，供新成员快速理解「我们的方向从哪来、为什么这样选」。调研范围：MLflow Model Registry 3.6、DataHub 0.13、OpenMetadata 1.4、Apache Iceberg spec、HuggingFace Hub、lakeFS、Pachyderm、DVC、K8s API Conventions（2026-05 一手文档）。

### 16.1 主要决策对照

| PRD 决策 | 最近似的参考实现 | 关键差异 |
|---------|---------------|---------|
| `logical_asset_id + revision` 版本链 | MLflow `RegisteredModel + ModelVersion` | 语义一致；我们审计和幂等远超 MLflow |
| `is_current=true + partial unique index` | DataHub `v=0`（aspect 当前态）/ MLflow `alias` | 语义等价；我们用 PG 约束替代，更强 |
| `before/after/changed_fields` 审计 payload | OpenMetadata `ChangeDescription`（fieldsAdded/Updated/Deleted） | 几乎 1:1 对齐；我们的格式更直觉 |
| `asset_events` 同事务写入 | DataHub `MCP → Kafka → MCL`（异步） | **我们更强**：同事务保证不丢事件；DataHub 是异步可丢 |
| GCS `r<n>-<uuid4>/` 路径约定 | Iceberg snapshot（单调整数 + 不覆盖）/ DVC `.dvc` metadata 分离 | 我们更简单：无需 manifest 层；UUID4 防并发碰撞 |
| `parent_asset_version` 乐观锁 + 409 | K8s `resourceVersion` CAS（etcd MVCC）| 语义等价；我们额外有 `Idempotency-Key`（K8s 无原生）|
| N1 严格线性，禁止分叉 | HF Hub `@main` 默认分支 / MLflow 默认单链 | 一致；lakeFS 分叉方案复杂度高，无业务诉求 |
| `asset_events.event_seq` 单调递增 | DataHub `Timeseries Aspect`（按时间戳） | **我们更适合增量订阅**：按 seq 推进，不漏不重 |
| `assets` + aspect 表（Entity-Aspect 模式） | DataHub Entity-Aspect、OpenMetadata Entity-Extension | 架构模式一致；我们不引入 DataHub runtime |
| Tag Source Registry 开放扩展 | OpenMetadata `Classification → Tag` 层级 FQN | 我们更扁平（单表 + registry yaml）；OM 更结构化但复杂 |
| `derived_from` 手动写入血缘边 | Pachyderm 自动 provenance tracking | 我们 P1 手动；P2 OpenLineage emit 阶段可参考 Pachyderm 自动化 |
| **Actions Framework**（§4.11） | DataHub Actions Framework（Python + Kafka） | 概念对齐；我们用 Go + PG events polling，零 Kafka，运维更轻 |
| **TagPropagator**（§7.6.1，P1.5） | Apache Atlas Classification Propagation | 概念对齐；Atlas 用 JanusGraph + Kafka，我们用 `asset_relations` + ActionHandler，零额外架构 |
| **发现层信号 + UsageStatsAccumulator**（P1.5） | Amundsen `popularity / usage` 模型 | view/download/favorited/trending_score；让运营按热度而非时间选资产 |
| **`validated_by` 边**（P1.5） | Apache Atlas `Process → output` 验证关系 | asset → eval_result 显式边，合规审计可反查 |
| **`asset_events` 三段式**（顶层 / system_metadata / payload）| DataHub MCL `SystemMetadata` | system_metadata 对齐 DataHub；顶层 4 字段独立列+索引（更快查询） |
| **`databrew://` URI** | ~~已移除（rev.11）~~ | 单实例不需要自定义 scheme，用 HTTP URL `GET /logical-assets/{id}/current` |
| **Data Products（P1.5）** | DataHub Data Product 一等概念 | 业务已有标准产品+增量更新场景，P1.5 引入 |
| **MetadataChangeProposal（MCP）** | DataHub MCP（Kafka 异步 envelope） | **P1 不做**（DataBrew 无多语言生产者 / 无 Kafka）；P1.5 视 Python SDK 落地情况上 **MCP-lite 同步端点**；P2 真出现多服务 ingest 时走 Pub/Sub。详见 §16.5 |

### 16.2 我们主动超越参考实现的维度

| 维度 | MLflow | DataHub | OpenMetadata | DataBrew PRD |
|------|--------|---------|-------------|-------------|
| 审计粒度 | 仅 user_id + timestamp | 快照对比（无内置 diff）| fieldsAdded/Updated/Deleted | **before/after/changed_fields（最细）** |
| 审计时机 | 无事务保证 | 异步 Kafka（可丢事件）| 异步 | **同事务写入，绝不丢** |
| 写入幂等 | 无（create 两次 = 两个版本）| createIfNotExists 弱语义 | 无 | **Idempotency-Key + 乐观锁 + DB unique 三层** |
| 并发安全 | alias last-write-wins | MySQL CAS（弱）| 无 | **PG partial unique + 409 强拒绝** |
| 字段级 diff | 无 | 需外部比较快照 | 有（三数组）| **内置，Writer 自动计算** |

**结论**：DataBrew 的版本、审计、幂等设计在业界参考实现基础上**主动加强**，不是简化版。在审计完整性和并发安全性上超过了所有被调研的参考实现。

### 16.3 dev 阶段可直接引入的组件（调研确认）

| 组件 | 用途 | 引入方式 |
|------|------|---------|
| `thanos-io/objstore` | GCS 不可变对象适配层 | `go get github.com/thanos-io/objstore` |
| `hallgren/eventsourcing` | PG append-only 事件流（对应 `asset_events`） | `go get github.com/hallgren/eventsourcing` |
| PostgreSQL `ltree` | 资产层级树 O(1) 子树查询 | `CREATE EXTENSION ltree;`（内置） |

**不引入（确认）**：MLflow Go SDK（功能不完整）/ lakeFS（无 embedded 模式）/ EventStoreDB（独立服务）/ OpenLineage Go client（以 job 为中心，模型错位）。

### 16.5 MetadataChangeProposal（MCP）—— 评估与立场

**结论：P1 不做；P1.5 视情况上 MCP-lite；P2 才考虑完整异步版。**

**DataHub 为什么要 MCP（外因，DataBrew 都不存在）：**

| DataHub 场景 | DataBrew 现状 |
|---|---|
| 多语言生产者（Python / Java / Spark / Airflow）异步推送元数据 | 单一 Go 后端；写入只走 REST |
| Pegasus PDL 强 schema-first 跨语言契约 | Go struct + JSON Schema codegen |
| 解耦生产者与 PG schema 演进 | 直接迭代 PG schema，单 repo |
| Kafka MCE → 验证 → 落 PG → MCL 异步管道 | 同步落 PG + 同事务 `asset_events`，**`asset_events` ≈ MCL** |
| 数百个上游 ingest | P1 只有 backend + algo_sdk 两类写入方 |

**MCP 的核心价值已被现有设计覆盖：**

| MCP 价值 | DataBrew 已覆盖于 |
|---------|---------------|
| 写入幂等 | `Idempotency-Key` + `parent_asset_version` 三层（§4.6 / I4 / I5） |
| 变更审计 | `asset_events` 三段式 + `system_metadata` 对齐 DataHub `SystemMetadata`（§4.9）|
| 副作用编排 | Actions Framework（§4.11）≈ DataHub mae-consumer-job |
| 写后强一致探测 | sync-check / `wait_for_seq`（§11.2.1 / §10.3.2）|

**演进路径：**

| 阶段 | 形态 | 触发条件 | 工作量 |
|------|------|---------|--------|
| **P1（现在）** | 不做。继续 typed REST（`POST /assets/{id}/algo-finish` / `POST /asset-tags` ...） | — | 0 |
| **P1.5（按需）** | **MCP-lite 同步端点**：单一 `POST /api/v1/aspects` 接受 `{entity_urn, aspect_name, aspect_value, system_metadata}` → 内部路由到对应 AssetWriter 方法。**仍同步 PG + asset_events**，零 Kafka / Pub/Sub | Python SDK 重度落地 + algo_sdk 跑成独立服务 | 1–2 周 |
| **P2（多服务异步）** | **MCP-on-PubSub**：producer 推 `metadata-change-proposals` topic → `MCPConsumer` 进程 validate + 落 PG + emit MCL（复用 §4.11.5 `PubSubEventSource` 抽象，自然衔接）| 多语言 producer / 跨 region / 第三方 ingest | 视架构需要 |

**MCP-lite 的杀手收益（如果触发了 P1.5）：**

1. **前向兼容**：新增 Aspect 后，老 SDK 无需改代码就能写新 Aspect（schemaless K-V wire format）
2. **统一 SDK 入口**：Python `client.upsert_aspect(urn, "asset_governance", {...})` 一个方法包打所有；不需要为每种写入封 N 个 REST 方法
3. **零运维成本**：仍走 HTTP 同步事务，调试 / 重试 / 告警全套沿用现有体系

**P1 显式不做的理由（避免反复讨论）：**

- DataBrew 没有 DataHub 的「100+ ingestion source」社区生态压力
- 抽象 envelope 会让前端 SDK 失去 type safety（typed REST 在 OpenAPI 里能 codegen TS 类型）
- 引入 wire format 后，前向兼容需要长期维护（DataHub Pegasus 团队为此投入巨大），收益不匹配
- `asset_events` 已经是 MCL 等价物，覆盖了 95% 的「事件流回放 / 审计 / 跨服务订阅」诉求

### 16.6 P2 参考方向

- **Pachyderm 自动 provenance**：P2 OpenLineage emit 阶段参考，将手动 `derived_from` 写入改为算法 SDK 自动上报
- **DataHub Timeseries Aspect**：`asset_events` 已用 event_seq 替代；如需大规模指标时序分析，参考 DataHub timeseries 查询模式
- **lakeFS content-addressed dedup**：长期若需要「同内容不同路径共享物理对象」，参考 lakeFS CAS 实现；当前 `ifGenerationMatch=0` 已够用
- **事件源升级到 GCP Pub/Sub**：DataBrew 已有 Pub/Sub 基础设施。出现「跨服务订阅 / 算法 SDK 独立部署 / 秒级延迟不够」时，加 `PubSubEventSource` 实现（§4.11.5）；ActionHandler 零侵入。**不引入 Kafka 集群**
- **DataBrew MCP Server（Model Context Protocol）**：参考 DataHub MCP Server（`acryldata/mcp-server-datahub`）的产品/代码设计，把 DataBrew 现有 REST API（`/queries/run` / Provenance / Resolver / `/audit/lineage-search`）封装为 MCP tools，供 Cursor / Claude Desktop / Windsurf / OpenAI 等 AI agent 直接查询资产元数据 + 写入 tag / revision。P1/P1.5 不做（PRD 现有 API 即数据源）；P2 真有 AI agent 集成需求时一周可启动
- **持续监控版 Assertions（数据 SLA）**：现有 `delivery_rules`（§9.2）是「commit 时事件式拦截」，对齐 DataHub Assertions 可扩展为「定时执行 + 历史 trends + 失败告警」的持续监控（例：「scenario=kitchen 的 clip 数每天 ≥ 100」「最近 24h 必须有新 segment」）。新表 `asset_assertions`（name / query_dsl / check / schedule / on_fail）+ ActionHandler `AssertionRunner` 定时执行 + 失败发飞书/Slack 告警 + 历史保留 evaluation 记录。**业务真出现数据 SLA 监控诉求时再启动**（约 2–4 周工程）|

---

## 附录 A：Provenance 边示意

```text
[raw_mcap:abc12345] --split_from--> [segment:def67890]
[segment:def67890] --split_from--> [clip:ghi11111]                  # clipA v1
[clip:ghi22222]    --revision_of--> [clip:ghi11111]                  # clipA v2 (algo=hand_track@2.0, run=R789)
[segment:def67890] --split_from--> [task:pqr44444]
[task:pqr44444]    --contains--> [action:stu55555]                   # L3 v1
[action:stu66666]  --revision_of--> [action:stu55555]                # L3 v2
[segment:def67890] --split_from--> [action:jkl22222]                 # L2 原子
[segment:def67890] --split_from--> [frame:mno33333]                  # frame_kind=set
[segment:def67890] --derived_from--> [derived:ppp44444]              # 算法产出
[segment:aaa] --merged_from--> [derived:bbb]
[segment:ccc] --merged_from--> [derived:bbb]
[derived:exp1] --derived_from--> [clip:ghi22222]                     # A/B 实验
[derived:exp2] --derived_from--> [clip:ghi22222]                     # A/B 对照
```

Timeline 示例（clipA）：

```text
v1: asset_created → algo_finished(1.0) → tag_upserted(scenario:kitchen)
    → delivery_item_added(cust_X) → asset_revised(by=v2)
v2: asset_created(reason=algo_rerun, run_id=R789, supersedes=v1)
    → algo_finished(2.0) → tag_upserted(quality:ok rule@1.0)
```

---

## 附录 B：Asset Type Registry（概念）

```yaml
raw_mcap:
  parents: []
  asset_level: 0
  governance: minimal

segment:
  parents: [raw_mcap]
  asset_level: 1
  governance: full
  versioning: enabled
  revision_triggers: [storage_uri, files, start_timestamp_ns, end_timestamp_ns]

clip:
  parents: [segment]
  asset_level: 2
  governance: full
  versioning: enabled
  revision_triggers: [storage_uri, files, start_timestamp_ns, end_timestamp_ns]
  gcs_prefix: "assets/{logical_asset_id}/r{revision}/"
  delivery_default_mode: materialized

frame:
  parents: [segment]
  asset_level: 2
  governance: full
  versioning: enabled
  variants: [single, set]
  revision_triggers: [storage_uri, files, frame_items]
  gcs_prefix: "assets/{logical_asset_id}/r{revision}/"
  delivery_default_mode: materialized

task:
  parents: [segment]
  asset_level: 2
  governance: full
  versioning: enabled
  revision_triggers: [start_timestamp_ns, end_timestamp_ns, task_kind]
  delivery_default_mode: virtual

action:
  parents: [segment, task]
  asset_level: 2_or_3                          # 由 parent 决定
  governance: full
  versioning: enabled
  aspect: actions
  time_within_parent: required_when_parent_is_task
  revision_triggers: [primary_label, labels, start_ns, end_ns]
  delivery_default_mode: virtual

derived_asset:
  parents: [segment, clip, action, frame, task, derived_asset]
  versioning: enabled
  revision_triggers: [storage_uri, files]
  gcs_prefix: "assets/{logical_asset_id}/r{revision}/"
```

---

## 附录 C：clipA 版本演化完整示例

```text
logical_asset_id = clipA_v1                                  ← 首版 asset_id 自引用（rev.11 命名约定）
display_name = "厨房采购演示片段 - clip A"
owner = team_alpha
current_revision = 2, total_revisions = 2                    ← rev.11：current_asset_id 列已砍，查询 assets WHERE logical_asset_id=clipA_v1 AND is_current=true 即可

asset_id    revision  is_current  lifecycle    materialization  storage_uri
clipA_v1    1         false       superseded   materialized     gs://cyb-prod/assets/clipA_v1/r1-550e8400/video.mp4
clipA_v2    2         true        ready        materialized     gs://cyb-prod/assets/clipA_v1/r2-f47ac10b/video.mp4

asset_relations:
  clipA_v1 --split_from--> segment_X
  clipA_v2 --split_from--> segment_X
  clipA_v2 --revision_of--> clipA_v1   (algo=hand_track@2.0, run_id=R789, reason=algo_rerun)

asset_tags（clipA_v2，5 类示例）:
  algo.hand_track.version=2.0          source=algo_sdk
  algo.hand_track.status=ok            source=algo_sdk
  rule.quality_check=ok                source=rule_engine source_version=1.0
  scenario=kitchen                     source=human source_name=labeler_007
  customer.cust_X.priority=high        source=ops

asset_algo_latest（clipA_v2）:
  hand_track  version=2.0 status=ok is_pinned=false
  action_detector version=2.0 status=ok is_pinned=false

asset_events:
  clipA_v1: asset_created, algo_finished(1.0), tag_upserted(scenario:kitchen),
            delivery_item_added(cust_X, delivery_42), asset_revised(by=v2)
  clipA_v2: asset_created(supersedes=v1, run_id=R789), algo_finished(2.0),
            tag_upserted(quality:ok rule@1.0)

delivery_items:
  delivery_42 → clipA_v1 (frozen, materialized)          ← 客户链接稳定
  delivery_99 → clipA_v2 (frozen, materialized, follow)  ← 续交付新版
```

---


# ADR-002: Unified Asset Catalog, Revisions & Labels

| 字段 | 值 |
|------|-----|
| ID | ADR-002 |
| 状态 | Accepted (2026-05-20) |
| 取代 | — |
| 被取代 | — |
| 关联 | PRD §3 / §4 / §7 / §8 / §9；Linear CYB-983；ADR-001（data-platform-design 主线） |
| 决策人 | 架构 + 平台开发 + 产品 |

> 本 ADR 与 PRD **同文件、同 source of truth**：PRD 描述「是什么 / 怎么用」，ADR 描述「为什么这样选 / 当时考虑过哪些方案 / 后果与风险」。两者交叉引用，不重复正文细节。

---

## 1. Context

DataBrew 的「资产 + 算法 + 交付」核心域已运行半年，出现六类瓶颈（PRD §1）：认知不统一、层级未 enforced、追溯不完整、检索与溯源割裂、版本迭代无显式建模、标签体系未一等建模。

业务侧明确诉求：

- **「一切皆 asset」**：clip / action / frame / task 与 segment 同等治理待遇
- 资产可能 **有 GCS 媒体产物**（要保护客户已下载链接）也可能 **保持 virtual 无产物**
- 算法会迭代、人工会返工，**同一逻辑产物**的多个版本必须显式可追溯
- 五类标签（algo 身份 / algo 结果 / 规则派生 / 人工标注 / 业务客户）要统一检索 → 驱动交付

工程侧约束：

- 现有栈 **PostgreSQL + Elasticsearch + GCS + asset_events**
- 半年内不引入 DataHub GMS / Kafka / 图数据库 / 多租户 / Gravitino L1 联邦
- 关键设计决策固化于本 PRD 各章；ADR §2 给出 6 组「为什么这样选」

---

## 2. Decision（5 大决策 + 17 小决策）

### D1. 实体模型：一等资产 + materialization 属性

**核心决策：** materialization 属性 / virtual→materialized 走原地升级 / virtual 是否进 delivery 由运营决定 / frame 支持 single + set 两种形态

- 所有业务类型（raw_mcap / segment / clip / action / frame / task / derived_asset）进 `assets` 表，独立 `asset_id`
- 引入 `materialization ∈ {virtual, materialized}` **属性列**，不为「无产物」单独建类型
- `virtual → materialized` 单向、走 A 路由原地升级（不新建版本）；`materialized → virtual` 禁止
- `frame_kind ∈ {single, set}` 允许单帧与帧集合共存
- virtual 资产能否进 delivery 由 **运营在 commit 时**决定（`payload_mode`），Registry 仅给 hint 不强制

→ 详细规则：PRD §3.4 / §3.5；不变式：PRD §3.7。

### D2. 层级与结构血缘：固定两层切分 + typed 边

**核心决策：** 层级树固定 / 边类型固定 / 写入契约强制写关系（详见 PRD §3.1 / §3.6 / §3.7）

- 层级树固定：`raw_mcap → segment → {clip, frame, task, action(L2)}`，`task → action(L3)`；禁止 `segment → segment`
- 边类型：`split_from` / `contains` / `merged_from` / `derived_from` / `revision_of` / `sampled_from`
- 写入契约 **强制**写 parent 与 typed relation；缺则 422

### D3. 版本血缘：logical_asset + 严格线性 revision + GCS 路径契约

**核心决策：** A/B 写入路由按关键字段清单自动判 / 严格线性版本链（A/B 实验走 derived_asset） / logical_assets 一等表 / 写入幂等（Idempotency-Key + 乐观锁 + DB partial unique + GCS deterministic name） / 交付默认快照 + 通知

- 「版本」是逻辑实体上的事，不是 `asset_id` 上的事：`logical_assets` 一等表 + `logical_asset_id / revision / is_current / revision_of` 边
- **N1 严格线性**：v1 → v2 → v3 单链；A/B 实验走 `derived_asset`，不走 revision
- 写入路由 **A vs B 由关键字段清单 M3** 决定（PRD §4.1）；关键字段在 Registry 配置
- GCS 强制 `assets/<logical_asset_id>/r<n>/...`；deterministic name + `ifGenerationMatch=0` 防重试双份
- **I4 幂等**：`Idempotency-Key` 必填 + `parent_asset_version` 乐观锁 + DB partial unique 兜底
- **S4 交付默认快照**（`delivery_items.asset_id` 永指具体版本）+ revision 发布通知 projector；S3 `follow_mode` 可选扩展

→ 详细：PRD §4 / §8 / §9.3。

### D4. 标签体系：开放多源 + asset_tags 单表 + actions 一版一行（已知 5 类，可扩展 N 类）

**核心决策：** 5 类标签来源全要 / 多版本结果并存（按 source_version） / 算法 current 默认时间 + 可显式 pin / actions 每版独立 action_id 走 revision_of / asset_tags 单表多源 + source/source_name/source_version 三列 / actions vs asset_metrics vs asset_eval_results 三表分工

- **当前已知 5 类来源**：a 算法身份 / b 算法结果 / c 规则派生 / d 人工 / e 业务
- **可扩展 N 类**：`asset_tags.source` 为开放字符串，**不在 DB CHECK 限定**；由 **Tag Source Registry** 治理（PRD §7.6）。新增类（LLM 自动 / 第三方供应商 / 众包 / 合规审查 / 客户反馈 …）= 在 `tag_sources.yaml` 加一行，**不动 schema、不动 ES mapping**
- **X 并存**：同算法多版本结果按 `source_version` 并存，旧结果永不丢
- **R 混合**：算法 current 默认按 `finished_at desc`，可显式 `is_pinned` 冻结
- **K1**：`actions` 每版独立 `action_id` 走 `revision_of`，不破「action 一等资产」契约
- **T1**：`asset_tags` 加 `source / source_name / source_version` 三列，多源同 key 共存（人标 + 规则双确认）
- **V1 三表分工**：`actions` 装离散事件、`asset_metrics` 装标量指标、`asset_eval_results` 装审计原档；不互相挤占

→ 详细：PRD §7 / §4.4。

### D5. 治理：Tag Registry + Delivery Rules + 二次校验

**核心决策：** ES 统一搜索入口 / 父文档 denormalize 一跳子摘要 / delivery_rules 一等表 + commit 二次校验 / tag_registry 严格白名单 + 前缀语义 / lifecycle 三态简化 + R1–R6 删除约束

- **G2 + W2**：`tag_registry.yaml` 严格白名单 + 前缀语义（`algo.*` / `customer.*`）+ 整 key `allowed_values`；缺命中 → 422
- **E3**：`delivery_rules` 一等表，复用 `queries/run` DSL；P1 commit 时校验，P1.5 加被动 projector 自动打标
- **C2**：delivery commit 时 PG 二次校验；不通过项分类（`stale_revisions / deleted / rule_failed / not_materialized`）由运营按行处理
- **U1 + D1**：ES 统一搜索；父文档 denormalize 一跳子摘要（`actions[]` / `child_tasks[].actions[]` / `child_frames[]`）
- **Lifecycle 三态简化为 `ready / archived / superseded`** + **R1–R6** 删除约束（已交付禁删、is_current 禁直接 archived 等）

→ 详细：PRD §6 / §9 / §10 / §11.4。

### D6. 全量审计契约：asset_events append-only + before/after diff

**核心决策：** 单一审计表 + append-only + before/after diff + actor/request_id/idempotency_key 必填（详见 PRD §4.9）

- **唯一审计表 `asset_events`**：append-only，按月分区，单调 `event_seq`
- **AE1**：任何写入 `assets / asset_tags / asset_algo_latest / actions / asset_metrics / asset_eval_results / deliveries / delivery_items / logical_assets` 必同事务写 event
- **AE2**：禁止 UPDATE / DELETE
- **AE3**：每条 event 必含 `actor / request_id / idempotency_key / caller_ip`
- **AE4**：更新类事件 payload 含 `before / after / changed_fields`（前端 timeline 可还原「谁把什么字段从 X 改成 Y」）
- **AE5**：hard-delete 前写 `asset_hard_deleted` event 含最后快照
- 新事件类型：`metric_upserted` / `eval_result_added` / `asset_hard_deleted`
- 读取入口：`GET /assets/{id}/events`（P1）；`GET /audit/search`（P1.5）

→ 覆盖 100% 资产变更（含 confidence 微调、tag 增删、pin/unpin、materialize、revision、归档、删除）；详细见 PRD §4.9。

---

## 3. Considered Alternatives（已否决方案 + 理由）

| 决策 | 否决方案 | 否决理由 |
|------|---------|----------|
| D1 物化属性 | B 独立类型（`virtual_clip` 等） | 类型膨胀；UI/API 重复实现 |
| D1 物化升级 | V2 走 revision B 路由 | virtual 无产物可保护，B 路由生「永不引用的空 v1」污染列表 |
| D2 层级 | 允许 `segment → segment` 子分段 | 与产品「只有 MCAP 下一层叫 segment」冲突 |
| D2 类型扩展 | 每类一张主表 | 与 DataHub aspect 模式相悖；新类型上线成本陡 |
| D3 幂等 | I1 客户端幂等 only | 不能防 race；多 writer 并发会双 current |
| D3 幂等 | I2 内容哈希 | 「同内容不同意图」误命中 |
| D3 版本链 | N2/N3 树状分叉 | 当前无真实需求；UI 复杂；N1 兼容未来扩 |
| D3 交付 | S2 纯动态 follow | 客户「同 URL 内容变」是事故级体验 |
| D3 commit | C1 严格快照不校验 | 可能交付已 reject 的资产 |
| D3 commit | C3 全自动替换 | 「悄悄改了运营的选择」事故风险 |
| D4 算法当前态 | P 隐式 only | rollback 必须假跑；A/B 锁版做不到 |
| D4 算法当前态 | Q 显式 only | 99% 场景多余配置成本 |
| D4 多版本结果 | Y 覆盖 | 历史只能去 events 翻；与「按 version 筛交付」冲突 |
| D4 multi-result PK | K2 JSONB versions 数组 | 破坏「action 一等资产」；JSONB 查询慢 |
| D4 multi-result PK | K3 同 action_id 多行 | 破坏「PK = asset_id」契约 |
| D4 tag 存储 | T2 分三表 | 跨表 UNION；ES 投影多处合并 |
| D4 tag 存储 | T3 c 独立表 | 折中无明显好处 |
| D4 algo result 表 | V2 合并 actions/metrics | 失去「action 一等资产」语义 |
| D4 algo result 表 | V3 删 eval_results | 合规/原档审计回放需求强 |
| D5 tag registry | G1 开放 | 半年内 tag 空间一定爆炸 |
| D5 tag registry | G3 分层 / G4 命名空间 | 一阶段过度设计；W2 前缀已覆盖 95% |
| D5 tag registry | W1 整 key 白名单 | `algo.*` / `customer.<id>.*` 不可穷举 |
| D5 delivery rules | E1 不存 / E2 saved_queries 兼用 | 缺权限/审计/客户绑定语义 |
| D5 commit | C4 整批失败 | 大批次几乎永远 commit 不了 |
| D5 ES denormalize | D0 不 denormalize | 父子筛选要两段查询 |
| D5 ES denormalize | D2 全树压平 / D3 ES join | 文档膨胀 / 性能差 |
| Lifecycle | 保留 7 态状态机 | 与「不绑硬门槛」决策冲突；过渡态归算法/events |
| 多租户 | 引入 `tenant_id` | 本期无业务诉求；引入成本 vs 收益不平衡 |

---

## 4. Consequences

### 4.1 Positive

- **AI / 新人友好**：6 类资产 + 6 类边 + A/B 两路写入 + 22 项决策矩阵全部文档化，单点可查
- **客户体验稳定**：S4 快照 + GCS 路径契约保证已下载链接永不被覆盖
- **横向可演进**：新增 `asset_type` 走 Registry，不需要新建主表
- **纵向可追溯**：每个资产从 created → split → algo → tag → revision → materialize → delivery 全在 events 时间线可重放
- **检索与交付统一**：U1 + D1 + E3 让运营一处入口完成「筛选 → 资格校验 → 交付」
- **写入幂等**：I4 + DB partial unique + GCS deterministic name 三重保护，重试/race 不产脏数据

### 4.2 Negative / Trade-offs

| 代价 | 缓解 |
|------|------|
| Schema 显著扩展（`assets +6 列` / `actions` PK 改 / `asset_tags +3 列` / 2 新表） | 两步迁移（PRD §12.3）；老数据回填可分批 |
| ES 文档膨胀（含 `actions[] / child_tasks[] / metrics[]`） | nested 字段；预期父文档 ≤50KB，仍在 ES 经济区间 |
| 子资产变化要触发父 reindex（D1 denormalize） | events → projector 异步；已有 `search_reindex_jobs` 机制 |
| 路由 A vs B 判定需 Registry 配置 | 出错时 fail-safe 走 B（更保守）；Registry 改动经 PR review |
| Tag registry 改动要走 PR | 接受；防止 tag 空间爆炸的代价 |
| `actions` 表行数 N 算法 × M 版本 翻番 | 元数据轻（≤1KB/行），10M 行 PG 无压力；过期归档 cold |

### 4.3 Risks

| 风险 | 缓解 |
|------|------|
| 现网历史资产无 `logical_asset_id` | 一次性 backfill = `asset_id`；幂等可重跑 |
| GCS 老对象路径不符合 `r<n>/...` | 仅 PG 补 `logical_asset_id`，文件不动；新数据按新契约 |
| 算法 SDK 客户端不传 `Idempotency-Key` | API 400 hard fail；SDK 升级期间 gateway 注入临时 key + 告警 |
| 一次大算法回灌 5000 资产 → 5000 GCS 写 + ES reindex | `algo_runs` 一等实体 P1.5 落地后批量监控；P1 暂用 events GROUP BY |
| `delivery_rules.query_dsl` 与 `queries/run` 语法漂移 | 共享同一 DSL parser；CI 校验规则可解析 |

### 4.4 Compliance / Audit

- 任意 `asset_id` 可从 `asset_events` 重放完整生命周期（created / algo / tag / revision / materialize / pin / delivery）
- `delivery_items` 永远指向具体 `asset_id + asset_version` → 客户已交付内容可审计、可回放
- Tag registry 与 delivery_rules 走 git PR，变更有审计

---

## 5. Implementation Plan（按 wave 拆解）

> **Source of truth 约定：** 本 ADR §5 仅给 P1 内部的 wave 实施顺序作补充说明；**阶段范围（P1 / P1.5 / P2 包含什么）以 PRD §12.4 为准**。两处冲突时，以 PRD 为准并回头修订本节。


### 5.1 P1（CYB-983 主体，预估 3–5 周）

```
WAVE 1: Schema 基线
  - migration: assets +6 列；asset_relations.relation_type CHECK；asset_events.event_type
  - migration: asset_algo_latest +pin 列
  - migration: 新表 logical_assets / delivery_rules
  - migration: asset_tags +source/source_name/source_version；PK 改造；老数据回填 source='human'
  - migration: frame_set → frame；task_demo → task
  - migration: assets backfill logical_asset_id = asset_id
  - DB partial unique uq_assets_current_per_logical

WAVE 2: 写入契约
  - AssetTypeRegistry YAML loader
  - AssetWriteValidator（层级 + materialization + revision + GCS 路径 + tag W2）
  - AssetWriter（首版 / A / B / Materialize / PinAlgo / WriteTag）
  - LogicalAssetWriter 同步逻辑
  - 全部写入 API 强制 Idempotency-Key + parent_asset_version 乐观锁

WAVE 3: 读模型 + ES
  - SearchDocumentBuilder：新字段 + D1 父子摘要
  - ProvenanceService：含 revisions / pin / timeline
  - GET /assets/{id}/provenance
  - GET /logical-assets/{id}
  - queries/run filter 扩展（is_current / logical_asset_id / algos_current.* / actions.* / child_tasks.* / metrics.*）

WAVE 4: 交付
  - delivery_rules CRUD + 校验
  - POST /deliveries/commit 走 C2 二次校验
  - asset_revised event projector → 通知
  - virtual 资产 materialize orchestrator + POST /assets/{id}/materialize
```

### 5.2 P1.5（扩展）

- `algo_runs` 一等实体 + 批量 rollback
- `DeliveryEligibility` 被动 projector 自动打 `delivery_ready` tag
- S3 `follow_mode` 在 `delivery_items`
- `tasks` aspect 表（若用量证明需要）
- tag registry UI / 审核流

### 5.3 P2（远期）

- Iceberg `silver_asset_lineage` 深度图
- OpenLineage emit（跨系统作业审计，非 segment 父子主库）
- 历史 GCS 老对象按 `r<n>/...` 路径 backfill 工具
- A/B 实验工作流（`derived_from` + 实验元数据）

---

## 6. Open Questions（未来 ADR 议题）

- 多租户与权限模型（`tenant_id` 全量穿透 vs RLS）
- 跨 segment 父查询性能边界（湖仓 vs PG 物化视图）
- `algo_runs` 模型与 OpenLineage 的兼容
- 长周期资产 retention 与 GDPR 数据删除流程
- `frame_set` 规模化（>1000 帧/资产）时的存储与查询
- L2 湖仓 catalog 联邦（Gravitino 等）何时启用

---

## 7. References

- PRD 主体：本文件 §1–§15 + 附录 A–D
- **ADR-001（占位 / 待迁移）**：`docs/adr/001-data-platform-design.md` —— 现仓库无此文件；建议把 `docs/review/data-platform-design.md` 升格为 ADR-001（数据平台总体设计主线），本 ADR-002 作为「资产目录 + 版本 + 标签」子域 ADR 挂在其下。完成前，本 ADR §1 的「现有栈与约束」直接引用 `docs/review/data-platform-design.md`。
- Schema 源：`schemas/pg-phase0.sql` / `backend/migrations/*.sql`
- 现状参考：
  - `docs/review/data-platform-design.md`（需修订：`action` 进 `assets`、segment 子层级、`frame_set→frame`、`lifecycle_state.superseded` 由 revision 驱动）
  - `docs/review/schema-reference.md`（同上）
  - `docs/review/eval-metrics-design.md`（V1 三表分工对齐）
- 设计讨论全程：本仓库 `agent-transcripts/`（CYB-983）
- Layers SOP（待沉淀）：`docs/review/schema-model-layers.md`
