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
| GET | `/api/v1/assets/{id}` | 获取资产详情 |
| PUT | `/api/v1/assets/{id}` | 更新资产 |
| POST | `/api/v1/assets/batch-get` | 批量获取资产 |
| GET | `/api/v1/assets/{id}/lineage` | 资产血缘 |
| GET | `/api/v1/assets/{id}/provenance` | 资产谱系 |
| POST | `/api/v1/assets/{id}/views` | 记录浏览 |
| POST | `/api/v1/assets/{id}/favorite` | 切换收藏 |

### Tags（标签管理）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/assets/{id}/tags` | 设置标签 |
| DELETE | `/api/v1/assets/{id}/tags/{key}` | 删除标签 |

### Deliveries（交付管理）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/deliveries` | 查询交付列表 |
| POST | `/api/v1/deliveries` | 创建交付 |
| GET | `/api/v1/deliveries/{id}` | 获取交付详情 |
| POST | `/api/v1/deliveries/draft` | 创建草稿 |
| POST | `/api/v1/deliveries/{id}/items` | 添加内容项 |
| POST | `/api/v1/deliveries/{id}/commit` | 提交交付 |
| POST | `/api/v1/deliveries/{id}/cancel` | 取消交付 |
| POST | `/api/v1/deliveries/{id}/retry` | 重试交付 |

### Algo Runs（算法运行）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/algo-runs` | 查询算法运行 |
| POST | `/api/v1/algo-runs` | 创建算法运行 |
| GET | `/api/v1/algo-runs/{id}` | 获取运行详情 |
| POST | `/api/v1/algo-runs/{id}/start` | 启动运行 |
| POST | `/api/v1/algo-runs/{id}/finish` | 完成运行 |
| POST | `/api/v1/algo-runs/{id}/cancel` | 取消运行 |

### Storage（文件存储）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/storage/files` | 文件列表 |
| GET | `/api/v1/storage/files/{id}` | 文件信息 |
| GET | `/api/v1/storage/files/{id}/download` | 下载 MCAP |
| GET | `/api/v1/storage/assets/{id}/download` | 资产关联下载 |
| GET | `/api/v1/storage/files/{id}/messages` | 遍历消息 |

### Customers（客户管理）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/customers` | 客户列表 |
| POST | `/api/v1/customers` | 创建客户 |
| GET | `/api/v1/customers/{id}` | 客户详情 |
| PUT | `/api/v1/customers/{id}` | 更新客户 |

### Queries（查询）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/queries/validate` | 验证 Query IR |
| POST | `/api/v1/queries/run` | 执行查询 |
| GET | `/api/v1/saved-queries` | 保存查询列表 |
| POST | `/api/v1/saved-queries` | 创建保存查询 |
| GET | `/api/v1/saved-queries/{id}` | 获取保存查询 |
| PUT | `/api/v1/saved-queries/{id}` | 更新保存查询 |
| DELETE | `/api/v1/saved-queries/{id}` | 删除保存查询 |

### Events（事件）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/events` | 全局事件 |
| GET | `/api/v1/events/assets/{id}` | 资产事件 |

### Audit（审计）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/audit/search` | 审计搜索 |

### Registry（注册中心）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/registry/algos` | 算法列表 |
| GET | `/api/v1/registry/tags` | 标签列表 |
| GET | `/api/v1/registry/metrics` | 指标列表 |

### Lakehouse（湖仓）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/lakehouse/report` | 湖仓报表 |
| GET | `/api/v1/lakehouse/status` | 湖仓状态 |
| GET | `/api/v1/lakehouse/tables` | 表信息 |
| GET | `/api/v1/lakehouse/sync-progress` | 同步进度 |

## 错误码

| HTTP 状态码 | 说明 |
|-------------|------|
| 200 | 请求成功 |
| 400 | 请求参数错误 |
| 401 | 认证失败（Token 无效） |
| 403 | 无权限访问 |
| 404 | 资源不存在 |
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
