# Asset Management

本章提供资产管理操作的完整指南，涵盖查询、创建到维护的全流程。

## 初始化客户端

```python
from cyber_databrew_sdk import CyberDatabrewClient

client = CyberDatabrewClient(
    token="g-xxx",
    base_url="https://cyber-databrew.cyberorigin.ai",
)
```

或通过环境变量：

```bash
export CYBER_DATABREW_BASE_URL="https://cyber-databrew.cyberorigin.ai"
export CYBER_DATABREW_TOKEN="g-xxx"
```

## 查询资产

### 搜索与列表

资产列表查询通过 Query API 进行：

```bash
curl -X POST "$BASE/api/v1/queries/run" \
  -H "X-Databrew-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "schema_version": "v1",
    "scope": {"resource": "assets"},
    "page": {"page": 1, "page_size": 20}
  }'
```

SDK 调用：

```python
# SDK 搜索接口（推荐）
result = client.search.assets(q="keyword", page=1, page_size=20)
for item in result["items"]:
    print(item["asset_id"], item["asset_type"])

# SDK 列表（当前需配合 queries 接口）
result = client.queries.run({
    "schema_version": "v1",
    "scope": {"resource": "assets"},
    "page": {"page": 1, "page_size": 20},
})
for item in result["items"]:
    print(item["asset_id"], item["lifecycle_state"])
```

### 获取单个资产

```python
asset = client.assets.get("aset0001")
print(f"Asset {asset['asset_id']}: state={asset['lifecycle_state']}")
```

```bash
curl "$BASE/api/v1/assets/aset0001" \
  -H "X-Databrew-Token: $TOKEN"
```

### 批量获取

```python
result = client.assets.batch_get(asset_ids=["aset0001", "aset0002"])
for item in result["items"]:
    print(item["asset_id"])
```

## 创建资产

```python
asset = client.assets.create({
    "mcap_file_id": "mcap0001",
    "start_timestamp_ns": 1700000000000000000,
    "end_timestamp_ns": 1700000060000000000,
    "reviewer": "alice",
    "owner": "team-a",
    "asset_type": "segment",
})
print(f"Created: {asset['asset_id']}")
```

| 参数 | 必填 | 说明 |
|------|------|------|
| `mcap_file_id` | 是 | 关联的 MCAP 文件 ID（须已存在） |
| `start_timestamp_ns` | 是 | 资产起始时间戳（纳秒，不能为 0） |
| `end_timestamp_ns` | 是 | 资产结束时间戳（须 > start） |
| `reviewer` | 是 | 审核人 |
| `owner` | 否 | 所有者 |
| `asset_type` | 否 | 资产类型，默认 segment |
| `tags` | 否 | 标签键值对 |

## 更新资产

```python
# 更新状态
asset = client.assets.update("aset0001", {"lifecycle_state": "ready"})

# 更新标签
asset = client.assets.update("aset0001", {
    "tags": {"quality": "good", "priority": "high"},
})
```

并发安全：使用乐观锁 `version` 字段。冲突时返回 `409 CONCURRENT_CONFLICT`，请重新拉取后再试。

## 标签管理

```python
# 设置标签
client.assets.set_tags("aset0001", tags=[
    {"key": "priority", "value": "high"},
    {"key": "scene", "value": "indoor"},
])

# 获取标签历史
history = client.assets.get_tag_history("aset0001")

# 删除标签
client.assets.delete_tag("aset0001", key="priority")
```

## 血缘与谱系

```python
# 血缘：查找上下游关系
lineage = client.assets.get_lineage("aset0001")
for node in lineage.get("nodes", []):
    print(f"{node['relation_type']}: {node['asset_id']}")

# 谱系：版本历史 + 血缘快照
provenance = client.assets.get_provenance("aset0001")
for rev in provenance.get("revisions", []):
    print(f"v{rev['revision']}: {rev['asset_id']}")
```

## 错误处理

```python
from cyber_databrew_sdk.exceptions import NotFoundError, BadRequestError

try:
    asset = client.assets.get("nonexistent")
except NotFoundError as e:
    print(f"资产不存在: {e.message}")
    print(f"错误码: {e.code}, request_id: {e.request_id}")
except BadRequestError as e:
    print(f"请求参数错误: {e.message}")
    print(f"HTTP 状态: {e.http_status}, details: {e.details}")
```

SDK 异常字段与 API 标准错误体保持一致：

| SDK 字段 | API 字段 | 说明 |
|----------|----------|------|
| `e.code` | `code` | 稳定错误码，适合程序分支判断 |
| `e.message` | `message` | 面向人的错误说明 |
| `e.request_id` | `request_id` / `X-Request-ID` | 排查问题时提供给平台团队 |
| `e.details` | `details` | 结构化错误上下文，可能为空 |
| `e.http_status` | HTTP status | SDK 根据状态码映射到 typed exception |
