# Tasks — BatchJobDetailPage 加载性能优化 (CYB-2872)

## Context files

- `Frontend/src/pages/BatchJobDetailPage.tsx` — 懒加载触发器 + 早退渲染路径（`if (loading && !job) return <Skeleton/>`）
- `Frontend/src/pages/WorkflowExecutionList.tsx` — 视口懒加载容器
- `backend/internal/handlers/backfill/handler.go` — node-summary handler
- `backend/internal/usecase/backfill/usecase.go` — sync 节流逻辑（`shouldSyncProgress`, `syncProgressMinInterval`）
- `backend/internal/handlers/pipeline/batch_handler.go` — 参考的既有 `batchProgressCache` 模式（89-117 行）
- `backend/internal/usecase/pipeline/usecase.go` — `listDurableRunChildren`（约 4281 行）
- `openspec/changes/CYB-2872-batch-detail-perf/design.md` — callback ref 架构决策，实现前必读

## Implementation

### P0 — node-summary 缓存 + sync 节流

- [ ] [backend] `backend/internal/handlers/backfill/handler.go` 新增 `batchNodeSummaryCache`（参考 `batchProgressCache` 模式，5s TTL，`sync.Mutex` 保护），`GetNodeSummary` 进入时先查缓存
- [ ] [backend] `backend/internal/usecase/backfill/usecase.go:GetBatchNodeSummary` 的 `refreshBatchReadModel` 调用改为只在 `shouldSyncProgress(jobID)` 为 true 时执行
- [ ] [backend] `syncProgressMinInterval` 下调至 10s（与缓存 5s TTL 错峰）

### P1a — AggregateNodeStatusByBatchJobID SQL

- [ ] [backend] 沿用 PR #271 已验证结论：`backfill_items_summary` 物化表不存在 → **跳过本次 SQL 改写**，保持现状（已记录在 `decisions.md`，无需重新验证 schema）

### P1b — 懒加载触发器（callback ref 重写，核心修复点）

- [ ] [frontend] `BatchJobDetailPage.tsx`：把 `nodeOverviewRef`（`useRef` + `useEffect`）改写成 callback ref 模式（见 `design.md` Decision 1），配 `useRef<IntersectionObserver | null>` 存 observer 实例，callback 内部 disconnect-before-create
- [ ] [frontend] `BatchJobDetailPage.tsx`：`subtaskListRef` 同样改写成 callback ref
- [ ] [frontend] 保留 `import.meta.env.TEST` 旁路（测试环境下直接触发 `loadHeavyDetail()` / `setSubtaskListStarted(true)`，不依赖 IO）
- [ ] [frontend] `WorkflowExecutionList.tsx`：确认 `active` prop 的"惰性首次刷新"模式与新 callback ref 触发时机兼容
- [ ] [frontend] 手动验证 StrictMode 双 mount 场景不产生重复 observer / 重复请求（Chrome DevTools Network 面板核对请求次数为 1 次而非 2 次）

### P1c — ListByParentRunID 改 SQL 分页（实测证伪"内存切片够用"，见 decisions.md）

- [ ] [backend] `backend/internal/repository/pipeline_repository.go`：`RunRelationRepository` 接口新增 `ListByParentRunIDPage(ctx, parentRunID string, page, pageSize int) ([]models.RunRelation, int, error)`，保留原 `ListByParentRunID` 不删除
- [ ] [backend] `backend/internal/postgres/pipeline_repo.go`：实现 `ListByParentRunIDPage`——`SELECT COUNT(*) FROM run_relations WHERE parent_run_id=$1` 拿 total；`SELECT ... LIMIT $2 OFFSET $3`，排序保持跟 `ListByParentRunID` 一致（`created_at ASC, child_run_id ASC, relation_type ASC`）；page/pageSize 做边界收敛（page<1→1，pageSize 范围 [1,100]）
- [ ] [backend] `backend/internal/usecase/pipeline/usecase.go:listDurableRunChildren`：改调用 `ListByParentRunIDPage`，删掉手动内存切片逻辑（`start`/`end` 计算）
- [ ] [backend] 更新所有实现 `RunRelationRepository` 接口的 mock（`usecase_test.go` 等）——**同一 commit 内完成**（AI-RULES：新增接口方法必须同 commit 更新全部 mock）
- [ ] [backend] `Total` 字段来自新 `COUNT(*)` 查询（维持既有响应契约不变）
- [ ] [backend] 确认既有 `listDurableRunChildrenReturnsRelationsAndSummary` 测试用例继续通过；补一个"深页（如 page=49）耗时不随全量数据线性增长"的验证（哪怕是手动 curl 验证，不一定要写自动化测试）

## API contract sync

- [ ] 无 OpenAPI 字段变更 — 响应 shape 不变，仅性能改善；PR body 注明「API contract: no change」

## Local verification

- [ ] [backend] `make fmt && make vet`
- [ ] [backend] `go test ./internal/handlers/backfill/... ./internal/usecase/backfill/... ./internal/usecase/pipeline/...`（Tier M）
- [ ] [backend] `go test ./...`（Tier L）
- [ ] [frontend] `npm run lint`（Tier S）
- [ ] [frontend] `npm run test -- --run`（Tier M）—— 注意：确认改动前就存在的 3 个 `batchJobPollIntervalMs` 相关测试失败是否依旧 pre-existing（PR #271 阶段已确认与本次改动无关），不应被新改动放大或掩盖
- [ ] [frontend] `npm run build`（Tier L）

## Deploy verification（mandatory）

- [ ] [backend] `source scripts/dev-backend-env.sh`
- [ ] [backend] `bash deploy/cloudrun/backend-dev.sh`（含 `--service-account`）
- [ ] [backend] curl 验证：第一次 `GET /api/v1/backfill/{id}/node-summary` 正常返回；5s 内第二次请求 < 50ms（缓存命中）
- [ ] [frontend] `bash deploy/cloudrun/frontend-dev.sh`
- [ ] [frontend] **Chrome DevTools MCP**（本次必须真正验证到 IO 触发，不能只看代码）：
  - 打开真实 batch 详情页（用有数据的 batch id）
  - Network 面板确认 mount 时**不**发起 `node-summary` / `children` / `listRuns` 请求
  - 滚动到节点概览卡片区域，确认此时才发起 `node-summary` + `children` 请求
  - 滚动到子任务列表区域，确认此时才发起 `listRuns` 请求
  - 用 patched `IntersectionObserver`（`evaluate_script` + `initScript`）或等效手段确认 observer 确实被创建，不能只凭 Network 面板行为反推
- [ ] [frontend] Performance trace：验证 LCP 相比改动前有实质改善

## OpenSpec checkpoint

- [ ] 等用户「OpenSpec OK，继续」
- [ ] 在 `decisions.md` 记录用户回复
