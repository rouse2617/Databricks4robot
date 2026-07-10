# Tasks — CYB-3298

- [ ] deploy-dev.yml backend build: 加 `--no-cache-filter=builder`
- [ ] deploy-dev.yml backend build: 加 `--build-arg COMMIT=$COMMIT_SHA` / `VERSION` / `BUILD_TIME`
- [ ] PR → 合 dev（合并即触发 deploy-dev）
- [ ] 验证 `/version` 显示真实 commit（非 unknown）
- [ ] 验证 CYB-3297-C 生效：触发 asset 事件后 doc 有 mcap.device_id；`vendor_agg`/`scene_agg` 出现非空桶（新事件/reindex 后）
- [ ] 跟进：deploy-prod.yml 同类修复（另立）
