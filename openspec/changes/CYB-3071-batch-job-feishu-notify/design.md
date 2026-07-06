# Design — CYB-3071

## Architecture Context
- `backfill_jobs` 的状态由 `backend/internal/usecase/backfill/usecase.go` 里的同步逻辑计算（`deriveJobStatus` 调用处，约第 1424 行），由一个内部 watcher 周期性触发，只扫描 `status='running'` 的任务（`FindIncompleteJobs`，`backend/internal/postgres/backfill_repo.go:973`）。
- 一旦任务变成终态，就不会再被这个扫描捞到——这次转变的检测只有一次机会，没有"下次 tick 重试"的自然空间。
- Cloud Run dev 后端 `MAX_INSTANCES=5`（`deploy/cloudrun/backend-dev.sh:29`），随时可能有多个实例同时观察到同一个任务的这次转变，需要跨实例安全的"只通知一次"保证。
- 已有 `pipeline_run_notification_candidates`（CYB-1565，`backend/migrations/048_pipeline_observability.sql`）是为 run/node 级事件设计的通用候选表，`AppendCandidate` 用 `INSERT ... ON CONFLICT (idempotency_key) DO NOTHING` 做 exactly-once。这次不复用它——批量任务是单一状态机事件，专用一列比拖入候选表的语义更直接，避免过早的跨领域抽象。

## Goals
- 批量任务终态通知功能完整可用，同时保持模块边界清晰：飞书发送机制、消息内容拼装、"何时该通知"的判断，三者互不知晓对方的实现细节。
- 通知发送的可靠性只需要覆盖"这次调用内的网络抖动"，不需要跨调用的持久重试基础设施。

## Non-Goals
- 不覆盖流水线 run/step 级失败通知（未来单独的需求）。
- 不做跨越"任务已终态、后续没有重新扫描"这个限制的持久重试。
- 不引入 Argo Events / Cloud Monitoring / 第三方飞书 SDK。

## Affected Modules
- `backend/internal/notify/feishu/client.go`（新）——`Config`、`Sender` 接口、`Client.SendText`。纯粹的"给一段文字和一个 webhook URL，把它发出去"，不 import `backfill`/`pipeline`/`models` 包。
- `backend/internal/usecase/backfill/usecase.go`（或紧邻的小文件，如 `notify.go`）——在终态跨越检测处新增认领 + 通知逻辑；消息文本的拼装逻辑放在这里，不放进 `feishu` 包。
- `backend/internal/usecase/backfill/usecase.go` 的 `Usecase` 结构体——新增 `notifier` 字段（`Sender` 接口类型）与构造/设置方法，参考现有 `SetArgoRunWebhook`（`backend/internal/usecase/pipeline/usecase.go:249`）的注入方式。
- `backend/internal/config/config.go`——新增 `BackfillNotifyFeishuWebhookURL`、`FrontendBaseURL` 等配置字段。
- `backend/migrations/0NN_backfill_notification_sent_at.sql`（新）——`ALTER TABLE backfill_jobs ADD COLUMN notification_sent_at TIMESTAMPTZ`。
- `backend/cmd/server/`——组装/注入 `feishu.Client` 到 `backfill.Usecase`。

## Architecture Decisions

### Decision 1: 专用列而非复用候选表
- **Approach**: `backfill_jobs.notification_sent_at`，一条原子 `UPDATE ... WHERE notification_sent_at IS NULL RETURNING id` 作为跨实例安全的认领。
- **Alternative**: 复用 `pipeline_run_notification_candidates`（`subject_type='batch_job'`），配一个通用投递 worker。
- **Rationale**: 现在只有批量任务这一个真实事件源；候选表 + 通用投递 worker 是为"多个事件源共享一套判断 + 投递逻辑"设计的，只有一个事件源时是提前抽象，大概率跟以后真做 run 级通知时的实际形状不一致。
- **Trade-off**: 以后如果真的把 run 级通知也做了，两条通知路径不共享"判断该不该发"这一层逻辑，只共享 `feishu.SendText` 这一个发送函数——这是可接受的重复，好于猜错的抽象。

### Decision 2: 同步发送，最佳努力，不做跨 tick 重试
- **Approach**: 认领成功后，在同一次调用里同步调用 `Sender.SendText`；失败只记日志，不重试。
- **Alternative**: 认领成功但发送失败时保留某种"待重试"标记，等待下一次机会重试。
- **Rationale**: 任务一旦终态就不会被 watcher 再扫描到（`FindIncompleteJobs` 只认 `status='running'`），没有自然的"下一次机会"——要做真重试，需要一个新的独立扫描逻辑，而这次明确要保持简单。
- **Trade-off**: 如果飞书在那个瞬间恰好不可用，这一次通知会丢失，且没有补偿；任务本身的完成状态不受影响，用户仍可在 UI 上看到正确结果。`feishu.Client` 的 `MaxRetries` 仍会在同一次调用内对网络抖动做几次重试，缓解大部分瞬时故障。

### Decision 3: `feishu` 包与业务逻辑解耦
- **Approach**: `notify/feishu` 包只知道"文本 + webhook URL"，不知道"批量任务"是什么；消息内容拼装、何时调用，都留在 `backfill` 包。`backfill` 依赖一个只有 `SendText` 一个方法的 `Sender` 接口，具体的 `feishu.Client` 实例化只发生在 `cmd/server` 的组装代码里。
- **Alternative**: 直接在 `backfill` usecase 里内联 HTTP 发送逻辑。
- **Rationale**: 保持关注点分离——以后无论是加别的批量事件通知、还是把发送实现换成别的方式（比如以后决定要用 `go-lark`），改动都局限在各自的包内，互不影响。
- **Trade-off**: 多了一个新包、一次依赖注入的组装代码，对于这么小的功能来说有一点仪式感，但换来的解耦对后续维护是值得的。

## Data Model Changes

### `backfill_jobs`（迁移新增列）
| 列 | 类型 | 说明 |
|---|---|---|
| `notification_sent_at` | `TIMESTAMPTZ`，nullable | 默认 `NULL`；非空即表示"已认领并尝试通知"，作为跨实例原子认领标记 |

## `notify/feishu` 包形状

```go
package feishu

type Config struct {
    WebhookURL string        // 空值 = 功能关闭，SendText 直接 no-op
    Timeout    time.Duration // 单次 HTTP 超时，零值默认 10s
    MaxRetries int           // 同一次调用内的重试次数（网络抖动），零值默认 2
}

type Client struct { /* ... */ }

func NewClient(cfg Config) *Client

// Sender 是调用方应该依赖的接口，方便在 usecase 测试里注入假实现。
type Sender interface {
    SendText(ctx context.Context, text string) error
}

func (c *Client) SendText(ctx context.Context, text string) error
```

Payload 格式（复用 `.github/scripts/feishu-notify.sh` 已验证的形状）：
```json
{"msg_type": "text", "content": {"text": "..."}}
```

## Message Format
```
【批量任务完成】<job 名称>
状态：<completed|failed>
总数：<total>　成功：<completed_count>　失败：<failed_count>
链接：<FRONTEND_BASE_URL>/pipeline/batch/<jobID>
```

## Config
| Env var | 说明 | 默认 |
|---|---|---|
| `BACKFILL_NOTIFY_FEISHU_WEBHOOK_URL` | 飞书 webhook 地址，空值关闭功能 | `""` |
| `FRONTEND_BASE_URL` | 拼消息里任务链接用的前端基址；通用配置，不与某个通知功能绑定 | `""`（为空时消息不含链接） |

**已确认**：`BACKFILL_NOTIFY_FEISHU_WEBHOOK_URL` 的值已经写入 dev 集群的 K8s Secret `cyber-databrew-secrets`（namespace `cyber-databrew-dev`），跟 `DB_HOST`/`DB_PASSWORD` 等现有 key 放在一起。`deploy/cloudrun/backend-dev.sh` 默认 `SOURCE_K8S_ENV=true`，部署时会自动把这个 Secret 里的每个 key 合并成 Cloud Run 的明文环境变量（见该脚本第 290-301 行的合并逻辑），不需要改脚本、不需要新建 GCP Secret Manager secret。跟 `ARGO_RUN_WEBHOOK_TOKEN`（走 Secret Manager + `--set-secrets`）是不同的机制——那个是给 Cloud Run 原生的 Secret Manager 绑定，这个是走 K8s Secret 合并，两条路在这个仓库里都是已有的、成立的模式，这次选择了对这个场景更省事的后者。

代码层面：`backend/internal/config/config.go` 仍然只是 `getenv("BACKFILL_NOTIFY_FEISHU_WEBHOOK_URL", "")` 读普通环境变量——从 Go 代码的角度看不出区别，区别只在这个环境变量的值是怎么被喂到 Cloud Run 进程里的。

## Risks / Trade-offs
| Risk | Impact | Mitigation |
|---|---|---|
| 发送失败无跨 tick 重试 | 极少数情况下漏发一条通知 | 任务真实状态不受影响，仅通知层面；后续如证明是真问题再加独立重试扫描 |
| 多实例并发认领 | 理论上可能重复发送 | 原子 `UPDATE ... WHERE notification_sent_at IS NULL` 保证只有一个实例认领成功 |
| `FRONTEND_BASE_URL` 未配置 | 消息里缺少任务链接 | 缺省时消息仍发送，只是不含链接，不阻塞功能 |
