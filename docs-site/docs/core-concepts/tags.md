# Tags & Search

Cyber Databrew 提供灵活的**标签系统**和**全局搜索**功能，帮助你快速组织和发现资产。

## 标签系统

标签以键值对形式附加在资产上，支持多源标签合并与历史记录追踪。

### 标签 CRUD

```python
from cyber_databrew_sdk import CyberDatabrewClient

client = CyberDatabrewClient(token="g-xxx")

# 设置标签
client.assets.set_tags("asset-id-xxx", tags=[
    {"key": "priority", "value": "high"},
    {"key": "source", "value": "camera-01"},
])

# 删除标签
client.assets.delete_tag("asset-id-xxx", key="priority")
```

### 标签注册中心

平台维护一个标签注册中心，记录了所有可用的标签键和允许的值：

```python
# 列出标签注册中心
tags = client.registry.list_tags()
```

## 搜索

平台使用 Elasticsearch 提供全文搜索功能。

```python
# 搜索资产
results = client.search.assets(q="some keyword", page=1, page_size=20)

# 获取搜索同步状态
status = client.search.get_sync_status()

# 获取同步进度
progress = client.search.get_sync_progress()
```

## Query IR 查询

平台支持基于**中间表示（Query IR）** 的查询引擎，适用于复杂查询场景。

### 验证与执行

```python
# 验证查询 IR
result = client.queries.validate(payload={...})

# 运行查询
result = client.queries.run(payload={...})
```

### 保存查询管理

```python
# 创建保存的查询
client.queries.create_saved(payload={
    "name": "my query",
    "query_ir": {...},
})

# 列出保存的查询
queries = client.queries.list_saved()

# 获取保存的查询
query = client.queries.get_saved("query-id-xxx")

# 更新保存的查询
client.queries.update_saved("query-id-xxx", payload={"name": "new name"})

# 删除保存的查询
client.queries.delete_saved("query-id-xxx")
```

## 事件流

关注资产的实时变化：

```python
# 全局事件
events = client.events.list_global(page=1, page_size=20)

# 资产事件
events = client.events.list_for_asset("asset-id-xxx")

# SSE 事件流（逐行迭代）
for event in client.events.stream_for_asset("asset-id-xxx"):
    print(event)
```
