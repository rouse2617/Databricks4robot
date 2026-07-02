# Assets

**资产（Asset）** 是 Cyber Databrew 平台的核心实体。每个 Asset 代表一条数据记录（通常是视频或多模态数据片段），包含元数据、状态、标签和关联的 MCAP 文件。

## 资产核心字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `asset_id` | string (8位) | 全局唯一标识符 |
| `asset_type` | enum | 资产类型（segment, clip, frame, task 等） |
| `mcap_file_id` | string (8位) | 关联的 MCAP 文件 ID |
| `start_timestamp_ns` | int64 | 起始时间戳（纳秒） |
| `end_timestamp_ns` | int64 | 结束时间戳（纳秒） |
| `lifecycle_state` | enum | 生命周期状态（created, processing, ready, delivered, archived, superseded, failed, rejected） |
| `owner` | string | 所有者 |
| `reviewer` | string | 审核人 |
| `duration_ms` | int64 | 时长（毫秒） |
| `version` | int | 乐观锁版本号 |
| `tags` | map | 键值标签（多源合并） |
| `algo_results` | object | 算法处理结果投影 |
| `created_at` | datetime | 创建时间 |
| `updated_at` | datetime | 更新时间 |

## 生命周期

```
created → processing → ready → delivered → archived
      ↘ failed        ↘ rejected
      ↘ superseded
```

详情见 [算法状态机](../guides/pipeline-operations.md)。

## 标签系统

资产标签支持多源共存（human / algo_sdk / rule_engine / system），同一 `(asset_id, tag_key)` 可有多条不同来源的记录。扁平投影 `tags` 字段按 `applied_at` 取最新值，完整多源数据通过 `tags_detailed` 字段和 `GET /api/v1/assets/{id}/tags` 接口访问。

## 版本与血缘

- **版本管理**：每次更新产生新 revision，通过乐观锁 `version` 字段做并发控制
- **血缘关系**：`GET /api/v1/assets/{id}/lineage` 返回上下游关系（split_from / derived_from / contains）
- **谱系追踪**：`GET /api/v1/assets/{id}/provenance` 返回完整版本历史 + 血缘快照

## 关联实体

- **MCAP File**：底层存储文件，Asset 与其为 N:1 关系
- **Algo Run**：算法处理记录，一个 Asset 可被多个算法处理
- **Delivery**：数据交付，一个 Delivery 可包含多个 Asset
- **Action/Annotation**：seg 内的时间分段标注

## 代码示例

具体的 SDK 和 API 调用示例见 [资产管理指南](../guides/asset-management.md)。
