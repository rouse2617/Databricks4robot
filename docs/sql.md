# 表结构速览

> **Note:** PostgreSQL is now the primary storage backend for local development. Set `STORAGE_BACKEND=postgres` (the default) and run `docker-compose up -d postgres` to get started. Bigtable is retained for backward compatibility and production use via `STORAGE_BACKEND=bigtable`.

| 表　　　　　　　 | 角色　　　　　　　　　　　　　 | 关系　　　　　　　　　　　　　|
| ------------------| --------------------------------| -------------------------------|
| `mcap_files`　　 | 原始文件元数据　　　　　　　　 | 1　　　　　　　　　　　　　　 |
| `assets`　　　　 | 有效 Segment（资产本体）　　　 | N，引用 `mcap_files`　　　　　|
| `deliveries`　　 | 一次交付事件（批次）　　　　　 | —　　　　　　　　　　　　　　 |
| `delivery_items` | asset ↔ delivery 关联表（M:N） | 连接 `assets` 与 `deliveries` |

---

## 表 1：`assets`（主表 / 资产 = Segment）

3 个列族：`cf:meta`（固定 22 字段）+ `cf:algo`（动态）+ `cf:tag`（动态）

| 列族 | 字段 | 类型 | 长度 | 作用 | 示例 | grace 对应 |
|---|---|---|---|---|---|---|
| cf:meta | `asset_id` | string (UUIDv4) | 36 | 资产主键（= Segment ID），逻辑唯一标识 | `a1b2c3d4-5e6f-4789-abcd-ef1234567890` | `annotation_segmentations.segmentation_id` |
| cf:meta | `status` | string | 32 | QA 状态 | `approved` | `annotation_segmentations.status` |
| cf:meta | `is_deleted` | bool | 1 | 软删 | `false` | `is_deleted` |
| cf:meta | `created_at` | timestamp | — | 创建时间 | `2026-04-21T08:00:00Z` | `BaseModel` |
| cf:meta | `updated_at` | timestamp | — | 更新时间 | `2026-04-21T08:10:00Z` | `BaseModel` |
| cf:meta | `start_timestamp_ns` | int64 | 8 | 段起点（纳秒） | `1700000000000000000` | `segmentations.start_timestamp` |
| cf:meta | `end_timestamp_ns` | int64 | 8 | 段终点（纳秒） | `1700000005000000000` | `segmentations.end_timestamp` |
| cf:meta | `duration_sec` | numeric | — | 段时长（秒） | `5.0` | 派生 `(end-start)/1e9` |
| cf:meta | `mcap_file_id` | string (UUIDv4) | 36 | 原始 MCAP 文件引用 | `7f8a9b2c-...` | `segmentations.video_id` |
| cf:meta | `type` | string | 32 | 片段类型 | `task_demo` | `segmentations.type` |
| cf:meta | `env` | string | 64 | 片段环境 | `kitchen` | `segmentations.env` |
| cf:meta | `task` | string | 64 | 片段任务 | `cook_pasta` | `segmentations.task` |
| cf:meta | `use_tool` | bool | 1 | 是否用工具 | `true` | `segmentations.use_tool` |
| cf:meta | `ground_truth` | bool | 1 | 是否 GT | `false` | `segmentations.ground_truth` |
| cf:meta | `recheck` | bool | 1 | 是否复检 | `false` | `segmentations.recheck` |
| cf:meta | `comments` | string | 1024 | QA 备注 | `lighting ok` | `segmentations.comments` |
| cf:meta | `template_version` | int | 4 | QA 模板版本 | `1` | `segmentations.template_version` |
| cf:meta | `annotation_run_id` | string (UUID) | 36 | 关联标注 run | `01HRUN...` | `segmentations.annotation_run_id` |
| cf:meta | `owner` | string | 64 | 资产 owner（用户/团队 URN，Phase 0 占位，Phase 1+ 接权限） | `urn:grace:user:alice` | 新增，占位 |
| cf:meta | `reviewer` | string | 64 | QA 审核人 | `alice` | 新增 |
| cf:meta | `last_delivered_at` | timestamp | — | 交付汇总：最近一次被交付的时间（NULL = 从未交付） | `2026-04-20T10:00:00Z` | 新增 |
| cf:meta | `delivery_count` | int | 4 | 交付汇总：累计被交付次数 | `3` | 新增 |
| cf:meta | `last_delivered_to` | string | 64 | 交付汇总：最近一次交付的客户 URN | `urn:grace:customer:A` | 新增 |
| cf:algo | `<algo>@<ver>:<field>` | 见表 1.1 | — | 算法处理记录（动态扁平列名） | `sam2@1.2.0:status = ok` | 新增 |
| cf:tag | `<tag_key>` | string | — | 用户/规则标签（动态列名，需在 tag_registry.yaml 注册） | `priority = A` | 新增 |
| cf:files | `<logical_name>` | string | — | 所有关联文件引用（动态列名，key=逻辑名, value=GCS URI） | `hand_tracking@1.2.0 = gs://...` | 新增 |

### 表 1.1：`cf:algo` 扁平化列约定

列名格式：`<algo_name>@<version>:<field>`，每个 field 独立 cell。

| field | 类型 | 长度 | 作用 | 示例 |
|---|---|---|---|---|
| `status` | string | 16 | 算法执行状态 | `pending` / `running` / `ok` / `failed` |
| `started_at` | timestamp | — | 开始处理时间（start 时写入） | `2026-04-21T08:05:00Z` |
| `finished_at` | timestamp | — | 完成处理时间（finish 时写入，nullable） | `2026-04-21T08:10:00Z` |
| `method` | string | 64 | 处理方法标识 | `ray_batch` / `local` |
| `run_id` | string | 64 | 外部执行批次 id（Dagster run / Ray job） | `dagster-run-abc` |
| `output_uri` | string | 512 | 产物 URI（status=ok 时写入） | `gs://derived/sam2/01HXYZ.mcap` |
| `reason` | string | 256 | 失败原因（status=failed 时写入） | `OOM at frame 1337` |

**状态机：**
```
blocked → pending → running → ok
                             → failed → pending（重试）
```

- `blocked`：有未满足的上游依赖，sensor 忽略此状态
- `pending`：所有依赖已满足（或无依赖），sensor 可以捡到
- `running`：正在处理中
- `ok`：处理成功
- `failed`：处理失败，可通过 reset 回到 pending

> **依赖解锁机制：** ingest 创建 asset 时，根据 `algo_registry.yaml` 的 `depends_on` 字段初始化状态：
> - `depends_on` 为空 → `pending`
> - `depends_on` 非空 → `blocked`
>
> `finish_algo(ok)` 时自动检查下游算法的依赖是否全部满足，满足则 `blocked → pending`。

完整列名示例：

```
cf:algo:sam2@1.2.0:status      = ok
cf:algo:sam2@1.2.0:started_at  = 2026-04-21T08:05:00Z
cf:algo:sam2@1.2.0:finished_at = 2026-04-21T08:10:00Z
cf:algo:sam2@1.2.0:method      = ray_batch
cf:algo:sam2@1.2.0:run_id      = dagster-run-abc
cf:algo:sam2@1.2.0:output_uri  = gs://derived/sam2/01HXYZ.mcap

cf:algo:tracker@2.0.0:status     = failed
cf:algo:tracker@2.0.0:started_at = 2026-04-21T09:00:00Z
cf:algo:tracker@2.0.0:finished_at= 2026-04-21T09:02:00Z
cf:algo:tracker@2.0.0:method     = ray_batch
cf:algo:tracker@2.0.0:run_id     = dagster-run-def
cf:algo:tracker@2.0.0:reason     = OOM at frame 1337
```

> **设计要点：**
> - `started_at` 和 `finished_at` 分开记录，便于计算处理耗时和排查超时
> - `pending` 状态用于 Dagster sensor 查询"待处理资产"（`filter=cf_algo.<algo>@<ver>:status:eq:pending`）
> - `method` 字段区分同一算法的不同执行方式（Ray batch vs 本地单机），便于性能分析
> - 所有字段通过 JSONB 合并（`cf_algo || jsonb_build_object(...)`）原子更新，不同算法并发完成不会互相覆盖

### 表 1.2：`cf:tag` 命名空间示例（按业务扩展）

> **注意：** 所有 tag key 必须在 `config/tag_registry.yaml` 中注册，未注册的 key 写入时会被拒绝。

| 命名空间 | 示例列名 | 列值形式 | 用途 |
|---|---|---|---|
| priority | `priority` | `critical` / `high` / `medium` / `low` | 优先级分类（enum） |
| quality | `quality` | `excellent` / `good` / `acceptable` / `poor` / `unusable` | 质量分类（enum） |
| scene | `scene` | `indoor` / `outdoor` / `warehouse` / `office` / `factory` | 采集场景（enum） |
| task | `task` | 自由文本 | 采集任务标识（string） |
| batch | `batch` | 自由文本 | 采集批次号（string） |
| notes | `notes` | 自由文本（≤500 字符） | 自由备注（string） |
| reviewed_by | `reviewed_by` | `alice` | 归属/责任人（string） |

### 表 1.3：`cf:files` 文件引用注册表

所有关联文件的 URI 收敛到一个地方，key = 逻辑名，value = GCS URI。

| 逻辑名 | 写入时机 | 示例值 |
|---|---|---|
| `raw_mcap` | ingest 时从 mcap_files.cf_meta.mcap_uri 复制 | `gs://grace-raw/mcap/2026-04/abc.mcap` |
| `<algo>@<ver>` | finish_algo(ok) 时同步写入 | `gs://grace-algo/ht/abc-seg01.npz` |
| `thumbnail` | 缩略图生成后写入 | `gs://grace-thumbs/abc-seg01.jpg` |

> **设计要点：**
> - finish_algo(ok) 时同时写 `cf_algo` 和 `cf_files`，一条 SQL 两个 JSONB merge
> - finish_algo(failed) 或 reset 时，`cf_files[algo_key] = null`
> - 一次查询拿到所有文件：`SELECT cf_files FROM assets WHERE asset_id = $1`
> - 存储治理：`SELECT asset_id, cf_files FROM assets WHERE cf_files ? 'deface@2.0.0'`
> - Phase 1 Bigtable：直接映射到 `cf:files` 列族

### 表 1.4：`cf:meta` 生命周期字段（ingest 时写入默认值）

| 字段 | 类型 | 默认值 | 写入时机 | 说明 |
|---|---|---|---|---|
| `retention_tier` | string | `standard` | ingest | `standard` / `archive` / `frozen` |
| `archive_after_days` | int | `90` | ingest | 多少天后自动降级到 Nearline/Archive |
| `delete_after_days` | int | `365` | ingest | 多少天后可以删除 |
| `total_size_bytes` | int64 | `0` | finish_algo 累加 | 该 asset 所有文件的总大小 |
| `last_accessed_at` | timestamp | `null` | GET asset 时更新 | 最后一次被访问的时间 |

> **设计要点：** 字段现在就埋，治理逻辑以后再写。成本几乎为零，省去未来加字段 + backfill 的痛苦。

---

## 表 2：`mcap_files`（引用表 / 文件级）

2 个列族：`cf:meta`（固定 20 字段）+ `cf:process`（固定 8 字段）

| 列族 | 字段 | 类型 | 长度 | 作用 | 示例 | grace 对应 |
|---|---|---|---|---|---|---|
| cf:meta | `mcap_file_id` | string (UUIDv4) | 36 | 文件主键 | `7f8a9b2c-...` | `grace_videos.id` |
| cf:meta | `d_type` | string | 16 | 数据类型 | `video` | `grace_videos.d_type` |
| cf:meta | `file_duration_sec` | numeric | — | 原文件总时长（秒） | `1800.0` | `grace_videos.duration_sec` |
| cf:meta | `is_deleted` | bool | 1 | 软删 | `false` | `is_deleted` |
| cf:meta | `created_at` | timestamp | — | 入库时间 | `2026-04-20T10:00:00Z` | `BaseModel` |
| cf:meta | `updated_at` | timestamp | — | 更新时间 | `2026-04-21T08:10:00Z` | `BaseModel` |
| cf:meta | `raw_hash_md5` | string | 32 | MCAP 文件 MD5 | `d41d8cd98f00b204...` | `grace_videos.raw_hash_md5` |
| cf:meta | `mcap_uri` | string | 512 | MCAP GCS URI | `gs://raw/x.mcap` | `grace_videos.storage_meta.uri` |
| cf:meta | `size_bytes` | int64 | 8 | 文件大小（字节） | `104857600` | `grace_videos.storage_meta.size` |
| cf:meta | `vendor_id` | string | 36 | 供应商 id | `vendor-01` | `collection_meta.vendor_id` |
| cf:meta | `collector_id` | string | 36 | 采集员 id | `u-0001` | `collection_meta.collector_id` |
| cf:meta | `task_id` | string | 36 | 采集任务 id | `task-01` | `collection_meta.task_id` |
| cf:meta | `device_id` | string | 128 | 设备序列号 | `CC2-00123` | `collection_meta.camera_serial_number` |
| cf:meta | `camera_model` | string | 128 | 相机型号 | `CyberCap2` | `grace_videos.camera_model` |
| cf:meta | `data_source` | string | 64 | 数据来源 | `internal` | `collection_meta.data_source` |
| cf:meta | `location_id` | string | 36 | 位置 id | `loc-01` | `collection_meta.location_id` |
| cf:meta | `scene_id` | string | 64 | 场景 id | `kitchen-01` | `collection_meta.scene_id` |
| cf:meta | `environment_id` | string | 64 | 环境 id | `indoor-01` | `collection_meta.environment_id` |
| cf:meta | `collection_method` | string | 32 | 采集方式 | `handheld` | `collection_meta.collection_method` |
| cf:meta | `ingest_state` | string | 32 | 入库处理状态 | `pending` / `summarized` / `failed` | 新增 |
| cf:meta | `channel_count` | int | 4 | MCAP 通道数 | `12` | 新增 |
| cf:meta | `chunk_count` | int | 4 | MCAP chunk 数 | `48` | 新增 |
| cf:meta | `owner` | string | 64 | 文件 owner（接入方/业务线 URN，Phase 0 占位） | `urn:grace:team:bay-area` | 新增，占位 |
| cf:process | `hand_tracking` | string | 32 | 手部追踪 | `completed` | `process_info.hand_tracking` |
| cf:process | `head_tracking` | string | 32 | 头部追踪 | `completed` | `process_info.head_tracking` |
| cf:process | `body_tracking` | string | 32 | 身体追踪 | `completed` | 派生自 `step_events` |
| cf:process | `deface` | string | 32 | 人脸模糊 | `completed` | `process_info.deface` |
| cf:process | `deblur` | string | 32 | 去模糊 | `completed` | `process_info.deblur` |
| cf:process | `exdel_flag` | int | 4 | 交付/独占标（0/1/2/3） | `1` | `grace_videos.exdel_flag` |
| cf:process | `archived_at` | timestamp | — | 归档时间 | `2026-04-22T00:00:00Z` | `grace_videos.archived_at` |
| cf:process | `failed_reason` | string | 256 | 处理失败原因 | `transcode_error` | `grace_videos.failed_reason` |

> **设计要点：**
> - `cf:meta` 合并 core + file + collection —— 入库时一次写齐，几乎不变
> - `cf:process` 独立：处理状态（tracking / deface / deblur …）持续更新，版本变化多，独立 CF 便于单独设 GC policy

---

## 表 3：`deliveries`（交付批次表）· 新增

1 个列族：`cf:meta`（固定 16 字段）

| 列族 | 字段 | 类型 | 长度 | 作用 | 示例 |
|---|---|---|---|---|---|
| cf:meta | `delivery_id` | string (UUIDv4) | 36 | 交付批次主键 | `d1b2c3d4-...` |
| cf:meta | `customer_id` | string | 64 | 客户 URN | `urn:grace:customer:A` |
| cf:meta | `contract_id` | string | 64 | 合同/订单号（可选） | `CT-2026-0401` |
| cf:meta | `delivered_at` | timestamp | — | 交付发出时间 | `2026-04-20T10:00:00Z` |
| cf:meta | `manifest_uri` | string | 512 | 交付清单（GCS 上的 manifest.json） | `gs://deliveries/d1b2c3/manifest.json` |
| cf:meta | `asset_count` | int | 4 | 本次交付 asset 数量（冗余，== `count(delivery_items)`） | `1200` |
| cf:meta | `total_size_bytes` | int64 | 8 | 本次交付总字节 | `858993459200` |
| cf:meta | `owner` | string | 64 | 交付负责人 URN | `urn:grace:user:alice` |
| cf:meta | `status` | string | 16 | 生命周期状态 | `pending` / `delivered` / `accepted` / `rejected` / `recalled` |
| cf:meta | `accepted_at` | timestamp | — | 客户签收时间（nullable） | `2026-04-22T12:00:00Z` |
| cf:meta | `rejected_at` | timestamp | — | 拒收时间（nullable） | `null` |
| cf:meta | `recalled_at` | timestamp | — | 召回时间（nullable，合规驱动） | `null` |
| cf:meta | `rejection_reason` | string | 256 | 拒收/召回原因 | `null` |
| cf:meta | `is_deleted` | bool | 1 | 软删 | `false` |
| cf:meta | `created_at` | timestamp | — | 行创建时间 | `2026-04-20T09:55:00Z` |
| cf:meta | `updated_at` | timestamp | — | 行更新时间 | `2026-04-22T12:00:00Z` |

> **设计要点：**
> - 1 行 = 1 个交付事件（不是交付明细），asset 明细在 `delivery_items`
> - 状态字段同 `cf:meta`，不单独拆 CF：状态只从 `pending → delivered → accepted/rejected → recalled` 最多 4 次写入，频率远低于 `mcap_files.cf:process`，没必要独立 CF
> - `asset_count` / `total_size_bytes` 冗余存，只读列表页免聚合；写入时由 service 层计算

---

## 表 4：`delivery_items`（asset ↔ delivery 关联表）· 新增

关联表（Junction Table），捕捉 asset 与 delivery 的 M:N 关系。

| 字段 | 类型 | 长度 | 作用 | 说明 |
|---|---|---|---|---|
| `delivery_id` | string (UUIDv4) | 36 | 属于哪个交付批次 | 复合主键 ① |
| `asset_id` | string (UUIDv4) | 36 | 哪个资产 | 复合主键 ② |

复合主键 = `(delivery_id, asset_id)`，无其他字段。

- **Phase 0 / PG：** 普通关联表，两列 FK + 复合 PK + 两个方向的索引
- **Phase 1 / Bigtable：** 无实体表，通过两个冗余索引实现双向 M:N 查询：
  - `idx_asset_deliveries` rowkey = `<asset_id>#<delivered_at>#<delivery_id>` —— "某资产交付给过哪些客户"
  - `idx_customer_deliveries` rowkey = `<customer_id>#<delivered_at>#<delivery_id>` —— "某客户收过哪些批次"

### 为什么单独建 `deliveries` + `delivery_items` 两张表（不在 assets 上加字段）

一个 asset 可以交付给多个客户，一个客户也可以批量收多个 asset（M:N）。

如果只在 assets 加字段：
- `last_delivered_at` / `count` / `last_delivered_to`（保留的 3 个汇总字段）→ 只能回答"是否交付过/最近一次"，回答不了"asset X 历史上交付给过哪些客户"
- 存数组 `delivered_to = [A, B, C]` → 丢失每次交付的时间、合同号、状态；也无法查"客户 A 上个月收了哪 1000 条"

关系模型（M:N）是唯一能完整回答下列高价值查询的形态：
1. 这个 asset 被交付给过哪些客户、哪几次 → 扫 `delivery_items WHERE asset_id=X JOIN deliveries`
2. 客户 A 在 Q1 收到了哪些 asset → 扫 `deliveries WHERE customer_id=A AND delivered_at∈Q1 JOIN delivery_items`
3. 召回某批次所有资产（合规/GDPR 场景）→ 查 `deliveries.status='recalled' JOIN delivery_items`
4. 避免重复交付同一 asset 给同一客户 → service 层 pre-check `(customer_id, asset_id)` 是否已存在
5. 详细讨论见 [ADR-011 交付追踪模型](../docs/adr/ADR-011-delivery-tracking.md)（待落）

---

## Row Key 设计

- `asset_id` / `mcap_file_id` 采用 UUIDv4（纯随机），rowkey 天然均匀打散到 tablet，无写入热点
- `v1#` 版本前缀：为未来 rowkey 格式演进（如多租户迁移）保留双写空间
- 不需要 salt：UUIDv4 本身就是随机前缀；没有 UUIDv7 的时间序问题要抵消
- 不需要 `asset_locator` 定位表：SDK 直接 `v1#` + `asset_id` 拼接 → 1 次 RPC 点查
- 不用组合 rowkey（`mcap_file_id#start_ts#asset_id`）：会牺牲 SDK `get_asset` 这个最高频路径 + 同 MCAP 多 segment 并发写入时前缀相同会热点，详见 [ADR-001](../docs/adr/ADR-001-wide-table-with-bigtable.md)

### 必要的二级索引

| 索引表 | row key | 用途 |
|---|---|---|
| `idx_segments_by_file` | `<mcap_file_id>#<start_timestamp_ns>#<asset_id>` | 按 MCAP 拿所有 segment，前缀扫一次 RPC |
| `idx_asset_deliveries` | `<asset_id>#<delivered_at_rev>#<delivery_id>` | 某资产交付给过哪些客户（Phase 1+ 替代 `delivery_items` 的 asset→delivery 方向） |
| `idx_customer_deliveries` | `<customer_id>#<delivered_at_rev>#<delivery_id>` | 某客户收过哪些批次（Phase 1+ 替代 `delivery_items` 的 customer→delivery 方向） |

> `delivered_at_rev = Long.MAX - delivered_at_ms`，保证最新交付排在前（前缀扫 limit 1 即"最近一次"）。
>
> 其他查询（按 vendor / task / status / algo / tag 过滤）一律走 OpenSearch；跨行聚合走 BigQuery External Table。

---

## 表关系

```
mcap_files ──(1)────(N)──► assets ◄────(N)──── delivery_items ────(N)────► deliveries
            mcap_file_id              asset_id                    delivery_id
                                                                                │
                                                                                └── customer_id
```

- `assets.mcap_file_id`（逻辑外键）→ `mcap_files.mcap_file_id`
- `delivery_items.asset_id`（逻辑外键）→ `assets.asset_id`
- `delivery_items.delivery_id`（逻辑外键）→ `deliveries.delivery_id`
- Phase 0 可在 PG 加 `FOREIGN KEY`，Phase 1 Bigtable 无外键概念，由 service 层保证一致性
- `assets.cf:meta.last_delivered_*` 三个汇总字段与 `delivery_items` 强一致：写 `delivery_items` 时同事务/同批次更新 `assets`（Phase 0 PG 事务；Phase 1 Bigtable 通过 service 层 + Outbox 最终一致）

---

## 字段总计

| 表 | 固定字段 | 动态列族 | 列族数 |
|---|---|---|---|
| `assets` | 23 | `cf:algo` + `cf:tag` + `cf:files` | 4 |
| `mcap_files` | 31 | — | 2 |
| `deliveries` | 16 | — | 1 |
| `delivery_items` | 2（复合主键） | — | 0（关联表） |
| `asset_algo_events` | 8 | — | 0（审计表） |

> 总计：80 个固定字段 + 3 个动态列族，7 个列族（+ 3 张索引表）+ 1 张审计表

---

## 备注

- **`owner` 字段说明：** Phase 0 只是占位（可写可不写，SDK/服务层不强校验），Phase 1+ 接权限系统后，它会参与可见性过滤和配额统计。见 [ADR-010（待落）字段治理](../docs/adr/) 中"owner 落地策略"章节。
- **交付追踪详细方案：** 见 [ADR-011 交付追踪模型](../docs/adr/ADR-011-delivery-tracking.md)（待落）。

---

## 表 5：`asset_algo_events`（算法事件审计表）· 新增

算法生命周期状态变更的审计日志。每次 start / finish / reset 操作插入一行，不覆盖。

| 字段 | 类型 | 长度 | 作用 | 示例 |
|---|---|---|---|---|
| `event_id` | UUID | 36 | 事件主键 | `e1b2c3d4-...` |
| `asset_id` | UUID | 36 | 关联资产（FK → assets） | `a1b2c3d4-...` |
| `algo_key` | string | 128 | 算法键（`<name>@<version>`） | `hand_tracking@1.2.0` |
| `prev_status` | string | 16 | 变更前状态（首次为 NULL） | `pending` |
| `new_status` | string | 16 | 变更后状态 | `running` |
| `run_id` | string | 64 | 外部执行批次 id（nullable） | `dagster-run-abc` |
| `reason` | string | 256 | 失败原因（nullable） | `OOM at frame 1337` |
| `created_at` | timestamp | — | 事件时间 | `2026-04-24T10:00:00Z` |

**索引：**
- `(asset_id, created_at DESC)` — 按资产查事件历史
- `(algo_key, new_status, created_at DESC)` — 按算法查状态分布
- `(run_id) WHERE run_id IS NOT NULL` — 按执行批次反查

> **设计要点：**
> - 追加写入，不更新不删除，天然适合按月分区
> - Phase 1 可保留在 PG 中（事件日志的查询模式偏分析，不适合 Bigtable），或迁移到 BigQuery
