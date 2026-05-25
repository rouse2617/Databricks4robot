# Tasks — CYB-1220

## Context files
- `backend/migrations/035_asset_relations_check_constraint.sql`
- `backend/migrations/001_init.sql` (asset_relations 建表)

## ⚠️ Off-limits zone
- `backend/migrations/` — user approved

## Implementation
- [ ] `[migration]` 创建 `035_asset_relations_check_constraint.sql`: ADD CONSTRAINT chk_relation_type

## Verification
- [ ] `[migration]` 确认现网 `SELECT DISTINCT relation_type FROM asset_relations` 无脏值
- [ ] `[migration]` 本地或 dev 应用 migration 成功
- [ ] `[backend]` Tier S: `make fmt && make vet`
- [ ] `[backend]` Tier L: `go build ./...`

## Deploy verification
- [ ] 应用 migration 到 dev
- [ ] 部署 backend dev，验证非法 relation_type INSERT 被拒绝
