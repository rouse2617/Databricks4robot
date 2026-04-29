# Schema Reference (PostgreSQL)

> **主键约定**：所有 `*_id` 列类型为 PostgreSQL `UUID`，由 Backend 应用层用 **UUIDv7**（时序前缀 + 随机后缀）颁发；存量 v4 与新增 v7 在 PG 中共存，不区分、不 backfill。

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

## 上线优先级

判断标准：**这张表不在，资产平台核心闭环（导入 → 切分 → 加 tag/算法 → 检索 → 交付）能不能跑**。

### 🟢 Tier 1 — 上线就得有（schema + backend 已实现，未投产）

| 表 | 必要性 |
|----|--------|
| `mcap_files` | 没它没有原料 |
| `assets` | 平台核心实体，所有功能都挂在这上面 |
| `deliveries` | 商业闭环的"出口" |
| `delivery_items` | M:N 明细，"召回某批 / 客户收过哪些"的依据 |
| `idempotency_keys` | API 防重复，第一天就要 |

这五张表 DDL 已落地、后端读写通路已打通，但当前还在 1.0 内部阶段，**尚未正式投产**。**上线前需要做的不是新建，而是**把字段提升到目标形态：
- `assets / mcap_files` 加 `asset_type / lifecycle_state / end_timestamp_ns / duration_ms / owner / retention_tier / expire_at` 标量列
- `assets.lifecycle_state` 和 `status` 双写一段时间，前端列表筛选切到 `lifecycle_state` 后下线 `status`

### 🟢 Tier 1+ — 已建已用（1.0 当前态）

| 表 | 状态 |
|----|------|
| `asset_tags` | **已上线**——后端唯一 tag 写入路径 |
| `asset_algo_latest` | **已上线**——后端唯一算法当前态投影 |
| `asset_events` | **outbox 起点**。ES 同步、Iceberg 入湖、审计、回放全靠它；没它就只能用 `assets.updated_at` 拉同步，会漏事件、不能回放、审计断链 |

> ⚠️ `asset_events` 上线**第一天**就要带 `event_seq` 和 `payload_schema_version`，否则后续添加是破坏性变更，需要补 backfill。

当前状态：后端已完成切换——`asset_tags / asset_algo_latest` 为唯一写入路径，`asset_events` 为统一事件表。

### 🟠 Tier 3 — 推荐但不阻塞上线

| 表 | 判断逻辑 |
|----|---------|
| `asset_relations` | 普通"一父多子"切分用 `assets.parent_asset_id` 就够；多父 / 融合 / 拼接出现前不必上 |
| `asset_algo_events` | Phase 0 遗留：**保留只读**，新事件统一走 `asset_events`；backfill 完成后 drop |

### 🔵 Tier 4 — 第二阶段再做（Phase 2+）

| 表 | 触发条件 |
|----|---------|
| `datasets` + `dataset_snapshots` | 开始做"可复现训练数据集"流程时再加 |
| `training_runs` | 有训练任务接入需要审计/复算时再加 |
| `catalog_objects` + `catalog_object_versions` | 出现多 provider（湖表 / ES index / Lance dataset）需要统一注册时再加；只有一份 Iceberg 时不必 |

这些表的共同特征是：**业务逻辑还没真正接入它们**。提前建只会晾着不维护，schema 漂移风险更高。

### ⛔ Tier 5 — 现在别动

`feature_sets` / `feature_jobs` / `training_sample_exports` —— 字段未冻结，直接照抄会被未来变更打脸。

### 上线最小 checklist

| 步骤 | 涉及表 | 状态 |
|------|--------|------|
| 1 | `assets` / `mcap_files` | 字段提升（`asset_type / lifecycle_state / duration_ms / owner / retention_tier / expire_at`）—— 进行中 |
| 2 | `asset_tags` | **已上线**——后端唯一 tag 写入路径 |
| 3 | `asset_algo_latest` | **已上线**——后端唯一算法投影路径 |
| 4 | `asset_events` | **已上线**——统一事件表（带 `event_seq` + `payload_schema_version`），Outbox Worker 消费从 2.0 开始 |
| 5 | `assets.lifecycle_state` | 与 `status` 双写中；前端列表过滤切到 `lifecycle_state` 后停写 `status` |

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

### `tag_key` 与 `tag_registry.yaml`（什么是「注册」）

| 概念 | 说明 |
|------|------|
| **`tag_key`** | 表 `asset_tags.tag_key` 一列，即「标签的名字」字符串，例如 `priority`、`quality`、`scene`。与 Grace `collection_meta` 等里的键**不是自动同名**，集成时要在字段字典里显式映射。 |
| **注册** | 指该 key 出现在仓库源文件 **[`backend/config/tag_registry.yaml`](../../backend/config/tag_registry.yaml)** 的 `tags:` 下。每个 key 对应一段定义：`description`、`type`（`enum` / `string` 等），`enum` 还带合法 `values` 白名单，`string` 可带 `max_length`。 |
| **运行时行为** | 服务端启动时加载 YAML；支持**热重载**（改文件后由 config watcher 重新加载，见后端实现）。对走 **tag 校验** 的写路径（如创建/更新资产时 body 里的 `tags`）：**`key` 必须在注册表里**，否则 API 返回校验错误；**`enum` 的 `value` 必须在 `values` 列表里**；**`string` 超长**会拒绝。 |
| **读路径** | **`GET /api/v1/tag-registry`** 把当前注册表下发给前端/SDK，用于下拉选项、表单校验，与后端规则一致。 |
| **新增业务 tag** | 1）编辑 `tag_registry.yaml` 增加 key 与类型/枚举；2）发版或热重载；3）必要时更新前端 facet / ES mapping（若该 tag 要进搜索聚合）。**不要**只往 DB 里插未注册的 key 却期望写 API 一定成功——是否允许未注册 key 以代码为准，当前主路径是**白名单**。 |

### 字段表

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| asset_id | UUID | 是 | 资产 ID |
| tag_key | TEXT | 是 | tag 名；**须与 `tag_registry.yaml` 中某一顶层 key 一致**（通过 API 写入时由后端校验） |
| tag_value | TEXT | 是 | tag 值（统一字符串表达） |
| tag_value_num | DOUBLE PRECISION | 否 | 数值型 tag，用于范围过滤 |
| tag_value_bool | BOOLEAN | 否 | 布尔型 tag |
| tag_type | TEXT | 是 | string / number / bool / enum |
| source_type | TEXT | 是 | human / algo / rule / system |
| source_name | TEXT | 否 | 算法/规则/人工名 |
| source_version | TEXT | 否 | 算法/规则版本 |
| run_id | TEXT | 否 | 外部批次 |
| confidence | DOUBLE PRECISION | 否 | 置信度 |
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
| status | TEXT | 是 | pending / running / ok / failed / blocked |
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
| event_source | TEXT | 是 | backend / worker / cron / system |
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
| retry_count | INT | 是 | 投递重试次数（默认 0） |
| last_error | TEXT | 否 | 最近一次投递失败的错误信息 |

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
| source_query_hash | TEXT | 否 | 查询条件 hash；UNIQUE 约束为 `(dataset_id, source_query_hash)`，即同一 dataset 内相同查询条件只创建一份快照 |
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

## outbox_sink_cursors

Outbox Worker 各 sink 的消费进度。

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| sink_name | TEXT | 是 | sink 标识（es / iceberg_bronze / vector） |
| last_published_seq | BIGINT | 是 | 该 sink 最后成功消费的 event_seq |
| updated_at | TIMESTAMPTZ | 是 | 更新时间 |

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
