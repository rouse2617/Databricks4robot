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
    base_url="https://api-cyber-databrew.cyberorigin.ai",
    token="your-token-here",
)
```

也支持通过环境变量配置：

```bash
export CYBER_DATABREW_BASE_URL="https://api-cyber-databrew.cyberorigin.ai"
export CYBER_DATABREW_TOKEN="your-token-here"
```

```python
import os
from cyber_databrew_sdk import CyberDatabrewClient

# 自动从环境变量读取配置
client = CyberDatabrewClient()
```

## 第三步：获取资产列表

```python
# 列出资产（支持分页、过滤、排序）
assets = client.assets.list(page=1, page_size=20)

for asset in assets:
    print(f"ID: {asset.asset_id}, Status: {asset.status}")
```

## 第四步：创建资产

```python
# 创建一条资产记录
asset = client.assets.create(
    mcap_file_id="file-001",
    t_start=1000000000,
    t_end=2000000000,
)
print(f"Created asset: {asset.asset_id}")
```

## 第五步：创建交付

```python
# 创建一个数据交付
delivery = client.deliveries.create(
    customer_id="cust-001",
    items=[
        {"asset_id": "asset-1", "mcap_file_id": "file-1"},
    ],
)
print(f"Delivery created: {delivery.delivery_id}")
```

## 下一步

- 详细了解 **[认证机制](authentication.md)**
- 深入学习 **[API 参考](../api/overview.md)**
- 探索 **[资产管理指南](../guides/asset-management.md)**
