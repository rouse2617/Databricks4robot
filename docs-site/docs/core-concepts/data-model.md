# Data Model

本章介绍 Cyber Databrew 平台的核心数据模型和实体关系。

## 实体关系图

```
┌─────────────┐       ┌──────────────┐
│   Customer  │       │   MCAP File  │
└──────┬──────┘       └──────┬───────┘
       │                     │
       │                     │
       │              ┌──────▼───────┐
       │              │    Asset     │
       │              │  (核心实体)   │
       │              └──────┬───────┘
       │                     │
       ▼                     │
┌─────────────┐              │
│  Delivery   │◄─────────────┘
│  (交付)     │
└─────────────┘

┌─────────────┐     ┌────────────────┐
│  Algo Run   │     │  Action/Anno   │
│  (算法运行)  │     │  (标注/操作)    │
└─────────────┘     └────────────────┘
```

## 核心实体

### Asset（资产）

最核心的数据实体，代表一条数据片段。

| 字段 | 类型 | 说明 |
|------|------|------|
| `asset_id` | string (8位) | 全局唯一标识 |
| `mcap_file_id` | string (8位) | 关联 MCAP 文件 |
| `start_timestamp_ns` | int64 | 数据起始时间（纳秒） |
| `end_timestamp_ns` | int64 | 数据结束时间（纳秒） |
| `lifecycle_state` | enum | 生命周期状态（created/processing/ready/delivered/archived/superseded/failed/rejected） |
| `asset_type` | enum | 资产类型（segment/clip/frame/task） |
| `duration_ms` | int64 | 时长（毫秒） |
| `reviewer` | string | 审核人 |
| `owner` | string | 所有者 |
| `tags` | map | 自定义标签（多源合并） |
| `version` | int | 乐观锁版本号 |
| `created_at` | datetime | 创建时间 |
| `updated_at` | datetime | 更新时间 |

### MCAP File

底层存储文件，与 Asset N:1 关联。

| 字段 | 类型 | 说明 |
|------|------|------|
| `mcap_file_id` | string (8位) | 文件唯一 ID |
| `gcs_path` | string | GCS 存储路径 |
| `size_bytes` | int64 | 文件大小 |
| `raw_hash_md5` | string | 文件校验和 |
| `ingest_state` | enum | 摄入状态 |
| `file_duration_ms` | int64 | 文件时长（毫秒） |

### Delivery（交付）

面向客户的数据交付记录。

| 字段 | 类型 | 说明 |
|------|------|------|
| `delivery_id` | UUID | 交付唯一 ID |
| `customer_id` | string | 目标客户 |
| `status` | enum | pending/delivered/failed/accepted/rejected/recalled/cancelled/archived |
| `asset_count` | int | 包含资产数量 |
| `owner` | string | 交付负责人 |
| `note` | string | 备注说明 |

### Algo Run（算法运行）

一次算法执行记录。

| 字段 | 类型 | 说明 |
|------|------|------|
| `run_id` | string (16位) | 运行唯一 ID |
| `algo_name` | string | 算法名称 |
| `algo_version` | string | 算法版本 |
| `status` | enum | pending/running/ok/failed/cancelled |
| `triggered_by` | string | 触发来源 |

## 关系说明

- **Asset ↔ MCAP File**: 一个 Asset 关联一个 MCAP File，多个 Asset 可共享同一 MCAP File
- **Asset ↔ Delivery**: 一个 Delivery 可包含多个 Asset，通过 DeliveryItem 关联
- **Asset ↔ Algo Run**: 一个 Algo Run 可处理多个 Asset
- **Asset ↔ Tag**: N:M，标签支持多源共存和历史追踪
- **Customer ↔ Delivery**: 一个 Customer 可以有多个 Delivery

## 审计日志

所有实体变更都会记录审计日志，支持按事件类型、时间范围搜索：

```bash
curl "$BASE/api/v1/assets/aset0001/events?page_size=20" \
  -H "X-Databrew-Token: $TOKEN"
```

SDK 调用可使用 `client.events.list_for_asset("aset0001")`。跨资产审计搜索属于内部/管理接口，使用前请以 [交互式 API Reference](../api/reference) 中当前部署的接口为准。
