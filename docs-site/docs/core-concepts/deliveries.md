# Deliveries

**交付（Delivery）** 是 Cyber Databrew 中面向客户的数据分发机制。通过 Delivery，你可以将选定的资产按照特定规则打包，交付给外部客户或合作伙伴。

## Delivery 的生命周期

一个 Delivery 经历以下状态：

```
草稿 (draft) → 已提交 (committed) → 处理中 → 已完成/失败
```

其中草稿 → 提交为两步提交模式，允许在提交前多次添加内容。

## 创建交付

### 一步创建

```python
from cyber_databrew_sdk import CyberDatabrewClient

client = CyberDatabrewClient(base_url="...", token="g-xxx")

# 直接创建交付（含内容项）
delivery = client.deliveries.create(
    customer_id="cust-001",
    items=[
        {"asset_id": "asset-1", "mcap_file_id": "file-1"},
        {"asset_id": "asset-2", "mcap_file_id": "file-2"},
    ],
)
```

### 两步提交模式

```python
# 第一步：创建草稿
draft = client.deliveries.create_draft(customer_id="cust-001")

# 第二步：添加内容项
client.deliveries.add_items(draft.delivery_id, items=[
    {"asset_id": "asset-1", "mcap_file_id": "file-1"},
])

# 第三步：提交
client.deliveries.commit(draft.delivery_id)
```

## 管理交付

### 取消交付

```python
client.deliveries.cancel("delivery-id-xxx")
```

### 重试交付

```python
client.deliveries.retry("delivery-id-xxx")
```

### 确认接收

```python
client.deliveries.acknowledge("delivery-id-xxx")
```

## 查询交付

```python
# 查询交付列表
deliveries = client.deliveries.list(page=1, page_size=20)

# 获取单个交付详情
delivery = client.deliveries.get("delivery-id-xxx")

# 列出交付中的内容项
items = client.deliveries.list_items("delivery-id-xxx")
```

## 交付规则

交付规则定义了数据的处理方式（如格式转换、压缩等）：

```python
# 列出所有交付规则
rules = client.deliveries.list_rules()
```

## 客户管理

SDK 提供基础客户管理功能：

```python
# 创建客户
client.customers.create(customer_id="cust-001", name="ACME Corp")

# 查询客户列表
customers = client.customers.list()

# 获取客户详情
customer = client.customers.get("cust-001")

# 更新客户信息
client.customers.update("cust-001", name="ACME Inc.")
```
