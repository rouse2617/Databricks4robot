# Decisions — CYB-2872 BatchJobDetailPage 加载性能优化

## 2026-07-01 — PR #271 关闭重做：P1b 懒加载竞态

- **Context**: PR #271 实现了 P0/P1a(跳过)/P1b/P1c。审查 + Chrome DevTools MCP 实测发现
  两处 P1b 问题：(1) `nodeOverviewRef` 声明了但从未 `ref={}` 绑定到 JSX 节点；(2) 更深层，
  即使补上绑定，`BatchJobDetailPage.tsx` 顶部 `if (loading && !job) return <Skeleton/>`
  会在 `job` 未就绪时提前返回，此时 ref 容器不在 DOM 树里，`ref.current` 为 `null`；
  `useEffect(() => {...}, [loadHeavyDetail])` 只执行这一次就早退，`job` 到位后重新渲染出
  真实容器时 `loadHeavyDetail` 引用未变，effect 不会重跑，`IntersectionObserver` 永远不会
  被创建。用 patched `IntersectionObserver` + Chrome DevTools MCP 在真实 batch 页面上验证：
  mount 后等待 8s+，从未创建过任何 observer 实例。
- **Decision**: 关闭 PR #271；P0/P1a/P1c 方案原样保留（backend 代码结构无问题）；P1b 改用
  **callback ref**（`ref={(el) => {...}}`，配 `useCallback` 记忆化）替代 `useRef` + 独立
  `useEffect`。callback ref 在 DOM 节点真正挂载的那次渲染就会被 React 调用，不受组件早退
  渲染路径影响。
- **Alternatives**:
  - 保留 `useRef` + `useEffect`，改依赖数组把 `job` 加进去强制重跑：可行但脆弱，任何未来在
    `job` 变化前插入的早退分支都会重新触发同类 bug；治标不治本
  - 用 `react-intersection-observer` 等第三方库：引入新依赖，callback ref 原生方案已够用
- **Rationale**: callback ref 是 React 官方推荐处理"DOM 节点可能在稍后渲染才出现"场景的
  模式，没有额外依赖，代码量相近，且从根上消除这类竞态

## 2026-07-01 — 范围选择

- **Context**: 用户报告 batch 详情页打开慢，分析定位到 4 个优化项（P0/P1a/P1b/P1c）
- **Decision**: 一次性纳入本次变更，全部实现 + 验证
- **Alternatives**:
  - 仅 P0：保守，单接口 P95 砍半，但 `WorkflowExecutionList` mount 即 fetch 与 `ListByParentRunID` 全量拉仍存在
  - P0 + P1b：覆盖「单接口」+「视口守卫」，但 SQL 层与分页层仍未触及
- **Rationale**: 四个改动相互独立（无共享代码路径），合在一个 PR 减少 review 成本；P0 是高 ROI，P1b/P1c 风险低，P1a 是可选 SQL 优化（依赖 schema）

## 2026-07-01 — 缓存模式参考

- **Context**: 既无 `batchProgressCache` 在 `backend/internal/handlers/pipeline/batch_handler.go:89-117` 是同 handler 的 in-memory 5s TTL 模式
- **Decision**: node-summary 新缓存复用该模式（同结构、同 TTL、同步锁实现）
- **Rationale**: 减少认知成本 + 与同模块缓存策略保持一致

## 2026-07-01 — sync 节流 TTL

- **Context**: 既有常量 `syncProgressMinInterval = 30 * time.Second`（`usecase.go:1294`）定义但未在 `GetBatchNodeSummary` 路径使用
- **Decision**: 复用并下调为 10s，与缓存 5s TTL 错峰运行（缓存 5s 重新计算，sync 每 10s 触发一次为下个 5s 缓存预热）
- **Rationale**: 既保留 sync 防止 staleness，又避免每次请求都跑 sync

## 2026-07-01 — listDurableRunChildren：改为 SQL 分页（修正此前"内存切片"决策）

- **Context**: 此前决策记录（下方保留存档）选择"内存切片、保持 SQL 不变"，理由是"内存切片成本可忽略"。
  重做本 issue 时用 dev 上真实 1000 子任务 batch (`0878ceb8-...`) 实测证伪：`page=1` 686ms，
  `page=49`（最后一页，offset=960）901ms——**耗时量级跟请求哪一页无关**，说明
  `ListByParentRunID(ctx, parentRunID)` 每次都从 Postgres 拉全部 1000 条 relation，内存切片
  只是省了 `FindByID` 的调用次数，没有省 DB 全量拉取本身的成本
- **Decision**: 改为 SQL push-down —— 新增 repository 方法 `ListByParentRunIDPage(ctx, parentRunID, page, pageSize) ([]RunRelation, total int, error)`，Postgres 实现用 `COUNT(*)` + `SELECT ... LIMIT $2 OFFSET $3`（排序保持跟原 `ListByParentRunID` 一致：`created_at ASC, child_run_id ASC, relation_type ASC`，确保翻页时结果稳定）
- **Alternatives**: 维持内存切片（已被实测证伪，成本没有被真正节省）
- **Rationale**: 实测数据说话，内存切片方案没有达到 P1c 想要的效果；SQL push-down 改动范围可控（repository interface + Postgres 实现 + mock + usecase 调用点，无需 migration）
- **Follow-up**: PR #271（已关闭）已经实现过这个方案，代码结构验证过可用，本次直接复用其 SQL 实现（`backend/internal/postgres/pipeline_repo.go:ListByParentRunIDPage`）

<details>
<summary>此前决策存档（已被证伪，保留供追溯）</summary>

### 2026-06-30 — listDurableRunChildren 内存分页而非 SQL 分页（已废弃）

- **Context**: P1c 要么 push page/limit 到 `ListByParentRunID` SQL，要么拉到内存后切片
- **Decision**: 内存切片（保持 SQL 不变）
- **Alternatives**: SQL push-down（需要改 repository interface、改 mock、改测试，影响面大）
- **Rationale**: 当前批次典型 1k~3k relations，内存切片成本可忽略；后续若 relation 量级到 10k+ 再 push-down

</details>

## 2026-07-01 — P1a SQL 改写条件性

- **Context**: 改 `AggregateNodeStatusByBatchJobID` 假设存在 `backfill_items_summary` 物化表
- **Decision**: 先查 schema，若不存在则跳过本次 SQL 改写、保留原 JOIN；记录到 decisions
- **Rationale**: 避免引入 migration 扩散本次 PR 范围

## 2026-07-01 — P1a 实施结果：跳过 SQL 改写

- **Verified**: `ls backend/migrations/ | grep summary` → 无；`grep backfill_items_summary backend/internal/{models,postgres}/*.go` → 无
- **Decision**: 跳过 P1a SQL 改写；`AggregateNodeStatusByBatchJobID` 保持原 `DISTINCT ON` 子查询
- **Rationale**: 本次不引入新 migration；5s 缓存已把该接口 P95 砍到 ~50ms 量级（10x 改善），SQL 改写边际收益 < cache 收益，且需要单独设计 schema + migration，超出本 PR 范围
- **Follow-up**: 另开 issue 评估「`backfill_item_latest` 物化表」+ partial index `(job_id, asset_id, status)`

## 2026-07-01 — 用户确认

- **Context**: 用户在本次变更开始时确认范围为 P0 + P1a + P1b + P1c
- **Decision**: 按此范围实施
- **Rationale**: 用户明确选择

## 2026-07-01 — 用户确认 OpenSpec（重做）

- **Context**: 重做版 proposal.md/tasks.md/design.md 完成，含 P1c 决策修正
- **Decision**: 用户回复「ok，你来吧」，确认按当前 tasks.md 开始实现
- **Rationale**: 用户明确批准
