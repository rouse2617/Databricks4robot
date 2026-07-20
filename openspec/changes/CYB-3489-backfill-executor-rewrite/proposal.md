# CYB-3489 — 重构 backfill executor:提交器 + 投影

> **评审底稿** — 本方案是一起 review 过的可视化文档《批量任务执行:现状梳理与第一性原理重构》的落地版。图/状态机/证据看底稿,落地口径看这里;下文按底稿的小节(§背景 / §01–§10)交叉引用,两者是同一套叙事、同一批证据。
> 底稿:https://claude.ai/code/artifact/3ba66e92-0a5f-44c5-a301-25823c734a6a

## 执行摘要(汇报大纲)

**病根 = 边界错位。** 所有顽疾(孤儿卡死、状态横跳、pool 假死、读超时)同一个根:backend 在一个专业分布式调度器(Argo)之上,又重新发明了一套带租约/认领/回收的"调度器"。

**解法 = 退位。** backend 放弃"执行队列"角色 —— 排队 / 调度 / 并发 / 执行 / 瞬态重试全归 Argo。backend 只留两个轻量纯粹职责:**提交器(Submitter)** 把任务丢给 Argo,**投影器(Projector)** 把 Argo 的真实状态同步回 DB。

**不推倒重来** —— 底层表结构基本不变(已核对 baseline:**零 schema 迁移**),三阶段逐步替换错误的代码模型:

- **P1 收敛状态投影** — 读请求禁止打 Argo(十几秒 → 毫秒);散落 8 处的写状态归一到一条通道(webhook 主 + 后台有界限流 poll 兜底);状态机只前进,靠 DB 原子性防旧盖新。
- **P2 无状态提交器** — 删掉 claim / lease / reaper / 固定线程池;确定性任务名 + K8s 名字冲突(409)自愈防重投;并发泵直接批量提交,不设人工限速,撞基础设施限流才退避。
- **P3 失败与重试矩阵** — 划清"没发出去"与"真跑挂了":瞬态异常 → Argo 自动闭环(backend 不介入,对用户透明);提交未完成 → 提交器下轮重投(视为未提交,非失败);真实业务失败 → Argo 停重试、Projector 标终态 surface 到 UI,交人工手动重试,不自动循环。

## Problem

backfill executor 在 Argo(本身就是队列 + 调度 + 并行控制 + 执行)之上**又自建了一套工作队列**:`ClaimNextItem` + 5 worker goroutine + `reaper` + `lease`(底稿 §02 双重排队,含**八处写状态**的对照表)。这套双重排队是近期一连串 bug 的共同根,底稿 §03 状态机把四个纠缠点画在一张图上:

- **A 振荡**(§03)— item 在 `running ⇄ pending` 之间来回(`reaper` 打回 pending;`syncJobProgress` 的 `mapRunStatusToItem("Pending")="running"` 又推回,还重置 `started_at`),半提交孤儿卡死。根子是**量级错配**:lease 120s 却对应 20min–2h 的任务。
- **B 读路径做重活**(§03)— `GetJob` / `node-summary` 走 `syncJobProgressForce`,一次读最多打 ~100 次同步 Argo 调用;实测 failed=0 秒开(~400ms)、failed=76 要 **12.7s**,严格随 failed 数缩放。
- **C 重复 reconcile**(§03)— "这个 failed 是不是其实 succeeded" 由 ④syncJobProgress + ⑤list refresh + ⑥GetRun + ⑦webhook + ⑧watcher 各答一遍,cap / 触发各不同,既重复打 Argo 又留缝。
- **D pool 死不自恢复**(§03,同事发现)— `runItems` worker 在 `ClaimNextItem` 返 nil 后退出;周期 reconciler 只同步状态不重启 pool;`ResumeIncompleteBatches` 只在 boot 跑一次。**每次部署撞上正在 dispatch 的 batch 就卡** —— dev 已复发 **3 次**:d655b177 / 3b3df404(§04 卡 29min)/ 898cc086(2026-07-15,部署换实例后卡 31min,手动 `resume` 后 67→0 秒清)。

## Root cause

**不是设计过度,是被事故逼的**(底稿 §背景 用 git commit + Linear 还原了时间线):从 CYB-1267 的"每 asset 跑一个 pipeline"fire-and-forget 起步,规模逼出 worker pool + 10k cap(`fba42a50`),CYB-2767 加 per-item 超时,直到 CYB-2774 建起 `claim` + `reaper` 的 DB 持久队列。

自建队列的**唯一硬理由**就是 CYB-2774:Cloud Run 重部署会杀掉内存 goroutine,导致未提交任务丢失(3000 的批次 4 次重部署丢了 614 个)。但 **workflow 一旦提交进 Argo 就已经活过 redeploy**(它在 etcd + controller 里跑)。因此真正需要持久的只是**"哪些 asset 还没提交"**,不是一整套 lease/reaper 执行队列。当年把"持久化提交进度"这个小需求过度解成了 work-queue,且工作单元建模错位(item 的 `running` 横跨整个 workflow 生命周期,而 lease 只有 120s)—— 之后所有 bug 都从这个错配长出来,靠补丁一层层盖(`bf604ad7` 的 dedup+Pending→running 是 A 的源头、`64ef05aa` 的 DOA outbox dispatcher、CYB-3424、CYB-3474/3480)。**重构就锚在底稿 §背景 那句话:持久化的是提交进度,Argo 拥有提交之后的一切。**

## What changes

(底稿 §06 目标图 + §07 现状 vs 目标对照)删掉 backend 自建执行队列。backend 只保留两个职责:

1. **提交器(Submitter)** — 幂等、可续、高并发地确保每个 asset 提交到 Argo。持久化的只是提交进度;撞 429/5xx 被动退避(Argo 顶得住,无预设限速)。
2. **投影器(Projector)** — 单一后台任务把 Argo 真值投影到 DB(webhook 主 + 30s active poll + ~3min drift sweep)。读接口纯读。

Argo 拥有:排队(Pending)、并行(`parallelism`)、执行、瞬态重试(`retryStrategy`)。

## Scope

- 重写 `backend/internal/usecase/backfill` 的执行路径(materialize/dispatch/reaper/sync)。
- 复用 `backfill_jobs` / `backfill_items` / `pipeline_runs` 表(仅语义调整;已核对 baseline **零 schema 迁移**,见 design.md"数据库改动"一节)。
- 复用 CYB-3078 的 webhook + watcher 作为投影主/兜底信号。

## Phases(底稿 §08 三阶段路线;每阶段单独可上 dev 验收)

- **P1(CYB-3490)** 读纯读 + 单一后台 reconciler。消 B/C,取代 CYB-3474。
- **P2(CYB-3491)** 提交器替换工作队列。消 A/D,取代 CYB-3480,删 claim/reaper/lease/pool。
- **P3(CYB-3492)** 重试三类失败(底稿 §09 矩阵,见 design.md)。

**D 止血 · 已决定先上(评审拍板)** — D 正在 dev 每次部署复发。先上一个 ~30 行的"pool 自恢复"临时补丁(周期性重启 worker pool / 周期性 resume 未完成 batch),换取重构期间 dev 不用人肉 `resume`。**独立小 PR + 独立 ticket**(不并进重构 PR),**P2 落地后即废弃删除**。

## 边界条件(评审补充,已并入 design.md)

同事评审提出的四条边界,已核代码后写入 design:

1. **提交幂等** — 确定性 workflow 名(非 `generateName`),键含 job_id:`backfill-{job_id}-{asset_id}-attempt-{N}`;撞 409/AlreadyExists → 视为已提交回填 uid。**顺带修现存隐患**:现命名键是 `(pipeline,asset)` 与去重键 `(job,asset)` 错配,跨 job 会撞 UNIQUE。
2. **drift sweep 有界** — 只扫 `status IN (submitted,running)`,分批(~100)限流,杜绝全表扫打爆 Argo/K8s。
3. **GC 竞态** — TTL(现 30 天)≫ sweep;DB=running 但 Argo 查无 workflow → 报警 + 显式异常态,不无限挂 running。
4. **读路径结构性纯读** — 从 `GetJob`/summary 代码路径摘掉 Argo client DI,防止 sync 逻辑被重新塞回读请求。

## Decisions(底稿 §10 决定纪要,已锁定)

- 重构采纳(未上线,直接建对的,不做一次性 band-aid)。
- 一 asset 一 Workflow CR(现规模够,不上 WorkflowTemplate 参数化)。
- 无预设限速 —— 现规模 Argo 顶得住(§07 实测:parallelism 150 · ~110 pod · 376 wf 无压力),高并发直提,撞 429 才退避。
- 重试三类三 handler;**②未提交 与 ③真失败 严格分开**(§09 关键不变量)。
- poll:webhook 主 + active 30s + ~3min drift,全后台,读不 poll。
- **投影范围**:持续只投 **workflow 级**(phase + `status.progress` N/M,~1000 行);**node 级懒汉** —— 钻取某 wf 才拉(有界+缓存),且**终态归档一次**(webhook 路径,≈现终态行为,无持续放大)。
- **重试粒度 · 步骤级**:手动重试走 Argo `retryWorkflow` 从失败步续跑(复用同名同 uid、保留已成功步),`attempts` 封顶 3;旧 wf 已 GC(>TTL)才回退全新重跑。
- **并发同 asset(跨 job)· 放行(A)**:`algo_run_results` 按 `(asset,algo,version)` upsert,并发后写覆盖(同 algo/version payload 等价、无损坏,仅白算 + latest 短暂 flap)。隐患早已存在、非重构引入;**不加跨-job 编排护栏**(会破坏零迁移/最小基础设施)。复议触发(实测明显浪费/latest 错乱)→ 上写入时单调守卫(C,无迁移)。
- **零 schema 迁移**:已核 baseline(§06 列级映射)。

## Out of scope

- WorkflowTemplate 参数化 / 减少 per-run configmap(现规模不急,记为后续)。
- CYB-2177(unschedulable batch 失败)单独处理。
- 前端列表状态陈旧的展示层(投影修好后自然缓解;若仍有另开)。
- 跨-job 的"每 asset 一个 active run"护栏(并发同 asset 隐患见 Decisions 放行 A;复议触发才上写入时守卫 C)。
- 跨 workflow 的"每步 rollup"实时聚合(workflow 级近实时已够;需要再懒算)。

## Rollout / migration

- 未上线,dev-only 灰度。三阶段分别上 dev 验收(见各阶段验收标准)。
- 每阶段与既有过渡修复(CYB-3474/3480)兼容或自然取代;不与线上流量冲突。
- 后端改动由用户把关合并(Linear → OpenSpec → checkpoint → dev 验证)。
