## 1. Implementation
- [x] Migration `052_pipeline_batch_run_id.sql`
- [x] `BatchCreateRunsByTemplateID` usecase + handler
- [x] Frontend `batchDeployTemplate` + deploy modal copy
- [x] Unit test `TestBatchCreateRunsByTemplateID`

## 2. Verification
- [ ] Dev: select 2+ assets → deploy → N workflows in executions tab
- [ ] API: runs share `batchRunId`
- [ ] Partial failure returns `failed[]` with 201 when any succeed
