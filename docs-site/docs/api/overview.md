# API Overview

Cyber Databrew 提供 RESTful API，所有操作均可通过 HTTP 请求完成。API 的基础 URL 为：

```
https://api-cyber-databrew.cyberorigin.ai
```

## 认证

所有 API 请求需要在 Header 中携带认证 Token：

```
X-Databrew-Token: g-xxx
```

## API 版本

当前 API 版本为 `0.2.0`，所有端点位于 `/api/v1/` 路径下。

## 端点概览

### Assets（资产管理）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/assets` | 查询资产列表 |
| POST | `/api/v1/assets` | 创建资产 |
| GET | `/api/v1/assets/{asset_id}` | 获取资产详情 |
| PATCH | `/api/v1/assets/{asset_id}` | 更新资产 |
| DELETE | `/api/v1/assets/{asset_id}` | 删除资产 |
| POST | `/api/v1/assets:batch_get` | 批量获取资产 |
| GET | `/api/v1/assets/{asset_id}/lineage` | 资产血缘 |
| GET | `/api/v1/assets/{asset_id}/provenance` | 资产谱系 |
| GET | `/api/v1/assets/{asset_id}/timeline` | 资产时间线 |
| GET | `/api/v1/assets/{asset_id}/mcap-locator` | MCAP 定位 |
| GET | `/api/v1/assets/{asset_id}/foxglove-source` | Foxglove 数据源 |
| GET | `/api/v1/assets/{asset_id}/deliveries` | 资产关联交付 |
| GET | `/api/v1/assets/{asset_id}/events` | 资产事件 |
| POST | `/api/v1/assets/{asset_id}/view` | 记录浏览 |
| POST | `/api/v1/assets/{asset_id}/favorite` | 收藏资产 |
| GET | `/api/v1/assets/{asset_id}/revisions` | 版本历史 |

### Tags（标签管理）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/assets/{asset_id}/tags` | 获取标签列表 |
| POST | `/api/v1/assets/{asset_id}/tags` | 设置标签 |
| GET | `/api/v1/assets/{asset_id}/tags/{key}` | 获取单个标签 |
| DELETE | `/api/v1/assets/{asset_id}/tags/{key}` | 删除标签 |
| GET | `/api/v1/assets/{asset_id}/tags/history` | 标签历史 |

### Deliveries（交付管理）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/deliveries` | 查询交付列表 |
| POST | `/api/v1/deliveries` | 创建/提交交付 |
| GET | `/api/v1/deliveries/{delivery_id}` | 获取交付详情 |
| POST | `/api/v1/deliveries/draft` | 创建草稿 |
| POST | `/api/v1/deliveries/{delivery_id}/commit` | 提交草稿 |
| POST | `/api/v1/deliveries/{delivery_id}/cancel` | 取消交付 |
| POST | `/api/v1/deliveries/{delivery_id}/retry` | 重试交付 |
| POST | `/api/v1/deliveries/{delivery_id}/ack` | 确认接收 |
| GET | `/api/v1/deliveries/{delivery_id}/items` | 内容项列表 |
| POST | `/api/v1/deliveries/{delivery_id}/items` | 添加内容项 |

### Algo Runs（算法运行）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/algo-runs` | 查询算法运行 |
| POST | `/api/v1/algo-runs` | 创建算法运行 |
| GET | `/api/v1/algo-runs/{run_id}` | 获取运行详情 |
| POST | `/api/v1/algo-runs/{run_id}/start` | 启动运行 |
| POST | `/api/v1/algo-runs/{run_id}/finish` | 完成运行 |
| POST | `/api/v1/algo-runs/{run_id}/cancel` | 取消运行 |
| GET | `/api/v1/algo-runs/{run_id}/affected-assets` | 受影响资产 |

### MCAP 文件存储

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/mcap-files` | 文件列表 |
| POST | `/api/v1/mcap-files` | 创建文件记录 |
| GET | `/api/v1/mcap-files/{mcap_id}` | 文件信息 |
| GET | `/api/v1/mcap-files/{mcap_file_id}/bytes` | 下载 MCAP 文件 |
| HEAD | `/api/v1/mcap-files/{mcap_file_id}/bytes` | 文件大小/元信息 |
| POST | `/api/v1/mcap/upload/finalize` | 上传完成确认 |
| GET | `/api/v1/mcap/{mcap_id}/messages` | 遍历 MCAP 消息 |

### Customers（客户管理）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/customers` | 客户列表 |
| POST | `/api/v1/customers` | 创建客户 |
| GET | `/api/v1/customers/{customer_id}` | 客户详情 |
| PATCH | `/api/v1/customers/{customer_id}` | 更新客户 |

### Queries（查询）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/queries/validate` | 验证 Query IR |
| POST | `/api/v1/queries/run` | 执行查询 |
| GET | `/api/v1/saved-queries` | 保存查询列表 |
| POST | `/api/v1/saved-queries` | 创建保存查询 |
| GET | `/api/v1/saved-queries/{query_id}` | 获取保存查询 |
| PATCH | `/api/v1/saved-queries/{query_id}` | 更新保存查询 |
| DELETE | `/api/v1/saved-queries/{query_id}` | 删除保存查询 |

### Events（事件）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/events` | 全局事件 |
| GET | `/api/v1/assets/{asset_id}/events` | 资产事件 |
| GET | `/api/v1/assets/{asset_id}/events/stream` | SSE 事件流 |

### Audit（审计）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/audit/search` | 审计搜索 |
| GET | `/api/v1/audit/lineage-search` | 血缘审计搜索 |

### Registry（注册中心）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/algo-registry` | 算法列表 |
| GET | `/api/v1/tag-registry` | 标签列表 |
| GET | `/api/v1/metric-registry` | 指标列表 |
| GET | `/api/v1/action-label-registry` | 操作标签列表 |
| GET | `/api/v1/lifecycle-states` | 生命周期状态列表 |

### Lakehouse（湖仓）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/lakehouse/report` | 湖仓报表 |
| GET | `/api/v1/lakehouse/status` | 湖仓状态 |
| GET | `/api/v1/lakehouse/sync-status` | 湖仓同步状态 |
| GET | `/api/v1/lakehouse/sync-progress` | 同步进度 |
| GET | `/api/v1/lakehouse/overview` | 湖仓概览 |
| GET | `/api/v1/lakehouse/tables` | 表信息 |
| GET | `/api/v1/lakehouse/asset-growth` | 资产增长数据 |
| GET | `/api/v1/lakehouse/event-daily` | 每日事件统计 |
| GET | `/api/v1/lakehouse/event-type-share` | 事件类型分布 |
| GET | `/api/v1/lakehouse/failure-clusters` | 故障分析 |
| GET | `/api/v1/lakehouse/quality-distribution` | 数据质量分布 |
| GET | `/api/v1/lakehouse/customer-replay` | 客户回放数据 |

## 错误码

| HTTP 状态码 | 说明 |
|-------------|------|
| 200 | 请求成功 |
| 400 | 请求参数错误 |
| 401 | 认证失败（Token 无效） |
| 403 | 无权限访问 |
| 404 | 资源不存在 |
| 409 | 资源冲突 |
| 429 | 请求过于频繁（限流） |
| 500 | 服务端内部错误 |

## 使用 curl

```bash
# 获取资产列表
curl -H "X-Databrew-Token: g-xxx" \
  https://api-cyber-databrew.cyberorigin.ai/api/v1/assets?page=1&page_size=10

# 创建资产
curl -X POST \
  -H "X-Databrew-Token: g-xxx" \
  -H "Content-Type: application/json" \
  -d '{"mcap_file_id": "file-001", "t_start": 1000000000, "t_end": 2000000000}' \
  https://api-cyber-databrew.cyberorigin.ai/api/v1/assets
```
