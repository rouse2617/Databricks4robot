# Tags & Search

Cyber Databrew 提供灵活的**标签系统**和**全局搜索**功能，帮助你快速组织和发现资产。

## 标签系统

标签以键值对形式附加在资产上，支持**多源标签合并**与历史记录追踪。

### 多源标签

同一 `(asset_id, tag_key)` 上允许 `human` / `algo_sdk` / `rule_engine` / `system` / `compliance` 多源共存：

- 扁平投影 `tags`：按 `applied_at` 取最新值（兼容旧消费者）
- 完整多源数据 `tags_detailed`：每行带 `source_type` / `source_name` / `source_version` / `run_id`
- 标签历史通过 `GET /api/v1/assets/{id}/tags/history` 查询

### 标签注册中心

`tag_registry.yaml` 定义了所有可用标签键和允许值（白名单）。未注册的 key 会被拒绝（`422 INVALID_TAG`）。

## Elasticsearch 搜索

平台使用 Elasticsearch 提供全文搜索，搜索入口统一走 Query API：

```bash
curl -X POST "$BASE/api/v1/queries/run" \
  -H "X-Databrew-Token: $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "schema_version": "v1",
    "mode": "keyword",
    "scope": {"resource": "assets"},
    "where": {"pred": {"field": "_fulltext", "op": "ilike", "value": "warehouse rain"}},
    "page": {"page": 1, "page_size": 20}
  }'
```

SDK 调用：

```python
from cyber_databrew_sdk import CyberDatabrewClient

client = CyberDatabrewClient(token="g-xxx")

# 搜索资产
result = client.search.assets(q="keyword", page=1, page_size=20)
for item in result["items"]:
    print(item["asset_id"], item["asset_type"])

# 同步状态
status = client.search.get_sync_status()
```

## Query IR 查询

平台支持基于**中间表示（Query IR）** 的查询引擎：

```python
# 执行查询
result = client.queries.run({
    "schema_version": "v1",
    "scope": {"resource": "assets"},
    "where": {"pred": {"field": "owner", "op": "eq", "value": "alice"}},
    "page": {"page": 1, "page_size": 20},
})
for item in result["items"]:
    print(item["asset_id"])

# 保存查询管理
client.queries.create_saved({
    "name": "my-query",
    "query_ir": {"schema_version": "v1", "scope": {"resource": "assets"}},
})
```

## 事件流

关注资产的实时变化：

```python
# 资产事件
result = client.events.list_for_asset("aset0001")
for event in result.get("items", []):
    print(event["event_type"], event["created_at"])

# SSE 流（实时推送）
for event in client.events.stream_for_asset("aset0001"):
    print(event["event_type"])
```

标签和搜索的完整操作见 [资产管理指南](../guides/asset-management.md)。
