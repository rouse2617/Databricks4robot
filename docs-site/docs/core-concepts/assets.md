# Assets

**资产（Asset）** 是 Cyber Databrew 平台的核心实体。每个 Asset 代表一条数据记录（通常是视频或多模态数据片段），包含元数据、状态、标签和关联的 MCAP 文件。

## 资产属性

每个资产包含以下核心字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| `asset_id` | string | 全局唯一标识符 |
| `mcap_file_id` | string | 关联的 MCAP 文件 ID |
| `t_start` | int64 | 起始时间戳（纳秒） |
| `t_end` | int64 | 结束时间戳（纳秒） |
| `status` | string | 生命周期状态 |
| `created_at` | datetime | 创建时间 |
| `updated_at` | datetime | 更新时间 |
| `tags` | map | 键值标签 |

## 资产 CRUD

### 查询资产列表

```python
from cyber_databrew_sdk import CyberDatabrewClient

client = CyberDatabrewClient(base_url="...", token="g-xxx")

# 查询资产（支持过滤、排序、分页）
assets = client.assets.list(
    page=1,
    page_size=20,
    sort_by="-created_at",
)

# 遍历结果
for asset in assets:
    print(asset.asset_id, asset.status)
```

### 获取单个资产

```python
# 获取单个资产详情
asset = client.assets.get("asset-id-xxx")
print(asset.asset_id, asset.status, asset.tags)
```

### 创建资产

```python
asset = client.assets.create(
    mcap_file_id="file-001",
    t_start=1000000000,
    t_end=2000000000,
)
print(f"Created: {asset.asset_id}")
```

### 更新资产

```python
asset = client.assets.update("asset-id-xxx", status="reviewed")
```

### 批量获取

```python
assets = client.assets.batch_get(asset_ids=["id1", "id2", "id3"])
```

## 血缘与谱系

```python
# 血缘关系：找出哪些资产由当前资产衍生而来
lineage = client.assets.lineage("asset-id-xxx")

# 谱系追踪：找出当前资产的来源
provenance = client.assets.provenance("asset-id-xxx")
```

## 标签管理

```python
# 设置标签
client.assets.set_tag("asset-id-xxx", key="priority", value="high")

# 删除标签
client.assets.delete_tag("asset-id-xxx", key="priority")
```

## 浏览与收藏

```python
# 记录浏览
client.assets.record_view("asset-id-xxx")

# 切换收藏
client.assets.toggle_favorite("asset-id-xxx")
```
