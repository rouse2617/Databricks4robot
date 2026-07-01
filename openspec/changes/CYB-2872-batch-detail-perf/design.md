# Design — CYB-2872

## Architecture Context

- **Constraints**: React 18（StrictMode 双 effect 调用）；`import.meta.env.TEST` 是既有
  测试兼容旁路，必须保留；不引入新 npm 依赖
- **Goals**: 懒加载触发器在组件任意渲染路径下都必须正确工作，不依赖"首次渲染时容器已存在"
  这个假设
- **Non-Goals**: 不重构 `BatchJobDetailPage.tsx` 的整体 loading/error 早退结构；不改
  `WorkflowExecutionList` 之外的懒加载模式（如果既有其它页面用同款 `useRef`+`useEffect`
  懒加载模式且工作正常，说明它们的早退时机跟这里不同，本次不牵连排查）

## Affected Modules

- `Frontend/src/pages/BatchJobDetailPage.tsx` — 节点概览 + 子任务列表两处懒加载触发器
- `Frontend/src/pages/WorkflowExecutionList.tsx` — 视口懒加载容器（P1b 新增）
- `backend/internal/handlers/backfill/handler.go` — node-summary 缓存（P0，复用 PR #271 方案）
- `backend/internal/usecase/backfill/usecase.go` — sync 节流（P0，复用 PR #271 方案）
- `backend/internal/usecase/pipeline/usecase.go` — `listDurableRunChildren` 内存分页（P1c，复用 PR #271 方案）

## Architecture Decisions

### Decision 1: 懒加载触发器用 callback ref，不用 `useRef` + `useEffect`

- **Approach**: 用 `useCallback` 包一个 ref 回调函数，在回调内部直接创建/清理
  `IntersectionObserver`；JSX 里 `ref={callbackRef}` 代替 `ref={someRef}`

  ```
  const nodeOverviewObserverRef = useRef<IntersectionObserver | null>(null);
  const nodeOverviewCallbackRef = useCallback((el: HTMLDivElement | null) => {
    nodeOverviewObserverRef.current?.disconnect();
    nodeOverviewObserverRef.current = null;
    if (!el) return;
    if (import.meta.env.TEST) { void loadHeavyDetail(); return; }
    const observer = new IntersectionObserver((entries) => {
      for (const entry of entries) {
        if (entry.isIntersecting) { void loadHeavyDetail(); observer.disconnect(); break; }
      }
    }, { threshold: 0.1 });
    observer.observe(el);
    nodeOverviewObserverRef.current = observer;
  }, [loadHeavyDetail]);
  ```

- **Alternative**: 保留 `useRef` + `useEffect`，把 `job`（或某个"容器已渲染"信号）加进依赖
  数组，强制在 `job` 到位后重跑一次
- **Rationale**: callback ref 由 React 在**每次该 DOM 节点被挂载/卸载时**调用，语义上就是
  "节点存在了，做点什么"，不管这发生在第 1 次渲染还是第 N 次渲染。用 `useEffect` 模拟同样
  效果需要精确知道"容器渲染依赖哪些状态"并把它们全部塞进依赖数组——这个信息随着组件后续被
  改动很容易腐化（比如以后有人在 `job` 判断之外又加一个早退分支，同类 bug 会复发）。
  callback ref 从机制上就不会有这个问题。
- **Trade-off**: callback ref 每次组件重渲染、如果 ref prop 本身引用变化（这里用
  `useCallback` 记忆化，只在 `loadHeavyDetail` 变化时才变，等同于旧 `useEffect` 版本的依赖
  行为），所以实际重建 observer 的频率跟旧方案一致，没有性能倒退
- **Risk**: React 18 StrictMode 下开发环境会 mount → unmount → mount 两次，callback ref
  会被调用 `el → null → el`；已在实现里对 `nodeOverviewObserverRef` 做 disconnect-before-
  create 处理，避免重复 observer 泄漏
- **Rollback**: 纯前端改动，无 API/数据变更，revert commit 即可回滚

### Decision 2: P1a（`AggregateNodeStatusByBatchJobID` SQL 改写）继续跳过

- **Approach**: 沿用 PR #271 `decisions.md` 里已验证的结论——`backfill_items_summary`
  物化表不存在，不引入新 migration，本次不做 SQL 改写
- **Alternative**: 新增物化表 + migration
- **Rationale**: 5s cache（P0）已经把该接口 P95 打到 ~50ms 量级，SQL 改写边际收益低于
  cache 收益，且需要独立评估 migration，超出本次范围
- **Rollback**: N/A（未改动）

## Data Flow

```
组件 mount
  │
  ├─ job === null → return <Skeleton/>          ← 早退分支，JSX 树里没有任何 ref 容器
  │
  ▼ (loadBatchJob 异步 resolve, job 更新)
组件重渲染
  │
  ├─ job !== null → 渲染完整 JSX
  │     │
  │     ├─ <div ref={nodeOverviewCallbackRef}>   ← callback ref 在这次渲染被 React 调用（el 非空）
  │     │     → 立即创建 IntersectionObserver，附着在真实 DOM 节点上
  │     │
  │     └─ <div ref={subtaskListCallbackRef}>    ← 同上
  │
  ▼ 用户滚动到视口
IntersectionObserver 回调触发 → loadHeavyDetail() / setSubtaskListStarted(true)
```

## Risks / Trade-offs

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| StrictMode 双 mount 导致 callback ref 被调用两轮 | 可能重复创建 observer | disconnect-before-create，配合 ref 存 observer 实例 |
| callback ref 写法比 `useRef`+`useEffect` 稍不直观 | 后续维护者不熟悉这个模式 | `design.md` + 代码注释说明原因（why，不是 what）|
| 5s node-summary 缓存窗口内状态字段可能滞后 | completed/failed 状态短暂不准 | 与既有 `GetBatchStatus` 5s 缓存策略一致，可接受（继承自 PR #271 decisions.md）|
