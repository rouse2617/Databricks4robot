# Tasks — CYB-4026

Branch: `fix/CYB-4026-dispatch-tuning-fixes`（from `origin/dev`）. Verification tier: **M**（backend usecase/repo + 测试；无 HTTP API、无 migration）。

## 1. D4 — burst floor + break 区分 ✅ 已实现
- [x] `submitter_cluster.go`：抽 `burstForRate(rate float64) int` = `max(1, ceil(rate*2))`。
- [x] 构造与 `applyConfig` 均改用 `burstForRate(...)`（原 `int(rate)*2` 两处）。
- [x] `submitter.go` limiter.Wait 错误分支：`ctx.Err()==nil` 时 WARN（区分真取消 vs 其它 limiter 错误），修正误标注释。

## 2. D3 — effective-limit refill ✅ 已实现
- [x] `submitJobBatch` 签名改为返回 `(attempts, limit, outcomes)`——本轮 effective limit。
- [x] `runClusterChannel` 用 `attempted >= effectiveLimit` 判 refill（替换编译期常量 `perJobSubmitBatch`）。

## 3. D5 — DLQ 失败原因落库 ✅ 已实现
- [x] 新增 `MarkItemFailedWithRun(id, runID, wf, errMsg)`：interface + postgres impl（写 pipeline_run_id + status=failed + error_message + finished_at）。
- [x] 更新 4 个 mock——同 commit。
- [x] `submitter.go` DLQ 路径 `runID != ""` 改用 `MarkItemFailedWithRun`（原 `UpdateItemPipelineRun` 丢 errMsg）。

## 4. 测试 ✅ 已加并通过
- [x] `TestBurstForRate_NeverZero` + `TestGovernorBurstFloor`（构造 & applyConfig(0.5) burst≥1）。
- [x] `TestSubmitJobBatch_OverrideBelowDefaultStillKicks`：编译期常量保持 128、override=2、3 pending → 下发 2 且 self-kick 触发。
- [x] `TestSubmitter_TransientRetriesThenDLQ` 扩断言：DLQ item `error_message` 非空且含 "max submit attempts"。

## 5. 验证（本机）✅
- [x] `gofmt -l` 干净；`CGO_ENABLED=0 go build ./...` 通过（本机 cgo `-E` 故障，用 CGO 关闭）。
- [x] `CGO_ENABLED=0 go test ./internal/usecase/backfill/... ./internal/postgres/... ./internal/repository/...` 全绿。

## 6. Deploy verify（dev）—— ⚠️ 待你自测（本轮不代跑）
无 migration。`bash deploy/cloudrun/backend-dev.sh`，`source scripts/dev-backend-env.sh` 后手动 smoke：
- **D4**：`PUT /api/v1/admin/dispatcher/clusters` 设某 cluster `rate_per_sec=0.5` → 给该 cluster 建批次 → 该 cluster **持续下发**（修复前 burst=0 会卡死空转）。
- **D3**：设 `submit_batch=2`，建 >2 pending 批次 → 单 dispatch cycle 下发 2 后**立即续跑**（非等 15s tick）。
- **D5**：制造 transient 失败到达 `maxSubmitAttempts` 的 item → DLQ 后该 item `error_message` **非空**且含失败原因。
- 验证 OK 后再 commit/push + PR（deploy-before-commit）。

## 7. API 契约同步
- N/A —— 无新增/变更 HTTP API（纯内部下发逻辑）。
