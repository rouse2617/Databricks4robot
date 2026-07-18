# Tasks

> 对应评审底稿 §08 三阶段路线。底稿:https://claude.ai/code/artifact/3ba66e92-0a5f-44c5-a301-25823c734a6a

## P0 — D 止血(独立小 PR,重构期间用,P2 后删)✅ 已落地
- [x] 周期性 resume:`StartPoolRecovery/StopPoolRecovery` 60s ticker 重跑 `ResumeIncompleteBatches`(PR #409,dev `fa463e56`)
- [x] 独立 PR 落 dev;幂等(pool 活着则 claim 续走,死了则新 spawn 顶替)
- [x] 代码已打标 "Deletes in P2 / DELETE in CYB-3489 P2"
- 备注:P0 单测在 CI 挂死已移除(PR #410,`e5f0a51f`)——P0 无单测覆盖,验收依赖 dev 实际行为;**仅在 dev 分支,prod(v1.1.0)没有**。P2 删除时连 Start/Stop 两个入口一起清。

## P1 — 读纯读 + 单一后台 reconciler (CYB-3490)
- [x] `GetJob` / `GetBatchNodeSummary` / `ListNodeFailures` / `ListRunSummaries` / `ListBatchAssetRuns` / batch children 全部改纯读(PR #411)
- [~] **结构性剥离(评审 #4)**:P1a 先以单测锁死(读路径 0 Argo 调用断言 + Argo 调用即 t.Fatal 的 mock);完整 DI 拆分随 P1b
- [x] 读时 reconcile 全删(`SyncBatchView`/`refreshBatchReadModel`/`refreshRunSummariesForList`);missing-run repair 挪进 60s reconciler;CYB-3474 failed-pass 仅后台触发(PR #411)
- [x] 后台:webhook 主(不变)+ 30s watcher 加**轮转游标**防饿死(= drift sweep 的落地形态,PR #411)
- [ ] **drift sweep 有界(评审 #2)**:只扫 `status IN (submitted,running)`,分批 ~100 + 限流,禁全表扫
- [ ] **GC 竞态(评审 #3)**:DB=running 但 Argo 查无 workflow → 报警 + 显式异常态,不挂死;校验 TTL(30d)≫ sweep
- [x] **投影范围(规模决策)**:workflow 级 `progress` 列(PR #412)——持续路径删掉 `ReplaceByRunID` 整删整插,`projectRunNodes` 终态归档一次(active→terminal 跃迁 + 缺档补写);GetRun 钻取保留 live;batch 列表 NodeProgress 用 `run.progress` 兜底
- [x] 迁移幂等化 + 事故复盘:CI schema paths-filter 误判 no-op 旧镜像 → dev 500;手动 apply + `IF NOT EXISTS`(PR #413);CI filter bug 另行修复(并行任务)
- [ ] **投影单调守卫(硬化 ①③)**:状态接受条件加 `OR incoming_attempt > stored_attempt`(冲破 `persistRunObservation:2117` 终态锁 + 拒 stale);进度守卫按 `(attempt, numerator)` 字典序单调(**非 plain GREATEST**)。单测:retry 后 Running 不被吞;乱序/跨 attempt 不闪
- [ ] **归档鲁棒(硬化 ②)**:终态归档绑 webhook/poll/sweep **任一**先看到终态者;写终态前本地无快照 → 先拉 Argo 补档(幂等);UI 钻取发现终态无快照 → 兜底补写。单测:丢 webhook 仍归档
- [ ] 单测:读接口不产生 Argo 调用;后台仍能收敛误判 failed;sweep 分批边界;GC-missing 触发异常态
- [ ] dev 验收:failed=76 batch 打开 node-summary < 1s;日志无 read-path Argo 调用

## P2 — 提交器替换工作队列 (CYB-3491)✅ 已落地(#414 词表先行 · #416 主体 · #417 语义修正)
- [x] 主体全部落地并 dev 实证:37 条冤案平反、真失败 3 条留案正确(job 9dcba916)
- [x] **语义修正(评审原则,#417)**:调度等待不是失败 —— 删 unschedulable 判死、历史判决可平反、读侧 BlockingReason 展示;镜像类确定性配置错误仍判死


- [ ] 新增周期性 Submitter:扫 `pending` 且 uid 空的 item,高并发提交,`already exists` 回填 uid
- [ ] item 状态机切 `pending→submitted→completed/failed`(删 running 回退)
- [ ] 删除 `ClaimNextItem` / `ResetStaleItems`(reaper)/ `lease` / `runItems` worker-pool
- [ ] **幂等(评审 #1)**:确定性 workflow 名(非 generateName),键含 job_id `backfill-{job_id}-{asset_id}-attempt-{N}`;**修 `batchSubtaskWorkflowName` 命名键 `(pipeline,asset)` 与去重键 `(job,asset)` 错配**;`argo/client.go` 提交路径补 409/AlreadyExists 特判 → 回填 uid 视为已提交
- [ ] uid-空判据 + `pipeline_runs.workflow_name` UNIQUE 双层锁;可续:周期驱动,不只 boot
- [ ] 单测:提交幂等、可续、并发不双提交;投影区分 ②未提交/③真失败
- [ ] dev 验收:部署中断/重部署后 batch 自动续提到 100%(不再人肉 resume)

## P3 — 重试三类失败 (CYB-3492)
- [ ] workflow 模板加 `retryStrategy{limit:2,retryPolicy:OnError,backoff}` + step `activeDeadlineSeconds`
- [ ] 提交器 re-submit 覆盖 ②(不算失败)
- [ ] 手动"重试失败项"= **Argo `retryWorkflow` 从失败步续跑**(复用同名同 uid、保留已成功步);批量 `RetryFailed` 从 `ForceNewAttempt` 改指到 `RetryWorkflow` 通道;`attempts` 封顶 3;旧 wf 已 GC → 回退全新提交
- [ ] **重试原子去重(硬化 ①④,无 Redis)**:重试走 `UPDATE ... SET status='running', attempts=attempts+1 WHERE id=? AND status='failed'` —— 影响 1 行才调 Argo,0 行 →「已在续跑中」;显式捕获 Argo 400「not in failed state」当良性提示(需跨进程互斥用现有 `pg_advisory_xact_lock`,不引 Redis)
- [ ] 单测/验收:瞬态自动重试;确定性失败 surface 不循环;真失败 vs 未提交分开计数

## 跨阶段
- [ ] 每阶段单独 PR + dev 验证(deploy-before-commit gate)
- [ ] 与 CYB-3474/3480(过渡止血)兼容;P1 取代 3474,P2 取代 3480,合并后清理其残留
- [ ] 并发同 asset(跨 job)决定**放行 A**,不建护栏;仅在 design/spec 记录 last-writer-wins 语义 + 复议触发(实测浪费/latest 错乱 → 上写入时单调守卫 C)
