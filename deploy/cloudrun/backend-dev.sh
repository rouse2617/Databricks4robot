#!/usr/bin/env bash
set -euo pipefail

# Docker push: ensure docker-credential-gcloud is on PATH (same directory as gcloud), e.g.
#   PATH="$(dirname "$(which gcloud)"):$PATH"
# Cloud Run already uses Secret Manager for DB_PASSWORD / ELASTICSEARCH_PASSWORD: pass
#   DB_PASSWORD_SECRET / ELASTICSEARCH_PASSWORD_SECRET
# when merging k8s env would overwrite those bindings or disable OUTBOX.
#
# K8s ConfigMap often sets ENV=development, ELASTICSEARCH_URL=http://elasticsearch:9200, OUTBOX_*=false.
# By default APPLY_CLOUDRUN_ENV_FIX=true fixes ENV, Elasticsearch URL, and OUTBOX after merge.
# Disable with APPLY_CLOUDRUN_ENV_FIX=false or tune CLOUDRUN_* variables below.

PROJECT_ID="${PROJECT_ID:-green-valley-442103}"
REGION="${REGION:-us-central1}"
SERVICE_NAME="${SERVICE_NAME:-cyber-databrew-backend-dev}"
IMAGE="${IMAGE:-us-central1-docker.pkg.dev/green-valley-442103/cyber-databrew-images/cyber-databrew-backend:cloudrun-dev-latest}"

USE_EXISTING_IMAGE="${USE_EXISTING_IMAGE:-false}"
SOURCE_K8S_ENV="${SOURCE_K8S_ENV:-true}"
K8S_NAMESPACE="${K8S_NAMESPACE:-cyber-databrew-dev}"
K8S_CONFIGMAP_NAME="${K8S_CONFIGMAP_NAME:-cyber-databrew-config}"
K8S_SECRET_NAME="${K8S_SECRET_NAME:-cyber-databrew-secrets}"
ENV_FILE="${ENV_FILE:-}"

# CPU/MEM raised from 1/2Gi (cyb-3491): the dev backend runs the batch
# submitter, the run-status projector, AND every exit-hook webhook on one
# service. Under load-test throughput the single vCPU was the bottleneck —
# dispatch stalled at ~236/min (of a possible ~512) because the submitter's
# per-cycle submits, the watcher, and hundreds of ~2.5s webhooks all contended
# for one core. 4 vCPU / 4Gi gives them room. Still env-overridable.
CPU="${CPU:-4}"
MEMORY="${MEMORY:-4Gi}"
# Keep one instance always warm. The batch submitter and the run-status
# projector are in-process background loops (15s / 30s tickers), not
# request-driven. With min-instances=0 Cloud Run scales to zero when idle and
# those loops STOP: batch dispatch and status projection freeze until the next
# HTTP request happens to wake an instance. That is exactly what stalled the
# CYB-3489 executor on dev — a batch would only advance while someone kept the
# page open. min-instances=1 makes the loops run continuously. (cyb-3491)
MIN_INSTANCES="${MIN_INSTANCES:-1}"
# Raised 5 → 30 to give the dev backend more headroom under batch-dispatch load
# (user request 2026-07-17). Still overridable via the MAX_INSTANCES env.
MAX_INSTANCES="${MAX_INSTANCES:-30}"
TIMEOUT="${TIMEOUT:-60}"
CPU_THROTTLING="${CPU_THROTTLING:-false}"
CPU_BOOST="${CPU_BOOST:-true}"
ALLOW_UNAUTHENTICATED="${ALLOW_UNAUTHENTICATED:-true}"

VPC_CONNECTOR="${VPC_CONNECTOR:-cr-central-conn}"
VPC_EGRESS="${VPC_EGRESS:-private-ranges-only}"

# Optional overrides for Cloud Run reachability.
DB_HOST_OVERRIDE="${DB_HOST_OVERRIDE:-172.27.160.7}"
DB_PORT_OVERRIDE="${DB_PORT_OVERRIDE:-5432}"
DB_USER_OVERRIDE="${DB_USER_OVERRIDE:-postgres}"
DB_PASSWORD_OVERRIDE="${DB_PASSWORD_OVERRIDE:-}"
DB_PASSWORD_SECRET="${DB_PASSWORD_SECRET:-cyber-databrew-dev-postgres-password}"
DB_PASSWORD_SECRET_VERSION="${DB_PASSWORD_SECRET_VERSION:-latest}"
DB_NAME_OVERRIDE="${DB_NAME_OVERRIDE:-cyber_databrew_dev}"
DATABREW_TOKEN_OVERRIDE="${DATABREW_TOKEN_OVERRIDE:-}"
# Argo Workflows server lives in the dev K8s cluster, not on Cloud Run.
# The Cloud Run service cyber-databrew-pipeline-ui-dev is a separate UI proxy
# (SSO/OIDC) and rejects K8s SA tokens with "unexpected signing method: RS256".
ARGO_SERVER_URL_OVERRIDE="${ARGO_SERVER_URL_OVERRIDE:-http://10.2.1.211:2746}"
# Completed Argo workflow CR retention (secondsAfterCompletion). Default 30 days.
ARGO_WORKFLOW_TTL_SECONDS_AFTER_COMPLETION_OVERRIDE="${ARGO_WORKFLOW_TTL_SECONDS_AFTER_COMPLETION_OVERRIDE:-2592000}"
# Argo run status push webhook (CYB-3058). ENABLED: the container curl exit hook
# is injected into every backend-initiated workflow and pokes this URL on
# terminal phase; DataBrew re-reads authoritative state from Argo. Validated
# end-to-end on dev (real run → HTTP 200 → run status pushed). The token is read
# from Secret Manager (backend) / K8s Secret databrew-run-webhook-token
# (workflow, present in cyber-databrew-dev, video-proc-dev, video-proc-prod).
# Kill switch: set ARGO_RUN_WEBHOOK_URL_OVERRIDE="" to instantly disable the hook.
ARGO_RUN_WEBHOOK_URL_OVERRIDE="${ARGO_RUN_WEBHOOK_URL_OVERRIDE:-https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app/api/v1/pipeline-runs/webhook}"
ARGO_RUN_WEBHOOK_TOKEN_SECRET="${ARGO_RUN_WEBHOOK_TOKEN_SECRET:-cyber-databrew-dev-argo-run-webhook-token}"
ARGO_RUN_WEBHOOK_TOKEN_SECRET_VERSION="${ARGO_RUN_WEBHOOK_TOKEN_SECRET_VERSION:-latest}"
# Batch job completion Feishu notification (CYB-3071). The webhook is a credential,
# so it is injected from Secret Manager (like ARGO_RUN_WEBHOOK_TOKEN) rather than a
# plain env var that --env-vars-file would wipe on every deploy. Empty
# BACKFILL_NOTIFY_FEISHU_WEBHOOK_SECRET disables injection. FRONTEND_BASE_URL builds
# the links in that notification; it is a public URL so it stays plain.
BACKFILL_NOTIFY_FEISHU_WEBHOOK_SECRET="${BACKFILL_NOTIFY_FEISHU_WEBHOOK_SECRET:-cyber-databrew-dev-backfill-feishu-webhook}"
BACKFILL_NOTIFY_FEISHU_WEBHOOK_SECRET_VERSION="${BACKFILL_NOTIFY_FEISHU_WEBHOOK_SECRET_VERSION:-latest}"
FRONTEND_BASE_URL_OVERRIDE="${FRONTEND_BASE_URL_OVERRIDE:-https://cyber-databrew-dev.cyberorigin.ai}"
# ADMIN_EMAILS (CYB-3154): comma-separated emails that get role=admin on web
# email-login, granting the "*" scope set (so they can manage API keys). Plain,
# non-sensitive config; baked in here so --env-vars-file deploys don't wipe it.
ADMIN_EMAILS_OVERRIDE="${ADMIN_EMAILS_OVERRIDE:-ruipeng.huang@cyberorigin.ai}"
# Push is now the primary status signal; the watcher is a low-frequency reconcile
# backstop. Set to 3 to temporarily restore high-frequency polling if needed.
PIPELINE_RUN_WATCHER_INTERVAL_SEC_OVERRIDE="${PIPELINE_RUN_WATCHER_INTERVAL_SEC_OVERRIDE:-30}"
# Grace video-duration sync (CYB-3072). URL + username are non-sensitive env; the
# password is the whole grace-api-dev Secret Manager JSON mounted as GRACE_PASSWORD
# (the backend extracts AUTH_PASSWORD). Empty GRACE_API_URL disables the sync loop.
GRACE_API_URL_OVERRIDE="${GRACE_API_URL_OVERRIDE:-https://dev.cyber-grace.pages.dev/api}"
GRACE_USERNAME_OVERRIDE="${GRACE_USERNAME_OVERRIDE:-grace-service-dev}"
GRACE_PASSWORD_SECRET="${GRACE_PASSWORD_SECRET:-grace-api-dev}"
GRACE_PASSWORD_SECRET_VERSION="${GRACE_PASSWORD_SECRET_VERSION:-latest}"
# Conservative dev execution ceilings. These are backend deploy-time guards for
# user-defined pipeline component resources; execution targets may override via quota_policy.
PIPELINE_RESOURCE_MAX_CPU_OVERRIDE="${PIPELINE_RESOURCE_MAX_CPU_OVERRIDE:-8}"
PIPELINE_RESOURCE_MAX_MEMORY_OVERRIDE="${PIPELINE_RESOURCE_MAX_MEMORY_OVERRIDE:-28Gi}"
PIPELINE_RESOURCE_MAX_DISK_OVERRIDE="${PIPELINE_RESOURCE_MAX_DISK_OVERRIDE:-250Gi}"
PIPELINE_RESOURCE_MAX_GPU_OVERRIDE="${PIPELINE_RESOURCE_MAX_GPU_OVERRIDE:-1}"
PIPELINE_UNSCHEDULABLE_PENDING_THRESHOLD_OVERRIDE="${PIPELINE_UNSCHEDULABLE_PENDING_THRESHOLD_OVERRIDE:-1h}"
K8S_API_ENDPOINT_OVERRIDE="${K8S_API_ENDPOINT_OVERRIDE:-https://34.59.48.233}"
K8S_AUDIENCE_OVERRIDE="${K8S_AUDIENCE_OVERRIDE:-}"
K8S_USE_METADATA_TOKEN_OVERRIDE="${K8S_USE_METADATA_TOKEN_OVERRIDE:-}"
K8S_INSECURE_SKIP_VERIFY_OVERRIDE="${K8S_INSECURE_SKIP_VERIFY_OVERRIDE:-}"
K8S_BEARER_TOKEN_SECRET="${K8S_BEARER_TOKEN_SECRET:-cyber-databrew-dev-k8s-bearer-token}"
K8S_BEARER_TOKEN_SECRET_VERSION="${K8S_BEARER_TOKEN_SECRET_VERSION:-latest}"
K8S_CA_DATA_SECRET="${K8S_CA_DATA_SECRET:-cyber-databrew-dev-k8s-ca-data}"
K8S_CA_DATA_SECRET_VERSION="${K8S_CA_DATA_SECRET_VERSION:-latest}"
LAKEHOUSE_BACKEND_OVERRIDE="${LAKEHOUSE_BACKEND_OVERRIDE:-}"
LAKEHOUSE_BQ_PROJECT_OVERRIDE="${LAKEHOUSE_BQ_PROJECT_OVERRIDE:-}"
LAKEHOUSE_BQ_DATASET_OVERRIDE="${LAKEHOUSE_BQ_DATASET_OVERRIDE:-}"
PUBSUB_PROJECT_OVERRIDE="${PUBSUB_PROJECT_OVERRIDE:-}"
TOPIC_ASSET_EVENTS_OVERRIDE="${TOPIC_ASSET_EVENTS_OVERRIDE:-}"
OUTBOX_ES_SUBSCRIPTION_OVERRIDE="${OUTBOX_ES_SUBSCRIPTION_OVERRIDE:-}"
ELASTICSEARCH_URL_OVERRIDE="${ELASTICSEARCH_URL_OVERRIDE:-}"
ELASTICSEARCH_PASSWORD_SECRET="${ELASTICSEARCH_PASSWORD_SECRET:-}"
ELASTICSEARCH_PASSWORD_SECRET_VERSION="${ELASTICSEARCH_PASSWORD_SECRET_VERSION:-latest}"
ELASTICSEARCH_PASSWORD_OVERRIDE="${ELASTICSEARCH_PASSWORD_OVERRIDE:-}"
PRICING_CONFIG_PATH_OVERRIDE="${PRICING_CONFIG_PATH_OVERRIDE:-/app/config/gcp_pricing.yaml}"
PIPELINE_TEMPLATE_NODE_SELECTOR_JSON_OVERRIDE="${PIPELINE_TEMPLATE_NODE_SELECTOR_JSON_OVERRIDE:-${PIPELINE_TEMPLATE_NODE_SELECTOR_JSON:-}}"
PIPELINE_TEMPLATE_TOLERATIONS_JSON_OVERRIDE="${PIPELINE_TEMPLATE_TOLERATIONS_JSON_OVERRIDE:-${PIPELINE_TEMPLATE_TOLERATIONS_JSON:-}}"
PIPELINE_RUNTIME_MOUNT_CATALOG_JSON_OVERRIDE="${PIPELINE_RUNTIME_MOUNT_CATALOG_JSON_OVERRIDE:-${PIPELINE_RUNTIME_MOUNT_CATALOG_JSON:-}}"
PIPELINE_RUNTIME_SECRET_RESOURCES_JSON_OVERRIDE="${PIPELINE_RUNTIME_SECRET_RESOURCES_JSON_OVERRIDE:-${PIPELINE_RUNTIME_SECRET_RESOURCES_JSON:-}}"
PIPELINE_RUNTIME_STORAGE_RESOURCES_JSON_OVERRIDE="${PIPELINE_RUNTIME_STORAGE_RESOURCES_JSON_OVERRIDE:-${PIPELINE_RUNTIME_STORAGE_RESOURCES_JSON:-}}"
PIPELINE_RUNTIME_MOUNT_TARGET_IDS_OVERRIDE="${PIPELINE_RUNTIME_MOUNT_TARGET_IDS_OVERRIDE:-${PIPELINE_RUNTIME_MOUNT_TARGET_IDS:-}}"
# The dev smoke SecretProviderClass exists only in the default DataBrew
# namespace. Keep it target-scoped so video-proc-dev workloads cannot mount a
# namespace-local resource that does not exist there.
PIPELINE_RUNTIME_SECRET_TARGET_IDS_OVERRIDE="${PIPELINE_RUNTIME_SECRET_TARGET_IDS_OVERRIDE:-${PIPELINE_RUNTIME_SECRET_TARGET_IDS:-default,cyber-databrew-dev}}"
PIPELINE_RUNTIME_STORAGE_EMPTYDIR_TARGET_IDS_OVERRIDE="${PIPELINE_RUNTIME_STORAGE_EMPTYDIR_TARGET_IDS_OVERRIDE:-${PIPELINE_RUNTIME_STORAGE_EMPTYDIR_TARGET_IDS:-}}"

# After merging Kubernetes ConfigMap/Secret, values are often meant for in-cluster pods (Service DNS),
# not Cloud Run. When APPLY_CLOUDRUN_ENV_FIX=true (default), we rewrite known footguns unless you
# disable the fix or set explicit CLOUDRUN_* overrides.
APPLY_CLOUDRUN_ENV_FIX="${APPLY_CLOUDRUN_ENV_FIX:-true}"
# Set to false to keep OUTBOX_* exactly as merged from K8s (still patches ENV/ELASTICSEARCH_URL unless disabled).
CLOUDRUN_PATCH_OUTBOX="${CLOUDRUN_PATCH_OUTBOX:-true}"
CLOUDRUN_ENV="${CLOUDRUN_ENV:-production}"
# VPC-reachable Elasticsearch for Cloud Run (override per environment).
CLOUDRUN_ELASTICSEARCH_URL="${CLOUDRUN_ELASTICSEARCH_URL:-http://10.2.0.10:9200}"
CLOUDRUN_OUTBOX_RELAY_ENABLED="${CLOUDRUN_OUTBOX_RELAY_ENABLED:-true}"
CLOUDRUN_OUTBOX_ES_SUBSCRIBER_ENABLED="${CLOUDRUN_OUTBOX_ES_SUBSCRIBER_ENABLED:-true}"
CLOUDRUN_OUTBOX_TRANSPORT="${CLOUDRUN_OUTBOX_TRANSPORT:-internal}"
CLOUDRUN_OUTBOX_RELAY_PARALLEL_KEYS="${CLOUDRUN_OUTBOX_RELAY_PARALLEL_KEYS:-8}"
CLOUDRUN_OUTBOX_INTERNAL_SUBSCRIBER_WORKERS="${CLOUDRUN_OUTBOX_INTERNAL_SUBSCRIBER_WORKERS:-16}"
# Relay ClaimPendingSafe limit per flush (backend default 200).
CLOUDRUN_OUTBOX_RELAY_BATCH_SIZE="${CLOUDRUN_OUTBOX_RELAY_BATCH_SIZE:-500}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
BACKEND_DIR="${REPO_ROOT}/backend"

upsert_env() {
  local key="$1"
  local value="$2"
  local file="$3"
  if grep -q "^${key}=" "$file"; then
    awk -F= -v k="$key" -v v="$value" 'BEGIN{OFS="="} $1==k{$0=k"="v} {print}' "$file" > "${file}.tmp"
    mv "${file}.tmp" "$file"
  else
    printf '%s=%s\n' "$key" "$value" >> "$file"
  fi
}

remove_env() {
  local key="$1"
  local file="$2"
  awk -F= -v k="$key" '$1!=k{print}' "$file" > "${file}.tmp"
  mv "${file}.tmp" "$file"
}

current_cloudrun_env_value() {
  local key="$1"
  gcloud run services describe "${SERVICE_NAME}" \
    --project "${PROJECT_ID}" \
    --region "${REGION}" \
    --format=json 2>/dev/null \
    | python3 -c 'import json,sys
key=sys.argv[1]
try:
    svc=json.load(sys.stdin)
except Exception:
    print("")
    raise SystemExit(0)
for item in svc.get("spec", {}).get("template", {}).get("spec", {}).get("containers", [{}])[0].get("env", []):
    if item.get("name") == key and "value" in item:
        print(item.get("value") or "")
        break
' "${key}"
}

# preserve_all_cloudrun_env_vars reads ALL env vars from the current Cloud Run
# service and adds any that are NOT in the deploy file AND NOT in the exclude
# list. This prevents silently dropping manually-configured env vars (e.g.
# K8S_USE_METADATA_TOKEN, PIPELINE_TEMPLATE_TOLERATIONS_JSON) every time
# backend-dev.sh runs with --env-vars-file.
preserve_all_cloudrun_env_vars() {
  local file="$1"
  local exclude="^(PORT|K8S_CA_DATA|K8S_CA_B64|K8S_BEARER_TOKEN|ELASTICSEARCH_PASSWORD|DB_PASSWORD|ARGO_BASE_URL|ARGO_SERVER_URL|ARGO_TOKEN|ARGO_AUTH_TOKEN|TRINO_ENABLED|TRINO_URL|TRINO_CATALOG|TRINO_SCHEMA)$"

  gcloud run services describe "${SERVICE_NAME}" \
    --project "${PROJECT_ID}" \
    --region "${REGION}" \
    --format=json 2>/dev/null \
    | python3 -c '
import json, re, sys

svc = json.load(sys.stdin)
file_path = "'"${file}"'"
exclude = re.compile(r"'"${exclude}"'")

# Read keys already in the deploy env file
file_keys = set()
with open(file_path, "r") as f:
    for line in f:
        line = line.strip()
        if "=" in line:
            file_keys.add(line.split("=", 1)[0])

# Check each Cloud Run env var
preserved = 0
for item in svc.get("spec", {}).get("template", {}).get("spec", {}).get("containers", [{}])[0].get("env", []):
    key = item.get("name", "")
    value = item.get("value", "")
    if not key or not value:
        continue  # skip Secret Manager refs and empty vars
    if exclude.match(key):
        continue  # intentionally removed by this script
    if key in file_keys:
        continue  # already in deploy file
    # Preserve: not excluded, not in deploy file
    with open(file_path, "a") as f:
        f.write(f"{key}={value}\n")
    preserved += 1
    print(f"INFO: Preserving existing Cloud Run env {key}.", file=sys.stderr)

if preserved == 0:
    print("INFO: No additional Cloud Run env vars to preserve.", file=sys.stderr)
else:
    print(f"INFO: Preserved {preserved} env var(s) from existing Cloud Run service.", file=sys.stderr)
'
}

preserve_current_cloudrun_env_if_unset() {
  local key="$1"
  local file="$2"
  if grep -q "^${key}=" "${file}"; then
    return 0
  fi
  local current_value
  current_value="$(current_cloudrun_env_value "${key}" || true)"
  if [[ -n "${current_value}" ]]; then
    echo "INFO: Preserving existing Cloud Run env ${key}."
    upsert_env "${key}" "${current_value}" "${file}"
  fi
}

# True if ELASTICSEARCH_URL points at in-cluster DNS / headless names Cloud Run cannot resolve.
_elasticsearch_url_is_in_cluster() {
  local u="$1"
  [[ -n "${u}" ]] || return 1
  case "${u}" in
  *://elasticsearch:* | *://elasticsearch/* | *://elasticsearch.* | *elasticsearch.*.svc* | *\.svc\.cluster\.local*)
    return 0
    ;;
  esac
  return 1
}

# Normalize ENV / Elasticsearch / Outbox after K8s merge so Cloud Run does not inherit dev-cluster-only settings.
apply_cloudrun_env_fix() {
  local f="$1"
  [[ "${APPLY_CLOUDRUN_ENV_FIX}" == "true" ]] || return 0

  upsert_env "ENV" "${CLOUDRUN_ENV}" "${f}"

  local es_current
  es_current="$(awk -F= '$1=="ELASTICSEARCH_URL"{print $2}' "${f}" | tail -n 1 || true)"
  if _elasticsearch_url_is_in_cluster "${es_current}"; then
    echo "INFO: ELASTICSEARCH_URL from merged env looks in-cluster (${es_current})."
    echo "      Replacing with CLOUDRUN_ELASTICSEARCH_URL=${CLOUDRUN_ELASTICSEARCH_URL}"
    upsert_env "ELASTICSEARCH_URL" "${CLOUDRUN_ELASTICSEARCH_URL}" "${f}"
  fi

  if [[ "${CLOUDRUN_PATCH_OUTBOX}" == "true" ]]; then
    upsert_env "OUTBOX_RELAY_ENABLED" "${CLOUDRUN_OUTBOX_RELAY_ENABLED}" "${f}"
    upsert_env "OUTBOX_ES_SUBSCRIBER_ENABLED" "${CLOUDRUN_OUTBOX_ES_SUBSCRIBER_ENABLED}" "${f}"
    upsert_env "OUTBOX_TRANSPORT" "${CLOUDRUN_OUTBOX_TRANSPORT}" "${f}"
    upsert_env "OUTBOX_RELAY_PARALLEL_KEYS" "${CLOUDRUN_OUTBOX_RELAY_PARALLEL_KEYS}" "${f}"
    upsert_env "OUTBOX_INTERNAL_SUBSCRIBER_WORKERS" "${CLOUDRUN_OUTBOX_INTERNAL_SUBSCRIBER_WORKERS}" "${f}"
    echo "INFO: Patched OUTBOX_* for Cloud Run (CLOUDRUN_PATCH_OUTBOX=false to keep K8s values)."
  fi
  upsert_env "OUTBOX_RELAY_BATCH_SIZE" "${CLOUDRUN_OUTBOX_RELAY_BATCH_SIZE}" "${f}"

  # Drop any historical Trino keys merged from K8s so Cloud Run env stays clean.
  remove_env "TRINO_ENABLED" "${f}"
  remove_env "TRINO_URL" "${f}"
  remove_env "TRINO_CATALOG" "${f}"
  remove_env "TRINO_SCHEMA" "${f}"
}

if [[ "${USE_EXISTING_IMAGE}" != "true" ]]; then
  echo "Building backend image: ${IMAGE}"
  docker build \
    --platform linux/amd64 \
    -f "${BACKEND_DIR}/Dockerfile" \
    -t "${IMAGE}" \
    "${BACKEND_DIR}"
  docker push "${IMAGE}"
else
  echo "Skipping build and reusing image: ${IMAGE}"
fi

ENV_KV_FILE="$(mktemp)"
ENV_VARS_FILE="$(mktemp)"
trap 'rm -f "${ENV_KV_FILE}" "${ENV_VARS_FILE}"' EXIT

if [[ "${SOURCE_K8S_ENV}" == "true" ]]; then
  echo "Loading env from k8s namespace ${K8S_NAMESPACE} (${K8S_CONFIGMAP_NAME}, ${K8S_SECRET_NAME})"
  kubectl -n "${K8S_NAMESPACE}" get configmap "${K8S_CONFIGMAP_NAME}" \
    -o go-template='{{range $k,$v := .data}}{{printf "%s=%s\n" $k $v}}{{end}}' > "${ENV_KV_FILE}"

  while IFS= read -r key; do
    [[ -z "${key}" ]] && continue
    value_b64="$(kubectl -n "${K8S_NAMESPACE}" get secret "${K8S_SECRET_NAME}" -o "jsonpath={.data.${key}}")"
    value="$(printf '%s' "${value_b64}" | base64 --decode)"
    printf '%s=%s\n' "${key}" "${value}" >> "${ENV_KV_FILE}"
  done < <(kubectl -n "${K8S_NAMESPACE}" get secret "${K8S_SECRET_NAME}" -o go-template='{{range $k,$v := .data}}{{printf "%s\n" $k}}{{end}}')
fi

if [[ -n "${ENV_FILE}" ]]; then
  if [[ ! -f "${ENV_FILE}" ]]; then
    echo "ERROR: ENV_FILE does not exist: ${ENV_FILE}" >&2
    exit 1
  fi
  echo "Merging env values from file: ${ENV_FILE}"
  printf '\n' >> "${ENV_KV_FILE}"
  sed -e 's/\r$//' "${ENV_FILE}" >> "${ENV_KV_FILE}"
  printf '\n' >> "${ENV_KV_FILE}"
fi

# Cloud Run defaults and explicit safety toggles.
remove_env "PORT" "${ENV_KV_FILE}"
# Do not blank ELASTICSEARCH_URL when ELASTICSEARCH_URL_OVERRIDE is empty (K8s-sourced value).

secret_mappings=()
remove_es_password_secret=false
if [[ -n "${ELASTICSEARCH_PASSWORD_SECRET}" ]]; then
  remove_env "ELASTICSEARCH_PASSWORD" "${ENV_KV_FILE}"
  secret_mappings+=("ELASTICSEARCH_PASSWORD=${ELASTICSEARCH_PASSWORD_SECRET}:${ELASTICSEARCH_PASSWORD_SECRET_VERSION}")
elif [[ -n "${ELASTICSEARCH_PASSWORD_OVERRIDE}" ]]; then
  # Prefer explicit non-secret override and clear any prior secret binding when possible.
  remove_es_password_secret=true
else
  # ES is configured without HTTP auth; clear plain env and prior secret refs.
  remove_env "ELASTICSEARCH_PASSWORD" "${ENV_KV_FILE}"
  remove_es_password_secret=true
fi

if [[ -n "${GRACE_PASSWORD_SECRET}" ]]; then
  remove_env "GRACE_PASSWORD" "${ENV_KV_FILE}"
  secret_mappings+=("GRACE_PASSWORD=${GRACE_PASSWORD_SECRET}:${GRACE_PASSWORD_SECRET_VERSION}")
fi

if [[ -n "${DB_PASSWORD_SECRET}" ]]; then
  remove_env "DB_PASSWORD" "${ENV_KV_FILE}"
  secret_mappings+=("DB_PASSWORD=${DB_PASSWORD_SECRET}:${DB_PASSWORD_SECRET_VERSION}")
elif [[ -n "${DB_PASSWORD_OVERRIDE}" ]]; then
  # DB_PASSWORD_OVERRIDE is handled via upsert below; leave it in the env file.
  :
else
  # Neither secret nor override: DB_PASSWORD came from K8s as plain text but
  # the existing Cloud Run service has it as a Secret Manager reference.
  # Remove it from the env file to avoid gcloud type conflict.
  echo "INFO: DB_PASSWORD_SECRET and DB_PASSWORD_OVERRIDE both unset."
  echo "      Removing DB_PASSWORD from deploy env to preserve existing Secret Manager config."
  remove_env "DB_PASSWORD" "${ENV_KV_FILE}"
fi

[[ -n "${DB_HOST_OVERRIDE}" ]] && upsert_env "DB_HOST" "${DB_HOST_OVERRIDE}" "${ENV_KV_FILE}"
[[ -n "${DB_PORT_OVERRIDE}" ]] && upsert_env "DB_PORT" "${DB_PORT_OVERRIDE}" "${ENV_KV_FILE}"
[[ -n "${DB_USER_OVERRIDE}" ]] && upsert_env "DB_USER" "${DB_USER_OVERRIDE}" "${ENV_KV_FILE}"
[[ -n "${DB_PASSWORD_OVERRIDE}" ]] && upsert_env "DB_PASSWORD" "${DB_PASSWORD_OVERRIDE}" "${ENV_KV_FILE}"
[[ -n "${DB_NAME_OVERRIDE}" ]] && upsert_env "DB_NAME" "${DB_NAME_OVERRIDE}" "${ENV_KV_FILE}"
[[ -n "${DATABREW_TOKEN_OVERRIDE}" ]] && upsert_env "DATABREW_TOKEN" "${DATABREW_TOKEN_OVERRIDE}" "${ENV_KV_FILE}"
[[ -n "${LAKEHOUSE_BACKEND_OVERRIDE}" ]] && upsert_env "LAKEHOUSE_BACKEND" "${LAKEHOUSE_BACKEND_OVERRIDE}" "${ENV_KV_FILE}"
[[ -n "${LAKEHOUSE_BQ_PROJECT_OVERRIDE}" ]] && upsert_env "LAKEHOUSE_BQ_PROJECT" "${LAKEHOUSE_BQ_PROJECT_OVERRIDE}" "${ENV_KV_FILE}"
[[ -n "${LAKEHOUSE_BQ_DATASET_OVERRIDE}" ]] && upsert_env "LAKEHOUSE_BQ_DATASET" "${LAKEHOUSE_BQ_DATASET_OVERRIDE}" "${ENV_KV_FILE}"
[[ -n "${PUBSUB_PROJECT_OVERRIDE}" ]] && upsert_env "PUBSUB_PROJECT" "${PUBSUB_PROJECT_OVERRIDE}" "${ENV_KV_FILE}"
[[ -n "${TOPIC_ASSET_EVENTS_OVERRIDE}" ]] && upsert_env "TOPIC_ASSET_EVENTS" "${TOPIC_ASSET_EVENTS_OVERRIDE}" "${ENV_KV_FILE}"
[[ -n "${OUTBOX_ES_SUBSCRIPTION_OVERRIDE}" ]] && upsert_env "OUTBOX_ES_SUBSCRIPTION" "${OUTBOX_ES_SUBSCRIPTION_OVERRIDE}" "${ENV_KV_FILE}"
[[ -n "${ELASTICSEARCH_URL_OVERRIDE}" ]] && upsert_env "ELASTICSEARCH_URL" "${ELASTICSEARCH_URL_OVERRIDE}" "${ENV_KV_FILE}"
[[ -n "${ELASTICSEARCH_PASSWORD_OVERRIDE}" ]] && upsert_env "ELASTICSEARCH_PASSWORD" "${ELASTICSEARCH_PASSWORD_OVERRIDE}" "${ENV_KV_FILE}"
[[ -n "${PRICING_CONFIG_PATH_OVERRIDE}" ]] && upsert_env "PRICING_CONFIG_PATH" "${PRICING_CONFIG_PATH_OVERRIDE}" "${ENV_KV_FILE}"
[[ -n "${K8S_API_ENDPOINT_OVERRIDE}" ]] && upsert_env "K8S_API_ENDPOINT" "${K8S_API_ENDPOINT_OVERRIDE}" "${ENV_KV_FILE}"
if [[ -n "${K8S_BEARER_TOKEN_SECRET}" ]]; then
  remove_env "K8S_BEARER_TOKEN" "${ENV_KV_FILE}"
  secret_mappings+=("K8S_BEARER_TOKEN=${K8S_BEARER_TOKEN_SECRET}:${K8S_BEARER_TOKEN_SECRET_VERSION}")
fi
if [[ -n "${K8S_CA_DATA_SECRET}" ]]; then
  remove_env "K8S_CA_DATA" "${ENV_KV_FILE}"
  remove_env "K8S_CA_B64" "${ENV_KV_FILE}"
  upsert_env "K8S_INSECURE_SKIP_VERIFY" "false" "${ENV_KV_FILE}"
  secret_mappings+=("K8S_CA_DATA=${K8S_CA_DATA_SECRET}:${K8S_CA_DATA_SECRET_VERSION}")
  # Keep K8S_CA_B64 bound to the same base64 PEM secret so revisions running
  # the older buildTLSConfig path still verify the GKE API certificate.
  secret_mappings+=("K8S_CA_B64=${K8S_CA_DATA_SECRET}:${K8S_CA_DATA_SECRET_VERSION}")
fi

# Workload Identity K8s auth — apply explicit override when set.
# (preserve_all_cloudrun_env_vars handles keeping existing Cloud Run values.)
	[[ -n "${K8S_AUDIENCE_OVERRIDE}" ]] && upsert_env "K8S_AUDIENCE" "${K8S_AUDIENCE_OVERRIDE}" "${ENV_KV_FILE}"
	[[ -n "${K8S_USE_METADATA_TOKEN_OVERRIDE}" ]] && upsert_env "K8S_USE_METADATA_TOKEN" "${K8S_USE_METADATA_TOKEN_OVERRIDE}" "${ENV_KV_FILE}"
	[[ -n "${K8S_INSECURE_SKIP_VERIFY_OVERRIDE}" ]] && upsert_env "K8S_INSECURE_SKIP_VERIFY" "${K8S_INSECURE_SKIP_VERIFY_OVERRIDE}" "${ENV_KV_FILE}"

# Argo Workflows server lives in K8s, not on Cloud Run. Drop any K8s-merged
# ARGO_BASE_URL (which points at the Cloud Run pipeline-ui proxy) and force
# ARGO_SERVER_URL to the in-cluster Argo API. No ARGO_TOKEN needed with
# --auth-mode=server (anonymous access).
remove_env "ARGO_BASE_URL" "${ENV_KV_FILE}"
remove_env "ARGO_SERVER_URL" "${ENV_KV_FILE}"
remove_env "ARGO_TOKEN" "${ENV_KV_FILE}"
remove_env "ARGO_AUTH_TOKEN" "${ENV_KV_FILE}"
[[ -n "${ARGO_SERVER_URL_OVERRIDE}" ]] && upsert_env "ARGO_SERVER_URL" "${ARGO_SERVER_URL_OVERRIDE}" "${ENV_KV_FILE}"
[[ -n "${ARGO_WORKFLOW_TTL_SECONDS_AFTER_COMPLETION_OVERRIDE}" ]] && upsert_env "ARGO_WORKFLOW_TTL_SECONDS_AFTER_COMPLETION" "${ARGO_WORKFLOW_TTL_SECONDS_AFTER_COMPLETION_OVERRIDE}" "${ENV_KV_FILE}"
[[ -n "${ARGO_RUN_WEBHOOK_URL_OVERRIDE}" ]] && upsert_env "ARGO_RUN_WEBHOOK_URL" "${ARGO_RUN_WEBHOOK_URL_OVERRIDE}" "${ENV_KV_FILE}"
[[ -n "${GRACE_API_URL_OVERRIDE}" ]] && upsert_env "GRACE_API_URL" "${GRACE_API_URL_OVERRIDE}" "${ENV_KV_FILE}"
[[ -n "${GRACE_USERNAME_OVERRIDE}" ]] && upsert_env "GRACE_USERNAME" "${GRACE_USERNAME_OVERRIDE}" "${ENV_KV_FILE}"
[[ -n "${PIPELINE_RUN_WATCHER_INTERVAL_SEC_OVERRIDE}" ]] && upsert_env "PIPELINE_RUN_WATCHER_INTERVAL_SEC" "${PIPELINE_RUN_WATCHER_INTERVAL_SEC_OVERRIDE}" "${ENV_KV_FILE}"
if [[ -n "${ARGO_RUN_WEBHOOK_TOKEN_SECRET}" ]]; then
  remove_env "ARGO_RUN_WEBHOOK_TOKEN" "${ENV_KV_FILE}"
  secret_mappings+=("ARGO_RUN_WEBHOOK_TOKEN=${ARGO_RUN_WEBHOOK_TOKEN_SECRET}:${ARGO_RUN_WEBHOOK_TOKEN_SECRET_VERSION}")
fi
[[ -n "${FRONTEND_BASE_URL_OVERRIDE}" ]] && upsert_env "FRONTEND_BASE_URL" "${FRONTEND_BASE_URL_OVERRIDE}" "${ENV_KV_FILE}"
[[ -n "${ADMIN_EMAILS_OVERRIDE}" ]] && upsert_env "ADMIN_EMAILS" "${ADMIN_EMAILS_OVERRIDE}" "${ENV_KV_FILE}"
if [[ -n "${BACKFILL_NOTIFY_FEISHU_WEBHOOK_SECRET}" ]]; then
  # Remove any plain value first: Cloud Run rejects an env var set as both a literal
  # and a secret. The secret becomes the single source of truth for the webhook.
  remove_env "BACKFILL_NOTIFY_FEISHU_WEBHOOK_URL" "${ENV_KV_FILE}"
  secret_mappings+=("BACKFILL_NOTIFY_FEISHU_WEBHOOK_URL=${BACKFILL_NOTIFY_FEISHU_WEBHOOK_SECRET}:${BACKFILL_NOTIFY_FEISHU_WEBHOOK_SECRET_VERSION}")
fi
[[ -n "${PIPELINE_RESOURCE_MAX_CPU_OVERRIDE}" ]] && upsert_env "PIPELINE_RESOURCE_MAX_CPU" "${PIPELINE_RESOURCE_MAX_CPU_OVERRIDE}" "${ENV_KV_FILE}"
[[ -n "${PIPELINE_RESOURCE_MAX_MEMORY_OVERRIDE}" ]] && upsert_env "PIPELINE_RESOURCE_MAX_MEMORY" "${PIPELINE_RESOURCE_MAX_MEMORY_OVERRIDE}" "${ENV_KV_FILE}"
[[ -n "${PIPELINE_RESOURCE_MAX_DISK_OVERRIDE}" ]] && upsert_env "PIPELINE_RESOURCE_MAX_DISK" "${PIPELINE_RESOURCE_MAX_DISK_OVERRIDE}" "${ENV_KV_FILE}"
[[ -n "${PIPELINE_RESOURCE_MAX_GPU_OVERRIDE}" ]] && upsert_env "PIPELINE_RESOURCE_MAX_GPU" "${PIPELINE_RESOURCE_MAX_GPU_OVERRIDE}" "${ENV_KV_FILE}"
[[ -n "${PIPELINE_UNSCHEDULABLE_PENDING_THRESHOLD_OVERRIDE}" ]] && upsert_env "PIPELINE_UNSCHEDULABLE_PENDING_THRESHOLD" "${PIPELINE_UNSCHEDULABLE_PENDING_THRESHOLD_OVERRIDE}" "${ENV_KV_FILE}"
[[ -z "${PIPELINE_TEMPLATE_NODE_SELECTOR_JSON_OVERRIDE}" ]] && preserve_current_cloudrun_env_if_unset "PIPELINE_TEMPLATE_NODE_SELECTOR_JSON" "${ENV_KV_FILE}"
[[ -z "${PIPELINE_TEMPLATE_TOLERATIONS_JSON_OVERRIDE}" ]] && preserve_current_cloudrun_env_if_unset "PIPELINE_TEMPLATE_TOLERATIONS_JSON" "${ENV_KV_FILE}"
[[ -z "${PIPELINE_RUNTIME_MOUNT_CATALOG_JSON_OVERRIDE}" ]] && preserve_current_cloudrun_env_if_unset "PIPELINE_RUNTIME_MOUNT_CATALOG_JSON" "${ENV_KV_FILE}"
[[ -z "${PIPELINE_RUNTIME_SECRET_RESOURCES_JSON_OVERRIDE}" ]] && preserve_current_cloudrun_env_if_unset "PIPELINE_RUNTIME_SECRET_RESOURCES_JSON" "${ENV_KV_FILE}"
[[ -z "${PIPELINE_RUNTIME_STORAGE_RESOURCES_JSON_OVERRIDE}" ]] && preserve_current_cloudrun_env_if_unset "PIPELINE_RUNTIME_STORAGE_RESOURCES_JSON" "${ENV_KV_FILE}"
[[ -z "${PIPELINE_RUNTIME_MOUNT_TARGET_IDS_OVERRIDE}" ]] && preserve_current_cloudrun_env_if_unset "PIPELINE_RUNTIME_MOUNT_TARGET_IDS" "${ENV_KV_FILE}"
[[ -z "${PIPELINE_RUNTIME_SECRET_TARGET_IDS_OVERRIDE}" ]] && preserve_current_cloudrun_env_if_unset "PIPELINE_RUNTIME_SECRET_TARGET_IDS" "${ENV_KV_FILE}"
[[ -z "${PIPELINE_RUNTIME_STORAGE_EMPTYDIR_TARGET_IDS_OVERRIDE}" ]] && preserve_current_cloudrun_env_if_unset "PIPELINE_RUNTIME_STORAGE_EMPTYDIR_TARGET_IDS" "${ENV_KV_FILE}"
[[ -n "${PIPELINE_TEMPLATE_NODE_SELECTOR_JSON_OVERRIDE}" ]] && upsert_env "PIPELINE_TEMPLATE_NODE_SELECTOR_JSON" "${PIPELINE_TEMPLATE_NODE_SELECTOR_JSON_OVERRIDE}" "${ENV_KV_FILE}"
[[ -n "${PIPELINE_TEMPLATE_TOLERATIONS_JSON_OVERRIDE}" ]] && upsert_env "PIPELINE_TEMPLATE_TOLERATIONS_JSON" "${PIPELINE_TEMPLATE_TOLERATIONS_JSON_OVERRIDE}" "${ENV_KV_FILE}"
[[ -n "${PIPELINE_RUNTIME_MOUNT_CATALOG_JSON_OVERRIDE}" ]] && upsert_env "PIPELINE_RUNTIME_MOUNT_CATALOG_JSON" "${PIPELINE_RUNTIME_MOUNT_CATALOG_JSON_OVERRIDE}" "${ENV_KV_FILE}"
[[ -n "${PIPELINE_RUNTIME_SECRET_RESOURCES_JSON_OVERRIDE}" ]] && upsert_env "PIPELINE_RUNTIME_SECRET_RESOURCES_JSON" "${PIPELINE_RUNTIME_SECRET_RESOURCES_JSON_OVERRIDE}" "${ENV_KV_FILE}"
[[ -n "${PIPELINE_RUNTIME_STORAGE_RESOURCES_JSON_OVERRIDE}" ]] && upsert_env "PIPELINE_RUNTIME_STORAGE_RESOURCES_JSON" "${PIPELINE_RUNTIME_STORAGE_RESOURCES_JSON_OVERRIDE}" "${ENV_KV_FILE}"
[[ -n "${PIPELINE_RUNTIME_MOUNT_TARGET_IDS_OVERRIDE}" ]] && upsert_env "PIPELINE_RUNTIME_MOUNT_TARGET_IDS" "${PIPELINE_RUNTIME_MOUNT_TARGET_IDS_OVERRIDE}" "${ENV_KV_FILE}"
[[ -n "${PIPELINE_RUNTIME_SECRET_TARGET_IDS_OVERRIDE}" ]] && upsert_env "PIPELINE_RUNTIME_SECRET_TARGET_IDS" "${PIPELINE_RUNTIME_SECRET_TARGET_IDS_OVERRIDE}" "${ENV_KV_FILE}"
[[ -n "${PIPELINE_RUNTIME_STORAGE_EMPTYDIR_TARGET_IDS_OVERRIDE}" ]] && upsert_env "PIPELINE_RUNTIME_STORAGE_EMPTYDIR_TARGET_IDS" "${PIPELINE_RUNTIME_STORAGE_EMPTYDIR_TARGET_IDS_OVERRIDE}" "${ENV_KV_FILE}"

apply_cloudrun_env_fix "${ENV_KV_FILE}"

effective_db_host="$(awk -F= '$1=="DB_HOST"{print $2}' "${ENV_KV_FILE}" | tail -n 1 || true)"
if [[ "${effective_db_host}" == "postgres" ]]; then
  echo "WARNING: DB_HOST=postgres comes from in-cluster DNS and is usually unreachable from Cloud Run."
  echo "         Set DB_HOST_OVERRIDE (and likely VPC_CONNECTOR) to a reachable PostgreSQL endpoint."
  if [[ -z "${DB_HOST_OVERRIDE}" ]]; then
    postgres_pod_ip="$(kubectl -n "${K8S_NAMESPACE}" get pod -l app.kubernetes.io/component=postgres -o jsonpath='{.items[0].status.podIP}' 2>/dev/null || true)"
    if [[ -n "${postgres_pod_ip}" ]]; then
      echo "INFO: Auto-detected PostgreSQL pod IP for Cloud Run fallback: ${postgres_pod_ip}"
      upsert_env "DB_HOST" "${postgres_pod_ip}" "${ENV_KV_FILE}"
    else
      echo "WARNING: Could not detect PostgreSQL pod IP automatically."
    fi
  fi
fi

preserve_all_cloudrun_env_vars "${ENV_KV_FILE}"

python3 - "${ENV_KV_FILE}" "${ENV_VARS_FILE}" <<'PY'
import json
import sys

src, dst = sys.argv[1], sys.argv[2]
data = {}
with open(src, "r", encoding="utf-8") as f:
    for raw in f:
        line = raw.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        k, v = line.split("=", 1)
        data[k] = v
with open(dst, "w", encoding="utf-8") as f:
    json.dump(data, f)
PY

echo "Deploying ${SERVICE_NAME} to Cloud Run (${REGION}, ${PROJECT_ID})"
deploy_args=(
  run deploy "${SERVICE_NAME}"
  --quiet
  --project "${PROJECT_ID}"
  --region "${REGION}"
  --platform managed
  --image "${IMAGE}"
  --port 8080
  --min-instances "${MIN_INSTANCES}"
  --max-instances "${MAX_INSTANCES}"
  --cpu "${CPU}"
  --memory "${MEMORY}"
  --timeout "${TIMEOUT}"
  --env-vars-file "${ENV_VARS_FILE}"
)
if [[ "${CPU_THROTTLING}" == "true" ]]; then
  deploy_args+=(--cpu-throttling)
else
  deploy_args+=(--no-cpu-throttling)
fi
if [[ "${CPU_BOOST}" == "true" ]]; then
  deploy_args+=(--cpu-boost)
else
  deploy_args+=(--no-cpu-boost)
fi
if [[ ${#secret_mappings[@]} -gt 0 ]]; then
  secret_arg="$(IFS=,; echo "${secret_mappings[*]}")"
  deploy_args+=(--set-secrets "${secret_arg}")
fi
if [[ "${remove_es_password_secret}" == "true" && ${#secret_mappings[@]} -eq 0 ]]; then
  deploy_args+=(--remove-secrets "ELASTICSEARCH_PASSWORD")
fi

# CYB-3486d1c: when set, the new revision is created idle. Traffic must be
# routed explicitly (e.g. by the deploy-dev.yml migrate step after Atlas
# migrations run). Without this, `gcloud run deploy` defaults to 100%-traffic
# on the new revision, which races the migration step — new code queries
# columns the old schema doesn't have yet, until traffic-switch completes.
if [[ "${DEPLOY_NO_TRAFFIC:-false}" == "true" ]]; then
  deploy_args+=(--no-traffic)
fi

if [[ "${ALLOW_UNAUTHENTICATED}" == "true" ]]; then
  deploy_args+=(--allow-unauthenticated)
else
  deploy_args+=(--no-allow-unauthenticated)
fi

if [[ -n "${VPC_CONNECTOR}" ]]; then
  deploy_args+=(--vpc-connector "${VPC_CONNECTOR}" --vpc-egress "${VPC_EGRESS}")
fi

gcloud "${deploy_args[@]}"

echo "Deployment complete."
gcloud run services describe "${SERVICE_NAME}" \
  --project "${PROJECT_ID}" \
  --region "${REGION}" \
  --platform managed \
  --format='value(status.url)'
