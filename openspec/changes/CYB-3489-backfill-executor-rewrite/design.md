# Design — backfill executor: 提交器 + 投影

> 对应评审底稿《批量任务执行:现状梳理与第一性原理重构》的 §06(目标图)/ §07(现状 vs 目标)/ §09(重试矩阵)/ §10(锁定决定)。底稿看全景与证据,本文档给落地口径。
> 底稿:https://claude.ai/code/artifact/3ba66e92-0a5f-44c5-a301-25823c734a6a

## 目标架构(底稿 §06)

```
backfill_items (提交账本 + 投影)
      │  submit (幂等/可续/高并发)
      ▼
   Submitter ──────────────► Argo (队列+parallelism+执行+retryStrategy)
                                   │  phase 变
                                   ▼
   Projector ◄── webhook(主) + 30s active poll + ~3min drift sweep
      │  投影
      ▼
backfill_items.status / job counters   ← 读接口只读这里,不碰 Argo
```

单向:items → 提交 → Argo → phase → 投影 → DB。没有 backend 侧执行队列。

## 职责边界

| 组件 | 负责 | 不负责 |
|---|---|---|
| Submitter | 确保每个 asset 有一个 Argo workflow(幂等) | 排队、并行控制、执行、等它跑完 |
| Argo | 排队(Pending)、`parallelism`、执行、瞬态重试 | — |
| Projector | 把 Argo phase 投影成 item/job 状态 | 触发执行、被读接口驱动 |
| 读接口 | 读账本 + 聚合 | reconcile、打 Argo |

这把底稿 §02 的**八处写状态(①ClaimNextItem ②reaper ③executeItem ④syncJobProgress ⑤list refresh ⑥GetRun ⑦webhook ⑧watcher,互相打架)**收敛成三个:Submitter 写 `submitted`;Projector(以 ⑦webhook 为主、⑧poll 兜底)写终态;读接口一个都不写。①②③(claim/reaper/executeItem)整类删除,④⑤⑥ 归并进 Projector 这一条后台路径。

## 数据模型(底稿 §01 四层账本;复用现有表,调整语义)

真相单向上流:**Argo(真相源)→ `pipeline_runs`(每子任务一条:`placeholder` uid 空=未提交 / `real` 有 uid=已提交)→ `backfill_items`(状态)→ `backfill_jobs`(计数,决定 settle)**。`backfill_items.status` 语义收敛为**提交生命周期 + Argo 投影**,严格前向:

| status | 含义 | 谁写 |
|---|---|---|
| `pending` | 未提交(待提交器处理) | materialize / 手动重试 |
| `submitted` | 已提交 Argo(有 uid),等/在跑 | Submitter |
| `completed` | Argo Succeeded | Projector |
| `failed` | Argo Failed/Error(真失败) | Projector |

- 删除 `running`↔`pending` 的回退语义(不再有 reaper 把 running 打回 pending)。
- `pipeline_runs.argo_workflow_uid` 是"是否已提交"的权威标记(Argo 建 workflow 即分配)。
- **不再需要** `lease` / `attempts`(reaper 语义)。保留 `attempts` 仅用于"手动重试次数"(P3)—— 步骤级重试走 Argo `retryWorkflow`,**复用同名同 uid**,不新建 run 行;`attempts` 只做封顶 3 计数。

## 数据库改动:无 schema 迁移(已核对 baseline)

已对 `20260708104125_baseline_from_dev.sql` 核对,**本重构不需要 schema 迁移**:

- `backfill_items.status` / `backfill_jobs.status` 是**纯 `text`**(`DEFAULT 'pending'`),**无 PG enum、无 CHECK 约束**。新增 `submitted` 只是写个新字符串值,不需 `ALTER TYPE` / 改约束。
- **没有 lease / locked_at 列**:reaper 的"lease 120s"是从 `backfill_items.started_at < NOW()-interval` **现算**的(`ResetStaleItems`)。删 reaper 纯删代码,无列可删。
- 幂等要的两把锁**都已存在**:`pipeline_runs.argo_workflow_uid`(`NOT NULL DEFAULT ''`,空=未提交/非空=已提交)+ `pipeline_runs.workflow_name` 上的 **UNIQUE 约束** `pipeline_runs_workflow_name_key`。
- `backfill_items.attempts`(已有列)从"reaper 重试计数"**改语义为"手动重试次数"**(P3),不加列。
- 现有 `idx_backfill_items_status` 足够支撑提交器扫 `pending`、drift sweep 扫 `submitted/running`;是否加复合索引留给压测,不预先迁移。

> 结论:改的是**代码语义**,不是**表结构**。

## Submitter(P2;底稿 §06"提交器该做的")

**核心:确保提交,而非排队执行。**

- 选取 `status='pending'` 的 items,高并发提交(goroutine 池仅用于**提交吞吐**,不是执行队列;可续跑、可周期性重启,不依赖单次 boot)。现状 `BACKFILL_CONCURRENCY=5`(~3 个/秒,1 万个约 55min,§07);目标高并发直提,速率由 Argo `parallelism` 一处控。
- 提交成功(Argo 回 uid)→ `CommitBatchSubtaskDeploy` 原子写 `uid + workflow_name`,item→`submitted`。**uid 落库前不算提交成功。**
- **幂等(评审补充 #1,必须做实)**:
  - workflow 名**确定性生成,不用 Argo `generateName`**。命名键必须含 **job_id**:`backfill-{job_id}-{asset_id}-attempt-{N}`。
  - ⚠️ **修一个现存隐患**:现 `batchSubtaskWorkflowName` 名字键是 `(pipelineName, assetID)`,而去重键是 `(batchJobID, assetID)`(`FindByBatchJobAndAssetID`)—— 两个不同 job 跑同 pipeline 同 asset 会生成**同名** workflow,撞 `pipeline_runs.workflow_name` UNIQUE。P2 把 job_id 纳入名字消除此错配。
  - **两层幂等锁**:① backend 侧 —— 确定性名 + `pipeline_runs.workflow_name` UNIQUE,重复插 run 行直接被 DB 挡;② Argo 侧 —— 提交撞 `AlreadyExists`(gRPC code 6 / HTTP 409)→ **捕获后视为已提交**,查回 uid 落库,item→`submitted`,**不算失败**。(现 `argo/client.go` 提交路径**未特判 409**,是 P2 要补的缺口。)
- **可续**:提交器由周期驱动(不只 boot)——扫 `pending` 且无 uid 的 item 继续提交。**这一条直接消灭 D**(pool 死/部署中断后,下一个周期自动续提)。
- **多实例/并发周期不重复尝试(gemini #2)**:选取 `pending` 用 `SELECT … FOR UPDATE SKIP LOCKED` **有界分批领取**(事务内提交并落 uid)——Cloud Run max=5 实例、周期与 materialize 并发时,不会同一 item 被多方同时提交。注意:这只是**提交吞吐去重**(事务级、秒级),不是执行租约/lease(分钟级、跨 workflow 生命周期)——不复活 reaper 模型;`AlreadyExists` 仍保留为**最后一层**兜底,不作为主并发控制。
- **自动重提不递增 attempts(gemini #3)**:`uid 空` 的周期重提**复用同一确定性名**(attempt 不变)→ 幂等成立;`attempts` **只**由手动重试(P3)递增。若自动重提也 +1,名字会变、幂等即破。
- 撞 429/5xx → 指数退避重试该次提交,不预设全局限速。

## Projector(P1;底稿 §06"投影器该做的")

- **webhook(CYB-3078 exit hook)为主**:workflow phase 变 → 投影 item 状态。
- **30s active poll 兜底**:扫 `submitted`(在跑)的 run,补 webhook 掉的。
- **~3min drift sweep(评审补充 #2,必须有界)**:**只查 DB 里 `status IN ('submitted','running')` 的记录**去跟 Argo 对齐,**绝不全表扫**。上万 running 一次性打 Argo/K8s API 会尖刺甚至 OOM → **分批执行**(每批 ~100,批间让步),整体限流。
- **GC 竞态(评审补充 #3)**:Argo `ttlStrategy.secondsAfterCompletion` 现为 **30 天**(`transpiler.DefaultTTLSecondsAfterCompletion`,env 可调),远大于 sweep 周期,保持"TTL ≫ sweep(至少 1h)"这条不变量即可。**但**若 DB 是 `submitted/running` 而 Argo 里**查不到**对应 workflow(被 GC 清理或外力删除)——**不能无限挂在 running**:投影成一个显式异常终态 + **触发报警**,交人处理(可选:按 ②未提交语义重提)。
- **幂等投影**:Argo phase 是唯一真相,投影只前向覆盖(Succeeded/Failed 是终态,不回退)。
- **投影范围(规模决策:1000 wf × 100 step = 10 万 node)**:只**饿汉投 workflow 级** —— `phase` + 进度 `N/M`(直接取 Argo `wf.Status.Progress`,1 行/workflow,近实时,不从 100 个 node 行反算);**node 级懒汉** —— 用户钻取某个 workflow 才拉它的 100 个 node(有界+缓存),**不全程投 10 万**。这把持续投影量从 10 万降到 1000,并**从持续路径上拿掉 `ReplaceByRunID`(整删整插)**。现 `attachBatchNodeProgress`(从 node 行算)改为取 `status.progress`。跨 workflow 的"每步 rollup"不做实时(需要再懒算)。**node 明细在每个 wf 终态(webhook 路径)归档一次**(≈现终态行为,10 万行一次性、无放大);drill-in:运行中懒拉 Argo、终态读 DB 档(扛过 Argo 30d GC)。node 更新时序见底稿 §06 序列图。
- **读接口纯读(评审补充 #4,结构性保证)**:`GetJob`(`usecase.go:889` 现调 `syncJobProgressForce`)、`GetBatchNodeSummary`、list **不触发投影、不打 Argo**(node 明细的钻取式懒拉是**单 workflow 有界 + 缓存**,不是全批 reconcile)。落地时**从这些读代码路径上摘掉 Argo client 的依赖注入**——用代码结构(而非纪律)保证没人再"顺手"把 sync 塞回读请求。

## 重试矩阵(P3;底稿 §09)

| 类 | 例子 | Handler | 做法 |
|---|---|---|---|
| ① 基础设施/瞬态 | spot 抢占 · node lost · OOM · GCS 卡死 | Argo `retryStrategy` | `limit:2 · retryPolicy:OnError · 指数 backoff`;`OnError` 只重试 Argo 级错误,不重试 app exit≠0;GCS"挂"加 `activeDeadlineSeconds` 让超时变 Error。零 backend 介入。 |
| ② 提交未完成 | submit 失败 / 中途挂 | Submitter re-submit | 幂等重提,不算失败。 |
| ③ 真实执行失败 | Argo 重试完仍 Failed / 坏数据 | surface + 手动 | 投影成 failed;手动"重试失败项"= **Argo `retryWorkflow` 从失败步续跑**(同 workflow、复用同名同 uid、保留已成功步),`attempts` 封顶 3,不自动循环。**旧 workflow 已被 GC(>TTL)→ 回退全新提交**(整轮重跑,复用现 `ForceNewAttempt`)。 |

**关键不变量**:投影必须把 **②(无 workflow / uid 空 = 没提交成功 → 该重提)** 与 **③(有 workflow 且 Argo=Failed = 真失败 → surface)** 严格分开。历史 bug 正是把 placeholder(②)当成 failed(③)。

## 投影一致性与重试竞态(硬化规则)

评审压出四个竞态/死锁。**①③④ 由同一原语解决:item 带 `attempts` 作版本号 + 投影按版本单调写入。**

- **① 终态锁死(retry 死锁,🔴 严重)** — `persistRunObservation`(`usecase.go:2117`)有硬规则:definitively-failed 被 active 观测覆盖时直接丢弃。`retryWorkflow` 复用同 uid,workflow 变回 Running 会被这条防线吞掉 → DB 永卡 `failed`,废掉步骤级重试。**解**:手动重试在同一事务 `attempts+1`;状态投影接受条件加 `OR incoming_attempt > stored_attempt`,让新一轮 Running 冲破终态锁,同时拒掉上一轮 stale 观测。
- **② 归档丢失(webhook 丢包)** — node 快照只由 webhook 归档;丢包时 poll/sweep 只同步 workflow 级、标 completed,漏归档 → 30d GC 后无据。**解**:归档绑「谁先看到终态谁归档」(webhook / poll / sweep 任一);**写终态前若本地无快照 → 先拉 Argo 补档(幂等)**;UI 钻取发现终态无快照 → 兜底补写。
- **③ 进度回退(乱序 / 多实例)** — max=5 实例乱序写 → 进度条闪。**解**:按 attempt 分层单调守卫 `WHERE incoming_attempt > stored OR (= AND incoming_numerator > stored_numerator)`。**不用 plain `GREATEST`** —— GC 全新重跑会把进度真重置到 0/100,`GREATEST` 会错保旧值。
- **④ 重试狂热(mutation 非幂等)** — `retryWorkflow` 是突变;对已 Running 重复调用 Argo 返 400。**解(无 Redis)**:重试走 DB 条件更新 `UPDATE ... SET status='running', attempts=attempts+1 WHERE id=? AND status='failed'` —— 影响 1 行才调 Argo,0 行 →「已在续跑中,刷新」。显式捕获 Argo 400「not in failed state」当良性,不当基础设施失败。(真需跨进程互斥用现有 `pg_advisory_xact_lock`,不引 Redis。)

**统一原语**:手动重试 = 原子条件更新(抢重试权 ④ + 抬版本破终态锁 ① + 重置进度基线 ③);所有投影按 `(attempts, progress_numerator)` 单调写入。

**迁移代价**:`attempts` 已有列(0 迁移)。持久化 workflow `progress` 的 numerator 守卫**可能加 1 个可空列**(additive / 安全,非 enum/约束那种);或 progress 仅活跃期临时算(守卫弱化到单实例)。这是核心"零迁移"之外唯一可能的小加列,如实标注。

## 分阶段迁移(每步可独立上 dev)

- **P1**:把 `syncJobProgress` 的 reconcile(含 CYB-3474 的 failed-pass)从读路径抽出,合并进一条后台 reconciler(webhook + 30s + 3min)。`GetJob/node-summary` 改为纯读。**不动提交队列**,风险最小,先解 dev 卡顿。
- **P2**:提交器上线,`materialize` 后由周期提交器接管;删 `ClaimNextItem`/`ResetStaleItems`/`lease`/`runItems` pool。item 状态机切到 `pending→submitted→terminal`。
- **P3**:workflow 模板加 `retryStrategy` + `activeDeadlineSeconds`;手动重试走 `retryWorkflow` 从失败步续跑(`attempts` 仅作封顶计数;旧 wf 已 GC 才回退全新提交)。

## Alternatives considered

- **保留自建队列 + 补 D(pool 自恢复,~30 行)**:能止血,但不动错配模型,C/A 仍在,继续攒补丁。仅作为重构未落地前的可选临时项(底稿 §08 待定,待用户拍板)。
- **复活 phase-4 outbox dispatcher**:那是"更好的自建队列",仍是在 Argo 之上再排一次;且 64ef05aa 已证 DOA。被本方案(直接靠 Argo)取代。
- **WorkflowTemplate + 参数化(少建 CR/configmap)**:正交优化,现规模不需要,记为后续。

## Risks

- webhook 丢失 → 靠 30s/3min poll 兜底;需确认 poll 周期 < workflow TTL(现 30 天,充裕)。
- 提交器周期与 materialize 初次提交并发 → 幂等 + `already exists` 处理保证不双提交。
- 删 reaper 后,"提交到一半 uid 未落库"的窗口 → 提交器续跑时按 `uid 空` 重提,`already exists` 回填,安全。
- **并发同 asset(跨 job)· 决定放行(A)**:`algo_run_results` 按 `(asset_id, algo_key, version)` upsert(`backfill_result_repo.go:54`),两个 job 并发同 asset 同 algo/version → **后写覆盖**(payload 等价、无损坏)+ 白算一遍 + `asset_algo_latest` 短暂 flap。此隐患**早已存在、非重构引入**,且同 algo/version 下基本无害。**不加**跨-job"每 asset 一个 active run"护栏(需部分唯一索引=破坏零迁移,或 advisory lock=跨 job 耦合,为不常见低危场景提前造基础设施,违背最小化)。**复议触发**:实测明显重复算力浪费或 latest 错乱 → 届时优先上 **C(写入时单调守卫**:`UpsertAlgoRunResult` 仅当来者 `finished_at`/run 更晚才覆盖,localized、无迁移**)**,而非编排层护栏。
