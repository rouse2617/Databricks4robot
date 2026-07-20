# Ralph Journal

## LOG (iter 31, inline PR#477 完整闭环)
- **阶段**: IDLE → CODING → AWAIT_CI → MERGED → DONE_ITEM(20 分钟内一气跑完)
- **变更**: `chore/CYB-3386-archive-openspec-kpi-row` 分支勾齐 tasks.md + 加"归档"状态说明,PR#477 → dev。
- **证据**:
  - push 通过 pre-push: `CI_LOCAL_BASE=origin/dev SKIP=terraform_validate,terraform_tflint,terraform_fmt` 组合成功(commitlint 只扫 dev..HEAD 一条本 commit,pass)
  - PR#477 checks 2/2 SUCCESS (preview + gitleaks),path-filter 跳过 backend-vet/openspec-gate
  - `gh pr merge 477 --squash --delete-branch` → MERGED @ b0848563 @ 2026-07-18T14:17:42Z
  - deploy-dev 未触发(path-filter,纯 tasks.md 文档),契约无变化,免 VERIFY(同 #403 模式)
- **教训**:
  - **L-M15 (CRITICAL)**: Ralph CODING 阶段 push 必须 `CI_LOCAL_BASE=origin/dev` + `SKIP=terraform_validate,terraform_tflint,terraform_fmt`。缺前者 commitlint 会扫 origin/main..HEAD(dev 历史含大量 legacy 非合规提交)导致所有新分支 push 失败。iter30 subagent 漏此 env 归因为"仓库基线 bug",实为 subagent 遗漏。
  - **L-M16**: 元项 vs 业务项分类 (L-M4) 补充:openspec/changes/**/tasks.md 的归档也是**业务项**(改仓库源码),必须走 PR;不同于 `.agent/loop/*` 元项无 PR。
- **backlog 变更**: +B012 DONE(PR#477 归档 CYB-3386 openspec);done_count 16 → 18(补记 M005 一起,之前 iter23 就已 DONE)。
- **下一步**: 循环回 IDLE,streak 归零;下轮可继续从 openspec-ghost-report 里挑另一个幽灵开 PR(每 PR 一个 change,单原子),或响应新 signals。

## META-REFLECT #3 (iter=23, done=15)
**观察 iter 1-22**:
- 有效模式:(1) 每轮首跑 discover-signals.sh 五段扫描 → 精确定位事件 vs 盲扫子系统 (2) 元项(改 .agent/loop/)零 PR 摩擦,快速闭环 done_count (3) `gh pr view --json` 多字段一次校准替代人肉猜测
- 反模式:(1) 静态扫 TODO 已扫穿 iter 1-22,主线码干净,继续扫是空转 (2) 期待自己 approve 通过分支保护 → 仓库未启 auto-merge,`gh pr merge --auto` 直接报错;需 human 介入(iter 22.6 佐证)
**下阶段动作**:
- A. 严格执行 RALPH-STRATEGY.md v2 DISCOVER 顺序,连续 3 轮段 1-5 全空 → 进入 T1-T7 baseline VERIFY 循环
- B. 推进 M004 claude-in-chrome:T5/T6 UI 回归空洞是当前最大盲区,补上后 backend 契约破坏能被抓到

## iter 28 (2026-07-18) IDLE → IDLE (cloudrun deep scan degrade)
- 尝试 3h Cloud Run backend-dev 日志聚类(WARN top/5xx/slow>3s)。首次 gcloud crashed ConnectTimeout(logging.googleapis.com 300s);重试 1 次亦无输出。按 L-M13 组合已 unset 代理,仍失败 → 判定 local-net-glitch 二次触发,非 backend 故障。
- 不进 backlog(无观察证据);empty_streak 保持 3。iter 27→28。

## LESSONS (新增)
- **L-M14** (iter28): gcloud logging read 对 logging.googleapis.com 偶发 ConnectTimeout=300s,即使已 unset 代理。判定为本地网络抖动,单次重试仍失败即 park 该采样,不作事件响应;下轮改试轻量 `gcloud run services describe`(REST 通路不同)或 curl backend /healthz 佐证服务态。

## iter 27 (2026-07-18) IDLE → IDLE
- HEAD=22ad2c8d 未变。signals: seg1 review-req=0, seg2 dev CI 5 latest run 全 success, seg3 my-open-PR=0。EMPTY_DISCOVER_STREAK 2→3。
- 触发 META-REFLECT #3 阶段动作 A 的门槛(连续 3 轮空):下一轮切入 baseline VERIFY(T1-T7 复跑)或激活 M004 claude-in-chrome 探路。
- 本轮零码零 PR,仅状态推进。

## LESSONS (通用规则,附加)
- **L-M11**: 仓库分支保护 + 无 auto-merge 时,Ralph 单人 PR 无法自动 merged;完成 CI 绿后应主动 park 并 DEFERRED-HUMAN,不再空转。
- **L-M12** (iter25): 本地 clash 代理 (127.0.0.1:7890) 对 backend host SSL_ERROR_SYSCALL 但对 frontend host 正常 = 代理侧节点/规则问题,不是 backend down。baseline 前若 T1 000/SSL_SYSCALL 且非代理 unset 后 DNS fail → 判定 local-net-glitch,推迟 baseline,不作 P0 事件响应。
- **L-M13** (iter26): baseline #3 green @2026-07-18 — T1=200 / T2 tot=10000 st=paused / T3 len=1 attempted=9836 keys_ok=true / T7 404 WORKFLOW_NOT_FOUND。unset {http,https,all}_proxy 后 curl 正常;subagent 环境需在每轮 baseline 前显式 unset,已固化为 L-M12+L-M13 组合。

## LOG (iter 26, IDLE stable, baseline #3 green)
- baseline #3 全 4 契约点通过,与 iter14 baseline #2 一致 → dev 主线无回归。
- signals: PRs=0 / stalled=0 / recent CI failures=5(均 iter22 前旧 case,已知不阻断) / cloudrun 段超时(非契约信号,跳过)。
- backlog TODO 仍仅 M004(等 Chrome 扩展),连续 IDLE 保持。

## LOG (iter 25, IDLE stable, baseline #3 attempt aborted)
- **阶段**: IDLE → IDLE。git fetch origin/dev HEAD 未推进 (22ad2c8d)。signals 5 段=0/5(全为已知历史 CI)/0/0/0。
- **baseline #3**: T1 frontend=200 OK;backend T1-T4/T7 全部 curl(35) LibreSSL SSL_ERROR_SYSCALL,retry 5 次不通;unset proxy 后 DNS 解析失败(内网需代理)→ 判定为**本地代理侧节点故障**,非 backend 故障(与上次 iter 20 baseline 全绿 + 期间多次 dev 部署成功交叉印证)。
- **结论**: baseline #3 未采集完成,推迟至下轮代理恢复。记 L-M12 教训供后续 Ralph 快速判定。

## LOG (iter 23, DONE M005 + META-REFLECT #3)
- **阶段**: IDLE → DONE_ITEM (M005)。落地 `.agent/loop/RALPH-STRATEGY.md`(v2 DISCOVER 顺序 + auto-merge 缺失应对);regression.md 顶部加策略指针;backlog M005 → DONE。
- **证据**: signals 5 段 counts=0/5/0/0/0 dev 稳定;strategy.md 32 行落盘;META #3 3 有效模式 + 2 反模式 + 2 下阶段动作 + L-M11。
- **backlog 变更**: M005 DONE,新增 B011(dev 07-17 08:41 deploy-dev CI failure 历史事件)。done_count 14→15。

## LOG (iter 22.6, manual batch merge)
- **阶段**: park → 4 x MERGED to dev,每个 deploy 后 verify
- **证据**:
  - #419 (base fix main→dev) merged @ a97f25e0 → deploy-dev success → T1=200 T2 tot=10000 T3 keys ok T7=404 WORKFLOW_NOT_FOUND
  - #476 (backend eol fix) merged @ b4ac19c9 → deploy-dev success → T1-T3+T7 全绿
  - #403 (openspec docs only) merged @ 56dcd11c → path-filter skip deploy(纯 docs)→ 无契约变化不测
  - #343 (frontend 14 files) merged @ 22ad2c8d → deploy-dev success → T1-T3+T7 全绿
- **教训**:
  - **L-M9**: nyx PR merge 后要看 path-filter 是否触发 deploy — 纯 docs/openspec PR 会 skip deploy-dev,不是失败;不用重跑 verify。
  - **L-M10**: `gh pr view <N> --json mergeable,mergeStateStatus` 在刚 push/merge 后可能返回 UNKNOWN,需 sleep 15-30s 让 GitHub 重算。
- **backlog 变更**: B005/B008/B010 全部 → DONE(4 PR 已 merge)。done_count 11→14。
- **下一步**: NEXT_META_REFLECT_AT=15,再一轮 DONE 就触发 META #3。M004(claude-in-chrome for T5)仍是最大空洞。

## LOG (iter 22.5, manual correction by user)
- **阶段**: park → MERGED (#419 base fix + squash to dev)
- **证据**: 用户指出 PR#419 base=main 违反红线。`gh pr edit 419 --base dev` 后 7/7 SUCCESS + reviewDecision="" (dev 不强制 review) + mergeStateStatus=CLEAN。`gh pr merge 419 --squash --delete-branch` → MERGED, squash=a97f25e09b3b @ 2026-07-18T03:46:19Z。
- **教训**:
  - **L-M7 (CRITICAL)**: Ralph 接手已存在的 PR 前,必须先 `gh pr view <N> --json baseRefName` 校验 base。base≠dev → 立即 `gh pr edit --base dev` 或标 DEFERRED-HUMAN。禁在 base=main 的 PR 上加 commit(违反"不碰 main")。
  - **L-M8**: dev 分支保护不强制 review(reviewDecision 空即可 merge);main 才强制。此前 iter15 误把 main 保护当 dev 保护,导致所有 stalled PR 判定错误。
- **backlog 变更**: B005 BLOCKED→DONE (PR=#419 MERGED)。B008/B010 需重新评估:如果 #476/#403/#343 base=dev 且 reviewDecision 空,应可直接合。
- **下一步**: 新一轮 subagent 先跑 `gh pr view <N> --json baseRefName,reviewDecision,mergeStateStatus` 对每个 stalled PR 校准,能合的直接合。#419 进入 AWAIT_DEPLOY,查 deploy-dev workflow。

## META-REFLECT #2 (iter=22, done=10)
- 触发: done_count 自 iter11 #1 起 +7,超阈值 +5。
- 3 有效模式(放大):
  1. **signals 脚本驱动 DISCOVER** — M001 落地后每轮首信号真实(review-req/failed CI/stalled PR/Cloud Run/stranded),取代无效 grep TODO。
  2. **契约多参数探针 + 双写 spec** — T7 nodeId 探测(缺省/带值/边界)一次入 regression.md,后续复跑稳定绿。
  3. **元项快闭环 (CODING→DONE_ITEM 无 PR)** — 8/10 DONE 属此类,`.agent/loop/*` 内闭环无 human 依赖,产出速度快。
- 3 反模式(改掉):
  1. **PR 硬等 review** — #419/#476/#403/#343 全绿卡 human;循环无进展却占 IN_FLIGHT_PR 心智。→ 改 park+释放。
  2. **DISCOVER 静态扫仓库** — 5 子系统全部 0 TODO,openspec 幽灵率 93%;继续扫是浪费。
  3. **UI 层反复失败** — T5/T6 SSO 永久 SKIP,却未升级前端 UI 自动化能力,真业务 UI 回归覆盖=0。
- 3 下阶段策略:
  1. 循环重心从"扫代码找 TODO"切到"跑 signals 响应事件"(M005)。
  2. 研究 claude-in-chrome extension 打通 T5 SSO,补 UI 回归空洞(M004)。
  3. PR review 卡关立即 park + 释放 IN_FLIGHT_PR,不占 STALL_LIMIT。
- backlog 变更: +M004(P0) +M005(P0);LESSONS +L-M4 +L-M5。

## LOG iter19 DONE_ITEM regression.md T7 契约细节
- 阶段: IDLE → CODING → DONE_ITEM (元项, 无 PR)
- 证据: 双探针 curl (a) 无 nodeId → HTTP=400 code=INVALID_ARGUMENT ("workflow name and nodeId are required"); (b) 带 nodeId=nonexistent-node-t7 → HTTP=404 code=WORKFLOW_NOT_FOUND。T7 段增补"契约细节确认 2026-07-18 UTC:nodeId 必需"并明确 400 是命令写错而非回归。修正后 T7 复跑 pass (HTTP=404 code=WORKFLOW_NOT_FOUND)。
- 教训 (新 LESSON): API 契约变更/确认时,先跑多参数组合(缺省/带值/边界)再入 spec;单次观察不定契约,尤其是"参数是否必需"这类分支。
- backlog 变更: 无。
- 下一步: iter20 回 DISCOVER。

## LOG iter16 DONE B006 discover-signals 段1 假阳性修复
- 阶段: IDLE → CODING → DONE_ITEM (B006)
- 证据: 段1 修前 hits=1 (PR#419 rouse2617 self-authored 因 review:required 假阳性), 修后 hits=0;全脚本 5 段跑通 (段2 5 CI runs / 段3 3 stalled PRs / 段4/5 空)
- 教训: `gh pr list --search` 支持 `-author:@me` (负号排除) 组合 `review-requested:@me` 才是真 review 队列;jq 后置 `select(.author.login!="rouse2617")` 双保险防 GitHub search 语义变更
- 下一步: B007 处理 stalled PRs #476/#403/#343 (subagent 逐个诊断)

## LOG iter15 PARK #419 + LAND M002 openspec-ghost-filter
- PR#419 park:`gh pr merge --auto` 失败(repo 未启用 enablePullRequestAutoMerge);降级为 comment #5009522974,标记 B005 BLOCKED-pending-review、B008 DEFERRED-HUMAN。循环不再重试。
- M002 落地:`.agent/loop/scripts/openspec-ghost-filter.sh` 完成。核心逻辑:抽 change-dir primary `CYB-NNNN` tag → `git log --all -i --grep="$T\b"`,命中即 GHOST (exit 1)。踩坑修 2 处:(a) 提交 message 用小写 `cyb-3386`,需 `-i`;(b) 之前用 grep 全文抓 `#N` 会误命中任意 PR 号,改为只用 dir 前缀 tag。自测 3/3:CYB-3386→GHOST(3 commits), CYB-1013→GHOST(2), CYB-1014→GHOST(2)——三个 openspec 目录其实都该归档,后续可批量清理(新 backlog 候选)。
- regression.md 加 M002 段:DISCOVER openspec/changes 前先跑此 filter。
- 下一步:iter16 起 DISCOVER 重跑 discover-signals.sh 找新信号;M003 (T5 auth 三选一) 仍等用户拍板。

## LOG iter14 AWAIT_CI PR#419 merge blocked by branch protection
- 二次校验:checks 7/7 SUCCESS, mergeable=MERGEABLE, 但 mergeStateStatus=BLOCKED, reviewDecision=REVIEW_REQUIRED (reviewRequests 空,无 REQUEST_CHANGES)。
- `gh pr merge 419 --squash --delete-branch` → 失败:"base branch policy prohibits the merge"。
- 决策:遵守硬规则,不用 --admin;保持 AWAIT_CI;记 B008 P0,提示人工 approve 或 --auto。
- 下一步:等人工 approve 或改用 --auto;下轮再校验。

## LOG iter13 FIX_CI PR#419 openspec-gate
- 阶段:IDLE → DISCOVER → FIX_CI → PUSH。signals top 信号 review-req=PR#419 其实是我 authored(rouse2617)+ CI failed openspec-gate → 归为"我 stalled + CI 失败"。
- 根因:PR#419 缺 `openspec/changes/CYB-3516-*/specs/pipeline/spec.md` (validate_openspec_change.sh 要求 ≥1 spec delta)。runtime 改动仅 backend go,未触红线。
- 动作:checkout FETCH_HEAD → 新增 spec.md(MODIFIED Requirement "Anomaly reconcile 判定仅依据生命周期时间戳" + 3 scenarios:aged / fresh-started / created-only) → 本地 `bash scripts/validate_openspec_change.sh` OK → commit `docs(docs): ...` (commitlint scope=docs,lowercase subject 两次微调后过) → `SKIP=terraform_validate,terraform_tflint,terraform_fmt`(pre-push 缺 terraform 二进制,非我改动)→ `--force-with-lease` push 647b02e0。
- 副产:B006 记 signals 脚本第 1 段应过滤 authored-by-me;B007 记 stalled PR 476/403/343 待处理。
- done_count 4→5 (触发下轮 META-REFLECT #2)。

## LOG iter12 DONE_ITEM M001
- 阶段:IDLE → CODING → DONE_ITEM (元改进,无 PR)。
- 证据:`.agent/loop/scripts/discover-signals.sh` chmod +x,首跑 5 段各返回 1/5/4/0/0 行(review-req=PR#419;failed CI 5 条含 deploy-dev/DB migrate lint/repo-wiki/OpenSpec/notify-feishu;我 4 条 stalled PR;Cloud Run 30m 内无 ERROR;15m 内无 stranded 日志)。regression.md 补"DISCOVER 第一信号源"段。done_count 3→4。
- 教训:signals 首跑就暴露 PR#419 review-requested 及 4 条自己 stalled PR — 说明旧 DISCOVER 完全漏侦,升级即刻见效。
- backlog 变更:M001 DOING→DONE。
- 下一步:下轮 IDLE→DISCOVER 用脚本驱动;若无 signals 优先做 M002 openspec 幽灵过滤 hook;PR#419 review 需求纳入下轮 DISCOVER 判断。

## META-REFLECT #1
- 2026-07-18 iter11 IDLE→IDLE (触发规则:高价值任务长期搁置)。
- 证据统计:iter=11 done=3(B002/B003/B004,全部 meta 类:契约修正+幽灵清账)、empty_discover_streak=1、backlog 剩 1 P3 + 1 DEFERRED-HUMAN,10 轮内 0 真业务 PR。
- 3 症状:
  1. DISCOVER 主信号(grep TODO / go vet / npm lint)在本仓库常态 0 命中 —— 主线代码本身干净,自动化找不到真实积压。
  2. openspec/changes tasks.md 勾选滞后于合并,discover 幽灵率高(B004 即典型)。
  3. T5 前端 UI 因 SSO 卡在 in-app browser,永远达不到 done。
- 3 策略(落 backlog):M001 信号源升级(gh pr/actions/logs) · M002 openspec 幽灵过滤前置(git log --grep 反查) · M003 T5 auth 三选一(需用户拍板)。
- backlog 变更:+M001 +M002 +M003(均 P0)。

## LESSONS
- L-M1 (META): DISCOVER 若仅靠源码静态扫描,在成熟仓库会长期空转;应加事件驱动信号源(open PR review requested / failed CI runs / prod error logs)。
- L-M2 (META): openspec/changes tasks.md 不可信,读取时必先 `git log --all --format='%h %s' --grep="<change-id>"` 反查,命中即勾并跳过。
- L-M3 (META): 前端 UI 自动化在企业 SSO 下需专门 auth 通道 —— 无 headless 方案时应优先加固 API/契约层(T7),而非在 UI 层反复失败。
- L-M4 (META): 元改进项(loop 自身工具/脚本/regression.md)不需 PR,但需 backlog DONE 追踪;仅真业务 PR 才走 CODING→AWAIT_CI→AWAIT_DEPLOY→VERIFY 全流程。
- L-M5 (META): PR review 卡关即 park,不占 STALL_LIMIT;park 时立即释放 IN_FLIGHT_PR,让循环能推进别的事项。

## LOG
- 2026-07-18 iter8 IDLE→DISCOVER→IDLE (handlers 轮换收官): `grep -rn "TODO\|FIXME\|XXX" backend/internal/handlers/` = 0 hits;54 impl / 32 test = 覆盖率约 59%,无显性债务。全轮换扫描完成(backfill/pipeline/Frontend/openspec/handlers 均已过)。净新增 0,streak=1。LESSON:handlers 是最干净的模块,discover 可降低轮换频率。下一步继续 IDLE,或人工注入优先级。
- 2026-07-18 iter7 IDLE→CODING→IDLE (B004 pre-merged skip): 读 spec 后 `grep -rn KpiRow Frontend/src/` = 0 hits;`git log --all -- 'Frontend/src/components/assets/AssetsKpiRow*'` → **1c36e570 "fix(frontend): remove assets kpi row (cyb-3386) (#378)"** 已合入 dev。任务实体已完成,只是 openspec tasks.md 未同步勾选造成 discover 误判。无需新 PR;branch fix/CYB-3386-remove-kpi-row 已删。B004→DONE(PR #378)。LESSON L6:discover 扫 openspec/changes 时应 cross-check `git log --all -- <path-hint>` 确认未被 pre-merged,避免出 phantom 任务。下一步 IDLE 待新 backlog 项。
- 2026-07-18 iter6 DISCOVER→IDLE: 扫 openspec/changes(169 项),抽样确认多个 spec 有未打勾任务(CYB-3385/3076/3386…);挑最小的 CYB-3386-remove-kpi-row(7 tasks 纯删除)为 B004 P2 入队。streak 归 0。下一步 IDLE 待 CODING 优先级排。

## LESSONS
- L1 (2026-07-18): dev 后端 smoke T1/T3 通过;T2 契约的 jq path 错(实际响应是 `{job:{...}}` 包一层)。
- L2: `/node-summary` 顶层 `{nodes:[...]}`,node keys=[attempted,counts,dagOrder,displayName,failureRate,pipelineNodeId]。
- L3 (2026-07-18 Ralph#3): 后端 JSON 字段一律 **camelCase**(`totalCount` 而非 `total_count`),Ralph#2 journal 里 snake_case 猜测也是错的。**regression.md 契约必须先跑真实 curl+jq 落地再入 spec,禁凭空/凭记忆写**。
- L4 (2026-07-18 Ralph#4): 离线态 `go vet` 会因 GOPROXY EOF 直接失败,discover 前先探测网络,失败即降级并记 unknown,不重试。

## LOG

### 2026-07-18 Ralph#5 DISCOVER -> DISCOVER (Frontend/src scan, no PR)
- 扫 Frontend/src(App.tsx, api/, components/, features/, hooks/)。
- 证据:`grep -rn "TODO|FIXME|XXX" Frontend/src/ *.ts *.tsx` = 0 hits;`npx tsc --noEmit` 因无 node_modules 降级(未 install);测试文件计数=85。
- 教训:L5 前端 discover 若要 tsc 必须先 `npm ci`,离线态下降级到静态 grep + 文件计数即可。
- backlog 无新增。empty_discover_streak: 1→2(达阈值,下轮可切 IDLE 或换扫其他目录)。CURRENT_STATE 保持 DISCOVER。

### 2026-07-18 Ralph#4 DISCOVER -> DISCOVER (pipeline/ scan, no PR)
- 扫 backend/internal/usecase/pipeline/(20+ 文件,batch/cost/scheduling/stats/subtask 等)。
- 证据:`grep -rn "TODO|FIXME|XXX" pipeline/` = 0 hits。`go vet pipeline/...` 因 GOPROXY EOF 网络故障降级(未重试),记为 unknown。
- 教训:L4 无网离线态下 go vet 不可用,后续 discover 若需 vet 应先 `go env GOPROXY` 探测。
- backlog 无新增。empty_discover_streak: 0→1。CURRENT_STATE 保持 DISCOVER。
- 下轮 DISCOVER 建议扫 Frontend/src/pages/BatchJobDetailPage.tsx。

### 2026-07-18 Ralph#3 DISCOVER -> DISCOVER (regression.md hotfix, no PR)
- 判定为 CODING 简化路径:B002/B003 是文档纠正,直接改 .agent/loop/,不动业务码/git。
- 真实 curl+jq 结果:T2 `.job.totalCount=10000, completedCount=9835, failedCount=1, status="paused"`;T3 `.nodes|length=1, keys=[attempted,counts,dagOrder,displayName,failureRate,pipelineNodeId], attempted=9836, counts type=object`。
- 副发现:Ralph#2 journal L1/L2 的字段名 snake_case (`total_count`) 也是错的 — 后端实为 camelCase。规约按真实响应重写。
- B002/B003 → DONE。CURRENT_STATE→DISCOVER 供下轮扫新子系统。
- LESSON L3 记录:契约必先真跑一次再入 spec。

### 2026-07-18 Ralph#2 IDLE -> DISCOVER (via VERIFY smoke baseline)
- PREFLIGHT: fetch origin dev,checkout dev(修复漂移)。本地落后 origin/dev 1 commit,未 pull(不影响只读 smoke)。
- T1 GET /api/v1/backfill?pageSize=1 → HTTP 200 (后端存活)。
- T2 GET /backfill/8f5cd676… → 200,但响应形状是 `{"job":{...}}`,规约 jq `.total_count` 得 null。**规约 bug**,非后端回归。加 B002。
- T3 GET /node-summary → nodes.length=1,keys=[attempted,counts,dagOrder,displayName,failureRate,pipelineNodeId]。基本通过(len>0),但 keys 与规约 `status/count` 不符。加 B003 修规约。
- 首次真实回归 baseline 建立:后端可用,契约文档过期。
- 下轮 DISCOVER 建议扫 Frontend/src/ 或 backend/internal/usecase/pipeline/(未扫过子系统)。


### 2026-07-18 Ralph#1 DISCOVER -> IDLE
- PREFLIGHT: 首轮,记忆文件不存在,初始化 state/backlog/journal。
- 分支实际为 fix/CYB-3667-argo-podgc (HEAD bdd7a25cd2),非任务预期 dev/bd387f8d。未切换(遵守只写 .agent/loop 硬规则)。
- 扫 backend/internal/usecase/backfill/: `grep TODO|FIXME|XXX` = 0 hits(近期修复已清理)。
- 扫 openspec/: changes 170 项、specs 存在。数量太多,未逐项验证落地状态,建 B001 作为后续待办。
- backend go build: 触发大量 go mod 下载,超时前未见编译错误(未见 error)。
- 已知运维事项(batch paused / Argo GC 历史 stranding)登记 DEFERRED-HUMAN-001,不新增代码。
- 新增 backlog 项 = 2 (含 1 DEFERRED),empty_discover_streak=0。
- 下一步:进入 IDLE,下轮可 PLAN B001 或按新指令。

### 2026-07-18 Ralph#9 IDLE -> IDLE (regression VERIFY baseline)
- T1 GET /backfill?pageSize=1 → 200。
- T2 batch 8f5cd676: total=10000, completed=9835, failed=1, status=paused。
- T3 node-summary: len=1, keys 严格匹配 [attempted,counts,dagOrder,displayName,failureRate,pipelineNodeId], attempted=9836, counts=object。
- T4 一致性: 9835+1=9836 ≤ 10000 ✓。
- T5 前端 UI: preview_start 打开 /runs?executionView=batch,localStorage 注入 dev-token 后 reload,页面仍返回登录表单("公司邮箱"+"登录"按钮)——前端不接受 X-Databrew-Token,需 Google SSO cookie。判为**环境限制**,非回归 bug。
- T6 console: onlyErrors=true → 空。
- LESSON L4:in-app browser (preview_start) 无法完成 Google SSO,前端 UI 冒烟必须由 maintainer 人肉或 headless+refresh-token 方案完成;当前 CI 侧仅能验证到登录页存活。
- backlog 新增 DEFERRED-HUMAN-002「前端 UI 冒烟需 SSO fallback 方案」。
- streak 保持 1,IDLE→IDLE。

## LOG 2026-07-18 iter=10
- STAGE: IDLE -> IDLE (+T7)
- EVIDENCE: 锚 commit 1a7b81cd (#464 gone workflow logs → graceful 404). T7 baseline: GET /api/v1/workflows/ralph-nonexistent-workflow-t7/logs?nodeId=nope → 404 + code=WORKFLOW_NOT_FOUND ✓
- LESSON: T7 baseline @2026-07-18: 404+WORKFLOW_NOT_FOUND on missing wf, guards against 500 regression of #464.
- BACKLOG: no change
- NEXT: DISCOVER,继续找 P1/P2 candidates

## iter 16 (2026-07-18)
- Signals: 段1/2/3/4/5 = 0/5(旧)/3/0/0. 段2 均为 24h 外或非 dev 主线,无新 P0。
- 选择动作: B009 openspec 幽灵批量报告。
- 结果: 扫 168 dir,157 GHOST,报告落 `.agent/loop/openspec-ghost-report.md`。**不移动目录**,只出清单。
- REFLECTION: 幽灵率 93%——openspec/changes 已严重漂移,建议下次触发 maintainer 一次性归档批处理。
- iter17 B007 DONE: #476/#403/#343 CI 全绿 reviewDecision=空 → 归 B010 pending-maintainer-review;非我能推。done_count=9

## iter 10 · 2026-07-18 · baseline VERIFY #2
- T1=200 T2=tot10000/comp9835/fail1/paused T3=n=1/attempted=9836/counts=object T4=pass(9835+1<=10000) T7=404 WORKFLOW_NOT_FOUND (需 nodeId 参数,已用 nodeId=x)
- 全绿 vs baseline #1(iter 9)。T7 契约学到:endpoint 需 nodeId query,否则 400 INVALID_ARGUMENT。已记入 regression.md 待补丁。
- signals: 3 stalled PR(#476/#403/#343 均 DEFERRED-HUMAN),无新 CI 红/无 Cloud Run ERROR。
- LESSON: baseline #2 green @ 2026-07-18; T7 curl 必须带 nodeId=x

## iter 24 (2026-07-18) — B011 CI-flake 结案
- signals 段1-5=0/5/0/0/0(段2 5 条多为他人分支;仅 dev 一条 = 29567283859)
- 处理:段 2 dev 唯一失败 run 29567283859(07-17 08:41,commit 5c2e608b PR#471 merge)→ 仅 `mcap-preview` sub-job Build&push image on VM failure,其他 backend/frontend/migrate 全 SUCCESS
- 后续 5 次 dev deploy-dev 全 conclusion=success(至 22ad2c8d 04:01)→ 判定为 mcap-preview 单点 flake,已自愈
- B011 → DONE
