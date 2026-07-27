# Pipeline Spec Delta — CYB-4026

## MODIFIED Requirements

### Requirement: 下发速率配置不得使集群停摆

The per-cluster dispatch rate limiter SHALL always permit at least one submission per token-bucket refill, for any positive configured `rate_per_sec` within the allowed range (≥ 0.1). A slow-rate configuration MUST throttle submissions, never halt them.

**Priority**: P0 (Critical)
**Rationale**: burst 由 `int(rate)*2` 计算，`0.1 ≤ rate < 1` 时归零，`limiter.Wait(1)` 报错、错误分支在计数前 break，集群永久空转、无 DLQ、无告警。

#### Scenario: 慢速率仍持续下发
- **Given** 某 cluster 的 `rate_per_sec` 被配置为 0.5（合法范围内）
- **When** 该 cluster 有 pending item 待下发
- **Then** 下发以约 0.5/s 的节流速率持续进行，不出现「零下发空转」
- **Then** limiter 的 burst ≥ 1（default 构造、`BACKFILL_CLUSTER_RATE` 环境变量、在线 applyConfig 三条路径一致）

#### Scenario: 区分取消与限速错误
- **Given** 一次下发因 burst/限速返回错误（非 context 取消）
- **When** 提交循环处理该错误
- **Then** 该次尝试计入 attempt 并按正常重试/DLQ 流程处理，不被当作 ctx 取消而静默 break

### Requirement: submit_batch 在线覆盖对 refill 生效

The refill / self-kick decision SHALL use the effective per-cluster `submit_batch` in force this cycle (governor override when set), not the compiled default.

**Priority**: P1 (High)
**Rationale**: refill 判定写死编译期常量 128，无视 governor per-cluster override(1–200)，调小静默失效、调大误 kick。

#### Scenario: 调小 submit_batch 仍持续喂满
- **Given** 某 cluster `submit_batch` 在线调为 2，且某 job 有 3 个 pending
- **When** 一个下发 cycle 执行
- **Then** 本轮下发 2 个，并触发一次 self-kick 继续处理剩余
- **Then** 不因 `2 >= 128` 为假而停到下一次 ticker

### Requirement: 下发进入 DLQ 时保留失败原因

When a submission reaches the max-attempts cap (DLQ), the failure reason SHALL be persisted where the batch / DLQ views can read it, regardless of whether the item already has a pipeline_run id.

**Priority**: P1 (High)
**Rationale**: 达 cap 且 `runID != ""` 的常见路径走 `UpdateItemPipelineRun(..., "failed")`，不写 `error_message`，失败原因丢失。

#### Scenario: DLQ item 携带失败原因
- **Given** 某 item 下发 transient 失败并达到 `maxSubmitAttempts`
- **When** 它被标记为 failed 进入 DLQ
- **Then** 该 item 的 `error_message`（或其 run 的 message）非空且包含原始错误（如 "max submit attempts (N) exceeded: dial tcp ..."）
