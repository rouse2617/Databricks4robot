# Tasks — CYB-2835

## Backend
- [ ] DB migration: `ALTER TABLE backfill_jobs ADD COLUMN IF NOT EXISTS created_by TEXT`
- [ ] `models.BackfillJob`: 新增 `CreatedBy string` + `FinishedAt *time.Time`
- [ ] `backfillJobSelectCols` + `scanBackfillJob`: 追加 `created_by, finished_at`
- [ ] `CreateBackfillJob` handler: 从 JWT claims 取 email 写入 `created_by`
- [ ] `api/openapi.yaml`: `BatchJob` schema 追加 `createdBy` + `finishedAt`
- [ ] `docs/review/api-guide.md`: 更新 BatchJob 字段说明

## Frontend
- [ ] `Frontend/src/api/batchJobApi.ts`: `BatchJob` interface 追加 `createdBy?` + `finishedAt?`
- [ ] `Frontend/src/pages/BatchJobList.tsx`: 新增所属用户、完成时间、耗时列

## 验证
- [ ] Tier M：`npm run build` 无错，后端 `go build ./...` 无错
- [ ] Chrome DevTools MCP：列表新列展示正确，历史记录 createdBy 显示 `—`
- [ ] API smoke：`GET /api/v1/batch-jobs` 响应含 `createdBy` + `finishedAt`
