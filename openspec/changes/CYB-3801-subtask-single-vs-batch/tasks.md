# CYB-3801 Tasks

- [x] Linear CYB-3801（child of CYB-3778）
- [x] OpenSpec proposal + design + spec delta
- [x] `pubsub.go`: `extractAssetIDs`（`asset_ids` 数组 + 预留 `topic`）；`Pull` 返回按消息分组 `Messages [][]string`
- [x] `subtask/usecase.go`: `RunCreator` 接口；`executeTask` 逐消息 run(1)/批次(>1)
- [x] `subtask/wiring.go` + `core.go`: 接 `CreateRunByTemplateID`（owner + target + version）
- [x] `pipeline`: `PipelineRunListFilter.CreatedBy` + `ListSummaries` WHERE owner；handler 读 `createdBy`
- [x] 无 subtask 测试 mock 需改（该包无 test 文件）
- [x] 契约同步：openapi（`/runs` + 别名 createdBy）、api-guide、smoke、integration 文档消息格式
- [x] 前端：`subscriptionTaskApi` 加 `listSubscriptionTaskRuns`；抽屉合并 run + 批次
- [x] Tier L（`go test ./...` 57 ok · tsc 0 · `npm run build` ✓）
- [ ] Dev 部署 + Chrome DevTools 验证（单条→run、多条→批次、历史合并）
- [ ] PR（Linear + OpenSpec change-id + 模板）
