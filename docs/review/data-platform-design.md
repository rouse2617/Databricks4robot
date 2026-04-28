# 数据平台方案设计

本文是机器人多模态资产平台（Databricks4robot）的整体方案设计，包含背景、目标、架构、核心表与字段、数据同步机制、API、部署、可靠性、监控与实施计划。

---

## 1. 背景

- **数据形态**：MCAP 文件（Foxglove 容器格式，传感器 + 视频多模态时序），所有数据采集到算法处理都围绕 MCAP。
- **用户角色**：内部算法用户（生产消费）+ 外部客户（接收处理后产物）。
- **平台定性**：**数据资产化管理 + 处理与交付**。
- **机器人形态**：当前以 AV 切入，长期需覆盖机械臂、人形、四足、室内导航 —— 因此行业语义字段（city / weather / scenario_type）**不焊进主表**，统一进 tag 系统。

---

## 2. 问题现状

| 问题 | 现状 | 本方案如何解决 |
|------|------|----------------|
| 资产定义模糊 | 老 grace 系统把 video / 处理状态 / 派生产物揉在一起 | §5.1 重新明确 `mcap_file / asset / segment / file` 概念 |
| 算法用户手工轮代码 | 上下游就绪靠脚本轮询 PG | §5.6.2 outbox + 事件驱动，用户用 SDK 等事件 |
| 缺触发机制 | 没有"上游完成 → 自动触发下游" | §5.6.2 + Dagster job |
| Video 是最小粒度 | 业务逻辑全在 `videos/handlers.go` | §5.1 资产化 + §5.5 SDK 抽象，handler 退化为 CRUD |
| 无统一检索 | 多表 JOIN 才能查派生产物/标签/QA | §5.6.1 ES 投影 + §5.2 投影表 |
| 无多模态检索 | 找不到"相似片段" | §5.6.1 Phase 2+ 向量检索（VAI/Lance） |

典型业务问题与回答路径：

| 业务问题 | 答这条问题的路径 |
|----------|------------------|
| 某次训练用了哪些 asset | `training_runs` → `dataset_snapshots.manifest_uri` → Iceberg gold 明细 |
| 某算法版本变更后哪些 asset 要重刷 | Trino 查 Iceberg `gold_recompute_candidates` |
| 某 tag 何时被算法追加 | `asset_events` (event_type=tag_upserted) → Iceberg 长期 |
| 上月所有 MCAP segment 质量分布 | Trino + Iceberg silver/gold |
| 某客户交付能否完整回放 | PG `deliveries.replay_manifest_uri` + Iceberg delivery_items 明细 |

---

## 3. 目标

| 目标 | 落地承诺 |
|------|----------|
| **算法用户零感知底层** | Python SDK `grace_sdk`，`asset.get(id)` / `asset.stream(topic)` 一行拿到所有 |
| **资产即一等公民** | 主键 `asset_id`；`asset_tags / asset_algo_latest / asset_events` 投影 |
| **统一元数据底座** | PG 主库 + ES 检索 + Iceberg 历史；Catalog 抽象支持上云不绑定厂商 |
| **事件驱动数据链路** | `asset_events` outbox + Go worker，PG `LISTEN/NOTIFY` 实时唤醒 |
| **多维检索** | Tag 过滤 / 全文 / 向量（Phase 2+） |
| **强追溯/审计** | `asset_events` 全量事件流；`training_runs` 自包含 catalog 引用 |
| **MCAP 原生** | SDK 走 HTTP Range Request 流式读，不下整文件 |

---

## 4. 整体架构

### 4.1 平台能力总览

下图描述数据采集流程、算法处理与平台服务的整体关系：上游数据采集（grace 数采链路）经过 QA 后落地为有效 segment；触发算法处理；处理结果通过统一抽象服务暴露给算法用户、前端、外部交付。

![平台能力建设总览](./assets/architecture-overview.png)

### 4.2 架构演进路线（1.0 → 3.1）

下图描述数据平台从单库直连演进到统一元数据层的四个阶段：

![架构演进 1.0 → 3.1](./assets/architecture-evolution.png)

| 阶段 | 形态 | 关键变化 |
|------|------|----------|
| 1.0 | 业务层 + PostgreSQL | 单库直连，所有业务/分析共用 PG |
| 2.0 | + Iceberg + Elasticsearch + Trino/Spark | 引入湖仓与检索层，PG 只承担在线业务；分析与全文检索分流 |
| 3.0 | 全链路事件驱动 | 引入 outbox + 同步机制，PG 通过事件流推动 ES / Iceberg 派生；多消费者并行 |
| 3.1 | + 统一元数据层（Catalog 抽象） | 跨引擎对象中立注册（catalog_objects）+ 版本引用，业务表不再绑定物理路径或厂商 ID，支持上云不重构 |

当前位置：**2.0 → 3.0 之间**。
- ES 同步：当前 Dagster sensor 60s 轮询触发 + Spark 批同步，**计划切换为 outbox + Go worker + LISTEN/NOTIFY**（详见 §5.6.2）
- 湖仓同步：当前 Dagster sensor 触发，**计划切换为 schedule + 读 `asset_events.event_seq` 增量**
- Catalog 抽象：3.1 形态，Phase 2 落地

### 4.3 运行时数据流

```text
┌──────────────────────────────────────────────────────────────────────────┐
│                            算法用户 / 客户                                │
│   grace_sdk (Python)         前端 Web UI         外部交付 (manifest)      │
└────────────────────────────┬─────────────────────────────────────────────┘
                             │
                             ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                Backend (Go + Gin) — 单进程                                │
│  - REST API: assets / mcap / deliveries / search / lakehouse              │
│  - 中间件: 认证(X-Grace-Token) / Request-ID / 限流 / 熔断 / 幂等          │
│  - 写路径: 业务表 + asset_events (同事务)                                 │
│  - 读路径: PG 点查 + ES 检索 + Trino 湖仓查询                             │
└──────┬──────────────────────────┬──────────────────────────┬─────────────┘
       │                          │                          │
       ▼                          ▼                          ▼
┌──────────────┐         ┌────────────────┐         ┌─────────────────┐
│ PostgreSQL   │ NOTIFY  │ Outbox Worker  │  bulk   │ Elasticsearch   │
│ 主库 + outbox├────────►│ (Go goroutine) ├────────►│  index=assets   │
│              │ replay  │  消费 events   │         │                 │
└──────┬───────┘         └────────┬───────┘         └─────────────────┘
       │                          │
       │ Dagster schedule (5min)  │ 触发湖仓 sync
       ▼                          ▼
┌──────────────────────────────────────────────────────────────────────┐
│  Lakehouse: Iceberg REST Catalog + MinIO/S3 + Spark + Trino          │
│   bronze_*  →  silver_*  →  gold_*                                   │
│   gold_asset_search_docs / gold_training_samples / ...                │
└──────────────────────────────────────────────────────────────────────┘
                                  ↓ Phase 2+
                       Daft + Lance 多模态处理 / 向量检索
```

### 4.4 分层职责

| 层级 | 组件 | 职责 |
|------|------|------|
| 在线业务层 | PostgreSQL | 点查、事务、当前态筛选、状态机、权限、幂等；权威主库 |
| 检索层 | Elasticsearch | 模糊查询、全文检索、多字段过滤、facets、资产发现 |
| Catalog 控制面 | Iceberg REST Catalog + PG Platform Catalog | 湖表事务、metadata pointer、跨引擎对象的中立引用 |
| 湖仓层 | Iceberg | 历史事实、训练集、审计回放、统计分析、重算 |
| 查询层 | Trino | 查询 Iceberg，服务复杂分析与离线报表 |
| 计算层 | Spark / Dagster | 批处理、回填、特征抽取、重算、导出 |
| 多模态层（Phase 2+） | Daft / Lance | AI 多模态样本处理、高性能随机读取、向量/张量存储 |
| 异步派生通道 | Outbox + Worker | PG 主库变更 → ES / 湖仓 / 向量库的事件驱动同步 |

设计约定：PostgreSQL 表结构先保障在线业务，再通过事件流（`asset_events` outbox）支撑 ES 与 Iceberg；任何外部数据对象都通过中立 Catalog 引用而非物理路径绑定。

---

## 5. 详细设计

### 5.1 资产概念定义

资产化管理的核心：**给每一份回传数据创建唯一 `asset_id`，解析元信息形成资产；元数据 + 原始数据 + 关联数据三段式**。

#### 物理 vs 业务划分

```text
MCAP File (物理事实，不变)
   │
   │ CommitQA(segments, reviewer) — 一次 N 个有效片段
   ▼
Asset = Segment (业务事实，元数据可演化)
   │
   ├── tags          → 检索维度        (asset_tags 投影表)
   ├── algo_latest   → 算法当前态       (asset_algo_latest 投影表)
   ├── events        → 全量变更事件     (asset_events 事件表)
   ├── relations     → 上下游血缘       (asset_relations / parent_asset_id)
   └── files         → 派生文件引用     (assets.files JSONB)
```

> Bigtable 已退役（`STORAGE_BACKEND=bigtable` 在 `main.go` 已 fail-fast）；
> tag / algo / qa / lineage / event 五类语义都用 PG 关系型独立表承载。

#### 资产分层

| 概念 | 表 | 关系 |
|------|----|------|
| 物理文件 | `mcap_files` | 1 |
| 业务资产 | `assets` (asset_type=segment / clip / frame_set / derived_asset) | 1 → N segment |
| 资产血缘 | `assets.parent_asset_id` 或 `asset_relations` | N → N |
| 派生文件 | `assets.files` JSONB（key=algo@ver, value=URI） | 1 → N |

资产生命周期状态机（`assets.lifecycle_state`）：

```text
created → processing → ready → delivered → archived
              ↓             ↓
           rejected       superseded
```

| 状态 | 含义 |
|------|------|
| created | 资产刚建好，未开始处理 |
| processing | 算法/切分流水线在跑 |
| ready | 已就绪，可供检索/交付/训练 |
| rejected | QA 或算法判定不合格 |
| delivered | 已交付给至少一个客户 |
| archived | 进入冷存档，不参与在线检索 |
| superseded | 被新版本资产替代（rework） |

任何状态转移必须**同事务**追加一条 `asset_events`（event_type=`asset_lifecycle_changed`），否则审计链断裂。

---

### 5.2 资产字段设计

#### 5.2.1 表清单与上线优先级

| 表 | 优先级 | 主键 | 一句话职责 |
|----|--------|------|------------|
| `mcap_files` | 🟢 Tier 1 已运行 | `mcap_file_id` | 原始 MCAP 文件当前态 |
| `assets` | 🟢 Tier 1 已运行 | `asset_id` | 资产当前态（segment / clip / frame_set / derived_asset） |
| `deliveries` | 🟢 Tier 1 已运行 | `delivery_id` | 客户交付批次当前态 |
| `delivery_items` | 🟢 Tier 1 已运行 | `(delivery_id, asset_id)` | Delivery ↔ Asset M:N 明细 |
| `idempotency_keys` | 🟢 Tier 1 已运行 | `(scope, idem_key)` | API 幂等保护 |
| `asset_tags` | 🟡 Tier 2 上线前 | `(asset_id, tag_key)` | tag 当前态投影，驱动 facet/filter/ES 文档 |
| `asset_algo_latest` | 🟡 Tier 2 上线前 | `(asset_id, algo_name)` | 每 (asset, algo) 最新一行算法状态 |
| `asset_events` | 🟡 Tier 2 上线前 | `event_id`（UNIQUE `event_seq`） | 统一业务事件 / 审计 / outbox |
| `asset_relations` | 🟠 Tier 3 可选 | `(parent_asset_id, child_asset_id, relation_type)` | 多父 / 融合 / 拼接血缘 |
| `datasets` / `dataset_snapshots` | 🔵 Tier 4 Phase 2 | 见后 | 数据集定义与训练快照 |
| `training_runs` | 🔵 Tier 4 Phase 2 | `training_run_id` | 训练任务记录（自包含 catalog 引用） |
| `catalog_objects` / `catalog_object_versions` | 🔵 Tier 4 Phase 2 | 见后 | 中立对象与版本注册 |

⛔ **不做**：`feature_sets / feature_jobs / training_sample_exports` 字段未冻结，落地前重新评审。

#### 5.2.2 字段设计原则（关键四条）

1. **高频过滤字段必须列化**：`tenant_id / project_id / asset_type / lifecycle_state / start_timestamp_ns / end_timestamp_ns / duration_ms / created_at` 一律真实列。JSONB 只放低频扩展。
2. **行业语义不焊主表**：AV 的 `city / weather / scenario_type / quality_level` 必须进 `asset_tags`，由 `backend/config/tag_registry.yaml` 声明。机械臂/人形/四足以同样方式扩展，不需要改主表 schema。
3. **外部数据对象用 Catalog 引用**：训练任务 / 数据集快照 / 导出表都引用 `catalog_name + namespace + object_name + version_ref` 四元组，**不直接绑物理路径或厂商 ID**，避免上云锁死。
4. **多租户硬约束**：`tenant_id / project_id` 现阶段 `NOT NULL DEFAULT '_default'`，启用多租户时去 DEFAULT + 加 RLS。避免 NULL 历史包袱。

#### 5.2.3 mcap_files —— 原始 MCAP 文件当前态

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| mcap_file_id | UUID | 是 | 文件主键 |
| raw_hash_md5 | TEXT | 否 | 文件 MD5；`is_deleted=FALSE` 范围 UNIQUE，重复 ingest 走幂等 |
| raw_hash_sha256 | TEXT | 否 | 长期内容指纹 |
| mcap_uri | TEXT | 是 | MCAP 对象存储地址 |
| size_bytes | BIGINT | 否 | 文件大小 |
| file_duration_ms | BIGINT | 否 | 文件总时长 |
| start_timestamp_ns / end_timestamp_ns | BIGINT | 否 | 文件起止时间 |
| channel_count / chunk_count | INT | 否 | MCAP 结构摘要 |
| ingest_state | TEXT | 是 | pending / ingesting / ready / failed |
| vendor_id / collector_id / task_id / device_id | TEXT | 否 | 采集 provenance |
| camera_model / data_source / location_id / scene_id / environment_id / collection_method | TEXT | 否 | 采集环境 |
| tenant_id / project_id | TEXT | 否 | 多租户 |
| owner | TEXT | 否 | 数据归属方 |
| retention_tier | TEXT | 否 | hot / warm / cold / archive |
| expire_at | TIMESTAMPTZ | 否 | 过期/可清理时间 |
| metadata | JSONB | 是 | 低频扩展元数据 |
| process_state | JSONB | 是 | 文件级处理状态扩展 |
| is_deleted | BOOLEAN | 是 | 软删除 |
| created_at / updated_at | TIMESTAMPTZ | 是 | 时间戳 |
| version | BIGINT | 是 | 乐观锁版本 |

#### 5.2.4 assets —— 资产当前态

行业相关 facet（city / weather / scenario_type 等）**不进本表**，统一进 `asset_tags`。

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| asset_id | UUID | 是 | 资产主键 |
| mcap_file_id | UUID | 是 | 来源 MCAP 文件 ID |
| asset_type | TEXT | 是 | segment / clip / frame_set / derived_asset |
| storage_uri / thumb_uri | TEXT | 否 | 资产/缩略图地址 |
| parent_asset_id / root_asset_id | UUID | 否 | 父/根资产 ID（普通切分用） |
| asset_level | INT | 是 | 资产层级（0=原始） |
| split_method / split_algo_name / split_algo_version / split_run_id / split_reason | TEXT | 否 | 切分 provenance |
| segment_index | INT | 否 | 父资产下片段序号 |
| parent_start_offset_ms / parent_end_offset_ms | BIGINT | 否 | 相对父资产偏移 |
| start_timestamp_ns / end_timestamp_ns / duration_ms | BIGINT | 否 | 时间范围 |
| lifecycle_state | TEXT | 是 | created / processing / ready / rejected / delivered / archived / superseded |
| tenant_id / project_id | TEXT | 否 | 多租户 |
| owner / reviewer | TEXT | 否 | 资产 owner / 审核人 |
| last_delivered_at / last_delivered_to / delivery_count | — | 否 | 交付汇总（冗余，由 delivery usecase 同事务刷新） |
| retention_tier / expire_at | — | 否 | 生命周期 |
| metadata / files | JSONB | 是 | 低频扩展 / 关联文件（key=algo@ver） |
| is_deleted | BOOLEAN | 是 | 软删除 |
| created_at / updated_at | TIMESTAMPTZ | 是 | 时间戳 |
| version | BIGINT | 是 | 乐观锁版本 |

#### 5.2.5 asset_tags —— tag 当前态投影

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| asset_id | UUID | 是 | 资产 ID |
| tag_key | TEXT | 是 | tag 名（`tag_registry.yaml` 注册） |
| tag_value | TEXT | 是 | 字符串值 |
| tag_value_num | DOUBLE PRECISION | 否 | 数值型，用于范围过滤 |
| tag_value_bool | BOOLEAN | 否 | 布尔型 |
| tag_type | TEXT | 是 | string / number / bool / enum |
| source_type / source_name / source_version | TEXT | 是/否 | human / algo / rule / system + 来源版本 |
| run_id | TEXT | 否 | 外部批次 |
| confidence | DOUBLE PRECISION | 否 | 置信度 |
| tenant_id / project_id | TEXT | 否 | 多租户（冗余，便于 RLS） |
| created_at / updated_at | TIMESTAMPTZ | 是 | 时间戳 |

#### 5.2.6 asset_algo_latest —— 算法最新状态投影

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| asset_id | UUID | 是 | 资产 ID |
| algo_name / algo_version | TEXT | 是 | 算法标识 |
| status | TEXT | 是 | pending / running / ok / failed / blocked / skipped |
| result_tag / result_score | TEXT / DOUBLE | 否 | 算法输出标签与分数 |
| result_summary | JSONB | 是 | 低频结果摘要 |
| run_id / method | TEXT | 否 | 执行批次 / 方式 |
| model_uri / output_uri | TEXT | 否 | 模型 / 输出地址 |
| error_code / error_message | TEXT | 否 | 失败原因 |
| started_at / finished_at | TIMESTAMPTZ | 否 | 起止时间 |
| tenant_id / project_id | TEXT | 否 | 多租户（冗余） |
| updated_at | TIMESTAMPTZ | 是 | 更新时间 |

完整算法生命周期写入 `asset_events`，本表仅留每 (asset, algo) 最新一行。

#### 5.2.7 asset_events —— 统一业务事件 / 审计 / outbox

消费者必须按 `event_seq` 严格递增推进 watermark，**不能用 `occurred_at`**（并发写入时间戳会冲突）。

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| event_id | UUID | 是 | 事件主键 |
| event_seq | BIGSERIAL | 是 | 单调递增序号，UNIQUE；消费 watermark 用 |
| event_type | TEXT | 是 | 事件类型（见下） |
| payload_schema_version | TEXT | 是 | event_payload schema 版本（v1 / v2 …） |
| asset_id / mcap_file_id | UUID | 否 | 关联实体 |
| tenant_id / project_id | TEXT | 否 | 多租户 |
| event_source | TEXT | 是 | backend / worker / dagster / spark / system |
| actor_type / actor_id | TEXT | 否 | user / service / algo / system + 操作者 |
| request_id / idempotency_key / run_id | TEXT | 否 | 追踪 / 幂等 / 批次 |
| occurred_at | TIMESTAMPTZ | 是 | 业务发生时间 |
| created_at | TIMESTAMPTZ | 是 | 入库时间 |
| publish_state | TEXT | 是 | pending / published / failed |
| published_at | TIMESTAMPTZ | 否 | 同步完成时间 |
| event_payload | JSONB | 是 | 类型相关字段（按 `payload_schema_version` 解析） |

典型事件类型：
`mcap_ingested` · `asset_created` · `asset_updated` · `asset_lifecycle_changed` · `tag_upserted` · `tag_deleted` · `algo_started` · `algo_finished` · `algo_failed` · `delivery_created` · `delivery_item_added` · `delivery_completed` · `dataset_snapshot_created` · `training_run_started` · `training_run_finished`

物理治理：单表行数超过 1000 万后按 `occurred_at` 月分区；retention 默认 90 天，过期后只在 Iceberg 留档。

#### 5.2.8 deliveries —— 客户交付批次当前态

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| delivery_id | UUID | 是 | 交付批次主键 |
| tenant_id / project_id | TEXT | 否 | 多租户 |
| customer_id / contract_id | TEXT | 是/否 | 客户 / 合同 |
| delivery_type | TEXT | 是 | asset_set / replay / dataset / … |
| status | TEXT | 是 | pending / delivered / accepted / rejected / recalled |
| requested_by / approved_by / delivered_by | TEXT | 否 | 流程参与者 |
| manifest_uri / replay_manifest_uri | TEXT | 否 | 交付清单 / 回放清单 |
| item_count / total_size_bytes | BIGINT | 是/否 | 数量与体积 |
| delivered_at / completed_at | TIMESTAMPTZ | 否 | 时间节点 |
| metadata | JSONB | 是 | 低频扩展 |
| is_deleted | BOOLEAN | 是 | 软删除 |
| created_at / updated_at | TIMESTAMPTZ | 是 | 时间戳 |
| version | BIGINT | 是 | 乐观锁版本 |

#### 5.2.9 delivery_items —— Delivery ↔ Asset 明细

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| delivery_id | UUID | 是 | 交付批次 ID |
| asset_id | UUID | 是 | 资产 ID |
| asset_version | BIGINT | 否 | 交付时资产版本（快照） |
| item_state | TEXT | 是 | pending / delivered / failed |
| checksum | TEXT | 否 | 导出文件校验 |
| export_uri | TEXT | 否 | 导出对象地址 |
| created_at | TIMESTAMPTZ | 是 | 创建时间 |

#### 5.2.10 idempotency_keys —— API 幂等

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| scope | TEXT | 是 | 幂等域（如 `deliveries.create`） |
| idem_key | TEXT | 是 | 客户端提供的幂等 key |
| resource_type / resource_id | TEXT | 否 | 资源指代 |
| request_hash | TEXT | 否 | 请求 hash |
| response_json | JSONB | 否 | 首次响应缓存 |
| status_code | INT | 否 | 首次响应状态码 |
| created_at | TIMESTAMPTZ | 是 | 创建时间 |
| expires_at | TIMESTAMPTZ | 否 | 过期时间（lifecycle job 清理） |

#### 5.2.11 Phase 2+ 表（字段简表）

> 本期暂不落地，列出主要字段以备评审；详细落地前会发独立 ADR。

**`asset_relations`** — 复杂血缘（多父 / 融合 / 拼接 / 采样）：
`parent_asset_id / child_asset_id / relation_type / method / algo_name / algo_version / run_id / parent_start_offset_ms / parent_end_offset_ms / created_at`

**`datasets`** — 数据集定义父表：
`dataset_id / name / description / owner / tenant_id / project_id / dataset_type / status / created_by / created_at / updated_at`

**`dataset_snapshots`** — 数据集快照（PK `(dataset_id, snapshot_id)`，**被引用后禁止 DELETE**）：
`dataset_id / snapshot_id / snapshot_version / created_by / query_spec / source_query_hash / source_catalog_name / source_namespace / source_object_name / source_object_version_ref / manifest_uri / item_count / status / created_at / updated_at`

**`training_runs`** — 训练任务记录（**软引用 dataset_snapshots，自包含 catalog 引用**）：
`training_run_id / tenant_id / project_id / dataset_id / snapshot_id / model_name / model_version / algo_name / code_version / config_uri / data_manifest_uri / data_catalog_name / data_namespace / data_object_name / data_object_version_ref / status / metrics / artifact_uri / started_at / finished_at / created_at / updated_at`

**`catalog_objects`** — 中立对象注册（UNIQUE `(catalog_name, namespace, object_name, object_type)`）：
`object_id / catalog_name / namespace / object_name / object_type / provider / format / storage_uri / external_ref / owner / tenant_id / project_id / description / tags / properties / status / created_at / updated_at`

**`catalog_object_versions`** — 对象版本引用：
`object_version_id / object_id / version_ref / version_type / schema_ref / manifest_uri / row_count / size_bytes / checksum / created_by / created_at / properties`

---

### 5.3 数据流与写路径

```text
Backend HTTP handler
   │
   ▼
usecase 层
   │
   ▼ BEGIN TRANSACTION
   ├── 写主表 (assets / asset_tags / asset_algo_latest / deliveries / …)
   ├── 写交付汇总冗余 (assets.last_delivered_at / delivery_count)
   ├── 写 asset_events (event_seq via BIGSERIAL, publish_state='pending')
   └── pg_notify('asset_events', event_seq)
   ▼ COMMIT
```

关键约定：
- **乐观锁**：`assets.version` 走 CAS（`postgres.AssetRepo.Set`），冲突返回 `409 CONCURRENT_CONFLICT`。
- **事件强一致**：业务写 + 事件写在同一事务，COMMIT 之后才发 NOTIFY；事件不依赖 trigger（避免业务逻辑下沉到 PG）。
- **审计强一致**：`audit.Log` 用 `Sink` 接口注入，PG backend 下 `postgres.AuditSink` 写 `audit_events`。

---

### 5.4 主库选型

#### 当前结论：PostgreSQL only

| 对比项 | PostgreSQL | Bigtable | Spanner / TiDB（未来） |
|--------|------------|----------|------------------------|
| 事务 | 强 ✅ | 弱（单行原子） | 强 |
| 关系约束 | 原生 FK / UNIQUE / CHECK ✅ | 应用层维护 | 原生 |
| 多条件筛选 | 索引 + WHERE ✅ | 需索引表 | SQL ✅ |
| 写吞吐 | 中高（够 < 10k QPS） | 极强 | 高 |
| 运维 | 低 ✅ | 中（GCP 托管） | 中 |
| 上云锁定 | 无 ✅ | GCP 绑定 | Spanner=GCP / TiDB=自托管 |

**为什么不选 Bigtable**：
1. 项目目前不强绑 GCP，要保留多云可移植性
2. 当前规模（< 100M assets）单实例 PG 足够
3. Bigtable 弱事务 + 弱关系约束，让 `asset_tags / asset_events / training_runs` 这类强引用关系实现起来反而更难
4. 业务真正爆量到 PB 级时再评估 Spanner / TiDB / 切片，**不必现在做**

历史代码：`backend/internal/bigtable/*` 保留为参考，运行时设置 `STORAGE_BACKEND=bigtable` 时进程 fail-fast，禁止启动。

---

### 5.5 DataLayer 抽象 (SDK)

目标：算法代码**不出现物理路径，不出现 Bigtable/PG/GCS 字样**。

```python
import grace_sdk

client = grace_sdk.Client()  # 认证 / 路由全在内部

# 检索
asset = client.get_asset("uuid-1234")
print(asset.tags)  # ['urban', 'rainy']

# 流式读 MCAP segment（HTTP Range Request，不下载整文件）
for frame in asset.stream_video_frames(topic="/camera/front/image"):
    model.predict(frame.image)

# 回写派生产物（SDK 内部解析为对象存储 URI + 注册到 assets.files）
asset.add_derived_file(kind="sam2_mask", file_path="./output.json")
```

抽象层职责（按重要性）：
1. **Asset-Centric I/O**：所有操作收敛于 `asset_id`，物理文件降级为挂载属性
2. **Virtual Namespace**：抹平多云路径（`/cyber/{biz}/{algo}/t1.mcap` → 路由解析为 `gs://...` / `s3://...`），通过 `catalog_objects` 路由表
3. **最小权限隔离**：屏蔽 AK/SK，按 tenant/algo 维度颁发短期 token

实现位置：仓库 `sdk/` 目录（Python + httpx + pydantic）。

---

### 5.6 数据索引与同步

#### 5.6.1 索引引擎选型

| 阶段 | 引擎 | 场景 | 状态 |
|------|------|------|------|
| Phase 1 | **Elasticsearch** | Tag 复杂组合过滤 / 全文检索 / facets | ✅ 已接入 |
| Phase 2+ | **Vertex AI Vector Search** 或 **Lance** | 多模态语义检索（以图搜图、文本搜片段） | 🔵 待启动 |

主库存向量 ID，真实 embedding 在向量引擎做 ANN。两阶段共用同一套 outbox 同步机制。

#### 5.6.2 数据同步机制

**核心原则**：PG 是权威主库，ES / 湖仓 / 向量库都是异步派生。**不轮询**，用 transactional outbox。

##### 设计

```text
┌──────────────────────┐
│ Backend write path   │
│  写业务表 + 写        │
│  asset_events        │
│  pg_notify(event_seq)│ 同事务
└──────────┬───────────┘
           │
           ▼
┌──────────────────────────────────────┐
│  PostgreSQL                          │
│  asset_events (publish_state=pending)│
└──────────┬───────────────────────────┘
           │
   ┌───────┴────────┬──────────────────┬──────────────┐
   │ NOTIFY 唤醒    │ NOTIFY 唤醒       │ schedule     │
   ▼                ▼                   ▼              │
┌────────────┐  ┌────────────┐  ┌────────────────┐    │
│Worker-ES   │  │Worker-Vec  │  │Dagster (5min)  │    │
│Go goroutine│  │(Phase 2+)  │  │schedule cron   │    │
│SELECT ...  │  │            │  │读 max event_seq │    │
│FOR UPDATE  │  │            │  │PG → Bronze →    │    │
│SKIP LOCKED │  │            │  │Silver → Gold    │    │
└─────┬──────┘  └─────┬──────┘  └────────┬───────┘    │
      ▼                ▼                  ▼            │
   ES _bulk        VAI / Lance        Iceberg          │
                                                       │
   COMMIT: UPDATE asset_events SET publish_state='published' ◄┘
```

##### 关键约定

| 维度 | 实现 |
|------|------|
| **入口** | backend usecase 在状态变更时同事务追加 `asset_events`，带 `event_seq BIGSERIAL UNIQUE` 和 `payload_schema_version` |
| **传输** | PG `LISTEN/NOTIFY`（亚秒级唤醒）+ outbox 持久化（兜底重放） |
| **消费并发** | `SELECT ... FOR UPDATE SKIP LOCKED LIMIT 500` 多 worker 实例并发 |
| **顺序** | **严格按 `event_seq` 推进**，不依赖 `occurred_at`（并发写入时间戳会冲突） |
| **payload 版本** | 每种 event_type 对应 `schemas/events/<event_type>.v<n>.json`；新增字段 minor，破坏性变更 major |
| **幂等** | ES bulk 用 `index` action + doc_id=asset_id；Iceberg 用 idempotency_key 去重 |
| **可观测** | `publish_state='pending'` 队列长度 → SLO；`max(event_seq) - watermark` → 延迟指标 |
| **partition** | `asset_events` 行数 > 1000 万后按 `occurred_at` 月分区，retention 默认 90 天 |

##### 延迟预期

| 目标 | 现状（sensor 60s 轮询） | Outbox + NOTIFY 后 | Phase 2 上 Debezium |
|------|------------------------|---------------------|---------------------|
| Elasticsearch | ≥ 60s + Spark 冷启 + 手动 materialize | < 1s | < 1s |
| Iceberg Bronze | ≥ 60s + Spark 冷启 | 5min（schedule） | < 30s |
| 向量库 | n/a | < 10s（embedding 异步 batch） | < 10s |

##### 为什么不直接上 Debezium / Kafka

- 当前规模用不上
- Kafka + connect cluster + schema registry 是过度工程
- 持续 > 1k events/s 或 ES 之外有 ≥ 3 个消费者时再切（对应实施计划中的 Phase 2 SLA）

##### 为什么不让 backend 直接同步写 ES

- 违反"PG 权威 + ES 异步派生"原则
- ES 慢/故障会拖死 API
- 事务里调外部 IO 是反模式

##### 为什么不只用 LISTEN/NOTIFY

- listener 断线期间通知**永久丢失**，无 replay
- NOTIFY 只是"快速唤醒"，outbox 才是"持久化保底"，两者互补不冲突

---

### 5.7 数据可视化（TODO）

候选：[Webviz](https://github.com/cruise-automation/webviz) / Foxglove Studio 嵌入。
集成方式：前端按 `asset_id` 拉取 manifest，传给 viewer 做 segment 在线预览。

不在 MVP 范围。

---

### 5.8 平台 API 设计

API 设计约定：

| 维度 | 约定 |
|------|------|
| Auth | `X-Grace-Token`（短期），后续切 OIDC / mTLS |
| Tracing | `X-Request-ID` 中间件，全链路串联 |
| Idempotency | `POST /api/v1/deliveries` 必须带 `Idempotency-Key` |
| 乐观锁 | `PATCH /api/v1/assets/{id}` 冲突返回 `409 CONCURRENT_CONFLICT`，客户端重试 |
| 错误格式 | 统一 envelope：`{ code, message, request_id, details }` |
| 分页 | `page / page_size / next_token` |

主要 endpoint 清单（v1）：

| Group | Method + Path | 用途 |
|-------|---------------|------|
| Asset | `GET /api/v1/assets` | 资产列表（基础筛选 + 分页） |
| Asset | `GET /api/v1/assets/{id}` | 资产详情 |
| Asset | `PATCH /api/v1/assets/{id}` | 更新资产（带乐观锁 version） |
| Asset | `POST /api/v1/assets/{id}/algo/{algo}/start` | 触发算法运行 |
| Asset | `POST /api/v1/assets/{id}/algo/{algo}/finish` | 算法完成回写 |
| Asset | `POST /api/v1/assets/{id}/tags` | upsert tag |
| Asset | `DELETE /api/v1/assets/{id}/tags/{key}` | 删除 tag |
| MCAP | `POST /api/v1/mcap` | 注册 MCAP 文件 |
| MCAP | `GET /api/v1/mcap/{id}` | MCAP 详情 |
| MCAP | `GET /api/v1/mcap` | MCAP 列表（按 ingest_state / owner 过滤） |
| Delivery | `POST /api/v1/deliveries` | 创建交付（必填 `Idempotency-Key`） |
| Delivery | `GET /api/v1/deliveries/{id}` | 交付详情 |
| Delivery | `POST /api/v1/deliveries/{id}/items` | 添加交付明细 |
| Delivery | `POST /api/v1/deliveries/{id}/complete` | 完成交付 |
| Search | `GET /api/v1/search/assets` | ES 复杂检索 + facets，ES 故障时 fallback PG |
| Lakehouse | `GET /api/v1/lakehouse/training-snapshots` | 训练快照查询（Trino） |
| Lakehouse | `GET /api/v1/lakehouse/recompute-candidates` | 重算候选（Trino） |
| Health | `GET /healthz` / `GET /readyz` | K8s 探针 |

---

## 6. 服务部署资源

### 6.1 本地开发

```bash
make all-up    # docker-compose: PG / MinIO / Iceberg REST / Spark / Trino / ES / Backend / Frontend
make all-down
make all-logs  # 查看任一服务日志
```

依赖容器：PostgreSQL 16、Elasticsearch 8.x、MinIO（S3 兼容）、Iceberg REST Catalog、Spark 3.5、Trino、Dagster、Backend、Frontend。本地开发默认全套自动起，资源占用约 6 GB RAM。

### 6.2 生产部署形态（建议）

| 组件 | 形态 | 备注 |
|------|------|------|
| Backend (Go) | Kubernetes Deployment，HPA 按 CPU 扩 | 单镜像，`STORAGE_BACKEND=postgres` |
| Outbox Worker | Backend 进程内 goroutine（Phase 1）→ 独立 Deployment（Phase 2+） | 多实例靠 `FOR UPDATE SKIP LOCKED` 协调 |
| PostgreSQL | 云托管（CloudSQL / RDS / Aliyun RDS） | 主从 + PITR；启用 logical replication 为 Phase 2 Debezium 准备 |
| Elasticsearch | 云托管（Elastic Cloud / 阿里云 ES） | 单 cluster，按 tenant 走 routing |
| Iceberg | REST Catalog (Tabular / Polaris / 自建) + S3/GCS warehouse | 上云时把 catalog 切到 Polaris/Glue/DLF，业务表不动 |
| Trino | 容器化，按需扩 worker | |
| Dagster | webserver + daemon，K8s deployment | schedule + 少量 sensor |
| 多模态（Phase 2+） | Daft on Ray Cluster + Lance on S3 | 独立子集群 |

---

## 7. 冗余与可靠性

| 故障域 | 影响 | 应对 |
|--------|------|------|
| Backend 单实例挂 | API 不可用 | 至少 3 副本 + K8s 健康检查 |
| PG 单点故障 | 全平台不可写 | 云托管主从 + 自动故障转移 + PITR；下游 ES/Iceberg 异步派生不影响主库可用性 |
| ES 故障 | 检索降级，列表页用 PG fallback | `search` handler 已实现 graceful degradation |
| Outbox worker 全挂 | 事件堆积，ES/湖仓延迟 | `publish_state='pending'` 监控 + 自动重启；事件不丢（持久化在 PG） |
| Dagster 故障 | 入湖延迟 | 短期手动 materialize；长期切 Debezium |
| 对象存储故障 | 文件读写失败 | 多 region 副本（云端） |
| Iceberg metadata 损坏 | 历史查询失败 | snapshot 多副本 + Polaris/Glue 备份 |

数据层面：
- **同事务保证**：业务变更 + 事件追加原子；不会出现"业务写了事件没写"
- **事件重放**：从任意 `event_seq` 重新消费，可重建 ES / 湖仓
- **快照不可硬删**：`dataset_snapshots` 被引用后只能 archived，审计链不断
- **训练记录自包含**：`training_runs` 不依赖 catalog FK，跨系统迁移零成本

---

## 8. 监控告警

### 8.1 必备指标

| 类别 | 指标 | 阈值建议 |
|------|------|----------|
| API | P99 latency / error rate / RPS | P99 < 500ms / errors < 1% |
| API | 限流 / 熔断触发数 | 任何触发都告警 |
| PG | TPS / 连接数 / 复制延迟 / 长事务 | 长事务 > 30s 告警 |
| Outbox | `publish_state='pending'` 队列长度 | > 10k 告警 |
| Outbox | event_seq lag（max - watermark） | > 5min 告警 |
| ES | 索引延迟 / cluster health | yellow 30min / red 立刻 |
| ES | bulk error rate | > 0.1% 告警 |
| Dagster | job 失败 / schedule miss | 任何失败告警 |
| Iceberg | snapshot 数量 / metadata 大小 | snapshot > 1000 提示 compact |
| 对象存储 | 4xx/5xx rate / 流量异常 | 模型 anomaly |

### 8.2 业务级 SLO

- API 可用性 ≥ 99.9%（月度）
- 资产从写入到 ES 可见 ≤ 2s（P99）
- 资产从写入到 Iceberg 可见 ≤ 10min（schedule 周期）
- 交付幂等性 100%（同 `Idempotency-Key` 必返回首次结果）

---

## 9. 实施计划

### 9.1 Phase 0（已完成）

- [x] Backend 单进程 + 分层架构（handler → usecase → repository）
- [x] PG schema：`mcap_files / assets / deliveries / delivery_items / asset_algo_events / idempotency_keys`
- [x] Bigtable runtime 弃用（`STORAGE_BACKEND=bigtable` fail-fast）
- [x] OpenSearch → Elasticsearch 迁移
- [x] Audit 模块解耦（Sink 接口）
- [x] `assets.version` CAS 乐观锁 → 409
- [x] 目标 schema 蓝图与可执行 DDL
- [x] 全表字段速查与上线优先级

### 9.2 Phase 1（上线前必须，1–3 个迭代）

按依赖顺序：

1. **Schema 字段提升**
   - `assets / mcap_files` 加 `tenant_id / project_id / asset_type / lifecycle_state / end_timestamp_ns / duration_ms / owner / retention_tier / expire_at`
   - `tenant_id NOT NULL DEFAULT '_default'`
   - 现有数据 backfill

2. **新建 `asset_events` 表**（带 `event_seq BIGSERIAL UNIQUE` + `payload_schema_version`）
   - DDL 迁移
   - schemas/events/*.json schema registry 雏形

3. **Backend 写路径接 outbox**
   - `asset / mcap / delivery / tag / algo` 所有 mutation 同事务追加 events
   - `pg_notify` 唤醒
   - 单元测试覆盖

4. **新建投影表 `asset_tags / asset_algo_latest`** + 进入双写期
   - 写 cf_tag/cf_algo 同时 upsert 投影表
   - cf_* 历史数据 backfill
   - 前端 facet/filter 切到投影表 + `tag_registry.yaml`

5. **Outbox Worker（ES 同步）**
   - Backend 进程内 goroutine（最简）
   - LISTEN/NOTIFY + 1s tick 兜底
   - `FOR UPDATE SKIP LOCKED` 并发安全
   - 监控指标接 Prometheus

6. **Dagster 改造**
   - sensor → schedule（cron `*/5 * * * *`）
   - 修 `lakehouse_sync_job` 包含 `gold_to_elasticsearch`（已知 group 漏配）
   - 增量键改读 `asset_events.event_seq`

7. **`assets.lifecycle_state` 切换**
   - 与 `status` 双写
   - 前端列表过滤切到 `lifecycle_state`
   - 老 `status` 退役

8. **可观测性**
   - Prometheus + Grafana dashboard（API / Outbox / PG / ES）
   - 关键告警接 PagerDuty 或同等

### 9.3 Phase 2+（按业务节奏）

| 能力 | 触发条件 |
|------|----------|
| `datasets / dataset_snapshots / training_runs` | 第一个真训练任务接入 |
| `catalog_objects / catalog_object_versions` | 出现多 provider 数据对象 |
| Vector search (VAI / Lance) | 多模态检索成为核心需求 |
| Debezium / Kafka | 写入持续 > 1k events/s 或 ES 之外 ≥ 3 消费者 |
| Daft + Lance 多模态处理 | 大规模训练样本物理格式优化 |
| 多租户 RLS 启用 | 第一个外部租户接入 |

### 9.4 不做（明确排除）

- ❌ 在 PG 重写血缘系统（用 OpenLineage / Marquez）
- ❌ 在 PG 重写权限引擎（用 Backend RBAC + Cloud IAM / Ranger）
- ❌ 在 PG 重写质量系统（用 Dagster asset checks / Great Expectations）
- ❌ 短期上 Bigtable / Spanner（PG 够用）
- ❌ 短期上 Kafka / Debezium（outbox 够用）
- ❌ `feature_sets / feature_jobs / training_sample_exports`（字段未冻结）

---

## 10. 风险与未决事项

| 编号 | 议题 | 影响 | 当前默认 | 拍板方 |
|------|------|------|----------|--------|
| R1 | 业务表如何引用 `catalog_objects` | 训练审计可移植性 | 软引用四元组 | 待 ADR（Phase 2 落地前） |
| R2 | Outbox worker 部署形态（进程内 vs 独立 service） | 运维复杂度 | Phase 1 进程内，Phase 2 独立 | Phase 2 切换前评审 |
| R3 | `lifecycle_state` 的 CHECK 约束 | 状态机非法转移 | 应用层校验 | 状态稳定后加 DB 约束 |
| R4 | 多租户启用时机 | RLS / index routing | 单租户 `_default` | 第一个外部租户前定 |
| R5 | 上云 Catalog 选型（Polaris vs Gravitino vs 云原生） | 跨引擎事务 / 元数据 | 暂不绑定，靠 catalog_objects 抽象 | 上云前评审 |
| R6 | PII / GDPR 删除链路 | 合规 | 软删 + retention_tier 标注 | 单独治理文档跟进 |

