#!/usr/bin/env bash
set -euo pipefail

# backend-prod.sh — deploy the PROD backend to Cloud Run.
#
# This is a thin wrapper over backend-dev.sh: it exports the prod-specific
# configuration as environment overrides and then execs the shared script, so
# there is exactly ONE deploy implementation and prod cannot silently drift
# from dev's logic. Every value below was reverse-engineered from the live
# cyber-databrew-backend-prod service (2026-07-28) so a re-deploy reproduces the
# working prod state instead of the pre-parity bare `gcloud run deploy`.
#
# Prod has NO Kubernetes ConfigMap/Secret to source env from (unlike dev's
# cyber-databrew-config / cyber-databrew-secrets), so SOURCE_K8S_ENV=false and
# every setting is provided explicitly here or preserved from the running
# service by backend-dev.sh's preserve_all_cloudrun_env_vars.
#
# Usage (CI passes the release-tagged image, already built + pushed):
#   USE_EXISTING_IMAGE=true IMAGE=<repo>/cyber-databrew-backend:vX.Y.Z \
#     bash deploy/cloudrun/backend-prod.sh
#
# ── SECURITY TODO (tracked for follow-up) ────────────────────────────────────
# The live prod service carries JWT_SECRET and ADMIN_TOKEN as PLAIN env vars.
# They are kept across deploys only by preserve_all_cloudrun_env_vars (not set
# here, to avoid committing secrets to git). They SHOULD become Secret Manager
# entries (cyber-databrew-prod-jwt-secret / cyber-databrew-prod-admin-token) and
# be bound via EXTRA_SECRET_MAPPINGS below. ADMIN_TOKEN is currently the
# throwaway value "tmp-reindex-001" and must be rotated when it becomes a secret.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# ── Service identity ─────────────────────────────────────────────────────────
export SERVICE_NAME="${SERVICE_NAME:-cyber-databrew-backend-prod}"
export PROJECT_ID="${PROJECT_ID:-green-valley-442103}"
export REGION="${REGION:-us-central1}"
# Prod never builds in this script — CI builds + pushes the release-tagged image
# on the build VM and passes it in. Fail loudly if it is missing.
export USE_EXISTING_IMAGE="${USE_EXISTING_IMAGE:-true}"
if [[ "${USE_EXISTING_IMAGE}" == "true" && -z "${IMAGE:-}" ]]; then
  echo "ERROR: IMAGE must be set (release-tagged prod backend image) when USE_EXISTING_IMAGE=true." >&2
  exit 1
fi

# Prod has no in-cluster ConfigMap/Secret to merge from.
export SOURCE_K8S_ENV="${SOURCE_K8S_ENV:-false}"

# Pin the runtime SA — this identity is what the K8s/Argo CRD client + the
# elasticquota/workflow RBAC bindings are granted to (CYB prod CRD cutover).
export SERVICE_ACCOUNT="${SERVICE_ACCOUNT:-cyber-databrew-cloudrun-prod@green-valley-442103.iam.gserviceaccount.com}"

# ── Sizing (matches live prod; smaller than dev's 4/4Gi — see PR notes) ───────
# NOTE: prod currently runs 1 vCPU / 512Mi. The backend runs the batch
# submitter + run-status watcher + in-process ES subscriber, so this is tight;
# bumping toward dev's 4/4Gi is a follow-up decision, not done here to avoid an
# unrequested cost/behavior change.
export CPU="${CPU:-1}"
export MEMORY="${MEMORY:-512Mi}"
# min-instances=1 keeps the in-process background loops (submitter / watcher /
# ES subscriber) running when idle; scale-to-zero would freeze them.
export MIN_INSTANCES="${MIN_INSTANCES:-1}"
export MAX_INSTANCES="${MAX_INSTANCES:-10}"
# Prod runs a single container (no GMP sidecar today — keeps it at 1 vCPU).
export ENABLE_GMP_SIDECAR="${ENABLE_GMP_SIDECAR:-false}"

# ── Database (prod CloudSQL) ─────────────────────────────────────────────────
export DB_HOST_OVERRIDE="${DB_HOST_OVERRIDE:-172.27.160.9}"
export DB_NAME_OVERRIDE="${DB_NAME_OVERRIDE:-cyber_databrew_prod}"
export DB_PASSWORD_SECRET="${DB_PASSWORD_SECRET:-cyber-databrew-prod-postgres-password}"

# ── Elasticsearch (prod ES StatefulSet via internal LB 10.2.0.33) ────────────
export ELASTICSEARCH_URL_OVERRIDE="${ELASTICSEARCH_URL_OVERRIDE:-http://10.2.0.33:9200}"
export CLOUDRUN_ELASTICSEARCH_URL="${CLOUDRUN_ELASTICSEARCH_URL:-http://10.2.0.33:9200}"
# Prod ES has no HTTP auth (xpack.security disabled), same as dev.
export ELASTICSEARCH_PASSWORD_SECRET="${ELASTICSEARCH_PASSWORD_SECRET:-}"

# ── Argo: CRD mode (leave ARGO_SERVER_URL empty, same as dev) ────────────────
# Empty ARGO_SERVER_URL → factory.go default-cluster fallback resolves to CRD;
# the backend creates Workflow CRs directly via the K8s API (preserves labels).
export ARGO_SERVER_URL_OVERRIDE="${ARGO_SERVER_URL_OVERRIDE:-}"
export ARGO_RUN_WEBHOOK_URL_OVERRIDE="${ARGO_RUN_WEBHOOK_URL_OVERRIDE:-https://cyber-databrew-backend-prod-wtttm6suaq-uc.a.run.app/api/v1/pipeline-runs/webhook}"
# Prod has no run-webhook token secret today (anonymous webhook). Do not bind one.
export ARGO_RUN_WEBHOOK_TOKEN_SECRET="${ARGO_RUN_WEBHOOK_TOKEN_SECRET:-}"

# ── Kubernetes client (same cyber-clust cluster; Workload Identity token) ────
export K8S_API_ENDPOINT_OVERRIDE="${K8S_API_ENDPOINT_OVERRIDE:-https://34.59.48.233}"
export K8S_AUDIENCE_OVERRIDE="${K8S_AUDIENCE_OVERRIDE:-https://34.59.48.233}"
export K8S_USE_METADATA_TOKEN_OVERRIDE="${K8S_USE_METADATA_TOKEN_OVERRIDE:-true}"
# Prod binds the CA secret AND runs insecure-skip-verify=true (matches live).
# The CA secret is shared (not env-scoped). The override below wins over the
# false that backend-dev.sh sets when binding the CA secret.
export K8S_CA_DATA_SECRET="${K8S_CA_DATA_SECRET:-cyber-databrew-k8s-ca-cert}"
export K8S_INSECURE_SKIP_VERIFY_OVERRIDE="${K8S_INSECURE_SKIP_VERIFY_OVERRIDE:-true}"
# Prod uses the metadata token, not a bearer-token secret.
export K8S_BEARER_TOKEN_SECRET="${K8S_BEARER_TOKEN_SECRET:-}"

# ── Grace video-duration sync: DISABLED on prod (no prod Grace endpoint) ──────
export GRACE_API_URL_OVERRIDE="${GRACE_API_URL_OVERRIDE:-}"
export GRACE_USERNAME_OVERRIDE="${GRACE_USERNAME_OVERRIDE:-}"
export GRACE_PASSWORD_SECRET="${GRACE_PASSWORD_SECRET:-}"

# ── Public URLs / roles ──────────────────────────────────────────────────────
export FRONTEND_BASE_URL_OVERRIDE="${FRONTEND_BASE_URL_OVERRIDE:-https://cyber-databrew.cyberorigin.ai}"
export ADMIN_EMAILS_OVERRIDE="${ADMIN_EMAILS_OVERRIDE:-ruipeng.huang@cyberorigin.ai}"
# Prod reuses the dev-owned Feishu webhook secret (cross-env by intent, same as
# the pre-parity deploy-prod.yml) until the envs get split webhooks.
export BACKFILL_NOTIFY_FEISHU_WEBHOOK_SECRET="${BACKFILL_NOTIFY_FEISHU_WEBHOOK_SECRET:-cyber-databrew-dev-backfill-feishu-webhook}"

# ── Pipeline runtime (prod target scoping) ───────────────────────────────────
export PIPELINE_RUNTIME_SECRET_TARGET_IDS_OVERRIDE="${PIPELINE_RUNTIME_SECRET_TARGET_IDS_OVERRIDE:-default,video-proc-prod}"

# Do NOT set PIPELINE_RESOURCE_MAX_* on prod. The live service has no resource
# ceilings; dev's (8 CPU / 28Gi / 1 GPU / 250Gi) would REJECT any prod pipeline
# component requesting more — a real behavior change caught by the controlled
# backend-prod.sh test on 2026-07-28. Adopting prod-appropriate guards is a
# follow-up decision. Empty override + backend-dev.sh's `-` default = unset.
export PIPELINE_RESOURCE_MAX_CPU_OVERRIDE="${PIPELINE_RESOURCE_MAX_CPU_OVERRIDE:-}"
export PIPELINE_RESOURCE_MAX_MEMORY_OVERRIDE="${PIPELINE_RESOURCE_MAX_MEMORY_OVERRIDE:-}"
export PIPELINE_RESOURCE_MAX_DISK_OVERRIDE="${PIPELINE_RESOURCE_MAX_DISK_OVERRIDE:-}"
export PIPELINE_RESOURCE_MAX_GPU_OVERRIDE="${PIPELINE_RESOURCE_MAX_GPU_OVERRIDE:-}"
# PIPELINE_RUNTIME_SECRET_RESOURCES_JSON is left unset so backend-dev.sh
# preserves the value already on the live service.

# ── Prod-only plain env with no named override in backend-dev.sh ─────────────
# GCS_DERIVED_BUCKET is a plain prod config var; DEPLOYMENT_VERSION is the
# release tag CI passes in (used for the /version banner). Both go through the
# generic EXTRA_ENV_VARS passthrough (comma-separated KEY=VALUE).
_prod_extra_env="GCS_DERIVED_BUCKET=grace-derived-green-valley-442103"
if [[ -n "${DEPLOYMENT_VERSION:-}" ]]; then
  _prod_extra_env="${_prod_extra_env},DEPLOYMENT_VERSION=${DEPLOYMENT_VERSION}"
fi
export EXTRA_ENV_VARS="${EXTRA_ENV_VARS:-${_prod_extra_env}}"

# ── Prod-only secret-backed env with no dedicated *_SECRET hook ──────────────
# DATABREW_TOKEN and COMPONENT_RELEASE_INGEST_TOKEN are Secret Manager refs on
# prod (dev sources them from the K8s Secret merge). Bind them here.
# When JWT_SECRET / ADMIN_TOKEN are migrated to Secret Manager, add them here:
#   JWT_SECRET=cyber-databrew-prod-jwt-secret:latest,ADMIN_TOKEN=cyber-databrew-prod-admin-token:latest
export EXTRA_SECRET_MAPPINGS="${EXTRA_SECRET_MAPPINGS:-DATABREW_TOKEN=cyber-databrew-prod-databrew-token:latest,COMPONENT_RELEASE_INGEST_TOKEN=cyber-databrew-prod-component-release-ingest-token:latest}"
# DATABREW_TOKEN is bound as a secret above — make sure it is not also set plain.
export DATABREW_TOKEN_OVERRIDE="${DATABREW_TOKEN_OVERRIDE:-}"

echo "==> backend-prod.sh: deploying ${SERVICE_NAME} (image=${IMAGE:-<build>}) via backend-dev.sh"
exec bash "${SCRIPT_DIR}/backend-dev.sh"
