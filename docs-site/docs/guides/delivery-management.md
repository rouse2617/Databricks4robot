# Delivery Management

本章提供交付管理的实践操作指南。

> SDK 管理器名称为 `client.delivery`（单数）。

## 创建交付

### 一步创建

```python
from cyber_databrew_sdk import CyberDatabrewClient

client = CyberDatabrewClient(token="g-xxx")

delivery = client.delivery.create({
    "asset_ids": ["aset0001", "aset0002"],
    "customer_id": "acme_corp",
    "note": "Q1 batch",
    "owner": "team-a",
})
print(f"Delivery: {delivery['delivery_id']}, status: {delivery['status']}")
```

### 两步提交模式

适用于需要多次添加内容的场景：

```python
# 第一步：创建草稿
draft = client.delivery.draft({
    "customer_id": "acme_corp",
    "asset_ids": ["aset0001"],
})

# 第二步：追加内容
client.delivery.add_items(draft["delivery_id"], {
    "asset_ids": ["aset0002", "aset0003"],
})

# 第三步：提交
client.delivery.commit(draft["delivery_id"], {
    "expected_revision": 1,
    "approved_by": "ops@databrew",
})
```

## 生命周期管理

```python
# 取消
client.delivery.cancel("delivery-id")

# 重试（仅 failed / cancelled）
new_delivery = client.delivery.retry("delivery-id")

# 客户确认（delivered → accepted）
client.delivery.ack("delivery-id")
```

## 查询交付

```python
# 列表（支持按状态、客户过滤）
result = client.delivery.list(page=1, page_size=20, status="delivered")
for item in result["items"]:
    print(item["delivery_id"], item["status"])

# 详情
delivery = client.delivery.get("delivery-id")
print(delivery["customer_id"], delivery["status"])

# 内容项
items = client.delivery.get_items("delivery-id")
for item in items.get("items", []):
    print(item["asset_id"])
```

## 交付规则

```python
rules = client.delivery.list_rules()
for rule in rules.get("items", []):
    print(rule["name"], rule["enforce_mode"])
```
