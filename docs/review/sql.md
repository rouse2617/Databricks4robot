# SQL Companion

> **作用定位**：这篇文档只保留 **schema / DDL / 字段命名** 相关内容，给 `schemas/pg-phase0.sql`、仓库内历史注释和代码 review 提供一个稳定锚点。
>
> **主设计已合并**：架构分层、ES / Iceberg / Outbox Worker、运维、阶段计划等内容，统一收敛到 [`data-platform-design.md`](./data-platform-design.md) 与 [`schema-reference.md`](./schema-reference.md)。本文件不再重复那部分长篇设计叙事。

---

## 0. 当前实现 vs 目标 Schema

当前评审基线以 `docs/review/README.md` 为准：

- **1.0 当前态**：运行时只有 PostgreSQL + Backend 在线。
- **主写入路径**：`assets` 标量列 + `asset_tags` / `asset_algo_latest` 投影表 + `asset_events` 统一事件表。
- **兼容层**：`cf_meta / cf_algo / cf_tag / cf_files` 仅保留历史兼容 / 回滚价值，不再是主写入路径。
- **2.0 起**：才启用 Outbox Worker，把 `asset_events` 同步到 ES / Iceberg / 其他 sink。

### 0.1 本文件与其他文档的分工

| 文档 | 用途 |
|------|------|
| [`data-platform-design.md`](./data-platform-design.md) | 架构、事件流、同步机制、部署、SLO、阶段计划 |
| [`schema-reference.md`](./schema-reference.md) | 人类可读的字段速查、上线优先级、索引与约束摘要 |
| [`sql.md`](./sql.md) | DDL 伴随文档：保留 section 编号，解释为什么这些表/列存在 |
| [`../../schemas/pg-phase0.sql`](../../schemas/pg-phase0.sql) | 当前 PG 参考 DDL，机器可读视图 |

### 0.2 表状态速览

| 表 / 能力 | 当前状态 | 备注 |
|-----------|----------|------|
| `mcap_files` / `assets` / `deliveries` / `delivery_items` / `idempotency_keys` | 已上线 | 在线业务主表 |
| `asset_tags` / `asset_algo_latest` | 已上线 | 当前态投影，服务筛选与详情回填 |
| `asset_events` | 已上线 | 统一业务事件表；1.0 只写不消费 |
| `asset_algo_events` | 遗留只读 | 历史兼容，不再新增写入 |
| `datasets` / `dataset_snapshots` / `training_runs` | 目标表 | 训练平台真正接入时启用 |
| `catalog_objects` / `catalog_object_versions` | 目标表 | 中立 Catalog 抽象，按需启用 |

---

## 1. 表族与职责

| 表族 | 说明 |
|------|------|
| 在线主表 | `mcap_files`、`assets`、`deliveries`、`delivery_items`、`idempotency_keys` |
| 当前态投影 | `asset_tags`、`asset_algo_latest` |
| 事件 / 审计 / outbox | `asset_events` |
| 训练 / 数据集 | `datasets`、`dataset_snapshots`、`training_runs` |
| Catalog 抽象 | `catalog_objects`、`catalog_object_versions` |

---

## 2. 关系总览

- `mcap_files 1 -> N assets`
- `assets 1 -> N asset_tags`
- `assets 1 -> N asset_algo_latest`
- `assets 1 -> N asset_events`
- `assets N <-> N deliveries`，通过 `delivery_items`
- `datasets 1 -> N dataset_snapshots`
- `dataset_snapshots 1 -> N training_runs`
- `catalog_objects 1 -> N catalog_object_versions`

---

## 3. 字段设计原则

### 3.1 高频过滤字段必须列化

以下字段属于在线筛选 / 排序 / 聚合高频路径，不能长期躲在 JSONB：

- `tenant_id`
- `project_id`
- `asset_type`
- `lifecycle_state`
- `start_timestamp_ns`
- `end_timestamp_ns`
- `duration_ms`
- `owner`
- `retention_tier`
- `expire_at`
- `created_at`
- `updated_at`

### 3.2 当前态与历史分离

- **当前态**：读 `assets` + `asset_tags` + `asset_algo_latest`
- **历史 / 审计 / 回放**：读 `asset_events`
- **兼容 JSONB**：只保留旧路径兼容，不再承担当前态权威职责

### 3.3 事件是业务事实的一部分

- 任何重要 mutation 都必须在**同一事务**里追加 `asset_events`
- 下游一律按 `event_seq` 推进，不能用 `occurred_at` 当 watermark
- `payload_schema_version` 从第一天开始就是必填，不允许后补

### 3.4 Catalog 引用优先于物理路径绑定

训练、导出、湖表、搜索索引、多模态 dataset 等跨系统对象，优先用：

- `catalog_name`
- `namespace`
- `object_name`
- `object_type`
- `provider`
- `version_ref`

`storage_uri` 只做运维定位，不做业务主键。

---

## 4. 主表字段详情

### 4.1 mcap_files（原始 MCAP 文件当前态，每个文件可切分多个 asset）

| 字段名             | 类型        | 必填 | 说明                | 同步目标        |
|--------------------|-------------|------|---------------------|-----------------|
| mcap_file_id       | UUID        | 是   | 文件主键            | ES / Iceberg    |
| raw_hash_md5       | TEXT        | 否   | 原始文件 hash       | Iceberg         |
| raw_hash_sha256    | TEXT        | 否   | 原始文件 SHA256，用于长期唯一校验 | Iceberg |
| mcap_uri           | TEXT        | 是   | MCAP 对象存储地址   | Iceberg         |
| size_bytes         | BIGINT      | 否   | 文件大小            | ES / Iceberg    |
| file_duration_ms   | BIGINT      | 否   | 文件总时长          | ES / Iceberg    |
| start_timestamp_ns | BIGINT      | 否   | 文件起始时间        | ES / Iceberg    |
| end_timestamp_ns   | BIGINT      | 否   | 文件结束时间        | ES / Iceberg    |
| channel_count      | INT         | 否   | MCAP channel 数     | Iceberg         |
| chunk_count        | INT         | 否   | MCAP chunk 数       | Iceberg         |
| ingest_state       | TEXT        | 是   | 状态[pending... ]   | ES / Iceberg    |
| vendor_id          | TEXT        | 否   | 数据供应商          | ES / Iceberg    |
| collector_id       | TEXT        | 否   | 采集方              | ES / Iceberg    |
| task_id            | TEXT        | 否   | 采集任务            | ES / Iceberg    |
| device_id          | TEXT        | 否   | 设备 ID             | ES / Iceberg    |
| camera_model       | TEXT        | 否   | 相机型号            | ES / Iceberg    |
| data_source        | TEXT        | 否   | 数据来源            | ES / Iceberg    |
| location_id        | TEXT        | 否   | 位置 ID             | ES / Iceberg    |
| scene_id           | TEXT        | 否   | 场景 ID             | ES / Iceberg    |
| environment_id     | TEXT        | 否   | 环境 ID             | ES / Iceberg    |
| collection_method  | TEXT        | 否   | 采集方式            | ES / Iceberg    |
| tenant_id          | TEXT        | 否   | 租户 ID，多租户隔离预留 | ES / Iceberg |
| project_id         | TEXT        | 否   | 项目 ID，按业务项目隔离和统计 | ES / Iceberg |
| owner              | TEXT        | 否   | 数据归属方          | ES / Iceberg    |
| retention_tier     | TEXT        | 否   | 生命周期层级：hot / warm / cold / archive | ES / Iceberg |
| expire_at          | TIMESTAMPTZ | 否   | 数据过期或可清理时间 | Iceberg |
| process_state      | JSONB       | 是   | 文件级处理状态扩展  | Iceberg         |
| metadata           | JSONB       | 是   | 低频扩展元数据      | Iceberg         |
| is_deleted         | BOOLEAN     | 是   | 软删除              | ES / Iceberg    |
| created_at         | TIMESTAMPTZ | 是   | 创建时间            | ES / Iceberg    |
| updated_at         | TIMESTAMPTZ | 是   | 更新时间            | ES / Iceberg    |
| version            | BIGINT      | 是   | 乐观锁版本           | Iceberg         |

**设计说明：**
- `raw_hash_md5` 在 `is_deleted = FALSE` 范围内 UNIQUE：重复 ingest 同一物理文件应走幂等路径，返回已存在的 `mcap_file_id`，不创建新行。`raw_hash_sha256` 是长期更可靠的内容指纹，建议同时记录但 UNIQUE 约束可放在 sha256 上。
- `mcap_uri` 与 `storage_uri` 一致原则：仅用于运维定位，不作业务唯一标识；上云换 bucket / endpoint 时业务表不变。

---

### 4.2 assets（资产当前态，通常为 MCAP 切出的 segment/clip）

| 字段名                 | 类型         | 必填 | 说明                                    | 同步目标         |
|------------------------|-------------|------|----------------------------------------|------------------|
| asset_id               | UUID        | 是   | 资产主键                               | ES / Iceberg     |
| mcap_file_id           | UUID        | 是   | 来源 MCAP 文件 ID                       | ES / Iceberg     |
| asset_type             | TEXT        | 是   | segment / clip / frame_set / derived_asset | ES / Iceberg  |
| storage_uri            | TEXT        | 否   | asset 本体对象存储地址                  | Iceberg          |
| thumb_uri              | TEXT        | 否   | 缩略图地址                              | ES               |
| parent_asset_id        | UUID        | 否   | 父资产 ID                              | ES / Iceberg     |
| root_asset_id          | UUID        | 否   | 根资产 ID                              | ES / Iceberg     |
| asset_level            | INT         | 是   | 资产层级(0=原始,1=二切...)              | ES / Iceberg     |
| split_method           | TEXT        | 否   | 切分方式（manual/algo/rule...）         | ES / Iceberg     |
| split_algo_name        | TEXT        | 否   | 切分算法名称                            | ES / Iceberg     |
| split_algo_version     | TEXT        | 否   | 切分算法版本                            | ES / Iceberg     |
| split_run_id           | TEXT        | 否   | 切分批次 ID                             | Iceberg          |
| split_reason           | TEXT        | 否   | 切分理由                                | Iceberg          |
| segment_index          | INT         | 否   | 父资产下片段序号                        | ES / Iceberg     |
| parent_start_offset_ms | BIGINT      | 否   | 父资产起始偏移 (ms)                     | Iceberg          |
| parent_end_offset_ms   | BIGINT      | 否   | 父资产结束偏移 (ms)                     | Iceberg          |
| start_timestamp_ns     | BIGINT      | 否   | asset 起始时间                          | ES / Iceberg     |
| end_timestamp_ns       | BIGINT      | 否   | asset 结束时间                          | ES / Iceberg     |
| duration_ms            | BIGINT      | 否   | asset 时长                              | ES / Iceberg     |
| lifecycle_state        | TEXT        | 是   | 资产生命周期状态：created / processing / ready / rejected / delivered / archived / superseded | ES / Iceberg |
| tenant_id              | TEXT        | 否   | 租户 ID，多租户隔离预留                  | ES / Iceberg     |
| project_id             | TEXT        | 否   | 项目 ID，按业务项目隔离和统计            | ES / Iceberg     |
| owner                  | TEXT        | 否   | 资产 owner/权限                         | ES / Iceberg     |
| reviewer               | TEXT        | 否   | 审核人                                  | ES / Iceberg     |
| last_delivered_at      | TIMESTAMPTZ | 否   | 最近交付时间(冗余)                        | ES / Iceberg     |
| delivery_count         | INT         | 是   | 累计交付次数(冗余)                        | ES / Iceberg     |
| last_delivered_to      | TEXT        | 否   | 最近交付客户(冗余)                        | ES / Iceberg     |
| retention_tier         | TEXT        | 否   | 生命周期层级：hot / warm / cold / archive | ES / Iceberg   |
| expire_at              | TIMESTAMPTZ | 否   | 数据过期或可清理时间                     | Iceberg          |
| metadata               | JSONB       | 是   | 低频扩展元数据                           | Iceberg          |
| files                  | JSONB       | 是   | 关联文件（thumbnail、algo output等）      | Iceberg          |
| is_deleted             | BOOLEAN     | 是   | 软删除                                  | ES / Iceberg     |
| created_at             | TIMESTAMPTZ | 是   | 创建时间                                | ES / Iceberg     |
| updated_at             | TIMESTAMPTZ | 是   | 更新时间                                | ES / Iceberg     |
| version                | BIGINT      | 是   | 乐观锁版本                               | Iceberg          |

**设计说明：**
- 权威资产当前态表，基础过滤优先查此表。
- 动态 tag 放入 asset_tags；算法最新状态放入 asset_algo_latest。
- 行业相关 facet（如 city、weather、scenario_type、quality_level）不进入 assets 主表，统一放入 asset_tags，并由 tag_registry.yaml 声明类型、枚举、是否必填、是否进入 ES facets。
- `lifecycle_state` 是资产状态的唯一权威字段；交付、审核、处理等历史变化写入 asset_events，必要的列表筛选可通过投影表或 ES 派生。
- 一父多子的普通切分可直接使用 parent_asset_id / root_asset_id / asset_level。
- 多父资产、融合资产、拼接资产等复杂血缘建议使用 asset_relations 关系表表达。

**lifecycle_state 状态机：**

合法取值与转移：

```text
       ┌──────────┐
       │ created  │  ← 新建
       └────┬─────┘
            │ kick off algo / split
            ▼
       ┌──────────┐
       │processing│
       └────┬─────┘
            │ algo finish + QA pass
            ▼
       ┌──────────┐    QA reject    ┌──────────┐
       │  ready   │ ───────────────►│ rejected │
       └────┬─────┘                  └──────────┘
            │ delivered to customer
            ▼
       ┌──────────┐
       │delivered │
       └────┬─────┘
            │ retention / supersede
            ▼
   ┌──────────────┬──────────────┐
   │ archived     │ superseded   │
   └──────────────┴──────────────┘
```

- 写入端：状态转移由 backend usecase 显式驱动，CHECK 约束或 application-level state machine 保证非法转移被拒。
- 详细状态语义和算法子状态机参考 [`docs/review/algo-lifecycle-and-data-model.md`](./review/algo-lifecycle-and-data-model.md)。
- 任何状态变更必须**同事务**追加 `asset_events(event_type='asset_lifecycle_changed', payload={from, to, reason, actor})`，否则审计链断裂。

**交付汇总（last_delivered_at / delivery_count / last_delivered_to）所有权：**

- 这三列是**写时缓存**，权威数据始终在 `deliveries + delivery_items`。
- 由 backend `delivery` usecase 在**完成交付的同一事务**内 UPDATE assets 三列 + 追加 `asset_events`。
- **不使用 trigger**，避免业务逻辑下沉到 PG 层（违反 outbox / 事件来源原则）。
- 下游需要权威视角时应 JOIN deliveries + delivery_items，不应只信 assets 上的快照；快照仅用于资产列表页的"最近交付"等弱一致展示。

---

### 4.2.1 asset_relations（复杂资产血缘关系，可选）

- 普通“一父多子”切分时，可以只用 assets.parent_asset_id。
- 如果出现多父资产、融合、拼接、采样、跨模态派生，建议使用此表。
- **主键**：PRIMARY KEY (parent_asset_id, child_asset_id, relation_type)

| 字段名             | 类型         | 必填 | 说明                                  | 同步目标     |
|--------------------|--------------|------|---------------------------------------|--------------|
| parent_asset_id    | UUID         | 是   | 父资产 ID                             | ES / Iceberg |
| child_asset_id     | UUID         | 是   | 子资产 ID                             | ES / Iceberg |
| relation_type      | TEXT         | 是   | split_from / derived_from / merged_from / sampled_from | ES / Iceberg |
| method             | TEXT         | 否   | 生成方式：manual / algo / rule / pipeline | Iceberg |
| algo_name          | TEXT         | 否   | 生成子资产的算法名称                  | ES / Iceberg |
| algo_version       | TEXT         | 否   | 生成子资产的算法版本                  | ES / Iceberg |
| run_id             | TEXT         | 否   | 外部执行批次                          | Iceberg |
| parent_start_offset_ms | BIGINT   | 否   | 子资产相对父资产的起始偏移            | Iceberg |
| parent_end_offset_ms   | BIGINT   | 否   | 子资产相对父资产的结束偏移            | Iceberg |
| created_at         | TIMESTAMPTZ  | 是   | 创建时间                              | ES / Iceberg |

**设计说明：**
- assets.parent_asset_id 适合快速查直接父资产。
- asset_relations 适合表达多父、多级、多类型血缘，并同步到 Iceberg 做长期分析。

---

### 4.3 asset_tags（Asset tag 当前态投影表）

- 一个 asset 多 tag，多行设计。
- **主键**：PRIMARY KEY (asset_id, tag_key)

| 字段名        | 类型              | 必填 | 说明                   | 同步目标      |
|---------------|-------------------|------|------------------------|---------------|
| asset_id      | UUID              | 是   | 资产 ID                | ES / Iceberg  |
| tenant_id     | TEXT              | 否   | 租户 ID，冗余便于 RLS / 分区 / 同步 | ES / Iceberg |
| project_id    | TEXT              | 否   | 项目 ID，冗余便于 RLS / 分区 / 同步 | ES / Iceberg |
| tag_key       | TEXT              | 是   | tag 名称               | ES / Iceberg  |
| tag_value     | TEXT              | 是   | tag 值                 | ES / Iceberg  |
| tag_value_num | DOUBLE PRECISION  | 否   | 数值型 tag 值，用于范围过滤 | ES / Iceberg |
| tag_value_bool| BOOLEAN           | 否   | 布尔型 tag 值，用于布尔过滤 | ES / Iceberg |
| tag_type      | TEXT              | 是   | string/number/bool/enum| ES / Iceberg  |
| source_type   | TEXT              | 是   | human/algo/rule/system | ES / Iceberg  |
| source_name   | TEXT              | 否   | 算法、规则名/人工      | ES / Iceberg  |
| source_version| TEXT              | 否   | 算法/规则版本          | ES / Iceberg  |
| run_id        | TEXT              | 否   | 外部批次               | Iceberg       |
| confidence    | DOUBLE PRECISION  | 否   | 置信度                 | ES / Iceberg  |
| created_at    | TIMESTAMPTZ       | 是   | 首次创建时间           | Iceberg       |
| updated_at    | TIMESTAMPTZ       | 是   | 最近更新时间           | ES / Iceberg  |

**设计说明：**
- 当前态用 asset_tags，历史变化查 asset_events，ES 文档 tags 字段来源此表。
- 高频 tag 需在 tag_registry.yaml 注册。
- asset 软删除后，asset_tags 不做级联硬删；在线查询通过 JOIN assets 并过滤 assets.is_deleted=false，审计和历史由 asset_events/Iceberg 保留。

**推荐索引：**
- `(asset_id, tag_key)` 作为主键。
- `(tag_key, tag_value, asset_id)` 支撑枚举/字符串 facet。
- `(tag_key, tag_value_num, asset_id) WHERE tag_type = 'number'` 支撑数值范围过滤。
- `(tag_key, tag_value_bool, asset_id) WHERE tag_type = 'bool'` 支撑布尔过滤。

---

### 4.4 asset_algo_latest（Asset 算法最新状态投影）

- 每个 asset + algo 保留一行最新状态  
- **主键**：PRIMARY KEY (asset_id, algo_name)

| 字段名         | 类型              | 必填 | 说明              | 同步目标      |
|----------------|------------------|------|-------------------|---------------|
| asset_id       | UUID             | 是   | 资产 ID           | ES / Iceberg  |
| tenant_id      | TEXT             | 否   | 租户 ID，冗余便于 RLS / 分区 / 同步 | ES / Iceberg |
| project_id     | TEXT             | 否   | 项目 ID，冗余便于 RLS / 分区 / 同步 | ES / Iceberg |
| algo_name      | TEXT             | 是   | 算法名称          | ES / Iceberg  |
| algo_version   | TEXT             | 是   | 算法版本          | ES / Iceberg  |
| status         | TEXT             | 是   | 状态（blocked...）| ES / Iceberg  |
| result_tag     | TEXT             | 否   | 算法输出标签      | ES / Iceberg  |
| result_score   | DOUBLE PRECISION | 否   | 算法分数          | ES / Iceberg  |
| result_summary | JSONB            | 是   | 低频结果摘要      | Iceberg       |
| run_id         | TEXT             | 否   | 执行批次          | ES / Iceberg  |
| method         | TEXT             | 否   | 执行方式          | Iceberg       |
| model_uri      | TEXT             | 否   | 模型文件地址      | Iceberg       |
| output_uri     | TEXT             | 否   | 输出地址          | Iceberg       |
| error_code     | TEXT             | 否   | 错误码            | Iceberg       |
| error_message  | TEXT             | 否   | 失败原因          | Iceberg       |
| started_at     | TIMESTAMPTZ      | 否   | 开始时间          | Iceberg       |
| finished_at    | TIMESTAMPTZ      | 否   | 完成时间          | ES / Iceberg  |
| updated_at     | TIMESTAMPTZ      | 是   | 更新时间          | ES / Iceberg  |

**推荐索引:**
- algo_name+status
- algo_name+algo_version
- run_id (WHERE run_id IS NOT NULL)
- updated_at DESC

**设计说明：**
- “哪些 asset 跑过 A 算法”查此表。
- 历史更精确分析走 Iceberg。
- 完整算法生命周期写入 asset_events。
- asset 软删除后，asset_algo_latest 不做级联硬删；在线查询通过 JOIN assets 并过滤 assets.is_deleted=false。

---

### 4.5 asset_events（统一事件表，审计/同步/回放核心来源）

- **主键**：event_id
- **单调序号**：event_seq BIGSERIAL UNIQUE，所有 outbox / CDC / replay 消费者必须按 event_seq 严格递增消费，不能用 occurred_at 排序

| 字段名 | 类型 | 必填 | 说明 | 同步目标 |
|--------|------|------|------|----------|
| event_id | UUID | 是 | 事件主键，业务唯一标识 | Iceberg |
| event_seq | BIGSERIAL | 是 | 单调递增序号，UNIQUE；消费者用此字段做"从 X 之后增量拉" | Iceberg |
| event_type | TEXT | 是 | 事件类型 | ES / Iceberg |
| payload_schema_version | TEXT | 是 | event_payload 的 schema 版本，如 v1 / v2；schema 注册到 schemas/events/&lt;event_type&gt;.json | Iceberg |
| asset_id | UUID | 否 | 关联 asset | ES / Iceberg |
| mcap_file_id | UUID | 否 | 关联 MCAP | Iceberg |
| tenant_id | TEXT | 否 | 租户 ID | ES / Iceberg |
| project_id | TEXT | 否 | 项目 ID | ES / Iceberg |
| event_source | TEXT | 是 | backend / worker / dagster / spark / daft / system | Iceberg |
| actor_type | TEXT | 否 | user / service / algo / system | Iceberg |
| actor_id | TEXT | 否 | 操作者 | Iceberg |
| request_id | TEXT | 否 | 请求追踪 | Iceberg |
| idempotency_key | TEXT | 否 | 幂等 key | Iceberg |
| run_id | TEXT | 否 | 外部执行批次 | Iceberg |
| occurred_at | TIMESTAMPTZ | 是 | 业务发生时间 | ES / Iceberg |
| created_at | TIMESTAMPTZ | 是 | 入库时间 | Iceberg |
| publish_state | TEXT | 是 | pending / published / failed | PostgreSQL |
| published_at | TIMESTAMPTZ | 否 | 同步完成时间 | PostgreSQL |
| event_payload | JSONB | 是 | 类型相关字段，如 tag/algo/delivery/dataset/training 详情 | Iceberg |

**典型事件类型：**
- mcap_ingested, asset_created, asset_updated, tag_upserted, tag_deleted, algo_started, algo_finished, algo_failed, delivery_created, delivery_item_added, delivery_completed, dataset_snapshot_created, training_run_started, training_run_finished

**设计说明：**
- 当前态表变更时，**同事务**追加 asset_events，保证业务写入和事件记录的强一致。
- **顺序保证**：消费者必须以 `event_seq` 严格递增推进 watermark，不能用 `occurred_at`（并发写入下时间戳会冲突 / 倒序）。Debezium 切换到 WAL 之后，order key 仍为 `event_seq`，避免 LSN 顺序与业务顺序错配。
- **payload schema 演进**：每种 `event_type` 对应一份 JSON Schema，文件位于 `schemas/events/&lt;event_type&gt;.v&lt;n&gt;.json`；写入端必须填 `payload_schema_version`；下游 Silver 平展时按版本路由到对应解析器。新增字段走 minor 版本（向后兼容），破坏性变更走 major 版本（旧版本同时保留消费一段时间）。
- 类型相关字段不要摊成大量 nullable 列，统一进入 `event_payload`。
- 在线查询主要使用 `(event_type, occurred_at DESC)`、`asset_id`、`run_id` 等 B-tree 索引；需要临时按 payload 检索时再加 GIN。
- Bronze 同步到 Iceberg 后，在 Silver 层按 `event_type + payload_schema_version` 把 `event_payload` 平展成宽表。
- publish_state=pending 可作为 outbox，由 worker 同步 ES / Iceberg / 下游消息。
- **分区策略**：单表行数预计超过千万后，按 `occurred_at` 月分区（`PARTITION BY RANGE`），retention 策略 drop old partitions。

---

### 4.6 deliveries（客户交付批次当前态）

| 字段名             | 类型        | 必填 | 说明              | 同步目标        |
|--------------------|------------|------|-------------------|-----------------|
| delivery_id        | UUID       | 是   | 交付批次主键      | ES / Iceberg    |
| tenant_id          | TEXT       | 否   | 租户 ID           | ES / Iceberg    |
| project_id         | TEXT       | 否   | 项目 ID           | ES / Iceberg    |
| customer_id        | TEXT       | 是   | 客户 ID           | ES / Iceberg    |
| contract_id        | TEXT       | 否   | 合同/订单号       | ES / Iceberg    |
| delivery_type      | TEXT       | 是   | 类型[asset_set...]| Iceberg         |
| status             | TEXT       | 是   | 状态[created...]  | ES / Iceberg    |
| requested_by       | TEXT       | 否   | 申请人            | Iceberg         |
| approved_by        | TEXT       | 否   | 审批人            | Iceberg         |
| delivered_by       | TEXT       | 否   | 交付人            | Iceberg         |
| manifest_uri       | TEXT       | 否   | 交付清单          | Iceberg         |
| replay_manifest_uri| TEXT       | 否   | 回放清单          | Iceberg         |
| item_count         | BIGINT     | 是   | 交付 asset 数量   | ES / Iceberg    |
| total_size_bytes   | BIGINT     | 否   | 交付总大小        | ES / Iceberg    |
| created_at         | TIMESTAMPTZ| 是   | 创建时间          | ES / Iceberg    |
| updated_at         | TIMESTAMPTZ| 是   | 更新时间          | ES / Iceberg    |
| completed_at       | TIMESTAMPTZ| 否   | 完成时间          | ES / Iceberg    |
| version            | BIGINT     | 是   | 乐观锁版本        | Iceberg         |

**索引推荐：**
- (customer_id, created_at DESC), (status), (completed_at DESC)

---

### 4.7 delivery_items（Delivery-Asset M:N 明细）

- **主键**：PRIMARY KEY (delivery_id, asset_id)

| 字段名       | 类型         | 必填 | 说明            | 同步目标  |
|--------------|-------------|------|-----------------|-----------|
| delivery_id  | UUID        | 是   | 交付批次 ID     | Iceberg   |
| asset_id     | UUID        | 是   | 资产 ID         | Iceberg   |
| asset_version| BIGINT      | 否   | 交付时资产版本   | Iceberg   |
| item_state   | TEXT        | 是   | 状态            | Iceberg   |
| checksum     | TEXT        | 否   | 导出文件校验     | Iceberg   |
| export_uri   | TEXT        | 否   | 导出对象地址     | Iceberg   |
| created_at   | TIMESTAMPTZ | 是   | 创建时间        | Iceberg   |

**设计说明**
- 单个 asset 可交付给多个客户；一个 delivery 可包多个 asset。回放查 manifest_uri+delivery_items。

---

### 4.8 dataset_snapshots（训练/评测数据集快照元信息）

- **主键**：PRIMARY KEY (dataset_id, snapshot_id)

#### datasets（数据集定义父表）

- **主键**：dataset_id

| 字段名 | 类型 | 必填 | 说明 | 同步目标 |
|--------|------|------|------|----------|
| dataset_id | UUID | 是 | 数据集 ID | ES / Iceberg |
| name | TEXT | 是 | 数据集名称 | ES / Iceberg |
| description | TEXT | 否 | 数据集描述 | ES / Iceberg |
| owner | TEXT | 否 | 负责人或团队 | ES / Iceberg |
| tenant_id | TEXT | 否 | 租户 ID | ES / Iceberg |
| project_id | TEXT | 否 | 项目 ID | ES / Iceberg |
| dataset_type | TEXT | 是 | training / eval / replay / delivery / experiment | ES / Iceberg |
| status | TEXT | 是 | active / archived / deleted | ES / Iceberg |
| created_by | TEXT | 否 | 创建人 | ES / Iceberg |
| created_at | TIMESTAMPTZ | 是 | 创建时间 | ES / Iceberg |
| updated_at | TIMESTAMPTZ | 是 | 更新时间 | ES / Iceberg |

#### dataset_snapshots（数据集快照）

| 字段名      | 类型         | 必填 | 说明              | 同步目标      |
|-------------|-------------|------|-------------------|---------------|
| dataset_id  | UUID        | 是   | 数据集 ID         | ES / Iceberg  |
| snapshot_id | UUID        | 是   | 快照 ID           | ES / Iceberg  |
| snapshot_version | TEXT    | 否   | 快照版本号，如 v1 / 2026-04-27 | ES / Iceberg |
| created_by  | TEXT        | 否   | 创建人            | ES / Iceberg  |
| query_spec  | JSONB       | 是   | 构建条件          | Iceberg       |
| source_query_hash | TEXT   | 否   | 查询条件 hash，用于判断快照是否可复现/可比较 | Iceberg |
| source_catalog_name | TEXT | 否 | 来源 Catalog，如 lakehouse | Iceberg |
| source_namespace | TEXT | 否 | 来源 namespace，如 robot.gold | Iceberg |
| source_object_name | TEXT | 否 | 来源表或视图，如 gold_training_samples | Iceberg |
| source_object_version_ref | TEXT | 否 | 来源对象版本，如 Iceberg snapshot_id | Iceberg |
| manifest_uri| TEXT        | 是   | 清单地址          | Iceberg       |
| item_count  | BIGINT      | 是   | asset 数          | ES / Iceberg  |
| status      | TEXT        | 是   | 状态[building...] | ES / Iceberg  |
| created_at  | TIMESTAMPTZ | 是   | 创建时间          | ES / Iceberg  |
| updated_at  | TIMESTAMPTZ | 是   | 更新时间          | ES / Iceberg  |

**索引推荐**：(dataset_id, created_at DESC)，status

**设计说明**：
- 快照大明细仅建议 Iceberg 存储，PG 留 manifest/摘要。
- 快照来源必须自包含记录 `source_catalog_name + source_namespace + source_object_name + source_object_version_ref`，不要只记录 SQL 文本、物理路径或可变 FK。
- **不可硬删约定**：一旦某个 `(dataset_id, snapshot_id)` 被 `training_runs` / `training_sample_exports` / `asset_events` 引用，**禁止物理 DELETE**，只能将 `status` 置为 `archived`。lifecycle / retention job 不会触碰被引用的 snapshot。这条约定是审计链不断的前提，必须由 backend usecase 和 lifecycle worker 共同保证。

---

### 4.9 training_runs（训练任务记录）

| 字段名       | 类型         | 必填 | 说明                | 同步目标      |
|--------------|-------------|------|---------------------|---------------|
| training_run_id| UUID      | 是   | 训练任务 ID         | ES / Iceberg  |
| tenant_id      | TEXT      | 否   | 租户 ID             | ES / Iceberg  |
| project_id     | TEXT      | 否   | 项目 ID             | ES / Iceberg  |
| dataset_id   | UUID        | 是   | 数据集 ID           | ES / Iceberg  |
| snapshot_id  | UUID        | 是   | 数据集快照 ID       | ES / Iceberg  |
| model_name   | TEXT        | 是   | 模型名称            | ES / Iceberg  |
| model_version| TEXT        | 否   | 模型版本            | ES / Iceberg  |
| algo_name    | TEXT        | 否   | 训练/评测算法       | ES / Iceberg  |
| code_version | TEXT        | 否   | 训练代码版本，如 git commit / image tag | Iceberg |
| config_uri   | TEXT        | 否   | 训练配置文件地址    | Iceberg |
| data_manifest_uri | TEXT   | 否   | 本次训练实际使用的数据清单地址 | Iceberg |
| data_catalog_name | TEXT | 否 | 本次训练数据所在 Catalog | Iceberg |
| data_namespace | TEXT | 否 | 本次训练数据 namespace | Iceberg |
| data_object_name | TEXT | 否 | 本次训练数据表 / Lance dataset / manifest 对象名 | Iceberg |
| data_object_version_ref | TEXT | 否 | 本次训练数据版本，如 Iceberg snapshot 或 Lance version | Iceberg |
| status       | TEXT        | 是   | 状态                | ES / Iceberg  |
| metrics      | JSONB       | 是   | 训练指标摘要        | Iceberg       |
| artifact_uri | TEXT        | 否   | 模型产物地址        | Iceberg       |
| started_at   | TIMESTAMPTZ | 否   | 开始时间            | ES / Iceberg  |
| finished_at  | TIMESTAMPTZ | 否   | 完成时间            | ES / Iceberg  |
| created_at   | TIMESTAMPTZ | 是   | 创建时间            | ES / Iceberg  |
| updated_at   | TIMESTAMPTZ | 是   | 更新时间            | ES / Iceberg  |

**设计说明**  
- 训练记录是审计证据，必须自包含记录当时使用的 catalog/namespace/object/version，不依赖可变的 catalog_objects FK。
- “某次训练用了哪些 asset”：查 training_runs -> dataset_snapshots 的 Catalog 引用和 manifest，再查 Iceberg 明细表。

---

### 4.10 后训练扩展表（feature_sets / feature_jobs / training_sample_exports）

> ⚠️ **本节字段尚未冻结**。当前 MVP 不落地这三张表；落地前需要重新评审。下方字段表只作为方向参考，不要直接抄。

这几张表用于长期演进，当前 MVP 可以先不落地。设计原则是：PostgreSQL 只保存定义、任务状态、manifest 和摘要，大规模样本明细、特征明细、训练样本行写入 Iceberg 或对象存储。

#### feature_sets（特征集合定义）

- **主键**：feature_set_id

| 字段名 | 类型 | 必填 | 说明 | 同步目标 |
|--------|------|------|------|----------|
| feature_set_id | UUID | 是 | 特征集合 ID | ES / Iceberg |
| tenant_id | TEXT | 否 | 租户 ID | ES / Iceberg |
| project_id | TEXT | 否 | 项目 ID | ES / Iceberg |
| name | TEXT | 是 | 特征集合名称 | ES / Iceberg |
| feature_set_version | TEXT | 是 | 特征集合版本 | ES / Iceberg |
| description | TEXT | 否 | 描述 | ES / Iceberg |
| owner | TEXT | 否 | 负责人或团队 | ES / Iceberg |
| schema_uri | TEXT | 否 | 特征 schema 文件地址 | Iceberg |
| feature_spec | JSONB | 是 | 特征定义、来源、依赖算法、字段说明 | Iceberg |
| status | TEXT | 是 | draft / active / deprecated | ES / Iceberg |
| created_at | TIMESTAMPTZ | 是 | 创建时间 | ES / Iceberg |
| updated_at | TIMESTAMPTZ | 是 | 更新时间 | ES / Iceberg |

#### feature_jobs（特征抽取 / 回填 / 实验分支任务）

- **主键**：feature_job_id

| 字段名 | 类型 | 必填 | 说明 | 同步目标 |
|--------|------|------|------|----------|
| feature_job_id | UUID | 是 | 特征任务 ID | ES / Iceberg |
| tenant_id | TEXT | 否 | 租户 ID | ES / Iceberg |
| project_id | TEXT | 否 | 项目 ID | ES / Iceberg |
| feature_set_id | UUID | 是 | 关联 feature_sets | ES / Iceberg |
| feature_set_version | TEXT | 是 | 特征集合版本 | ES / Iceberg |
| job_type | TEXT | 是 | extract / backfill / branch_experiment / merge | ES / Iceberg |
| branch_name | TEXT | 否 | 实验分支名称，用于特征调研 | ES / Iceberg |
| source_snapshot_id | UUID | 否 | 来源数据集快照 | Iceberg |
| run_id | TEXT | 否 | Spark / Ray / Dagster 执行批次 | Iceberg |
| input_manifest_uri | TEXT | 否 | 输入 asset 清单 | Iceberg |
| output_catalog_name | TEXT | 否 | 输出对象 Catalog，如 lakehouse / multimodal | Iceberg |
| output_namespace | TEXT | 否 | 输出对象 namespace，如 robot.gold | Iceberg |
| output_object_name | TEXT | 否 | 输出表 / Lance dataset 名称 | Iceberg |
| output_object_type | TEXT | 否 | iceberg_table / lance_dataset / manifest | Iceberg |
| output_object_version_ref | TEXT | 否 | 输出对象版本，如 Iceberg snapshot_id / Lance version | Iceberg |
| output_manifest_uri | TEXT | 否 | 输出文件或 manifest 地址 | Iceberg |
| status | TEXT | 是 | pending / running / succeeded / failed | ES / Iceberg |
| metrics | JSONB | 是 | 行数、耗时、失败数等摘要 | Iceberg |
| started_at | TIMESTAMPTZ | 否 | 开始时间 | ES / Iceberg |
| finished_at | TIMESTAMPTZ | 否 | 完成时间 | ES / Iceberg |
| created_at | TIMESTAMPTZ | 是 | 创建时间 | ES / Iceberg |
| updated_at | TIMESTAMPTZ | 是 | 更新时间 | ES / Iceberg |

#### training_sample_exports（训练样本导出任务）

- **主键**：export_id

| 字段名 | 类型 | 必填 | 说明 | 同步目标 |
|--------|------|------|------|----------|
| export_id | UUID | 是 | 导出任务 ID | ES / Iceberg |
| tenant_id | TEXT | 否 | 租户 ID | ES / Iceberg |
| project_id | TEXT | 否 | 项目 ID | ES / Iceberg |
| dataset_id | UUID | 是 | 数据集 ID | ES / Iceberg |
| snapshot_id | UUID | 是 | 数据集快照 ID | ES / Iceberg |
| feature_set_id | UUID | 否 | 使用的特征集合 | ES / Iceberg |
| feature_set_version | TEXT | 否 | 特征集合版本 | ES / Iceberg |
| sample_catalog_name | TEXT | 否 | 训练样本所在 Catalog，如 lakehouse / multimodal | Iceberg |
| sample_namespace | TEXT | 否 | 训练样本 namespace，如 robot.gold | Iceberg |
| sample_object_name | TEXT | 否 | 训练样本表 / Lance dataset 名称 | Iceberg |
| sample_object_type | TEXT | 否 | iceberg_table / lance_dataset | Iceberg |
| sample_object_version_ref | TEXT | 否 | 训练样本对象版本，如 Iceberg snapshot_id / Lance version | Iceberg |
| manifest_uri | TEXT | 是 | 训练实际读取的 manifest / parquet / arrow 地址 | Iceberg |
| file_format | TEXT | 是 | parquet / arrow / tfrecord / jsonl | Iceberg |
| partition_spec | JSONB | 否 | 导出分区策略 | Iceberg |
| item_count | BIGINT | 是 | 样本数 | ES / Iceberg |
| total_size_bytes | BIGINT | 否 | 导出总大小 | ES / Iceberg |
| status | TEXT | 是 | building / ready / failed | ES / Iceberg |
| created_at | TIMESTAMPTZ | 是 | 创建时间 | ES / Iceberg |
| updated_at | TIMESTAMPTZ | 是 | 更新时间 | ES / Iceberg |

**设计说明**
- 训练不要通过 API 一条条读取 asset，应读取 Iceberg Gold 表或导出的 Parquet / Arrow / TFRecord。
- 特征调研可以先写实验 `branch_name` 或实验 `sample_object_name`，验证后再升级为正式 feature_set_version。
- PostgreSQL 只记录任务和 manifest，训练样本行级明细由 Iceberg 承载。

---

### 4.11 Platform Catalog 抽象表

这组表是短中期的轻量 Platform Catalog。它不替代 Iceberg REST Catalog 的事务能力，也不绑定 Gravitino / Polaris / DLF / Glue。它负责把平台里的外部数据对象统一注册起来，让业务层长期只依赖中立引用。

> 🚧 **未决：业务表如何引用 catalog_objects**（待 ADR）
>
> 现在 `training_runs / dataset_snapshots / training_sample_exports` 都是直接嵌 `catalog_name + namespace + object_name + version_ref` 字符串四元组，**没有引用 `catalog_objects.object_id` 或 `catalog_object_versions.object_version_id`**。两种方案二选一，必须在 Phase 2 落地前定下来：
>
> - **方案 A（软引用 / 字符串四元组）**：业务表只存四元组，`catalog_objects` 仅作 UI 注册和元数据展示。约定：**rename = 新对象**；老对象保留为 `status='deprecated'`。优点：审计证据自包含、跨 catalog 系统迁移零成本。缺点：catalog_objects 无强引用根，理论上可不一致。
> - **方案 B（硬引用 / object_version_id FK）**：业务表存 `catalog_object_version_id` 外键。约定：rename 不允许，只能 deprecate + 新建。优点：强引用完整性。缺点：训练记录被 catalog 物理表绑定，跨系统迁移需要数据迁移。
>
> **当前文档默认走 A**（与 §3.4 / §4.9 / §4.8 一致）；未来如改用 B 需要全表 ALTER + 业务层重写。

#### catalog_objects（数据对象注册表）

- **主键**：object_id
- **唯一约束建议**：UNIQUE (catalog_name, namespace, object_name, object_type)

| 字段名 | 类型 | 必填 | 说明 | 同步目标 |
|--------|------|------|------|----------|
| object_id | UUID | 是 | 平台对象 ID | Iceberg |
| catalog_name | TEXT | 是 | 平台 Catalog 名称，如 online / lakehouse / search / multimodal | Iceberg |
| namespace | TEXT | 是 | 命名空间，如 public / robot.bronze / robot.gold | Iceberg |
| object_name | TEXT | 是 | 对象名，如 assets / gold_training_samples / robot_video_frames | Iceberg |
| object_type | TEXT | 是 | postgres_table / iceberg_table / elasticsearch_index / lance_dataset / object_prefix / manifest | Iceberg |
| provider | TEXT | 是 | iceberg_rest / postgres / elasticsearch / lance / polaris / gravitino / glue / dlf | Iceberg |
| format | TEXT | 是 | postgres / iceberg / elasticsearch / lance / parquet / webdataset / manifest | Iceberg |
| storage_uri | TEXT | 否 | 物理存储位置，仅作定位，不作业务唯一标识 | Iceberg |
| external_ref | TEXT | 否 | 外部系统对象引用，如 REST catalog identifier / index UUID | Iceberg |
| owner | TEXT | 否 | 负责人或团队 | ES / Iceberg |
| tenant_id | TEXT | 否 | 租户 ID | ES / Iceberg |
| project_id | TEXT | 否 | 项目 ID | ES / Iceberg |
| description | TEXT | 否 | 描述 | ES / Iceberg |
| tags | JSONB | 是 | 对象级标签，如 pii / training / gold / deprecated | ES / Iceberg |
| properties | JSONB | 是 | provider 扩展属性 | Iceberg |
| status | TEXT | 是 | active / deprecated / deleted | ES / Iceberg |
| created_at | TIMESTAMPTZ | 是 | 创建时间 | ES / Iceberg |
| updated_at | TIMESTAMPTZ | 是 | 更新时间 | ES / Iceberg |

**设计说明**
- `catalog_name + namespace + object_name + object_type` 是平台内稳定引用。
- `provider` 可以后续从 `iceberg_rest` 换成 `polaris` / `gravitino` / `dlf`，业务表不需要改。
- `storage_uri` 允许变更，不应该被训练任务、数据集快照当作唯一版本。

#### catalog_object_versions（数据对象版本表）

- **主键**：object_version_id
- **索引推荐**：(object_id, created_at DESC)，(object_id, version_ref)

| 字段名 | 类型 | 必填 | 说明 | 同步目标 |
|--------|------|------|------|----------|
| object_version_id | UUID | 是 | 对象版本 ID | Iceberg |
| object_id | UUID | 是 | 对应 catalog_objects.object_id | Iceberg |
| version_ref | TEXT | 是 | 外部版本引用，如 Iceberg snapshot_id / Lance version / ES index generation | Iceberg |
| version_type | TEXT | 是 | iceberg_snapshot / lance_version / manifest_version / index_generation / schema_version | Iceberg |
| schema_ref | TEXT | 否 | schema 文件、schema hash 或 catalog schema version | Iceberg |
| manifest_uri | TEXT | 否 | manifest 或 metadata 文件位置 | Iceberg |
| row_count | BIGINT | 否 | 行数或对象数 | ES / Iceberg |
| size_bytes | BIGINT | 否 | 数据大小 | ES / Iceberg |
| checksum | TEXT | 否 | 校验值 | Iceberg |
| created_by | TEXT | 否 | 创建人或任务 | Iceberg |
| created_at | TIMESTAMPTZ | 是 | 创建时间 | ES / Iceberg |
| properties | JSONB | 是 | 扩展属性 | Iceberg |

**设计说明**
- Iceberg 的 snapshot_id、Lance 的 version、训练 manifest 版本都统一放到 `version_ref`。
- dataset snapshot 和 training run 引用版本时，优先引用 `object_id + version_ref` 或 `object_version_id`。

#### 不在 PG 中重写的能力

| 能力 | 短期处理 | 中长期组件 |
|------|----------|------------|
| 对象级血缘 | 任务记录里保留 `run_id / source / output` 摘要 | OpenLineage / Marquez |
| 权限策略 | Backend RBAC + JWT scope + 租户/项目过滤 | Ranger / Cloud IAM / Lake Formation / DLF / Dataplex |
| 数据质量 | Dagster asset checks 写状态摘要和 dashboard 链接 | Great Expectations / Soda / Dagster Checks |

PG 里的 Platform Catalog 只保留 `catalog_objects + catalog_object_versions` 两张核心表，避免在早期重写 Gravitino / Polaris / OpenLineage / Ranger / Great Expectations 的职责。

---

### 4.12 idempotency_keys（幂等写入保护）

- **主键**：PRIMARY KEY (scope, idem_key)
- **清理**：`expires_at` 由独立 lifecycle job 定期清理（GDPR 合规：response_json 可能含 PII，不能无限保留）

| 字段名        | 类型         | 必填 | 说明                   | 同步目标 |
|---------------|-------------|------|------------------------|----------|
| scope         | TEXT        | 是   | 幂等域，如 deliveries.create | 不同步 |
| idem_key      | TEXT        | 是   | 幂等 key（客户端提供） | 不同步   |
| resource_type | TEXT        | 否   | 资源类型               | 不同步   |
| resource_id   | TEXT        | 否   | 资源 ID                | 不同步   |
| request_hash  | TEXT        | 否   | 请求 hash              | 不同步   |
| response_json | JSONB       | 否   | 首次响应缓存           | 不同步   |
| status_code   | INT         | 否   | 首次响应状态码         | 不同步   |
| created_at    | TIMESTAMPTZ | 是   | 创建时间               | 不同步   |
| expires_at    | TIMESTAMPTZ | 否   | 过期时间               | 不同步   |

**索引推荐**：`expires_at WHERE expires_at IS NOT NULL`

---

## 5. Elasticsearch 文档设计

已合并到 [`data-platform-design.md` §5.6.1](./data-platform-design.md)。本文件不再重复维护 ES 文档结构、mapping 与 fallback 设计。

---

## 6. Iceberg 表映射

已合并到 [`data-platform-design.md` §5.6.2 / §5.6.4](./data-platform-design.md)。本文件只保留 `datasets`、`training_runs`、`catalog_objects` 等 PG 元数据表本身，不再重复 Bronze/Silver/Gold 分层与同步阶段说明。

---

## 7. 典型问题与推荐查询路径一览

已合并到 [`api-guide.md`](./api-guide.md) 与 [`use-cases.md`](./use-cases.md)。查询路径的维护粒度更适合放在 API / use case 文档，而不是 schema companion。
