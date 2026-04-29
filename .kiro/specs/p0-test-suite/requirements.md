# Requirements — P0-T: 测试同窗口配套

## Introduction

补齐 P0 阶段的测试基础设施：Outbox Worker 集成测试、CI 集成、backfill 对账脚本、前端覆盖率守门。

Source of truth:
- #[[file:docs/review/outbox-worker-design.md]] §12（测试策略）
- #[[file:docs/review/next-steps-tasks.md]] P0-T-1 ~ P0-T-4

## Requirements

### R1: Outbox Worker 集成测试（P0-T-1）
testcontainers-go 起 PG + ES 容器，build tag `//go:build integration`，五件套：
- `TestE2E_Notify_HappyPath`
- `TestE2E_DedupBatch`
- `TestE2E_RestartReplay`
- `TestE2E_ConcurrentAck_NoSeqGap`
- `TestE2E_ESDown_Backpressure`

### R2: CI 接集成测试（P0-T-2）
GitHub Actions 矩阵跑单测 + integration build tag，PR 中两条流水线都绿。

### R3: Backfill 对账脚本（P0-T-3）
`scripts/backfill-audit.sh` 输出 JSON 报告；新字段空值率 < 0.1%；双写一致率 100%。

### R4: 前端单测覆盖率守门（P0-T-4）
vitest coverage gate，核心组件行覆盖 ≥ 70%，CI 上线最低门槛。
