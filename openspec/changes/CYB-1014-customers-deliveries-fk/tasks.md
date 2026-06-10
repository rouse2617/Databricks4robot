# Tasks — CYB-1014

- [x] OpenSpec proposal + tasks
- [x] Migration `029_customers.sql` + backfill + FK (dev PG: 50 placeholders + `fk_deliveries_customer`)
- [x] Customer model + repository + postgres
- [x] Customer handler POST/GET/PATCH
- [x] Delivery List `customer_id` filter + commit FK guard
- [x] Tests
- [x] Deploy dev + API smoke (before commit approval)
- [x] API contract sync: `api/openapi.yaml`, `api-guide.md`, `api-guide-smoke.sh`, spec delta (SDK deferred)

## Dev verification (2026-05-22)

| Check | Result |
|-------|--------|
| Cloud SQL `cyber-databrew-pg-dev` / `029_customers.sql` | PASS — `customers` table, 50 backfill rows, FK |
| Cloud Run `cyber-databrew-backend-dev-00144-xhn` image `0b0b046-cyb1014` | PASS |
| `POST /api/v1/customers` | 201 |
| `GET /api/v1/customers/:id` | 200 |
| `POST /api/v1/deliveries` unknown customer | 422 |
| `POST /api/v1/deliveries` valid customer + asset | 201 |
| `GET /api/v1/deliveries?customer_id=` | 200, filtered total=1 |
