# Requirements — P1-T: 2.0 测试套件

## Introduction

2.0 阶段测试：PG↔ES 一致性对账、Outbox 性能压测、SDK 集成测试、事件 schema CI 守门测试、前端 E2E。

Source of truth:
- #[[file:docs/review/outbox-worker-design.md]] §12（测试策略）
- #[[file:docs/review/data-platform-design.md]] §8.2（业务级 SLO）
- #[[file:docs/review/next-steps-tasks.md]] P1-T-1 ~ P1-T-5

## Requirements

### R1: PG↔ES 一致性对账（P1-T-1）
- `scripts/es-pg-audit.sh`：一致率 > 99.9%
- diff 报告输出 missing / stale / extra
- CI nightly job

### R2: Outbox 性能压测（P1-T-2）
- 50 events/s × 60s → 端到端 P99 ≤ 60s（G1）
- 基线报告归档

### R3: SDK 集成测试（P1-T-3）
- 启 backend container 跑 pytest E2E
- `sdk/tests/e2e/` 全绿
- CI 矩阵 `sdk-e2e` job

### R4: 事件 schema CI 守门测试（P1-T-4，blocked-by P1-7）
- 模拟 producer 加字段不 bump version → CI 必失败
- 至少 1 次 minor / 1 次 major 模拟用例

### R5: 前端 E2E 框架（P1-T-5）
- Playwright 跑通 1 条 happy-path
- CI 加 e2e job（可选 nightly）
