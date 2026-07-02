# Quick Start

本指南帮助你在 5 分钟内完成 Cyber Databrew 的 SDK 安装和首次 API 调用。

## 前置条件

- Python 3.10 或更高版本
- 有效的 API Token（联系管理员获取）

## 第一步：安装 SDK

```bash
pip install cyber-databrew-sdk
```

> **内部安装**：使用 Artifact Registry 的完整安装命令请参考 [SDK 安装指南](sdk-installation.md)。

## 第二步：初始化客户端

```python
from cyber_databrew_sdk import CyberDatabrewClient

client = CyberDatabrewClient(
    token="your-token-here",
    base_url="https://cyber-databrew.cyberorigin.ai",
)
```

也支持通过环境变量配置：

```bash
export CYBER_DATABREW_BASE_URL="https://cyber-databrew.cyberorigin.ai"
export CYBER_DATABREW_TOKEN="your-token-here"
```

```python
# 自动从环境变量读取配置
client = CyberDatabrewClient()
```

## 第三步：查询资产

```python
# 获取单个资产
asset = client.assets.get("aset0001")
print(asset["asset_id"], asset["lifecycle_state"])

# 搜索资产
result = client.search.assets(q="keyword", page=1, page_size=20)
for item in result["items"]:
    print(item["asset_id"], item["asset_type"])
```

## 第四步：创建资产

```python
asset = client.assets.create({
    "mcap_file_id": "mcap0001",
    "start_timestamp_ns": 1700000000000000000,
    "end_timestamp_ns": 1700000060000000000,
    "reviewer": "alice",
    "owner": "team-a",
})
print(f"Created: {asset['asset_id']}")
```

## 第五步：创建交付

```python
delivery = client.delivery.create({
    "asset_ids": ["aset0001", "aset0002"],
    "customer_id": "acme_corp",
    "note": "My first delivery",
})
print(f"Delivery: {delivery['delivery_id']}, status: {delivery['status']}")
```

## 下一步

- 详细了解 **[认证机制](authentication.md)**
- 深入学习 **[API 参考](../api/overview.md)**
- 探索 **[资产管理指南](../guides/asset-management.md)**
