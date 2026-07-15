# Tasks

- [x] `ClaimNextItem`: 候选集纳入卡死的 placeholder 孤儿(uid='' + `-batch-` 名 + run.created_at 超 5min),`FOR UPDATE SKIP LOCKED` 不变,排序 pending 优先
- [x] `FindIncompleteJobs`: 同谓词把"有孤儿的 job"算作未完成,使 boot 的 ResumeIncompleteBatches 兜底 drain
- [x] 孤儿判据 key 在 `pipeline_runs.created_at`(稳定)而非 `backfill_items.started_at`(被振荡刷新)
- [x] 集成测试:pending 优先认领 / 孤儿被回收 / 真 running(有 uid)不被误认领 / 新建 run(<5min)不被误认领 / deploy 后(uid 置位)不再认领
- [x] 编译 + vet(含 integration tag)+ 非集成全量回归
- [x] 集成测试真库(postgres:16)通过
- [x] 新 WHERE 谓词在真 dev 数据上只读验证(命中 12 孤儿、排除 9 真 running)
- [ ] 部署到 dev 后验证:job d655b177 残留孤儿被 boot/resume 自动认领+投递,batch 收敛 settle
