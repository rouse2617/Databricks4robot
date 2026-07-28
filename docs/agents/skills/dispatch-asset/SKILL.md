---
name: dispatch-asset
description: 向 Databrew 下发资产触发流水线运行,通过 GCP Pub/Sub 消息实现。当用户说「下发资产到 databrew」「发消息触发 pipeline」「让 databrew 跑一下这个资产」「触发订阅任务」「publish asset to databrew」「dispatch to subscription task」「trigger a databrew pipeline for asset X」时触发。覆盖:消息格式、单条 vs 批次语义、gcloud/Python 发消息示例、权限要求,以及如何验证下发是否成功。
---

# Dispatch an asset to Databrew

外部系统把资产 ID 通过 **GCP Pub/Sub** 推给 Databrew,Databrew 订阅后自动建 pipeline run 或批次。**这是 Databrew 唯一的对外自动化下发入口**(不走 REST 定时任务)。

## When to use

用户想让 Databrew 处理某个/某几个资产 —— 触发一次流水线执行,而不是登录 UI 手动建批次。典型说法:

- "帮我下发资产 XXX 到 databrew"
- "让 cc2-mcap-jinyao 跑一下这个 asset"
- "publish this asset id to trigger the pipeline"
- "触发订阅任务处理这几个 id"

## What happens

```
publish → Pub/Sub topic → Databrew 消费者(10s 轮询)→ 建 run 或批次
                                                    ↓
                             1 个资产 → 1 pipeline run(进「执行记录」)
                             ≥2 个    → 1 批次(进「批量任务」)
```

- **每个订阅任务的每个流水线绑定各下发一次**(fan-out)。绑定 2 个模板 → 一条消息触发 2 个 run/批次。
- 时延:发出到下发通常 ~5s(轮询节拍 10s,均值 5s)。
- **At-least-once**:失败会重试,可能造成重复下发(资产层幂等)。

## Prerequisites (一次性配置)

### 1. GCP publisher 权限

发布方的 GCP 身份(SA 或人)需要 topic 的 publisher 权限。运维一次性执行:

```bash
gcloud pubsub topics add-iam-policy-binding cyber-databrew-asset-events \
  --project=green-valley-442103 \
  --member="serviceAccount:<发布方-sa>@<其项目>.iam.gserviceaccount.com" \
  --role="roles/pubsub.publisher"
# 用户账号: --member="user:xxx@cyberorigin.ai"
```

### 2. Databrew 侧订阅任务已配置

必须在 Databrew UI(流水线 → 订阅任务)预先建好订阅任务,指定订阅 ID + 流水线绑定。**该任务不启用则消息不会被处理。**

## Send a message

### 推荐:用仓库脚本 `scripts/dispatch-asset.sh`

一条命令搞定,不用手写 JSON、不用背 gcloud 参数、可自带验证:

```bash
# 单个资产 → 建 1 个 pipeline run
scripts/dispatch-asset.sh 019de9a4-176a-705a-987b-8bf49e0c9075

# 一批资产 → 建 1 个批次
scripts/dispatch-asset.sh asset-a asset-b asset-c

# 从文件批量(每行一个 id,支持 # 注释)
scripts/dispatch-asset.sh --file my-assets.txt

# 预留 topic 字段(未来路由/标注,当前不生效)
scripts/dispatch-asset.sh -t mcap-slimmer 019de9a4-...

# 发完自动验证 30s 内 Databrew 是否消费到
scripts/dispatch-asset.sh --verify 019de9a4-...
```

环境变量可覆盖(默认为 dev):
- `DATABREW_PROJECT`(默认 `green-valley-442103`)
- `DATABREW_TOPIC`(默认 `cyber-databrew-asset-events`)
- `DATABREW_CLOUDRUN_SERVICE`(默认 `cyber-databrew-backend-dev`,仅 `--verify` 用)

### Topic + 消息格式(参考,脚本已封装)

- **Topic**: `projects/green-valley-442103/topics/cyber-databrew-asset-events`
- **Data**(JSON):

```json
{
  "asset_ids": ["019de9a4-176a-705a-987b-8bf49e0c9075"],
  "topic": "可选、预留"
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `asset_ids` | `string[]` **必填** | 1 个 → 建单 run;≥2 → 建批次 |
| `topic` | `string` 可选 | 预留字段,当前解析但不使用(未来路由/标注) |
| 其他字段 | — | 一律忽略(前向可扩展) |

**不再支持**:旧的 `{"asset_id": "x"}` 单字段格式(消息会被跳过)。

### 直接调用(不用脚本时)

<details><summary>gcloud CLI</summary>

```bash
gcloud pubsub topics publish cyber-databrew-asset-events \
  --project=green-valley-442103 \
  --message='{"asset_ids": ["019de9a4-176a-705a-987b-8bf49e0c9075"]}'
```
</details>

<details><summary>Python</summary>

```python
from google.cloud import pubsub_v1
import json

pub = pubsub_v1.PublisherClient()
topic = "projects/green-valley-442103/topics/cyber-databrew-asset-events"
message_id = pub.publish(
    topic,
    json.dumps({"asset_ids": ["019de9a4-176a-705a-987b-8bf49e0c9075"]}).encode("utf-8"),
).result()
print("published:", message_id)
```
</details>

<details><summary>Go</summary>

```go
client, _ := pubsub.NewClient(ctx, "green-valley-442103")
result := client.Topic("cyber-databrew-asset-events").Publish(ctx, &pubsub.Message{
    Data: []byte(`{"asset_ids": ["019de9a4-..."]}`),
})
msgID, _ := result.Get(ctx)
```
</details>

## Verify(必做)

发完消息务必验一次下发发生了 —— 不然出问题不知道:

### A. UI(最直观)

Databrew UI → 流水线 → **订阅任务** tab → 点任务名 → **下发历史抽屉**
- 应该看到新的一行(newest-first),标签「批次」或「单 run」
- 「单 run」可点进 run 详情看节点执行
- 「批次」可展开看每个资产的状态

### B. gcloud(命令行验证)

```bash
# 30s 内看有没有 "subtask: dispatched" 日志
gcloud logging read \
  'resource.type="cloud_run_revision"
   AND resource.labels.service_name="cyber-databrew-backend-dev"
   AND jsonPayload.msg="subtask: dispatched"' \
  --project=green-valley-442103 --freshness=2m --limit=5 \
  --format="value(timestamp,jsonPayload.runs,jsonPayload.batches,jsonPayload.messages)"
# runs=N batches=M messages=K → 消费了 K 条消息,建了 N 个 run + M 个批次
```

### C. Databrew REST API(需要 X-Databrew-Token)

```bash
BASE=https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app  # dev
TASK_ID=sub_...   # 从 GET /api/v1/subscription-tasks 里查

# 该任务下发过的单 run
curl -sS -H "X-Databrew-Token: $TOK" \
  "$BASE/api/v1/runs?createdBy=subscription-task:$TASK_ID&excludeBatch=true"

# 该任务下发过的批次
curl -sS -H "X-Databrew-Token: $TOK" \
  "$BASE/api/v1/backfill?createdBy=subscription-task:$TASK_ID"
```

## 常见坑

1. **消息发了但没下发** → 90% 是订阅任务被禁用/发到了错的 topic。先在 UI 看任务 `enabled` 状态,再确认 topic 名字。
2. **看不到「单 run」,只看到批次** → 你发了 `{"asset_ids": ["a"]}` 但**版本比 CYB-3801 老**;确认 dev/prod 是否已部署此特性。
3. **PermissionDenied** → 发布方 SA 没 `roles/pubsub.publisher`(找运维加);Databrew SA 没 `roles/pubsub.subscriber`(不同问题,内部错误)。
4. **多任务抢消息** → 一个订阅只挂一个订阅任务。若两个任务共用同一订阅,消息会被瓜分(谁先 pull 谁拿到)。要专属处理路径就建专属 topic + 订阅。
5. **旧格式 `{"asset_id": "x"}`** → 已不支持;必须用数组形式 `{"asset_ids": ["x"]}`。

## Environment

| 环境 | GCP 项目 | Backend Base URL(用于 API 验证) |
|------|---------|----------------------------------|
| dev | `green-valley-442103` | `https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app` |
| prod | `green-valley-442103` | `https://cyber-databrew-backend-prod-wtttm6suaq-uc.a.run.app` |

Topic 名字目前 dev/prod 共用 `cyber-databrew-asset-events`。若你的环境不同,以订阅任务配置为准。

## 深入阅读

- [`docs/review/subscription-task-integration.md`](../../../review/subscription-task-integration.md) — 完整发布方接入指南
- [`docs/review/api-guide.md`](../../../review/api-guide.md) — Subscription Tasks / 历史下发 REST 接口
- CYB-3778(架构)/ CYB-3798(下发历史)/ CYB-3801(单 vs 批 + 消息格式)
