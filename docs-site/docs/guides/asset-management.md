# Asset Management

本章提供资产管理操作的完整指南，涵盖从创建、查询到维护的全流程操作。

## 初始化客户端

```python
from cyber_databrew_sdk import CyberDatabrewClient

client = CyberDatabrewClient(
    base_url="https://api-cyber-databrew.cyberorigin.ai",
    token="g-xxx",
)
```

## 创建资产

### 基本创建

```python
asset = client.assets.create(
    mcap_file_id="file-001",
    t_start=1000000000,
    t_end=2000000000,
)
```

### 参数说明

| 参数 | 必填 | 说明 |
|------|------|------|
| `mcap_file_id` | 是 | 关联的 MCAP 存储文件 ID |
| `t_start` | 是 | 资产起始时间戳（纳秒） |
| `t_end` | 是 | 资产结束时间戳（纳秒） |

## 查询资产

### 列表查询

```python
# 基础分页查询
assets = client.assets.list(page=1, page_size=20)

# 按创建时间倒序
assets = client.assets.list(page=1, page_size=20, sort_by="-created_at")

# 按状态过滤
assets = client.assets.list(page=1, page_size=20, status="reviewed")
```

### 获取单个资产

```python
asset = client.assets.get("asset-id-xxx")
print(f"Asset {asset.asset_id}: status={asset.status}")
```

### 批量获取

```python
assets = client.assets.batch_get(asset_ids=["id1", "id2", "id3"])
```

## 更新资产

```python
# 更新状态
asset = client.assets.update("asset-id-xxx", status="reviewed")

# 更新时间范围
asset = client.assets.update("asset-id-xxx", t_start=2000000000, t_end=3000000000)
```

## 标签管理

```python
# 设置标签
client.assets.set_tag("asset-id-xxx", key="priority", value="high")
client.assets.set_tag("asset-id-xxx", key="source", value="lidar")

# 删除标签
client.assets.delete_tag("asset-id-xxx", key="priority")
```

## 血缘谱系

```python
# 血缘：找出衍生资产
lineage = client.assets.lineage("asset-id-xxx")
for child in lineage:
    print(f"Child: {child.asset_id}")

# 谱系：找出来源
provenance = client.assets.provenance("asset-id-xxx")
for parent in provenance:
    print(f"Parent: {parent.asset_id}")
```

## 浏览与收藏

```python
# 记录浏览（用于热度分析/推荐）
client.assets.record_view("asset-id-xxx")

# 收藏切换
client.assets.toggle_favorite("asset-id-xxx")

# 再次调用取消收藏
client.assets.toggle_favorite("asset-id-xxx")
```

## 错误处理

```python
from cyber_databrew_sdk.exceptions import NotFoundError, BadRequestError

try:
    asset = client.assets.get("nonexistent-id")
except NotFoundError as e:
    print(f"资产不存在: {e}")
except BadRequestError as e:
    print(f"请求参数错误: {e}")
```
