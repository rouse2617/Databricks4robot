# Tasks — CYB-3297

## Phase C（本 PR）
- [ ] builder.go: mcap.* 子字段非空时写入
- [ ] builder_test.go: 断言 mcap.vendor_id/device_id/scene_id 等被 emit
- [ ] `go test`（CGO_ENABLED=0）通过
- [ ] PR → 合 dev → 部署
- [ ] dev: 跑 `/admin/search/reindex` 回填
- [ ] dev 验证: `GET /search/assets` 的 vendor_agg/scene_agg 出现非空桶；按 vendor_id 过滤能命中

## 后续阶段
- [ ] A: tag.* 统一
- [ ] B: env 可发现
- [ ] D: raw_mcap 实时入索引
