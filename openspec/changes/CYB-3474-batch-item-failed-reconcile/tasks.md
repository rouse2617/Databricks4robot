# Tasks

- [ ] `syncJobProgressInternal`: 新增有界 failed-item reconcile pass（捞 job 的 `failed` item，上限 `maxFailedReconcilePerSync`，默认 50）
- [ ] 对每个 failed item 的 run 走既有 `reconcileMisclassifiedRunFromArgo` 路径;命中 Argo 非失败 → 经 `syncBackfillItemStatusFromRun` 回写 item
- [ ] 有界性:一次最多处理 N 个,剩余留待下次 sync 收敛(log 丢弃计数,不静默截断)
- [ ] 单测:failed + Argo=Succeeded → completed
- [ ] 单测:failed + Argo=Failed → 保持 failed(不误纠)
- [ ] 单测:failed + Argo=Running → running
- [ ] 单测:failed item 数 > N 时只处理 N 个,且不影响既有 running/pending 路径
- [ ] 回归:现有 `syncJobProgress` / `SummarizeItemStatuses` 相关测试全绿
- [ ] dev 部署后验证:job d655b177 的 `failedCount` 收敛到 ≈ Argo 真实 Failed 数(~29),不再虚高
