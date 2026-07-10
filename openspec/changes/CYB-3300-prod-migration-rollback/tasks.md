# Tasks — CYB-3300 (design checkpoint)

## Checkpoint（待选型）
- [ ] 确认主线：expand/contract + 破坏性迁移 CI 门禁（方案 1）
- [ ] 是否先做方案 3（prod smoke gate，最小降险）
- [ ] 确认 CloudSQL 备份/PITR 权限（方案 2 兜底）

## 实现（选型后，各自 PR，需二审）
- [ ] 方案 1：db-migrate-lint 破坏性检测设为 main 必需 + 迁移规范文档
- [ ] 方案 3：deploy-prod 流量切换前 smoke gate
- [ ] 方案 2：迁移前 PITR/备份校验
