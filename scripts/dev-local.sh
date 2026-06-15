#!/usr/bin/env bash
# Local Vite frontend + remote GKE backend Pod (no frontend deploy).
#
# Usage:
#   bash scripts/dev-local.sh                    # shared dev backend Pod (default)
#   bash scripts/dev-local.sh --shared           # same as default
#   bash scripts/dev-local.sh --preview          # HEAD preview Pod; deploy if missing
#   bash scripts/dev-local.sh --preview-id <id>    # existing preview Pod
#   bash scripts/dev-local.sh --latest             # newest preview Pod (no deploy)
#   bash scripts/dev-local.sh --deploy [REF]       # deploy backend preview, then start Vite
#
# Environment:
#   PREVIEW_NAMESPACE          default: cyber-databrew-dev
#   PREVIEW_PUBLIC_BASE_URL    default: https://cyber-databrew-dev.cyberorigin.ai
#   DEV_K8S_NAMESPACE          default: cyber-databrew-dev
#   DEV_BACKEND_SERVICE        default: cyber-databrew-backend
#   SKIP_API_CHECK=true        skip preflight curl
set -euo pipefail

usage() {
  sed -n '2,12p' "$0" | sed 's/^# \?//'
}

die() {
  echo "ERROR: $*" >&2
  exit 1
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "missing required command: $1"
}

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"
frontend_dir="${repo_root}/Frontend"
preview_script="${repo_root}/deploy/preview/fast-preview.sh"

preview_namespace="${PREVIEW_NAMESPACE:-cyber-databrew-dev}"
dev_namespace="${DEV_K8S_NAMESPACE:-cyber-databrew-dev}"
dev_backend_service="${DEV_BACKEND_SERVICE:-cyber-databrew-backend}"
preview_host="${PREVIEW_PUBLIC_BASE_URL:-https://cyber-databrew-dev.cyberorigin.ai}"
preview_host="${preview_host%/}"
secret_name="${DEV_SECRET_NAME:-cyber-databrew-secrets}"

mode="shared"
preview_id=""
deploy_ref=""
deploy_wait=(--wait)

while [[ $# -gt 0 ]]; do
  case "$1" in
    --shared)
      mode="shared"
      shift
      ;;
    --preview)
      mode="preview"
      shift
      ;;
    --preview-id)
      preview_id="${2:-}"
      mode="preview"
      shift 2
      ;;
    --latest)
      mode="latest"
      shift
      ;;
    --deploy)
      mode="deploy"
      if [[ $# -ge 2 && "$2" != --* ]]; then
        deploy_ref="$2"
        shift 2
      else
        deploy_ref="HEAD"
        shift
      fi
      ;;
    --no-wait)
      deploy_wait=(--no-wait)
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      die "unknown option: $1 (try --help)"
      ;;
  esac
done

require_cmd kubectl
require_cmd curl
require_cmd git
[[ -d "${frontend_dir}" ]] || die "missing Frontend directory: ${frontend_dir}"

resolve_token() {
  local token=""
  for key in GRACE_TOKEN DATABREW_TOKEN; do
    token="$(kubectl get secret "${secret_name}" -n "${dev_namespace}" \
      -o "jsonpath={.data.${key}}" 2>/dev/null | base64 -d 2>/dev/null || true)"
    if [[ -n "${token}" ]]; then
      printf '%s' "${token}"
      return 0
    fi
  done
  echo "WARNING: could not read GRACE_TOKEN/DATABREW_TOKEN from ${dev_namespace}/${secret_name}; login may fail" >&2
  printf ''
}

resolve_head_preview_id() {
  git -C "${repo_root}" rev-parse --short=12 HEAD
}

preview_service_name() {
  printf 'cdb-pv-%s-api' "$1"
}

preview_service_exists() {
  kubectl get svc "$(preview_service_name "$1")" -n "${preview_namespace}" >/dev/null 2>&1
}

latest_preview_id() {
  local name id
  name="$(
    kubectl get svc -n "${preview_namespace}" --sort-by=.metadata.creationTimestamp -o name 2>/dev/null \
      | grep -E '/cdb-pv-[0-9a-f]{7,40}-api$' \
      | tail -1 \
      || true
  )"
  [[ -n "${name}" ]] || return 1
  id="${name##*/}"
  id="${id#cdb-pv-}"
  id="${id%-api}"
  printf '%s' "${id}"
}

resolve_shared_backend_url() {
  local ip=""
  ip="$(kubectl get svc "${dev_backend_service}" -n "${dev_namespace}" \
    -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || true)"
  [[ -n "${ip}" ]] || die "could not resolve LoadBalancer IP for ${dev_namespace}/${dev_backend_service}"
  printf 'http://%s' "${ip}"
}

check_shared_backend() {
  local base="$1"
  curl -sfS --max-time 15 "${base}/healthz" >/dev/null
}

check_preview_backend() {
  local id="$1"
  curl -sfS --max-time 20 "${preview_host}/preview/${id}/api/healthz" >/dev/null
}

ensure_preview_backend() {
  local ref="$1"
  [[ -f "${preview_script}" ]] || die "missing ${preview_script}"
  {
    echo "Deploying backend preview Pod only (--frontend local) for ${ref}..."
    bash "${preview_script}" "${ref}" --frontend local "${deploy_wait[@]}"
  } >&2
}

resolve_preview_id_for_mode() {
  local id=""
  case "${mode}" in
    preview)
      id="${preview_id:-$(resolve_head_preview_id)}"
      if preview_service_exists "${id}"; then
        printf '%s' "${id}"
        return 0
      fi
      echo "Preview Pod cdb-pv-${id}-api not found; deploying backend for HEAD..." >&2
      ensure_preview_backend "HEAD"
      id="$(resolve_head_preview_id)"
      preview_service_exists "${id}" || die "preview service still missing after deploy: cdb-pv-${id}-api"
      printf '%s' "${id}"
      ;;
    latest)
      id="$(latest_preview_id || true)"
      [[ -n "${id}" ]] || die "no preview backend Pod found (cdb-pv-*-api in ${preview_namespace})"
      printf '%s' "${id}"
      ;;
    deploy)
      ensure_preview_backend "${deploy_ref:-HEAD}"
      id="${preview_id:-$(resolve_head_preview_id)}"
      preview_service_exists "${id}" || die "preview service missing after deploy: cdb-pv-${id}-api"
      printf '%s' "${id}"
      ;;
    preview-id)
      id="${preview_id}"
      [[ -n "${id}" ]] || die "--preview-id requires a value"
      preview_service_exists "${id}" || die "preview service not found: cdb-pv-${id}-api"
      printf '%s' "${id}"
      ;;
    *)
      return 1
      ;;
  esac
}

if [[ "${mode}" == "preview" && -n "${preview_id}" ]]; then
  mode="preview-id"
fi

token="$(resolve_token)"
shared_url=""
resolved_preview_id=""

if [[ "${mode}" == "shared" ]]; then
  shared_url="$(resolve_shared_backend_url)"
  if [[ "${SKIP_API_CHECK:-false}" != "true" ]]; then
    echo "Checking shared backend ${shared_url}/healthz ..."
    check_shared_backend "${shared_url}" || die "shared backend health check failed"
  fi
else
  resolved_preview_id="$(resolve_preview_id_for_mode)"
  if [[ "${SKIP_API_CHECK:-false}" != "true" ]]; then
    echo "Checking preview backend ${preview_host}/preview/${resolved_preview_id}/api/healthz ..."
    check_preview_backend "${resolved_preview_id}" || die "preview backend health check failed"
  fi
fi

if [[ ! -d "${frontend_dir}/node_modules" ]]; then
  echo "Installing frontend dependencies..."
  (cd "${frontend_dir}" && npm install)
fi

echo ""
echo "Starting local frontend (Vite HMR, no frontend Pod deploy)"
if [[ "${mode}" == "shared" ]]; then
  echo "  mode:    shared dev backend Pod"
  echo "  API:     ${shared_url}"
  echo "  UI:      http://127.0.0.1:5176/"
  echo ""
  exec env \
    VITE_API_BASE_URL="${shared_url}" \
    VITE_DEV_ACCESS_TOKEN="${token}" \
    npm --prefix "${frontend_dir}" run dev
else
  echo "  mode:    preview backend Pod"
  echo "  preview: ${resolved_preview_id}"
  echo "  API:     ${preview_host}/preview/${resolved_preview_id}/api"
  echo "  UI:      http://127.0.0.1:5176/"
  echo "  check:   概览左下角应显示 preview/${resolved_preview_id}"
  echo ""
  exec env \
    VITE_PREVIEW_ID="${resolved_preview_id}" \
    VITE_PREVIEW_HOST="${preview_host}" \
    VITE_DEV_ACCESS_TOKEN="${token}" \
    npm --prefix "${frontend_dir}" run dev
fi
