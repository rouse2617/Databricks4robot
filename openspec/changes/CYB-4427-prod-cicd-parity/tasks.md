# CYB-4427 tasks

## backend-dev.sh (additive, dev-safe)
- [x] Add `EXTRA_ENV_VARS` hook (comma-separated KEY=VALUE upsert)
- [x] Add `EXTRA_SECRET_MAPPINGS` hook (comma-separated KEY=SECRET:VER, applied before env-vars-file JSON gen)
- [x] Add `SERVICE_ACCOUNT` hook (empty → keep existing SA)
- [x] Add `DRY_RUN` hook (render env + args, exit before gcloud)
- [x] Switch 5 disable-able defaults `${V:-d}` → `${V-d}` (GRACE_API_URL/USERNAME, GRACE_PASSWORD_SECRET, ARGO_RUN_WEBHOOK_TOKEN_SECRET, K8S_BEARER_TOKEN_SECRET)

## backend-prod.sh (new)
- [x] Prod overrides reproducing the live service (DB, ES, ARGO=CRD, K8S WI, SA, grace disabled, DEPLOYMENT_VERSION passthrough)
- [x] EXTRA_SECRET_MAPPINGS for prod-only secrets (DATABREW_TOKEN, COMPONENT_RELEASE_INGEST_TOKEN)
- [x] SECURITY TODO comment for JWT_SECRET / ADMIN_TOKEN → Secret Manager
- [x] `exec backend-dev.sh`; `chmod +x`

## deploy-prod.yml
- [x] Add checkout-release-ref step (runner needs the script)
- [x] Replace bare `gcloud run deploy` with `bash deploy/cloudrun/backend-prod.sh`

## deploy/k8s/prod-deps (new)
- [x] elasticsearch.yaml (mirror dev-deps, no namespace)
- [x] elasticsearch-internal-lb.yaml (mirror dev-deps)
- [x] kustomization.yaml (namespace: cyber-databrew-prod, ES only — no postgres)

## Verify
- [x] `bash -n` both scripts
- [x] DRY_RUN render diff vs live prod (reproduces env + secrets; no regression)
- [x] `kubectl kustomize deploy/k8s/prod-deps/`
- [x] YAML load workflow + manifests

## Ship
- [ ] PR → dev; review; merge → next prod release exercises backend-prod.sh
- [ ] First prod deploy after merge: confirm the service env unchanged (DRY_RUN parity holds live)
