# Outbox Relay/Subscriber 测试计划（当前主线）

> 本文档面向当前运行时链路：`asset_events`（PostgreSQL）→ relay（Pub/Sub）→ ES subscriber（Elasticsearch）。
>
> 不再以 Pure CDC 作为默认主线；CDC/WAL 相关文档仅用于历史方案或后续入湖扩展参考。

---

## 1. 范围与目标

### 1.1 测试范围

- `backend/internal/outbox/relay.go`
- `backend/internal/outbox/publisher.go`
- `backend/internal/outbox/es_subscriber.go`
- 与 outbox 状态可观测相关的 API 行为：
  - `GET /api/v1/search/sync-status`
  - `POST /api/v1/admin/search/reindex`（兜底重建）

### 1.2 测试目标

| 目标 | 说明 | 通过标准 |
|---|---|---|
| G1 | relay 只按 `event_seq` 推进且成功后标记 published | 无跳号、失败可重试 |
| G2 | subscriber 正确处理 upsert / delete / version conflict | 非冲突失败可重试，409 不误报失败 |
| G3 | 同步模式可观测 | `sync-status` 返回 outbox 相关字段并与配置一致 |
| G4 | 失败可降级 | subscriber 未启用时可执行 reindex 兜底 |

---

## 2. 当前实现基线

| 组件 | 当前实现 | 备注 |
|---|---|---|
| 事件写入 | 业务事务内写 `asset_events` | 由 usecase/repo 保证 |
| Relay | 轮询 pending，发布 Pub/Sub，成功后 `MarkPublished` | 见 `relay.go` |
| Subscriber | 消费 Pub/Sub，重建 ES 文档并 `BulkIndex` | 见 `es_subscriber.go` |
| DLQ | 超过重试阈值迁移到 `outbox_dlq` | 由仓储层提供 |
| Cursor 表 | `outbox_sink_cursors` 已移除 | `019_drop_outbox_sink_cursors.sql` |

---

## 3. 现有测试资产（仓库内）

| 文件 | 覆盖点 |
|---|---|
| `backend/internal/handlers/search/handler_test.go` | `sync-status` 返回 outbox 模式字段 |
| `backend/routes/routes_test.go` | `GET /healthz/outbox` 已移除（404）与路由回归 |
| `backend/internal/postgres/repos_test.go` | `asset_events` 仓储基础行为（append/list 等） |

> 当前 `internal/outbox` 目录尚无专门 `*_test.go`。本计划将该缺口列为必补项。

---

## 4. 必补测试（建议本轮落地）

### 4.1 `backend/internal/outbox/relay_test.go`

- `flushOnce` happy path：发布成功后调用 `MarkPublished`
- 发布失败路径：调用 `MarkFailed`，不推进成功位点
- `MaxRetries` 行为：超过阈值的事件不继续发布
- ordering key 规则：`asset_id` 优先，其次 `mcap:<id>`，最后 `_na`

### 4.2 `backend/internal/outbox/es_subscriber_test.go`

- message 反序列化失败：返回 error（触发 Nack）
- `Builder.Build` 返回 `ok=false` 时走 `DeleteDocument`
- bulk 部分失败且 `status=409`：视为可接受冲突
- bulk 其它失败：返回 error（触发重试）

### 4.3 `backend/internal/outbox/publisher_test.go`

- project/topic 为空时返回参数错误
- 空 ordering key 自动回退 `_na`
- `ResumePublishAfterError` 空 key 不 panic

---

## 5. 执行与验收

### 5.1 自动化命令

```bash
cd backend
go test ./internal/outbox ./internal/handlers/search ./routes
go test ./...
```

### 5.2 最小手工验收

1. 启动后端并确保 ES 可用
2. 调 `GET /api/v1/search/sync-status`，确认：
   - `outbox_relay_enabled`
   - `outbox_es_subscriber_enabled`
   - `search_index_mode`（`outbox_es_subscriber` / `local_reconcile` / `manual`）
3. 写入一条会触发 `asset_events` 的业务请求，验证 ES 文档最终可查
4. 关闭 subscriber 后执行 `POST /api/v1/admin/search/reindex`，验证可重建索引

---

## 6. 失败排查最小清单

1. 后端日志是否出现 `outbox relay starting` / `outbox es subscriber starting`
2. `GET /api/v1/search/sync-status` 字段是否与配置一致
3. `asset_events` 中是否存在长期 `publish_state='pending'`
4. `outbox_dlq` 是否持续增长

---

## 7. 结论口径

- 当前主线是 outbox relay/subscriber，不再使用 `outbox_sink_cursors`。
- 测试策略以 `internal/outbox` 单元测试 + 路由/状态接口回归 + 必要手工验收为主。
- 若未来重启 CDC/WAL 入湖方案，应新增独立测试计划，不覆盖本文件主线口径。
