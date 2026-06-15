# Optimizer Loop Report — 2026-06-15

## 2026-06-15 08:50 — LOOP 1
- 目标: backfill goroutine 错误上报修复（SSOT P-1 / C-1）
- 改: backend/internal/usecase/backfill/usecase.go
- 测: go test ./internal/usecase/backfill/... -v
- 结果: PASS (9/9 tests)
- 行数: +15 -1

**改动摘要:**
- `materializeAndRunBatch` 返回 `<-chan error` 而非 `void`
- Fatal errors (SaveItems 失败、UpdateJobStatus 失败) 通过 buffered channel 上报
- `CreateBackfill` 两处 goroutine 启动处：用 `<-errCh` 接收并用 `slog.Error` 记录
- 原有 Warn 日志保持不变（UpsertBatchSubtaskRun/UpdateItemPipelineRun 失败不影响批创建）

**核心风险修复:**
- ❌ → ✅ SaveItems 失败不再静默；错误写 logger + job 置 failed + channel 关闭
- ❌ → ✅ UpdateJobStatus 失败不再静默；错误写 logger + channel 关闭

---
