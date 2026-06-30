# CYB-2832 — 批量任务列表搜索与状态筛选

## 问题

`BatchJobList` 没有搜索控件，而流水线管理和单次执行均有名称搜索 + 状态筛选。
任务增多后用户无法快速定位目标批次。

## 方案

客户端筛选（`listBatchJobs` 全量返回，后端无过滤参数）：

1. `Input.Search` — 模糊匹配 `job.name`（大小写不敏感）
2. `Select` 状态筛选 — running / paused / completed / failed / all

## 范围

- 仅修改 `Frontend/src/pages/BatchJobList.tsx`
- 无新 HTTP API，无 Backend 变更
