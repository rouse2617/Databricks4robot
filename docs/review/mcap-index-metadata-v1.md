# MCAP 索引元数据入库（V1）- 真实样本证据

## 目的

明确第一阶段（轻索引）要从 MCAP 提取哪些元数据、放在哪里，以及每个字段的真实解析证据。

---

## MCAP 索引放哪里（总览）

| 索引类型 | 存放位置 | 说明 |
|----------|----------|------|
| **原始 MCAP 字节** | **对象存储（GCS）** 等，`mcap_uri` 指向 | 唯一真源；不复制、不转码 |
| **文件级标量 + 解析统计** | **PostgreSQL `mcap_files`** 列 + `metadata` / `process_state` 约定键 | 在线 API、权限、与 `assets` 关联 |
| **通道级摘要（topic/schema/条数等）** | **PostgreSQL `mcap_files.metadata.channels`** JSON 数组（V1 已定） | 不建 `mcap_channels` 表；**不同步湖仓**（当前无全库通道挖掘需求） |
| **资产可操作时间窗** | **PostgreSQL `assets`** `start/end_timestamp_ns` | 与 `log_time_ns` 同口径时，支撑「资产 + 时间 + 相机」裁边 |
| **Chunk / 帧 / 消息级指针**（后续） | **Apache Iceberg**（BigQuery） | 大体量、时间窗 → `offset/length`、帧表等 |
| **工作台检索 Facet**（可选） | **Elasticsearch** | outbox 同步副本，非唯一真源 |

---

## 真实样本与解析方式

- 样本文件：`/Users/rick/Downloads/e4840f0bed39689ef28bcfc4afa53b8d.mcap`
- 文件大小：`1334578288` bytes（约 `1.27 GB`）
- 解析时间（UTC）：`2026-05-14T02:12:15Z`
- 解析输出目录：`/tmp/mcap_extract_e4840f0b_20260514`

本次使用 Go + `github.com/foxglove/mcap/go/mcap` 对真实 MCAP 做解析，产出：

- `summary.json`
- `channels.json`
- `schemas.json`
- `chunk_indexes.json`
- `message_samples.json`

---

## 证据摘录（真实解析结果）

### 1) 文件级 summary 证据

```json
{
  "source_file": "/Users/rick/Downloads/e4840f0bed39689ef28bcfc4afa53b8d.mcap",
  "generated_at_utc": "2026-05-14T02:12:15Z",
  "file_size_bytes": 1334578288,
  "library": "mcap go v1.8.0; libmcap 2.1.2",
  "message_count": 224596,
  "schema_count": 4,
  "channel_count": 8,
  "chunk_count": 5428,
  "start_ns": 1775434966056908229,
  "end_ns": 1775435393121465906,
  "duration_sec": 427.064557677,
  "chunk_compression": {
    "": 5428
  },
  "compression_ratio": 1
}
```

### 2) 通道级 channels 证据（节选）

```json
[
  {
    "channel_id": 1,
    "topic": "/imu/front",
    "schema_id": 1,
    "schema_name": "sensor_msgs/Imu",
    "schema_encoding": "jsonschema",
    "message_encoding": "json",
    "message_count": 139810,
    "freq_hz_approx": 327.3743922007734
  },
  {
    "channel_id": 3,
    "topic": "/camera/front/image_raw/compressed",
    "schema_id": 2,
    "schema_name": "foxglove.CompressedVideo",
    "schema_encoding": "protobuf",
    "message_encoding": "protobuf",
    "message_count": 12709,
    "freq_hz_approx": 29.758966815532716
  },
  {
    "channel_id": 2,
    "topic": "/audio/usb_mic/raw",
    "schema_id": 4,
    "schema_name": "foxglove.RawAudio",
    "schema_encoding": "protobuf",
    "message_encoding": "protobuf",
    "message_count": 8533,
    "freq_hz_approx": 19.98058571382018
  }
]
```

### 3) Schema 证据（节选）

```json
[
  {
    "schema_id": 1,
    "name": "sensor_msgs/Imu",
    "encoding": "jsonschema",
    "data_bytes": 895
  },
  {
    "schema_id": 2,
    "name": "foxglove.CompressedVideo",
    "encoding": "protobuf",
    "data_bytes": 493
  }
]
```

---

## 第一阶段入库字段（V1）

> 目标：上传后秒级可查询，避免实时扫 MCAP 正文。
> **仓库真源**：`docs/review/sql.md` §4.1、`schemas/pg-phase0.sql` 中 `mcap_files` 定义；API 层 `McapFile.gcs_path` 入库列名为 **`mcap_uri`**（见 `backend/internal/postgres/repos.go`）。

### 与现有 `mcap_files` 列对齐（避免重复设计）

以下 PG 列**已存在**，索引器只需 **回填/更新**，无需新增 DDL：

| 解析来源（MCAP summary） | 已有 PG 列 | 说明 |
|--------------------------|------------|------|
| 对象路径 | `mcap_uri` | 与 `size_bytes` 等同上传 finalize 已知信息时可一并写 |
| 文件大小 | `size_bytes` | 同上 |
| 日志时间起止 | `start_timestamp_ns`, `end_timestamp_ns` | 对应 Statistics 的 message 时间边界 |
| 时长 | `file_duration_ms` | 可由 `end-start` 派生，与解析 `duration_sec` 一致 |
| 通道数、chunk 数 | `channel_count`, `chunk_count` | 已有 |
| 摄取状态 | `ingest_state` | 已有；索引完成后由业务定义是否写入 `summarized`/`ready` 等枚举 |
| 扩展 | `metadata`, `process_state` | JSONB；**适合**承载解析多出的统计项与索引流水线状态 |
| 内容指纹 | `raw_hash_md5`, `raw_hash_sha256` | 已有；幂等与「是否需重算索引」可依赖 hash |
| 采集/租户等 | `vendor_id` … `tenant_id`/`project_id` 等 | 已有；与 MCAP 解析无强绑定，不必由 indexer 改写 |

**通道明细（V1 已选定）**：写入 **`mcap_files.metadata` → `channels` 键**（值为 **JSON 数组**）。不新增 `mcap_channels` 表、**本阶段不做 migration**；与 `sql.md` §3「低频扩展走 `metadata` JSONB」一致。表 `mcap_files` 上已有 **`GIN (metadata)`**（`idx_mcap_files_metadata_gin`），支持 `@>` 等包含查询；若日后全库按 `topic` 筛选成为瓶颈，再评估拆 **`mcap_channels` + btree(topic)** 或加表达式 GIN。

下列解析项**没有独立标量列**，落在 **`metadata` / `process_state` 约定键**（与 `channels` 并列，勿覆盖整段 `metadata`）：

| 解析项 | 建议键（示例） | 样本值 |
|--------|----------------|--------|
| 全文件 message 条数 | `metadata.mcap_message_count` | `224596` |
| schema 个数 | `metadata.mcap_schema_count` | `4` |
| 写入端 library | `metadata.mcap_library` | `mcap go v1.8.0; libmcap 2.1.2` |
| profile | `metadata.mcap_profile` | `""` |
| chunk 压缩分布 | `metadata.mcap_chunk_compression` | `{"":5428}` |
| 压缩比 | `metadata.mcap_compression_ratio` | `1` |
| 摘要解析 UTC 时间 | `process_state.mcap_summary_extracted_at` | ISO8601 字符串 |

### A. 文件级（落 `mcap_files`）

| 字段 | 来源证据 | 样本值 | 与现有列关系 |
|---|---|---|---|
| `mcap_file_id` | 业务生成 | 例如 `e4840f0b` | **已有** PK |
| `mcap_uri` | 上传输入 | `gs://.../e4840f0bed39689ef28bcfc4afa53b8d.mcap` | **已有** |
| `size_bytes` | `summary.file_size_bytes` | `1334578288` | **已有** |
| `start_timestamp_ns` | `summary.start_ns` | `1775434966056908229` | **已有** |
| `end_timestamp_ns` | `summary.end_ns` | `1775435393121465906` | **已有** |
| `file_duration_ms` | `summary.duration_sec * 1000` | `427064`（约） | **已有** |
| `channel_count` | `summary.channel_count` | `8` | **已有** |
| `chunk_count` | `summary.chunk_count` | `5428` | **已有** |
| `metadata.mcap_message_count` | `summary.message_count` | `224596` | **建议键**，见上表 |
| `metadata.mcap_schema_count` | `summary.schema_count` | `4` | **建议键** |
| `metadata.mcap_library` | `summary.library` | `mcap go v1.8.0; libmcap 2.1.2` | **建议键** |
| `metadata.mcap_profile` | `summary.profile` | `""` | **建议键** |
| `metadata.mcap_chunk_compression` | `summary.chunk_compression` | `{"":5428}` | **建议键** |
| `metadata.mcap_compression_ratio` | `summary.compression_ratio` | `1` | **建议键** |
| `process_state.mcap_summary_extracted_at` | 解析时间 | `2026-05-14T02:12:15Z` | **建议键** |
| `ingest_state` | 业务状态机 | `summarized` 等 | **已有** 列，值由产品定义 |

### B. 通道级（**固定**：`metadata.channels` JSON 数组）

**存储位置**：`mcap_files.metadata` 顶层键 **`channels`**（`jsonb` 数组）。索引器写入时应 **`metadata = metadata || jsonb_build_object('channels', …)`**（或应用层合并），避免覆盖 `metadata` 中其它业务键。

**数组元素形状**（每个元素 = 原 MCAP 一列 channel 摘要；字段名建议与解析器一致，便于前后端共用）：

| JSON 键 | 类型 | 来源 | 样本值 | 必填 |
|---------|------|------|--------|------|
| `channel_id` | number | MCAP | `1` | 是 |
| `topic` | string | MCAP | `/imu/front` | 是 |
| `schema_id` | number | MCAP | `1` | 是 |
| `schema_name` | string | MCAP | `sensor_msgs/Imu` | 是 |
| `schema_encoding` | string | MCAP | `jsonschema` | 是 |
| `message_encoding` | string | MCAP | `json` | 是 |
| `message_count` | number | Statistics | `139810` | 是 |
| `freq_hz_est` | number | 派生 | `327.374392...` | 否 |
| `channel_metadata` | object | MCAP channel metadata | `{}` | 否 |

**一致性校验（建议索引器自检）**：

- `jsonb_array_length(metadata->'channels')` 应等于标量列 **`channel_count`**（或与解析结果一致后再回写 `channel_count`）。
- 可选：`process_state.mcap_channel_index_version`（如 `v1`），便于以后改数组形状时迁移。

#### 查询示例（PostgreSQL）

```sql
-- 某文件全部通道展开
SELECT mcap_file_id, ch
FROM mcap_files,
     jsonb_array_elements(metadata->'channels') AS ch
WHERE mcap_file_id = :id AND is_deleted = FALSE;

-- 包含某 topic 的文件（channels 为数组；元素为对象且至少含 topic）
SELECT mcap_file_id
FROM mcap_files
WHERE is_deleted = FALSE
  AND metadata ? 'channels'
  AND metadata->'channels' @> '[{"topic": "/imu/front"}]'::jsonb;
```

更复杂的「任意 topic 等值」可用 `EXISTS` + `jsonb_array_elements`（见团队 SQL 规范与 `EXPLAIN` 调优）。

### C. 资产时间窗（已有，供「资产 + 时间 + 相机」对齐）

`assets.start_timestamp_ns` / `end_timestamp_ns` / `duration_ms`（见 `sql.md` §4.2）已与 MCAP **`log_time`** 口径对齐需求一致；索引服务写 `mcap_files` 通道明细时**不必改 `assets` 表结构**。

---

## 增量触发与幂等（GCS finalize 方案）

V1 增量推荐直接使用 **Cloud Storage notifications -> Pub/Sub** 的 `OBJECT_FINALIZE` 事件触发 indexer；`mcap_upload_finalized` 可保留为业务侧显式确认事件（或兜底补偿），二者共存时由幂等键去重。

### 事件输入（最小字段）

- `bucketId`
- `objectId`
- `objectGeneration`
- `eventType`（应为 `OBJECT_FINALIZE`）

由此拼出 `mcap_uri = gs://{bucketId}/{objectId}`，并用 `(bucketId, objectId, objectGeneration)` 作为事件级幂等键。

### 反查与分支处理

1. 收到 finalize 事件后，先用 `mcap_uri` 反查 `mcap_files`。
2. **若已存在**：不新建记录，直接走 summary 解析与回填（`summary_index_state` 状态机）。
3. **若不存在**：按最小字段建档（`mcap_uri`、`size_bytes` 等可得字段），`summary_index_state='pending'`，随后进入同一解析流程。
4. 成功后回写 `summary_index_state='done'`、`summary_indexed_at`、`summary_index_version`，并更新 `metadata/process_state` 约定键。
5. 失败则 `summary_index_state='failed'` + `summary_index_error`，由定时补偿任务重试（扫 pending/failed）。

建议把以下事件足迹写入 `process_state`（便于排障和重放）：

- `process_state.gcs_event_generation`
- `process_state.gcs_event_bucket`
- `process_state.gcs_event_object`
- `process_state.gcs_event_received_at`

### 参考 SQL（按 URI 反查）

```sql
SELECT mcap_file_id, summary_index_state, ingest_state
FROM mcap_files
WHERE mcap_uri = $1
  AND is_deleted = FALSE
ORDER BY created_at DESC
LIMIT 1;
```

> 注意：GCS/Pub/Sub 为 at-least-once 投递，同一对象可能重复触发；若对象覆盖上传（generation 变化），应视为新版本事件并按策略决定重算。

---

### C. Chunk 偏移（GCS 旁路 JSON，不入 PG 主表）

**设计原则**：单个文件可达数千行 chunk 索引（样本 5428 条，477 KB）。若直接存入 PG 列，查询引擎全表扫描时会把这些大 BLOB 全部加载到内存，导致网络带宽瓶颈和内存溢出。因此采用**旁路存储（Sidecar Storage）**——PG 只存 URI 指针，实际数据放在 GCS。

**存储位置**：GCS 与原 MCAP 同级目录，文件名 `{去协议和扩展名}.chunks.json`。

例：`gs://bucket/path/e4840f0b.mcap` → `gs://bucket/path/e4840f0b.chunks.json`

**PG 只存指针**（落在 `metadata` / `process_state` 约定键）：

| 解析项 | 建议键 | 类型 | 样本值 |
|--------|--------|------|--------|
| 旁路文件 URI | `metadata.chunk_index_uri` | string | `gs://bucket/path/e4840f0b.chunks.json` |
| 旁路文件大小 | `metadata.chunk_index_size_bytes` | number | `476928` |
| chunk 条目数 | `metadata.chunk_index_count` | number | `5428` |
| 生成状态 | `process_state.chunk_index_state` | string | `pending` / `running` / `done` / `failed` |
| 生成错误 | `process_state.chunk_index_error` | string | 最近一次失败摘要 |

**旁路文件 JSON 形状**（精简版，仅含定位必需字段，每条 89 bytes）：

```json
[
  {"i":0, "t0":1775434966056908229, "t1":1775434968659925955, "off":575, "len":258604},
  {"i":1, "t0":1775434968659925956, "t1":1775434969259925955, "off":259179, "len":262144},
  ...
]
```

| JSON 键 | 类型 | 说明 |
|---------|------|------|
| `i` | number | Chunk 序号，与 `info.ChunkIndexes` 下标一致 |
| `t0` | uint64 | Chunk 内第一条消息的 `log_time`（纳秒） |
| `t1` | uint64 | Chunk 内最后一条消息的 `log_time`（纳秒） |
| `off` | uint64 | Chunk 在 MCAP 文件中的起始字节偏移 |
| `len` | uint64 | Chunk 压缩后长度（可直接用于 `Range: bytes=off-off+len-1`） |

**消费方式**：

1. 客户端 HTTP GET 旁路 JSON（477 KB，一次性，可浏览器缓存）
2. 本地二分查找 `t0 ≤ target_ns ≤ t1` → 定位目标 chunk
3. 对原 MCAP 发起精准 GCS Range Read：`bytes={off}-{off+len-1}`
4. 解压 chunk → 遍历消息 → 取目标通道的数据

每次 Seek 仅需 **1 次** GCS 往返（chunk JSON 缓存后为 0 次额外请求）。

**规模参考**（基于 1.27 GB / 427 秒 / 5428 chunk 的真实样本）：

| 文件数 | chunks.json 总占用 | GCS 标准存储月费（$0.020/GB） |
|--------|--------------------|-----------------------------|
| 1 千 | 466 MB | $0.01 |
| 1 万 | 4.5 GB | $0.10 |
| 10 万 | 45.5 GB | $0.98 |
| 100 万 | 455 GB | $9.77 |

---

## V1 暂不入库（后续阶段）

以下字段可解析，但 V1 不纳入上述存储方案：

- **Chunk 内消息明细**：每条消息的 `channel_id`、`offset`、`log_time` 在 chunk 内的偏移
  - 原因：单文件可达 22 万条（样本），体量过大（~9 MB/文件）。若需要跨文件按 topic+time 检索消息，走 V2/Iceberg。
- **Message 明细（逐条消息级）**
  - 原因：体量过大，V1 不需要

**后续方向**：

- **V2**：Iceberg `bronze_mcap_chunk_index` — 当需要对大量文件做跨文件时间窗裁剪时，将 chunk 索引以 Parquet 格式注册为 Iceberg 表，利用 BigQuery 的 CBO 做文件级裁剪。
- **V3**：Iceberg `bronze_video_frame_index` — 解析 `VideoTimestampMap` 通道，建立「时间戳 → 帧在 chunk 内偏移」的精确定位，实现单帧毫秒级拉取。

---

## 场景收益（V1 完成后）

- 按 `topic/schema/freq/时长` 快速发现录包，不扫对象存储
- Preview 前置判断（是否有目标通道、频率是否满足）
- 转 Parquet 前先筛候选，减少无效 ETL
- 采集治理（缺通道、频率漂移、无压缩录制）
- **资产 + 时间戳 → 精准 Chunk**：播放器 / 在线 API 通过旁路 JSON 定位目标 chunk，每次 Seek 仅 1 次 GCS 往返
- **Action 驱动的帧检索**：Action 的时间窗 → chunks 索引 → 精准拉取对应通道的画面/传感器数据

---

## 总结（写入版）

### 存储分层（推荐长期形态）

| 层级 | 存什么 | 技术选型 |
|------|--------|----------|
| 真源 | 完整 `.mcap` 字节 | 对象存储（GCS） |
| 在线权威 | 文件级标量列 + **`metadata.channels` 通道摘要**、资产、Action、权限 | PostgreSQL |
| 旁路索引 | Chunk 偏移（时间 → 字节范围映射） | GCS 旁路 JSON，PG 只存 URI 指针 |
| 大体量索引 | 消息级 / 视频帧级指针 | Apache Iceberg via BigQuery |
| 检索 Facet | topic、schema、标签等（副本） | Elasticsearch（outbox 同步） |

### 实施顺序

1. **V1**：`mcap_files` 标量列回填 + **`metadata.channels` 数组** + **GCS 旁路 `chunks.json` 生成** + `metadata.mcap_*` / `process_state` 约定键（无新表，零 migration）。
2. **V2**：Iceberg `bronze_mcap_chunk_index` — 跨文件时间窗裁剪（BigQuery 分析场景）。
3. **V3**：Iceberg `bronze_video_frame_index`（`VideoTimestampMap` 与 `CompressedVideo` 对齐）+ `bronze_mcap_message_index`（topic 白名单）。
4. **标签**：帧/区间与物理指针解耦——`bronze_frame_labels` 长表或 JSON + Gold 投影。

### 产品终极目标（查询契约）

统一支持两类入口（时间轴均为 MCAP **`log_time_ns`**，与 `assets.start/end_timestamp_ns` 同口径）：

- **点查**：`资产 id` + **`log_time_ns`（或等价时间戳）** + **`camera_topic`（哪一路视频）** → 唯一帧（或最近帧，需约定 `exact | nearest | floor`）→ 回放指针 + 标签。
- **段查**：`资产 id` + **`[t0, t1]` 时间段** + **`camera_topic`** → 帧序列或子片段列表 → 浏览、批量打标签、区间描述（可与现有 `actions` 或区间标注模型对齐）。

「相机」= 多路视频中的**一路**，在 MCAP 里通常用 **完整 topic** 标识（如 `/camera/front/image_raw/compressed`）；与 `VideoTimestampMap` 成对使用。

---

## 结论

基于真实样本解析，第一阶段 MCAP 元数据入库建议为：

1. **文件级**：优先**回填** `mcap_files` 已有标量列（`mcap_uri`、`size_bytes`、`start/end_timestamp_ns`、`file_duration_ms`、`channel_count`、`chunk_count`、`ingest_state`）；解析多出的统计项写入 **`metadata` / `process_state` 约定键**（见 §「与现有 mcap_files 列对齐」）
2. **通道级**：写入 **`mcap_files.metadata.channels`** JSON 数组（**不新建** `mcap_channels` 表）；各 topic **平等一条**
3. **Chunk 偏移**：写入 **GCS 旁路 JSON**（`{mcap_uri}.chunks.json`，~477 KB/文件），PG 仅存 URI 指针（`metadata.chunk_index_uri`）；播放器和在线 API 通过 HTTP GET 直接消费，实现「时间戳 → 精准 Range Read」的毫秒级定位
4. 消息级 / 帧级索引进入后续阶段，落 Iceberg；标签表随后续目标在 Iceberg 展开
