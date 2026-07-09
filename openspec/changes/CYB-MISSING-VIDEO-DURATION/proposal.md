# 修复视频时长在单次执行列表中缺失

## 问题

在管道执行列表页面 (`/pipeline?tab=executions`)，**单次执行** 模式下的 "时长" 列始终为空。

**根本原因**：后端 `ListRunSummaries` 方法中，`attachVideoDurations()` 仅在过滤批量任务时调用（`BatchJobID != ""`），不会在列出所有单次执行时调用。

```go
// 当前代码（usecase.go:3849-3872）
if len(filter) > 0 {
    items, total, err := uc.runRepo.ListSummaries(ctx, filter[0])
    // ...
    if filter[0].BatchJobID != "" {
        uc.attachBatchNodeProgress(ctx, items)
        uc.attachVideoDurations(ctx, items)  // ❌ 只在 BatchJobID != "" 时
    }
    // ...
}
items, err := uc.runRepo.FindAllSummaries(ctx)  // ❌ 没有调用 attachVideoDurations
```

## 解决方案

在 `ListRunSummaries` 的所有执行路径中都调用 `attachVideoDurations()`，确保：
1. **批量任务列表** — 继续有视频时长
2. **单次执行列表** — 新增视频时长数据

修改点：
- `backend/internal/usecase/pipeline/usecase.go`：在 line 3870 左右添加 `uc.attachVideoDurations(ctx, items)`

## 影响范围

- **无 API 变更** — 只是填充现有的 `videoDurationSec` 字段
- **无前端改动** — 前端已支持显示该字段
- **无数据库改动** — `video_durations` 表已存在

## 验证方式

1. 打开 `/pipeline?tab=executions` 单次执行列表
2. 检查 "时长" 列是否有数据显示
3. 对比 `/pipeline?tab=executions&executionView=batch` 批量任务列表（应该都显示时长）
