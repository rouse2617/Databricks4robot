# CYB-4427: prod CI/CD parity — backend-prod.sh + prod ES manifest

## Context

The 2026-07-28 prod CRD cutover exposed that prod's deploy pipeline is far
behind dev's. Dev deploys via the rich, fully env-var-parameterized
`deploy/cloudrun/backend-dev.sh` (full env, secrets, ARGO empty = CRD mode,
`ELASTICSEARCH_URL`, `OUTBOX_*`, K8S Workload Identity, pinned identity). Prod's
`deploy-prod.yml` did a **bare `gcloud run deploy`** setting only
`DEPLOYMENT_VERSION` + `ADMIN_EMAILS` + 6 secrets, and relied on manually-set
env surviving via `--update-env-vars` merge. Prod Elasticsearch was **never
deployed** (ES manifests live in `deploy/k8s/dev-deps/`; no prod equivalent).

Consequence: the CRD flip, ES StatefulSet, index mappings, reindex, RBAC, and a
dozen env vars all had to be hand-patched directly on the live prod service —
none of it in git, all lost on a service rebuild.

## Change (scope 1 + 2)

### 1. `deploy/cloudrun/backend-prod.sh` (new) + additive hooks in backend-dev.sh
- `backend-prod.sh` is a **thin wrapper**: it exports prod-specific overrides and
  `exec`s the shared `backend-dev.sh`, so there is exactly ONE deploy
  implementation and prod cannot drift from dev's logic.
- Prod overrides reproduce the live service: `SOURCE_K8S_ENV=false` (prod has no
  in-cluster ConfigMap), prod DB (`172.27.160.9` / `cyber_databrew_prod`), prod
  ES (`http://10.2.0.33:9200`), `ARGO_SERVER_URL` empty → CRD, prod
  `ARGO_RUN_WEBHOOK_URL`, K8S Workload-Identity token, pinned runtime SA
  (`cyber-databrew-cloudrun-prod`), Grace sync **disabled** (no prod Grace).
- `backend-dev.sh` gains four **additive, default-noop** hooks so the wrapper
  needs no logic duplication: `EXTRA_ENV_VARS`, `EXTRA_SECRET_MAPPINGS`,
  `SERVICE_ACCOUNT`, `DRY_RUN`. Five disable-able defaults switch `${V:-d}` →
  `${V-d}` so an explicit empty override actually disables (dev never sets them,
  so dev is unchanged).
- `deploy-prod.yml` checks out the release ref and calls `backend-prod.sh` in
  place of the bare `gcloud run deploy` (image still prebuilt on the VM).

### 2. `deploy/k8s/prod-deps/` (new) — prod Elasticsearch
- `elasticsearch.yaml` + `elasticsearch-internal-lb.yaml` mirror `dev-deps`,
  `kustomization.yaml` pins `namespace: cyber-databrew-prod`. `kubectl apply -k
  deploy/k8s/prod-deps/` reproduces the ES StatefulSet + internal LB
  (10.2.0.33) deployed by hand on 2026-07-28. No `postgres.yaml` — prod DB is
  CloudSQL, not in-cluster.

## Verification

- `DRY_RUN=true … backend-prod.sh` renders the resolved env + secret bindings and
  exits before deploying. Output matches the live prod service's env + secret
  set exactly, plus the dev-parity vars prod was missing (`FRONTEND_BASE_URL`,
  `PIPELINE_RESOURCE_MAX_*`, `ARGO_WORKFLOW_TTL…`, watcher interval, pricing
  path). No regressions.
- `bash -n` on both scripts; `kubectl kustomize deploy/k8s/prod-deps/`;
  `yaml.safe_load_all` on the workflow + manifests.

## Non-goals (follow-ups)

- Terraform `service-identity/prod` + codifying the hand-applied K8s RBAC
  (workflow email-User subjects, elasticquota ClusterRole, backend-argo KSA/WI).
- Migrating `JWT_SECRET` and `ADMIN_TOKEN` from plaintext env to Secret Manager
  (kept via `preserve_all_cloudrun_env_vars` for now — flagged in-script).
- Bumping prod backend resources (currently 1 vCPU / 512Mi vs dev 4 / 4Gi).
