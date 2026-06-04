# Tasks — CYB-1649

## Part A: transpiler 默认超时

- [ ] A1. 在 `transpiler.go` 添加 `DefaultActiveDeadlineSeconds` 常量 (7200s)
- [ ] A2. 在 `Transpile()` 中添加零值默认 fallback
- [ ] A3. 添加 `TestTranspileDefaultActiveDeadlineSeconds` 测试

## Part B: 后端 staleness 检测

- [ ] B1. 在 `usecase.go` 添加 `stuckPodGracePeriod` / `workflowHardTimeout` 常量
- [ ] B2. 在 `usecase.go` 添加 `detectStuckWorkflow` 辅助函数
- [ ] B3. 修改 `refreshRunStatus` 在 phase copy 后调用 staleness 检测
- [ ] B4. `PipelineRunRepository.UpdateStatus` 加 `message` 参数
- [ ] B5. `PipelineRunRepo.UpdateStatus` SQL 支持 message 持久化
- [ ] B6. 更新 mock + 测试

## Verification

- [ ] V1. `go test ./internal/transpiler/...` ✓
- [ ] V2. `go test ./internal/usecase/pipeline/...` ✓
- [ ] V3. `make fmt && make vet` ✓
- [ ] V4. Deploy dev + smoke test
