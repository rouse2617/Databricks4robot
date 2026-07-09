# 任务清单

- [ ] **修改后端** — 在 `ListRunSummaries` 的所有执行路径中调用 `attachVideoDurations()`
  - 文件：`backend/internal/usecase/pipeline/usecase.go`
  - 位置：line 3870 附近（第二个 `items, err := uc.runRepo.FindAllSummaries(ctx)` 之后）
  - 添加：`uc.attachVideoDurations(ctx, items)` 调用

- [ ] **本地验证**
  - 构建后端：`make build-backend`
  - 启动后端：`source scripts/dev-backend-env.sh && ./bin/server`
  - 打开浏览器：`https://cyber-databrew-dev.cyberorigin.ai/pipeline?tab=executions`
  - 检查单次执行列表中 "时长" 列是否显示数据

- [ ] **提交代码**
  - Commit message：`fix(backend): attach video durations to single-run execution list`
  - 签名：Co-Authored-By

- [ ] **创建 PR**
  - Base: `dev`
  - Title：`fix(backend): show video duration in single-run execution list`
  - 描述：包含 proposal.md 和验证步骤
