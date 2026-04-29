# Tasks — P1-T: 2.0 测试套件

> Source of truth: #[[file:docs/review/outbox-worker-design.md]] §12, #[[file:docs/review/data-platform-design.md]] §8.2

## Milestone 1: PG↔ES 一致性对账（P1-T-1）

- [ ] 1.1 新增 `scripts/es-pg-audit.sh`
- [ ] 1.2 比对 PG `assets` 与 ES `assets` 的 (asset_id, version) 集合
- [ ] 1.3 diff 报告输出 missing / stale / extra
- [ ] 1.4 一致率 > 99.9%
- [ ] 1.5 CI nightly job

## Milestone 2: Outbox 性能压测（P1-T-2）

- [ ] 2.1 压测脚本：50 events/s × 60s 持续写入
- [ ] 2.2 验证端到端 P99 ≤ 60s
- [ ] 2.3 规模上扬到 200 events/s 观测 worker 行为
- [ ] 2.4 基线报告归档

## Milestone 3: SDK 集成测试（P1-T-3）

- [ ] 3.1 `sdk/tests/e2e/` 目录结构
- [ ] 3.2 启 backend container（testcontainers 或 docker compose）
- [ ] 3.3 pytest E2E 用例全绿
- [ ] 3.4 CI 矩阵 `sdk-e2e` job

## Milestone 4: 事件 Schema CI 守门测试（P1-T-4，blocked-by P1-7）

- [ ] 4.1 模拟 producer 加字段不 bump version → CI 失败
- [ ] 4.2 模拟 1 次 minor bump 用例
- [ ] 4.3 模拟 1 次 major bump 用例

## Milestone 5: 前端 E2E（P1-T-5）

- [ ] 5.1 Playwright 配置
- [ ] 5.2 1 条 happy-path：登录 → 资产列表筛选 → 详情 → 事件时间线
- [ ] 5.3 CI 加 e2e job（可选 nightly）
