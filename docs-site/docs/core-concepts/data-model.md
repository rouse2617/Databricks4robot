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
| `asset_id` | UUID string | 全局唯一标识 |
| `mcap_file_id` | string | 关联 MCAP 文件 |
| `t_start` | int64 (ns) | 数据起始时间 |
| `t_end` | int64 (ns) | 数据结束时间 |
| `status` | enum | 生命周期状态 |
| `tags` | map[str,str] | 自定义标签 |
| `created_at` | datetime | 创建时间 |
| `updated_at` | datetime | 更新时间 |

### MCAP File

底层存储文件，与 Asset 1:1 或 N:1 关联。

| 字段 | 类型 | 说明 |
|------|------|------|
| `file_id` | string | 文件唯一 ID |
| `storage_path` | string | GCS 存储路径 |
| `size_bytes` | int64 | 文件大小 |
| `md5_hash` | string | 文件校验和 |

### Delivery（交付）

面向客户的数据交付记录。

| 字段 | 类型 | 说明 |
|------|------|------|
| `delivery_id` | UUID string | 交付唯一 ID |
| `customer_id` | string | 目标客户 |
| `status` | enum | draft/committed/processing/done/failed |
| `items` | DeliveryItem[] | 交付内容列表 |

### Algo Run（算法运行）

一次算法执行记录。

| 字段 | 类型 | 说明 |
|------|------|------|
| `run_id` | UUID string | 运行唯一 ID |
| `algo_key` | string | 算法标识 |
| `status` | enum | pending/running/done/failed/cancelled |
| `asset_ids` | string[] | 处理的资产列表 |

## 关系说明

- **Asset ↔ MCAP File**: 一个 Asset 关联一个 MCAP File，多个 Asset 可以共享同一个 MCAP File
- **Asset ↔ Delivery**: 一个 Delivery 可以包含多个 Asset，通过 DeliveryItem 关联
- **Asset ↔ Algo Run**: 一个 Algo Run 可以处理多个 Asset
- **Asset ↔ Tag**: N:M，标签支持多源合并和历史追踪
- **Customer ↔ Delivery**: 一个 Customer 可以有多个 Delivery

## 审计日志

所有实体变更都会记录审计日志，支持按事件类型、时间范围搜索：

```python
from cyber_databrew_sdk import CyberDatabrewClient

client = CyberDatabrewClient(base_url="...", token="g-xxx")

# 审计搜索
results = client.audit.search(
    asset_id="asset-xxx",
    event_type="delivery.commit",
    from_date="2026-01-01",
    to_date="2026-05-25",
)

# 血缘审计
results = client.audit.lineage_search(asset_id="asset-xxx")
```
