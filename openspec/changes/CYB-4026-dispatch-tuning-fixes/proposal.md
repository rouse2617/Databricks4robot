# Proposal — CYB-4026

## Why

CYB-3679（下发链路在线调参）合并后，代码审查在 `backend/internal/usecase/backfill/` 发现 3 个缺陷，均在 dev 现行 `3fb495aa` 复核确认：

- **D4（最高优先，静默停摆）**：token bucket burst 用 `int(ratePerSec)*2`（`submitter_cluster.go:97` 构造、`:125` applyConfig）。合法范围下限 0.1（`dispatcher_config.go`/前端 `dispatcher.ts`），`0.1 ≤ rate < 1` → burst=0。burst=0 时 `limiter.Wait(1)` 返回 "exceeds limiter's burst 0"，错误分支在 `attempts++` 前 `break`（注释误标 ctx cancelled）→ 该 cluster 每 tick 空转、永不下发、无 DLQ、无告警。运维填 0.5「慢速保护」或 `BACKFILL_CLUSTER_RATE=0.5` 即触发。
- **D3（吞吐退化）**：`submitter_cluster.go:293` 用编译期常量 `perJobSubmitBatch`(128) 判 refill/self-kick，但实际批量已被 governor `submitBatchLimit()` per-cluster override(1–200) 覆盖 → 调小静默失效（退化成 ~8 条/min）、调大误 kick 空转。
- **D5（可观测性）**：DLQ 达 `maxSubmitAttempts` 时构造了 `errMsg`，但常见路径（`runID != ""`，`submitter.go:467`）走 `UpdateItemPipelineRun(..., "failed")`（SQL 无 error_message），失败原因丢失；仅 `runID==""`（`:469`）才写原因。

## What Changes

### Modified Capabilities
- **pipeline**: 每-cluster 下发速率无论配置为何都保证至少 1 的 burst（合法慢速率不再使 cluster 停摆）；submit_batch 的 per-cluster override 生效于 refill/self-kick 判定（在线调小仍持续喂满、调大不误触发）；下发达最大重试进入 DLQ 时，失败原因必须落到可被批次/DLQ 视图读取之处。

## Impact
- **Affected code**:
  - `backend/internal/usecase/backfill/submitter_cluster.go`：抽 `burstForRate(rate float64) int`（≥1），构造(97)与 applyConfig(125) 共用；refill 判定(293)改用本轮 effective limit。
  - `backend/internal/usecase/backfill/submitter.go`：错误分支区分 ctx 取消 vs burst 错误，勿在 attempts++ 前误 break；`submitJobBatch` 返回 effective limit/`filledBatch`；DLQ 路径写失败原因。
  - `backend/internal/postgres/backfill_repo.go` + `backend/internal/repository/backfill_repository.go`：新增/扩展能同时绑定 run/status/error 的 item 更新方法。
  - 测试：`dispatcher_config_test.go`、submitter 相关 `_test.go`。
- **New APIs**: 无（内部下发逻辑）。
- **Dependencies**: 无新增。

## Scope
- **In scope**: D4 burst floor + break 区分；D3 effective-limit refill；D5 DLQ 失败原因落库；对应回归测试。
- **Out of scope**: D1（占位名/真名分裂，run 状态机，单独 CYB）、D2（Deploy 恒真条件 + UID 覆写）、D6（占位 run 空烧 starvation）——均另开。CYB-3679 已交付的在线调参功能本身不变。

## Success Criteria
- [ ] `burstForRate` 对任意正 rate（含 0.1/0.5/0.9）返回 ≥1；覆盖 default 构造、env(`BACKFILL_CLUSTER_RATE`)、在线 applyConfig 三条路径，断言 `limiter.Burst() >= 1`。
- [ ] rate=0.5 的 cluster 能持续下发（不再空转）；ctx 取消与 burst 错误在日志/计数上可区分。
- [ ] SubmitBatch=2 且 3 个 pending：单 cycle 下发 2 且触发一次 kick（断言 kick 发生）；SubmitBatch 未满不误 kick。
- [ ] DLQ 失败 item 的 `error_message`（或 run.message）非空且含原始错误。
- [ ] Tier M（`go test ./internal/usecase/backfill/... ./internal/postgres/...`）+ dev 验证通过。
