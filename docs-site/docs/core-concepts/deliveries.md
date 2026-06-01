# Deliveries

**交付（Delivery）** 是 Cyber Databrew 中面向客户的数据分发机制。通过 Delivery，你可以将选定的资产按照特定规则打包，交付给外部客户或合作伙伴。

> SDK 管理器名称为 `client.delivery`（单数）。

## 生命周期

```
draft → delivered → accepted
                  ↘ cancelled
```

两步提交模式：先创建草稿，添加内容项，确认无误后提交。

## 核心字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `delivery_id` | UUID | 交付唯一 ID |
| `customer_id` | string | 目标客户 slug |
| `status` | enum | draft / delivered / cancelled / accepted |
| `asset_count` | int | 包含的资产数量 |
| `owner` | string | 交付负责人 |
| `note` | string | 备注 |
| `created_at` | datetime | 创建时间 |
| `delivered_at` | datetime | 交付时间 |

## 交付规则引擎

提交时自动校验，命中 block 规则时拒绝交付。客户级 `exclude_tags` 同样会拦截。命中时返回 `422 DELIVERY_RULE_FAILED`。

## 幂等性

`POST /deliveries` 要求 `Idempotency-Key` header：
- 相同 key + 相同 body → 返回已有结果（幂等）
- 相同 key + 不同 body → 409 冲突

## 代码示例

具体的 SDK 和 API 调用示例见 [交付管理指南](../guides/delivery-management.md)。
