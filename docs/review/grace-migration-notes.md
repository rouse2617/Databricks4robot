# Grace（cyber-grace）↔ 数据平台迁移对照说明

> **目的**：把生产侧 **`cyber-grace`**（Grace Collector：`service/` + `grace_videos` 等）与本仓库 **data-platform**（MCAP 导向资产、投影表、outbox）之间的概念与字段对上号，便于迁移、双写、对账与集成，**不替代** `schema-reference.md` 或 `api/openapi.yaml` 的权威定义。
>
> Grace 仓库路径（本地开发）：与本仓库并列的 `cyber-grace/`（见该仓 `README.md`、`service/models/videos/model.go`）。

---

## 1. 系统分工（读本文前先建立预期）

| 维度 | cyber-grace（Grace） | data-platform（本仓库） |
|------|----------------------|-------------------------|
| **资产粒度** | **`grace_videos`**：以**视频/采集条**为核心实体 | **`assets` + `mcap_files`**：以 **MCAP 文件 + segment/clip 等资产**为核心 |
| **Schema 来源** | GORM 模型 + 启动自动迁移 | `backend/migrations/*.sql` |
| **算法工序态** | **`process_info` JSONB**（hand_tracking、deface…） | **`asset_algo_latest` + `asset_events`**（含 `algo_started` / `algo_finished` 等） |
| **HTTP** | `/grace/videos` 等，**Basic/JWT** | `/api/v1/*`，**`X-Databrew-Token`**（Phase 0） |
| **列表过滤** | `filter=key:op:value` | `filter=field:op:value`（见 `api-guide`） |

**原则**：Grace 仍是当前 many 团队的管理面事实源；数据平台侧若承接同一条业务数据，应用**明确的映射表或同步任务**维护 `grace_video.id`（或 `raw_hash_md5`）与 `asset_id` / `mcap_file_id` 的关系，避免两套库各自“隐式对应”。

---

## 2. 实体级映射（建议）

| Grace | 数据平台 | 说明 |
|-------|----------|------|
| `grace_videos.id`（UUID） | `assets.asset_id`（**8 位字母数字**）或独立 **`grace_external_ref`** / 映射表 | Grace 使用 UUID；本平台资产主键为短 id，集成层需显式映射（勿假定二者可直接相等）；建议在集成层维护双向 ID 或 dedicated 映射表（本文不固定 DDL） |
| `grace_videos.raw_hash_md5` | `mcap_files.raw_hash_md5` | **ingest 幂等 / 去重主键**：本平台在 `is_deleted = FALSE` 上对 `raw_hash_md5` 有 **UNIQUE**（见 `001_init.sql`） |
| `grace_videos` 一行 | 常对应 **`mcap_files` 1 行 + `assets` 1…N 行** | MCAP 可对应多个 segment；若 1:1 迁移，先定产物是「仅文件级」还是「文件 + 默认 segment」 |
| `grace_videos.is_deleted` / `archived_at` | `assets.is_deleted`、`lifecycle_state` 等 | 语义需逐个对齐（软删 vs 归档） |

---

## 3. 字段级映射（高流量字段）

### 3.1 采集与场站（Grace `collection_meta`）

Grace `collection_meta`（JSONB）典型键包括：`vendor_id`、`scene_id`、`environment_id`、`collector_id`、`location_id`、`task_id`、`data_source` 等（见 `cyber-grace/service/models/videos/model.go`）。

数据平台侧常见落点：

- **提升为标量或 facet**：进入 `mcap_files` / `assets` 的列，或 **`asset_tags`** / 搜索索引里的 **`tags_flat.*`**、**`mcap.vendor_id`**、**`mcap.scene_id`** 等（以当前 ES mapping 与 `filter` 支持为准）。
- **过渡期**：未提升的低频字段统一落在 `metadata` JSONB；高频检索字段按本仓 `schema-reference.md` 提升为标量列或投影表行。

建议在同一份**字段字典**里维护「Grace JSON 路径 → 平台列 / tag_key」，避免口头约定漂移。

### 3.2 存储（Grace `storage_meta`）

对齐 **`mcap_files`** 的对象 URI、大小、`gcs_path` / `object_uri` 等；具体键名以 Grace `types` 与平台 `mcap_files` / `metadata` 约定为准。

对视频衍生路径建议直接落资产列，避免仅保留在深层 JSON：

- `storage_meta.gcs.algo_inputs.*.uri` → `assets.algo_inputs_uris`（JSONB，`name -> uri`）
- `storage_meta.gcs.annot_inputs.*.uri` → `assets.annot_inputs_uris`（JSONB，`name -> uri`）

### 3.3 工序与算法（Grace `process_info`）

Grace 将各算法状态放在 **JSONB**（如 `hand_tracking`、`deface` 等字符串状态）。

数据平台：

- **主路径**：**`asset_algo_latest`** 每 (`asset_id`, `algo_name`) 一行 + **`asset_events`** 记录状态变迁。
- **平台侧契约**：写入只走算法 API（`start` / `finish` / `reset`），由后端落到 `asset_algo_latest + asset_events`。

迁移时建议：**按 algo 名与版本**把 `process_info` 译为 `asset_algo_latest` 行 + 可选一条 `algo_finished` 类事件（是否需要回放历史事件由合规/审计决定）。

### 3.4 内容哈希

| 字段 | Grace | 数据平台 |
|------|-------|----------|
| MD5 | `raw_hash_md5`，**必填、索引** | **`mcap_files.raw_hash_md5`**，非空时唯一（活跃行） |
| SHA-256 | 若 Grace 后续增加，建议在集成层同步 | **`mcap_files.raw_hash_sha256`**（可选；UNIQUE 策略以 migration 为准） |

**与 Grace 对齐的 ingest 幂等**：优先以 **MD5** 与 Grace 一致；SHA-256 可作为更强指纹或对象存储校验，**不作为与 Grace 冲突的第二套“主”幂等键**，除非业务明确切换。

---

## 4. API 与集成注意点

- **鉴权不同**：集成服务需同时处理 Grace 的 Basic/JWT 与本平台的 **`X-Databrew-Token`**（或未来 OIDC）。
- **Filter 语法**形似（`field:op:value`），**具体字段名不全相同**；迁移脚本不要直接拼接 Grace 的 filter 字符串到本平台 without 映射层。
- **交付 / QC**：Grace 有 `deliveries`、质检等独立模块；本平台也有 **`deliveries`**，**ID 与语义不要默认相同**，需集成规范。

---

## 5. 推荐落地顺序（工程上）

1. **定 ID 策略**：`grace_video_id` ↔ `mcap_file_id` / `asset_id` 是否存在 1:1、1:N。
2. **定幂等键**：默认 **`raw_hash_md5`** 与 Grace 一致。
3. **字段字典**：`collection_meta` / `process_info` → 平台列与 tags 清单。
4. **先只读同步或对账**：双写前用 batch job 校验条数、hash、关键标签一致率。
5. **再写路径切换**：工序写入统一走 **`asset_algo_latest` + `asset_events`**（与 `algo-lifecycle-and-data-model.md` 一致）。

---

## 6. 相关文档（本仓库）

- [`data-platform-design.md`](https://www.feishu.cn/wiki/QiNWwqLlWinHQpkf9Pbcy0pfniB) — 整体阶段与存储主线
- [`schema-reference.md`](https://www.feishu.cn/wiki/BUvcwpQeAiPWNtkLpKecDwkcnxd) — PG 表与索引
- [`algo-lifecycle-and-data-model.md`](https://www.feishu.cn/wiki/EoYowiw4ji5BO0kxODgce1hgnPh) — 算法投影与事件
- [`api-guide.md`](https://www.feishu.cn/wiki/OEG4wYA48i3Kvpk0N1XccwW8nqe) — HTTP 与 filter 示例
