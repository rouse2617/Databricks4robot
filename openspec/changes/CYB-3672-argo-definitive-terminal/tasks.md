# Tasks — CYB-3672

- [x] [backend] `reconcileMisclassifiedRunFromArgo` 加 age-gated early-return(仅当 message 无 diagnostic 且 CreatedAt/StartedAt 非零时生效)
- [x] [backend] 新 helper `isNoisyMisclassifiedMessage`(空消息或已知 stale marker)
- [x] [backend] 新 helper `runHasCreatedTimestamp`(生产 DB rows 保证非零;测试 fixture 常常留零)
- [x] [backend] 单测 `TestReconcileMisclassified_SkipsArgo_WhenPastRevivalAge`(60d 老 + Error + 空消息 → 0 次 GetWorkflow)
- [x] [backend] 单测 `TestReconcileMisclassified_CallsArgo_WithinRevivalAge`(30min 新 + Error + 真诊断消息 → 1 次 GetWorkflow)
- [x] [backend] 现有单测:`Test*_ReconcilesMisclassifiedError` / `TestReconcileMisclassified_RevivesLegacyUnschedulableVerdict` / `TestReconcileTerminalRun*` / 全量 pipeline+backfill 测试全绿

## Dev verification
- [ ] merge 后 dev 部署完成,`kubectl -n cyber-databrew-dev logs deploy/argo-server --since=5m | grep -c ERROR` 显著下降(baseline ~300/5m → 期望 < 60/5m)
- [ ] `kubectl -n cyber-databrew-dev logs deploy/argo-server --since=1m` 中 not-found 的 workflow name 集中在近期(<48h)创建的,不再出现 batch-parent-<UUID> / youxin-<TS>-* 这类 legacy
