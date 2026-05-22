# Tasks — CYB-1020

- [x] OpenSpec proposal + design + spec
- [x] Migration `032_delivery_rules.sql`
- [x] Models + repo + `deliveryrules` evaluator
- [x] `POST/GET /delivery-rules` handlers
- [x] Integrate check in `POST /deliveries`
- [x] Unit tests + `go test` (deliveryrules, routes, server)
- [x] OpenAPI + api-guide + `scripts/smoke-delivery-rules-dev.sh`
- [x] Deploy dev + smoke + frontend regression (Chrome)

## Deploy record — CYB-1020

| Service | Image tag | Cloud Run revision | URL |
|---------|-----------|-------------------|-----|
| backend-dev | `cyber-databrew-backend:a64a76c` | `cyber-databrew-backend-dev-00154-wqd` → redeploy `a64a76c` | https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app |
| frontend-dev | `cyber-databrew-frontend:60ed161` | `cyber-databrew-frontend-dev-00187-hvg` | https://cyber-databrew-frontend-dev-wtttm6suaq-uc.a.run.app |

Smoke: `ASSET_ID=9KnuP7F3 bash scripts/smoke-delivery-rules-dev.sh` — 5/5 OK

- [ ] PR → `dev`; Linear Done
