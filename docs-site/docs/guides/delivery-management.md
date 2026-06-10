# Delivery Management

本章提供交付管理的完整操作指南。

> ⚠️ 注意：SDK 中的管理器名称为 `client.delivery`（单数），而非 `client.deliveries`。

## 创建交付

### 快速创建

```python
from cyber_databrew_sdk import CyberDatabrewClient

client = CyberDatabrewClient(token="g-xxx")

# 创建交付并添加内容
delivery = client.delivery.create(payload={
    "customer_id": "cust-001",
    "items": [
        {"asset_id": "asset-1", "mcap_file_id": "file-1"},
    ],
})
print(f"Delivery created: {delivery.delivery_id}")
```

### 两步提交模式

适用于需要多次添加内容的场景：

```python
from cyber_databrew_sdk import CyberDatabrewClient

client = CyberDatabrewClient(token="g-xxx")

# 第一步：创建草稿
draft = client.delivery.draft(payload={"customer_id": "cust-001"})

# 第二步：逐批添加内容
client.delivery.add_items(draft["delivery_id"], payload={
    "items": [
        {"asset_id": "asset-1", "mcap_file_id": "file-1"},
    ],
})
client.delivery.add_items(draft["delivery_id"], payload={
    "items": [
        {"asset_id": "asset-2", "mcap_file_id": "file-2"},
    ],
})

# 第三步：提交
client.delivery.commit(draft["delivery_id"])
```

## 交付生命周期管理

### 取消交付

```python
# 取消尚未完成的交付
client.delivery.cancel("delivery-id-xxx")
```

### 重试失败交付

```python
# 重试失败的交付
client.delivery.retry("delivery-id-xxx")
```

### 确认接收

```python
# 客户确认接收
client.delivery.ack("delivery-id-xxx")
```

## 查询交付

### 列表查询

```python
deliveries = client.delivery.list(page=1, page_size=20)
for delivery in deliveries:
    print(f"{delivery.delivery_id}: {delivery.status}")
```

### 详情查询

```python
delivery = client.delivery.get("delivery-id-xxx")
print(f"Customer: {delivery.customer_id}, Status: {delivery.status}")
```

### 内容项查询

```python
items = client.delivery.get_items("delivery-id-xxx")
for item in items:
    print(f"Asset: {item.asset_id}, File: {item.mcap_file_id}")
```

## 交付规则

```python
# 列出交付规则
rules = client.delivery.list_rules()
for rule in rules:
    print(f"Rule: {rule}")
```

## 客户管理

```python
# 创建客户
client.customers.create(payload={"customer_id": "cust-001", "name": "ACME Corp"})

# 查询客户列表
customers = client.customers.list()

# 获取客户详情
customer = client.customers.get("cust-001")

# 更新客户
client.customers.update("cust-001", payload={"name": "ACME Inc."})
```

## Lakehouse 湖仓

交付数据同步到湖仓后的查询：

```python
# 湖仓报表
report = client.lakehouse.get_report()

# 湖仓状态
status = client.lakehouse.get_status()

# 表信息
tables = client.lakehouse.get_tables()

# 概览
overview = client.lakehouse.get_overview()

# 同步进度
progress = client.lakehouse.get_sync_progress()
```
