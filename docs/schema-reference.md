# Schema Reference (PostgreSQL)

> 精简表 / 字段速查；只保留字段定义和必要说明，**不含设计取舍 / 架构解释 / 迁移路径**。
> 完整背景见 [`docs/sql.md`](./sql.md)；可执行 DDL 见 [`schemas/pg-phase0.sql`](../schemas/pg-phase0.sql)。

---

## 表清单

| 表名 | 类型 | 主键 | 一句话职责 |
|------|------|------|------------|
| [mcap_files](#mcap_files) | 主表 | `mcap_file_id` | 原始 MCAP 文件当前态 |
| [assets](#assets) | 主表 | `asset_id` | 资产当前态（segment / clip / frame_set / derived_asset） |
| [asset_relations](#asset_relations) | 关系表 | `(parent_asset_id, child_asset_id, relation_type)` | 复杂血缘（多父 / 融合 / 拼接 / 派生） |
| [asset_tags](#asset_tags) | 投影表 | `(asset_id, tag_key)` | 资产 tag 当前态，驱动 facet/filter/ES 文档 |
| [asset_algo_latest](#asset_algo_latest) | 投影表 | `(asset_id, algo_name)` | 每 (asset, algo) 最新一行算法状态 |
| [asset_events](#asset_events) | 事件表 | `event_id` (UNIQUE `event_seq`) | 统一业务事件 / 审计 / outbox |
| [asset_algo_events](#asset_algo_events) | 遗留 | `event_id` | Phase 0 算法状态变更审计；Phase 1 后被 asset_events 取代 |
| [deliveries](#deliveries) | 主表 | `delivery_id` | 客户交付批次当前态 |
| [delivery_items](#delivery_items) | 关联表 | `(delivery_id, asset_id)` | Delivery ↔ Asset 明细 |
| [datasets](#datasets) | 主表 | `dataset_id` | 数据集定义 |
| [dataset_snapshots](#dataset_snapshots) | 主表 | `(dataset_id, snapshot_id)` | 训练/评测数据集快照元信息 |
| [training_runs](#training_runs) | 主表 | `training_run_id` | 训练任务记录（自包含 catalog 引用） |
| [catalog_objects](#catalog_objects) | Catalog | `object_id` (UNIQUE 四元组) | 中立对象注册（湖表 / PG 表 / ES index / Lance dataset…） |
| [catalog_object_versions](#catalog_object_versions) | Catalog | `object_version_id` | 对象版本引用（Iceberg snapshot / Lance version / …） |
| [idempotency_keys](#idempotency_keys) | 控制表 | `(scope, idem_key)` | API 幂等保护 |

后训练扩展（**字段未冻结，落地前重新评审**）：`feature_sets` / `feature_jobs` / `training_sample_exports`。

---

## mcap_files

原始 MCAP 文件当前态。

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| mcap_file_id | UUID | 是 | 文件主键 |
| raw_hash_md5 | TEXT | 否 | 文件 MD5；`is_deleted=FALSE` 范围 UNIQUE，重复 ingest 走幂等 |
| raw_hash_sha256 | TEXT | 否 | 长期内容指纹 |
| mcap_uri | TEXT | 是 | MCAP 对象存储地址 |
| size_bytes | BIGINT | 否 | 文件大小 |
| file_duration_ms | BIGINT | 否 | 文件总时长 |
| start_timestamp_ns | BIGINT | 否 | 文件起始时间 |
| end_timestamp_ns | BIGINT | 否 | 文件结束时间 |
| channel_count | INT | 否 | MCAP channel 数 |
| chunk_count | INT | 否 | MCAP chunk 数 |
| ingest_state | TEXT | 是 | pending / ingesting / ready / failed |
| vendor_id | TEXT | 否 | 数据供应商 |
| collector_id | TEXT | 否 | 采集方 |
| task_id | TEXT | 否 | 采集任务 |
| device_id | TEXT | 否 | 设备 ID |
| camera_model | TEXT | 否 | 相机型号 |
| data_source | TEXT | 否 | 数据来源 |
| location_id | TEXT | 否 | 位置 ID |
| scene_id | TEXT | 否 | 场景 ID |
| environment_id | TEXT | 否 | 环境 ID |
| collection_method | TEXT | 否 | 采集方式 |
| tenant_id | TEXT | 否 | 租户 ID（单租户阶段默认 `_default`） |
| project_id | TEXT | 否 | 项目 ID |
| owner | TEXT | 否 | 数据归属方 |
| retention_tier | TEXT | 否 | hot / warm / cold / archive |
| expire_at | TIMESTAMPTZ | 否 | 过期/可清理时间 |
| metadata | JSONB | 是 | 低频扩展元数据 |
| process_state | JSONB | 是 | 文件级处理状态扩展 |
| is_deleted | BOOLEAN | 是 | 软删除 |
| created_at | TIMESTAMPTZ | 是 | 创建时间 |
| updated_at | TIMESTAMPTZ | 是 | 更新时间 |
| version | BIGINT | 是 | 乐观锁版本 |

---

## assets

资产当前态。行业相关 facet（city / weather / scenario_type 等）**不进本表**，统一进 `asset_tags`。

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| asset_id | UUID | 是 | 资产主键 |
| mcap_file_id | UUID | 是 | 来源 MCAP 文件 ID |
| asset_type | TEXT | 是 | segment / clip / frame_set / derived_asset |
| storage_uri | TEXT | 否 | asset 本体对象存储地址 |
| thumb_uri | TEXT | 否 | 缩略图地址 |
| parent_asset_id | UUID | 否 | 父资产 ID |
| root_asset_id | UUID | 否 | 根资产 ID |
| asset_level | INT | 是 | 资产层级（0=原始） |
| split_method | TEXT | 否 | manual / algo / rule / pipeline |
| split_algo_name | TEXT | 否 | 切分算法名称 |
| split_algo_version | TEXT | 否 | 切分算法版本 |
| split_run_id | TEXT | 否 | 切分批次 ID |
| split_reason | TEXT | 否 | 切分理由 |
| segment_index | INT | 否 | 父资产下片段序号 |
| parent_start_offset_ms | BIGINT | 否 | 相对父资产起始偏移 |
| parent_end_offset_ms | BIGINT | 否 | 相对父资产结束偏移 |
| start_timestamp_ns | BIGINT | 否 | asset 起始时间 |
| end_timestamp_ns | BIGINT | 否 | asset 结束时间 |
| duration_ms | BIGINT | 否 | asset 时长 |
| lifecycle_state | TEXT | 是 | created / processing / ready / rejected / delivered / archived / superseded |
| tenant_id | TEXT | 否 | 租户 ID |
| project_id | TEXT | 否 | 项目 ID |
| owner | TEXT | 否 | 资产 owner |
| reviewer | TEXT | 否 | 审核人 |
| last_delivered_at | TIMESTAMPTZ | 否 | 最近交付时间（冗余，由 delivery usecase 同事务刷新） |
| last_delivered_to | TEXT | 否 | 最近交付客户（冗余） |
| delivery_count | INT | 是 | 累计交付次数（冗余） |
| retention_tier | TEXT | 否 | hot / warm / cold / archive |
| expire_at | TIMESTAMPTZ | 否 | 过期/可清理时间 |
| metadata | JSONB | 是 | 低频扩展元数据 |
| files | JSONB | 是 | 关联文件（thumbnail、algo output…） |
| is_deleted | BOOLEAN | 是 | 软删除 |
| created_at | TIMESTAMPTZ | 是 | 创建时间 |
| updated_at | TIMESTAMPTZ | 是 | 更新时间 |
| version | BIGINT | 是 | 乐观锁版本 |

---

## asset_relations

复杂资产血缘（多父 / 融合 / 拼接 / 采样）。一父多子普通切分用 `assets.parent_asset_id` 即可。

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| parent_asset_id | UUID | 是 | 父资产 ID |
| child_asset_id | UUID | 是 | 子资产 ID |
| relation_type | TEXT | 是 | split_from / derived_from / merged_from / sampled_from |
| method | TEXT | 否 | manual / algo / rule / pipeline |
| algo_name | TEXT | 否 | 算法名称 |
| algo_version | TEXT | 否 | 算法版本 |
| run_id | TEXT | 否 | 外部执行批次 |
| parent_start_offset_ms | BIGINT | 否 | 相对父资产起始偏移 |
| parent_end_offset_ms | BIGINT | 否 | 相对父资产结束偏移 |
| created_at | TIMESTAMPTZ | 是 | 创建时间 |

---

## asset_tags

资产 tag 当前态投影。一 asset 多 tag，多行设计。

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| asset_id | UUID | 是 | 资产 ID |
| tag_key | TEXT | 是 | tag 名称（在 `tag_registry.yaml` 注册） |
| tag_value | TEXT | 是 | tag 值（统一字符串表达） |
| tag_value_num | DOUBLE PRECISION | 否 | 数值型 tag，用于范围过滤 |
| tag_value_bool | BOOLEAN | 否 | 布尔型 tag |
| tag_type | TEXT | 是 | string / number / bool / enum |
| source_type | TEXT | 是 | human / algo / rule / system |
| source_name | TEXT | 否 | 算法/规则/人工名 |
| source_version | TEXT | 否 | 算法/规则版本 |
| run_id | TEXT | 否 | 外部批次 |
| confidence | DOUBLE PRECISION | 否 | 置信度 |
| tenant_id | TEXT | 否 | 租户 ID（冗余） |
| project_id | TEXT | 否 | 项目 ID（冗余） |
| created_at | TIMESTAMPTZ | 是 | 首次创建时间 |
| updated_at | TIMESTAMPTZ | 是 | 最近更新时间 |

---

## asset_algo_latest

每 (asset, algo) 最新一行算法状态。完整生命周期写 `asset_events`。

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| asset_id | UUID | 是 | 资产 ID |
| algo_name | TEXT | 是 | 算法名称 |
| algo_version | TEXT | 是 | 算法版本 |
| status | TEXT | 是 | pending / running / ok / failed / blocked / skipped |
| result_tag | TEXT | 否 | 算法输出标签 |
| result_score | DOUBLE PRECISION | 否 | 算法分数 |
| result_summary | JSONB | 是 | 低频结果摘要 |
| run_id | TEXT | 否 | 执行批次 |
| method | TEXT | 否 | 执行方式 |
| model_uri | TEXT | 否 | 模型文件地址 |
| output_uri | TEXT | 否 | 输出地址 |
| error_code | TEXT | 否 | 错误码 |
| error_message | TEXT | 否 | 失败原因 |
| started_at | TIMESTAMPTZ | 否 | 开始时间 |
| finished_at | TIMESTAMPTZ | 否 | 完成时间 |
| tenant_id | TEXT | 否 | 租户 ID（冗余） |
| project_id | TEXT | 否 | 项目 ID（冗余） |
| updated_at | TIMESTAMPTZ | 是 | 更新时间 |

---

## asset_events

统一业务事件 / 审计 / outbox。**消费者按 `event_seq` 严格递增推进 watermark**，不能用 `occurred_at`。

典型事件类型：`mcap_ingested` · `asset_created` · `asset_updated` · `asset_lifecycle_changed` · `tag_upserted` · `tag_deleted` · `algo_started` · `algo_finished` · `algo_failed` · `delivery_created` · `delivery_item_added` · `delivery_completed` · `dataset_snapshot_created` · `training_run_started` · `training_run_finished`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| event_id | UUID | 是 | 事件主键 |
| event_seq | BIGSERIAL | 是 | 单调递增序号，UNIQUE；消费 watermark 用 |
| event_type | TEXT | 是 | 事件类型 |
| payload_schema_version | TEXT | 是 | event_payload schema 版本（如 v1 / v2） |
| asset_id | UUID | 否 | 关联 asset |
| mcap_file_id | UUID | 否 | 关联 MCAP |
| tenant_id | TEXT | 否 | 租户 ID |
| project_id | TEXT | 否 | 项目 ID |
| event_source | TEXT | 是 | backend / worker / dagster / spark / daft / system |
| actor_type | TEXT | 否 | user / service / algo / system |
| actor_id | TEXT | 否 | 操作者 |
| request_id | TEXT | 否 | 请求追踪 |
| idempotency_key | TEXT | 否 | 幂等 key |
| run_id | TEXT | 否 | 外部执行批次 |
| occurred_at | TIMESTAMPTZ | 是 | 业务发生时间 |
| created_at | TIMESTAMPTZ | 是 | 入库时间 |
| publish_state | TEXT | 是 | pending / published / failed |
| published_at | TIMESTAMPTZ | 否 | 同步完成时间 |
| event_payload | JSONB | 是 | 类型相关字段（按 `payload_schema_version` 解析） |

---

## asset_algo_events

> Phase 0 遗留：算法状态变更审计；Phase 1 后被 `asset_events` 取代，新代码不要写。

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| event_id | UUID | 是 | 事件主键 |
| asset_id | UUID | 是 | 资产 ID |
| algo_key | TEXT | 是 | 算法 key（`<name>@<version>`） |
| prev_status | TEXT | 否 | 转换前状态 |
| new_status | TEXT | 是 | 转换后状态 |
| run_id | TEXT | 否 | 外部批次 |
| reason | TEXT | 否 | 转换原因 |
| created_at | TIMESTAMPTZ | 是 | 创建时间 |

---

## deliveries

客户交付批次当前态。

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| delivery_id | UUID | 是 | 交付批次主键 |
| tenant_id | TEXT | 否 | 租户 ID |
| project_id | TEXT | 否 | 项目 ID |
| customer_id | TEXT | 是 | 客户 ID |
| contract_id | TEXT | 否 | 合同/订单号 |
| delivery_type | TEXT | 是 | asset_set / replay / dataset / … |
| status | TEXT | 是 | pending / delivered / accepted / rejected / recalled |
| requested_by | TEXT | 否 | 申请人 |
| approved_by | TEXT | 否 | 审批人 |
| delivered_by | TEXT | 否 | 交付人 |
| manifest_uri | TEXT | 否 | 交付清单 |
| replay_manifest_uri | TEXT | 否 | 回放清单 |
| item_count | BIGINT | 是 | 交付 asset 数量 |
| total_size_bytes | BIGINT | 否 | 交付总大小 |
| delivered_at | TIMESTAMPTZ | 否 | 交付时间 |
| completed_at | TIMESTAMPTZ | 否 | 完成时间 |
| metadata | JSONB | 是 | 低频扩展元数据 |
| is_deleted | BOOLEAN | 是 | 软删除 |
| created_at | TIMESTAMPTZ | 是 | 创建时间 |
| updated_at | TIMESTAMPTZ | 是 | 更新时间 |
| version | BIGINT | 是 | 乐观锁版本 |

---

## delivery_items

Delivery ↔ Asset M:N 明细。

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| delivery_id | UUID | 是 | 交付批次 ID |
| asset_id | UUID | 是 | 资产 ID |
| asset_version | BIGINT | 否 | 交付时资产版本（快照） |
| item_state | TEXT | 是 | pending / delivered / failed |
| checksum | TEXT | 否 | 导出文件校验 |
| export_uri | TEXT | 否 | 导出对象地址 |
| created_at | TIMESTAMPTZ | 是 | 创建时间 |

---

## datasets

数据集定义父表。明细在 `dataset_snapshots` + Iceberg。

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| dataset_id | UUID | 是 | 数据集 ID |
| name | TEXT | 是 | 数据集名称 |
| description | TEXT | 否 | 描述 |
| owner | TEXT | 否 | 负责人或团队 |
| tenant_id | TEXT | 否 | 租户 ID |
| project_id | TEXT | 否 | 项目 ID |
| dataset_type | TEXT | 是 | training / eval / replay / delivery / experiment |
| status | TEXT | 是 | active / archived / deleted |
| created_by | TEXT | 否 | 创建人 |
| created_at | TIMESTAMPTZ | 是 | 创建时间 |
| updated_at | TIMESTAMPTZ | 是 | 更新时间 |

---

## dataset_snapshots

数据集快照元信息。**被 `training_runs` / `training_sample_exports` / `asset_events` 引用后禁止 DELETE，只能 `status='archived'`**。

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| dataset_id | UUID | 是 | 数据集 ID |
| snapshot_id | UUID | 是 | 快照 ID |
| snapshot_version | TEXT | 否 | 快照版本号（如 v1 / 2026-04-27） |
| created_by | TEXT | 否 | 创建人 |
| query_spec | JSONB | 是 | 构建条件 |
| source_query_hash | TEXT | 否 | 查询条件 hash（判断快照可复现/可比较） |
| source_catalog_name | TEXT | 否 | 来源 Catalog（如 lakehouse） |
| source_namespace | TEXT | 否 | 来源 namespace（如 robot.gold） |
| source_object_name | TEXT | 否 | 来源表/视图名 |
| source_object_version_ref | TEXT | 否 | 来源对象版本（Iceberg snapshot_id 等） |
| manifest_uri | TEXT | 是 | 清单地址 |
| item_count | BIGINT | 是 | asset 数 |
| status | TEXT | 是 | building / ready / failed / archived |
| created_at | TIMESTAMPTZ | 是 | 创建时间 |
| updated_at | TIMESTAMPTZ | 是 | 更新时间 |

---

## training_runs

训练任务记录。**自包含 catalog 引用，不强依赖 `dataset_snapshots` 的物理 FK**。

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| training_run_id | UUID | 是 | 训练任务 ID |
| tenant_id | TEXT | 否 | 租户 ID |
| project_id | TEXT | 否 | 项目 ID |
| dataset_id | UUID | 是 | 数据集 ID（软引用） |
| snapshot_id | UUID | 是 | 快照 ID（软引用） |
| model_name | TEXT | 是 | 模型名称 |
| model_version | TEXT | 否 | 模型版本 |
| algo_name | TEXT | 否 | 训练/评测算法 |
| code_version | TEXT | 否 | git commit / image tag |
| config_uri | TEXT | 否 | 训练配置地址 |
| data_manifest_uri | TEXT | 否 | 实际读取的数据清单 |
| data_catalog_name | TEXT | 否 | 数据所在 Catalog |
| data_namespace | TEXT | 否 | 数据 namespace |
| data_object_name | TEXT | 否 | 训练数据对象名 |
| data_object_version_ref | TEXT | 否 | 训练数据版本 |
| status | TEXT | 是 | pending / running / succeeded / failed / cancelled |
| metrics | JSONB | 是 | 训练指标摘要 |
| artifact_uri | TEXT | 否 | 模型产物地址 |
| started_at | TIMESTAMPTZ | 否 | 开始时间 |
| finished_at | TIMESTAMPTZ | 否 | 完成时间 |
| created_at | TIMESTAMPTZ | 是 | 创建时间 |
| updated_at | TIMESTAMPTZ | 是 | 更新时间 |

---

## catalog_objects

中立对象注册（湖表 / PG 表 / ES 索引 / Lance dataset…）。
唯一约束：`(catalog_name, namespace, object_name, object_type)`。

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| object_id | UUID | 是 | 平台对象 ID |
| catalog_name | TEXT | 是 | online / lakehouse / search / multimodal |
| namespace | TEXT | 是 | 命名空间（如 robot.gold） |
| object_name | TEXT | 是 | 对象名 |
| object_type | TEXT | 是 | postgres_table / iceberg_table / elasticsearch_index / lance_dataset / object_prefix / manifest |
| provider | TEXT | 是 | iceberg_rest / postgres / elasticsearch / lance / polaris / gravitino / glue / dlf |
| format | TEXT | 是 | postgres / iceberg / elasticsearch / lance / parquet / webdataset / manifest |
| storage_uri | TEXT | 否 | 物理存储位置（仅运维定位） |
| external_ref | TEXT | 否 | 外部系统对象引用 |
| owner | TEXT | 否 | 负责人或团队 |
| tenant_id | TEXT | 否 | 租户 ID |
| project_id | TEXT | 否 | 项目 ID |
| description | TEXT | 否 | 描述 |
| tags | JSONB | 是 | 对象级标签（pii / training / gold / deprecated） |
| properties | JSONB | 是 | provider 扩展属性 |
| status | TEXT | 是 | active / deprecated / deleted |
| created_at | TIMESTAMPTZ | 是 | 创建时间 |
| updated_at | TIMESTAMPTZ | 是 | 更新时间 |

---

## catalog_object_versions

对象版本引用（Iceberg snapshot / Lance version / ES index generation / …）。

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| object_version_id | UUID | 是 | 对象版本 ID |
| object_id | UUID | 是 | 关联 catalog_objects.object_id |
| version_ref | TEXT | 是 | 外部版本引用（snapshot_id / lance version / index generation） |
| version_type | TEXT | 是 | iceberg_snapshot / lance_version / manifest_version / index_generation / schema_version |
| schema_ref | TEXT | 否 | schema 文件 / hash / catalog schema version |
| manifest_uri | TEXT | 否 | manifest 或 metadata 文件位置 |
| row_count | BIGINT | 否 | 行数或对象数 |
| size_bytes | BIGINT | 否 | 数据大小 |
| checksum | TEXT | 否 | 校验值 |
| created_by | TEXT | 否 | 创建人或任务 |
| created_at | TIMESTAMPTZ | 是 | 创建时间 |
| properties | JSONB | 是 | 扩展属性 |

---

## idempotency_keys

API 幂等保护。`expires_at` 由 lifecycle job 清理（`response_json` 可能含 PII）。

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| scope | TEXT | 是 | 幂等域（如 `deliveries.create`） |
| idem_key | TEXT | 是 | 幂等 key（客户端提供） |
| resource_type | TEXT | 否 | 资源类型 |
| resource_id | TEXT | 否 | 资源 ID |
| request_hash | TEXT | 否 | 请求 hash |
| response_json | JSONB | 否 | 首次响应缓存 |
| status_code | INT | 否 | 首次响应状态码 |
| created_at | TIMESTAMPTZ | 是 | 创建时间 |
| expires_at | TIMESTAMPTZ | 否 | 过期时间 |

---

## 关系总览

```text
mcap_files 1 ── N assets
assets     1 ── N asset_tags
assets     1 ── N asset_algo_latest
assets     1 ── N asset_events
assets     N ── N deliveries        (via delivery_items)
assets     N ── N assets             (via asset_relations)
datasets   1 ── N dataset_snapshots
dataset_snapshots 1 ── N training_runs    (软引用，无 FK)
catalog_objects   1 ── N catalog_object_versions
```
