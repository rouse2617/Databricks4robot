# Prod deploy spec — CYB-4427

## backend-prod.sh contract

- Runs the SAME logic as `backend-dev.sh` (it `exec`s it) with prod overrides.
- Requires `IMAGE` when `USE_EXISTING_IMAGE=true` (default) — errors otherwise.
- `SOURCE_K8S_ENV=false`: no in-cluster ConfigMap/Secret merge (prod has none).
- Resolved runtime env MUST equal the live `cyber-databrew-backend-prod` env for
  the keys prod already had, plus the dev-parity keys prod was missing. It MUST
  NOT introduce dev-only settings that don't apply to prod (Grace sync,
  dev bearer/webhook-token secrets) — those are disabled via empty overrides.
- Secret bindings MUST be exactly: `DATABREW_TOKEN`, `DB_PASSWORD`,
  `K8S_CA_DATA`, `K8S_CA_B64`, `COMPONENT_RELEASE_INGEST_TOKEN`,
  `BACKFILL_NOTIFY_FEISHU_WEBHOOK_URL` (matches the pre-parity `--set-secrets`).
- Runtime SA pinned to `cyber-databrew-cloudrun-prod@…` (the CRD/RBAC identity).

## backend-dev.sh hooks (must stay default-noop for dev)

- `EXTRA_ENV_VARS="K=V,…"` upserted into the deploy env before the Cloud Run fix.
- `EXTRA_SECRET_MAPPINGS="K=SECRET:VER,…"` appended to `--set-secrets`, plain key
  of the same name dropped, applied BEFORE the env-vars-file JSON is generated.
- `SERVICE_ACCOUNT` → `--service-account` when non-empty; empty keeps existing.
- `DRY_RUN=true` prints resolved env + secret bindings + gcloud args, exits 0.
- The five disable-able vars use `${V-default}` (unset→default, empty→empty).

## Empty ⇒ default is unchanged for dev

Because dev CI (`deploy-dev.yml`) and local runs never set the five switched
vars, `${V-default}` yields the default exactly as `${V:-default}` did. Only an
explicit empty export (prod) now disables the feature.

## prod ES manifest

- `kubectl apply -k deploy/k8s/prod-deps/` deploys a single-node ES StatefulSet
  (8.13.4, no auth, 512m heap, 30Gi PVC) + headless Service + internal
  LoadBalancer Service into `cyber-databrew-prod`. The ILB serves 10.2.0.33
  (what `backend-prod.sh` sets as `ELASTICSEARCH_URL`).
- No Postgres (prod DB is CloudSQL), unlike `dev-deps`.
