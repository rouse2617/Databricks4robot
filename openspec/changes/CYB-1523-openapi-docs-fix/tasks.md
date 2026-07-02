---
change-id: CYB-1523
slug: openapi-docs-fix
---

# Tasks

## 1. OpenAPI: add missing requestBody

- [x] `POST /api/v1/assets` — add `requestBody` → `$ref: '#/components/schemas/CreateAssetRequest'`
- [x] `POST /api/v1/assets:batch_get` — add `requestBody` → `$ref: '#/components/schemas/BatchGetAssetsRequest'`; fix YAML indent on response `properties`
- [ ] `POST /api/v1/assets/{id}/view` — no requestBody needed (handler doesn't bind body)
- [ ] `POST /api/v1/assets/{id}/favorite` — no requestBody needed (handler doesn't bind body)
- [x] `POST /api/v1/assets/{id}/revisions` — add `requestBody` → `$ref: '#/components/schemas/PromoteRevisionRequest'`

## 2. Docusaurus: show models

- [x] `docs-site/docusaurus.config.ts` — change `hideModels: true` → `hideModels: false`

## 3. Fix base URL inconsistency

- [x] `docs-site/docs/getting-started/authentication.md` — replace `api-cyber-databrew.cyberorigin.ai` → `cyber-databrew.cyberorigin.ai`

## 4. Verify

- [x] Validate OpenAPI file is valid YAML
- [x] Confirm schemas referenced in requestBody exist in components/schemas
