# CYB-1046,1094,1112 — Quick Wins

## What to build

三条独立的小改动：
1. GET /customers list with filters (CYB-1046)
2. asset_usage_stats 表 (CYB-1094)
3. README gap table 刷新 (CYB-1112)

## Acceptance criteria

- [ ] GET /api/v1/customers 支持 status/sla_tier/region 分页
- [ ] asset_usage_stats 表 migration + model
- [ ] UAC README gap table 更新为 dev 现状
- [ ] OpenAPI + smoke (customers)
- [ ] 单元测试
