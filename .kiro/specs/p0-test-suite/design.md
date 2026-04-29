# Design — P0-T: 测试同窗口配套

## Source of Truth

- 集成测试矩阵：#[[file:docs/review/outbox-worker-design.md]] §12
- 对账脚本：#[[file:docs/review/next-steps-tasks.md]] P0-T-3
- 前端测试：#[[file:Frontend/README.md]]

## 变更范围

### Outbox 集成测试（P0-T-1）
- 新增 `internal/outbox/e2e_test.go`（build tag `//go:build integration`）
- 使用 testcontainers-go 启 PG + ES 容器
- 五个用例覆盖 happy path、dedup、restart replay、concurrent ack、ES down backpressure

### CI（P0-T-2）
- `.github/workflows/test.yml`（或等价）新增 integration job
- testcontainers 镜像 cache

### Backfill 对账（P0-T-3）
- `scripts/backfill-audit.sh`：扫 `assets / mcap_files`，对比新字段空值率、双写一致率
- 输出 JSON 报告

### 前端覆盖率（P0-T-4）
- `vitest.config.ts` 加 coverage 配置
- CI 加 coverage gate（核心组件 ≥ 70%）
