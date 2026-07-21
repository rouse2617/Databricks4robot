# Subscription Task — Publisher Integration Guide

Databrew 订阅任务通过 GCP Pub/Sub 接收 asset ID，自动创建批量处理任务。外部系统往指定 topic 发消息，Databrew 订阅消费后自动下发。

## 接入步骤

### 1. 创建 Pub/Sub 资源

在 GCP Console 或 gcloud CLI 创建 topic + subscription：

```bash
# 创建 topic
gcloud pubsub topics create databrew-ingest-<your-name> \
  --project=green-valley-442103

# 创建 subscription（Databrew 消费端）
gcloud pubsub subscriptions create databrew-ingest-<your-name>-sub \
  --topic=databrew-ingest-<your-name> \
  --project=green-valley-442103 \
  --ack-deadline=60 \
  --message-retention-duration=7d
```

### 2. 配置权限

| 角色 | 授予对象 | 说明 |
|------|---------|------|
| `roles/pubsub.publisher` | 发布方 SA | 允许向 topic 发消息 |
| `roles/pubsub.subscriber` | Databrew SA (`cyber-databrew-dev@...`) | 允许从 subscription 拉消息 |

```bash
# 授予 Databrew SA 订阅权限
gcloud pubsub subscriptions add-iam-policy-binding databrew-ingest-<your-name>-sub \
  --project=green-valley-442103 \
  --member="serviceAccount:cyber-databrew-dev@green-valley-442103.iam.gserviceaccount.com" \
  --role="roles/pubsub.subscriber"
```

### 3. 在 Databrew UI 创建订阅任务

进入 **流水线 → 订阅任务** tab → 新建：

| 字段 | 说明 | 示例 |
|------|------|------|
| 名称 | 任务标识 | `youxin-ingest` |
| GCP 项目 ID | Pub/Sub 所在项目 | `green-valley-442103` |
| 订阅 ID | subscription 名称 | `databrew-ingest-youxin-sub` |
| 拉取间隔（秒） | 消费频率 | `10`（默认） |
| 单次最大消息数 | 每次 pull 上限 | `1000`（默认） |
| 流水线绑定 | 一个或多个「模板 + 资源池」；每条消息对**每个模板各下发一个批次** | `youxin-all v12 → video-proc-dev` |

### 4. 发送消息

每条消息的 data 字段必须是 JSON，包含 `asset_id`：

```json
{"asset_id": "abc123-uuid-here"}
```

支持在消息 attributes 中携带可选元数据（当前版本不解析 attributes，预留扩展）。

#### Python 示例

```python
from google.cloud import pubsub_v1
import json

publisher = pubsub_v1.PublisherClient()
topic = "projects/green-valley-442103/topics/databrew-ingest-youxin"

# 单条
publisher.publish(topic, json.dumps({"asset_id": "video-001"}).encode("utf-8"))

# 批量
for vid in ["video-001", "video-002", "video-003"]:
    publisher.publish(topic, json.dumps({"asset_id": vid}).encode("utf-8"))
```

#### Go 示例

```go
client, _ := pubsub.NewClient(ctx, "green-valley-442103")
topic := client.Topic("databrew-ingest-youxin")
topic.Publish(ctx, &pubsub.Message{
    Data: []byte(`{"asset_id": "video-001"}`),
})
```

#### gcloud CLI（调试用）

```bash
gcloud pubsub topics publish databrew-ingest-youxin \
  --project=green-valley-442103 \
  --message='{"asset_id": "test-video-001"}'
```

## 消费行为

| 行为 | 说明 |
|------|------|
| 拉取频率 | 每 N 秒（默认 10s，per-task 可配） |
| 批次创建 | 同一次 pull 的所有 asset_id 合并；对每个绑定的模板各下发一个 batch（fan-out，N 个模板 → N 个 batch） |
| Ack 时机 | 所有绑定的 batch 都创建成功后 ack；任一失败则 nack 整条消息重试（重试可能重复下发已成功的模板） |
| 空 pull | 无消息时跳过，不创建空 batch |
| 去重 | 不做 — 批量任务层面本身幂等（同 asset 重复下发不会重复处理） |
| 消息格式错误 | 解析失败的消息会 ack（避免毒消息阻塞队列），错误记录到日志 |

## 监控

- Databrew UI 订阅任务列表展示：最近执行时间、状态、批次 ID
- 执行失败时飞书告警通知
- GCP Console 可查看 subscription 的未确认消息数（backlog）
