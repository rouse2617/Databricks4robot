# Tasks — CYB-3297 Phase D
- [x] createFileTx 发 asset_created（AssetID 设值）事件
- [x] rawmcap_index_test.go 断言 asset_created + AssetID
- [x] go test ./internal/handlers/mcap/ 通过；go build ./... OK
- [ ] 合 dev + 部署 + dev 验证（新建 mcap → 立即可搜）
