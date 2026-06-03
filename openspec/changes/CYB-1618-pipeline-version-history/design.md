# Design — CYB-1618 Pipeline Algorithm Iteration UX

## Research Summary

Researched Prefect, Dagster, MLflow, Kubeflow, Argo Workflows, Jenkins.

Key patterns adopted:
- **Prefect**: auto-versioning, per-run metadata, one-click rollback
- **Dagster**: deployment history modal, collapse similar entries
- **MLflow**: side-by-side run comparison, parallel coordinates plots
- **Argo Rollouts**: canary/A/B deployment with automated metric analysis

Sources:
- https://docs.prefect.io/v3/how-to-guides/deployments/versioning
- https://docs.dagster.io/guides/build/projects/dagster-plus-project-history
- https://www.databricks.com/blog/accelerate-your-model-development-new-mlflow-experiments-ui
- https://blog.argoproj.io/whats-new-in-argo-workflows-v3-5-f260e8603ca6

---

## Capability 1: Version History + Execution Filter (P0)

### Component Tree

```
PipelineListPage
├── PipelineCard
│   └── VersionBadge (clickable v{N} chip) → opens VersionHistoryDrawer
│
VersionHistoryDrawer
├── Header: "{name} 版本历史"
├── Timeline: version entries (version, nodeCount, createdAt)
└── Rollback button (Phase 4)
│
WorkflowExecutionList
└── FilterBar
    └── VersionFilter (Select dropdown)
```

### New Endpoint

```
GET /api/v1/pipelines/:name/versions
→ { versions: [{ version: 2, nodeCount: 1, createdAt: "...", updatedAt: "..." }, ...] }
```

---

## Capability 2: Run Parameter & Metric Recording (P0)

### Data Model

Extend `PipelineDeployment` to carry structured run metadata:

```json
{
  "id": "...",
  "templateId": "...",
  "templateVersion": 2,
  "workflowName": "my-pipeline-abc123",
  "status": "Succeeded",
  "params": {
    "input_path": "s3://bucket/data/v3",
    "threshold": 0.85,
    "batch_size": 32
  },
  "metrics": {
    "duration": "43s",
    "cost": "$0.0003",
    "cpuSeconds": 1,
    "memorySeconds": 31,
    "nodeCount": 2
  },
  "createdAt": "2026-06-03T07:26:11Z",
  "finishedAt": "2026-06-03T07:26:54Z"
}
```

### UI: Workflow Detail → Metadata Card

```
┌─ 运行参数 ──────────────────────────────┐
│ 模板版本  v2                             │
│ input_path  s3://bucket/data/v3          │
│ threshold   0.85                          │
│ batch_size  32                            │
├─ 运行指标 ──────────────────────────────┤
│ 耗时       43s                            │
│ 成本       $0.0003                        │
│ CPU        1 core·s                       │
│ Memory     31 MB·s                        │
│ 状态       Succeeded                      │
└──────────────────────────────────────────┘
```

---

## Capability 3: Multi-Run Comparison (P1)

### Interaction

1. Execution list → checkboxes appear on each row
2. Select 2-3 runs → "Compare selected" button lights up
3. Click → side-by-side panel slides up

### UI: Run Comparison Panel

```
┌─ Run Comparison ──────────────────────────────────────────┐
│                  │ run-abc (v2)  │ run-def (v1)  │ run-ghi (v2) │
├──────────────────┼───────────────┼───────────────┼──────────────┤
│ Status           │ Succeeded     │ Succeeded     │ Failed       │
│ Version          │ v2            │ v1            │ v2           │
│ Duration         │ 43s           │ 20s           │ 10s          │
│ Cost             │ $0.0003       │ $0.0003       │ $0.0010      │
│ CPU              │ 1 core·s      │ 0.5 core·s    │ 0.3 core·s   │
│ Memory           │ 31 MB·s       │ 15 MB·s       │ 10 MB·s      │
│ threshold        │ 0.85          │ 0.80          │ 0.85         │
│ batch_size       │ 32            │ 32            │ 64           │
└──────────────────┴───────────────┴───────────────┴──────────────┘
```

Green/red highlights for best/worst values per row.

---

## Capability 4: One-Click Rollback (P1)

### Endpoint

```
POST /api/v1/pipelines/:name/rollback
Body: { "version": 14 }
→ { "message": "ok", "activeVersion": 14 }
```

### UI Flow

```
VersionHistoryDrawer
  v16 (current)  [● active]
  v15
  v14  [Rollback to v14]
  ...

  Click "Rollback to v14"
  → Confirmation modal: "Roll back my-pipeline to v14? Next run will use v14 config."
  → Confirm
  → Toast: "Rolled back to v14"
  → Drawer updates: v14 now shows [● active]
```

### Implementation Note

Rollback creates a new `PipelineDeployment` pointing to the old template version. It does NOT delete the newer version — just sets the "active" version for future runs.

---

## Full User Journey

```
1. 打开流水线管理页
   → 看到 my-pipeline [v16 ▾]
   → 点击 → 版本历史抽屉打开
   → 看到 v1...v16 的时间线

2. 切到执行记录
   → 筛选 "模板版本 = v15"
   → 看到 v15 的全部运行记录
   → 勾选 3 条 → 点 "对比选中"
   → 并排看到参数、耗时、成本对比

3. 发现 v16 有问题
   → 回到版本历史抽屉
   → 点 v15 旁边的 "回滚"
   → 确认 → 下次运行用 v15 配置
```
