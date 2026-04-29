# Tasks — P0-T: 测试同窗口配套

> Source of truth: #[[file:docs/review/outbox-worker-design.md]] §12

## Milestone 1: Outbox Worker 集成测试（P0-T-1，blocked-by P0-4）

- [ ] 1.1 新增 `internal/outbox/e2e_test.go`，build tag `//go:build integration`
- [ ] 1.2 testcontainers-go 启 PG + ES 容器
- [ ] 1.3 `TestE2E_Notify_HappyPath`：写 1 event → < 2s ES 可查
- [ ] 1.4 `TestE2E_DedupBatch`：同 asset 写 100 events → ES BulkIndex doc 数 = 1
- [ ] 1.5 `TestE2E_RestartReplay`：写 100 → 杀 worker → 再写 100 → 重启 → ES 最终 200
- [ ] 1.6 `TestE2E_ConcurrentAck_NoSeqGap`：模拟乱序 publish → cursor 永不越过 MIN(pending)-1
- [ ] 1.7 `TestE2E_ESDown_Backpressure`：ES 停 30s → 事件累积 → ES 恢复后全量到位

## Milestone 2: CI 集成（P0-T-2，blocked-by P0-T-1）

- [ ] 2.1 GitHub Actions workflow 新增 `test-integration` job
- [ ] 2.2 testcontainers 镜像 cache 配置
- [ ] 2.3 PR check 两条流水线（unit + integration）都绿

## Milestone 3: Backfill 对账脚本（P0-T-3）

- [ ] 3.1 新增 `scripts/backfill-audit.sh`
- [ ] 3.2 扫 `assets` 新字段空值率，输出 JSON
- [ ] 3.3 扫 `mcap_files` 新字段空值率
- [ ] 3.4 双写一致率检查（新列 vs cf_* JSONB）
- [ ] 3.5 空值率 < 0.1%，一致率 100%

## Milestone 4: 前端覆盖率守门（P0-T-4）

- [ ] 4.1 `vitest.config.ts` 加 coverage 配置（istanbul / v8）
- [ ] 4.2 核心组件（assets / asset-detail）行覆盖 ≥ 70%
- [ ] 4.3 CI 加 coverage gate，不达标禁止合 PR
