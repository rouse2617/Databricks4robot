# 数据平台架构与核心表结构整理

---

## 1. 架构分层与组件职责

| 层级   | 组件              | 职责说明                                       |
|--------|-------------------|------------------------------------------------|
| 在线业务层 | PostgreSQL        | 点查、事务、当前态筛选、状态机、权限、幂等       |
| 检索层   | ES / OpenSearch   | 模糊查询、全文检索、多字段过滤、facets、资产发现 |
| 湖仓层   | Iceberg           | 历史事实、训练集、审计回放、统计分析、重算       |
| 查询层   | Trino             | 查询 Iceberg，服务复杂分析和离线报表            |
| 计算层   | Spark / Dagster   | 批处理、回填、特征抽取、重算、导出              |

- PostgreSQL：在线权威主库，负责事务、点查、当前态、权限、幂等、状态更新。
- Elasticsearch/OpenSearch：资产检索层，模糊搜索、全文检索、多字段过滤、聚合与资产发现。
- Iceberg：历史事实分析层，负责大规模历史查询、训练集、审计回放、算法重算、长期统计。
- 设计约定：PostgreSQL 表结构先保障在线业务，再通过事件流与同步服务支撑 ES 和 Iceberg。

---

## 2. 核心表清单及主职责

| 表名             | 类型   | 核心职责                                     |
|------------------|--------|----------------------------------------------|
| mcap_files       | 主表   | 原始 MCAP 文件当前态                          |
| assets           | 主表   | Segment / Clip / Asset 当前态                 |
| asset_tags       | 投影表 | 资产 tag 当前态，支持在线过滤、ES 文档生成    |
| asset_algo_latest| 投影表 | 资产算法最新状态，支持在线过滤、ES 文档生成   |
| asset_events     | 事件表 | 统一业务事件、审计、同步 Iceberg/ES           |
| deliveries       | 主表   | 客户交付批次当前态                            |
| delivery_items   | 关联表 | Delivery 和 Asset 的 M:N 明细                 |
| dataset_snapshots| 主表   | 训练/评测数据集快照元信息                     |
| training_runs    | 主表   | 训练任务与数据集使用记录                      |
| idempotency_keys | 控制表 | 幂等写入保护                                  |

长期后训练场景可扩展：

| 表名 | 类型 | 核心职责 |
|------|------|----------|
| feature_sets | 主表 | 特征集合定义，描述一组可复用训练特征 |
| feature_jobs | 主表 | 特征抽取、回填、实验分支任务记录 |
| training_sample_exports | 主表 | 训练样本导出任务元信息，明细写 Iceberg / 对象存储 |

**表关系简表：**
- mcap_files 1 ─── N assets
- assets     1 ─── N asset_tags
- assets     1 ─── N asset_algo_latest
- assets     1 ─── N asset_events
- assets     N ─── N deliveries        (via delivery_items)
- dataset_snapshots 1 ─── N training_runs

---

## 3. 字段设计原则

### 3.1 高频过滤字段列化
> 资产管理页经常过滤字段必须用普通列存储，不能长期放在 JSONB 里：

- city
- road_type
- weather
- time_of_day
- scenario_type
- quality_level
- processing_state
- review_state
- delivery_state
- start_timestamp_ns
- end_timestamp_ns
- created_at
- updated_at

### 3.2 tag 与算法状态用投影表

- 高频 tag / algo 不要写 assets.metadata  
  - tag 当前态：asset_tags
  - 算法当前状态：asset_algo_latest
  - 历史/审计：asset_events
  - 复杂检索：OpenSearch
  - 历史分析/重算：Iceberg + Trino

### 3.3 JSONB 只作扩展区

- 合理用途：
  - 低频扩展字段
  - 详情页展示
  - 非核心查询条件
  - 事件 payload
  - 算法结果摘要
- 不推荐：
  - 资产列表筛选/主过滤
  - 高频 tag / algo 状态
  - 权限判断
  - 幂等控制

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
| city                   | TEXT        | 否   | 城市                                    | ES / Iceberg     |
| road_type              | TEXT        | 否   | 道路类型                                | ES / Iceberg     |
| weather                | TEXT        | 否   | 天气                                    | ES / Iceberg     |
| time_of_day            | TEXT        | 否   | 白天/夜晚/黄昏等                         | ES / Iceberg     |
| scenario_type          | TEXT        | 否   | 场景类型                                | ES / Iceberg     |
| quality_level          | TEXT        | 否   | 质量等级                                | ES / Iceberg     |
| processing_state       | TEXT        | 是   | 处理状态                                | ES / Iceberg     |
| review_state           | TEXT        | 是   | 审核状态                                | ES / Iceberg     |
| delivery_state         | TEXT        | 是   | 交付状态                                | ES / Iceberg     |
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
- 一父多子的普通切分可直接使用 parent_asset_id / root_asset_id / asset_level。
- 多父资产、融合资产、拼接资产等复杂血缘建议使用 asset_relations 关系表表达。

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

---

### 4.4 asset_algo_latest（Asset 算法最新状态投影）

- 每个 asset + algo 保留一行最新状态  
- **主键**：PRIMARY KEY (asset_id, algo_name)

| 字段名         | 类型              | 必填 | 说明              | 同步目标      |
|----------------|------------------|------|-------------------|---------------|
| asset_id       | UUID             | 是   | 资产 ID           | ES / Iceberg  |
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

---

### 4.5 asset_events（统一事件表，审计/同步/回放核心来源；暂定稿）

- **主键**：event_id

| 字段名         | 类型        | 必填 | 说明                  | 同步目标        |
|----------------|------------|------|-----------------------|-----------------|
| event_id       | UUID       | 是   | 事件主键              | Iceberg         |
| asset_id       | UUID       | 否   | 关联 asset            | ES / Iceberg    |
| parent_asset_id| UUID       | 否   | 父资产 ID，用于血缘事件分析 | Iceberg    |
| root_asset_id  | UUID       | 否   | 根资产 ID，用于全链路追踪 | Iceberg     |
| mcap_file_id   | UUID       | 否   | 关联 MCAP             | Iceberg         |
| event_type     | TEXT       | 是   | 事件类型              | ES / Iceberg    |
| event_source   | TEXT       | 是   | backend/worker/...    | Iceberg         |
| actor_type     | TEXT       | 否   | user/service/algo     | Iceberg         |
| actor_id       | TEXT       | 否   | 操作者                | Iceberg         |
| request_id     | TEXT       | 否   | 请求追踪              | Iceberg         |
| idempotency_key| TEXT       | 否   | 幂等 key              | Iceberg         |
| tag_key        | TEXT       | 否   | tag 名称              | ES / Iceberg    |
| old_tag_value  | TEXT       | 否   | 旧 tag 值             | Iceberg         |
| new_tag_value  | TEXT       | 否   | 新 tag 值             | ES / Iceberg    |
| algo_name      | TEXT       | 否   | 算法名称              | ES / Iceberg    |
| algo_version   | TEXT       | 否   | 算法版本              | ES / Iceberg    |
| run_id         | TEXT       | 否   | 批次                  | Iceberg         |
| algo_status    | TEXT       | 否   | 算法状态              | ES / Iceberg    |
| delivery_id    | UUID       | 否   | 交付批次              | Iceberg         |
| dataset_id     | UUID       | 否   | 数据集 ID             | Iceberg         |
| training_run_id| UUID       | 否   | 训练任务 ID           | Iceberg         |
| event_payload  | JSONB      | 是   | 事件扩展内容          | Iceberg         |
| occurred_at    | TIMESTAMPTZ| 是   | 业务发生时间          | ES / Iceberg    |
| created_at     | TIMESTAMPTZ| 是   | 入库时间              | Iceberg         |
| publish_state  | TEXT       | 是   | 状态[pending...]      | PostgreSQL      |
| published_at   | TIMESTAMPTZ| 否   | 同步完成时间          | PostgreSQL      |

**典型事件类型：**
- mcap_ingested, asset_created, asset_updated, tag_upserted, tag_deleted, algo_started, algo_finished, algo_failed, delivery_created, delivery_item_added, delivery_completed, dataset_snapshot_created, training_run_started, training_run_finished

**设计说明：**
- 当前态表变更时，同事务追加 asset_events
- publish_state=pending 可异步同步 ES
- Iceberg 可用 CDC/批同步读取事件

---

### 4.6 deliveries（客户交付批次当前态）

| 字段名             | 类型        | 必填 | 说明              | 同步目标        |
|--------------------|------------|------|-------------------|-----------------|
| delivery_id        | UUID       | 是   | 交付批次主键      | ES / Iceberg    |
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

| 字段名      | 类型         | 必填 | 说明              | 同步目标      |
|-------------|-------------|------|-------------------|---------------|
| dataset_id  | UUID        | 是   | 数据集 ID         | ES / Iceberg  |
| snapshot_id | UUID        | 是   | 快照 ID           | ES / Iceberg  |
| snapshot_version | TEXT    | 否   | 快照版本号，如 v1 / 2026-04-27 | ES / Iceberg |
| name        | TEXT        | 是   | 数据集名称        | ES / Iceberg  |
| description | TEXT        | 否   | 描述              | ES / Iceberg  |
| created_by  | TEXT        | 否   | 创建人            | ES / Iceberg  |
| query_spec  | JSONB       | 是   | 构建条件          | Iceberg       |
| source_query_hash | TEXT   | 否   | 查询条件 hash，用于判断快照是否可复现/可比较 | Iceberg |
| manifest_uri| TEXT        | 是   | 清单地址          | Iceberg       |
| item_count  | BIGINT      | 是   | asset 数          | ES / Iceberg  |
| status      | TEXT        | 是   | 状态[building...] | ES / Iceberg  |
| created_at  | TIMESTAMPTZ | 是   | 创建时间          | ES / Iceberg  |
| updated_at  | TIMESTAMPTZ | 是   | 更新时间          | ES / Iceberg  |

**索引推荐**：name，created_at DESC，status

**设计说明**：快照大明细仅建议 Iceberg 存储，PG 留 manifest/摘要。

---

### 4.9 training_runs（训练任务记录）

| 字段名       | 类型         | 必填 | 说明                | 同步目标      |
|--------------|-------------|------|---------------------|---------------|
| training_run_id| UUID      | 是   | 训练任务 ID         | ES / Iceberg  |
| dataset_id   | UUID        | 是   | 数据集 ID           | ES / Iceberg  |
| snapshot_id  | UUID        | 是   | 数据集快照 ID       | ES / Iceberg  |
| model_name   | TEXT        | 是   | 模型名称            | ES / Iceberg  |
| model_version| TEXT        | 否   | 模型版本            | ES / Iceberg  |
| algo_name    | TEXT        | 否   | 训练/评测算法       | ES / Iceberg  |
| code_version | TEXT        | 否   | 训练代码版本，如 git commit / image tag | Iceberg |
| config_uri   | TEXT        | 否   | 训练配置文件地址    | Iceberg |
| data_manifest_uri | TEXT   | 否   | 本次训练实际使用的数据清单地址 | Iceberg |
| status       | TEXT        | 是   | 状态                | ES / Iceberg  |
| metrics      | JSONB       | 是   | 训练指标摘要        | Iceberg       |
| artifact_uri | TEXT        | 否   | 模型产物地址        | Iceberg       |
| started_at   | TIMESTAMPTZ | 否   | 开始时间            | ES / Iceberg  |
| finished_at  | TIMESTAMPTZ | 否   | 完成时间            | ES / Iceberg  |
| created_at   | TIMESTAMPTZ | 是   | 创建时间            | ES / Iceberg  |
| updated_at   | TIMESTAMPTZ | 是   | 更新时间            | ES / Iceberg  |

**设计说明**  
- “某次训练用了哪些 asset”：查 training_runs -> dataset_snapshots.manifest_uri 再查 Iceberg 明细表。

---

### 4.10 后训练扩展表（feature_sets / feature_jobs / training_sample_exports）

这几张表用于长期演进，当前 MVP 可以先不落地。设计原则是：PostgreSQL 只保存定义、任务状态、manifest 和摘要，大规模样本明细、特征明细、训练样本行写入 Iceberg 或对象存储。

#### feature_sets（特征集合定义）

- **主键**：feature_set_id

| 字段名 | 类型 | 必填 | 说明 | 同步目标 |
|--------|------|------|------|----------|
| feature_set_id | UUID | 是 | 特征集合 ID | ES / Iceberg |
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
| feature_set_id | UUID | 是 | 关联 feature_sets | ES / Iceberg |
| feature_set_version | TEXT | 是 | 特征集合版本 | ES / Iceberg |
| job_type | TEXT | 是 | extract / backfill / branch_experiment / merge | ES / Iceberg |
| branch_name | TEXT | 否 | 实验分支名称，用于特征调研 | ES / Iceberg |
| source_snapshot_id | UUID | 否 | 来源数据集快照 | Iceberg |
| run_id | TEXT | 否 | Spark / Ray / Dagster 执行批次 | Iceberg |
| input_manifest_uri | TEXT | 否 | 输入 asset 清单 | Iceberg |
| output_table | TEXT | 否 | 输出 Iceberg 表名 | Iceberg |
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
| dataset_id | UUID | 是 | 数据集 ID | ES / Iceberg |
| snapshot_id | UUID | 是 | 数据集快照 ID | ES / Iceberg |
| feature_set_id | UUID | 否 | 使用的特征集合 | ES / Iceberg |
| feature_set_version | TEXT | 否 | 特征集合版本 | ES / Iceberg |
| sample_table | TEXT | 否 | 训练样本 Iceberg 表名 | Iceberg |
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
- 特征调研可以先写实验 branch_name 或实验 sample_table，验证后再升级为正式 feature_set_version。
- PostgreSQL 只记录任务和 manifest，训练样本行级明细由 Iceberg 承载。

---

### 4.11 idempotency_keys（幂等写入保护）

| 字段名        | 类型         | 必填 | 说明                   | 同步目标 |
|---------------|-------------|------|------------------------|----------|
| key           | TEXT        | 是   | 幂等 key               | 不同步   |
| resource_type | TEXT        | 是   | 资源类型               | 不同步   |
| resource_id   | TEXT        | 否   | 资源 ID                | 不同步   |
| request_hash  | TEXT        | 否   | 请求 hash              | 不同步   |
| response_body | JSONB       | 否   | 首次响应缓存           | 不同步   |
| status_code   | INT         | 否   | 首次响应状态码         | 不同步   |
| created_at    | TIMESTAMPTZ | 是   | 创建时间               | 不同步   |
| expires_at    | TIMESTAMPTZ | 否   | 过期时间               | 不同步   |

**索引推荐**：expires_at

---

## 5. ES / OpenSearch 文档设计

- **ES 只做检索，非权威主库。**
- 推荐主表：asset_search_docs，主键：asset_id
- **建议文档结构：**

```json
{
  "asset_id": "uuid",
  "mcap_file_id": "uuid",
  "asset_type": "segment",
  "city": "shanghai",
  "road_type": "urban",
  "weather": "rain",
  "time_of_day": "night",
  "scenario_type": "cut_in",
  "quality_level": "good",
  "processing_state": "ready",
  "review_state": "approved",
  "delivery_state": "undelivered",
  "tags":     { "quality": "good", "has_pedestrian": "true" },
  "algos":    { "hand_tracking": { "version":"1.2.0", "status":"ok", "score":0.96 } },
  "text":     "rain night cut in pedestrian",
  "thumb_uri": "s3://...",
  "updated_at": "2026-04-27T00:00:00Z"
}
```

- **同步来源参考表：**

| ES 字段     | PostgreSQL 来源          |
|-------------|-------------------------|
| 基础字段    | assets                  |
| 采集字段    | mcap_files              |
| tags        | asset_tags              |
| algos       | asset_algo_latest       |
| 删除状态    | assets.is_deleted       |
| 更新时间    | assets.updated_at / asset_events.occurred_at |

- **查询原则：**
  - 复杂搜索走 ES，结果 asset_id 列表
  - 权威详情回查 PostgreSQL（assets/tags/algo_latest）
  - 历史分析走 Trino 查询 Iceberg

---

## 6. Iceberg 表映射

**Bronze 层**

| Iceberg 表                  | 来源表           | 用途             |
|-----------------------------|------------------|------------------|
| bronze_mcap_files           | mcap_files       | 原始文件快照     |
| bronze_assets               | assets           | 资产快照         |
| bronze_asset_tags           | asset_tags       | tag 快照         |
| bronze_asset_algo_latest    | asset_algo_latest| 算法快照         |
| bronze_asset_events         | asset_events     | 事件事实         |
| bronze_asset_relations      | asset_relations  | 资产血缘         |
| bronze_deliveries           | deliveries       | 交付批次         |
| bronze_delivery_items       | delivery_items   | 交付明细         |
| bronze_dataset_snapshots    | dataset_snapshots| 数据集快照元信息 |
| bronze_training_runs        | training_runs    | 训练记录         |
| bronze_feature_sets         | feature_sets     | 特征集合定义     |
| bronze_feature_jobs         | feature_jobs     | 特征任务记录     |

**Silver 层**

| Iceberg 表                  | 用途                   |
|-----------------------------|------------------------|
| silver_assets_current       | 资产当前态宽表         |
| silver_asset_tag_history    | tag 历史变化           |
| silver_asset_algo_runs      | 算法运行历史           |
| silver_asset_lineage        | 资产父子血缘           |
| silver_delivery_items       | 客户交付明细           |
| silver_training_dataset_usage|训练任务与数据集关系    |
| silver_feature_jobs         | 特征任务历史           |
| silver_feature_samples      | 标准化特征样本         |

**Gold 层**

| Iceberg 表                   | 用途                                     |
|------------------------------|------------------------------------------|
| gold_dataset_snapshot_items  | 数据集快照 asset 明细                    |
| gold_training_samples        | 训练可直接读取的样本表                   |
| gold_feature_samples         | 特征工程与特征调研样本表                 |
| gold_eval_samples            | 评测样本表                               |
| gold_feature_branch_samples  | 实验分支特征样本表                       |
| gold_recompute_candidates    | 算法版本变动后重算候选                   |
| gold_customer_delivery_replay| 客户交付回放                             |
| gold_quality_distribution    | 质量分布统计                             |

---

## 7. 典型问题与推荐查询路径一览

| 问题                       | 推荐查询路径                        |
|----------------------------|-------------------------------------|
| 单个 asset 详情            | PostgreSQL                          |
| 资产列表基础筛选           | PostgreSQL                          |
| 多字段模糊检索和 facets    | ES / OpenSearch                     |
| 哪些 asset 跑过某个算法    | asset_algo_latest，历史用 Trino      |
| 某 tag 何时被算法追加      | asset_events，长期历史用 Trino       |
| 某次训练用了哪些 asset     | PG 找 snapshot，Iceberg 查明细       |
| 某次训练实际读取哪些样本文件 | PG 查 training_sample_exports，训练读 manifest |
| 新特征如何做实验回填       | PG 查 feature_jobs，Iceberg 写 feature branch/sample 表 |
| 某算法版本变更后要重算     | Trino + Iceberg                     |
| 上月 MCAP segment 质量分布 | Trino + Iceberg                     |
| 某客户交付是否能完整回放   | PG 查 delivery，Iceberg 查明细       |
