# Backlog

| id | 优先级 | 任务 | 类型 | file:line | 影响 | PR | 尝试次数 |
|----|--------|------|------|-----------|------|----|----|
| B001 | P3 | 审阅 openspec/changes 170 项是否有未落地 spec | openspec | openspec/changes/ | 潜在漂移 | - | 0 |
| B005 | P0 | DONE (iter22.6) PR#419 用户手动 approve + squash MERGED @ a97f25e0;deploy-dev success;T1-T3+T7 baseline 全绿 | ci-fix | openspec/changes/CYB-3516-fix-watcher-anomaly-reconcile-loop/specs/pipeline/spec.md | 解锁 PR#419 合并 | #419 | 1 |
| B006 | P1 | DONE signals 段1 加 `-author:@me` GitHub search filter + jq 后置 select 双保险;修前 hits=1 (#419 rouse2617 self), 修后 hits=0;全脚本 5 段跑通 | meta | .agent/loop/scripts/discover-signals.sh | 减 discover 假阳性 | - | 1 |
| B007 | P2 | DONE 三 PR 状态分析:#476/#403/#343 均 CI 全绿(preview+gitleaks SUCCESS), reviewDecision="" 无 reviewer 请求 → 归为 pending-maintainer-review;非我能推,记入 B010 待人工 | ops | github | 生产运维 | 476/403/343 | 1 |
| B010 | P2 | DONE (iter22.6) PR#476 MERGED @ b4ac19c9 + deploy verified; #403 MERGED @ 56dcd11c (docs-only, no deploy); #343 MERGED @ 22ad2c8d + deploy verified | ops | github branch protection | 三 PR 卡住 | 476/403/343 | 0 |
| B008 | P0 | DONE (iter22.5) 根因是 #419 base=main 违反红线;`gh pr edit --base dev` 后 reviewDecision 空即可 merge。L-M7/L-M8 已入 LESSONS 防复发 | ops | github branch protection | 阻塞 dev 合入 | #419 | 2 |
| DEFERRED-HUMAN-001 | P2 | batch 8f5cd676 paused + 17 pending + 147 submitted 卡住(Argo GC 历史 stranding pre-#471) | ops | - | 生产运维 | - | 0 |
| B002 | P2 | DONE 修 regression.md T2 jq path → `.job.totalCount` 等(camelCase);curl+jq 输出匹配 spec | docs | .agent/loop/regression.md T2 | 回归误报 | - | 1 |
| B003 | P2 | DONE 修 regression.md T3:顶层 `.nodes`,keys=[attempted,counts,dagOrder,displayName,failureRate,pipelineNodeId];curl+jq 输出匹配 spec | docs | .agent/loop/regression.md T3 | 回归误报 | - | 1 |
| B004 | P2 | DONE 已在 PR #378 (1c36e570) 合入,openspec tasks.md 未及时勾选造成漂移;spec 建议归档 | openspec | openspec/changes/CYB-3386-remove-kpi-row/ | 前端瘦身 | #378 | 0 |
| M001 | P0 | DONE 元改进:DISCOVER 信号源升级 → `.agent/loop/scripts/discover-signals.sh`(5 段:review-req PR / failed CI / stalled my-PR / backend ERROR / stranded 日志);首跑各 sec 分别返回 1/5/4/0/0 行;regression.md 已加"DISCOVER 第一信号源"段 | meta | .agent/loop/scripts/discover-signals.sh | 循环产出 | - | 1 |
| M002 | P0 | DONE 元改进:`.agent/loop/scripts/openspec-ghost-filter.sh` 已落地,基于 change-dir primary CYB tag 反查 git log(-i case-insensitive, \b word boundary)。自测 CYB-3386/1013/1014 三个 change 均正确判定为 GHOST(合入 commit 存在)。regression.md M002 段落已更新。 | meta | .agent/loop/scripts/openspec-ghost-filter.sh | 减 discover 假阳性 | - | 1 |
| M003 | P0 | 元改进:T5 前端 auth 方案定案(a) in-app browser 一次性 OAuth dump auth-state.json (b) dev 前端加 X-Databrew-Token 后门 (c) 放弃 UI 自动化,加强 T7+API 契约 —— **需用户确认三选一**,Ralph 可先起草 (a) 落地脚本 | meta | .agent/loop/regression.md T5 | 前端 UI 无法自动化 | - | 0 |

## DEFERRED-HUMAN-002 前端 UI 冒烟需 SSO fallback
- 现象:in-app browser 打开 https://cyber-databrew-dev.cyberorigin.ai/runs 只显示"公司邮箱"登录表单,localStorage 注入 databrew_token 无效。
- 需要:maintainer 提供 headless SSO 方案(refresh token / service account cookie),或在 dev 前端加 `?token=<devtoken>` 后门。
- 影响:T5/T6 无法自动化,回归依赖人工点击。
| M004 | P0 | 元改进:研究 claude-in-chrome extension 是否可承接 T5 前端 UI 自动化(用户已登录 Chrome session,绕过 SSO)。若可行,补 T5/T6 至绿,填 UI 回归空洞。 | meta | .agent/loop/regression.md T5/T6 | UI 回归 0→>0 | - | 0 |
| M005 | P0 | DONE 策略文档 `.agent/loop/RALPH-STRATEGY.md` 落地;regression.md 顶部指针加入;吸收 iter 22.6 用户手动介入的教训(auto-merge 缺失应对) | meta | .agent/loop/RALPH-STRATEGY.md | 循环产出 | - | 1 |
| B011 | P2 | DONE run 29567283859 仅 mcap-preview sub-job build image failure(其他 backend/frontend/migrate 全 SUCCESS);后续 5 次 dev deploy-dev 全绿(至 22ad2c8d);判定 mcap-preview 单点 flake 已自愈 | ci-fix | .github/workflows/deploy-dev | dev 部署健康 | - | 1 |
| B009 | P2 | DONE openspec 幽灵批量扫描:157/168 change dir 命中 primary CYB tag 在 git log,报告见 `.agent/loop/openspec-ghost-report.md`。仅出报告,未归档(需 maintainer 批处理,规则偏松可能含 WIP 分支 commit) | meta | .agent/loop/openspec-ghost-report.md | 减 openspec 漂移 | - | 1 |
