# Tasks — CYB-1218

## Context files
- `backend/migrations/036_assets_mcap_file_id_nullable.sql`
- `backend/migrations/001_init.sql` (assets 建表)

## ⚠️ Off-limits zone
- `backend/migrations/` — user approved

## Implementation
- [ ] `[migration]` 创建 `036_assets_mcap_file_id_nullable.sql`: DROP NOT NULL + ADD CHECK

## Verification
- [ ] `[backend]` 检查所有 INSERT/UPDATE assets 代码路径不受 NULL mcap_file_id 影响
- [ ] `[backend]` Tier S: `make fmt && make vet`
- [ ] `[backend]` Tier L: `go build ./...`
- [ ] `[migration]` 本地或 dev 应用 migration 成功

## Deploy verification
- [ ] 应用 migration 到 dev
- [ ] 部署 backend dev，验证 derived_asset 允许 NULL mcap_file_id
- [ ] 验证非 derived_asset 拒绝 NULL mcap_file_id
