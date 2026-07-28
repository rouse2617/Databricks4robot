# CYB-3798 Tasks

- [x] Linear issue CYB-3798（child of CYB-3778）
- [x] OpenSpec proposal + design + spec delta
- [x] Backend repo: `FindAllJobs(ctx, createdBy)` 加 `WHERE created_by`；`FindItemsByJobID` 复用
- [x] Backend usecase: `ListJobs(createdBy)` + `ListItems(jobID)`
- [x] Backend handler/routes: `GET /backfill?createdBy=` + `GET /backfill/:id/items`
- [x] 更新所有 `BackfillRepository` mock（interface 变更规则）
- [x] API contract sync: OpenAPI（含 BackfillItem 补 pipelineRunId/attempts）、api-guide、smoke 脚本
- [x] Frontend: `subscriptionTaskApi.ts`（listSubscriptionTaskBatches / listDispatchBatchItems）
- [x] Frontend: `SubscriptionTasksPanel.tsx` 详情抽屉（历史批次 → 资产）
- [x] Tier L 验证（`go test ./...` 57 ok · `npm run build` ✓ · tsc 0）
- [ ] Dev 部署 + Chrome DevTools MCP UI 验证（diff 触碰 Frontend）
- [ ] PR（含 Linear ID + OpenSpec change-id + 模板）
